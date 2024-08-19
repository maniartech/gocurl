package gocurl_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/stretchr/testify/assert"
)

func TestNewRequestOptions(t *testing.T) {
	opts := gocurl.NewRequestOptions()

	assert.Empty(t, opts.Method)
	assert.Empty(t, opts.URL)
	assert.NotNil(t, opts.Headers)
	assert.Empty(t, opts.Body)
	assert.NotNil(t, opts.Form)
	assert.NotNil(t, opts.QueryParams)
	assert.True(t, opts.FollowRedirects)
	assert.Equal(t, 10, opts.MaxRedirects)
	assert.True(t, opts.Compress)
	assert.NotNil(t, opts.Context)
}

func TestRequestOptions_Clone(t *testing.T) {
	opts := &gocurl.RequestOptions{
		Method:            "GET",
		URL:               "https://example.com",
		Headers:           http.Header{"Content-Type": []string{"application/json"}},
		Body:              "test body",
		Form:              url.Values{"key": []string{"value"}},
		QueryParams:       url.Values{"q": []string{"query"}},
		BasicAuth:         &gocurl.BasicAuth{Username: "user", Password: "pass"},
		BearerToken:       "token",
		CertFile:          "cert.pem",
		KeyFile:           "key.pem",
		CAFile:            "ca.pem",
		Insecure:          true,
		TLSConfig:         &tls.Config{},
		Proxy:             "http://proxy.com",
		Timeout:           30 * time.Second,
		ConnectTimeout:    10 * time.Second,
		FollowRedirects:   true,
		MaxRedirects:      5,
		Compress:          true,
		HTTP2:             true,
		HTTP2Only:         true,
		Cookies:           []*http.Cookie{{Name: "cookie", Value: "value"}},
		CookieJar:         nil,
		UserAgent:         "GoCurl",
		Referer:           "https://referer.com",
		FileUpload:        &gocurl.FileUpload{FieldName: "file", FileName: "file.txt", FilePath: "/path/to/file.txt"},
		RetryConfig:       &gocurl.RetryConfig{MaxRetries: 3, RetryDelay: 5 * time.Second, RetryOnHTTP: []int{500}},
		OutputFile:        "output.txt",
		Silent:            true,
		Verbose:           true,
		Context:           context.Background(),
		RequestID:         "request-id",
		Middleware:        []gocurl.MiddlewareFunc{},
		ResponseBodyLimit: 1024,
		ResponseDecoder:   nil,
		Metrics:           &gocurl.RequestMetrics{StartTime: time.Now(), Duration: 1 * time.Second, RetryCount: 1, ResponseSize: 1024},
	}

	clone := opts.Clone()

	assert.True(t, reflect.DeepEqual(opts, clone))
}

func TestRequestOptions_SetBasicAuth(t *testing.T) {
	opts := gocurl.NewRequestOptions()
	opts.SetBasicAuth("user", "pass")

	assert.NotNil(t, opts.BasicAuth)
	assert.Equal(t, "user", opts.BasicAuth.Username)
	assert.Equal(t, "pass", opts.BasicAuth.Password)
}

func TestRequestOptions_AddHeader(t *testing.T) {
	opts := gocurl.NewRequestOptions()
	opts.AddHeader("Content-Type", "application/json")

	assert.Equal(t, "application/json", opts.Headers.Get("Content-Type"))
}

func TestRequestOptions_SetHeader(t *testing.T) {
	opts := gocurl.NewRequestOptions()
	opts.SetHeader("Content-Type", "application/json")

	assert.Equal(t, "application/json", opts.Headers.Get("Content-Type"))
}

func TestRequestOptions_AddQueryParam(t *testing.T) {
	opts := gocurl.NewRequestOptions()
	opts.AddQueryParam("q", "query")

	assert.Equal(t, "query", opts.QueryParams.Get("q"))
}

func TestRequestOptions_SetQueryParam(t *testing.T) {
	opts := gocurl.NewRequestOptions()
	opts.SetQueryParam("q", "query")

	assert.Equal(t, "query", opts.QueryParams.Get("q"))
}
