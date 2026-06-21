package benchcmp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	resty "github.com/go-resty/resty/v2"
	"github.com/imroc/req/v3"
	"github.com/maniartech/gocurl"
)

// Client-OVERHEAD benchmarks (Spec 14 §B, reliability of the measurement itself).
//
// The full round-trip arms (bench_vendor_test.go) go over a real loopback TCP
// connection. A CPU profile of that path shows ~45% of time in the Windows network
// syscall (runtime.cgocall) plus ~15-20% in goroutine scheduling between the client
// and the in-process server — all SHARED by every arm and dominated by OS/machine
// load, not by the library. That is why the round-trip ns/op is noisy and rank-flips.
//
// These arms replace the transport with stubRT, which returns a canned response with
// NO network and NO second goroutine. What remains is exactly the per-request CPU and
// allocation each library adds ABOVE the transport — the thing we actually optimize —
// so the numbers are stable and reproducible run to run. This is the honest client-
// overhead comparison; the loopback arms remain for end-to-end realism (advisory ns/op).

// stubRT returns a fresh, canned 200 response on every call without any I/O. The body
// is a new reader each time (the client consumes it), so there is no shared state.
type stubRT struct{ body string }

func (s stubRT) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(strings.NewReader(s.body)),
		ContentLength: int64(len(s.body)),
		Request:       r,
	}, nil
}

var stubBody = `{"ok":true}`

func BenchmarkOverhead_NetHTTP(b *testing.B) {
	client := &http.Client{Transport: stubRT{stubBody}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://stub.invalid/")
		if err != nil {
			b.Fatal(err)
		}
		drain(resp)
	}
}

func BenchmarkOverhead_Gocurl_Prepared(b *testing.B) {
	c, err := gocurl.New(gocurl.WithTransport(stubRT{stubBody}))
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl http://stub.invalid/")
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		drain(resp)
	}
}

func BenchmarkOverhead_Resty(b *testing.B) {
	client := resty.New().SetTransport(stubRT{stubBody})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.R().Get("http://stub.invalid/")
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body()
	}
}

func BenchmarkOverhead_Req(b *testing.B) {
	client := req.C()
	client.GetClient().Transport = stubRT{stubBody}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.R().Get("http://stub.invalid/")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
