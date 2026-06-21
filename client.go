package gocurl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/maniartech/gocurl/options"
)

// httpClient is an interface for making HTTP requests.
// This allows for mocking and custom client implementations. It remains the
// low-level injection seam (see WithTransport); Client is the high-level,
// configure-once/reuse type built on top of it.
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Ensure *http.Client implements httpClient
var _ httpClient = (*http.Client)(nil)

// Client is a reusable, concurrency-safe HTTP client. Configure it once with
// functional Options, then execute prepared Requests many times — connections
// are pooled and reused across calls. A Client owns its transport, so closing
// one Client never affects another.
//
// See specs/01-client-api.md.
type Client struct {
	cfg        *config
	httpClient *http.Client
	mw         []Middleware
	obs        *obs

	mu       sync.Mutex
	inflight sync.WaitGroup
	closed   bool
}

// New constructs a Client from functional Options.
func New(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// When the SSRF guard is enabled, re-check every redirect hop through the
	// request-context redirect seam (composing with any user-supplied Allow hook),
	// so a public URL that redirects to an internal address is blocked.
	if cfg.ssrfPolicy != nil {
		pol := *cfg.ssrfPolicy
		prev := cfg.redirectAllow
		cfg.redirectAllow = func(req *http.Request, via []*http.Request) error {
			if req.URL != nil {
				if err := pol.CheckSSRF(req.Context(), req.URL.Host); err != nil {
					return err
				}
			}
			if prev != nil {
				return prev(req, via)
			}
			return nil
		}
	}

	transport, err := cfg.buildTransport()
	if err != nil {
		return nil, err
	}

	hc := &http.Client{
		Transport:     transport,
		Timeout:       cfg.timeout,
		CheckRedirect: redirectFromContext,
	}
	if cfg.cookieFile != "" {
		jar, err := NewPersistentCookieJar(cfg.cookieFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create cookie jar: %w", err)
		}
		hc.Jar = jar
	}

	return &Client{cfg: cfg, httpClient: hc, mw: cfg.middleware, obs: resolveObs(cfg)}, nil
}

// redirectSettings carries per-request redirect policy through the request
// context so a SHARED http.Client can honor per-request -L/--max-redirs and an
// optional per-redirect authorization hook.
type redirectSettings struct {
	follow bool
	max    int
	allow  func(req *http.Request, via []*http.Request) error
}

type redirectCtxKey struct{}

func withRedirectSettings(ctx context.Context, rs redirectSettings) context.Context {
	return context.WithValue(ctx, redirectCtxKey{}, rs)
}

func redirectFromContext(req *http.Request, via []*http.Request) error {
	rs, _ := req.Context().Value(redirectCtxKey{}).(redirectSettings)
	if !rs.follow {
		return http.ErrUseLastResponse
	}
	if rs.max > 0 && len(via) >= rs.max {
		return fmt.Errorf("stopped after %d redirects", rs.max)
	}
	if rs.allow != nil {
		return rs.allow(req, via)
	}
	return nil
}

// effectiveOptions merges the Client's request-level defaults onto a copy of the
// prepared request's options. Connection-level settings (TLS/proxy) come from
// the Client's transport and are not taken from the request here.
func (c *Client) effectiveOptions(req *Request) *options.RequestOptions {
	o := req.opts.Clone()
	if o.UserAgent == "" && c.cfg.userAgent != "" {
		o.UserAgent = c.cfg.userAgent
	}
	if !o.FollowRedirects && c.cfg.followRedirects {
		o.FollowRedirects = true
		o.MaxRedirects = c.cfg.maxRedirects
	}
	if c.cfg.failOnStatus {
		o.FailOnError = true
	}
	if c.cfg.allowInsecureAuth {
		o.AllowInsecureAuth = true
	}
	for key, values := range c.cfg.defaultHeaders {
		if o.Headers == nil {
			o.Headers = http.Header{}
		}
		if o.Headers.Get(key) == "" {
			for _, v := range values {
				o.Headers.Add(key, v)
			}
		}
	}
	return o
}

// resolveRetryPolicy picks the effective retry policy for this request: a
// per-request override (Request.WithRetryPolicy) beats the Client default
// (WithRetry/WithRetryAttempts), which beats the legacy options.RetryConfig
// (method-agnostic). It returns a COPY with the body-replay ceiling and the
// Client's retry budget applied, so a shared policy value is never mutated.
func (c *Client) resolveRetryPolicy(req *Request, opts *options.RequestOptions) *RetryPolicy {
	var p *RetryPolicy
	switch {
	case req.retryPolicy != nil:
		p = req.retryPolicy
	case c.cfg.retryPolicy != nil:
		p = c.cfg.retryPolicy
	default:
		// Legacy fallback: already a fresh, owned policy (or nil) — apply the cap
		// and the Client retry budget, then return directly.
		lp := legacyPolicyFromRetryConfig(opts.RetryConfig)
		if lp == nil {
			return nil
		}
		lp.maxReplayBytes = c.cfg.maxReplayBytes
		if lp.Budget == nil && c.cfg.retryBudget != nil {
			lp.Budget = c.cfg.retryBudget
		}
		return lp
	}
	eff := p.withReplayCap(c.cfg.maxReplayBytes)
	if eff.Budget == nil && c.cfg.retryBudget != nil {
		eff.Budget = c.cfg.retryBudget
	}
	return eff
}

// Do executes a prepared Request and returns the live response. The caller owns
// resp.Body and must Close it. The body is streamed (never buffered or written
// to stdout by the Client).
func (c *Client) Do(ctx context.Context, req *Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.inflight.Add(1)
	c.mu.Unlock()
	defer c.inflight.Done()

	if ctx == nil {
		ctx = context.Background()
	}

	opts := c.effectiveOptions(req)
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	// Establish the OVERALL deadline that bounds the entire operation INCLUDING
	// retries and their backoff sleeps — curl's --max-time semantics. A per-request
	// --max-time (opts.Timeout) wins; otherwise the Client-wide WithTimeout
	// (cfg.timeout) applies. Without this, WithTimeout only reaches http.Client.Timeout
	// (a per-attempt bound), so a retryable storm runs MaxAttempts*backoff past the
	// deadline — a retry amplifier. http.Client.Timeout remains the per-attempt net.
	overall := opts.Timeout
	if overall <= 0 {
		overall = c.cfg.timeout
	}
	var cancel context.CancelFunc
	if overall > 0 {
		ctx, cancel = context.WithTimeout(ctx, overall)
	}

	ctx = withRedirectSettings(ctx, redirectSettings{
		follow: opts.FollowRedirects,
		max:    opts.MaxRedirects,
		allow:  c.cfg.redirectAllow,
	})

	httpReq, err := createRequest(ctx, opts)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, err
	}

	// Legacy request-mutating middleware on the prepared request still runs.
	httpReq, err = applyMiddleware(httpReq, opts.Middleware)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, err
	}

	// Resolve the effective retry policy (per-request override > Client default >
	// legacy RetryConfig), then build the base handler: the retry engine over the
	// Client's owned http.Client, with a per-request *rand.Rand for jitter.
	policy := c.resolveRetryPolicy(req, opts)
	base := Handler(func(hr *http.Request) (*http.Response, error) {
		return executeWithRetries(c.httpClient, hr, opts, policy, newRand())
	})
	// Compose, outermost-first: instrumentation -> circuit breaker -> rate limiter
	// -> user middleware -> retry loop. Instrumentation is outermost so its span
	// and latency bracket the breaker fast-fail and limiter wait too; it is only
	// present when a sink/hook is configured (zero overhead otherwise).
	var allMW []Middleware
	if c.obs.active {
		allMW = append(allMW, c.obs.instrument)
	}
	allMW = append(allMW, c.cfg.systemMiddleware()...)
	allMW = append(allMW, c.mw...)
	resp, err := chain(base, allMW...)(httpReq)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, wrapTransportError(opts.URL, err)
	}

	printResponseVerbose(opts, resp)

	if opts.Compress {
		if derr := decompressResponse(resp); derr != nil {
			resp.Body.Close()
			if cancel != nil {
				cancel()
			}
			return nil, fmt.Errorf("failed to decompress response: %w", derr)
		}
	}

	body := resp.Body
	if opts.ResponseBodyLimit > 0 {
		body = newLimitedBody(body, opts.ResponseBodyLimit)
	}
	// Ensure a per-request context (--max-time) is cancelled once the body is
	// fully consumed/closed, without truncating the stream early.
	resp.Body = &cancelOnCloseBody{ReadCloser: body, cancel: cancel}

	// Fail-on-status (opt-in): a >=400 response becomes a typed error, but the
	// live response is still returned so the caller can read the error body.
	if ferr := failOnStatus(resp, opts); ferr != nil {
		return resp, ferr
	}
	return resp, nil
}

// cancelOnCloseBody calls cancel when the body is closed, releasing a per-request
// timeout context without cutting the stream short.
type cancelOnCloseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *cancelOnCloseBody) Close() error {
	err := b.ReadCloser.Close()
	if b.cancel != nil {
		b.cancel()
	}
	return err
}

// Close releases idle connections held by the Client's transport. It does NOT
// abort in-flight requests; use Shutdown to drain them first.
func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	if t, ok := c.httpClient.Transport.(interface{ CloseIdleConnections() }); ok {
		t.CloseIdleConnections()
	} else {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

// Shutdown marks the Client closed, waits for in-flight requests to finish (or
// the context to expire), then releases idle connections.
func (c *Client) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()

	done := make(chan struct{})
	go func() {
		c.inflight.Wait()
		close(done)
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		c.httpClient.CloseIdleConnections()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// --- Convenience methods (mirror the package-level Curl* helpers) ---

// Curl prepares (parsing env vars) and executes a curl command in one call.
func (c *Client) Curl(ctx context.Context, command string) (*http.Response, error) {
	req, err := c.Prepare(command)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// CurlString executes a curl command and returns the body as a string.
func (c *Client) CurlString(ctx context.Context, command string) (string, *http.Response, error) {
	resp, err := c.Curl(ctx, command)
	if err != nil {
		return "", resp, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp, fmt.Errorf("failed to read response body: %w", err)
	}
	return string(b), resp, nil
}

// CurlBytes executes a curl command and returns the body as bytes.
func (c *Client) CurlBytes(ctx context.Context, command string) ([]byte, *http.Response, error) {
	resp, err := c.Curl(ctx, command)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}
	return b, resp, nil
}

// CurlJSON executes a curl command and decodes the JSON response into v.
func (c *Client) CurlJSON(ctx context.Context, v interface{}, command string) (*http.Response, error) {
	resp, err := c.Curl(ctx, command)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return resp, nil
}

// CurlDownload executes a curl command and streams the body to filePath.
func (c *Client) CurlDownload(ctx context.Context, filePath, command string) (int64, *http.Response, error) {
	resp, err := c.Curl(ctx, command)
	if err != nil {
		return 0, resp, err
	}
	defer resp.Body.Close()
	f, err := os.Create(filePath)
	if err != nil {
		return 0, resp, fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return n, resp, fmt.Errorf("failed to write to file: %w", err)
	}
	return n, resp, nil
}
