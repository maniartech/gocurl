package gocurl_test

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

// Coverage for the explicit-form public entry points: the *Command and *Args
// siblings of the variadic Curl* helpers. They ship in api.txt and are documented,
// but had ZERO test coverage — this file closes that.
//
// The point of these tests is not "it returns 200". It is the SEMANTIC CONTRACT that
// distinguishes the two forms:
//
//	*Command(ctx, "curl -H 'X: a b' url")  -> ONE string, tokenized like a shell line
//	*Args(ctx, "-H", "X: a b", "url")      -> pre-split tokens, each taken LITERALLY
//
// The literal-token property is what makes *Args safe for values containing spaces or
// quotes (they are never re-tokenized), so each test below asserts it explicitly.

// variantServer echoes the parts of the request the tests assert on.
func variantServer(t *testing.T) (*httptest.Server, func() (method, hdr, body string)) {
	t.Helper()
	var method, hdr, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		hdr = r.Header.Get("X-Probe")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"n":7}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() (string, string, string) { return method, hdr, body }
}

// spacey is a header/body value containing spaces AND embedded quotes. Passed through
// an *Args form it must arrive byte-identical; that is the reason the Args form exists.
const spacey = `hello world "quoted" value`

func TestVariants_StringCommandAndArgs(t *testing.T) {
	ctx := context.Background()

	t.Run("Command tokenizes one shell-like string", func(t *testing.T) {
		srv, probe := variantServer(t)
		body, resp, err := gocurl.CurlStringCommand(ctx,
			`curl -H "X-Probe: from-command" `+srv.URL)
		if err != nil {
			t.Fatalf("CurlStringCommand: %v", err)
		}
		defer resp.Body.Close()
		_, hdr, _ := probe()
		if hdr != "from-command" {
			t.Errorf("X-Probe = %q, want from-command (quoted arg not tokenized correctly)", hdr)
		}
		if !strings.Contains(body, `"ok":true`) {
			t.Errorf("body = %q, want the JSON payload", body)
		}
	})

	t.Run("Args takes each token literally", func(t *testing.T) {
		srv, probe := variantServer(t)
		body, resp, err := gocurl.CurlStringArgs(ctx,
			"-H", "X-Probe: "+spacey, srv.URL)
		if err != nil {
			t.Fatalf("CurlStringArgs: %v", err)
		}
		defer resp.Body.Close()
		_, hdr, _ := probe()
		if hdr != spacey {
			t.Errorf("X-Probe = %q, want %q — Args must NOT re-tokenize a value", hdr, spacey)
		}
		if body == "" {
			t.Error("empty body")
		}
	})
}

func TestVariants_BytesCommandAndArgs(t *testing.T) {
	ctx := context.Background()

	t.Run("Command", func(t *testing.T) {
		srv, _ := variantServer(t)
		b, resp, err := gocurl.CurlBytesCommand(ctx, "curl "+srv.URL)
		if err != nil {
			t.Fatalf("CurlBytesCommand: %v", err)
		}
		defer resp.Body.Close()
		if len(b) == 0 || !strings.Contains(string(b), `"ok"`) {
			t.Errorf("bytes = %q, want the JSON payload", b)
		}
	})

	t.Run("Args preserves a spacey POST body", func(t *testing.T) {
		srv, probe := variantServer(t)
		b, resp, err := gocurl.CurlBytesArgs(ctx, "-d", spacey, srv.URL)
		if err != nil {
			t.Fatalf("CurlBytesArgs: %v", err)
		}
		defer resp.Body.Close()
		method, _, body := probe()
		if method != http.MethodPost {
			t.Errorf("method = %q, want POST (-d implies POST, like curl)", method)
		}
		if body != spacey {
			t.Errorf("body = %q, want %q byte-identical", body, spacey)
		}
		if len(b) == 0 {
			t.Error("empty response bytes")
		}
	})
}

func TestVariants_JSONCommandAndArgs(t *testing.T) {
	ctx := context.Background()
	type payload struct {
		OK bool `json:"ok"`
		N  int  `json:"n"`
	}

	t.Run("Command decodes into the struct", func(t *testing.T) {
		srv, _ := variantServer(t)
		var out payload
		resp, err := gocurl.CurlJSONCommand(ctx, &out, "curl "+srv.URL)
		if err != nil {
			t.Fatalf("CurlJSONCommand: %v", err)
		}
		defer resp.Body.Close()
		if !out.OK || out.N != 7 {
			t.Errorf("decoded %+v, want {true 7}", out)
		}
	})

	t.Run("Args decodes into the struct", func(t *testing.T) {
		srv, probe := variantServer(t)
		var out payload
		resp, err := gocurl.CurlJSONArgs(ctx, &out,
			"-H", "X-Probe: "+spacey, srv.URL)
		if err != nil {
			t.Fatalf("CurlJSONArgs: %v", err)
		}
		defer resp.Body.Close()
		if !out.OK || out.N != 7 {
			t.Errorf("decoded %+v, want {true 7}", out)
		}
		if _, hdr, _ := probe(); hdr != spacey {
			t.Errorf("X-Probe = %q, want %q", hdr, spacey)
		}
	})

	t.Run("malformed JSON surfaces an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{not json`))
		}))
		defer srv.Close()
		var out payload
		resp, err := gocurl.CurlJSONCommand(ctx, &out, "curl "+srv.URL)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Error("expected a decode error for malformed JSON")
		}
	})
}

func TestVariants_DownloadCommandAndArgs(t *testing.T) {
	ctx := context.Background()
	const want = `{"ok":true,"n":7}`

	t.Run("Command writes the file", func(t *testing.T) {
		srv, _ := variantServer(t)
		dst := filepath.Join(t.TempDir(), "cmd.json")
		n, resp, err := gocurl.CurlDownloadCommand(ctx, dst, "curl "+srv.URL)
		if err != nil {
			t.Fatalf("CurlDownloadCommand: %v", err)
		}
		defer resp.Body.Close()
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("file not written: %v", err)
		}
		if string(got) != want {
			t.Errorf("file = %q, want %q", got, want)
		}
		if int64(len(got)) != n {
			t.Errorf("reported %d bytes, wrote %d", n, len(got))
		}
	})

	t.Run("Args writes the file", func(t *testing.T) {
		srv, probe := variantServer(t)
		dst := filepath.Join(t.TempDir(), "args.json")
		n, resp, err := gocurl.CurlDownloadArgs(ctx, dst,
			"-H", "X-Probe: "+spacey, srv.URL)
		if err != nil {
			t.Fatalf("CurlDownloadArgs: %v", err)
		}
		defer resp.Body.Close()
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("file not written: %v", err)
		}
		if int64(len(got)) != n || string(got) != want {
			t.Errorf("file = %q (n=%d), want %q", got, n, want)
		}
		if _, hdr, _ := probe(); hdr != spacey {
			t.Errorf("X-Probe = %q, want %q", hdr, spacey)
		}
	})
}

// TestVariants_CurlCommandWithVars proves the explicit-variables form substitutes from
// the supplied map and — the security-relevant part — does NOT fall back to the process
// environment for a same-named variable.
func TestVariants_CurlCommandWithVars(t *testing.T) {
	srv, probe := variantServer(t)
	t.Setenv("probe", "FROM-ENVIRONMENT-MUST-NOT-APPEAR")

	vars := gocurl.Variables{"probe": "from-explicit-map"}
	resp, err := gocurl.CurlCommandWithVars(context.Background(), vars,
		`curl -H "X-Probe: ${probe}" `+srv.URL)
	if err != nil {
		t.Fatalf("CurlCommandWithVars: %v", err)
	}
	defer resp.Body.Close()

	_, hdr, _ := probe()
	if hdr != "from-explicit-map" {
		t.Errorf("X-Probe = %q, want from-explicit-map", hdr)
	}
	if strings.Contains(hdr, "ENVIRONMENT") {
		t.Error("CurlCommandWithVars leaked a process environment variable")
	}
}

// TestVariants_ErrorPaths asserts each explicit-form entry point surfaces an error
// rather than panicking or hanging on an unusable command.
func TestVariants_ErrorPaths(t *testing.T) {
	ctx := context.Background()
	bad := "curl http://gocurl-nonexistent-host.invalid/"

	if _, _, err := gocurl.CurlStringCommand(ctx, bad); err == nil {
		t.Error("CurlStringCommand: expected an error")
	}
	if _, _, err := gocurl.CurlBytesCommand(ctx, bad); err == nil {
		t.Error("CurlBytesCommand: expected an error")
	}
	var sink struct{}
	if _, err := gocurl.CurlJSONCommand(ctx, &sink, bad); err == nil {
		t.Error("CurlJSONCommand: expected an error")
	}
	dst := filepath.Join(t.TempDir(), "never.bin")
	if _, _, err := gocurl.CurlDownloadCommand(ctx, dst, bad); err == nil {
		t.Error("CurlDownloadCommand: expected an error")
	}
	if _, err := gocurl.CurlCommandWithVars(ctx, gocurl.Variables{}, bad); err == nil {
		t.Error("CurlCommandWithVars: expected an error")
	}
	// A well-formed but empty invocation must be rejected, not panic.
	if _, _, err := gocurl.CurlStringArgs(ctx); err == nil {
		t.Error("CurlStringArgs with no args: expected a validation error")
	}
}
