package book_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Example harness.
//
// Compiling an example proves it still matches the API; it does NOT prove it runs. This
// harness executes each example as a real subprocess and requires a clean exit, so a
// panic, a nil dereference, or a mis-parsed curl command fails the build.
//
// The examples deliberately keep the real documentation URL as their literal (that is what
// a reader copies). Each testable example additionally honors GOCURL_BOOK_SERVER: when the
// harness sets it, the example points at the local server below instead of the internet.
// An example that does not yet honor the override is REPORTED as un-runnable rather than
// silently skipped — see TestExamples_CoverageReport.

const overrideEnv = "GOCURL_BOOK_SERVER"

// bookServer answers anything an example might ask for with a plausible JSON payload, so
// examples exercise their real code path (parse -> execute -> decode) without the network.
func bookServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/users/"):
			_, _ = w.Write([]byte(`{"login":"octocat","id":1,"name":"The Octocat","public_repos":8}`))
		case strings.Contains(r.URL.Path, "/chat/completions"):
			// OpenAI chat-completion shape, so the example's decode finds its choices.
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` +
				`"HTTP is a request/response protocol."}}],` +
				`"usage":{"prompt_tokens":12,"completion_tokens":8,"total_tokens":20}}`))
		case strings.Contains(r.URL.Path, "/charges"):
			_, _ = w.Write([]byte(`{"id":"ch_test_123","amount":2000,"currency":"usd","status":"succeeded"}`))
		case strings.Contains(r.URL.Path, "/repos/"):
			_, _ = w.Write([]byte(`{"full_name":"golang/go","stargazers_count":120000,"language":"Go"}`))
		default:
			// httpbin-style echo: enough shape for the POST/JSON examples.
			resp := map[string]any{
				"url":     r.URL.String(),
				"method":  r.Method,
				"headers": map[string]string{"User-Agent": r.Header.Get("User-Agent")},
				"json":    map[string]any{"ok": true},
				"args":    map[string]string{},
				"form":    map[string]string{},
				"data":    "",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// exampleDirs returns every directory under book/ that holds a runnable main.go.
func exampleDirs(t *testing.T) []string {
	t.Helper()
	var dirs []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "main.go" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(dirs) == 0 {
		t.Fatal("no examples found — the harness is looking in the wrong place")
	}
	return dirs
}

// honorsOverride reports whether an example reads GOCURL_BOOK_SERVER, i.e. whether the
// harness can run it hermetically.
func honorsOverride(t *testing.T, dir string) bool {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	return strings.Contains(string(src), "demo.URL(")
}

// TestExamples_Run executes every override-aware example against the local server and
// requires a clean exit.
func TestExamples_Run(t *testing.T) {
	if testing.Short() {
		t.Skip("example subprocess runs skipped in -short")
	}
	srv := bookServer(t)

	for _, dir := range exampleDirs(t) {
		if !honorsOverride(t, dir) {
			continue // reported by TestExamples_CoverageReport
		}
		t.Run(filepath.ToSlash(dir), func(t *testing.T) {
			cmd := exec.Command("go", "run", ".")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(),
				overrideEnv+"="+srv.URL,
				// Examples that read credentials must not block or fail on a missing key.
				"GITHUB_TOKEN=test-token",
				"OPENAI_API_KEY=sk-test",
				"API_KEY=test-key",
				"STRIPE_SECRET_KEY=sk_test_dummy",
				// The local harness server is plain http://, and gocurl fail-closes
				// basic/bearer auth over plaintext. That policy firing here is the
				// SECURITY FEATURE working — examples that authenticate would otherwise
				// leak credentials in the clear — so the harness opts out explicitly
				// rather than the examples weakening themselves.
				"GOCURL_ALLOW_INSECURE_AUTH=1",
				"SUPABASE_URL="+srv.URL,
				"SUPABASE_KEY=test-key",
				"SLACK_WEBHOOK_URL="+srv.URL+"/services/test",
			)
			out, err := runWithTimeout(cmd, 60*time.Second)
			if err != nil {
				t.Fatalf("example did not run cleanly: %v\n--- output ---\n%s", err, out)
			}
			if strings.Contains(out, "panic:") {
				t.Errorf("example panicked:\n%s", out)
			}
		})
	}
}

// TestExamples_CoverageReport makes the untested remainder visible instead of letting it
// hide. It lists every example that cannot yet run hermetically. It does not fail the
// build — the count is tracked down deliberately, and the ceiling below ratchets.
func TestExamples_CoverageReport(t *testing.T) {
	dirs := exampleDirs(t)
	var runnable, blocked []string
	for _, d := range dirs {
		if honorsOverride(t, d) {
			runnable = append(runnable, d)
		} else {
			blocked = append(blocked, d)
		}
	}
	t.Logf("examples: %d total, %d runnable hermetically, %d still network-bound",
		len(dirs), len(runnable), len(blocked))
	for _, b := range blocked {
		t.Logf("  network-bound: %s", filepath.ToSlash(b))
	}

	// Ratchet: this ceiling must only ever go DOWN. Lower it as examples are converted.
	const maxNetworkBound = 36
	if len(blocked) > maxNetworkBound {
		t.Errorf("network-bound examples = %d, ceiling %d — a new example was added without "+
			"honoring %s", len(blocked), maxNetworkBound, overrideEnv)
	}
}

func runWithTimeout(cmd *exec.Cmd, d time.Duration) (string, error) {
	var sb strings.Builder
	cmd.Stdout = &sb
	cmd.Stderr = &sb
	if err := cmd.Start(); err != nil {
		return sb.String(), err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return sb.String(), err
	case <-time.After(d):
		_ = cmd.Process.Kill()
		return sb.String(), os.ErrDeadlineExceeded
	}
}
