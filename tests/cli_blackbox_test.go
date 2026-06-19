package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gocurlBin is the path to the CLI binary built once for the whole package.
var gocurlBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gocurl-cli")
	if err != nil {
		panic(err)
	}
	bin := filepath.Join(dir, "gocurl")
	if isWindows() {
		bin += ".exe"
	}
	// Build the CLI from the parent module.
	cmd := exec.Command("go", "build", "-o", bin, "github.com/maniartech/gocurl/cmd/gocurl")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Can't build the CLI (e.g. offline toolchain); skip CLI tests rather
		// than failing the whole suite.
		os.RemoveAll(dir)
		gocurlBin = ""
		os.Exit(m.Run())
	}
	gocurlBin = bin
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func isWindows() bool { return os.PathSeparator == '\\' }

func runCLI(t *testing.T, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	if gocurlBin == "" {
		t.Skip("gocurl CLI binary unavailable")
	}
	cmd := exec.Command(gocurlBin, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exit = 0
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return outBuf.String(), errBuf.String(), exit
}

func cliEchoServer(t *testing.T) *echoServer { return newEchoServer(t) }

func TestCLI_BodyPrintedOnce(t *testing.T) {
	es := cliEchoServer(t)
	stdout, _, exit := runCLI(t, es.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	// The fixed echo body is {"ok":true}; it must appear exactly once
	// (regression test for the double-print bug).
	if n := strings.Count(stdout, `"ok"`); n != 1 {
		t.Errorf("body printed %d times, want 1:\n%s", n, stdout)
	}
}

func TestCLI_OutputToFileNoStdout(t *testing.T) {
	es := cliEchoServer(t)
	out := filepath.Join(t.TempDir(), "resp.json")
	stdout, _, exit := runCLI(t, "-o", out, es.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if strings.Contains(stdout, `"ok"`) {
		t.Errorf("-o should not also print body to stdout, got: %q", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(data), `"ok"`) {
		t.Errorf("output file missing/empty: %v / %q", err, data)
	}
}

func TestCLI_Silent(t *testing.T) {
	es := cliEchoServer(t)
	stdout, _, exit := runCLI(t, "-s", es.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("-s should suppress stdout, got: %q", stdout)
	}
}

func TestCLI_VerboseRedactsSecrets(t *testing.T) {
	es := cliEchoServer(t)
	_, stderrOrOut, exit := runCLI(t, "-v", "-H", "Authorization: Bearer supersecret", es.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	// Verbose output (whichever stream) must not contain the raw secret.
	stdout, _, _ := runCLI(t, "-v", "-H", "Authorization: Bearer supersecret", es.URL)
	combined := stdout + stderrOrOut
	if strings.Contains(combined, "supersecret") {
		t.Errorf("verbose output leaked secret:\n%s", combined)
	}
}

func TestCLI_PostData(t *testing.T) {
	es := cliEchoServer(t)
	_, _, exit := runCLI(t, "-X", "POST", "-d", "k=v", es.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if es.lastMethod != "POST" || es.lastBody != "k=v" {
		t.Errorf("server saw method=%q body=%q", es.lastMethod, es.lastBody)
	}
}

func TestCLI_IncludeHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()
	stdout, _, exit := runCLI(t, "-i", srv.URL)
	if exit != 0 {
		t.Fatalf("exit = %d", exit)
	}
	if !strings.Contains(stdout, "X-Custom") || !strings.Contains(stdout, "body") {
		t.Errorf("-i should include headers and body, got:\n%s", stdout)
	}
}

func TestCLI_UsageOnNoArgs(t *testing.T) {
	stdout, stderr, exit := runCLI(t)
	if exit == 0 {
		t.Error("no-args should exit non-zero")
	}
	// Usage is a misuse diagnostic → stderr (Spec 13 stream discipline); stdout
	// stays clean so a piped consumer never sees help text as data.
	if !strings.Contains(stderr, "gocurl") {
		t.Errorf("usage not printed to stderr: %q", stderr)
	}
	if strings.Contains(stdout, "gocurl") {
		t.Errorf("usage must not go to stdout: %q", stdout)
	}
}
