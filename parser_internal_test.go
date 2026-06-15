package gocurl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/tokenizer"
)

// parseCmd runs the string-command parsing pipeline (tokenize -> env-expand ->
// convert) exactly as CurlCommand does, but stops before execution.
func parseCmd(t *testing.T, cmd string) *options.RequestOptions {
	t.Helper()
	processed := preprocessMultilineCommand(cmd)
	tok := tokenizer.NewTokenizer()
	if err := tok.Tokenize(processed); err != nil {
		t.Fatalf("tokenize(%q): %v", cmd, err)
	}
	tokens := expandEnvInTokens(tok.GetTokens())
	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		t.Fatalf("convert(%q): %v", cmd, err)
	}
	return opts
}

func TestParser_QuotedJSONBody(t *testing.T) {
	opts := parseCmd(t, `curl -X POST -d '{"key":"value"}' https://api.example.com/data`)
	if opts.Body != `{"key":"value"}` {
		t.Errorf("body = %q, want clean JSON without surrounding quotes", opts.Body)
	}
	if got := opts.Headers.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", got)
	}
}

func TestParser_QuotedHeaderWithEnvVar(t *testing.T) {
	t.Setenv("GC_TOKEN", "secret123")
	opts := parseCmd(t, `curl -H 'Authorization: Bearer $GC_TOKEN' https://api.example.com`)
	if got := opts.Headers.Get("Authorization"); got != "Bearer secret123" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer secret123")
	}
}

func TestParser_QuotedURL(t *testing.T) {
	opts := parseCmd(t, `curl 'https://api.example.com/users?q=a b'`)
	if !strings.HasPrefix(opts.URL, "https://api.example.com/users") {
		t.Errorf("URL = %q, want quoted URL accepted", opts.URL)
	}
}

func TestParser_BareHostGetsScheme(t *testing.T) {
	for _, in := range []string{"example.com/health", "localhost:8080/api"} {
		opts := parseCmd(t, "curl "+in)
		if !strings.HasPrefix(opts.URL, "http://") {
			t.Errorf("bare host %q -> URL %q, want http:// scheme", in, opts.URL)
		}
	}
}

func TestParser_DataAtFileReadsContents(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(p, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Use the args path so the @path is not shell-split.
	tokens := []tokenizer.Token{
		{Type: tokenizer.TokenValue, Value: "curl"},
		{Type: tokenizer.TokenValue, Value: "-d"},
		{Type: tokenizer.TokenValue, Value: "@" + p},
		{Type: tokenizer.TokenValue, Value: "https://api.example.com"},
	}
	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Body != `{"a":1}` {
		t.Errorf("body = %q, want file contents", opts.Body)
	}
}

func TestParser_GetModeMovesDataToQuery(t *testing.T) {
	opts := parseCmd(t, `curl -G -d q=golang -d page=2 https://api.example.com/search`)
	if opts.Method != "GET" {
		t.Errorf("method = %q, want GET", opts.Method)
	}
	if !strings.Contains(opts.URL, "q=golang") || !strings.Contains(opts.URL, "page=2") {
		t.Errorf("URL = %q, want data in query string", opts.URL)
	}
	if opts.Body != "" {
		t.Errorf("body = %q, want empty in -G mode", opts.Body)
	}
}

func TestParser_DataURLEncode(t *testing.T) {
	opts := parseCmd(t, `curl --data-urlencode 'name=John Doe' https://api.example.com`)
	if opts.Body != "name=John+Doe" {
		t.Errorf("body = %q, want url-encoded name=John+Doe", opts.Body)
	}
}

func TestParser_FollowRedirectsDefaultsMaxRedirs(t *testing.T) {
	opts := parseCmd(t, `curl -L https://api.example.com`)
	if !opts.FollowRedirects {
		t.Fatal("FollowRedirects should be true with -L")
	}
	if opts.MaxRedirects <= 0 {
		t.Errorf("MaxRedirects = %d, want a sane positive default with -L", opts.MaxRedirects)
	}
}

func TestParser_CookieJarWired(t *testing.T) {
	opts := parseCmd(t, `curl -c cookies.txt https://api.example.com`)
	if opts.CookieFile != "cookies.txt" {
		t.Errorf("CookieFile = %q, want cookies.txt", opts.CookieFile)
	}
}

func TestParser_NewFlagsRecognized(t *testing.T) {
	// These previously hard-failed with "unknown flag".
	cases := []string{
		`curl -I https://api.example.com`,
		`curl --connect-timeout 5 https://api.example.com`,
		`curl --retry 3 https://api.example.com`,
		`curl --url https://api.example.com`,
	}
	for _, c := range cases {
		if opts := parseCmd(t, c); opts.URL == "" {
			t.Errorf("%q produced empty URL", c)
		}
	}
}

func TestParser_UnknownFlagStillErrors(t *testing.T) {
	tok := tokenizer.NewTokenizer()
	_ = tok.Tokenize("curl --definitely-not-a-flag https://api.example.com")
	if _, err := convertTokensToRequestOptions(tok.GetTokens()); err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestParser_WithVarsDoesNotLeakEnv(t *testing.T) {
	t.Setenv("GC_SECRET", "leaked-value")
	// Explicit-vars path with an empty map must NOT pull GC_SECRET from the env.
	tokens := []tokenizer.Token{
		{Type: tokenizer.TokenValue, Value: "curl"},
		{Type: tokenizer.TokenValue, Value: "-H"},
		{Type: tokenizer.TokenValue, Value: "Authorization: Bearer $GC_SECRET"},
		{Type: tokenizer.TokenValue, Value: "https://api.example.com"},
	}
	expanded := expandVarsInTokens(tokens, Variables{})
	opts, err := convertTokensToRequestOptions(expanded)
	if err != nil {
		t.Fatal(err)
	}
	if got := opts.Headers.Get("Authorization"); strings.Contains(got, "leaked-value") {
		t.Errorf("env value leaked into explicit-vars path: %q", got)
	}
}
