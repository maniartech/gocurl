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

	if requestOptions.Method != "GET" {
		t.Errorf("expected method to be GET, got %s", requestOptions.Method)
	}

	if requestOptions.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", requestOptions.URL)
	}

	if requestOptions.Headers.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header to be application/json, got %s", requestOptions.Headers.Get("Accept"))
	}

	if requestOptions.Body != "{\"key\":\"value\"}" {
		t.Errorf("expected body to be {\"key\":\"value\"}, got %s", requestOptions.Body)
	}

	if requestOptions.Form.Get("field") != "value" {
		t.Errorf("expected form field to be value, got %s", requestOptions.Form.Get("field"))
	}

	if requestOptions.QueryParams.Get("query") != "param" {
		t.Errorf("expected query param to be param, got %s", requestOptions.QueryParams.Get("query"))
	}

	if requestOptions.QueryParams.Get("additional") != "param" {
		t.Errorf("expected additional query param to be param, got %s", requestOptions.QueryParams.Get("additional"))
	}

	if requestOptions.BasicAuth == nil || requestOptions.BasicAuth.Username != "user" || requestOptions.BasicAuth.Password != "pass" {
		t.Errorf("expected basic auth credentials to be user:pass")
	}

	if requestOptions.BearerToken != "token123" {
		t.Errorf("expected bearer token to be token123, got %s", requestOptions.BearerToken)
	}

	if requestOptions.CertFile != "cert.pem" {
		t.Errorf("expected cert file to be cert.pem, got %s", requestOptions.CertFile)
	}

	if requestOptions.KeyFile != "key.pem" {
		t.Errorf("expected key file to be key.pem, got %s", requestOptions.KeyFile)
	}

	if requestOptions.CAFile != "ca.pem" {
		t.Errorf("expected CA file to be ca.pem, got %s", requestOptions.CAFile)
	}

	if !requestOptions.Insecure {
		t.Errorf("expected insecure to be true")
	}

	if requestOptions.Proxy != "http://proxy.example.com" {
		t.Errorf("expected proxy to be http://proxy.example.com, got %s", requestOptions.Proxy)
	}

	if requestOptions.Timeout != 30*time.Second {
		t.Errorf("expected timeout to be 30s, got %s", requestOptions.Timeout)
	}

	if requestOptions.ConnectTimeout != 10*time.Second {
		t.Errorf("expected connect timeout to be 10s, got %s", requestOptions.ConnectTimeout)
	}

	if !requestOptions.FollowRedirects {
		t.Errorf("expected follow redirects to be true")
	}

	if requestOptions.MaxRedirects != 5 {
		t.Errorf("expected max redirects to be 5, got %d", requestOptions.MaxRedirects)
	}

	if !requestOptions.Compress {
		t.Errorf("expected compress to be true")
	}

	if !requestOptions.HTTP2 {
		t.Errorf("expected HTTP2 to be true")
	}

	if requestOptions.HTTP2Only {
		t.Errorf("expected HTTP2Only to be false")
	}

	if len(requestOptions.Cookies) != 1 || requestOptions.Cookies[0].Name != "session_id" || requestOptions.Cookies[0].Value != "abc123" {
		t.Errorf("expected cookie session_id=abc123")
	}

	if requestOptions.UserAgent != "TestAgent" {
		t.Errorf("expected User-Agent to be TestAgent, got %s", requestOptions.UserAgent)
	}

	if requestOptions.Referer != "https://referer.example.com" {
		t.Errorf("expected Referer to be https://referer.example.com, got %s", requestOptions.Referer)
	}

	if requestOptions.FileUpload == nil || requestOptions.FileUpload.FilePath != "file.txt" || requestOptions.FileUpload.FieldName != "upload" {
		t.Errorf("expected file upload to be file.txt with field name upload")
	}

	if requestOptions.RetryConfig == nil || requestOptions.RetryConfig.MaxRetries != 3 || requestOptions.RetryConfig.RetryDelay != 2*time.Second {
		t.Errorf("expected retry config with max retries 3 and interval 2s")
	}

	if requestOptions.OutputFile != "output.txt" {
		t.Errorf("expected output file to be output.txt, got %s", requestOptions.OutputFile)
	}

	if requestOptions.Silent {
		t.Errorf("expected silent to be false")
	}

	if !requestOptions.Verbose {
		t.Errorf("expected verbose to be true")
	}
}

func TestScenarioOrientedMethods(t *testing.T) {
	// Test POST method
	builder := options.NewRequestOptionsBuilder()
	requestOptions := builder.Post("https://example.com", "{\"key\":\"value\"}", http.Header{"Content-Type": []string{"application/json"}}).Build()

	if requestOptions.Method != "POST" {
		t.Errorf("expected method to be POST, got %s", requestOptions.Method)
	}

	if requestOptions.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", requestOptions.URL)
	}

	if requestOptions.Body != "{\"key\":\"value\"}" {
		t.Errorf("expected body to be {\"key\":\"value\"}, got %s", requestOptions.Body)
	}

	if requestOptions.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type header to be application/json, got %s", requestOptions.Headers.Get("Content-Type"))
	}

	// Test GET method
	builder = options.NewRequestOptionsBuilder()
	requestOptions = builder.Get("https://example.com", http.Header{"Accept": []string{"application/json"}}).Build()

	if requestOptions.Method != "GET" {
		t.Errorf("expected method to be GET, got %s", requestOptions.Method)
	}

	if requestOptions.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", requestOptions.URL)
	}

	if requestOptions.Headers.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header to be application/json, got %s", requestOptions.Headers.Get("Accept"))
	}

	// Test PUT method
	builder = options.NewRequestOptionsBuilder()
	requestOptions = builder.Put("https://example.com", "{\"key\":\"value\"}", http.Header{"Content-Type": []string{"application/json"}}).Build()

	if requestOptions.Method != "PUT" {
		t.Errorf("expected method to be PUT, got %s", requestOptions.Method)
	}

	if requestOptions.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", requestOptions.URL)
	}

	if requestOptions.Body != "{\"key\":\"value\"}" {
		t.Errorf("expected body to be {\"key\":\"value\"}, got %s", requestOptions.Body)
	}

	if requestOptions.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type header to be application/json, got %s", requestOptions.Headers.Get("Content-Type"))
	}

	// Test DELETE method
	builder = options.NewRequestOptionsBuilder()
	requestOptions = builder.Delete("https://example.com", http.Header{"Accept": []string{"application/json"}}).Build()

	if requestOptions.Method != "DELETE" {
		t.Errorf("expected method to be DELETE, got %s", requestOptions.Method)
	}

	if requestOptions.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", requestOptions.URL)
	}

	if requestOptions.Headers.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header to be application/json, got %s", requestOptions.Headers.Get("Accept"))
	}

	// Test PATCH method
	builder = options.NewRequestOptionsBuilder()
	requestOptions = builder.Patch("https://example.com", "{\"key\":\"value\"}", http.Header{"Content-Type": []string{"application/json"}}).Build()

	if requestOptions.Method != "PATCH" {
		t.Errorf("expected method to be PATCH, got %s", requestOptions.Method)
	}

	if requestOptions.URL != "https://example.com" {
		t.Errorf("expected URL to be https://example.com, got %s", requestOptions.URL)
	}

	if requestOptions.Body != "{\"key\":\"value\"}" {
		t.Errorf("expected body to be {\"key\":\"value\"}, got %s", requestOptions.Body)
	}

	if requestOptions.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type header to be application/json, got %s", requestOptions.Headers.Get("Content-Type"))
	}
}

func TestSetHeaders(t *testing.T) {
	builder := options.NewRequestOptionsBuilder()
	requestOptions := builder.SetHeaders(http.Header{"Accept": []string{"application/json"}}).Build()

	if requestOptions.Headers.Get("Accept") != "application/json" {
		t.Errorf("expected Accept header to be application/json, got %s", requestOptions.Headers.Get("Accept"))
	}
}
