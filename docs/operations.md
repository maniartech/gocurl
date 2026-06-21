# Operating gocurl in production

This is the operations contract for running gocurl in a mission-critical service:
the timeout model, how to size pools and retry budgets, a failure-mode playbook for
every error `Kind`, the observability runbook, and the untrusted-server memory bounds.

Every reliability claim here is backed by a named, un-skipped test in the suite — cited
inline so you can run the proof yourself (`go test -run <Name> .`). This is enforced by
the honesty doc-lint (`TestDocHonestyLint`).

> **Motto: persuasion by example, not by marketing.** If a sentence below makes a promise,
> it names the test that keeps it. Run them.

## Timeout taxonomy

gocurl has **layered** timeouts; understand which one bounds what:

| Knob | Bounds | Default |
|---|---|---|
| `WithTimeout` / `--max-time` | The **whole operation**, including all retries and their backoff sleeps | none (unbounded) |
| `WithConnectTimeout` / `--connect-timeout` | TCP connect (dial) | none |
| `WithResponseHeaderTimeout` | Time from request written to response headers received | none |
| `WithTLSHandshakeTimeout` | TLS handshake | 10s |
| Per-attempt deadline (`RetryPolicy.PerAttempt`) | One attempt's time-to-response | none |
| `context.WithTimeout(ctx, …)` | The whole operation (same layer as `WithTimeout`) | caller's choice |

**The critical one:** `WithTimeout` is an **overall** bound — it is layered *under*
`http.Client.Timeout` so a retry storm cannot run `MaxAttempts × backoff` past your
deadline. Proven by `TestFault_OverallRetryBudget` (Client path) and
`TestFault_OneShotMaxTimeBoundsRetries` (one-shot path). Always set it in production.

A per-attempt deadline that fires while the overall context is still alive is a
**retryable** timeout; a fired overall deadline is terminal. See
`TestFault_PerAttemptDeadline`.

## Connection pool & capacity planning

gocurl's default transport (per `Client`): `MaxIdleConns=100`, `MaxIdleConnsPerHost=10`,
`MaxConnsPerHost=0` (**unlimited**), `IdleConnTimeout=90s`.

- **`MaxConnsPerHost` is unlimited by default.** Under a burst to one host this can open
  connections without bound and exhaust file descriptors (EMFILE). For a mission-critical
  service set `WithMaxConnsPerHost(n)` to apply backpressure: excess requests **block then
  drain** rather than dialing without limit. Proven by `TestClient_Soak_Backpressure` and
  `TestFaultT2_PoolExhaustionSerializes` (no deadlock; the server never sees more than `n`
  concurrent connections).
- **Reuse one `Client`.** It pools connections; constructing a `Client` per request discards
  the pool. Connection reuse is proven by `TestClient_Do_ReusesConnections`.
- **Drain and close every body** (`io.Copy(io.Discard, resp.Body); resp.Body.Close()`).
  A partially-read body poisons keep-alive reuse. The convenience helpers do this for you.

## Retry budget tuning

Retries are **idempotency-aware**: only idempotent methods and retryable outcomes
(`KindConnect`, `KindTimeout`, and configured retryable statuses) are retried. A POST is
not retried unless you opt in.

- Set `WithRetryBudget` to cap the **fraction** of requests that may be retries — this
  prevents a retry storm from amplifying load on an already-struggling dependency.
- `WithTimeout` bounds the whole retry loop (above). Backoff sleeps are clamped to the
  remaining budget, so a retry never sleeps past your deadline.
- HTTP/2 `GOAWAY`/`RST_STREAM` are classified as retryable connection errors (h2 is the
  default TLS path). Proven by `TestFault_H2ErrorsRetried`.

## Failure-mode playbook (per `Kind`)

Every failure is a classifiable `*GocurlError`; branch with `errors.Is` / `KindOf`.

| `Kind` | Sentinel | What it means | Operator action |
|---|---|---|---|
| `KindConnect` | `ErrConnect` | Dial/DNS/connection reset/h2 GOAWAY | Retryable. Check target reachability, DNS, pool exhaustion. Proven: `TestFaultT2_DNSFailure`. |
| `KindTimeout` | `ErrTimeout` | Deadline/per-attempt timeout | Retryable. Tune timeouts; check dependency latency. Proven: `TestFaultT2_ResponseHeaderTimeout`. |
| `KindTLS` | `ErrTLS` | Handshake/cert/pin failure | **Not** retryable. Check certs, pins, TLS version. |
| `KindBodyRead` | `ErrBodyRead` | Premature EOF / over-limit body | Check the server; a body cap may have tripped. Proven: `TestFaultT2_PrematureBodyEOF`, `TestFault_BufferingHelpersBoundedAgainstBomb`. |
| `KindRetryExhausted` | — | All retry attempts failed | Inspect the wrapped cause; the dependency is down or budget too low. |
| (redirect cap) | `ErrTooManyRedirects` | `--max-redirs` exceeded | Loop or misconfig. Proven: `TestFault_ClientRedirectCapClassifiable`. |
| (circuit open) | `ErrCircuitOpen` | Breaker tripped, fast-failing | The dependency is unhealthy; the breaker is protecting it. Proven: `TestFault_CircuitBreakerTrips`. |
| (SSRF) | `ErrSSRFBlocked` | Target blocked by the SSRF guard | Expected when the guard is on; the target resolved to a blocked range. |

`gocurl.IsRetryable(err)` reports whether a retry could help.

## Untrusted-server memory bounds

When you call an endpoint you do not control, gocurl bounds what it buffers **on your
behalf**:

- **Response headers** are capped at 1 MiB per response (`MaxResponseHeaderBytes`),
  tighter than net/http's 10 MiB default.
- **Buffered convenience reads** (`CurlString`, `CurlBytes`, `CurlJSON`) are capped at
  64 MiB (`defaultBufferedResponseLimit`) so a decompression bomb on a `--compressed`
  response cannot OOM you. Proven by `TestFault_BufferingHelpersBoundedAgainstBomb`.
- **Streaming is deliberately unbounded** (`Curl`, `CurlDownload`): you control the
  memory by streaming. To handle a body larger than 64 MiB, stream it instead of buffering.
- **Secrets never leak on failure paths.** Credentials in a URL/header are redacted in
  errors and verbose output even when a request fails mid-retry. Proven by
  `TestFault_NoSecretLeakOnFailurePaths`.

## Observability runbook

Wire `WithTracer` / `WithMetrics` / `WithLogger` / `WithHooks` (OpenTelemetry and
Prometheus adapters ship in `observability/otel` and `observability/prometheus`). The
instrumentation is **only present when configured** (zero overhead otherwise) and does not
leak under sustained load — proven by the instrumented arm of `TestClient_Soak`, which also
publishes the per-`Do` allocation delta the instrumentation adds.

- A span brackets one **logical** request (all retry attempts).
- `IncInFlight(+1/-1)`, `IncRetry`, and `IncError(kind, …)` give you saturation, retry-rate,
  and error-rate-by-kind — the three signals to alert on.
- A request ID (`WithRequestIDFunc`) is generated when absent, propagated across retries,
  and attached to spans/logs.

## Easy start → production checklist

The zero-config one-liner works on day one — and still surfaces a correct classified error
on the failure paths (`TestFault_EasyCurlStillWorks`). Before you depend on it in
production, set:

- [ ] **`WithTimeout`** — an overall deadline. Non-negotiable.
- [ ] **`WithMaxConnsPerHost`** — bound the pool; avoid EMFILE under burst.
- [ ] **`WithRetry` + `WithRetryBudget`** — idempotency-aware retries with a budget cap.
- [ ] **`WithCircuitBreaker`** — fast-fail an unhealthy dependency.
- [ ] **Observability** — tracer + metrics + logger; alert on error-by-kind and retry-rate.
- [ ] **Reuse one `Client`**; drain and close every body.
- [ ] **`WithSSRFGuard`** if any URL is influenced by user input.
- [ ] Pin a version; read the [CHANGELOG](../CHANGELOG.md) on upgrade.

See [SECURITY.md](../SECURITY.md) for the threat model and
[benchmarking.md](benchmarking.md) for the performance methodology.
