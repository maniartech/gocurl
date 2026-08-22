package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// HTTPS-through-HTTP-proxy (CONNECT tunnel) coverage.
//
// This is the security-sensitive path: the client opens a plain TCP connection to the
// proxy, issues CONNECT, and then runs a TLS handshake with the ORIGIN over that tunnel.
// If the handshake did not verify the origin's certificate, a malicious proxy could
// terminate TLS itself and read everything. establishTLS had 0% coverage before this file.
//
// The tests below drive the real code path (createDialTLSContext -> sendConnectRequest ->
// verifyConnectResponse -> establishTLS -> prepareTLSConfig -> performTLSHandshake) against
// a real CONNECT proxy and a real TLS origin, and assert both that it works AND that it
// fails closed against an untrusted certificate.

// connectProxy is a minimal CONNECT-tunnelling HTTP proxy for tests. It records the
// Proxy-Authorization header it received and can be told to reject with a status.
type connectProxy struct {
	ln         net.Listener
	rejectWith string // e.g. "407 Proxy Authentication Required"; empty = accept

	mu       sync.Mutex
	gotAuth  string
	gotHost  string
	requests int
}

func newConnectProxy(t *testing.T, rejectWith string) *connectProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	p := &connectProxy{ln: ln, rejectWith: rejectWith}
	go p.serve()
	t.Cleanup(func() { ln.Close() })
	return p
}

func (p *connectProxy) addr() string { return p.ln.Addr().String() }

func (p *connectProxy) snapshot() (auth, host string, n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gotAuth, p.gotHost, p.requests
}

func (p *connectProxy) serve() {
	for {
		conn, err := p.ln.Accept()
		if err != nil {
			return
		}
		go p.handle(conn)
	}
}

func (p *connectProxy) handle(client net.Conn) {
	defer client.Close()

	br := bufio.NewReader(client)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	p.mu.Lock()
	p.gotAuth = req.Header.Get("Proxy-Authorization")
	p.gotHost = req.Host
	p.requests++
	p.mu.Unlock()

	if req.Method != http.MethodConnect {
		io.WriteString(client, "HTTP/1.1 405 Method Not Allowed\r\n\r\n")
		return
	}
	if p.rejectWith != "" {
		io.WriteString(client, "HTTP/1.1 "+p.rejectWith+"\r\n\r\n")
		return
	}

	// Dial the real origin and splice the two connections together.
	origin, err := net.DialTimeout("tcp", req.Host, 5*time.Second)
	if err != nil {
		io.WriteString(client, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		return
	}
	defer origin.Close()

	if _, err := io.WriteString(client, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(origin, br) }()
	go func() { defer wg.Done(); io.Copy(client, origin) }()
	wg.Wait()
}

// tlsOrigin starts an HTTPS server that records the SNI name it was asked for.
func tlsOrigin(t *testing.T, sni *string, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tunnelled-ok"))
	}))
	srv.TLS = &tls.Config{
		GetConfigForClient: func(hi *tls.ClientHelloInfo) (*tls.Config, error) {
			if sni != nil {
				mu.Lock()
				*sni = hi.ServerName
				mu.Unlock()
			}
			return nil, nil
		},
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

// trustPool returns a CertPool trusting the test origin's certificate.
func trustPool(t *testing.T, srv *httptest.Server) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return pool
}

// TestHTTPProxy_TLSTunnel_Success drives a real HTTPS request through a real CONNECT
// proxy: the full establishTLS path, end to end.
func TestHTTPProxy_TLSTunnel_Success(t *testing.T) {
	var mu sync.Mutex
	var sni string
	origin := tlsOrigin(t, &sni, &mu)
	px := newConnectProxy(t, "")

	transport, err := NewTransport(ProxyConfig{
		Type:      ProxyTypeHTTP,
		Address:   px.addr(),
		TLSConfig: &tls.Config{RootCAs: trustPool(t, origin)},
	})
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	resp, err := client.Get(origin.URL)
	if err != nil {
		t.Fatalf("HTTPS through CONNECT proxy failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if string(body) != "tunnelled-ok" {
		t.Errorf("body = %q, want tunnelled-ok", body)
	}
	_, host, n := px.snapshot()
	if n == 0 {
		t.Error("the proxy was never used — the request bypassed the tunnel")
	}
	if !strings.Contains(host, "127.0.0.1") {
		t.Errorf("CONNECT host = %q, want the origin host:port", host)
	}
	// NOTE: no SNI assertion here on purpose. The tunnel target is an IP literal, and
	// per RFC 6066 a TLS client must not send SNI for an IP address, so the wire value
	// is legitimately empty. The ServerName logic is asserted directly in
	// TestHTTPProxy_PrepareTLSConfig below.
	mu.Lock()
	_ = sni
	mu.Unlock()
}

// TestHTTPProxy_PrepareTLSConfig asserts the SNI/ServerName rules directly: derive the
// host from the tunnel target, never clobber an explicit ServerName, and never mutate the
// caller's *tls.Config (it is shared across connections).
func TestHTTPProxy_PrepareTLSConfig(t *testing.T) {
	t.Run("derives ServerName from the tunnel target", func(t *testing.T) {
		hp := &HTTPProxy{}
		got := hp.prepareTLSConfig("api.example.com:443")
		if got.ServerName != "api.example.com" {
			t.Errorf("ServerName = %q, want api.example.com", got.ServerName)
		}
	})

	t.Run("target without a port still yields a host", func(t *testing.T) {
		hp := &HTTPProxy{}
		got := hp.prepareTLSConfig("api.example.com")
		if got.ServerName != "api.example.com" {
			t.Errorf("ServerName = %q, want api.example.com", got.ServerName)
		}
	})

	t.Run("an explicit ServerName wins", func(t *testing.T) {
		hp := &HTTPProxy{TLSConfig: &tls.Config{ServerName: "pinned.example"}}
		got := hp.prepareTLSConfig("api.example.com:443")
		if got.ServerName != "pinned.example" {
			t.Errorf("ServerName = %q, want the caller's pinned.example", got.ServerName)
		}
	})

	t.Run("does not mutate the caller's config", func(t *testing.T) {
		base := &tls.Config{}
		hp := &HTTPProxy{TLSConfig: base}
		got := hp.prepareTLSConfig("api.example.com:443")
		if base.ServerName != "" {
			t.Errorf("caller's TLSConfig was mutated: ServerName = %q", base.ServerName)
		}
		if got == base {
			t.Error("prepareTLSConfig returned the caller's config instead of a clone")
		}
	})
}

// TestHTTPProxy_TLSTunnel_VerifiesOriginCertificate is the security assertion: with a
// CertPool that does NOT trust the origin, the tunnelled handshake must fail. If this
// ever passes silently, a hostile proxy could MITM every tunnelled request.
func TestHTTPProxy_TLSTunnel_VerifiesOriginCertificate(t *testing.T) {
	origin := tlsOrigin(t, nil, nil)
	px := newConnectProxy(t, "")

	transport, err := NewTransport(ProxyConfig{
		Type:    ProxyTypeHTTP,
		Address: px.addr(),
		// Empty pool: the origin's self-signed cert is not trusted.
		TLSConfig: &tls.Config{RootCAs: x509.NewCertPool()},
	})
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	resp, err := client.Get(origin.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("tunnelled TLS handshake succeeded against an UNTRUSTED certificate — " +
			"a malicious proxy could MITM this connection")
	}
	if !strings.Contains(err.Error(), "TLS handshake failed") &&
		!strings.Contains(err.Error(), "certificate") {
		t.Errorf("error = %v, want a TLS/certificate verification failure", err)
	}
}

// TestHTTPProxy_TLSTunnel_InsecureSkipVerify proves the documented opt-out works, so the
// verification above is a policy we control rather than an accident.
func TestHTTPProxy_TLSTunnel_InsecureSkipVerify(t *testing.T) {
	origin := tlsOrigin(t, nil, nil)
	px := newConnectProxy(t, "")

	transport, err := NewTransport(ProxyConfig{
		Type:      ProxyTypeHTTP,
		Address:   px.addr(),
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	resp, err := client.Get(origin.URL)
	if err != nil {
		t.Fatalf("InsecureSkipVerify should permit the untrusted origin: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "tunnelled-ok" {
		t.Errorf("body = %q, want tunnelled-ok", body)
	}
}

// TestHTTPProxy_TLSTunnel_SendsProxyAuth asserts CONNECT carries Proxy-Authorization when
// credentials are configured (createConnectRequest), including the password-optional form.
func TestHTTPProxy_TLSTunnel_SendsProxyAuth(t *testing.T) {
	for _, tc := range []struct {
		name, user, pass string
	}{
		{"user and password", "alice", "s3cret"},
		{"username only", "alice", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			origin := tlsOrigin(t, nil, nil)
			px := newConnectProxy(t, "")

			transport, err := NewTransport(ProxyConfig{
				Type:      ProxyTypeHTTP,
				Address:   px.addr(),
				Username:  tc.user,
				Password:  tc.pass,
				TLSConfig: &tls.Config{RootCAs: trustPool(t, origin)},
			})
			if err != nil {
				t.Fatalf("NewTransport: %v", err)
			}
			client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
			resp, err := client.Get(origin.URL)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			resp.Body.Close()

			auth, _, _ := px.snapshot()
			if !strings.HasPrefix(auth, "Basic ") {
				t.Fatalf("Proxy-Authorization = %q, want a Basic credential", auth)
			}
		})
	}
}

// TestHTTPProxy_TLSTunnel_RejectedConnect asserts a proxy refusal surfaces as an error
// rather than a hang or a silent plaintext fallback (verifyConnectResponse).
func TestHTTPProxy_TLSTunnel_RejectedConnect(t *testing.T) {
	origin := tlsOrigin(t, nil, nil)
	px := newConnectProxy(t, "407 Proxy Authentication Required")

	transport, err := NewTransport(ProxyConfig{
		Type:      ProxyTypeHTTP,
		Address:   px.addr(),
		TLSConfig: &tls.Config{RootCAs: trustPool(t, origin)},
	})
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	resp, err := client.Get(origin.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("a rejected CONNECT must not yield a usable connection")
	}
	// The refusal may surface from our verifyConnectResponse ("proxy CONNECT failed:
	// 407 ...") or from net/http's own proxy handling ("Proxy Authentication Required"),
	// depending on which layer owns the tunnel for this target. Both fail closed, which
	// is the property under test; accept either wording.
	msg := err.Error()
	if !strings.Contains(msg, "CONNECT failed") &&
		!strings.Contains(msg, "407") &&
		!strings.Contains(strings.ToLower(msg), "proxy authentication required") {
		t.Errorf("error = %v, want the proxy CONNECT rejection surfaced", err)
	}
}

// TestHTTPProxy_TLSTunnel_ContextCancel proves performTLSHandshake honors context
// cancellation instead of blocking forever on a proxy that accepts but never speaks TLS.
func TestHTTPProxy_TLSTunnel_ContextCancel(t *testing.T) {
	// A listener that accepts CONNECT, replies 200, then goes silent — the handshake
	// can never complete.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				if _, err := http.ReadRequest(br); err != nil {
					return
				}
				io.WriteString(c, "HTTP/1.1 200 Connection established\r\n\r\n")
				<-done // never negotiate TLS
			}(c)
		}
	}()

	hp := &HTTPProxy{Address: ln.Addr().String()}
	dialTLS := hp.createDialTLSContext(&net.Dialer{Timeout: 5 * time.Second})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	conn, err := dialTLS(ctx, "tcp", "example.com:443")
	if err == nil {
		conn.Close()
		t.Fatal("expected the stalled handshake to fail")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("handshake was not bounded by context: %v", elapsed)
	}
}
