package gocurl

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/maniartech/gocurl/middlewares"
)

func okHandler() Handler {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	}
}

func TestChain_OutermostFirstOrder(t *testing.T) {
	var order []string
	mk := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(req *http.Request) (*http.Response, error) {
				order = append(order, "in:"+name)
				resp, err := next(req)
				order = append(order, "out:"+name)
				return resp, err
			}
		}
	}
	base := Handler(func(req *http.Request) (*http.Response, error) {
		order = append(order, "base")
		return okHandler()(req)
	})

	h := chain(base, mk("a"), mk("b"), nil) // nil middleware must be skipped
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if _, err := h(req); err != nil {
		t.Fatal(err)
	}
	want := []string{"in:a", "in:b", "base", "out:b", "out:a"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestChain_NoMiddleware(t *testing.T) {
	called := false
	base := Handler(func(req *http.Request) (*http.Response, error) {
		called = true
		return okHandler()(req)
	})
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if _, err := chain(base)(req); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("base not called")
	}
}

func TestFromMiddlewareFunc_MutatesRequest(t *testing.T) {
	mf := middlewares.MiddlewareFunc(func(r *http.Request) (*http.Request, error) {
		r.Header.Set("X-Added", "yes")
		return r, nil
	})
	var seen string
	base := Handler(func(r *http.Request) (*http.Response, error) {
		seen = r.Header.Get("X-Added")
		return okHandler()(r)
	})
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if _, err := chain(base, FromMiddlewareFunc(mf))(req); err != nil {
		t.Fatal(err)
	}
	if seen != "yes" {
		t.Errorf("middleware mutation not seen: %q", seen)
	}
}

func TestFromMiddlewareFunc_ErrorShortCircuits(t *testing.T) {
	mf := middlewares.MiddlewareFunc(func(r *http.Request) (*http.Request, error) {
		return nil, fmt.Errorf("boom")
	})
	baseCalled := false
	base := Handler(func(r *http.Request) (*http.Response, error) {
		baseCalled = true
		return okHandler()(r)
	})
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := chain(base, FromMiddlewareFunc(mf))(req)
	if err == nil || err.Error() != "boom" {
		t.Errorf("err = %v, want boom", err)
	}
	if baseCalled {
		t.Error("base must not be called after middleware error")
	}
}

func TestRoundTripperHandlerAdapters(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 204, Body: http.NoBody}, nil
	})
	h := HandlerFromRoundTripper(rt)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := h(req)
	if err != nil || resp.StatusCode != 204 {
		t.Fatalf("HandlerFromRoundTripper: resp=%v err=%v", resp, err)
	}

	back := RoundTripperFromHandler(h)
	resp2, err := back.RoundTrip(req)
	if err != nil || resp2.StatusCode != 204 {
		t.Fatalf("RoundTripperFromHandler: resp=%v err=%v", resp2, err)
	}
}
