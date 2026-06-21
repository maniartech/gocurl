package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tier-2 fault harness (Spec 14 §A.1). These exercise REAL transport behavior that
// the RoundTripper-injected faultyRT cannot reach — because faultyRT *replaces* the
// net/http transport, so transport fields (ResponseHeaderTimeout, MaxConnsPerHost),
// real DNS, and on-the-wire framing never run. Tier-2 drives a real transport over
// httptest / a hijacked connection. All hermetic and timing-tolerant.

// TestFaultT2_ResponseHeaderTimeout proves a server that stalls before sending
// response headers is bounded by WithResponseHeaderTimeout (a transport field), not
// left to hang — and the failure classifies as a timeout.
func TestFaultT2_ResponseHeaderTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-time.After(2 * time.Second):
		}
		w.Write([]byte("late"))
	}))
	defer srv.Close()
	defer close(release)

	client, err := New(WithResponseHeaderTimeout(120 * time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close()

	start := time.Now()
	resp, err := client.Curl(context.Background(), "curl "+srv.URL)
	drain(resp)
	if err == nil {
		t.Fatal("expected a response-header timeout")
	}
	if !IsTimeout(err) {
		t.Errorf("want timeout, got Kind=%v err=%v", KindOf(err), err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("ResponseHeaderTimeout did not bound the stall: %v", elapsed)
	}
}

// TestFaultT2_PoolExhaustionSerializes proves WithMaxConnsPerHost(1) applies real
// backpressure — concurrent requests serialize over one connection with no deadlock,
// rather than dialing without bound (the unlimited-by-default EMFILE path).
func TestFaultT2_PoolExhaustionSerializes(t *testing.T) {
	var inFlight, maxInFlight int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&inFlight, 1)
		for {
			m := atomic.LoadInt32(&maxInFlight)
			if c <= m || atomic.CompareAndSwapInt32(&maxInFlight, m, c) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client, err := New(WithMaxConnsPerHost(1))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Curl(context.Background(), "curl "+srv.URL)
			if err == nil {
				drain(resp)
			}
		}()
	}
	wg.Wait() // no deadlock

	if got := atomic.LoadInt32(&maxInFlight); got > 1 {
		t.Errorf("MaxConnsPerHost(1) did not serialize: peak concurrent connections = %d", got)
	}
}

// TestFaultT2_DNSFailure proves a name that cannot resolve (.invalid, RFC 6761) is
// classified as a connect failure and bounded by --connect-timeout — not a hang.
func TestFaultT2_DNSFailure(t *testing.T) {
	start := time.Now()
	_, err := CurlArgs(context.Background(), "--connect-timeout", "3",
		"http://gocurl-nonexistent-host.invalid/")
	if err == nil {
		t.Fatal("expected a DNS/connect error")
	}
	if k := KindOf(err); k != KindConnect {
		t.Errorf("DNS failure should classify as KindConnect, got %v: %v", k, err)
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("DNS failure was not bounded: %v", elapsed)
	}
}

// TestFaultT2_PrematureBodyEOF proves a server that promises a Content-Length but
// closes the connection early surfaces a body-read error — the stream is NOT silently
// truncated (a real on-the-wire failure faultyRT cannot model).
func TestFaultT2_PrematureBodyEOF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("server does not support hijacking")
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			return
		}
		// Promise 100 bytes, send 5, then close — a premature EOF.
		bufrw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort")
		bufrw.Flush()
		conn.Close()
	}))
	defer srv.Close()

	resp, err := Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatalf("Curl (headers should arrive): %v", err)
	}
	_, rerr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if rerr == nil {
		t.Fatal("a premature connection close must surface a body-read error, not silent truncation")
	}
}
