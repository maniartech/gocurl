package gocurl

import (
	"strings"
	"testing"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestPrepare_EnvExpansion(t *testing.T) {
	t.Setenv("GC_REQ_TOKEN", "tok42")
	c := newTestClient(t)
	r, err := c.Prepare(`curl -H 'Authorization: Bearer $GC_REQ_TOKEN' https://api.example.com`)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.opts.Headers.Get("Authorization"); got != "Bearer tok42" {
		t.Errorf("Authorization = %q, want expanded", got)
	}
	if r.Method() != "GET" || r.URL() != "https://api.example.com" {
		t.Errorf("method/url = %q %q", r.Method(), r.URL())
	}
}

func TestPrepareNoEnv_KeepsLiteral(t *testing.T) {
	t.Setenv("GC_REQ_TOKEN", "tok42")
	c := newTestClient(t)
	r, err := c.PrepareNoEnv(`curl -H 'Authorization: Bearer $GC_REQ_TOKEN' https://api.example.com`)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.opts.Headers.Get("Authorization"); got != "Bearer $GC_REQ_TOKEN" {
		t.Errorf("Authorization = %q, want literal", got)
	}
}

func TestPrepareWithVars(t *testing.T) {
	t.Setenv("GC_REQ_TOKEN", "from-env")
	c := newTestClient(t)
	r, err := c.PrepareWithVars(Variables{"GC_REQ_TOKEN": "from-map"},
		`curl -H 'Authorization: Bearer $GC_REQ_TOKEN' https://api.example.com`)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.opts.Headers.Get("Authorization"); got != "Bearer from-map" {
		t.Errorf("Authorization = %q, want from-map (not env)", got)
	}
}

func TestPrepare_ParseError(t *testing.T) {
	c := newTestClient(t)
	if _, err := c.Prepare(`curl --not-a-real-flag https://x`); err == nil {
		t.Error("expected parse error for unknown flag")
	}
}

func TestNewRequest_Programmatic(t *testing.T) {
	r, err := NewRequest("POST", "example.com/items",
		Header("X-Token", "abc"),
		Query("page", "2"),
		Body([]byte(`{"a":1}`)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if r.Method() != "POST" {
		t.Errorf("method = %q", r.Method())
	}
	if !strings.HasPrefix(r.URL(), "http://example.com") {
		t.Errorf("url not normalized: %q", r.URL())
	}
	if r.opts.Headers.Get("X-Token") != "abc" {
		t.Errorf("header missing")
	}
	if r.opts.QueryParams.Get("page") != "2" {
		t.Errorf("query missing")
	}
	if r.opts.Body != `{"a":1}` {
		t.Errorf("body = %q", r.opts.Body)
	}
}

func TestNewRequest_BodyReader(t *testing.T) {
	r, err := NewRequest("PUT", "https://x", BodyReader(strings.NewReader("streamed")))
	if err != nil {
		t.Fatal(err)
	}
	if r.opts.Body != "streamed" {
		t.Errorf("body = %q", r.opts.Body)
	}
}

func TestNewRequest_Errors(t *testing.T) {
	if _, err := NewRequest("GET", ""); err == nil {
		t.Error("empty URL should error")
	}
	if _, err := NewRequest("GET", "https://x", Header("", "v")); err == nil {
		t.Error("empty header key should error")
	}
}

func TestRequest_BuilderImmutability(t *testing.T) {
	base, err := NewRequest("GET", "https://x", Header("X-Base", "1"))
	if err != nil {
		t.Fatal(err)
	}
	withH := base.WithHeader("X-New", "v")
	added := base.AddHeader("X-Base", "2")
	withQ := base.WithQuery("q", "1")
	withB := base.WithBody([]byte("body"))
	clone := base.Clone()

	// Original must be unchanged by any builder.
	if base.opts.Headers.Get("X-New") != "" {
		t.Error("WithHeader mutated the original")
	}
	if len(base.opts.Headers["X-Base"]) != 1 {
		t.Error("AddHeader mutated the original")
	}
	if base.opts.QueryParams.Get("q") != "" {
		t.Error("WithQuery mutated the original")
	}
	if base.opts.Body != "" {
		t.Error("WithBody mutated the original")
	}
	// Copies have the change.
	if withH.opts.Headers.Get("X-New") != "v" {
		t.Error("WithHeader copy missing change")
	}
	if len(added.opts.Headers["X-Base"]) != 2 {
		t.Error("AddHeader copy missing change")
	}
	if withQ.opts.QueryParams.Get("q") != "1" {
		t.Error("WithQuery copy missing change")
	}
	if withB.opts.Body != "body" {
		t.Error("WithBody copy missing change")
	}
	if clone.URL() != base.URL() {
		t.Error("Clone lost URL")
	}
}

func TestRequest_WithVars(t *testing.T) {
	c := newTestClient(t)
	r, err := c.PrepareNoEnv(`curl -H 'Authorization: Bearer $TK' https://x`)
	if err != nil {
		t.Fatal(err)
	}
	rebound := r.WithVars(Variables{"TK": "abc"})
	if got := rebound.opts.Headers.Get("Authorization"); got != "Bearer abc" {
		t.Errorf("WithVars rebind = %q", got)
	}
	// Original unchanged.
	if got := r.opts.Headers.Get("Authorization"); got != "Bearer $TK" {
		t.Errorf("original mutated: %q", got)
	}
	// Programmatic request: WithVars is a no-op copy.
	prog, _ := NewRequest("GET", "https://x")
	if prog.WithVars(Variables{"TK": "abc"}).URL() != prog.URL() {
		t.Error("programmatic WithVars should be a no-op copy")
	}
}

func TestRequest_OptionsReturnsCopy(t *testing.T) {
	r, _ := NewRequest("GET", "https://x", Header("X", "1"))
	o := r.Options()
	o.Headers.Set("X", "tampered")
	if r.opts.Headers.Get("X") != "1" {
		t.Error("Options() must return an independent copy")
	}
}
