package options

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// RequestOptionsBuilder is a builder for RequestOptions.
type RequestOptionsBuilder struct {
	options *RequestOptions
}

// NewRequestOptionsBuilder creates a new instance of RequestOptionsBuilder.
func NewRequestOptionsBuilder() *RequestOptionsBuilder {
	return &RequestOptionsBuilder{
		options: &RequestOptions{
			Headers:     http.Header{},
			Form:        url.Values{},
			QueryParams: url.Values{},
		},
	}
}

// SetMethod sets the HTTP method.
func (b *RequestOptionsBuilder) SetMethod(method string) *RequestOptionsBuilder {
	b.options.Method = method
	return b
}

// SetURL sets the request URL.
func (b *RequestOptionsBuilder) SetURL(url string) *RequestOptionsBuilder {
	b.options.URL = url
	return b
}

// AddHeader adds a header to the request.
func (b *RequestOptionsBuilder) AddHeader(key, value string) *RequestOptionsBuilder {
	b.options.Headers.Add(key, value)
	return b
}

// SetHeaders sets multiple headers for the request.
func (b *RequestOptionsBuilder) SetHeaders(headers http.Header) *RequestOptionsBuilder {
	b.options.Headers = headers
	return b
}

// SetBody sets the request body.
func (b *RequestOptionsBuilder) SetBody(body string) *RequestOptionsBuilder {
	b.options.Body = body
	return b
}

// SetForm sets the form data for the request.
func (b *RequestOptionsBuilder) SetForm(form url.Values) *RequestOptionsBuilder {
	b.options.Form = form
	return b
}

// SetQueryParams sets the query parameters for the request.
func (b *RequestOptionsBuilder) SetQueryParams(queryParams url.Values) *RequestOptionsBuilder {
	b.options.QueryParams = queryParams
	return b
}

// AddQueryParam adds a query parameter to the request.
func (b *RequestOptionsBuilder) AddQueryParam(key, value string) *RequestOptionsBuilder {
	b.options.QueryParams.Add(key, value)
	return b
}

// SetBasicAuth sets the basic authentication credentials.
func (b *RequestOptionsBuilder) SetBasicAuth(username, password string) *RequestOptionsBuilder {
	b.options.BasicAuth = &BasicAuth{
		Username: username,
		Password: password,
	}
	return b
}

// SetBearerToken sets the bearer token for authentication.
func (b *RequestOptionsBuilder) SetBearerToken(token string) *RequestOptionsBuilder {
	b.options.BearerToken = token
	return b
}

// SetCertFile sets the certificate file for TLS.
func (b *RequestOptionsBuilder) SetCertFile(certFile string) *RequestOptionsBuilder {
	b.options.CertFile = certFile
	return b
}

// SetKeyFile sets the key file for TLS.
func (b *RequestOptionsBuilder) SetKeyFile(keyFile string) *RequestOptionsBuilder {
	b.options.KeyFile = keyFile
	return b
}

// SetCAFile sets the CA file for TLS.
func (b *RequestOptionsBuilder) SetCAFile(caFile string) *RequestOptionsBuilder {
	b.options.CAFile = caFile
	return b
}

// SetInsecure sets whether to skip TLS verification.
func (b *RequestOptionsBuilder) SetInsecure(insecure bool) *RequestOptionsBuilder {
	b.options.Insecure = insecure
	return b
}

// SetTLSConfig sets the TLS configuration.
func (b *RequestOptionsBuilder) SetTLSConfig(tlsConfig *tls.Config) *RequestOptionsBuilder {
	b.options.TLSConfig = tlsConfig
	return b
}

// SetProxy sets the proxy URL.
func (b *RequestOptionsBuilder) SetProxy(proxy string) *RequestOptionsBuilder {
	b.options.Proxy = proxy
	return b
}

// SetTimeout sets the request timeout.
func (b *RequestOptionsBuilder) SetTimeout(timeout time.Duration) *RequestOptionsBuilder {
	b.options.Timeout = timeout
	return b
}

// SetConnectTimeout sets the connect timeout for the request.
func (b *RequestOptionsBuilder) SetConnectTimeout(connectTimeout time.Duration) *RequestOptionsBuilder {
	b.options.ConnectTimeout = connectTimeout
	return b
}

// SetFollowRedirects sets whether to follow redirects.
func (b *RequestOptionsBuilder) SetFollowRedirects(follow bool) *RequestOptionsBuilder {
	b.options.FollowRedirects = follow
	return b
}

// SetMaxRedirects sets the maximum number of redirects to follow.
func (b *RequestOptionsBuilder) SetMaxRedirects(maxRedirects int) *RequestOptionsBuilder {
	b.options.MaxRedirects = maxRedirects
	return b
}

// SetCompress sets whether to enable compression.
func (b *RequestOptionsBuilder) SetCompress(compress bool) *RequestOptionsBuilder {
	b.options.Compress = compress
	return b
}

// SetHTTP2 enables or disables HTTP/2.
func (b *RequestOptionsBuilder) SetHTTP2(http2 bool) *RequestOptionsBuilder {
	b.options.HTTP2 = http2
	return b
}

// SetHTTP2Only enables or disables HTTP/2-only mode.
func (b *RequestOptionsBuilder) SetHTTP2Only(http2Only bool) *RequestOptionsBuilder {
	b.options.HTTP2Only = http2Only
	return b
}

// SetCookie adds a cookie to the request.
func (b *RequestOptionsBuilder) SetCookie(cookie *http.Cookie) *RequestOptionsBuilder {
	b.options.Cookies = append(b.options.Cookies, cookie)
	return b
}

// SetUserAgent sets the User-Agent header.
func (b *RequestOptionsBuilder) SetUserAgent(userAgent string) *RequestOptionsBuilder {
	b.options.UserAgent = userAgent
	return b
}

// SetReferer sets the Referer header.
func (b *RequestOptionsBuilder) SetReferer(referer string) *RequestOptionsBuilder {
	b.options.Referer = referer
	return b
}

// SetFileUpload sets the file upload configuration.
func (b *RequestOptionsBuilder) SetFileUpload(fileUpload *FileUpload) *RequestOptionsBuilder {
	b.options.FileUpload = fileUpload
	return b
}

// SetRetryConfig sets the retry configuration.
func (b *RequestOptionsBuilder) SetRetryConfig(retryConfig *RetryConfig) *RequestOptionsBuilder {
	b.options.RetryConfig = retryConfig
	return b
}

// SetOutputFile sets the output file for the response.
func (b *RequestOptionsBuilder) SetOutputFile(outputFile string) *RequestOptionsBuilder {
	b.options.OutputFile = outputFile
	return b
}

// SetSilent sets whether the request should be silent.
func (b *RequestOptionsBuilder) SetSilent(silent bool) *RequestOptionsBuilder {
	b.options.Silent = silent
	return b
}

// SetVerbose sets whether the request should be verbose.
func (b *RequestOptionsBuilder) SetVerbose(verbose bool) *RequestOptionsBuilder {
	b.options.Verbose = verbose
	return b
}

// POST creates a POST request with the given URL, body, and headers.
func (b *RequestOptionsBuilder) POST(url string, body string, headers http.Header) *RequestOptionsBuilder {
	b.options.Method = "POST"
	b.options.URL = url
	b.options.Body = body
	b.options.Headers = headers
	return b
}

// GET creates a GET request with the given URL and headers.
func (b *RequestOptionsBuilder) GET(url string, headers http.Header) *RequestOptionsBuilder {
	b.options.Method = "GET"
	b.options.URL = url
	b.options.Headers = headers
	return b
}

// PUT creates a PUT request with the given URL, body, and headers.
func (b *RequestOptionsBuilder) PUT(url string, body string, headers http.Header) *RequestOptionsBuilder {
	b.options.Method = "PUT"
	b.options.URL = url
	b.options.Body = body
	b.options.Headers = headers
	return b
}

// DELETE creates a DELETE request with the given URL and headers.
func (b *RequestOptionsBuilder) DELETE(url string, headers http.Header) *RequestOptionsBuilder {
	b.options.Method = "DELETE"
	b.options.URL = url
	b.options.Headers = headers
	return b
}

// PATCH creates a PATCH request with the given URL, body, and headers.
func (b *RequestOptionsBuilder) PATCH(url string, body string, headers http.Header) *RequestOptionsBuilder {
	b.options.Method = "PATCH"
	b.options.URL = url
	b.options.Body = body
	b.options.Headers = headers
	return b
}

// Build returns the configured RequestOptions instance.
func (b *RequestOptionsBuilder) Build() *RequestOptions {
	return b.options.Clone()
}

// Example usage
func example() {
	builder := NewRequestOptionsBuilder()
	requestOptions := builder.
		POST("https://example.com", "{\"name\":\"example\"}", http.Header{"Content-Type": []string{"application/json"}}).
		SetTimeout(30 * time.Second).
		SetFollowRedirects(true).
		SetVerbose(true).
		Build()

	// Use requestOptions as needed
	_ = requestOptions
}
