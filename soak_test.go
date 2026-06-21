package gocurl

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestClient_Soak runs a sustained request loop against an httptest server in TWO
// arms — uninstrumented and fully instrumented (recording Tracer + Metrics + Logger
// + Hooks, the config operators actually run) — and asserts a stable goroutine count
// and no unbounded heap growth in BOTH, so the observability machinery itself does
// not leak under load. It publishes the per-Do alloc delta between the arms so the
// instrumentation cost is honest and visible. Skipped in -short (runs in the
// scheduled CI job, never on the PR path).
//
// Duration: by default it runs a fixed iteration count; set GOCURL_SOAK=<duration>
// (e.g. 30s, 5m) to run each arm for that long instead. GOCURL_PROFILE=<dir> writes
// cpu/mem pprof for the uninstrumented arm.
//
//	GOCURL_SOAK=2m GOCURL_PROFILE=$(mktemp -d) go test -run TestClient_Soak .
func TestClient_Soak(t *testing.T) {
	if testing.Short() {
		t.Skip("soak test skipped in -short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	budget := soakBudget()

	// Arm 1: uninstrumented (the baseline).
	plain, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Close()
	baseAllocs := runSoakArm(t, "uninstrumented", plain, srv.URL, budget, os.Getenv("GOCURL_PROFILE"))

	// Arm 2: fully instrumented with recording sinks (what operators actually run).
	rec := &soakRecorder{}
	inst, err := New(
		WithTracer(rec),
		WithMetrics(rec),
		WithLogger(rec),
		WithHooks(Hooks{
			OnRequest: func(context.Context, *http.Request) { atomic.AddInt64(&rec.hookCalls, 1) },
			OnResponse: func(context.Context, *http.Request, *http.Response, time.Duration) {
				atomic.AddInt64(&rec.hookCalls, 1)
			},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	instAllocs := runSoakArm(t, "instrumented", inst, srv.URL, budget, "")

	if rec.requests.Load() == 0 || rec.spans.Load() == 0 {
		t.Errorf("instrumented arm recorded nothing: requests=%d spans=%d", rec.requests.Load(), rec.spans.Load())
	}
	t.Logf("per-Do allocs: uninstrumented=%.0f instrumented=%.0f (instrumentation delta=%.0f)",
		baseAllocs, instAllocs, instAllocs-baseAllocs)
}

// soakBudget returns the run budget: a duration if GOCURL_SOAK parses as one,
// otherwise a fixed iteration count.
func soakBudget() soakRun {
	if v := os.Getenv("GOCURL_SOAK"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return soakRun{dur: d}
		}
	}
	return soakRun{iters: 3000}
}

type soakRun struct {
	dur   time.Duration
	iters int
}

// runSoakArm drives one arm, sampling goroutines + heap at checkpoints, and returns
// the measured per-Do allocs/op. It fails the test on goroutine growth or a clear
// upward heap trend.
func runSoakArm(t *testing.T, name string, c *Client, url string, budget soakRun, profileDir string) float64 {
	t.Helper()

	var cpuFile *os.File
	if profileDir != "" {
		f, err := os.Create(filepath.Join(profileDir, "cpu.pprof"))
		if err != nil {
			t.Fatalf("create cpu profile: %v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			t.Fatalf("start cpu profile: %v", err)
		}
		cpuFile = f
		defer func() { pprof.StopCPUProfile(); cpuFile.Close() }()
	}

	for i := 0; i < 20; i++ { // warm up the pooled transport
		drainOnce(t, c, url)
	}
	runtime.GC()
	base := goroutinesAtMost(0, 200*time.Millisecond)
	heapStart := heapAllocAfterGC()

	const checkpoints = 8
	heapSamples := make([]uint64, 0, checkpoints)

	run := func(i int) { drainOnce(t, c, url) }
	if budget.dur > 0 {
		deadline := time.Now().Add(budget.dur)
		i := 0
		next := budget.dur / checkpoints
		nextSample := time.Now().Add(next)
		for time.Now().Before(deadline) {
			run(i)
			i++
			if time.Now().After(nextSample) {
				heapSamples = append(heapSamples, heapAllocAfterGC())
				nextSample = nextSample.Add(next)
			}
		}
		t.Logf("%s: completed %d requests over %v", name, i, budget.dur)
	} else {
		step := budget.iters / checkpoints
		for i := 0; i < budget.iters; i++ {
			run(i)
			if step > 0 && i%step == step-1 {
				heapSamples = append(heapSamples, heapAllocAfterGC())
			}
		}
		t.Logf("%s: completed %d requests", name, budget.iters)
	}

	if profileDir != "" {
		f, err := os.Create(filepath.Join(profileDir, "mem.pprof"))
		if err != nil {
			t.Fatalf("create mem profile: %v", err)
		}
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			t.Fatalf("write heap profile: %v", err)
		}
		f.Close()
	}

	// Goroutine count must return near baseline — a per-request leak scales with the
	// request count and blows past this small margin.
	if final := goroutinesAtMost(base+15, 3*time.Second); final > base+15 {
		t.Errorf("%s: goroutine growth: base=%d final=%d", name, base, final)
	}

	// Heap must not trend unbounded. Over thousands of requests a real leak grows
	// without bound; a generous steady-state ceiling catches that without flaking on
	// allocator noise.
	heapEnd := heapAllocAfterGC()
	t.Logf("%s: heap start=%dKiB end=%dKiB samples=%v",
		name, heapStart/1024, heapEnd/1024, kib(heapSamples))
	if heapStart > 0 && heapEnd > heapStart*4 {
		t.Errorf("%s: heap grew from %dKiB to %dKiB (>4x) — possible leak", name, heapStart/1024, heapEnd/1024)
	}

	// Per-Do allocation, measured after warm-up so transport construction is excluded.
	return testing.AllocsPerRun(200, func() { drainOnce(t, c, url) })
}

func heapAllocAfterGC() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

func kib(samples []uint64) []uint64 {
	out := make([]uint64, len(samples))
	for i, s := range samples {
		out[i] = s / 1024
	}
	return out
}

// TestClient_Soak_Backpressure drives sustained CONCURRENT load through a Client
// pinned to a single connection per host (MaxConnsPerHost=1). Requests must queue
// and drain — applying real backpressure — without deadlocking or leaking
// goroutines, and the server must never see more than one concurrent connection.
func TestClient_Soak_Backpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("backpressure soak skipped in -short mode")
	}

	var inFlight, peak int32
	var connMu sync.Mutex
	conns := map[net.Conn]struct{}{}
	maxConns := 0

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&inFlight, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if c <= p || atomic.CompareAndSwapInt32(&peak, p, c) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		_, _ = w.Write([]byte("ok"))
	}))
	srv.Config.ConnState = func(nc net.Conn, s http.ConnState) {
		connMu.Lock()
		switch s {
		case http.StateNew:
			conns[nc] = struct{}{}
			if len(conns) > maxConns {
				maxConns = len(conns)
			}
		case http.StateClosed, http.StateHijacked:
			delete(conns, nc)
		}
		connMu.Unlock()
	}
	srv.Start()
	defer srv.Close()

	c, err := New(WithMaxConnsPerHost(1))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	base := goroutinesAtMost(0, 200*time.Millisecond)

	const workers, perWorker = 16, 60
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				resp, err := c.Curl(context.Background(), "curl "+srv.URL)
				if err != nil {
					t.Errorf("Curl: %v", err)
					return
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}
	wg.Wait() // a missing token-return / deadlock would hang here

	if got := atomic.LoadInt32(&peak); got > 1 {
		t.Errorf("MaxConnsPerHost(1) breached: peak concurrent server-side requests = %d", got)
	}
	connMu.Lock()
	mc := maxConns
	connMu.Unlock()
	if mc > 1 {
		t.Errorf("MaxConnsPerHost(1) breached: server saw %d concurrent connections", mc)
	}
	if final := goroutinesAtMost(base+15, 3*time.Second); final > base+15 {
		t.Errorf("goroutine growth under backpressure: base=%d final=%d", base, final)
	}
}

// --- recording observability sink for the instrumented soak arm ---

type soakRecorder struct {
	requests  atomic.Int64
	spans     atomic.Int64
	logs      atomic.Int64
	hookCalls int64
}

func (r *soakRecorder) IncRequest(RequestInfo)                   { r.requests.Add(1) }
func (r *soakRecorder) IncInFlight(int)                          {}
func (r *soakRecorder) ObserveLatency(time.Duration, ResultInfo) {}
func (r *soakRecorder) IncRetry(RequestInfo)                     {}
func (r *soakRecorder) IncError(Kind, RequestInfo)               {}

func (r *soakRecorder) Log(context.Context, Level, string, ...Field) { r.logs.Add(1) }

func (r *soakRecorder) StartSpan(ctx context.Context, name string, attrs ...Field) (context.Context, Span) {
	r.spans.Add(1)
	return ctx, soakSpan{}
}

type soakSpan struct{}

func (soakSpan) SetAttributes(...Field)    {}
func (soakSpan) AddEvent(string, ...Field) {}
func (soakSpan) RecordError(error)         {}
func (soakSpan) End()                      {}

func drainOnce(t *testing.T, c *Client, url string) {
	t.Helper()
	req, err := NewRequest("GET", url)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
