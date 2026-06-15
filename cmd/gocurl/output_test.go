package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newResp(status int, header http.Header, reqHeader http.Header) *http.Response {
	req := httptest.NewRequest("GET", "http://example.com/path", nil)
	if reqHeader != nil {
		req.Header = reqHeader
	}
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		Proto:      "HTTP/1.1",
		Status:     http.StatusText(status),
		StatusCode: status,
		Header:     header,
		Request:    req,
	}
}

func TestSeparateFlags(t *testing.T) {
	flags, curlArgs := separateFlags([]string{
		"-v", "-o", "out.txt", "-X", "POST", "https://example.com",
	})
	if !contains(flags, "-v") || !contains(flags, "-o") || !contains(flags, "out.txt") {
		t.Errorf("flags = %v", flags)
	}
	if !contains(curlArgs, "-X") || !contains(curlArgs, "POST") || !contains(curlArgs, "https://example.com") {
		t.Errorf("curlArgs = %v", curlArgs)
	}
	if contains(curlArgs, "-o") || contains(curlArgs, "out.txt") {
		t.Errorf("gocurl flags leaked into curlArgs: %v", curlArgs)
	}
}

func TestGetFirstNonEmpty(t *testing.T) {
	if got := getFirstNonEmpty("", "", "x", "y"); got != "x" {
		t.Errorf("got %q", got)
	}
	if got := getFirstNonEmpty("", ""); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestGetExitCode(t *testing.T) {
	cases := map[string]int{
		"no URL provided":    3,
		"request timeout":    28,
		"connection refused": 7,
		"some other error":   1,
	}
	for msg, want := range cases {
		if got := getExitCode(errString(msg)); got != want {
			t.Errorf("getExitCode(%q) = %d, want %d", msg, got, want)
		}
	}
	if got := getExitCode(nil); got != 0 {
		t.Errorf("nil err -> %d, want 0", got)
	}
}

func TestFormatBodyOutput_PrettyJSON(t *testing.T) {
	resp := newResp(200, http.Header{"Content-Type": {"application/json"}}, nil)
	out := formatBodyOutput(resp, []byte(`{"b":2,"a":1}`))
	if !strings.Contains(out, "\n") || !strings.Contains(out, "\"a\": 1") {
		t.Errorf("expected pretty-printed JSON, got: %q", out)
	}
}

func TestFormatBodyOutput_NonJSONVerbatim(t *testing.T) {
	resp := newResp(200, http.Header{"Content-Type": {"text/plain"}}, nil)
	out := formatBodyOutput(resp, []byte("plain text"))
	if out != "plain text" {
		t.Errorf("got %q", out)
	}
}

func TestFormatHeaderOutput(t *testing.T) {
	resp := newResp(200, http.Header{"X-Custom": {"yes"}}, nil)
	out := formatHeaderOutput(resp, []byte("body"))
	if !strings.Contains(out, "X-Custom: yes") || !strings.Contains(out, "body") {
		t.Errorf("got %q", out)
	}
}

func TestFormatVerboseOutput_RedactsSecrets(t *testing.T) {
	reqHeader := http.Header{
		"Authorization": {"Bearer supersecret"},
		"X-Public":      {"visible"},
	}
	resp := newResp(200, http.Header{"Set-Cookie": {"session=abc"}}, reqHeader)
	out := formatVerboseOutput(resp, []byte("body"))
	if strings.Contains(out, "supersecret") {
		t.Errorf("Authorization not redacted: %q", out)
	}
	if strings.Contains(out, "session=abc") {
		t.Errorf("Set-Cookie not redacted: %q", out)
	}
	if !strings.Contains(out, "visible") {
		t.Errorf("non-sensitive header missing: %q", out)
	}
}

func TestFormatWriteOut(t *testing.T) {
	resp := newResp(201, http.Header{"Content-Type": {"application/json"}}, nil)
	got := formatWriteOut(resp, "%{http_code} %{content_type}\\n")
	if !strings.Contains(got, "201") || !strings.Contains(got, "application/json") {
		t.Errorf("got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline expansion, got %q", got)
	}
}

func TestFormatAndPrintResponse_FileOutput(t *testing.T) {
	resp := newResp(200, http.Header{"Content-Type": {"text/plain"}}, nil)
	path := filepath.Join(t.TempDir(), "out.txt")
	opts := OutputOptions{OutputFile: path}
	if err := FormatAndPrintResponse(resp, []byte("file body"), opts); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "file body" {
		t.Errorf("file = %q err=%v", data, err)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

type errString string

func (e errString) Error() string { return string(e) }
