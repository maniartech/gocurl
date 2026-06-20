package gocurl

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
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
// It reuses gzip/zlib readers via a sync.Pool to reduce per-response allocations (not a
// zero-allocation guarantee).
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
		dr, derr := newDeflateReader(resp.Body)
		if derr != nil {
			return fmt.Errorf("failed to create deflate reader: %w", derr)
		}
		reader = dr

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

// deflateReader decompresses a Content-Encoding: deflate body. Per RFC 7230 the
// "deflate" coding is zlib-wrapped DEFLATE (RFC 1950), but some servers send raw
// DEFLATE (RFC 1951); we detect which and use compress/zlib or compress/flate.
type deflateReader struct {
	body io.ReadCloser
	dec  io.ReadCloser
}

func newDeflateReader(body io.ReadCloser) (*deflateReader, error) {
	br := bufio.NewReader(body)
	var dec io.ReadCloser
	if peek, err := br.Peek(2); err == nil && isZlibHeader(peek) {
		zr, zerr := zlib.NewReader(br)
		if zerr != nil {
			return nil, zerr
		}
		dec = zr
	} else {
		dec = flate.NewReader(br)
	}
	return &deflateReader{body: body, dec: dec}, nil
}

// isZlibHeader reports whether b looks like a zlib header (RFC 1950): the low
// nibble of CMF is 8 (the deflate method) and CMF*256+FLG is a multiple of 31.
func isZlibHeader(b []byte) bool {
	if len(b) < 2 || b[0]&0x0f != 0x08 {
		return false
	}
	return (uint16(b[0])<<8|uint16(b[1]))%31 == 0
}

func (r *deflateReader) Read(p []byte) (int, error) {
	return r.dec.Read(p)
}

func (r *deflateReader) Close() error {
	err := r.dec.Close()
	if cerr := r.body.Close(); err == nil {
		err = cerr
	}
	return err
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
