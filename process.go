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
	"golang.org/x/net/http2"
)

// Process executes the curl command based on the provided options.RequestOptions
// Process executes the curl command based on the provided options.RequestOptions
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
	// Validate options
	if err := ValidateOptions(opts); err != nil {
		return nil, "", err
	}

	// Use custom client if provided, otherwise create standard HTTP client
	var httpClient options.HTTPClient
	if opts.CustomClient != nil {
		httpClient = opts.CustomClient
	} else {
		client, err := CreateHTTPClient(ctx, opts)
		if err != nil {
			return nil, "", err
		}
		httpClient = client
	}

	// Create request
	req, err := CreateRequest(ctx, opts)
	if err != nil {
		return nil, "", err
	}

	// Apply middleware
	req, err = ApplyMiddleware(req, opts.Middleware)
	if err != nil {
		return nil, "", err
	}

	// Execute request with retries
	resp, err := executeWithRetries(httpClient, req, opts)
	if err != nil {
		return nil, "", err
	}

	// Print verbose response details (curl -v style)
	printResponseVerbose(opts, resp)

	// Decompress response if needed
	if opts.Compress {
		if err := DecompressResponse(resp); err != nil {
			resp.Body.Close()
			return nil, "", fmt.Errorf("failed to decompress response: %w", err)
		}
	}

	// Read the response body with optional size limit
	var bodyBytes []byte
	if opts.ResponseBodyLimit > 0 {
		// Use LimitReader to enforce size limit
		limitedReader := io.LimitReader(resp.Body, opts.ResponseBodyLimit+1) // +1 to detect overflow
		bodyBytes, err = io.ReadAll(limitedReader)
		if err != nil {
			resp.Body.Close()
			return nil, "", fmt.Errorf("failed to read response body: %v", err)
		}

		// Check if we hit or exceeded the limit
		if int64(len(bodyBytes)) > opts.ResponseBodyLimit {
			resp.Body.Close()
			return nil, "", fmt.Errorf("response body size (%d bytes) exceeds limit of %d bytes",
				len(bodyBytes)-1, opts.ResponseBodyLimit)
		}
	} else {
		// No limit - read entire response
		bodyBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, "", fmt.Errorf("failed to read response body: %v", err)
		}
	}

	resp.Body.Close()
	bodyString := string(bodyBytes)

	// Handle output
	err = HandleOutput(bodyString, opts)
	if err != nil {
		return nil, "", err
	}

	// Print verbose connection close info (curl -v style)
	printConnectionClose(opts)

	// Recreate the response body for further use
	resp.Body = io.NopCloser(strings.NewReader(bodyString))

	return resp, bodyString, nil
}

func ValidateOptions(opts *options.RequestOptions) error {
	// Use the new security validation
	return ValidateRequestOptions(opts)
}

func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
	// Load TLS configuration
	tlsConfig, err := LoadTLSConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Create base transport
	transport, err := createHTTPTransport(tlsConfig, opts)
	if err != nil {
		return nil, err
	}

	// Determine timeout based on context
	clientTimeout := determineClientTimeout(ctx, opts)

	// Create client with redirect policy
	client := createClientWithRedirects(transport, clientTimeout, opts)

	// Configure HTTP/2 if needed
	err = configureHTTP2(client, opts)
	if err != nil {
		return nil, err
	}

	// Configure cookie jar
	err = configureCookieJar(client, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// createHTTPTransport creates the base HTTP transport with proxy support
func createHTTPTransport(tlsConfig *tls.Config, opts *options.RequestOptions) (*http.Transport, error) {
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	// Configure compression
	ConfigureCompressionForTransport(transport, opts.Compress)

	// Handle proxy configuration
	if opts.Proxy != "" {
		proxyTransport, err := createProxyTransport(opts, tlsConfig)
		if err != nil {
			return nil, err
		}
		return proxyTransport, nil
	}

	return transport, nil
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

// createClientWithRedirects creates an HTTP client with redirect policy
func createClientWithRedirects(transport *http.Transport, timeout time.Duration, opts *options.RequestOptions) *http.Client {
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !opts.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= opts.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", opts.MaxRedirects)
			}
			return nil
		},
	}
}

// configureHTTP2 configures HTTP/2 support for the client
func configureHTTP2(client *http.Client, opts *options.RequestOptions) error {
	if !opts.HTTP2 && !opts.HTTP2Only {
		return nil
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		return nil // Proxy transport, skip HTTP/2 config
	}

	if opts.HTTP2Only {
		// HTTP/2 only mode
		http2Transport := &http2.Transport{
			TLSClientConfig: transport.TLSClientConfig,
		}
		client.Transport = http2Transport
		return nil
	}

	// Enable HTTP/2 with HTTP/1.1 fallback
	if err := http2.ConfigureTransport(transport); err != nil {
		return fmt.Errorf("failed to configure HTTP/2: %v", err)
	}

	return nil
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

	// Set user agent
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
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
