package gocurl

import (
	"net/http"
	"testing"
	"time"
)

func buildConfig(t *testing.T, opts ...Option) *config {
	t.Helper()
	cfg := defaultConfig()
	for _, o := range opts {
		if err := o(cfg); err != nil {
			t.Fatalf("option error: %v", err)
		}
	}
	return cfg
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.followRedirects || cfg.maxRedirects != 0 || cfg.timeout != 0 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if cfg.defaultHeaders == nil {
		t.Error("defaultHeaders should be initialized")
	}
}

func TestOptions_HappyPaths(t *testing.T) {
	cfg := buildConfig(t,
		WithTimeout(5*time.Second),
		WithConnectTimeout(2*time.Second),
		WithMaxRedirects(7),
		WithFollowRedirects(true),
		WithProxy("http://proxy:8080"),
		WithInsecure(true),
		WithUserAgent("ua/1"),
		WithDefaultHeader("X-A", "1"),
		WithDefaultHeader("X-A", "2"),
		WithCookieFile("cookies.txt"),
	)
	if cfg.timeout != 5*time.Second || cfg.connectTimeout != 2*time.Second {
		t.Errorf("timeouts: %+v", cfg)
	}
	// WithFollowRedirects(true) must not clobber an explicit WithMaxRedirects(7).
	if !cfg.followRedirects || cfg.maxRedirects != 7 {
		t.Errorf("redirects: follow=%v max=%d", cfg.followRedirects, cfg.maxRedirects)
	}
	if cfg.proxy != "http://proxy:8080" || !cfg.insecure || cfg.userAgent != "ua/1" {
		t.Errorf("conn/ua: %+v", cfg)
	}
	if got := cfg.defaultHeaders["X-A"]; len(got) != 2 {
		t.Errorf("default headers additive: %v", got)
	}
	if cfg.cookieFile != "cookies.txt" {
		t.Errorf("cookieFile: %q", cfg.cookieFile)
	}
}

func TestWithFollowRedirects_DefaultsMax(t *testing.T) {
	cfg := buildConfig(t, WithFollowRedirects(true))
	if cfg.maxRedirects != 30 {
		t.Errorf("maxRedirects = %d, want 30 default", cfg.maxRedirects)
	}
}

func TestWithMiddlewareAndTransport(t *testing.T) {
	mw := func(next Handler) Handler { return next }
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })
	cfg := buildConfig(t, WithMiddleware(mw, mw), WithTransport(rt))
	if len(cfg.middleware) != 2 {
		t.Errorf("middleware count = %d", len(cfg.middleware))
	}
	if cfg.transport == nil {
		t.Error("transport not set")
	}
}

func TestOptions_ErrorPaths(t *testing.T) {
	cases := map[string]Option{
		"neg timeout":         WithTimeout(-1),
		"neg connect timeout": WithConnectTimeout(-1),
		"neg max redirects":   WithMaxRedirects(-1),
		"empty header key":    WithDefaultHeader("", "v"),
		"nil transport":       WithTransport(nil),
	}
	for name, opt := range cases {
		if err := opt(defaultConfig()); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestBaseOptions(t *testing.T) {
	cfg := buildConfig(t, WithProxy("socks5://p:1080"), WithInsecure(true), WithConnectTimeout(3*time.Second))
	o := cfg.baseOptions()
	if o.Proxy != "socks5://p:1080" || !o.Insecure || o.ConnectTimeout != 3*time.Second {
		t.Errorf("baseOptions projection wrong: %+v", o)
	}
}
