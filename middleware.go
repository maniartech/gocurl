package gocurl

import (
	"fmt"
	"net/http"

	"github.com/maniartech/gocurl/middlewares"
	"github.com/maniartech/gocurl/options"
)

// Handler executes a single HTTP request and returns its response. It has the
// same shape as http.RoundTripper.RoundTrip but as a function, so behaviors can
// be composed. The innermost Handler in a Client's chain is backed by the
// Client's pooled net/http transport (the request execution engine).
//
// See specs/12-middleware.md.
type Handler func(*http.Request) (*http.Response, error)

// Middleware wraps a Handler to add cross-cutting behavior such as retry,
// observability, circuit breaking, rate limiting, or SSRF checks. Middlewares
// compose so that the FIRST middleware
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

// HandlerFromRoundTripper adapts an http.RoundTripper into a Handler. A chain
// built from this adapter includes only the Middleware values the caller
// explicitly composes; Client options, cookies, redirect policy, and default
// headers are not applied.
func HandlerFromRoundTripper(rt http.RoundTripper) Handler {
	rt = withSSRFDialPinning(rt)
	return func(req *http.Request) (*http.Response, error) {
		return rt.RoundTrip(req)
	}
}

// roundTripperFunc adapts a Handler to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// RoundTripperFromHandler adapts a Handler into an http.RoundTripper. The
// surrounding http.Client, rather than gocurl, owns redirects, cookies, and its
// Timeout. Compose Retry, Observe, and any protection middleware explicitly.
func RoundTripperFromHandler(h Handler) http.RoundTripper {
	return roundTripperFunc(h)
}

type handlerHTTPClient struct{ next Handler }

func (c handlerHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.next(req)
}

// Retry returns a body-replay-aware Middleware that applies policy to requests
// passing through an exported Handler chain. The default policy remains
// idempotency-aware: POST, PATCH, and CONNECT are sent once unless AllowMethods,
// Retryable, or an Idempotency-Key explicitly opts them in.
//
// Retry should normally be the innermost resilience middleware so a circuit
// breaker and rate limiter observe one final logical outcome rather than every
// attempt. Its elapsed budget covers attempts and backoff; the surrounding
// request context or http.Client should provide any overall timeout.
func Retry(policy RetryPolicy) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			if req == nil {
				return nil, fmt.Errorf("retry middleware: nil request")
			}
			// RetryPolicy is documented as an immutable value and may be shared by
			// concurrent SDK requests. Copy it before filling the internal replay cap
			// so this middleware never mutates caller-owned state. Standalone chains
			// use the same conservative 1 MiB default as the one-shot engine.
			effective := policy
			effective.maxReplayBytes = DefaultMaxReplayBytes

			// The shared retry engine accepts RequestOptions only to enrich a final
			// RetryError with its sanitized URL. Request execution remains delegated
			// to next through handlerHTTPClient; no Client-only options leak into the
			// standalone composition path.
			opts := options.NewRequestOptions("")
			if req.URL != nil {
				opts.URL = req.URL.String()
			}
			return executeWithRetries(handlerHTTPClient{next: next}, req, opts, &effective, nil)
		}
	}
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
