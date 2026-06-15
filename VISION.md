# GoCurl — Vision & Positioning

> Paste any curl command from any API doc straight into Go. Test it in the shell,
> run the exact same command in your code. No translation, no guesswork.

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

GoCurl is at its best wherever HTTP is **glue, not the hot path**:

- **Integrating a new third-party API** — copy the doc's curl example and go.
- **Prototypes and internal tooling** — wire up a call in seconds, harden later.
- **Scripts, CI checks, and API smoke tests** — the CLI-to-code loop is the workflow.
- **Config-driven / declarative HTTP** — store curl commands as data and execute them.
- **Onboarding** — a new teammate reads the curl in the docs and already understands the code.

When you later need a typed, hand-tuned client on a high-throughput path, write one with
`net/http` — GoCurl got you to a working integration first.

## What GoCurl is

- A **curl-ergonomic HTTP client built on `net/http`**, with a CLI that shares the exact
  same syntax.
- An honest convenience layer: it parses real curl commands (string or `[]string`),
  expands variables, executes the request, and hands you a standard `*http.Response`
  plus typed helpers (`CurlString`, `CurlBytes`, `CurlJSON`, `CurlDownload`).
- Faithful to curl's HTTP/HTTPS semantics for the flags that appear in real API docs.

## What GoCurl is *not*

- **Not a `net/http` replacement.** It's built on `net/http` and embraces it. Use a
  reusable client for high-throughput services; GoCurl is for getting integrations
  working fast.
- **Not a performance play.** We make no zero-allocation / "faster than net/http"
  claims. Any performance statement in our docs will be backed by a reproducible,
  un-skipped benchmark or it won't be made.
- **Not full curl.** HTTP/HTTPS only — no FTP/SMTP/etc., and only the HTTP flags that
  show up in real-world API documentation.
- **Not a code generator or SDK builder.** The curl command is the interface.

## Design principles

1. **The doc's curl command works verbatim.** If a command from Stripe/GitHub/OpenAI
   docs doesn't run unmodified, that's a bug. Correctness of the parser is the product.
2. **Behave like a library.** No printing to stdout, no hidden global state, stream by
   default, return the response and let the caller decide.
3. **Honest by default.** Claims are measured; security behavior (redaction, validation,
   env handling) is real on the path users actually call, not just in the builder.
4. **Small, stable surface.** A focused v1 API a newcomer can learn in five minutes.
5. **Secure handling of secrets.** Tokens and credentials are redacted in verbose output
   and never leaked through side effects.

## Success looks like

A Go developer asks *"how do I call this API in Go?"*, copies the curl example from the
docs into `gocurl.CurlString(...)`, and it just works — first try, no translation.

---

*This document defines what we are building and why. The README, the CLI help, and the
implementation roadmap all derive from it. If a feature or a claim doesn't serve the
promise above, it doesn't ship.*
