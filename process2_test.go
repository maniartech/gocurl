package gocurl_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostRequests(t *testing.T) {
	t.Run("POST with URL-encoded form data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			err := r.ParseForm()
			require.NoError(t, err)
			assert.Equal(t, "value1", r.Form.Get("key1"))
			assert.Equal(t, "value2", r.Form.Get("key2"))
			fmt.Fprint(w, "Form data received")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			Method: "POST",
			URL:    server.URL,
			Form: url.Values{
				"key1": {"value1"},
				"key2": {"value2"},
			},
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Form data received", body)
	})

	t.Run("POST with multipart form data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			err := r.ParseMultipartForm(10 << 20) // 10 MB
			require.NoError(t, err)
			assert.Equal(t, "value1", r.FormValue("key1"))
			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()
			assert.Equal(t, "test.txt", header.Filename)
			content, err := ioutil.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, "test file content", string(content))
			fmt.Fprint(w, "Multipart form data received")
		}))
		defer server.Close()

		// Create a temporary file for testing
		tmpfile, err := ioutil.TempFile("", "test.txt")
		require.NoError(t, err)
		defer os.Remove(tmpfile.Name()) // clean up

		_, err = tmpfile.Write([]byte("test file content"))
		require.NoError(t, err)
		err = tmpfile.Close()
		require.NoError(t, err)

		opts := &options.RequestOptions{
			Method: "POST",
			URL:    server.URL,
			FileUpload: &options.FileUpload{
				FieldName: "file",
				FileName:  "test.txt",
				FilePath:  tmpfile.Name(),
			},
			Form: url.Values{
				"key1": {"value1"},
			},
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Multipart form data received", body)
	})
}

func TestCustomTLSConfig(t *testing.T) {
	t.Run("Custom TLS configuration", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Secure connection established")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL:      server.URL,
			Insecure: true, // Required when using InsecureSkipVerify
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true, // Only for testing purposes
			},
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Secure connection established", body)
	})
}

func TestRedirectBehavior(t *testing.T) {
	t.Run("Follow redirects", func(t *testing.T) {
		redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Redirected successfully")
		}))
		defer redirectServer.Close()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, redirectServer.URL, http.StatusFound)
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL:             server.URL,
			FollowRedirects: true,
			MaxRedirects:    10,
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Redirected successfully", body)
	})

	t.Run("Do not follow redirects", func(t *testing.T) {
		redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "This should not be reached")
		}))
		defer redirectServer.Close()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, redirectServer.URL, http.StatusFound)
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL:             server.URL,
			FollowRedirects: false,
		}

		resp, _, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusFound, resp.StatusCode)
	})
}

func TestHTTPVersions(t *testing.T) {
	t.Run("HTTP/2 request", func(t *testing.T) {
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Protocol: %s", r.Proto)
		}))
		server.EnableHTTP2 = true
		server.StartTLS()
		defer server.Close()

		opts := &options.RequestOptions{
			URL:      server.URL,
			Insecure: true, // Required when using InsecureSkipVerify
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			HTTP2: true,
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, body, "HTTP/2.0")
	})
}

func TestCustomUserAgent(t *testing.T) {
	t.Run("Custom User-Agent string", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GoCurl/1.0", r.Header.Get("User-Agent"))
			fmt.Fprint(w, "Custom User-Agent received")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL:       server.URL,
			UserAgent: "GoCurl/1.0",
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Custom User-Agent received", body)
	})
}

func TestTimeoutBehavior(t *testing.T) {
	t.Run("Request timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			fmt.Fprint(w, "This should not be reached")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL:     server.URL,
			Timeout: 1 * time.Second,
		}

		_, _, err := gocurl.Process(context.Background(), opts)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout"))
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("Large response body", func(t *testing.T) {
		largeBody := strings.Repeat("a", 10*1024*1024) // 10 MB
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, largeBody)
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL: server.URL,
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, len(largeBody), len(body))
	})

	t.Run("Slow server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			fmt.Fprint(w, "Slow response")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL:     server.URL,
			Timeout: 200 * time.Millisecond,
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Slow response", body)
	})

	t.Run("Malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "10")
			fmt.Fprint(w, "Short")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL: server.URL,
		}

		_, _, err := gocurl.Process(context.Background(), opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected EOF")
	})
}
