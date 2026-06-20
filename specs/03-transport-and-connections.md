# Transport & Connection Management

> Status: Draft for review · Spec 03

## Goals

- Make connection reuse and the full timeout taxonomy first-class, configurable, and predictable — the "reliability and predictability under load" half of the production-grade promise.
- Expose net/http transport tuning (`MaxIdleConns`, `MaxIdleConnsPerHost`, `MaxConnsPerHost`, `IdleConnTimeout`, keep-alive) and the timeout taxonomy (connect, TLS handshake, response-header, idle, overall) through functional options on the reusable `Client`, while keeping curl-string ergonomics.
- Close two real correctness gaps in the current code:
  1. `opts.ConnectTimeout` is **silently ignored on the default (non-proxy) path** — `newTransport` in `clientpool.go` never sets `DialContext`, so `--connect-timeout` only works when a proxy is used (it reaches a dialer via `proxy.NewTransport`).
  2. `opts.HTTP2Only` returns a bare `&http2.Transport{}` (clientpool.go) with **none** of the idle tuning or timeouts the HTTP/1.1 transport gets.
- Define one coherent model for **transport caching/reuse vs. per-`Client` transports**, building on the existing `transportCache` in `clientpool.go`, and specify exactly *when a fresh transport is mandatory* (custom `*tls.Config`, proxy, injected transport).
- Specify HTTP/1.1 and HTTP/2 (h2 over TLS, h2c over cleartext) selection; document HTTP/3 (quic-go) as an explicit non-goal for v1.
- Specify custom transport injection (`WithTransport`) and how it interacts with all of the above (it opts out of gocurl's tuning by design).

## Non-goals

- No HTTP/3 / QUIC in v1. `WithTransport` is the forward-compatible seam for an experimental quic-go transport later; this spec only reserves the design space.
- Not replacing net/http or adopting fasthttp. The execution engine remains `net/http` + `golang.org/x/net/http2`.
- No DNS-level features (custom resolver, `--resolve`, happy-eyeballs tuning, interface binding). Those are out of scope here; only the `net.Dialer` hooks that connect timeouts and keep-alive ride on are specified.
- No retry/circuit-breaker/rate-limit semantics — those live in the resilience spec (Spec 04). This spec only guarantees the transport is shaped so per-attempt deadlines are *enforceable*.
- No change to TLS material loading — `LoadTLSConfig` in `security.go` remains the single TLS source. This spec consumes the `*tls.Config` it produces; it does not duplicate cert/pin/version logic.

## Design

### Layering

```
Client (Spec 01)
  └── config  ──────────────► transport selection (this spec)
        ├── WithTransport(rt)        → use rt verbatim (opt out of tuning)
        ├── WithProxy(...)           → per-Client proxy transport (uncacheable)
        ├── WithTLS / custom tls.Config → fresh tuned transport (uncacheable key)
        └── (default)                → cached, shared, tuned *http.Transport
```

Today the one-shot package funcs reach the transport through `doRequest → CreateHTTPClient → getRoundTripper` (process.go, clientpool.go). Under Spec 01 the reusable `Client` owns its transport; the package-level `Curl*` funcs become thin wrappers over a lazily-initialized default `Client`. This spec defines the rules `getRoundTripper` is generalized into.

### Connection-management options

A single immutable value, derived from `config`, drives transport construction. It also becomes the cache key so two `Client`s with identical connection settings can share one underlying transport (current behavior, formalized). New fields (`MaxConnsPerHost`, `TLSHandshakeTimeout`, `ResponseHeaderTimeout`, keep-alive) do **not** exist on `options.RequestOptions` today and must be added there too so the curl/CLI path can reach them.

```go
// ConnConfig captures every connection-relevant knob. Immutable once built.
type ConnConfig struct {
    // Pool sizing (net/http defaults shown; gocurl keeps the current 100/10).
    MaxIdleConns        int // total idle conns; 0 = unlimited
    MaxIdleConnsPerHost int // idle conns per host; 0 = net/http default (2)
    MaxConnsPerHost     int // total conns per host incl. active; 0 = unlimited

    // Timeout taxonomy. Zero = "no limit at this layer" (net/http semantics),
    // except ConnectTimeout==0 means "rely on overall Timeout / context".
    ConnectTimeout        time.Duration // dial deadline  (net.Dialer.Timeout)
    TLSHandshakeTimeout   time.Duration // TLS handshake  (Transport field)
    ResponseHeaderTimeout time.Duration // request→first byte of resp header
    IdleConnTimeout       time.Duration // how long an idle conn is kept
    KeepAlive             time.Duration // TCP keep-alive probe interval
    ExpectContinueTimeout time.Duration // 100-continue wait

    // Protocol selection.
    HTTP2     bool // allow h2 upgrade over TLS (ForceAttemptHTTP2 + ConfigureTransport)
    HTTP2Only bool // force h2/h2c, no HTTP/1.1 fallback
    AllowH2C  bool // permit h2c (cleartext h2) — only meaningful with HTTP2Only

    DisableKeepAlives bool
    DisableCompression bool // set true; gocurl decompresses itself (compression.go)
}
```

Functional options that feed it (names locked by the program brief):

```go
func WithConnectTimeout(d time.Duration) Option
func WithTLSHandshakeTimeout(d time.Duration) Option
func WithResponseHeaderTimeout(d time.Duration) Option
func WithIdleConnTimeout(d time.Duration) Option
func WithKeepAlive(d time.Duration) Option
func WithMaxIdleConns(n int) Option
func WithMaxIdleConnsPerHost(n int) Option
func WithMaxConnsPerHost(n int) Option   // brief-named
func WithHTTP2(enabled bool) Option
func WithHTTP2Only(allowH2C bool) Option
func WithTransport(rt http.RoundTripper) Option // opt out of all tuning
```

### Transport construction (generalizes `newTransport`)

```go
// buildTransport produces a fully-configured, idle-tuned *http.Transport.
// It is the single place a default/TLS/proxy transport is shaped.
func buildTransport(cc ConnConfig, tlsConfig *tls.Config) (*http.Transport, error) {
    d := &net.Dialer{
        Timeout:   cc.ConnectTimeout, // FIX: wire connect timeout on default path
        KeepAlive: orDefault(cc.KeepAlive, 30*time.Second),
    }
    t := &http.Transport{
        DialContext:           d.DialContext, // was MISSING in clientpool.go
        TLSClientConfig:       tlsConfig,
        Proxy:                 http.ProxyFromEnvironment,
        ForceAttemptHTTP2:     cc.HTTP2 || !cc.HTTP2Only,
        MaxIdleConns:          orDefault(cc.MaxIdleConns, 100),
        MaxIdleConnsPerHost:   orDefault(cc.MaxIdleConnsPerHost, 10),
        MaxConnsPerHost:       cc.MaxConnsPerHost, // 0 = unlimited
        IdleConnTimeout:       orDefault(cc.IdleConnTimeout, 90*time.Second),
        TLSHandshakeTimeout:   orDefault(cc.TLSHandshakeTimeout, 10*time.Second),
        ResponseHeaderTimeout: cc.ResponseHeaderTimeout, // 0 = none
        ExpectContinueTimeout: orDefault(cc.ExpectContinueTimeout, 1*time.Second),
        DisableKeepAlives:     cc.DisableKeepAlives,
        DisableCompression:    true, // ConfigureCompressionForTransport
    }
    if cc.HTTP2 { // h2 upgrade path, preserves connection pooling
        if err := http2.ConfigureTransport(t); err != nil { return nil, err }
    }
    return t, nil
}
```

`orDefault` only applies a default when the field is the zero value, so an explicit option always wins. `ConnectTimeout==0` deliberately leaves `net.Dialer.Timeout==0` (no dial deadline) and relies on the overall `Timeout`/context — this matches curl, where `--connect-timeout` is optional.

### Transport caching / reuse (build on `transportCache`)

The current cache (clientpool.go: `transportMu` + `transportCache` keyed by `transportKey`) is retained and the key is extended to cover the new knobs. Rules:

- **Cacheable & shared** — default direct transports with no opaque custom `*tls.Config`, no proxy, not `HTTP2Only`. Key includes TLS material identity (insecure/min/max/CA/cert/key/SNI/pins, from `transportKey`) **plus** pool sizes and every timeout. Two `Client`s with identical `ConnConfig` and TLS get the same `*http.Transport`, so the idle pool is genuinely shared.
- **Fresh, uncacheable (mandatory new transport)** — any of:
  - **Custom `*tls.Config`** (`opts.TLSConfig != nil`): opaque, cannot be a reliable key (current code already does this).
  - **Proxy** (`WithProxy`/`opts.Proxy != ""`): proxy transports are built per-`Client` via `proxy.NewTransport` and are **not** added to the cache; the proxy package itself sets `DialContext`/CONNECT and its own pool defaults (factory.go).
  - **Injected transport** (`WithTransport`): used verbatim; cache bypassed entirely.
  - **`HTTP2Only`**: a forced-h2 transport (see below).
- **Lifecycle** — cached transports live for process lifetime (as today). A reusable `Client` should call `CloseIdleConnections()` on `Client.Close()`; **a `Client` must never close a transport it shares from the cache** unless it is the sole owner. Open question below.

```go
type transportCache struct {
    mu sync.Mutex
    m  map[string]*http.Transport
}
func (c *transportCache) get(key string, build func() (*http.Transport, error)) (*http.Transport, error)
```

### Proxy integration

`WithProxy` produces a per-`Client` proxy transport via the existing `proxy` package (`createProxyTransport`/`createProxyConfig` in process.go → `proxy.NewTransport` in factory.go). This spec requires that the connection knobs flow into it: `ProxyConfig.Timeout` already carries `ConnectTimeout`; `ProxyConfig` must gain `MaxIdleConns(PerHost)`, `MaxConnsPerHost`, `IdleConnTimeout`, `TLSHandshakeTimeout`, `ResponseHeaderTimeout`, `KeepAlive` so `NewTransport` (which currently hardcodes `MaxIdleConns:100`, `IdleConnTimeout:90s`, `TLSHandshakeTimeout:10s`) honors them. HTTP CONNECT tunneling and SOCKS5 dialing (httpproxy.go, socks5.go) are unchanged.

### HTTP version selection (1.0, 1.1, h2, h2c, HTTP/3)

curl's version flags are mutually exclusive and **last-wins** (`curl --http2 --http1.1` runs 1.1). gocurl mirrors that: the parser resolves the version flags to a single choice, and the transport builder applies exactly one of the rules below.

- **Default**: HTTP/1.1 with transparent h2 negotiation via ALPN when `ForceAttemptHTTP2` is true (the net/http default). No extra config needed for the common case.
- **`WithHTTP2(true)`** / `--http2`: explicitly run `http2.ConfigureTransport(t)` on the `*http.Transport` so the same transport handles both 1.1 and 2 with one pool (current `opts.HTTP2` behavior).
- **`WithHTTP2Only(allowH2C)`** / `--http2-only`, `--http2-prior-knowledge`: force HTTP/2. Replaces today's bare `&http2.Transport{TLSClientConfig: tlsConfig}` with a tuned one that sets `IdleConnTimeout`, `ReadIdleTimeout`, `PingTimeout`, and — when `allowH2C` — a `DialTLSContext`/`DialTLS` that performs a **cleartext** dial for `http://` targets (prior-knowledge h2c). h2c is opt-in and only valid for known-h2c servers; there is no h2c upgrade dance.
- **`WithHTTP11()` / `--http1.1`** (force HTTP/1.1, suppress h2): on the tuned `*http.Transport`, set `ForceAttemptHTTP2 = false`, **skip** `http2.ConfigureTransport`, and install a non-nil **empty** `TLSNextProto` map (`map[string]func(string, *tls.Conn) http.RoundTripper{}`). The empty map is load-bearing: a `nil` map lets the runtime auto-enable h2, so it must be explicitly empty, and the builder must also ensure a caller-supplied `TLSClientConfig.NextProtos` does not advertise `"h2"`. The `--http2-only` `http2.Transport` path is bypassed when a 1.x pin is set (a 1.1 pin and forced-h2 are contradictory). **Mechanism note:** the go.mod floor is `go 1.23`; the Go 1.24 `http.Transport.Protocols` API is *not* available, so the empty-`TLSNextProto` trick is the correct mechanism (revisit and switch to `Protocols` only if the floor moves to ≥1.24).
- **`--http1.0`, `-0`** (best-effort, **curl-flag / one-shot path only**): Go's `net/http` **client cannot emit an HTTP/1.0 request line** — `Request.write` hardcodes `HTTP/1.1` and ignores `req.Proto*` on the write path. gocurl therefore treats `--http1.0` as the closest reachable approximation: pin the HTTP/1.1-only transport (as above) and set `Connection: close` (`req.Close = true`, matching curl's 1.0 no-keep-alive default), plus a stderr warning (unless `--silent`) that the wire version is 1.1. There is **no `WithHTTP10()` Client option**: on a reusable Client, `Connection: close` on every request would defeat the connection pool — a foot-gun. True HTTP/1.0-on-the-wire is **out of scope** (same bucket as HTTP/3) — it would require a hand-written `RoundTripper` over a raw `net.Conn`, abandoning the pool/redirect/TLS plumbing.
- **HTTP/3**: documented future add-on. `WithTransport` is the injection seam; gocurl ships no quic-go dependency in v1.

## Behavior & edge cases

- **Connect-timeout regression (must-fix)**: after this spec, `--connect-timeout 5` / `WithConnectTimeout(5s)` enforces a 5s dial deadline on the **default** path, not just the proxy path. A unit/integration test must assert dialing a black-hole address fails within ~the connect timeout, with no proxy configured.
- **Timeout layering**: `WithConnectTimeout` (dial) → `WithTLSHandshakeTimeout` → `WithResponseHeaderTimeout` are per-phase and independent of the overall `Timeout`/context. `determineClientTimeout` (process.go) keeps its rule: a context deadline wins over `Timeout` (returns 0 to avoid nested overall timeouts). Per-phase transport timeouts always apply regardless. Per-attempt deadlines (Spec 04) wrap the context, not these fields.
- **`MaxConnsPerHost > 0`**: requests beyond the limit block on `DialContext` until a conn frees up or the context/`ConnectTimeout` fires. Document that this introduces backpressure (the intended behavior) and that a too-low value can serialize throughput.
- **`MaxIdleConnsPerHost==0`** maps to net/http's default of 2, which is surprisingly low; gocurl keeps its historical default of 10 via `orDefault`. Setting it explicitly to a value below `MaxConnsPerHost` causes conns to be closed rather than pooled after each request — call this out in docs.
- **Custom `*tls.Config` + cache**: providing `WithTLS`/opaque config always yields a fresh transport (no sharing) — a deliberate correctness-over-reuse tradeoff (current behavior).
- **`WithTransport` precedence**: an injected `http.RoundTripper` is used as-is. All `With*` connection/timeout/HTTP2 options are then **ignored** (we cannot mutate an arbitrary `RoundTripper`); `New` returns an error if conflicting transport-shaping options are combined with `WithTransport`. The overall `Timeout` (set on `http.Client`) and redirect policy still apply because they live above the transport.
- **Compression invariant**: `DisableCompression` stays `true` on every gocurl-built transport (gocurl decompresses via `compression.go`/`DecompressResponse`). An injected transport is the caller's responsibility.
- **Concurrency**: `*http.Transport` is safe for concurrent use; it is fully configured under the cache lock before being shared (current invariant in clientpool.go) and never mutated per-request. This must remain true.
- **HTTP2Only over `http://` without h2c**: error early ("HTTP/2-only requires TLS unless h2c is enabled").
- **HTTP/1.x pin in the cache key**: the version pin (`HTTP10`/`HTTP11`) MUST feed `transportKey`, otherwise a 1.1-pinned request could reuse a cached h2-capable transport (or vice versa) — a silent protocol-version bug. The one-shot path keys per-command; the `Client` path keys per-`config`.
- **`--http1.0` is an approximation, not 1.0-on-the-wire**: the wire request line stays `HTTP/1.1`; only the connection semantics (`Connection: close`, no `Expect: 100-continue`) and the no-h2 pin are honored. This deviation is surfaced via the one-time warning and documented in `specs/02` and the README, so a pasted `--http1.0` command does not silently mislead.
- **Version-flag mutual exclusion**: last-wins (curl-faithful). The parser clears the other version bits when it sees a version flag, so at most one of `HTTP2`/`HTTP2Only`/`HTTP11`/`HTTP10` is ever set; the transport builder resolves a single rule with no ambiguous combination.
- **Idle eviction**: `IdleConnTimeout` governs how long pooled conns survive; servers may close sooner. Retries (Spec 04) must treat a closed-idle-conn error on an idempotent request as retryable.

## Acceptance criteria / Definition of Done

- [ ] `options.RequestOptions` gains `MaxIdleConns`, `MaxIdleConnsPerHost`, `MaxConnsPerHost`, `TLSHandshakeTimeout`, `ResponseHeaderTimeout`, `KeepAlive`, `DisableKeepAlives` (with JSON tags + `Clone` coverage), and the matching `With*` options exist on `Client`.
- [ ] `buildTransport` (generalizing `newTransport`) sets `DialContext` from a `net.Dialer` whose `Timeout = ConnectTimeout` on the **default, non-proxy** path. A test proves `--connect-timeout`/`WithConnectTimeout` fires without a proxy (the current gap).
- [ ] All new pool/timeout fields are plumbed onto both the default `*http.Transport` and the proxy transport (`proxy.ProxyConfig` + `proxy.NewTransport` extended).
- [ ] The transport cache key (`transportKey`) is extended to include pool sizes and all timeouts; two `Client`s with identical `ConnConfig`+TLS share one `*http.Transport` (verified by pointer identity in a test), and differing configs do not.
- [ ] Custom `*tls.Config`, proxy, `HTTP2Only`, and `WithTransport` each bypass the cache (verified).
- [ ] `WithHTTP2Only(allowH2C)` produces a tuned `http2.Transport` (idle/ping timeouts set) and supports prior-knowledge h2c for `http://` when `allowH2C` is true; `http://` + h2c-off errors clearly.
- [ ] `options.RequestOptions` gains `HTTP10` and `HTTP11` bools (JSON tags + `Clone` coverage — shallow copy already handles scalars), with `--http1.1` → `HTTP11`, `--http1.0`/`-0` → `HTTP10` parsed in `convert.go` `processSimpleFlag`, builder `SetHTTP10`/`SetHTTP11`, and the `Client` option `WithHTTP11()` (no `WithHTTP10` — see the foot-gun note above).
- [ ] `--http1.1`/`WithHTTP11` forces HTTP/1.1: a test against an h2-capable (`EnableHTTP2`) TLS test server asserts `resp.Proto == "HTTP/1.1"` (not `HTTP/2.0`), via the empty-`TLSNextProto` mechanism, with `ForceAttemptHTTP2 == false` and no `http2.ConfigureTransport`.
- [ ] `--http1.0`/`-0` pins HTTP/1.1, sets `Connection: close`, and emits the wire-version warning; a test asserts `resp.Proto == "HTTP/1.1"` and the request carried `Connection: close`.
- [ ] The version pin is included in `transportKey`; a 1.1-pinned client never shares a transport with an h2-capable one (pointer-identity test).
- [ ] Version flags are mutually exclusive last-wins; a parser test proves `--http2 --http1.1` ends with only `HTTP11` set.
- [ ] `WithTransport(rt)` uses `rt` verbatim; combining it with transport-shaping options returns an error from `New`.
- [ ] `DisableCompression == true` on every gocurl-built transport (default, TLS, proxy, h2-only); regression test.
- [ ] `Client.Close()` calls `CloseIdleConnections()` only on transports it solely owns; cache-shared transports are not force-closed. (Pending the ownership decision below.)
- [ ] Race-clean under `go test -race`; no per-request transport mutation.
- [ ] README/docs document the timeout taxonomy and pool knobs with the honest framing (parity + predictability, no perf superiority claims).

## Dependencies

- **Spec 01 (Client & options)** — defines `Client`, `config`, `New`, `Option`, `WithTransport`, `Close`; this spec consumes them.
- **Spec 02 (Request / Prepare)** — the prepared `Request` carries no transport state; transport lives on `Client`. Confirms parse-once does not re-tune per request.
- **Spec 04 (Resilience)** — consumes per-phase timeouts and the closed-idle-conn-is-retryable rule; owns per-attempt deadlines.
- Builds directly on existing files: `clientpool.go` (`getRoundTripper`, `newTransport`, `transportKey`, `transportCache`), `process.go` (`CreateHTTPClient`, `createProxyTransport`, `determineClientTimeout`), `security.go` (`LoadTLSConfig`), `compression.go` (`ConfigureCompressionForTransport`), `proxy/factory.go` (`NewTransport`, `ProxyConfig`), `proxy/httpproxy.go`, `proxy/socks5.go`, `proxy/no-proxy.go`, `options/options.go`.

## Open questions / decisions to confirm in review

- **Shared-transport ownership on `Close()`**: if N `Client`s share one cached transport, when is it safe to `CloseIdleConnections()`? Proposed: refcount cache entries, or simply never close cache-shared transports (process-lifetime pool) and only close per-`Client` (proxy/TLS/injected) transports. Need a decision.
- **Default `MaxConnsPerHost`**: keep `0` (unlimited, current net/http default) or pick a safe ceiling for "mission-critical" defaults? Unlimited is least surprising vs. existing behavior; a default cap is safer under load. Recommend keeping `0` and documenting the knob.
- **Keep `HTTP2Only` distinct from `HTTP2`?** curl has no exact analog; `HTTP2Only` means "no HTTP/1.1 fallback." Confirm we keep both flags rather than collapsing to one tri-state.
- **h2c surface**: is prior-knowledge h2c (no upgrade dance) sufficient for v1, or do we need the HTTP Upgrade mechanism too? Proposed: prior-knowledge only.
- **Proxy transport caching**: currently proxy transports are rebuilt per `Client` and never cached. For a long-lived `Client` this is fine (built once). Confirm we do **not** add proxy transports to the global cache (keying credentials is risky).
- **Exposing the raw `*http.Transport`**: should `Client` offer a read-only accessor / `TransportStats()` for observability (idle conn counts), or is that deferred to Spec 06 (observability)? Recommend deferring.
