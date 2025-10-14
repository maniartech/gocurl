package gocurl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/maniartech/gocurl/middlewares"
	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/proxy"
	"github.com/maniartech/gocurl/tokenizer"
	"golang.org/x/net/http2"
)

func Curl(ctx context.Context, command string) (*http.Response, string, error) {
	tokenizer := tokenizer.NewTokenizer()

	err := tokenizer.Tokenize(command)
	if err != nil {
		return nil, "", err
	}

	tokens := tokenizer.GetTokens()

	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		return nil, "", err
	}

	return Process(ctx, opts)
}

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
		bodyBytes, err = ioutil.ReadAll(limitedReader)
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
		bodyBytes, err = ioutil.ReadAll(resp.Body)
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

	// Recreate the response body for further use
	resp.Body = ioutil.NopCloser(strings.NewReader(bodyString))

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

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	// Configure compression - disable auto compression to handle manually with pooled readers
	ConfigureCompressionForTransport(transport, opts.Compress)

	// Handle proxy configuration
	if opts.Proxy != "" {
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

		// Apply proxy to transport
		proxyTransport, err := proxy.NewTransport(proxyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy transport: %w", err)
		}

		transport = proxyTransport
	}

	// INDUSTRY STANDARD: Context Priority Pattern
	// Determine timeout based on context deadline vs opts.Timeout
	var clientTimeout time.Duration

	if ctx != nil {
		// If context has a deadline, prefer it over opts.Timeout
		if _, hasDeadline := ctx.Deadline(); hasDeadline {
			// Don't set client.Timeout, let context handle it
			// This prevents nested timeouts and unpredictable behavior
			clientTimeout = 0
		} else {
			// No deadline in context, use opts.Timeout as fallback
			clientTimeout = opts.Timeout
		}
	} else {
		// No context provided, use opts.Timeout
		clientTimeout = opts.Timeout
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   clientTimeout, // Set based on context priority pattern
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

	// Add HTTP/2 support based on the options
	if opts.HTTP2 || opts.HTTP2Only {
		// If HTTP2Only is set, create a new HTTP/2 transport
		if opts.HTTP2Only {
			http2Transport := &http2.Transport{
				TLSClientConfig: transport.TLSClientConfig,
			}
			client.Transport = http2Transport
		} else {
			// Enable HTTP/2 support if possible, while still allowing fallback to HTTP/1.1
			if err := http2.ConfigureTransport(transport); err != nil {
				return nil, fmt.Errorf("failed to configure HTTP/2: %v", err)
			}
		}
	}

	// Handle cookie jar
	if opts.CookieFile != "" {
		// Create persistent cookie jar
		jar, err := NewPersistentCookieJar(opts.CookieFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create cookie jar: %w", err)
		}
		client.Jar = jar
	} else if opts.CookieJar != nil {
		client.Jar = opts.CookieJar
	}

	return client, nil
}

func CreateRequest(ctx context.Context, opts *options.RequestOptions) (*http.Request, error) {
	method := opts.Method
	if method == "" {
		method = "GET"
	}

	url := opts.URL
	if len(opts.QueryParams) > 0 {
		if strings.Contains(url, "?") {
			url += "&" + opts.QueryParams.Encode()
		} else {
			url += "?" + opts.QueryParams.Encode()
		}
	}

	var body io.Reader
	var contentType string

	if opts.Body != "" {
		body = strings.NewReader(opts.Body)
	} else if len(opts.Form) > 0 && opts.FileUpload == nil {
		// URL-encoded form data
		body = strings.NewReader(opts.Form.Encode())
		contentType = "application/x-www-form-urlencoded"
	} else if opts.FileUpload != nil {
		// Multipart form data
		var b bytes.Buffer
		w := multipart.NewWriter(&b)

		// Add file
		file, err := os.Open(opts.FileUpload.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for upload: %v", err)
		}
		defer file.Close()

		part, err := w.CreateFormFile(opts.FileUpload.FieldName, opts.FileUpload.FileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %v", err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file content: %v", err)
		}

		// Add other form fields
		for key, values := range opts.Form {
			for _, value := range values {
				err := w.WriteField(key, value)
				if err != nil {
					return nil, fmt.Errorf("failed to write form field: %v", err)
				}
			}
		}

		err = w.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to close multipart writer: %v", err)
		}

		body = &b
		contentType = w.FormDataContentType()
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, values := range opts.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set content type if not already set
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set basic auth
	if opts.BasicAuth != nil {
		req.SetBasicAuth(opts.BasicAuth.Username, opts.BasicAuth.Password)
	}

	// Set bearer token
	if opts.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+opts.BearerToken)
	}

	// Set user agent
	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	// Set referer
	if opts.Referer != "" {
		req.Header.Set("Referer", opts.Referer)
	}

	// Set Accept-Encoding if compression is enabled
	if opts.Compress {
		acceptEncoding := GetAcceptEncodingHeader(opts.Compress, opts.CompressionMethods)
		if acceptEncoding != "" && req.Header.Get("Accept-Encoding") == "" {
			req.Header.Set("Accept-Encoding", acceptEncoding)
		}
	}

	return req, nil
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
		err := ioutil.WriteFile(opts.OutputFile, []byte(body), 0644)
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
