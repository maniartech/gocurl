package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
)

// recordingServer is an httptest server that captures the last request it saw,
// so a CLI invocation and the equivalent library call can be compared.
type recordingServer struct {
	*httptest.Server
	method string
	path   string
	body   string
	header http.Header
}

func newRecordingServer(t *testing.T, status int, respBody string) *recordingServer {
	t.Helper()
	rs := &recordingServer{}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rs.method = r.Method
		rs.path = r.URL.Path
		rs.body = string(b)
		rs.header = r.Header.Clone()
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(rs.Close)
	return rs
}

// runArgs drives the CLI in-process with buffers and returns the captured streams.
func runArgs(args ...string) (stdout, stderr string, exit int) {
	var out, errb strings.Builder
	exit = run(args, &out, &errb)
	return out.String(), errb.String(), exit
}

func TestRun_DefaultBodyToStdout(t *testing.T) {
	rs := newRecordingServer(t, 200, "HELLO-BODY")
	stdout, stderr, exit := runArgs(rs.URL)
	if exit != 0 {
		t.Fatalf("exit = %d, stderr=%q", exit, stderr)
	}
	if n := strings.Count(stdout, "HELLO-BODY"); n != 1 {
		t.Errorf("body printed %d times, want 1: %q", n, stdout)
	}
	if stderr != "" {
		t.Errorf("stderr should be empty on success, got: %q", stderr)
	}
}

func TestRun_VerboseTraceToStderr_BodyToStdout(t *testing.T) {
	rs := newRecordingServer(t, 200, "BODYMARKER")
	stdout, stderr, exit := runArgs("-v", "-H", "Authorization: Bearer supersecret", rs.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	// Body on stdout only.
	if !strings.Contains(stdout, "BODYMARKER") {
		t.Errorf("body missing from stdout: %q", stdout)
	}
	if strings.Contains(stderr, "BODYMARKER") {
		t.Errorf("body must not appear on stderr (pipe-clean stdout): %q", stderr)
	}
	// Verbose trace markers on stderr.
	if !strings.Contains(stderr, "> ") || !strings.Contains(stderr, "< ") {
		t.Errorf("verbose trace not on stderr: %q", stderr)
	}
	// Secret redacted on every stream.
	if strings.Contains(stdout+stderr, "supersecret") {
		t.Errorf("verbose output leaked the secret: %q / %q", stdout, stderr)
	}
}

func TestRun_IncludeHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()
	stdout, _, exit := runArgs("-i", srv.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if !strings.Contains(stdout, "X-Custom") || !strings.Contains(stdout, "body") {
		t.Errorf("-i should include headers and body: %q", stdout)
	}
}

func TestRun_Silent(t *testing.T) {
	rs := newRecordingServer(t, 200, "shh")
	stdout, stderr, exit := runArgs("-s", rs.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("-s should suppress stdout, got: %q", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("-s should suppress diagnostics, got: %q", stderr)
	}
}

func TestRun_OutputFileNoStdout(t *testing.T) {
	rs := newRecordingServer(t, 200, "FILEBODY")
	out := filepath.Join(t.TempDir(), "resp.txt")
	stdout, _, exit := runArgs("-o", out, rs.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if strings.Contains(stdout, "FILEBODY") {
		t.Errorf("-o must not also print to stdout: %q", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(data), "FILEBODY") {
		t.Errorf("output file missing/empty: %v / %q", err, data)
	}
}

func TestRun_WriteOut(t *testing.T) {
	rs := newRecordingServer(t, 201, "x")
	stdout, _, exit := runArgs("-w", "code=%{http_code}", rs.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if !strings.Contains(stdout, "code=201") {
		t.Errorf("-w expansion missing: %q", stdout)
	}
}

func TestRun_Fail_ExitCode22(t *testing.T) {
	rs := newRecordingServer(t, 404, "not found")
	_, stderr, exit := runArgs("-f", rs.URL)
	if exit != 22 {
		t.Fatalf("--fail on 404 should exit 22, got %d (stderr=%q)", exit, stderr)
	}
}

func TestRun_NoFail_Exit0_On404(t *testing.T) {
	rs := newRecordingServer(t, 404, "still-data")
	stdout, _, exit := runArgs(rs.URL)
	if exit != 0 {
		t.Fatalf("without --fail a 404 is not an error, exit=%d", exit)
	}
	if !strings.Contains(stdout, "still-data") {
		t.Errorf("4xx body should still print: %q", stdout)
	}
}

func TestRun_UnknownFlag_Exit2(t *testing.T) {
	_, stderr, exit := runArgs("--definitely-not-a-flag", "http://example.invalid")
	if exit != 2 {
		t.Fatalf("unknown curl flag should exit 2 (KindParse), got %d", exit)
	}
	if !strings.Contains(stderr, "definitely-not-a-flag") {
		t.Errorf("error should name the offending flag: %q", stderr)
	}
}

func TestRun_NoArgs_UsageToStderr_Exit2(t *testing.T) {
	stdout, stderr, exit := runArgs()
	if exit != 2 {
		t.Errorf("no-args should exit 2, got %d", exit)
	}
	if !strings.Contains(stderr, "gocurl") {
		t.Errorf("usage should print to stderr: %q", stderr)
	}
	if stdout != "" {
		t.Errorf("usage must not go to stdout: %q", stdout)
	}
}

// TestRun_LibraryCLIParity is the Spec 13 parity check: the SAME curl command,
// once as a library string call and once as the CLI arg slice, must produce an
// identical request on the wire (method, path, body, and a representative header).
func TestRun_LibraryCLIParity(t *testing.T) {
	// Library path.
	libSrv := newRecordingServer(t, 200, "ok")
	_, err := gocurl.Curl(context.Background(),
		"curl -X POST -d k=v -H 'X-Parity: yes' "+libSrv.URL+"/p")
	if err != nil {
		t.Fatalf("library call failed: %v", err)
	}

	// CLI path (equivalent argv; presentation flags absent so both route the same).
	cliSrv := newRecordingServer(t, 200, "ok")
	_, _, exit := runArgs("-X", "POST", "-d", "k=v", "-H", "X-Parity: yes", cliSrv.URL+"/p")
	if exit != 0 {
		t.Fatalf("CLI call exit = %d", exit)
	}

	if libSrv.method != cliSrv.method {
		t.Errorf("method mismatch: lib=%q cli=%q", libSrv.method, cliSrv.method)
	}
	if libSrv.path != cliSrv.path {
		t.Errorf("path mismatch: lib=%q cli=%q", libSrv.path, cliSrv.path)
	}
	if libSrv.body != cliSrv.body {
		t.Errorf("body mismatch: lib=%q cli=%q", libSrv.body, cliSrv.body)
	}
	if libSrv.header.Get("X-Parity") != cliSrv.header.Get("X-Parity") {
		t.Errorf("header mismatch: lib=%q cli=%q",
			libSrv.header.Get("X-Parity"), cliSrv.header.Get("X-Parity"))
	}
}

// TestRun_OutputFileVerbatimJSON guards against the pretty-print corruption bug:
// -o must save the server's exact bytes (no key reorder, no added newline).
func TestRun_OutputFileVerbatimJSON(t *testing.T) {
	const raw = `{"b":2,"a":1}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer srv.Close()
	out := filepath.Join(t.TempDir(), "resp.json")
	if _, _, exit := runArgs("-o", out, srv.URL); exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != raw {
		t.Errorf("-o must save verbatim bytes, got %q want %q", data, raw)
	}
}

// TestRun_SilentStillEmitsWriteOut covers the `curl -s -o /dev/null -w …` idiom:
// -w output is explicit data and survives -s; the body does not.
func TestRun_SilentStillEmitsWriteOut(t *testing.T) {
	rs := newRecordingServer(t, 200, "BODYX")
	stdout, _, exit := runArgs("-s", "-w", "code=%{http_code}", rs.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if !strings.Contains(stdout, "code=200") {
		t.Errorf("-w must still print under -s, got %q", stdout)
	}
	if strings.Contains(stdout, "BODYX") {
		t.Errorf("-s must still suppress the body, got %q", stdout)
	}
}

// TestRun_SizeDownloadChunked verifies %{size_download} reports the real byte
// count even when the server sends no Content-Length (chunked).
func TestRun_SizeDownloadChunked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Content-Length set + Flush → chunked transfer-encoding.
		_, _ = w.Write([]byte("0123456789"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()
	stdout, _, exit := runArgs("-s", "-w", "size=%{size_download}", srv.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if !strings.Contains(stdout, "size=10") {
		t.Errorf("size_download should be the actual 10 bytes, got %q", stdout)
	}
}

func TestRun_MissingURL_Exit3(t *testing.T) {
	if _, _, exit := runArgs("-X", "POST"); exit != 3 {
		t.Fatalf("missing URL should exit 3 (KindValidation), got %d", exit)
	}
}

func TestRun_MalformedURL_Exit3(t *testing.T) {
	for _, u := range []string{"ht!tp://exa mple", "http://"} {
		if _, _, exit := runArgs(u); exit != 3 {
			t.Errorf("malformed URL %q should exit 3, got %d", u, exit)
		}
	}
}

// TestRun_TooManyRedirects_Exit47 drives a self-redirect loop past the cap.
func TestRun_TooManyRedirects_Exit47(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound) // relative → same host, infinite
	}))
	defer srv.Close()
	if _, _, exit := runArgs("-L", srv.URL); exit != 47 {
		t.Fatalf("redirect loop should exit 47, got %d", exit)
	}
}

// TestRun_BodyPrintedOnceAcrossModes guards the double-print regression across
// every stdout-bearing flag combination.
func TestRun_BodyPrintedOnceAcrossModes(t *testing.T) {
	const marker = "UNIQUEBODY"
	combos := [][]string{
		{},
		{"-i"},
		{"-v"},
		{"-v", "-i"},
		{"-w", "x=%{http_code}"},
	}
	for _, extra := range combos {
		t.Run(strings.Join(append([]string{"args"}, extra...), "_"), func(t *testing.T) {
			rs := newRecordingServer(t, 200, marker)
			args := append(append([]string{}, extra...), rs.URL)
			stdout, _, exit := runArgs(args...)
			if exit != 0 {
				t.Fatalf("exit = %d", exit)
			}
			if n := strings.Count(stdout, marker); n != 1 {
				t.Errorf("body printed %d times (want 1) for %v: %q", n, extra, stdout)
			}
		})
	}
}
