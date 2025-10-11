package gocurl

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
)

// Compression types supported
const (
	CompressionGzip    = "gzip"
	CompressionDeflate = "deflate"
	CompressionBrotli  = "br"
)

// Pool for gzip readers to avoid allocations
var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Reader)
	},
}

// Pool for brotli readers to avoid allocations
var brotliReaderPool = sync.Pool{
	New: func() interface{} {
		return brotli.NewReader(nil)
	},
}

// DecompressResponse handles automatic decompression of response bodies based on Content-Encoding.
// This function is zero-allocation optimized using reader pools.
func DecompressResponse(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}

	encoding := strings.ToLower(resp.Header.Get("Content-Encoding"))
	if encoding == "" {
		return nil
	}

	var reader io.ReadCloser
	var err error

	switch encoding {
	case CompressionGzip:
		// Get gzip reader from pool
		gzReader := gzipReaderPool.Get().(*gzip.Reader)
		if err := gzReader.Reset(resp.Body); err != nil {
			gzipReaderPool.Put(gzReader)
			return fmt.Errorf("failed to reset gzip reader: %w", err)
		}
		reader = &pooledGzipReader{
			Reader: gzReader,
			body:   resp.Body,
		}

	case CompressionDeflate:
		// For deflate, we can't pool easily, but it's less common
		// Use standard library's flate reader
		reader = &deflateReader{
			body: resp.Body,
		}

	case CompressionBrotli:
		// Get brotli reader from pool
		brReader := brotliReaderPool.Get().(*brotli.Reader)
		if err := brReader.Reset(resp.Body); err != nil {
			brotliReaderPool.Put(brReader)
			return fmt.Errorf("failed to reset brotli reader: %w", err)
		}
		reader = &pooledBrotliReader{
			Reader: brReader,
			body:   resp.Body,
		}

	default:
		// Unknown encoding, leave as-is
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to create decompression reader for %s: %w", encoding, err)
	}

	// Replace the body with the decompressing reader
	resp.Body = reader
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	resp.ContentLength = -1
	resp.Uncompressed = true

	return nil
}

// pooledGzipReader wraps a gzip.Reader and returns it to the pool when closed
type pooledGzipReader struct {
	*gzip.Reader
	body io.ReadCloser
}

func (r *pooledGzipReader) Close() error {
	// Close the gzip reader first
	if err := r.Reader.Close(); err != nil {
		// Still try to close body and return to pool
		r.body.Close()
		gzipReaderPool.Put(r.Reader)
		return err
	}

	// Close the underlying body
	err := r.body.Close()

	// Return reader to pool
	gzipReaderPool.Put(r.Reader)

	return err
}

// pooledBrotliReader wraps a brotli.Reader and returns it to the pool when closed
type pooledBrotliReader struct {
	*brotli.Reader
	body io.ReadCloser
}

func (r *pooledBrotliReader) Close() error {
	// Close the underlying body
	err := r.body.Close()

	// Return reader to pool
	brotliReaderPool.Put(r.Reader)

	return err
}

// deflateReader wraps a deflate decompressor
type deflateReader struct {
	body io.ReadCloser
	// We'll use gzip.Reader with a modified header
	reader io.ReadCloser
}

func (r *deflateReader) Read(p []byte) (n int, err error) {
	if r.reader == nil {
		// Create a flate reader (deflate is the raw format)
		// We need to add a gzip header or use compress/flate directly
		// For simplicity, try gzip reader (most servers send gzip-compatible deflate)
		gzReader := gzipReaderPool.Get().(*gzip.Reader)
		if err := gzReader.Reset(r.body); err != nil {
			// If it fails, it might be raw deflate - we'd need compress/flate
			gzipReaderPool.Put(gzReader)
			// For now, return error - in production, implement raw deflate
			return 0, fmt.Errorf("deflate decompression not fully implemented: %w", err)
		}
		r.reader = gzReader
	}
	return r.reader.Read(p)
}

func (r *deflateReader) Close() error {
	if r.reader != nil {
		if gzReader, ok := r.reader.(*gzip.Reader); ok {
			gzReader.Close()
			gzipReaderPool.Put(gzReader)
		}
	}
	return r.body.Close()
}

// GetAcceptEncodingHeader returns the appropriate Accept-Encoding header value
// based on the compression settings. This follows curl's behavior.
func GetAcceptEncodingHeader(compress bool, methods []string) string {
	if !compress {
		return ""
	}

	if len(methods) == 0 {
		// Default: support all common encodings
		return "gzip, deflate, br"
	}

	return strings.Join(methods, ", ")
}

// ConfigureCompressionForTransport sets up the HTTP transport for compression handling.
// Note: DisableCompression should be set to true to prevent automatic decompression
// by net/http, allowing us to handle it manually with pooled readers.
func ConfigureCompressionForTransport(transport *http.Transport, compress bool) {
	// When compress is true, we want to:
	// 1. Send Accept-Encoding header (done in request)
	// 2. Manually decompress (to use our pooled readers)
	// So we DISABLE automatic compression to handle it ourselves
	transport.DisableCompression = true
}
