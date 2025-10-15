package options_test

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

func TestRequestOptionsBuilder(t *testing.T) {
	builder := options.NewRequestOptionsBuilder()
	requestOptions := builder.
		SetMethod("GET").
		SetURL("https://example.com").
		AddHeader("Accept", "application/json").
		SetBody("{\"key\":\"value\"}").
		SetForm(url.Values{"field": []string{"value"}}).
		SetQueryParams(url.Values{"query": []string{"param"}}).
		AddQueryParam("additional", "param").
		SetBasicAuth("user", "pass").
		SetBearerToken("token123").
		SetCertFile("cert.pem").
		SetKeyFile("key.pem").
		SetCAFile("ca.pem").
		SetInsecure(true).
		SetTLSConfig(&tls.Config{InsecureSkipVerify: true}).
		SetProxy("http://proxy.example.com").
		SetTimeout(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetFollowRedirects(true).
		SetMaxRedirects(5).
		SetCompress(true).
		SetHTTP2(true).
		SetHTTP2Only(false).
		SetCookie(&http.Cookie{Name: "session_id", Value: "abc123"}).
		SetUserAgent("TestAgent").
		SetReferer("https://referer.example.com").
		SetFileUpload(&options.FileUpload{FilePath: "file.txt", FieldName: "upload"}).
		SetRetryConfig(&options.RetryConfig{MaxRetries: 3, RetryDelay: 2 * time.Second}).
		SetOutputFile("output.txt").
		SetSilent(false).
		SetVerbose(true).
		Build()

	// Test basic fields
	testBasicFields(t, requestOptions)

	// Test headers and body
	testHeadersAndBody(t, requestOptions)

	// Test form and query params
	testFormAndQueryParams(t, requestOptions)

	// Test authentication
	testAuthentication(t, requestOptions)

	// Test TLS configuration
	testTLSConfiguration(t, requestOptions)

	// Test network settings
	testNetworkSettings(t, requestOptions)

	// Test HTTP settings
	testHTTPSettings(t, requestOptions)

	// Test additional options
	testAdditionalOptions(t, requestOptions)
}

// testBasicFields validates basic request options
func testBasicFields(t *testing.T, opts *options.RequestOptions) {
	if opts.Method != "GET" {
		t.Errorf("expected method to be GET, got %s", opts.Method)
	}
	if opts.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", opts.URL)
	}
}

// testHeadersAndBody validates headers and body
func testHeadersAndBody(t *testing.T, opts *options.RequestOptions) {
	if opts.Headers.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header to be application/json, got %s", opts.Headers.Get("Accept"))
	}
	if opts.Body != "{\"key\":\"value\"}" {
		t.Errorf("expected body to be {\"key\":\"value\"}, got %s", opts.Body)
	}
}

// testFormAndQueryParams validates form fields and query parameters
func testFormAndQueryParams(t *testing.T, opts *options.RequestOptions) {
	if opts.Form.Get("field") != "value" {
		t.Errorf("expected form field to be value, got %s", opts.Form.Get("field"))
	}
	if opts.QueryParams.Get("query") != "param" {
		t.Errorf("expected query param to be param, got %s", opts.QueryParams.Get("query"))
	}
	if opts.QueryParams.Get("additional") != "param" {
		t.Errorf("expected additional query param to be param, got %s", opts.QueryParams.Get("additional"))
	}
}

// testAuthentication validates authentication settings
func testAuthentication(t *testing.T, opts *options.RequestOptions) {
	if opts.BasicAuth == nil || opts.BasicAuth.Username != "user" || opts.BasicAuth.Password != "pass" {
		t.Errorf("expected basic auth credentials to be user:pass")
	}
	if opts.BearerToken != "token123" {
		t.Errorf("expected bearer token to be token123, got %s", opts.BearerToken)
	}
}

// testTLSConfiguration validates TLS settings
func testTLSConfiguration(t *testing.T, opts *options.RequestOptions) {
	if opts.CertFile != "cert.pem" {
		t.Errorf("expected cert file to be cert.pem, got %s", opts.CertFile)
	}
	if opts.KeyFile != "key.pem" {
		t.Errorf("expected key file to be key.pem, got %s", opts.KeyFile)
	}
	if opts.CAFile != "ca.pem" {
		t.Errorf("expected CA file to be ca.pem, got %s", opts.CAFile)
	}
	if !opts.Insecure {
		t.Errorf("expected insecure to be true")
	}
}

// testNetworkSettings validates proxy and timeout settings
func testNetworkSettings(t *testing.T, opts *options.RequestOptions) {
	if opts.Proxy != "http://proxy.example.com" {
		t.Errorf("expected proxy to be http://proxy.example.com, got %s", opts.Proxy)
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("expected timeout to be 30s, got %s", opts.Timeout)
	}
	if opts.ConnectTimeout != 10*time.Second {
		t.Errorf("expected connect timeout to be 10s, got %s", opts.ConnectTimeout)
	}
}

// testHTTPSettings validates HTTP-specific settings
func testHTTPSettings(t *testing.T, opts *options.RequestOptions) {
	if !opts.FollowRedirects {
		t.Errorf("expected follow redirects to be true")
	}
	if opts.MaxRedirects != 5 {
		t.Errorf("expected max redirects to be 5, got %d", opts.MaxRedirects)
	}
	if !opts.Compress {
		t.Errorf("expected compress to be true")
	}
	if !opts.HTTP2 {
		t.Errorf("expected HTTP2 to be true")
	}
	if opts.HTTP2Only {
		t.Errorf("expected HTTP2Only to be false")
	}
}

// testAdditionalOptions validates cookies, user agent, and other options
func testAdditionalOptions(t *testing.T, opts *options.RequestOptions) {
	if len(opts.Cookies) != 1 || opts.Cookies[0].Name != "session_id" || opts.Cookies[0].Value != "abc123" {
		t.Errorf("expected cookie session_id=abc123")
	}
	if opts.UserAgent != "TestAgent" {
		t.Errorf("expected User-Agent to be TestAgent, got %s", opts.UserAgent)
	}
	if opts.Referer != "https://referer.example.com" {
		t.Errorf("expected Referer to be https://referer.example.com, got %s", opts.Referer)
	}
	if opts.FileUpload == nil || opts.FileUpload.FilePath != "file.txt" || opts.FileUpload.FieldName != "upload" {
		t.Errorf("expected file upload to be file.txt with field name upload")
	}
	if opts.RetryConfig == nil || opts.RetryConfig.MaxRetries != 3 || opts.RetryConfig.RetryDelay != 2*time.Second {
		t.Errorf("expected retry config with max retries 3 and interval 2s")
	}
	if opts.OutputFile != "output.txt" {
		t.Errorf("expected output file to be output.txt, got %s", opts.OutputFile)
	}
	if opts.Silent {
		t.Errorf("expected silent to be false")
	}
	if !opts.Verbose {
		t.Errorf("expected verbose to be true")
	}
}

func TestScenarioOrientedMethods(t *testing.T) {
	tests := []struct {
		name           string
		setupBuilder   func() *options.RequestOptionsBuilder
		expectedMethod string
		expectedURL    string
		expectedBody   string
		expectedHeader string
		headerValue    string
	}{
		{
			name: "POST method",
			setupBuilder: func() *options.RequestOptionsBuilder {
				return options.NewRequestOptionsBuilder().Post("https://example.com", "{\"key\":\"value\"}", http.Header{"Content-Type": []string{"application/json"}})
			},
			expectedMethod: "POST",
			expectedURL:    "https://example.com",
			expectedBody:   "{\"key\":\"value\"}",
			expectedHeader: "Content-Type",
			headerValue:    "application/json",
		},
		{
			name: "GET method",
			setupBuilder: func() *options.RequestOptionsBuilder {
				return options.NewRequestOptionsBuilder().Get("https://example.com", http.Header{"Accept": []string{"application/json"}})
			},
			expectedMethod: "GET",
			expectedURL:    "https://example.com",
			expectedBody:   "",
			expectedHeader: "Accept",
			headerValue:    "application/json",
		},
		{
			name: "PUT method",
			setupBuilder: func() *options.RequestOptionsBuilder {
				return options.NewRequestOptionsBuilder().Put("https://example.com", "{\"key\":\"value\"}", http.Header{"Content-Type": []string{"application/json"}})
			},
			expectedMethod: "PUT",
			expectedURL:    "https://example.com",
			expectedBody:   "{\"key\":\"value\"}",
			expectedHeader: "Content-Type",
			headerValue:    "application/json",
		},
		{
			name: "DELETE method",
			setupBuilder: func() *options.RequestOptionsBuilder {
				return options.NewRequestOptionsBuilder().Delete("https://example.com", http.Header{"Accept": []string{"application/json"}})
			},
			expectedMethod: "DELETE",
			expectedURL:    "https://example.com",
			expectedBody:   "",
			expectedHeader: "Accept",
			headerValue:    "application/json",
		},
		{
			name: "PATCH method",
			setupBuilder: func() *options.RequestOptionsBuilder {
				return options.NewRequestOptionsBuilder().Patch("https://example.com", "{\"key\":\"value\"}", http.Header{"Content-Type": []string{"application/json"}})
			},
			expectedMethod: "PATCH",
			expectedURL:    "https://example.com",
			expectedBody:   "{\"key\":\"value\"}",
			expectedHeader: "Content-Type",
			headerValue:    "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setupBuilder().Build()

			if opts.Method != tt.expectedMethod {
				t.Errorf("expected method to be %s, got %s", tt.expectedMethod, opts.Method)
			}
			if opts.URL != tt.expectedURL {
				t.Errorf("expected URL to be %s, got %s", tt.expectedURL, opts.URL)
			}
			if opts.Body != tt.expectedBody {
				t.Errorf("expected body to be %s, got %s", tt.expectedBody, opts.Body)
			}
			if opts.Headers.Get(tt.expectedHeader) != tt.headerValue {
				t.Errorf("expected %s header to be %s, got %s", tt.expectedHeader, tt.headerValue, opts.Headers.Get(tt.expectedHeader))
			}
		})
	}
}

func TestSetHeaders(t *testing.T) {
	builder := options.NewRequestOptionsBuilder()
	requestOptions := builder.SetHeaders(http.Header{"Accept": []string{"application/json"}}).Build()

	if requestOptions.Headers.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header to be application/json, got %s", requestOptions.Headers.Get("Accept"))
	}
}
