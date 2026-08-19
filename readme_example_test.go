package gocurl_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// README example verification.
//
// The README is the first thing a visitor reads, so its code must be executable truth,
// not prose. Two layers guard it:
//
//  1. TestReadmeExamples_SymbolsExist — every gocurl.X / options.X identifier used in a
//     README Go block must exist in the locked public API surface. This covers ALL blocks,
//     including ones added later, and catches a rename or removal immediately.
//  2. The TestReadmeExample_* behavioral tests below run each documented pattern against a
//     hermetic httptest server and assert it does what the README says it does.
//
// This is the motto ("persuasion by example") applied to the front door.

// readmeGoBlocks returns the contents of every ```go fenced block in README.md.
func readmeGoBlocks(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	var blocks []string
	var cur []string
	inBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !inBlock && strings.HasPrefix(trimmed, "```go") {
			inBlock = true
			cur = nil
			continue
		}
		if inBlock && trimmed == "```" {
			inBlock = false
			blocks = append(blocks, strings.Join(cur, "\n"))
			continue
		}
		if inBlock {
			cur = append(cur, line)
		}
	}
	if len(blocks) == 0 {
		t.Fatal("no ```go blocks found in README.md — the extractor is broken")
	}
	return blocks
}

// TestReadmeExamples_SymbolsExist asserts every package-qualified identifier the README
// demonstrates is actually part of the locked public surface. A renamed or deleted API
// would otherwise leave the README quietly lying to every new user.
func TestReadmeExamples_SymbolsExist(t *testing.T) {
	surface := map[string]string{
		"gocurl":  "api.txt",
		"options": "api_options.txt",
	}
	loaded := map[string]string{}
	for pkg, file := range surface {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		loaded[pkg] = string(data)
	}

	ref := regexp.MustCompile(`\b(gocurl|options)\.([A-Z][A-Za-z0-9_]*)`)
	seen := map[string]bool{}
	for i, block := range readmeGoBlocks(t) {
		for _, m := range ref.FindAllStringSubmatch(block, -1) {
			pkg, name := m[1], m[2]
			key := pkg + "." + name
			if seen[key] {
				continue
			}
			seen[key] = true
			// Word-boundary match so "Curl" does not satisfy a lookup for "CurlJSON".
			word := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
			if !word.MatchString(loaded[pkg]) {
				t.Errorf("README block %d uses %s, which is not in %s — the example is stale",
					i+1, key, surface[pkg])
			}
		}
	}
	if len(seen) == 0 {
		t.Error("no gocurl./options. references found in README examples — extractor likely broken")
	}
	t.Logf("verified %d distinct public symbols used across README examples", len(seen))
}

// readmeServer returns a hermetic server standing in for a real API: it echoes what it
// received so tests can assert the wire request, and returns a small JSON body.
func readmeServer(t *testing.T, seen func(*http.Request, string)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if seen != nil {
			seen(r, string(b))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"full_name":"golang/go","stargazers_count":42}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestReadmeExample_QuickStartRawString covers the README's opening block: a single raw
// string holding a curl command, executed with gocurl.Curl.
func TestReadmeExample_QuickStartRawString(t *testing.T) {
	var gotMethod, gotAccept string
	srv := readmeServer(t, func(r *http.Request, _ string) {
		gotMethod, gotAccept = r.Method, r.Header.Get("Accept")
	})
	ctx := context.Background()

	resp, err := gocurl.Curl(ctx, `
	  curl `+srv.URL+`/repos/golang/go
	`)
	if err != nil {
		t.Fatalf("Curl: %v", err)
	}
	defer resp.Body.Close()

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotAccept != "*/*" { // curl parity: curl always sends Accept: */*
		t.Errorf("Accept = %q, want */*", gotAccept)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// TestReadmeExample_OpenAIStyleRawString locks the README's headline example: a SINGLE
// raw-string curl command, pasted from a provider's docs, with the API key left as the
// $ENV_VAR the docs themselves show.
//
// It asserts the three things the README implicitly promises:
//  1. $OPENAI_API_KEY is expanded from the process environment into the Authorization header.
//  2. A multi-line raw string with backslash continuations parses as one command.
//  3. The JSON body survives intact and CurlJSON decodes the response into a struct.
func TestReadmeExample_OpenAIStyleRawString(t *testing.T) {
	var gotAuth, gotCT, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hello there."}}]}`))
	}))
	defer srv.Close()

	// The key never appears in the source — exactly as the README shows.
	t.Setenv("OPENAI_API_KEY", "sk-test-secret-123")

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// This is the README snippet, verbatim except for the URL (hermetic test server).
	resp, err := gocurl.CurlJSON(context.Background(), &out, `
		curl `+srv.URL+`/v1/chat/completions \
		  -H "Content-Type: application/json" \
		  -H "Authorization: Bearer $OPENAI_API_KEY" \
		  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Say hello."}]}'
	`)
	if err != nil {
		t.Fatalf("CurlJSON: %v", err)
	}
	defer resp.Body.Close()

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (a -d body implies POST, like curl)", gotMethod)
	}
	if gotAuth != "Bearer sk-test-secret-123" {
		t.Errorf("Authorization = %q — $OPENAI_API_KEY did not expand from the environment", gotAuth)
	}
	if !strings.HasPrefix(gotCT, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("server received malformed JSON body %q: %v", gotBody, err)
	}
	if sent["model"] != "gpt-4o-mini" {
		t.Errorf("body model = %v, want gpt-4o-mini (body did not survive parsing)", sent["model"])
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content != "Hello there." {
		t.Errorf("response did not decode into the struct: %+v", out)
	}
}

// TestReadmeExample_SeparateArgs covers the "As a library" block: each token passed as a
// separate argument, like os.Args.
func TestReadmeExample_SeparateArgs(t *testing.T) {
	var gotAccept string
	srv := readmeServer(t, func(r *http.Request, _ string) {
		gotAccept = r.Header.Get("Accept")
	})

	resp, err := gocurl.Curl(context.Background(),
		"-H", "Accept: application/vnd.github+json",
		srv.URL+"/repos/golang/go",
	)
	if err != nil {
		t.Fatalf("Curl: %v", err)
	}
	defer resp.Body.Close()

	if gotAccept != "application/vnd.github+json" {
		t.Errorf("Accept = %q — an explicit -H must win over curl's default */*", gotAccept)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// TestReadmeExample_ConvenienceHelpers covers the CurlString / CurlJSON / CurlDownload /
// CurlBytes block, including decoding into the documented struct shape.
func TestReadmeExample_ConvenienceHelpers(t *testing.T) {
	srv := readmeServer(t, nil)
	ctx := context.Background()

	body, resp, err := gocurl.CurlString(ctx, srv.URL+"/repos/golang/go")
	if err != nil {
		t.Fatalf("CurlString: %v", err)
	}
	resp.Body.Close()
	if !strings.Contains(body, "golang/go") {
		t.Errorf("CurlString body = %q, want it to contain the payload", body)
	}

	var repo struct {
		FullName string `json:"full_name"`
		Stars    int    `json:"stargazers_count"`
	}
	resp, err = gocurl.CurlJSON(ctx, &repo, srv.URL+"/repos/golang/go")
	if err != nil {
		t.Fatalf("CurlJSON: %v", err)
	}
	resp.Body.Close()
	if repo.FullName != "golang/go" || repo.Stars != 42 {
		t.Errorf("CurlJSON decoded %+v, want {golang/go 42}", repo)
	}

	raw, resp, err := gocurl.CurlBytes(ctx, srv.URL+"/repos/golang/go")
	if err != nil {
		t.Fatalf("CurlBytes: %v", err)
	}
	resp.Body.Close()
	if len(raw) == 0 {
		t.Error("CurlBytes returned an empty body")
	}

	dst := filepath.Join(t.TempDir(), "repo.json")
	n, resp, err := gocurl.CurlDownload(ctx, dst, srv.URL+"/repos/golang/go")
	if err != nil {
		t.Fatalf("CurlDownload: %v", err)
	}
	resp.Body.Close()
	onDisk, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("CurlDownload did not write the file: %v", err)
	}
	if int64(len(onDisk)) != n {
		t.Errorf("CurlDownload reported %d bytes but wrote %d", n, len(onDisk))
	}
}

// TestReadmeExample_ReusableClient covers the pooled-Client block: New with functional
// options, client.Curl, and the parse-once/execute-many Prepare + Do pair.
func TestReadmeExample_ReusableClient(t *testing.T) {
	var gotUA, gotAccept string
	var calls int
	srv := readmeServer(t, func(r *http.Request, _ string) {
		gotUA = r.Header.Get("User-Agent")
		if a := r.Header.Get("Accept"); a != "*/*" {
			gotAccept = a
		}
		calls++
	})
	ctx := context.Background()

	client, err := gocurl.New(
		gocurl.WithTimeout(10*time.Second),
		gocurl.WithRetryAttempts(3),
		gocurl.WithUserAgent("myapp/1.0"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close()

	resp, err := client.Curl(ctx, "curl "+srv.URL+"/repos/golang/go")
	if err != nil {
		t.Fatalf("client.Curl: %v", err)
	}
	resp.Body.Close()
	if gotUA != "myapp/1.0" {
		t.Errorf("User-Agent = %q, want myapp/1.0 (WithUserAgent not applied)", gotUA)
	}

	// Prepare once, execute many — the documented hot path.
	req, err := client.Prepare("curl -H 'Accept: application/json' " + srv.URL + "/v1/items")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	for i := 0; i < 3; i++ {
		resp, err = client.Do(ctx, req)
		if err != nil {
			t.Fatalf("Do #%d: %v", i+1, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json from the prepared request", gotAccept)
	}
	if calls != 4 {
		t.Errorf("server saw %d requests, want 4 (1 Curl + 3 Do)", calls)
	}
}

// TestReadmeExample_TypedBuilder covers the options-builder block: assemble a request
// programmatically, then run it with Execute.
func TestReadmeExample_TypedBuilder(t *testing.T) {
	var gotMethod, gotAuth, gotBody string
	srv := readmeServer(t, func(r *http.Request, body string) {
		gotMethod, gotAuth, gotBody = r.Method, r.Header.Get("Authorization"), body
	})
	token := "tkn-abc"

	opts := options.NewRequestOptionsBuilder().
		SetURL(srv.URL + "/v1/items").
		SetMethod("POST").
		AddHeader("Authorization", "Bearer "+token).
		SetBody(`{"name":"widget"}`).
		Build()

	resp, err := gocurl.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	defer resp.Body.Close()

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotAuth != "Bearer "+token {
		t.Errorf("Authorization = %q, want Bearer %s", gotAuth, token)
	}
	if !strings.Contains(gotBody, `"name":"widget"`) {
		t.Errorf("body = %q, want the builder's JSON body", gotBody)
	}
}

// TestReadmeExample_EnvVarSubstitution covers the default $VAR expansion block.
func TestReadmeExample_EnvVarSubstitution(t *testing.T) {
	var gotAuth string
	srv := readmeServer(t, func(r *http.Request, _ string) {
		gotAuth = r.Header.Get("Authorization")
	})
	t.Setenv("GITHUB_TOKEN", "ghp-from-env")

	resp, err := gocurl.Curl(context.Background(),
		"-H", "Authorization: Bearer $GITHUB_TOKEN",
		srv.URL+"/user",
	)
	if err != nil {
		t.Fatalf("Curl: %v", err)
	}
	defer resp.Body.Close()

	if gotAuth != "Bearer ghp-from-env" {
		t.Errorf("Authorization = %q — $GITHUB_TOKEN did not expand", gotAuth)
	}
}

// TestReadmeExample_CurlWithVars covers the explicit Variables block, and asserts the
// README's security claim: the process environment is NOT consulted on this path.
func TestReadmeExample_CurlWithVars(t *testing.T) {
	var gotAuth string
	srv := readmeServer(t, func(r *http.Request, _ string) {
		gotAuth = r.Header.Get("Authorization")
	})
	// Deliberately set an env var of the same name; it must be ignored.
	t.Setenv("token", "from-environment-MUST-NOT-BE-USED")
	myToken := "explicit-map-token"

	vars := gocurl.Variables{"token": myToken}
	resp, err := gocurl.CurlWithVars(context.Background(), vars,
		"-H", "Authorization: Bearer ${token}",
		srv.URL+"/user",
	)
	if err != nil {
		t.Fatalf("CurlWithVars: %v", err)
	}
	defer resp.Body.Close()

	if gotAuth != "Bearer "+myToken {
		t.Errorf("Authorization = %q, want the explicit map value %q", gotAuth, myToken)
	}
	if strings.Contains(gotAuth, "environment") {
		t.Error("CurlWithVars leaked a process environment variable — it must use only the supplied map")
	}
}

// TestReadmeExample_SDKComposition covers the middleware-composition block: the exact
// nesting order the README prints must compile, build a working http.Client, and serve a
// request through the whole chain.
func TestReadmeExample_SDKComposition(t *testing.T) {
	var served int
	srv := readmeServer(t, func(*http.Request, string) { served++ })

	base := gocurl.HandlerFromRoundTripper(http.DefaultTransport)
	limiter := gocurl.NewTokenBucket(20, 40)

	h := gocurl.Observe(gocurl.Hooks{}, nil, nil, nil)(
		gocurl.CircuitBreaker(gocurl.BreakerConfig{})(
			gocurl.RateLimiter(limiter)(
				gocurl.Retry(gocurl.DefaultRetryPolicy(3))(base))))

	sdkHTTPClient := &http.Client{
		Transport: gocurl.RoundTripperFromHandler(h),
		Timeout:   30 * time.Second,
	}

	resp, err := sdkHTTPClient.Get(srv.URL + "/v1/items")
	if err != nil {
		t.Fatalf("composed client request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if served != 1 {
		t.Errorf("server saw %d requests, want 1 through the composed chain", served)
	}
}
