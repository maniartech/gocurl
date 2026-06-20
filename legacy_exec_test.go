package gocurl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/maniartech/gocurl/options"
)

// processForTest reproduces the contract of the removed deprecated Process() for
// the whitebox tests that drive the engine directly via *RequestOptions: it runs
// the live one-shot pipeline, buffers the body (honoring opts.ResponseBodyLimit),
// and re-wraps resp.Body so it can be read again. It performs NO output side
// effects (the deleted Process wrote to stdout/OutputFile; the library never does).
func processForTest(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
	resp, err := doRequest(ctx, opts)
	if err != nil {
		return nil, "", err
	}
	b, err := readBufferWithLimit(resp.Body, opts.ResponseBodyLimit)
	resp.Body.Close()
	if err != nil {
		return nil, "", err
	}
	resp.Body = io.NopCloser(strings.NewReader(string(b)))
	return resp, string(b), nil
}

func readBufferWithLimit(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(r)
	}
	b, err := io.ReadAll(io.LimitReader(r, limit+1)) // +1 to detect overflow
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("response body size (%d bytes) exceeds limit of %d bytes", len(b)-1, limit)
	}
	return b, nil
}
