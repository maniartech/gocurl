package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// M12-T2 "clone-the-small" validation matrix (Spec 14 §B).
//
// effectiveOptions prepares the per-Do options by merging the Client's defaults
// onto the prepared Request's recipe. It must:
//   1. NEVER mutate the shared, prepared Request (safe across goroutines / Clients).
//   2. NEVER let one Client's defaults bleed into another Client reusing the Request.
//   3. NOT deep-clone the read-only recipe (Form/QueryParams/Cookies) on every Do —
//      the prepared recipe is immutable on the Do path, so only the header map that
//      actually receives Client-default merges needs to be owned.

// TestCloneSmall_NoDeepClonePerDo is the RED→GREEN driver. The pre-optimization
// effectiveOptions called req.opts.Clone(), which unconditionally allocates fresh
// Form + QueryParams maps (and clones Headers) on every Do even though they are
// read-only downstream. With no Client default headers, preparing the per-Do
// options should cost at most the single RequestOptions struct it returns.
func TestCloneSmall_NoDeepClonePerDo(t *testing.T) {
	req, err := NewRequest("POST", "http://example.test/items",
		Header("X-Recipe", "static"),
		Query("page", "1"),
		Query("page", "2"),
	)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New() // no default headers: the header template can be shared read-only
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	allocs := testing.AllocsPerRun(200, func() {
		_ = c.effectiveOptions(req)
	})
	if allocs > 2 {
		t.Errorf("effectiveOptions allocated %.0f objects/op; clone-the-small should not "+
			"deep-clone the read-only recipe (want <= 2)", allocs)
	}
}

// TestCloneSmall_DoesNotMutateSharedRequest drives effectiveOptions concurrently
// and asserts the prepared Request's own option maps are never touched — no added
// default header, no changed scalar. Run under -race.
func TestCloneSmall_DoesNotMutateSharedRequest(t *testing.T) {
	req, err := NewRequest("GET", "http://example.test/", Header("X-Recipe", "static"))
	if err != nil {
		t.Fatal(err)
	}
	beforeLen := len(req.opts.Headers)

	c, err := New(WithDefaultHeader("X-Default", "d"), WithUserAgent("ua/1"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o := c.effectiveOptions(req)
			if o.Headers.Get("X-Default") != "d" {
				t.Errorf("Client default header not applied to the per-Do options")
			}
			if o.UserAgent != "ua/1" {
				t.Errorf("Client default user-agent not applied")
			}
		}()
	}
	wg.Wait()

	if got := req.opts.Headers.Get("X-Default"); got != "" {
		t.Errorf("shared Request template was mutated: X-Default = %q", got)
	}
	if len(req.opts.Headers) != beforeLen {
		t.Errorf("shared Request header map grew from %d to %d", beforeLen, len(req.opts.Headers))
	}
	if req.opts.UserAgent != "" {
		t.Errorf("shared Request user-agent was mutated: %q", req.opts.UserAgent)
	}
}

// TestCloneSmall_NoDefaultBleedAcrossClients proves two Clients reusing ONE prepared
// Request never see each other's defaults — the per-Do options must be independent.
func TestCloneSmall_NoDefaultBleedAcrossClients(t *testing.T) {
	req, err := NewRequest("GET", "http://example.test/")
	if err != nil {
		t.Fatal(err)
	}
	c1, _ := New(WithDefaultHeader("X-Client", "one"))
	c2, _ := New(WithDefaultHeader("X-Client", "two"))
	defer c1.Close()
	defer c2.Close()

	o1 := c1.effectiveOptions(req)
	o2 := c2.effectiveOptions(req)

	if o1.Headers.Get("X-Client") != "one" {
		t.Errorf("c1 default missing: %q", o1.Headers.Get("X-Client"))
	}
	if o2.Headers.Get("X-Client") != "two" {
		t.Errorf("c2 default missing: %q", o2.Headers.Get("X-Client"))
	}
	if req.opts.Headers.Get("X-Client") != "" {
		t.Errorf("shared Request template absorbed a Client default")
	}
}

// TestCloneSmall_ConcurrentWireCorrectness drives real concurrent Do calls over one
// prepared Request with a per-call request-ID generator, and asserts every request
// reached the server with a DISTINCT X-Request-ID and the prepared recipe header —
// i.e. the owned-vs-shared split produces correct, isolated wire requests under load.
func TestCloneSmall_ConcurrentWireCorrectness(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]int{}
	recipeOK := int32(1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.Header.Get("X-Request-ID")]++
		mu.Unlock()
		if r.Header.Get("X-Recipe") != "static" {
			atomic.StoreInt32(&recipeOK, 0)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var ctr int64
	c, err := New(WithRequestIDFunc(func() string {
		return "rid-" + itoa(atomic.AddInt64(&ctr, 1))
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	req, err := c.Prepare("curl -H 'X-Recipe: static' " + srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	const N = 50
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.Do(context.Background(), req)
			if err != nil {
				t.Errorf("Do: %v", err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&recipeOK) != 1 {
		t.Error("a request reached the server without the prepared X-Recipe header")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != N {
		t.Errorf("expected %d distinct request IDs, got %d", N, len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("request ID %q used %d times (want 1) — IDs collided across Do calls", id, n)
		}
	}
}

// itoa is a tiny dependency-free int->string for the test above.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
