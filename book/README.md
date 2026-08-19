# GoCurl book

The maintained manuscript and runnable examples live in [`../book2`](../book2/README.md).
This directory is intentionally a compatibility landing page for links that used the
original `book/` path. Keeping one manuscript prevents corrections from being copied into
two trees and drifting again.

## Release compatibility

- GoCurl requires **Go 1.25 or later**. This baseline permits the patched networking
  dependencies used by the current security release.
- `Curl*` and `Execute` use the one-shot engine. They do not inherit managed-client
  middleware, observability, circuit breaking, rate limiting, or SSRF protection.
- `Client.Prepare`/`Client.Do` is the production path when those guarantees are needed.
- Modern middleware uses `gocurl.Handler` and `gocurl.Middleware`. The older
  `middlewares.MiddlewareFunc` remains a request-mutator compatibility API.

## Production composition

For a managed client, configure resilience, observability, and SSRF protection as client
options. Retry defaults are idempotency-aware, so POST, PATCH, and CONNECT are sent once
unless an `Idempotency-Key`, `AllowMethods`, or a custom `Retryable` decision opts in.

```go
client, err := gocurl.New(
    gocurl.WithRetry(gocurl.RetryPolicy{
        MaxAttempts: 3,
        MaxElapsed:  10 * time.Second,
    }),
    gocurl.WithSSRFGuard(gocurl.DefaultSSRFPolicy()),
    gocurl.WithHooks(gocurl.Hooks{
        OnError: func(ctx context.Context, req *http.Request, err error, kind gocurl.Kind) {
            // Forward only redacted, bounded-cardinality data to an observability sink.
        },
    }),
)
if err != nil {
    return err
}
defer client.Close()
```

SSRF protection is opt-in to preserve legitimate access to private services. Once enabled,
it checks the initial destination and every redirect, rejects resolution failure, and pins
the socket dial to validated IPs to prevent DNS rebinding. An allow-list entry weakens that
boundary and should be reviewed like a firewall rule.

For standalone `net/http` composition, use `gocurl.Retry`, `gocurl.Observe`, and
`gocurl.SSRFGuard` with `gocurl.HandlerFromRoundTripper`. A raw handler cannot enforce an
SSRF dial pin unless it explicitly honors the pinning contract, so opaque transports fail
closed.

See the maintained [API reference](../book2/API_REFERENCE.md) and
[benchmark methodology](../docs/benchmarking.md) for the complete release guidance.
