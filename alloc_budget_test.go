package gocurl_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maniartech/gocurl"
)

// These tests turn allocation regressions into hard failures using
// testing.AllocsPerRun. The budgets are CEILINGS chosen from the measured
// baseline plus headroom — NOT "zero-allocation" claims. Lowering a budget is a
// deliberate, reviewed change; raising one needs a one-line justification.

func TestAllocBudget_ExpandVariables(t *testing.T) {
	vars := gocurl.Variables{"token": "my-secret-token", "url": "https://example.com", "data": "important data"}
	text := "Authorization: Bearer ${token}, URL: ${url}, Data: ${data}"

	avg := testing.AllocsPerRun(1000, func() {
		_, _ = gocurl.ExpandVariables(text, vars)
	})
	t.Logf("ExpandVariables allocs/op = %.0f", avg)

	const budget = 6
	if avg > budget {
		t.Fatalf("ExpandVariables allocs/op = %.0f, budget %d (update intentionally if justified)", avg, budget)
	}
}

func TestAllocBudget_Prepare(t *testing.T) {
	c, err := gocurl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	avg := testing.AllocsPerRun(200, func() {
		_, _ = c.Prepare(`curl -X POST -H "Content-Type: application/json" -d {"k":"v"} https://example.com`)
	})
	t.Logf("Prepare allocs/op = %.0f", avg)

	const budget = 45
	if avg > budget {
		t.Fatalf("Prepare allocs/op = %.0f, budget %d (update intentionally if justified)", avg, budget)
	}
}

func TestAllocBudget_Do(t *testing.T) {
	if testing.Short() {
		t.Skip("alloc budget over real I/O skipped in -short")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := gocurl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl " + srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	do := func() {
		resp, err := c.Do(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	do() // warm up the cached transport

	avg := testing.AllocsPerRun(200, do)
	t.Logf("Do (round-trip over httptest) allocs/op = %.0f", avg)

	// Ratcheted from 100 to baseline (75, measured after clone-the-small + the
	// lazy-rand fix) + small headroom. The old 24-alloc slack would have let a
	// regression hide; raising this needs a one-line justification in the commit.
	const budget = 85
	if avg > budget {
		t.Fatalf("Do allocs/op = %.0f, budget %d (update intentionally if justified)", avg, budget)
	}
}

// TestByteBudget_Do guards the BYTES-per-op of the prepared Do hot path. The
// allocs/op count barely moved when we made the jitter RNG lazy, but the bytes did:
// a per-Do newRand() allocated a ~4.9 KiB [607]int64 rngSource even with no retries.
// testing.AllocsPerRun only counts allocations, not bytes, so this uses
// testing.Benchmark to assert AllocedBytesPerOp stays well under the pre-fix ~13 KiB —
// catching any regression that reintroduces a large per-request allocation.
func TestByteBudget_Do(t *testing.T) {
	if testing.Short() {
		t.Skip("byte budget over real I/O skipped in -short")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := gocurl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl " + srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	res := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := c.Do(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
	bpo := res.AllocedBytesPerOp()
	t.Logf("Do (prepared round-trip) bytes/op = %d", bpo)

	// Baseline ~7.4 KiB after the lazy-rand fix; the pre-fix path was ~13 KiB. A
	// budget of 10 KiB leaves headroom for net/http variation while failing hard if a
	// large per-request allocation (e.g. an eager RNG) sneaks back onto the hot path.
	const budget = 10 * 1024
	if bpo > budget {
		t.Fatalf("Do bytes/op = %d, budget %d (a large per-request allocation regressed onto the hot path)", bpo, budget)
	}
}
