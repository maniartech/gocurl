package gocurl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
)

const (
	// MaxPooledResponseSize is the maximum size for pooled buffers (1MB)
	MaxPooledResponseSize = 1024 * 1024
	// StreamingThreshold is when we switch to streaming mode (1MB)
	StreamingThreshold = 1024 * 1024
)

// Buffer pool for small responses
var responseBufferPool = sync.Pool{
	New: func() interface{} {
		// Allocate 64KB buffers by default
		return bytes.NewBuffer(make([]byte, 0, 64*1024))
	},
}

// readResponseBody intelligently reads response bodies based on size
// Small responses (<1MB) use pooled buffers, large responses stream
func readResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("response or body is nil")
	}
	defer resp.Body.Close()

	// Check Content-Length to decide strategy
	if resp.ContentLength > 0 {
		if resp.ContentLength > StreamingThreshold {
			// Large response - read directly without pooling
			return io.ReadAll(resp.Body)
		}

		// Small response with known size - use pool if suitable
		if resp.ContentLength <= MaxPooledResponseSize {
			return readWithPooledBuffer(resp.Body, int(resp.ContentLength))
		}
	}

	// Unknown size - read with pooled buffer, up to threshold
	return readWithPooledBuffer(resp.Body, 0)
}

// readWithPooledBuffer reads data using a pooled buffer
func readWithPooledBuffer(r io.Reader, expectedSize int) ([]byte, error) {
	buf := responseBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer responseBufferPool.Put(buf)

	if expectedSize > 0 {
		buf.Grow(expectedSize)
	}

	written, err := io.Copy(buf, r)
	if err != nil {
		return nil, err
	}

	// If the response is too large for pooling, just return the bytes
	if written > MaxPooledResponseSize {
		return io.ReadAll(bytes.NewReader(buf.Bytes()))
	}

	// Make a copy since we're returning the buffer to the pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// streamResponse handles large response streaming
// This can be used for downloading large files
func streamResponse(resp *http.Response, writer io.Writer) (int64, error) {
	if resp == nil || resp.Body == nil {
		return 0, fmt.Errorf("response or body is nil")
	}
	defer resp.Body.Close()

	// Use pooled buffer for streaming copy
	buf := responseBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer responseBufferPool.Put(buf)

	// Grow buffer to a good size for streaming
	buf.Grow(64 * 1024) // 64KB chunks

	return io.CopyBuffer(writer, resp.Body, buf.Bytes()[:cap(buf.Bytes())])
}

// ResponseReader wraps a response for efficient reading
type ResponseReader struct {
	resp   *http.Response
	body   []byte
	reader *bytes.Reader
	read   bool
}

// NewResponseReader creates a reader for efficient response handling
func NewResponseReader(resp *http.Response) *ResponseReader {
	return &ResponseReader{
		resp: resp,
	}
}

// Read implements io.Reader for streaming
func (rr *ResponseReader) Read(p []byte) (int, error) {
	if !rr.read {
		// First read - buffer the body
		body, err := readResponseBody(rr.resp)
		if err != nil {
			return 0, err
		}
		rr.body = body
		rr.reader = bytes.NewReader(body)
		rr.read = true
	}

	return rr.reader.Read(p)
}

// Bytes returns the full body as bytes
func (rr *ResponseReader) Bytes() ([]byte, error) {
	if !rr.read {
		body, err := readResponseBody(rr.resp)
		if err != nil {
			return nil, err
		}
		rr.body = body
		rr.read = true
	}
	return rr.body, nil
}

// String returns the body as a string
func (rr *ResponseReader) String() (string, error) {
	b, err := rr.Bytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Reset allows the reader to be reused
func (rr *ResponseReader) Reset() {
	rr.read = false
	rr.body = nil
	rr.reader = nil
}
