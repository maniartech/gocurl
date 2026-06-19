package gocurl

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_ImmediateWhenTokensAvailable(t *testing.T) {
	tb := NewTokenBucket(10, 5)
	for i := 0; i < 5; i++ {
		if err := tb.Wait(context.Background()); err != nil {
			t.Fatalf("burst token %d should be immediate, got %v", i, err)
		}
	}
}

func TestTokenBucket_BlocksThenProceeds(t *testing.T) {
	tb := NewTokenBucket(100, 1) // 1 token, refills 100/sec => ~10ms per token
	if err := tb.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := tb.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 5*time.Millisecond {
		t.Errorf("second Wait should have blocked ~10ms, took %v", elapsed)
	}
}

func TestTokenBucket_HonorsCancelledContext(t *testing.T) {
	tb := NewTokenBucket(1, 1)
	tb.Wait(context.Background()) // drain the single token

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tb.Wait(ctx); err == nil {
		t.Fatal("Wait should return the context error when cancelled")
	}
}

func TestTokenBucket_CancelDuringWait(t *testing.T) {
	tb := NewTokenBucket(0.5, 1) // ~2s per token after the first
	tb.Wait(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := tb.Wait(ctx); err == nil {
		t.Fatal("Wait should be cancelled by the short context")
	}
	if time.Since(start) > time.Second {
		t.Error("Wait should have returned promptly on cancellation, not waited for the token")
	}
}

func TestTokenBucket_RefillCappedAtBurst(t *testing.T) {
	tb := NewTokenBucket(1000, 3)
	now := time.Unix(0, 0)
	tb.nowFn = func() time.Time { return now }
	// Drain.
	tb.mu.Lock()
	tb.tokens = 0
	tb.last = now
	tb.mu.Unlock()
	// Advance a long time; refill must cap at burst (3), not accumulate unbounded.
	now = now.Add(time.Hour)
	tb.mu.Lock()
	tb.refillLocked(now)
	got := tb.tokens
	tb.mu.Unlock()
	if got != 3 {
		t.Errorf("tokens after long idle = %v, want capped at burst 3", got)
	}
}

func TestTokenBucket_ConcurrentRaceClean(t *testing.T) {
	tb := NewTokenBucket(10000, 100)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = tb.Wait(context.Background())
			}
		}()
	}
	wg.Wait()
}

func TestRateLimiter_Middleware(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://x", nil)

	// Pass-through when a token is available.
	called := 0
	h := RateLimiter(NewTokenBucket(1000, 10))(func(*http.Request) (*http.Response, error) {
		called++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if resp, err := h(req); err != nil || resp.StatusCode != 200 || called != 1 {
		t.Errorf("passthrough failed: called=%d err=%v", called, err)
	}

	// A nil limiter is a pass-through.
	h2 := RateLimiter(nil)(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if resp, err := h2(req); err != nil || resp.StatusCode != 204 {
		t.Error("nil limiter should pass through")
	}

	// When Wait fails (cancelled ctx), next is not called.
	tb := NewTokenBucket(1, 1)
	tb.Wait(context.Background()) // drain
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called2 := 0
	h3 := RateLimiter(tb)(func(*http.Request) (*http.Response, error) {
		called2++
		return nil, nil
	})
	if _, err := h3(req.WithContext(ctx)); err == nil {
		t.Error("limiter should block on a cancelled context")
	}
	if called2 != 0 {
		t.Error("next must not run when the limiter Wait fails")
	}
}
