package gocurl

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/maniartech/gocurl/middlewares"
	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/proxy"
)

// doRequest runs the shared request pipeline (validate, client, build, retries,
// verbose, decompress) and returns the live response with its body unread and
// open. It performs NO output side effects, so library callers control the body.
func doRequest(ctx context.Context, opts *options.RequestOptions) (*http.Response, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	var httpClient options.HTTPClient
	if opts.CustomClient != nil {
		httpClient = opts.CustomClient
	} else {
		client, err := createHTTPClient(ctx, opts)
		if err != nil {
			return nil, err
		}
		httpClient = client
	}

	req, err := createRequest(ctx, opts)
	if err != nil {
		return nil, err
	}

	req, err = applyMiddleware(req, opts.Middleware)
	if err != nil {
		return nil, err
	}

	// The one-shot path has no Client, so only the legacy options.RetryConfig
	// (method-agnostic, also set by the --retry flag) can drive retries here.
	resp, err := executeWithRetries(httpClient, req, opts, legacyPolicyFromRetryConfig(opts.RetryConfig), newRand())
	if err != nil {
		return nil, wrapTransportError(opts.URL, err)
	}

	printResponseVerbose(opts, resp)

	if opts.Compress {
		if err := decompressResponse(resp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decompress response: %w", err)
		}
	}

	return resp, nil
}

// failOnStatus implements the opt-in curl -f/--fail policy. With FailOnError set,
// a response with StatusCode >= 400 yields a ServerStatusError; the caller still
// receives the live *http.Response so it may read the error body. Without the
// opt-in, a non-2xx status is NOT an error (the default contract).
func failOnStatus(resp *http.Response, opts *options.RequestOptions) error {
	if resp == nil || !opts.FailOnError || resp.StatusCode < 400 {
		return nil
	}
	return ServerStatusError(opts.URL, resp.StatusCode)
}

// limitedBody wraps a response body to enforce a maximum size while streaming.
type limitedBody struct {
	rc    io.ReadCloser
	limit int64
	read  int64
}

func newLimitedBody(rc io.ReadCloser, limit int64) io.ReadCloser {
	return &limitedBody{rc: rc, limit: limit}
}

func (l *limitedBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	remaining := l.limit - l.read
	if remaining <= 0 {
		// We have already delivered exactly `limit` bytes. Probe a single byte to
		// distinguish a body that is exactly at the limit (clean EOF -> success)
		// from one that runs over it, without buffering more than that one byte.
		var probe [1]byte
		n, err := l.rc.Read(probe[:])
		if n > 0 {
			return 0, l.tooLargeErr()
		}
		return 0, err // EOF (exact-limit body) or a transient zero-byte read.
	}
	// Cap the slice handed to the underlying reader so it can pull at most one
	// byte past the cap into memory — that single byte is only ever used to
	// detect overflow on this Read; it is never returned to the caller.
	if int64(len(p)) > remaining+1 {
		p = p[:remaining+1]
	}
	n, err := l.rc.Read(p)
	l.read += int64(n)
	if l.read > l.limit {
		// Truncate the returned count back to the cap so the caller never sees a
		// byte past the limit, then surface a typed, classifiable error.
		n -= int(l.read - l.limit)
		l.read = l.limit
		return n, l.tooLargeErr()
	}
	return n, err
}

// tooLargeErr reports an over-limit response body as a classifiable
// KindBodyRead error (matchable via errors.Is(err, ErrBodyRead) / KindOf).
func (l *limitedBody) tooLargeErr() error {
	return &GocurlError{
		Op:   "body read",
		Kind: KindBodyRead,
		Err:  fmt.Errorf("response body size exceeds limit of %d bytes", l.limit),
	}
}

func (l *limitedBody) Close() error { return l.rc.Close() }

func validateOptions(opts *options.RequestOptions) error {
	// Use the new security validation
	return validateRequestOptions(opts)
}

func createHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
	// Obtain a (possibly cached) round tripper so connections are reused across
	// requests that share the same connection-relevant configuration.
	transport, err := getRoundTripper(opts)
	if err != nil {
		return nil, err
	}

	// Create client with per-request redirect policy and timeout.
	client := &http.Client{
		Transport:     transport,
		Timeout:       determineClientTimeout(ctx, opts),
		CheckRedirect: redirectPolicy(opts),
	}

	// Configure cookie jar
	if err := configureCookieJar(client, opts); err != nil {
		return nil, err
	}

	return client, nil
}

// createProxyTransport creates a transport with proxy configuration
func createProxyTransport(opts *options.RequestOptions, tlsConfig *tls.Config) (*http.Transport, error) {
	proxyURL, err := url.Parse(opts.Proxy)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %v", err)
	}

	// Determine proxy type
	proxyType := proxy.ProxyTypeHTTP
	if proxyURL.Scheme == "socks5" {
		proxyType = proxy.ProxyTypeSOCKS5
	}

	// Create proxy config
	proxyConfig := createProxyConfig(proxyURL, proxyType, tlsConfig, opts)

	// Apply proxy to transport
	proxyTransport, err := proxy.NewTransport(proxyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy transport: %w", err)
	}

	return proxyTransport, nil
}

// createProxyConfig creates a proxy configuration
func createProxyConfig(proxyURL *url.URL, proxyType proxy.ProxyType, tlsConfig *tls.Config, opts *options.RequestOptions) proxy.ProxyConfig {
	proxyConfig := proxy.ProxyConfig{
		Type:      proxyType,
		Address:   proxyURL.Host,
		Username:  proxyURL.User.Username(),
		TLSConfig: tlsConfig,
		NoProxy:   opts.ProxyNoProxy,
		Timeout:   opts.ConnectTimeout,
		// Proxy TLS authentication (curl-compatible)
		ClientCert: opts.ProxyCert,
		ClientKey:  opts.ProxyKey,
		CACert:     opts.ProxyCACert,
		Insecure:   opts.ProxyInsecure,
	}

	if password, hasPassword := proxyURL.User.Password(); hasPassword {
		proxyConfig.Password = password
	}

	return proxyConfig
}

// determineClientTimeout determines timeout based on context deadline vs opts.Timeout
// INDUSTRY STANDARD: Context Priority Pattern
func determineClientTimeout(ctx context.Context, opts *options.RequestOptions) time.Duration {
	if ctx != nil {
		// If context has a deadline, prefer it over opts.Timeout
		if _, hasDeadline := ctx.Deadline(); hasDeadline {
			// Don't set client.Timeout, let context handle it
			// This prevents nested timeouts and unpredictable behavior
			return 0
		}
		// No deadline in context, use opts.Timeout as fallback
		return opts.Timeout
	}
	// No context provided, use opts.Timeout
	return opts.Timeout
}

// redirectPolicy returns a CheckRedirect function honoring the request options.
func redirectPolicy(opts *options.RequestOptions) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !opts.FollowRedirects {
			return http.ErrUseLastResponse
		}
		if len(via) >= opts.MaxRedirects {
			// Wrap the sentinel so errors.Is(err, ErrTooManyRedirects) resolves
			// through net/http's *url.Error and the engine's GocurlError wrapper —
			// the CLI maps this to curl's exit 47.
			return fmt.Errorf("stopped after %d redirects: %w", opts.MaxRedirects, ErrTooManyRedirects)
		}
		return nil
	}
}

// configureCookieJar configures the cookie jar for the client
func configureCookieJar(client *http.Client, opts *options.RequestOptions) error {
	if opts.CookieFile != "" {
		jar, err := NewPersistentCookieJar(opts.CookieFile)
		if err != nil {
			return fmt.Errorf("failed to create cookie jar: %w", err)
		}
		client.Jar = jar
		return nil
	}

	if opts.CookieJar != nil {
		client.Jar = opts.CookieJar
	}

	return nil
}

func createRequest(ctx context.Context, opts *options.RequestOptions) (*http.Request, error) {
	method := getMethod(opts)
	url := buildURL(opts)
	body, contentType, err := createRequestBody(opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// For a streaming BodySource, set Content-Length when known and a GetBody so
	// retries/redirects can replay a rewindable body without buffering it.
	if opts.BodyStream != nil {
		if n, ok := opts.BodyStream.Len(); ok {
			req.ContentLength = n
		}
		if opts.BodyStream.Rewindable() {
			bs := opts.BodyStream
			req.GetBody = func() (io.ReadCloser, error) { return bs.Open() }
		}
	}

	// Apply all request configurations
	applyHeaders(req, opts, contentType)
	applyAuth(req, opts)
	applyCookies(req, opts)
	applyCompression(req, opts)
	applyRequestID(req, opts)

	// Print verbose info
	printConnectionInfo(opts, req)
	printRequestVerbose(opts, req)

	return req, nil
}

// getMethod returns the HTTP method, defaulting to GET
func getMethod(opts *options.RequestOptions) string {
	if opts.Method == "" {
		return "GET"
	}
	return opts.Method
}

// buildURL constructs the full URL with query parameters
func buildURL(opts *options.RequestOptions) string {
	url := opts.URL
	if len(opts.QueryParams) > 0 {
		separator := "?"
		if strings.Contains(url, "?") {
			separator = "&"
		}
		url += separator + opts.QueryParams.Encode()
	}
	return url
}

// createRequestBody creates the request body and determines content type
func createRequestBody(opts *options.RequestOptions) (io.Reader, string, error) {
	if opts.BodyStream != nil {
		rc, err := opts.BodyStream.Open()
		if err != nil {
			return nil, "", fmt.Errorf("failed to open request body: %w", err)
		}
		contentType := ""
		if ct, ok := opts.BodyStream.(options.ContentTyper); ok {
			contentType = ct.ContentType()
		}
		return rc, contentType, nil
	}

	if opts.Body != "" {
		return strings.NewReader(opts.Body), "", nil
	}

	if len(opts.Form) > 0 && opts.FileUpload == nil {
		return createFormBody(opts.Form)
	}

	if opts.FileUpload != nil {
		return createMultipartBody(opts)
	}

	return nil, "", nil
}

// createFormBody creates URL-encoded form body
func createFormBody(form url.Values) (io.Reader, string, error) {
	body := strings.NewReader(form.Encode())
	contentType := "application/x-www-form-urlencoded"
	return body, contentType, nil
}

// createMultipartBody creates multipart form data with file upload
func createMultipartBody(opts *options.RequestOptions) (io.Reader, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add file
	err := addFileToMultipart(w, opts.FileUpload)
	if err != nil {
		return nil, "", err
	}

	// Add other form fields
	err = addFormFieldsToMultipart(w, opts.Form)
	if err != nil {
		return nil, "", err
	}

	err = w.Close()
	if err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	return &b, w.FormDataContentType(), nil
}

// addFileToMultipart adds a file to the multipart writer
func addFileToMultipart(w *multipart.Writer, fileUpload *options.FileUpload) error {
	file, err := os.Open(fileUpload.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file for upload: %v", err)
	}
	defer file.Close()

	part, err := w.CreateFormFile(fileUpload.FieldName, fileUpload.FileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	return nil
}

// addFormFieldsToMultipart adds form fields to the multipart writer
func addFormFieldsToMultipart(w *multipart.Writer, form url.Values) error {
	for key, values := range form {
		for _, value := range values {
			err := w.WriteField(key, value)
			if err != nil {
				return fmt.Errorf("failed to write form field: %v", err)
			}
		}
	}
	return nil
}

// applyHeaders applies all headers to the request
func applyHeaders(req *http.Request, opts *options.RequestOptions, contentType string) {
	// Add custom headers
	for key, values := range opts.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set content type if determined from body
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set user agent (curl always sends a User-Agent header)
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	} else {
		// Default to "gocurl/VERSION" to match curl's behavior (curl/VERSION)
		req.Header.Set("User-Agent", "gocurl/"+Version)
	}

	// Set referer
	if opts.Referer != "" {
		req.Header.Set("Referer", opts.Referer)
	}
}

// applyAuth applies authentication to the request
func applyAuth(req *http.Request, opts *options.RequestOptions) {
	if opts.BasicAuth != nil {
		req.SetBasicAuth(opts.BasicAuth.Username, opts.BasicAuth.Password)
	}

	if opts.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.BearerToken)
	}
}

// applyCookies applies cookies to the request
func applyCookies(req *http.Request, opts *options.RequestOptions) {
	for _, cookie := range opts.Cookies {
		req.AddCookie(cookie)
	}
}

// applyCompression applies compression headers to the request
func applyCompression(req *http.Request, opts *options.RequestOptions) {
	if opts.Compress {
		acceptEncoding := getAcceptEncodingHeader(opts.Compress, opts.CompressionMethods)
		if acceptEncoding != "" && req.Header.Get("Accept-Encoding") == "" {
			req.Header.Set("Accept-Encoding", acceptEncoding)
		}
	}
}

// applyRequestID applies request ID header for distributed tracing
func applyRequestID(req *http.Request, opts *options.RequestOptions) {
	if opts.RequestID != "" {
		req.Header.Set("X-Request-ID", opts.RequestID)
	}
}

func applyMiddleware(req *http.Request, middleware []middlewares.MiddlewareFunc) (*http.Request, error) {
	var err error
	for _, mw := range middleware {
		req, err = mw(req)
		if err != nil {
			return nil, fmt.Errorf("middleware error: %v", err)
		}
	}
	return req, nil
}
