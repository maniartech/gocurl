package gocurl

import (
	"net/http"

	"github.com/maniartech/gocurl/middlewares"
)

// Handler executes a single HTTP request and returns its response. It has the
// same shape as http.RoundTripper.RoundTrip but as a function, so behaviors can
// be composed. The innermost Handler in a Client's chain is backed by the
// Client's pooled net/http transport (the request execution engine).
//
// See specs/12-middleware.md.
type Handler func(*http.Request) (*http.Response, error)

// Middleware wraps a Handler to add cross-cutting behavior (retry, tracing,
// redaction, SSRF guard, …). Middlewares compose so that the FIRST middleware
// passed is the OUTERMOST: it runs first on the way out and last on the way back.
type Middleware func(next Handler) Handler

// chain composes middlewares around base so that mw[0] is outermost:
//
//	chain(base, a, b) == a(b(base))
func chain(base Handler, mw ...Middleware) Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		if mw[i] == nil {
			continue
		}
		base = mw[i](base)
	}
	return base
}

// HandlerFromRoundTripper adapts an http.RoundTripper into a Handler.
func HandlerFromRoundTripper(rt http.RoundTripper) Handler {
	return func(req *http.Request) (*http.Response, error) {
		return rt.RoundTrip(req)
	}
}

// roundTripperFunc adapts a Handler to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// RoundTripperFromHandler adapts a Handler into an http.RoundTripper.
func RoundTripperFromHandler(h Handler) http.RoundTripper {
	return roundTripperFunc(h)
}

// FromMiddlewareFunc adapts a legacy request-mutating middlewares.MiddlewareFunc
// (func(*http.Request) (*http.Request, error)) into a Middleware, preserving the
// behavior of options.RequestOptions.Middleware.
func FromMiddlewareFunc(f middlewares.MiddlewareFunc) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			r, err := f(req)
			if err != nil {
				return nil, err
			}
			return next(r)
		}
	}
}
