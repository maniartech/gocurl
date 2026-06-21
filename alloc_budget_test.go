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

	// Ratcheted from 100 to baseline (77, measured after the clone-the-small
	// optimization) + small headroom. The old 24-alloc slack would have let a
	// regression hide; raising this needs a one-line justification in the commit.
	const budget = 88
	if avg > budget {
		t.Fatalf("Do allocs/op = %.0f, budget %d (update intentionally if justified)", avg, budget)
	}
}
