package gocurl

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

// TestNetHTTP_DoesNotMutateRequestHeader validates the core assumption behind the
// compiled-execution-plan optimization (Spec 14 §B): if net/http's client/transport
// does not mutate req.Header during a round trip, then a precompiled static header
// map can be SHARED read-only across many per-Do *http.Request instances (with
// copy-on-write only when a dynamic header must be added) — eliminating the
// Header.Clone-per-Do cost. If this test ever fails, that sharing is unsafe and the
// design must fall back to per-Do cloning.
func TestNetHTTP_DoesNotMutateRequestHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Static-A", "1")
	req.Header.Set("X-Static-B", "2")
	before := req.Header.Clone()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if !reflect.DeepEqual(map[string][]string(req.Header), map[string][]string(before)) {
		t.Fatalf("net/http mutated req.Header during RoundTrip — shared-header COW is UNSAFE.\n before=%v\n after =%v",
			before, req.Header)
	}
}

// TestNetHTTP_SharedHeaderAcrossConcurrentRequests is the stronger proof: many
// concurrent requests that share ONE header map round-trip cleanly under -race.
func TestNetHTTP_SharedHeaderAcrossConcurrentRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Shared") != "v" {
			t.Errorf("server lost shared header: %v", r.Header)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	shared := http.Header{"X-Shared": []string{"v"}} // ONE map, shared by all requests

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
			req.Header = shared // share read-only, do not clone
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("Do: %v", err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	wg.Wait()

	if got := shared.Get("X-Shared"); got != "v" || len(shared) != 1 {
		t.Fatalf("shared header was mutated by concurrent round trips: %v", shared)
	}
}
