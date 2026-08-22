package gocurl

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestBookReleaseParity keeps the published learning material on the same API and
// toolchain baseline as the executable library. The books contain many non-compiled
// Markdown snippets, so ordinary go test coverage cannot detect a renamed package or
// removed function there; this small semantic guard closes that release gap.
func TestBookReleaseParity(t *testing.T) {
	t.Parallel()

	markdown := readBookMarkdown(t, "book")
	forbidden := []struct {
		name    string
		pattern string
	}{
		{name: "former module owner", pattern: `github\.com/stackql/gocurl`},
		{name: "removed Process call", pattern: `gocurl\.Process\b`},
		{name: "removed Process declaration", pattern: `func Process\s*\(`},
		{name: "root RequestOptions constructor", pattern: `gocurl\.NewRequestOptions(?:Builder)?\b`},
		{name: "root legacy retry type", pattern: `gocurl\.RetryConfig\b`},
		{name: "root legacy middleware type", pattern: `gocurl\.MiddlewareFunc\b`},
		{name: "obsolete Go prerequisite", pattern: `(?i)Go 1\.(?:18|21)(?:\+|\s+or\s+(?:later|higher))`},
		{name: "obsolete workflow Go", pattern: `go-version:\s*['\"]1\.21`},
		{name: "obsolete Go container", pattern: `golang:1\.21\b`},
	}
	for _, check := range forbidden {
		check := check
		t.Run(check.name, func(t *testing.T) {
			re := regexp.MustCompile(check.pattern)
			for name, body := range markdown {
				if loc := re.FindStringIndex(body); loc != nil {
					line := 1 + strings.Count(body[:loc[0]], "\n")
					t.Errorf("%s:%d contains %s", name, line, check.name)
				}
			}
		})
	}

	requireBookStatements(t, markdown, "book/README.md", []string{
		"Go 1.25", "gocurl.Retry", "gocurl.Observe", "gocurl.WithSSRFGuard",
		"DNS rebinding", "Client.Prepare", "Client.Do",
	})
	requireBookStatements(t, markdown, "book/API_REFERENCE.md", []string{
		"Minimum Go:** 1.25", "gocurl.Retry", "gocurl.Observe",
		"gocurl.WithSSRFGuard", "DNS-rebinding", "HandlerFromRoundTripper",
	})

	// The manuscript's runnable examples are a separate module. Requiring the same
	// directive in both modules prevents prose saying 1.25 while examples silently
	// retain an older, vulnerable dependency graph.
	for _, name := range []string{"go.mod", filepath.Join("book", "go.mod")} {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(body), "go 1.25.0") {
			t.Errorf("%s must declare go 1.25.0", name)
		}
	}
}

func readBookMarkdown(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			result[filepath.ToSlash(path)] = string(body)
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	return result
}

func requireBookStatements(t *testing.T, markdown map[string]string, name string, required []string) {
	t.Helper()
	body, ok := markdown[name]
	if !ok {
		t.Fatalf("required book file %s is missing", name)
	}
	for _, statement := range required {
		if !strings.Contains(body, statement) {
			t.Errorf("%s must document %q", name, statement)
		}
	}
}
