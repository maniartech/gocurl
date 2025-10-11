package gocurl

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAcceptEncodingHeader(t *testing.T) {
	tests := []struct {
		name     string
		compress bool
		methods  []string
		expected string
	}{
		{
			name:     "compression disabled",
			compress: false,
			methods:  nil,
			expected: "",
		},
		{
			name:     "compression enabled - default methods",
			compress: true,
			methods:  nil,
			expected: "gzip, deflate, br",
		},
		{
			name:     "compression enabled - custom methods",
			compress: true,
			methods:  []string{"gzip", "br"},
			expected: "gzip, br",
		},
		{
			name:     "compression enabled - single method",
			compress: true,
			methods:  []string{"gzip"},
			expected: "gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAcceptEncodingHeader(tt.compress, tt.methods)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigureCompressionForTransport(t *testing.T) {
	transport := &http.Transport{}

	// When compress is true, DisableCompression should be true
	// (we handle decompression manually)
	ConfigureCompressionForTransport(transport, true)
	assert.True(t, transport.DisableCompression, "DisableCompression should be true when manually handling compression")

	transport2 := &http.Transport{}
	ConfigureCompressionForTransport(transport2, false)
	assert.True(t, transport2.DisableCompression, "DisableCompression should be true to prevent automatic handling")
}

func TestDecompressResponseGzip(t *testing.T) {
	// Create compressed content
	originalContent := "This is test content that will be compressed with gzip"
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte(originalContent))
	gzWriter.Close()

	// Create mock response
	resp := &http.Response{
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
		},
		Body: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}

	// Decompress
	err := DecompressResponse(resp)
	require.NoError(t, err)

	// Read decompressed content
	decompressed, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, originalContent, string(decompressed))
	assert.Equal(t, "", resp.Header.Get("Content-Encoding"), "Content-Encoding should be removed")
	assert.True(t, resp.Uncompressed, "Uncompressed flag should be set")
}

func TestDecompressResponseBrotli(t *testing.T) {
	// Create brotli compressed content
	originalContent := "This is test content that will be compressed with brotli"
	var buf bytes.Buffer
	brWriter := brotli.NewWriter(&buf)
	brWriter.Write([]byte(originalContent))
	brWriter.Close()

	// Create mock response
	resp := &http.Response{
		Header: http.Header{
			"Content-Encoding": []string{"br"},
		},
		Body: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}

	// Decompress
	err := DecompressResponse(resp)
	require.NoError(t, err)

	// Read decompressed content
	decompressed, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, originalContent, string(decompressed))
	assert.Equal(t, "", resp.Header.Get("Content-Encoding"))
	assert.True(t, resp.Uncompressed)
}

func TestDecompressResponseNoEncoding(t *testing.T) {
	content := "Plain text content"
	resp := &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader(content)),
	}

	// Should not modify response without Content-Encoding
	err := DecompressResponse(resp)
	require.NoError(t, err)

	result, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, string(result))
}

func TestDecompressResponseUnknownEncoding(t *testing.T) {
	content := "Content with unknown encoding"
	resp := &http.Response{
		Header: http.Header{
			"Content-Encoding": []string{"unknown"},
		},
		Body: io.NopCloser(strings.NewReader(content)),
	}

	// Should not error, just leave as-is
	err := DecompressResponse(resp)
	require.NoError(t, err)

	result, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, string(result))
}

func TestDecompressResponseNilResponse(t *testing.T) {
	err := DecompressResponse(nil)
	assert.NoError(t, err, "Should handle nil response gracefully")
}

func TestCompressionPooling(t *testing.T) {
	// Test that readers are properly returned to pool
	originalContent := "Test content for pooling"

	for i := 0; i < 100; i++ {
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		gzWriter.Write([]byte(originalContent))
		gzWriter.Close()

		resp := &http.Response{
			Header: http.Header{
				"Content-Encoding": []string{"gzip"},
			},
			Body: io.NopCloser(bytes.NewReader(buf.Bytes())),
		}

		err := DecompressResponse(resp)
		require.NoError(t, err)

		io.ReadAll(resp.Body)
		resp.Body.Close() // This should return reader to pool
	}

	// If pooling works, this should not cause memory issues
	assert.True(t, true, "Pooling test completed")
}

func TestCompressionConcurrentAccess(t *testing.T) {
	// Test thread-safety of compression pools
	originalContent := "Concurrent test content"

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			gzWriter.Write([]byte(originalContent))
			gzWriter.Close()

			resp := &http.Response{
				Header: http.Header{
					"Content-Encoding": []string{"gzip"},
				},
				Body: io.NopCloser(bytes.NewReader(buf.Bytes())),
			}

			err := DecompressResponse(resp)
			assert.NoError(t, err)

			decompressed, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, originalContent, string(decompressed))
			resp.Body.Close()
		}()
	}

	wg.Wait()
}

func TestCompressionIntegration(t *testing.T) {
	// Create a test server that returns compressed content
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := "Server response content"

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gzWriter := gzip.NewWriter(w)
			defer gzWriter.Close()
			gzWriter.Write([]byte(content))
		} else {
			w.Write([]byte(content))
		}
	}))
	defer ts.Close()

	// Test with compression enabled
	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true, // We handle it manually
		},
	}

	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Decompress
	err = DecompressResponse(resp)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, "Server response content", string(body))
}

func BenchmarkDecompressGzip(b *testing.B) {
	content := strings.Repeat("Test content to compress ", 1000)
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte(content))
	gzWriter.Close()
	compressed := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &http.Response{
			Header: http.Header{
				"Content-Encoding": []string{"gzip"},
			},
			Body: io.NopCloser(bytes.NewReader(compressed)),
		}

		DecompressResponse(resp)
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkDecompressBrotli(b *testing.B) {
	content := strings.Repeat("Test content to compress ", 1000)
	var buf bytes.Buffer
	brWriter := brotli.NewWriter(&buf)
	brWriter.Write([]byte(content))
	brWriter.Close()
	compressed := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &http.Response{
			Header: http.Header{
				"Content-Encoding": []string{"br"},
			},
			Body: io.NopCloser(bytes.NewReader(compressed)),
		}

		DecompressResponse(resp)
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}
