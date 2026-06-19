package gocurl

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

// scriptedClient is a mock options.HTTPClient returning scripted outcomes per
// call and capturing the request body each attempt sent.
type scriptedClient struct {
	mu         sync.Mutex
	calls      int
	script     []scriptedResp
	bodiesSeen []string
}

type scriptedResp struct {
	status     int
	err        error
	retryAfter string
}

func (c *scriptedClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	i := c.calls
	c.calls++
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	c.bodiesSeen = append(c.bodiesSeen, body)
	sr := c.script[len(c.script)-1]
	if i < len(c.script) {
		sr = c.script[i]
	}
	c.mu.Unlock()

	if sr.err != nil {
		return nil, sr.err
	}
	h := http.Header{}
	if sr.retryAfter != "" {
		h.Set("Retry-After", sr.retryAfter)
	}
	return &http.Response{StatusCode: sr.status, Header: h, Body: io.NopCloser(strings.NewReader("body"))}, nil
}

func (c *scriptedClient) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func seededRand() *rand.Rand { return rand.New(rand.NewSource(42)) }

func optsFor(url string) *options.RequestOptions {
	o := options.NewRequestOptions(url)
	o.Headers = http.Header{}
	return o
}

func newReq(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// noSleep returns a copy of p with a zero-delay backoff so tests don't wait.
func noSleep(p RetryPolicy) *RetryPolicy {
	p.Backoff = ConstantBackoff(0)
	return &p
}

func TestExponentialJitter_DeterministicAndCapped(t *testing.T) {
	bo := ExponentialJitter(100*time.Millisecond, 5*time.Second)
	rnd := seededRand()
	// Equal jitter: each delay in [d/2, d] where d = 100ms*2^(n-1), capped at 5s.
	for n, dnom := range map[int]time.Duration{
		1: 100 * time.Millisecond,
		2: 200 * time.Millisecond,
		3: 400 * time.Millisecond,
	} {
		d := bo.Delay(n, rnd)
		if d < dnom/2 || d > dnom {
			t.Errorf("Delay(%d) = %v, want in [%v, %v]", n, d, dnom/2, dnom)
		}
	}
	// Large attempt is capped at max (5s); equal jitter keeps it in [2.5s, 5s].
	big := bo.Delay(20, rnd)
	if big < 2500*time.Millisecond || big > 5*time.Second {
		t.Errorf("Delay(20) = %v, want in [2.5s, 5s]", big)
	}
}

func TestConstantBackoff(t *testing.T) {
	bo := ConstantBackoff(250 * time.Millisecond)
	if d := bo.Delay(1, seededRand()); d != 250*time.Millisecond {
		t.Errorf("Delay(1) = %v, want 250ms", d)
	}
	if d := bo.Delay(5, seededRand()); d != 250*time.Millisecond {
		t.Errorf("Delay(5) = %v, want 250ms", d)
	}
}

func TestLegacyExponentialBackoff_DeterministicNoJitter(t *testing.T) {
	// The legacy default backoff must reproduce the historical 100ms*2^(n-1)
	// schedule with NO jitter (so retry_test.go timing stays stable).
	p := legacyPolicyFromRetryConfig(&options.RetryConfig{MaxRetries: 3})
	if p == nil {
		t.Fatal("expected a policy")
	}
	bo := p.backoff()
	if d := bo.Delay(1, seededRand()); d != 100*time.Millisecond {
		t.Errorf("legacy Delay(1) = %v, want 100ms", d)
	}
	if d := bo.Delay(2, seededRand()); d != 200*time.Millisecond {
		t.Errorf("legacy Delay(2) = %v, want 200ms", d)
	}
}

func TestParseRetryAfter(t *testing.T) {
	mk := func(v string) *http.Response {
		h := http.Header{}
		if v != "" {
			h.Set("Retry-After", v)
		}
		return &http.Response{Header: h}
	}
	if d := parseRetryAfter(mk("5")); d != 5*time.Second {
		t.Errorf("delta-seconds: got %v", d)
	}
	if d := parseRetryAfter(mk("")); d != 0 {
		t.Errorf("empty: got %v", d)
	}
	if d := parseRetryAfter(mk("garbage")); d != 0 {
		t.Errorf("garbage: got %v want 0", d)
	}
	if d := parseRetryAfter(mk("0")); d != 0 {
		t.Errorf("zero: got %v", d)
	}
	// HTTP-date in the future yields a positive delay.
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	if d := parseRetryAfter(mk(future)); d <= 0 || d > 31*time.Second {
		t.Errorf("http-date: got %v, want ~30s", d)
	}
	// A past date yields 0 (no wait).
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if d := parseRetryAfter(mk(past)); d != 0 {
		t.Errorf("past date: got %v want 0", d)
	}
	if parseRetryAfter(nil) != 0 {
		t.Error("nil response should yield 0")
	}
}

func TestNextDelay_RetryAfter(t *testing.T) {
	p := DefaultRetryPolicy(3)
	p.Backoff = ConstantBackoff(10 * time.Millisecond)
	resp := &http.Response{StatusCode: 503, Header: http.Header{"Retry-After": []string{"2"}}}

	// Effective delay = max(backoff, Retry-After).
	if d := nextDelay(&p, 1, resp, seededRand()); d != 2*time.Second {
		t.Errorf("with Retry-After: got %v, want 2s", d)
	}
	// Disabled: Retry-After ignored.
	p.RespectRetryAfter = false
	if d := nextDelay(&p, 1, resp, seededRand()); d != 10*time.Millisecond {
		t.Errorf("RespectRetryAfter=false: got %v, want 10ms", d)
	}
	// Re-enabled, but a non-429/503 status does not consult Retry-After.
	p.RespectRetryAfter = true
	resp200 := &http.Response{StatusCode: 500, Header: http.Header{"Retry-After": []string{"2"}}}
	if d := nextDelay(&p, 1, resp200, seededRand()); d != 10*time.Millisecond {
		t.Errorf("Retry-After only on 429/503: got %v, want 10ms", d)
	}
}

func TestRetryPolicy_MethodEligibility(t *testing.T) {
	p := DefaultRetryPolicy(3)
	for _, m := range []string{"GET", "HEAD", "OPTIONS", "TRACE", "PUT", "DELETE"} {
		if !p.methodEligible(m) {
			t.Errorf("%s should be eligible by default", m)
		}
	}
	for _, m := range []string{"POST", "PATCH", "CONNECT"} {
		if p.methodEligible(m) {
			t.Errorf("%s should NOT be eligible by default", m)
		}
	}
	// AllowMethods overrides the default set.
	p2 := DefaultRetryPolicy(3)
	p2.AllowMethods = []string{"POST"}
	if !p2.methodEligible("post") || p2.methodEligible("GET") {
		t.Error("AllowMethods=[POST] should make POST eligible and GET not")
	}
}

func TestRetryPolicy_IdempotencyKeyEscapeHatch(t *testing.T) {
	p := DefaultRetryPolicy(3)
	req := newReq(t, "POST", "http://x", strings.NewReader("d"))
	if p.eligibleForRetry(req) {
		t.Error("POST without Idempotency-Key should not be eligible")
	}
	req.Header.Set("Idempotency-Key", "  abc123  ")
	if !p.eligibleForRetry(req) {
		t.Error("POST with a (padded, mixed-case) Idempotency-Key should be eligible")
	}
	// Legacy policy is always eligible (method-agnostic).
	lp := legacyPolicyFromRetryConfig(&options.RetryConfig{MaxRetries: 2})
	if !lp.eligibleForRetry(newReq(t, "POST", "http://x", strings.NewReader("d"))) {
		t.Error("legacy policy should be method-agnostic")
	}
}

func TestLegacyPolicyMapping(t *testing.T) {
	if legacyPolicyFromRetryConfig(nil) != nil {
		t.Error("nil RetryConfig => nil policy")
	}
	if legacyPolicyFromRetryConfig(&options.RetryConfig{MaxRetries: 0}) != nil {
		t.Error("MaxRetries=0 => nil policy (single attempt)")
	}
	p := legacyPolicyFromRetryConfig(&options.RetryConfig{MaxRetries: 3, RetryDelay: 50 * time.Millisecond, RetryOnHTTP: []int{503}})
	if p.MaxAttempts != 4 || !p.fromLegacy || !p.RespectRetryAfter {
		t.Errorf("mapping wrong: %+v", p)
	}
	if d := p.backoff().Delay(1, seededRand()); d != 50*time.Millisecond {
		t.Errorf("RetryDelay should map to ConstantBackoff: got %v", d)
	}
}

func TestEngine_GET503Retried(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 503}, {status: 200}}}
	req := newReq(t, "GET", "http://x", nil)
	resp, err := executeWithRetries(c, req, optsFor("http://x"), noSleep(DefaultRetryPolicy(3)), seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || c.count() != 3 {
		t.Errorf("status=%d calls=%d, want 200 and 3", resp.StatusCode, c.count())
	}
}

func TestEngine_POST503NotRetried(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 200}}}
	req := newReq(t, "POST", "http://x", strings.NewReader("d"))
	resp, err := executeWithRetries(c, req, optsFor("http://x"), noSleep(DefaultRetryPolicy(3)), seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 503 || c.count() != 1 {
		t.Errorf("status=%d calls=%d, want 503 and 1 (POST not retried)", resp.StatusCode, c.count())
	}
}

func TestEngine_POSTWithIdempotencyKeyRetried(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 200}}}
	req := newReq(t, "POST", "http://x", strings.NewReader("payload"))
	req.Header.Set("Idempotency-Key", "k-1")
	resp, err := executeWithRetries(c, req, optsFor("http://x"), noSleep(DefaultRetryPolicy(3)), seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || c.count() != 2 {
		t.Errorf("status=%d calls=%d, want 200 and 2", resp.StatusCode, c.count())
	}
	// Body replayed identically (via GetBody from the strings.Reader).
	for i, b := range c.bodiesSeen {
		if b != "payload" {
			t.Errorf("attempt %d body = %q, want payload", i, b)
		}
	}
}

func TestEngine_LegacyPOSTRetried(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 502}, {status: 200}}}
	req := newReq(t, "POST", "http://x", strings.NewReader("d"))
	policy := legacyPolicyFromRetryConfig(&options.RetryConfig{MaxRetries: 3, RetryDelay: time.Millisecond, RetryOnHTTP: []int{502}})
	resp, err := executeWithRetries(c, req, optsFor("http://x"), policy, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || c.count() != 2 {
		t.Errorf("status=%d calls=%d, want 200 and 2 (legacy method-agnostic)", resp.StatusCode, c.count())
	}
}

func TestEngine_BufferedBodyReplay(t *testing.T) {
	// A body with no GetBody (opaque ReadCloser) is buffered and replayed.
	c := &scriptedClient{script: []scriptedResp{{status: 500}, {status: 200}}}
	req := newReq(t, "POST", "http://x", io.NopCloser(strings.NewReader("BUFFERED")))
	if req.GetBody != nil {
		t.Fatal("precondition: opaque body should have no GetBody")
	}
	p := noSleep(DefaultRetryPolicy(3))
	p.AllowMethods = []string{"POST"}
	resp, err := executeWithRetries(c, req, optsFor("http://x"), p, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || c.count() != 2 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, c.count())
	}
	for i, b := range c.bodiesSeen {
		if b != "BUFFERED" {
			t.Errorf("attempt %d body = %q, want BUFFERED", i, b)
		}
	}
}

func TestEngine_NonReplayableBodyShortCircuits(t *testing.T) {
	// Body exceeds the replay cap and has no GetBody: attempt 1 sends the full
	// body, but it is not retried.
	c := &scriptedClient{script: []scriptedResp{{status: 500}, {status: 200}}}
	req := newReq(t, "POST", "http://x", io.NopCloser(strings.NewReader("0123456789")))
	p := noSleep(DefaultRetryPolicy(3))
	p.AllowMethods = []string{"POST"}
	p.maxReplayBytes = 4 // body (10) > cap
	resp, err := executeWithRetries(c, req, optsFor("http://x"), p, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if c.count() != 1 {
		t.Errorf("calls=%d, want 1 (non-replayable body not retried)", c.count())
	}
	if resp.StatusCode != 500 {
		t.Errorf("status=%d, want 500", resp.StatusCode)
	}
	if c.bodiesSeen[0] != "0123456789" {
		t.Errorf("attempt 1 must send the full body, got %q", c.bodiesSeen[0])
	}
}

func TestEngine_MaxElapsedStops(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 503}, {status: 200}}}
	req := newReq(t, "GET", "http://x", nil)
	p := DefaultRetryPolicy(5)
	p.Backoff = ConstantBackoff(100 * time.Millisecond)
	p.MaxElapsed = time.Millisecond // next delay (100ms) exceeds the budget
	resp, err := executeWithRetries(c, req, optsFor("http://x"), &p, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if c.count() != 1 || resp.StatusCode != 503 {
		t.Errorf("calls=%d status=%d, want 1 and 503 (MaxElapsed stops before sleeping)", c.count(), resp.StatusCode)
	}
}

func TestEngine_ExhaustionWrapsRetryError(t *testing.T) {
	connErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	c := &scriptedClient{script: []scriptedResp{{err: connErr}}}
	req := newReq(t, "GET", "http://x", nil)
	_, err := executeWithRetries(c, req, optsFor("http://x"), noSleep(DefaultRetryPolicy(3)), seededRand())
	if err == nil {
		t.Fatal("expected an error")
	}
	if KindOf(err) != KindRetryExhausted {
		t.Errorf("KindOf = %v, want KindRetryExhausted", KindOf(err))
	}
	if c.count() != 3 {
		t.Errorf("calls=%d, want 3", c.count())
	}
}

func TestEngine_PerAttemptTimeoutIsRetryable(t *testing.T) {
	// A per-attempt deadline (parent context alive) is a retryable timeout.
	calls := 0
	c := blockingClientFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls < 2 {
			<-req.Context().Done() // exceed PerAttempt
			return nil, req.Context().Err()
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	req := newReq(t, "GET", "http://x", nil)
	p := DefaultRetryPolicy(3)
	p.Backoff = ConstantBackoff(0)
	p.PerAttempt = 20 * time.Millisecond
	resp, err := executeWithRetries(c, req, optsFor("http://x"), &p, seededRand())
	if err != nil {
		t.Fatalf("per-attempt timeout should be retried, got err: %v", err)
	}
	if resp.StatusCode != 200 || calls != 2 {
		t.Errorf("status=%d calls=%d, want 200 and 2", resp.StatusCode, calls)
	}
}

func TestEngine_ParentCancelIsTerminal(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 200}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://x", nil)
	_, err := executeWithRetries(c, req, optsFor("http://x"), noSleep(DefaultRetryPolicy(3)), seededRand())
	if err == nil {
		t.Fatal("expected a context error")
	}
	if c.count() != 0 {
		t.Errorf("calls=%d, want 0 (cancelled before execution)", c.count())
	}
}

func TestEngine_BudgetSuppressesRetry(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 503}, {status: 200}}}
	req := newReq(t, "GET", "http://x", nil)
	p := noSleep(DefaultRetryPolicy(5))
	p.Budget = NewRetryBudget(0, 0) // empty bucket (max ~1, but ratio 0 => starts at 1)
	// Drain the single starting token so the first retry is suppressed.
	p.Budget.Consume()
	resp, err := executeWithRetries(c, req, optsFor("http://x"), p, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if c.count() != 1 || resp.StatusCode != 503 {
		t.Errorf("calls=%d status=%d, want 1 and 503 (budget suppressed retry)", c.count(), resp.StatusCode)
	}
}

func TestEngine_CustomRetryableGovernsPOST(t *testing.T) {
	// A custom Retryable override fully governs the decision: it opts a POST into
	// retries even though POST is not in the default idempotent set.
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 200}}}
	req := newReq(t, "POST", "http://x", strings.NewReader("payload"))
	p := noSleep(DefaultRetryPolicy(3))
	p.Retryable = func(a *Attempt) bool { return a.Response != nil && a.Response.StatusCode == 503 }
	resp, err := executeWithRetries(c, req, optsFor("http://x"), p, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || c.count() != 2 {
		t.Errorf("custom Retryable should govern a POST: status=%d calls=%d, want 200 and 2", resp.StatusCode, c.count())
	}
}

func TestEngine_SingleAttemptWhenNoPolicy(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}}}
	req := newReq(t, "GET", "http://x", nil)
	resp, err := executeWithRetries(c, req, optsFor("http://x"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.count() != 1 || resp.StatusCode != 503 {
		t.Errorf("calls=%d, want 1 (nil policy => single attempt)", c.count())
	}
}

func TestEngine_GetBodyRewindErrorIsTerminal(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 503}, {status: 200}}}
	req := newReq(t, "GET", "http://x", nil)
	req.Body = io.NopCloser(strings.NewReader("x"))
	req.GetBody = func() (io.ReadCloser, error) { return nil, errors.New("rewind failed") }

	_, err := executeWithRetries(c, req, optsFor("http://x"), noSleep(DefaultRetryPolicy(3)), seededRand())
	if err == nil || !strings.Contains(err.Error(), "rewind") {
		t.Fatalf("expected a rewind error, got %v", err)
	}
	if c.count() != 1 {
		t.Errorf("calls=%d, want 1 (rewind failure before attempt 2's Do)", c.count())
	}
}

func TestEngine_UnlimitedReplayBuffers(t *testing.T) {
	c := &scriptedClient{script: []scriptedResp{{status: 500}, {status: 200}}}
	req := newReq(t, "POST", "http://x", io.NopCloser(strings.NewReader("UNLIMITED")))
	p := noSleep(DefaultRetryPolicy(3))
	p.AllowMethods = []string{"POST"}
	p.maxReplayBytes = 0 // unlimited => buffer fully, replayable
	resp, err := executeWithRetries(c, req, optsFor("http://x"), p, seededRand())
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || c.count() != 2 {
		t.Fatalf("status=%d calls=%d", resp.StatusCode, c.count())
	}
	for i, b := range c.bodiesSeen {
		if b != "UNLIMITED" {
			t.Errorf("attempt %d body = %q", i, b)
		}
	}
}

func TestReadMultiCloser_Close(t *testing.T) {
	closed := false
	r := &readMultiCloser{Reader: strings.NewReader("abc"), closer: closerFunc(func() error { closed = true; return nil })}
	io.ReadAll(r)
	r.Close()
	if !closed {
		t.Error("Close should delegate to the underlying closer")
	}
}

func TestRetryBudget_RefillsOverTime(t *testing.T) {
	b := NewRetryBudget(0, 10) // 10 tokens/sec floor
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	b.mu.Lock()
	b.tokens = 0
	b.lastRefill = now
	b.mu.Unlock()

	if b.Consume() {
		t.Fatal("empty bucket should not permit a retry")
	}
	now = now.Add(time.Second) // +10 tokens
	if !b.Consume() {
		t.Error("budget should refill over time at MinPerSec")
	}
}

func TestRetryBudget_ConsumeRefund(t *testing.T) {
	b := NewRetryBudget(1, 0) // ratio 1, max bucket ~1
	if !b.Consume() {
		t.Fatal("first consume should succeed (bucket starts full)")
	}
	if b.Consume() {
		t.Error("second consume should fail (bucket empty)")
	}
	b.Refund() // +1 token
	if !b.Consume() {
		t.Error("consume after refund should succeed")
	}
	// nil budget always permits and never panics.
	var nilB *RetryBudget
	if !nilB.Consume() {
		t.Error("nil budget Consume should permit")
	}
	nilB.Refund()
}

func TestRetryBudget_ConcurrentRaceClean(t *testing.T) {
	b := NewRetryBudget(1, 100)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if b.Consume() {
					b.Refund()
				}
			}
		}()
	}
	wg.Wait()
}

// --- test helpers / minimal fakes ---

type blockingClientFunc func(*http.Request) (*http.Response, error)

func (f blockingClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
