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

// Process executes a request and buffers the full response body, returning it as
// a string and re-wrapping resp.Body so it can be read again.
//
// Deprecated: Process buffers the entire response in memory and writes to the
// configured OutputFile/stdout as a side effect. Prefer the Curl* functions,
// which stream the live response body and never touch stdout. Process is
// retained for backward compatibility.
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
	resp, err := doRequest(ctx, opts)
	if err != nil {
		return nil, "", err
	}

	bodyBytes, err := readBodyWithLimit(resp.Body, opts.ResponseBodyLimit)
	resp.Body.Close()
	if err != nil {
		return nil, "", err
	}
	bodyString := string(bodyBytes)

	// Handle output (OutputFile / stdout) — a Process-only side effect.
	if err := HandleOutput(bodyString, opts); err != nil {
		return nil, "", err
	}

	printConnectionClose(opts)

	// Recreate the response body so callers can read it again.
	resp.Body = io.NopCloser(strings.NewReader(bodyString))

	return resp, bodyString, nil
}

// doRequest runs the shared request pipeline (validate, client, build, retries,
// verbose, decompress) and returns the live response with its body unread and
// open. It performs NO output side effects, so library callers control the body.
func doRequest(ctx context.Context, opts *options.RequestOptions) (*http.Response, error) {
	if err := ValidateOptions(opts); err != nil {
		return nil, err
	}

	var httpClient options.HTTPClient
	if opts.CustomClient != nil {
		httpClient = opts.CustomClient
	} else {
		client, err := CreateHTTPClient(ctx, opts)
		if err != nil {
			return nil, err
		}
		httpClient = client
	}

	req, err := CreateRequest(ctx, opts)
	if err != nil {
		return nil, err
	}

	req, err = ApplyMiddleware(req, opts.Middleware)
	if err != nil {
		return nil, err
	}

	resp, err := executeWithRetries(httpClient, req, opts)
	if err != nil {
		return nil, wrapTransportError(opts.URL, err)
	}

	printResponseVerbose(opts, resp)

	if opts.Compress {
		if err := DecompressResponse(resp); err != nil {
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

// readBodyWithLimit reads body fully, enforcing an optional size limit.
func readBodyWithLimit(body io.Reader, limit int64) ([]byte, error) {
	if limit > 0 {
		limited := io.LimitReader(body, limit+1) // +1 to detect overflow
		b, err := io.ReadAll(limited)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}
		if int64(len(b)) > limit {
			return nil, fmt.Errorf("response body size (%d bytes) exceeds limit of %d bytes",
				len(b)-1, limit)
		}
		return b, nil
	}
	return io.ReadAll(body)
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
	n, err := l.rc.Read(p)
	l.read += int64(n)
	if l.read > l.limit {
		return n, fmt.Errorf("response body size exceeds limit of %d bytes", l.limit)
	}
	return n, err
}

func (l *limitedBody) Close() error { return l.rc.Close() }

func ValidateOptions(opts *options.RequestOptions) error {
	// Use the new security validation
	return ValidateRequestOptions(opts)
}

func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
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
			return fmt.Errorf("stopped after %d redirects", opts.MaxRedirects)
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

func CreateRequest(ctx context.Context, opts *options.RequestOptions) (*http.Request, error) {
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
		acceptEncoding := GetAcceptEncodingHeader(opts.Compress, opts.CompressionMethods)
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

func ApplyMiddleware(req *http.Request, middleware []middlewares.MiddlewareFunc) (*http.Request, error) {
	var err error
	for _, mw := range middleware {
		req, err = mw(req)
		if err != nil {
			return nil, fmt.Errorf("middleware error: %v", err)
		}
	}
	return req, nil
}

func HandleOutput(body string, opts *options.RequestOptions) error {
	if opts.OutputFile != "" {
		err := os.WriteFile(opts.OutputFile, []byte(body), 0644)
		if err != nil {
			return fmt.Errorf("failed to write response to file: %v", err)
		}
	} else if !opts.Silent {
		_, err := fmt.Fprint(os.Stdout, body)
		if err != nil {
			return fmt.Errorf("failed to write response to stdout: %v", err)
		}
	}

	return nil

}
