package gocurl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Process executes the curl command based on the provided RequestOptions
func Process(ctx context.Context, opts *RequestOptions) (*http.Response, error) {
	// Validate options
	if err := ValidateOptions(opts); err != nil {
		return nil, err
	}

	// Create HTTP client
	client, err := CreateHTTPClient(opts)
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := CreateRequest(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Apply middleware
	req, err = ApplyMiddleware(req, opts.Middleware)
	if err != nil {
		return nil, err
	}

	// Execute request with retries
	resp, err := ExecuteRequestWithRetries(client, req, opts)
	if err != nil {
		return nil, err
	}

	// Handle output
	err = HandleOutput(resp, opts)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func ValidateOptions(opts *RequestOptions) error {
	if opts.URL == "" {
		return fmt.Errorf("URL is required")
	}
	// Add more validation as needed
	return nil
}

func CreateHTTPClient(opts *RequestOptions) (*http.Client, error) {
	transport := &http.Transport{
		TLSClientConfig:    opts.TLSConfig,
		DisableCompression: !opts.Compress,
		Proxy:              http.ProxyFromEnvironment,
	}

	if opts.Proxy != "" {
		proxyURL, err := url.Parse(opts.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %v", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
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

	if opts.CookieJar != nil {
		client.Jar = opts.CookieJar
	}

	return client, nil
}

func CreateRequest(ctx context.Context, opts *RequestOptions) (*http.Request, error) {
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
	if opts.Body != "" {
		body = strings.NewReader(opts.Body)
	} else if len(opts.Form) > 0 {
		body = strings.NewReader(opts.Form.Encode())
	} else if opts.FileUpload != nil {
		// Handle file upload
		// This is a simplified version. You might want to create a more robust multipart form data handler
		file, err := os.Open(opts.FileUpload.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for upload: %v", err)
		}
		defer file.Close()
		body = file
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

	return req, nil
}

func ApplyMiddleware(req *http.Request, middleware []MiddlewareFunc) (*http.Request, error) {
	var err error
	for _, mw := range middleware {
		req, err = mw(req)
		if err != nil {
			return nil, fmt.Errorf("middleware error: %v", err)
		}
	}
	return req, nil
}

func ExecuteRequestWithRetries(client *http.Client, req *http.Request, opts *RequestOptions) (*http.Response, error) {
	var resp *http.Response
	var err error

	retries := 0
	if opts.RetryConfig != nil {
		retries = opts.RetryConfig.MaxRetries
	}

	for i := 0; i <= retries; i++ {
		resp, err = client.Do(req)
		if err == nil {
			if opts.RetryConfig == nil || !shouldRetry(resp.StatusCode, opts.RetryConfig.RetryOnHTTP) {
				break
			}
		}

		if i < retries {
			time.Sleep(opts.RetryConfig.RetryDelay)
		}
	}

	return resp, err
}

func shouldRetry(statusCode int, retryOnHTTP []int) bool {
	for _, code := range retryOnHTTP {
		if statusCode == code {
			return true
		}
	}
	return false
}

func HandleOutput(resp *http.Response, opts *RequestOptions) error {
	if opts.OutputFile != "" {
		file, err := os.Create(opts.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write response to file: %v", err)
		}
	} else if !opts.Silent {
		_, err := io.Copy(os.Stdout, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write response to stdout: %v", err)
		}
	}

	return nil
}
