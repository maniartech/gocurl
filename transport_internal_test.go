package gocurl

import (
	"context"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

func TestConfig_TransportDefaults(t *testing.T) {
	cfg := defaultConfig()
	if cfg.maxIdleConns != 100 || cfg.maxIdleConnsPerHost != 10 || cfg.maxConnsPerHost != 0 {
		t.Errorf("conn defaults: %+v", cfg)
	}
	if cfg.idleConnTimeout != 90*time.Second || cfg.tlsHandshakeTimeout != 10*time.Second ||
		cfg.expectContinueTimeout != 1*time.Second {
		t.Errorf("timeout defaults: %+v", cfg)
	}
	if !cfg.http2 {
		t.Error("http2 should default on")
	}
}

func TestTransportOptions_SetFields(t *testing.T) {
	cfg := buildConfig(t,
		WithMaxIdleConns(7),
		WithMaxIdleConnsPerHost(3),
		WithMaxConnsPerHost(5),
		WithIdleConnTimeout(time.Second),
		WithTLSHandshakeTimeout(2*time.Second),
		WithResponseHeaderTimeout(3*time.Second),
		WithExpectContinueTimeout(4*time.Second),
		WithHTTP2(false),
	)
	if cfg.maxIdleConns != 7 || cfg.maxIdleConnsPerHost != 3 || cfg.maxConnsPerHost != 5 {
		t.Errorf("conn fields: %+v", cfg)
	}
	if cfg.idleConnTimeout != time.Second || cfg.tlsHandshakeTimeout != 2*time.Second ||
		cfg.responseHeaderTimeout != 3*time.Second || cfg.expectContinueTimeout != 4*time.Second {
		t.Errorf("timeout fields: %+v", cfg)
	}
	if cfg.http2 {
		t.Error("WithHTTP2(false) should disable http2")
	}
}

func TestTransportOptions_Errors(t *testing.T) {
	cases := map[string]Option{
		"max idle":          WithMaxIdleConns(-1),
		"max idle per host": WithMaxIdleConnsPerHost(-1),
		"max conns":         WithMaxConnsPerHost(-1),
		"idle timeout":      WithIdleConnTimeout(-1),
		"tls handshake":     WithTLSHandshakeTimeout(-1),
		"resp header":       WithResponseHeaderTimeout(-1),
		"expect continue":   WithExpectContinueTimeout(-1),
		"redirect max":      WithRedirectPolicy(RedirectPolicy{Max: -1}),
	}
	for name, opt := range cases {
		if err := opt(defaultConfig()); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestBuildTransport_DefaultTuning(t *testing.T) {
	rt, err := defaultConfig().buildTransport()
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", rt)
	}
	if tr.MaxIdleConns != 100 || tr.MaxIdleConnsPerHost != 10 || tr.MaxConnsPerHost != 0 {
		t.Errorf("transport conn tuning: %+v", tr)
	}
	if tr.IdleConnTimeout != 90*time.Second || tr.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("transport timeouts wrong")
	}
	if tr.DialContext == nil {
		t.Error("DialContext should be set (connect timeout seam)")
	}
	if !tr.DisableCompression {
		t.Error("DisableCompression should be true (manual decompression)")
	}
	if !tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true by default")
	}
}

func TestBuildTransport_CustomOverride(t *testing.T) {
	custom := roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })
	cfg := buildConfig(t, WithTransport(custom))
	rt, err := cfg.buildTransport()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rt.(roundTripperFunc); !ok {
		t.Errorf("WithTransport override not returned: %T", rt)
	}
}

func TestBuildTransport_HTTP2Only(t *testing.T) {
	cfg := buildConfig(t, WithHTTP2Only(true))
	rt, err := cfg.buildTransport()
	if err != nil {
		t.Fatal(err)
	}
	h2, ok := rt.(*http2.Transport)
	if !ok {
		t.Fatalf("expected *http2.Transport, got %T", rt)
	}
	if !h2.AllowHTTP {
		t.Error("h2c (AllowHTTP) should be enabled when allowCleartext=true")
	}
}

func TestBuildTransport_HTTP2Disabled(t *testing.T) {
	cfg := buildConfig(t, WithHTTP2(false))
	rt, err := cfg.buildTransport()
	if err != nil {
		t.Fatal(err)
	}
	tr := rt.(*http.Transport)
	if tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be false")
	}
	if tr.TLSNextProto != nil {
		t.Error("h2 should not be configured when disabled")
	}
}

func TestBuildTransport_Proxy(t *testing.T) {
	cfg := buildConfig(t, WithProxy("http://127.0.0.1:9"))
	rt, err := cfg.buildTransport()
	if err != nil || rt == nil {
		t.Fatalf("proxy transport: rt=%v err=%v", rt, err)
	}
}

func TestWithRedirectPolicy(t *testing.T) {
	cfg := buildConfig(t, WithRedirectPolicy(RedirectPolicy{Follow: true}))
	if !cfg.followRedirects || cfg.maxRedirects != 30 {
		t.Errorf("follow default max: %+v", cfg)
	}
	allow := func(*http.Request, []*http.Request) error { return nil }
	cfg2 := buildConfig(t, WithRedirectPolicy(RedirectPolicy{Follow: true, Max: 5, Allow: allow}))
	if cfg2.maxRedirects != 5 || cfg2.redirectAllow == nil {
		t.Errorf("explicit max/allow: %+v", cfg2)
	}
}

func TestRedirectFromContext_AllowHook(t *testing.T) {
	blocked := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, _ := http.NewRequest("GET", "http://x", nil)
	req = req.WithContext(withRedirectSettings(context.Background(), redirectSettings{follow: true, max: 5, allow: blocked}))
	if err := redirectFromContext(req, nil); err != http.ErrUseLastResponse {
		t.Errorf("allow hook error not propagated: %v", err)
	}
}
