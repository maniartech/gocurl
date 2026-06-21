# GoCurl — Vision & Positioning

> Paste any curl command from any API doc straight into Go. Test it in the shell,
> run the exact same command in your code. No translation, no guesswork.

> **Our motto: persuasion by example, not by marketing.** Every performance and reliability
> claim cites a test or benchmark you can run yourself — enforced by an automated doc-lint
> (`TestDocHonestyLint`). We would rather show you a proof than sell you an adjective.

## The problem we solve

Every REST API on earth documents itself with **curl**. Almost none ship a Go SDK for
their long-tail endpoints. So every Go developer integrating a new API pays the same
tax, over and over: mentally compiling a curl snippet from the docs into
`http.NewRequest`, headers, body encoding, auth, and query params — then debugging the
parts they got subtly wrong.

GoCurl removes that tax. The curl command **is** the code.

```go
resp, err := gocurl.CurlString(ctx, `
  curl https://api.stripe.com/v1/charges \
    -u $STRIPE_KEY: \
    -d amount=2000 \
    -d currency=usd
`)
```

The same string works on the command line:

```bash
gocurl https://api.stripe.com/v1/charges -u $STRIPE_KEY: -d amount=2000 -d currency=usd
```

Verify in the shell, paste into Go. One representation, not two.

## Who it's for

GoCurl fits any Go service that talks to HTTP APIs and wants the curl command to *be* the code:

- **Third-party API integration** — copy the doc's curl example and run it in production, with
  retries, timeouts, tracing, and metrics around it.
- **Service-to-service HTTP** — a pooled `Client` with circuit breaking, rate limiting, and SSRF
  protection.
- **Scripts, CI checks, and API smoke tests** — the CLI-to-code loop is the workflow.
- **Config-driven / declarative HTTP** — store curl commands as data and execute them.
- **Onboarding** — a new teammate reads the curl in the docs and already understands the code.

Built on `net/http` with **parse once, execute many** (`Prepare` a request once, `Do` it many
times over a pooled `Client`), the per-request overhead above a hand-written request is small and
constant — so the curl ergonomics never cost you reliability or predictability under load.

## Why GoCurl is the better choice

The edge is not "a nicer curl parser" — it's the **execution pipeline**. With raw
`net/http` you start from a blank `http.Request` every time and re-derive, per service, all
the things a mission-critical integration needs: an overall timeout that survives retries,
idempotency-aware retry logic, error classification, secret redaction, memory bounds against
a hostile server. Most teams get some of it wrong, and it rots as the code changes.

GoCurl receives the **curl recipe**, so it knows your intent and assembles the correct
pipeline around it automatically. That is the developer edge: you describe *what* to call (the
curl command you already tested in your shell), and GoCurl owns *how* to execute it safely
under production conditions.

Crucially, we earn the word "production-grade" with evidence, not adjectives: a two-tier
fault-injection harness, competitive benchmarks reported with their losses, an extended soak,
and an automated honesty doc-lint that fails the build if any claim lacks a named,
un-skipped test (`TestDocHonestyLint`). See the
[v1.0-readiness checklist](docs/v1-readiness.md) and [operations guide](docs/operations.md).

## What GoCurl is

- A **curl-ergonomic HTTP client built on `net/http`**, with a CLI that shares the exact
  same syntax.
- It parses real curl commands (string or `[]string`), expands variables, executes the
  request, and hands you a standard `*http.Response` plus typed helpers (`CurlString`,
  `CurlBytes`, `CurlJSON`, `CurlDownload`).
- A **production middleware stack** on the reusable, pooled `Client`: idempotency-aware
  retries with backoff, a circuit breaker, a rate limiter, tracing/metrics/logging hooks
  (with OpenTelemetry and Prometheus adapters), an opt-in SSRF guard, secret redaction, TLS
  pinning, and typed, classifiable errors.
- Faithful to curl's HTTP/HTTPS semantics for the flags that appear in real API docs.

## What GoCurl is *not*

- **Not a `net/http` replacement.** It's built on `net/http` and embraces it — GoCurl adds
  curl ergonomics and a production middleware stack (resilience, observability, security) on
  top of the standard engine rather than replacing it.
- **Not a performance play.** We make no zero-allocation / "faster than net/http" claims; the
  target is parity with a well-tuned `net/http` client. Any performance statement in our docs
  is backed by a reproducible, un-skipped benchmark or it isn't made.
- **Not full curl.** HTTP/HTTPS only — no FTP/SMTP/etc., and only the HTTP flags that
  show up in real-world API documentation.
- **Not a code generator or SDK builder.** The curl command is the interface.

## Design principles

1. **Persuasion by example, not by marketing.** This is the first principle because it
   governs the others. Every performance/reliability claim cites a named, un-skipped
   test or benchmark; an automated doc-lint fails the build if a claim ships without its
   proof. We show, we don't sell.
2. **The doc's curl command works verbatim.** If a command from Stripe/GitHub/OpenAI
   docs doesn't run unmodified, that's a bug. Correctness of the parser is the product.
3. **Behave like a library.** No printing to stdout, no hidden global state, stream by
   default, return the response and let the caller decide.
4. **Honest by default.** Claims are measured; security behavior (redaction, validation,
   env handling) is real on the path users actually call, not just in the builder.
5. **Small, stable surface.** A focused v1 API a newcomer can learn in five minutes.
6. **Secure handling of secrets.** Tokens and credentials are redacted in verbose output
   and never leaked through side effects.

## Success looks like

A Go developer asks *"how do I call this API in Go?"*, copies the curl example from the
docs into `gocurl.CurlString(...)`, and it just works — first try, no translation.

---

*This document defines what we are building and why. The README, the CLI help, and the
implementation roadmap all derive from it. If a feature or a claim doesn't serve the
promise above, it doesn't ship.*
