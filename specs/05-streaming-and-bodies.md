# Streaming & Request/Response Bodies

> Status: Draft for review · Spec 05

## Goals

- Formalize the **response streaming contract** that the high-level `Curl*` functions already
  implement (`executeOpts` in `api.go` returns the live `*http.Response`; the caller reads and
  closes `resp.Body`), so body ownership is unambiguous and documented.
- Make **request-side uploads stream**: `-T/--upload-file`, `-d @file`, and `-F field=@file`
  must not buffer the whole file into memory. Today they do (`os.ReadFile` into the
  `string`-typed `opts.Body` in `processDataFlags`, and `bytes.Buffer` in `createMultipartBody`).
- Define **first-class body sources** so a `Request` can carry a streaming body (file, reader,
  or in-memory bytes) instead of only `options.RequestOptions.Body string`.
- Specify correct interaction of streaming bodies with **multipart/form-data**, **chunked
  transfer encoding**, **SSE / incremental response consumption**, and **HTTP trailers**.
- Build the **response body size limit** into a single, correct streaming guard, fixing the
  `limitedBody` truncation/error edge cases in `process.go`.
- Resolve the **streaming ↔ retry-replay tension** with explicit, documented rules: a body that
  cannot be rewound makes the request non-replayable, and the engine must refuse to silently
  retry it.
- State **zero-surprise ownership rules**: exactly who closes each body, what happens on error,
  and what the library never does (no implicit `io.ReadAll`, no writes to `os.Stdout`).

## Non-goals

- **WebSockets** — explicitly out of scope.
- HTTP/3 / QUIC streaming (deferred; quic-go is a future add-on per project direction).
- Replacing `net/http` as the execution engine. All streaming rides on `*http.Request.Body`,
  `http.Response.Body`, `Request.GetBody`, and Go's chunked/trailer support.
- Bidirectional / full-duplex streaming semantics beyond what `net/http` offers.
- Progress callbacks / throttling for uploads & downloads (candidate for a later observability
  or transfer spec; noted under Open questions).
- Re-specifying the deprecated `Process`/`Execute` buffering path (Spec 02 territory); this spec
  only defines how the new streaming model coexists with it.

## Design

### 1. Body sources (the "parse once" body artifact)

`options.RequestOptions.Body` is a `string`, which structurally forces buffering and conflates
"I have bytes" with "I have a file to stream". Introduce an explicit body source on the prepared
`Request` (Spec 01) while keeping `Body string` working for back-compat.

```go
// BodySource produces the request body. It must be able to produce the body
// once (Open) and report whether it can produce it AGAIN (Rewindable) so the
// retry engine knows if a replay is safe.
type BodySource interface {
    // Open returns a fresh ReadCloser positioned at the start of the body.
    // Called once per attempt. The engine closes the returned ReadCloser.
    Open() (io.ReadCloser, error)
    // Len reports the content length, or -1 if unknown (forces chunked).
    Len() int64
    // Rewindable reports whether Open may be called more than once. A network
    // pipe or one-shot io.Reader returns false; a file or []byte returns true.
    Rewindable() bool
}
```

Concrete sources (constructors live alongside the body-building code that today is in
`createRequestBody`/`createMultipartBody` in `process.go`):

```go
func BytesBody(b []byte) BodySource                 // Rewindable, Len = len(b)
func StringBody(s string) BodySource                // current opts.Body path
func FileBody(path string) BodySource               // streams via os.Open; -T, -d @file, -F @file
func ReaderBody(r io.Reader, length int64) BodySource // length<0 => chunked; Rewindable=false
func MultipartBody(parts []Part) BodySource         // streams via io.Pipe + multipart.Writer
```

`FileBody.Open` does `os.Open` (NOT `os.ReadFile`) and `Len` uses `os.Stat`. `ReaderBody`
wraps an arbitrary `io.Reader`; if it is not already `io.ReadSeeker`/`*bytes.Reader`, it is
**not rewindable**.

### 2. Wiring into the request build

`CreateRequest` (in `process.go`) currently calls `createRequestBody` which returns an
`io.Reader`. Change it to consult a `BodySource` and populate `http.Request.GetBody` so that
`net/http` (redirects) and our retry loop can both obtain fresh bodies:

```go
func buildBody(req *http.Request, src BodySource) error {
    rc, err := src.Open()
    if err != nil { return err }
    req.Body = rc
    req.ContentLength = src.Len() // -1 => chunked transfer encoding
    if src.Rewindable() {
        req.GetBody = func() (io.ReadCloser, error) { return src.Open() }
    }
    return nil
}
```

When `ContentLength == -1`, `net/http` automatically uses `Transfer-Encoding: chunked`. When a
known length is set, a fixed `Content-Length` is sent. This replaces the implicit
`strings.NewReader(opts.Body)` path while preserving it for `StringBody`.

### 3. Streaming multipart

`createMultipartBody`/`addFileToMultipart` today buffer everything into a `bytes.Buffer`.
Replace with an `io.Pipe`-backed streaming writer so file parts stream from disk:

```go
func (m *multipartSource) Open() (io.ReadCloser, error) {
    pr, pw := io.Pipe()
    mw := multipart.NewWriter(pw)
    m.contentType = mw.FormDataContentType() // set before first Read
    go func() {
        err := m.writeParts(mw)          // CreateFormFile + io.Copy(part, file)
        cerr := mw.Close()
        if err == nil { err = cerr }
        pw.CloseWithError(err)           // propagate to the reader side
    }()
    return pr, nil
}
```

`Content-Type` (the multipart boundary) is computed at `Open` time and applied in
`applyHeaders`. A streaming multipart body is **not rewindable** (the pipe is one-shot) unless
every part is itself rewindable and small; default to `Rewindable() == false`.

### 4. Response streaming contract (formalize existing behavior)

The contract already implemented by `executeOpts` and `doRequest`:

- `client.Do` / package `Curl*` return the **live** `*http.Response`. The body is **not**
  read, **not** decompressed into memory, and **never** written to `os.Stdout` (that side
  effect lives only in the deprecated `Process`/`HandleOutput`).
- **The caller owns `resp.Body` and must `Close()` it.** Convenience wrappers
  (`CurlString`, `CurlBytes`, `CurlJSON`, `CurlDownload`) read-then-close internally and the
  caller does not touch the body.
- Decompression (`DecompressResponse`) wraps the body in a streaming decoder
  (`pooledGzipReader`, `deflateReader`, `pooledBrotliReader`) whose `Close()` both closes the
  underlying body and returns the pooled reader. Closing the returned body is sufficient and
  required.
- On any error returned from `Do`, `resp` is `nil` and there is no body to close.

### 5. SSE / incremental consumption

No new transport feature is required — streaming is already preserved. The spec mandates that
nothing in the pipeline defeats it for `text/event-stream`:

- Do **not** auto-buffer or auto-decode when `Content-Type: text/event-stream`.
- The caller consumes incrementally, e.g. `bufio.NewScanner(resp.Body)`.
- A `ResponseBodyLimit` MUST NOT be applied (or must be explicitly opt-out) for unbounded
  streams; see Open questions.

### 6. Trailers

`net/http` exposes response trailers in `resp.Trailer` only **after** the body is fully read.
The contract: trailers are valid to read once `resp.Body` returns `io.EOF`. The library does
not strip or pre-read trailers. For requests, an advanced API may set `req.Trailer`; not part
of the curl surface in v1.

### 7. Response body size limit (consolidate `limitedBody`)

Replace the ad-hoc `limitedBody` (process.go) with a correct streaming limiter used by
`executeOpts`:

```go
type limitedBody struct {
    rc    io.ReadCloser
    limit int64
    read  int64
}

func (l *limitedBody) Read(p []byte) (int, error) {
    if len(p) == 0 {
        return 0, nil
    }
    remaining := l.limit - l.read
    if remaining <= 0 {
        // Already delivered exactly `limit` bytes — probe ONE byte to tell a body
        // that is exactly at the cap (clean EOF -> success) from one that runs over,
        // without buffering more than that single byte.
        var probe [1]byte
        n, err := l.rc.Read(probe[:])
        if n > 0 {
            return 0, l.tooLargeErr()
        }
        return 0, err
    }
    if int64(len(p)) > remaining+1 {
        p = p[:remaining+1] // never pull more than one byte past the cap
    }
    n, err := l.rc.Read(p)
    l.read += int64(n)
    if l.read > l.limit {
        n -= int(l.read - l.limit) // hide the probe byte from the caller
        l.read = l.limit
        return n, l.tooLargeErr()
    }
    return n, err
}

// tooLargeErr classifies the over-limit case as KindBodyRead (errors.Is(err,
// ErrBodyRead) / KindOf), the consolidated body-read failure kind.
func (l *limitedBody) tooLargeErr() error {
    return &GocurlError{Op: "body read", Kind: KindBodyRead,
        Err: fmt.Errorf("response body size exceeds limit of %d bytes", l.limit)}
}
```

This fixes the two issues in the original implementation: it caps the slice handed to the
underlying `Read` (so it pulls at most one byte — the overflow probe — past the cap rather than a
whole buffer chunk), and it returns a typed, classifiable error (Spec 03) instead of a bare
`fmt.Errorf`. The error uses the existing `KindBodyRead` taxonomy — body-read failures, including
the over-limit/truncation case, are consolidated under one kind; there is no separate
`ErrBodyTooLarge`. A body exactly at `limit` succeeds (the probe reads a clean EOF); the first
byte over errors. The CLI `--max-filesize`-style flag maps to `ResponseBodyLimit`.

### 8. Streaming ↔ retry replay

`executeWithRetries`/`bufferRequestBody` (retry.go) currently calls `io.ReadAll` on the body
and re-wraps it in a `bytes.Reader` so each attempt gets a fresh copy. This buffers the entire
upload — defeating streaming. New rule:

- The retry engine obtains replay bodies via **`req.GetBody`** (set from a rewindable
  `BodySource`), not `io.ReadAll`.
- If `req.GetBody == nil` (non-rewindable body, e.g. `ReaderBody` or streaming multipart):
  - The **first** attempt streams normally.
  - A retry MUST NOT be attempted after the body has begun streaming. The engine returns the
    original error wrapped as **non-retryable** (`ErrBodyNotReplayable`), even if the policy
    would otherwise retry.
- This combines with the idempotency rule (Spec 04): non-idempotent methods are not retried by
  default anyway; a non-rewindable body makes *any* method non-replayable.
- Small bodies (`StringBody`/`BytesBody`) remain trivially rewindable with no extra cost.

## Behavior & edge cases

- **`-T -` / `-d @-` (stdin):** stdin is a one-shot stream → `ReaderBody(os.Stdin, -1)`,
  `Rewindable() == false`, chunked encoding, no retry after streaming starts.
- **Empty body:** `Len() == 0` with a non-nil source still sets `Content-Length: 0`; a nil
  source uses `http.NoBody` (matches current `bufferRequestBody` guard on `http.NoBody`).
- **File shrinks/grows between attempts:** `FileBody.Open` re-stats; if length differs from the
  `Content-Length` already sent, the attempt fails fast rather than sending a truncated body.
- **File disappears before first byte:** `Open` error surfaces from `buildBody` before the
  request is dispatched.
- **Redirects with a body:** `net/http` calls `req.GetBody` for 307/308 replays; non-rewindable
  bodies cannot follow such redirects and the redirect fails with a clear error rather than
  sending an empty body.
- **Decompression + limit:** `ResponseBodyLimit` is applied to the **decompressed** stream
  (the limiter wraps the body after `DecompressResponse`), matching `executeOpts` ordering, so
  the cap reflects bytes the caller actually reads.
- **Limit boundary:** a body exactly equal to `limit` succeeds; the first byte over the limit
  errors. (Current code's `+1` overflow trick in `readBodyWithLimit` is unnecessary in the
  streaming limiter.)
- **Caller forgets to close:** documented as a connection-pool leak (the transport cannot reuse
  the connection). The library does not auto-close; convenience wrappers do.
- **Partial read then close:** allowed; `net/http` drains/discards as needed. `CurlDownload`
  streams via `io.Copy` and closes — large downloads never fully buffer.
- **SSE never reaching EOF:** caller is responsible for cancelling via `ctx`; a `ResponseBodyLimit`
  on an SSE stream would abort it mid-stream and is therefore discouraged.

## Acceptance criteria / Definition of Done

- [ ] `-T file`, `-d @file`, and `-F field=@file` stream from disk; a memory-usage test uploading
      a large (e.g. ≥256 MB) file shows allocation that does **not** scale with file size.
- [ ] `BodySource` exists with `BytesBody`, `StringBody`, `FileBody`, `ReaderBody`,
      `MultipartBody`, and a `Rewindable()` that returns `true` only for replayable sources.
- [ ] `CreateRequest`/`buildBody` sets `req.GetBody` for rewindable sources and `ContentLength = -1`
      (chunked) for unknown-length sources; verified by an httptest server asserting
      `Transfer-Encoding: chunked` vs a fixed `Content-Length`.
- [ ] Multipart uploads stream via `io.Pipe`; no `bytes.Buffer` of the full file remains in
      `createMultipartBody`.
- [ ] Retry engine uses `req.GetBody` (not `io.ReadAll`) for replays; a non-rewindable body that
      has started streaming is **never** retried and returns a classifiable `ErrBodyNotReplayable`.
- [ ] Response streaming contract documented on `Curl`/`client.Do`: caller closes `resp.Body`;
      convenience wrappers close it themselves; no `os.Stdout` writes outside deprecated `Process`.
- [x] `limitedBody` caps the underlying read slice (pulls at most one overflow-probe byte past
      the cap), never over-reads, and returns a typed, classifiable `KindBodyRead` error
      (`errors.Is(err, ErrBodyRead)`); boundary test (size == limit passes, size == limit+1 fails)
      — `TestLimitedBody_*` + `TestResponseBodyLimit_{ExactLimit,OneByteOver}`.
- [x] `ResponseBodyLimit` applies to the decompressed stream — `TestResponseBodyLimit_DecompressedStream`
      gzips a 64 KiB body (compressed < cap) and verifies the limit still fires on the inflated bytes.
- [ ] SSE test: a `text/event-stream` endpoint is consumed event-by-event with
      `bufio.Scanner` without the library buffering the full stream.
- [ ] Trailers: a test reads `resp.Trailer` successfully after full body read and confirms it is
      empty before EOF.
- [ ] All new tests are hermetic (httptest), race-clean (`go test -race`), and include both
      whitebox siblings and blackbox `tests/` coverage.

## Dependencies

- **Spec 01** — `Request` prepared template & `BodySource` field; `client.Prepare`/`client.Do`.
- **Spec 03** — typed error classification (`ErrBodyTooLarge`, `ErrBodyNotReplayable`,
  Retryable/Temporary) built on `GocurlError`.
- **Spec 04** — idempotency-aware `RetryPolicy`; this spec adds the non-rewindable-body
  non-replay rule on top of it.

## Open questions / decisions to confirm in review

1. **`Body string` deprecation path:** keep `options.RequestOptions.Body string` as a
   convenience that maps to `StringBody`, or deprecate it in favor of a `BodySource` field on
   `RequestOptions`? (Proposed: keep `Body` for back-compat, add `BodySource` taking precedence.)
2. **Default `ResponseBodyLimit`:** today the limit is opt-in (`> 0`). For "mission-critical"
   safety, should there be a sane default cap (e.g. unbounded but warn) and an explicit
   "stream, no limit" sentinel for SSE/downloads? (Proposed: keep opt-in; document SSE caveat.)
3. **Auto-detect non-replayable upgrade:** should a non-rewindable `io.Reader` body be
   transparently buffered up to a small threshold (e.g. 64 KB) to keep retries working for tiny
   streams, or always refuse? (Proposed: never buffer silently; require explicit `BytesBody`.)
4. **Multipart rewindability:** is it worth supporting rewindable multipart when all parts are
   files/bytes (re-open the pipe per attempt), or always mark non-rewindable? (Proposed: always
   non-rewindable in v1 for simplicity.)
5. **Progress hooks:** do upload/download progress callbacks belong here or in the observability
   spec? (Proposed: observability spec; reference only.)
6. **Request trailers (`req.Trailer`):** expose in v1 or defer? (Proposed: defer; no curl flag
   maps to it.)
