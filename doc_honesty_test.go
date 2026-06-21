package gocurl_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDocHonestyLint enforces the M12 honesty rule (Spec 14 §D): a document that
// makes a strong reliability/performance claim must cite at least one named test or
// benchmark that actually exists AND is not unconditionally skipped — so a claim can
// never outrun its evidence. A `*_test.go`-citing convention keeps the docs and the
// proof in lockstep: delete or skip the test, and the doc-lint fails until the claim
// is removed too.
//
// Dependency-free (mirrors api_guard_test.go's approach): it scans the docs and the
// test sources by hand, no go/parser, no external tools.
func TestDocHonestyLint(t *testing.T) {
	// Docs that make claims and therefore must cite evidence.
	claimDocs := []string{
		"README.md",
		"VISION.md",
		"docs/operations.md",
		"docs/benchmarking.md",
		"docs/v1-readiness.md",
		"SECURITY.md",
	}
	// Keywords that constitute a strong, evidence-requiring claim.
	claimKeywords := []string{"production-grade", "mission-critical", "best-in-class"}

	citation := regexp.MustCompile("`((?:Test|Benchmark|Fuzz)[A-Za-z0-9_]+)`")

	tests := collectTestFuncs(t)

	for _, doc := range claimDocs {
		data, err := os.ReadFile(doc)
		if err != nil {
			t.Errorf("read %s: %v", doc, err)
			continue
		}
		text := string(data)
		lower := strings.ToLower(text)

		makesClaim := false
		for _, kw := range claimKeywords {
			if strings.Contains(lower, kw) {
				makesClaim = true
				break
			}
		}
		if !makesClaim {
			continue
		}

		cites := citation.FindAllStringSubmatch(text, -1)
		if len(cites) == 0 {
			t.Errorf("%s makes a strong claim (%v) but cites no `Test*`/`Benchmark*` evidence — "+
				"every such claim must name an un-skipped test/benchmark", doc, claimKeywords)
			continue
		}
		for _, m := range cites {
			name := m[1]
			body, ok := tests[name]
			if !ok {
				t.Errorf("%s cites `%s`, which does not exist as a Test/Benchmark/Fuzz func — "+
					"a claim must cite REAL evidence", doc, name)
				continue
			}
			// "Un-skipped": an UNCONDITIONAL Skip cannot back a claim. A conditional
			// skip is fine — a -short gate (still runs in the full job) or an
			// environment guard (curl absent, hijack unsupported) still runs normally.
			// Heuristic: a Skip is unconditional only if no `if` guard precedes it in
			// the function body.
			if isUnconditionallySkipped(body) {
				t.Errorf("%s cites `%s`, which is unconditionally skipped — a skipped test "+
					"cannot back a claim", doc, name)
			}
		}
	}

	// The README's "production-grade" wording is gated on M12-T1 (reliability) being
	// landed in the ROADMAP — the claim must not precede its milestone.
	if readme, err := os.ReadFile("README.md"); err == nil {
		if strings.Contains(strings.ToLower(string(readme)), "production-grade") {
			roadmap, rerr := os.ReadFile("ROADMAP.md")
			if rerr != nil {
				t.Errorf("read ROADMAP.md: %v", rerr)
			} else if !roadmapM12T1Done(string(roadmap)) {
				t.Errorf("README claims 'production-grade' but ROADMAP M12-T1 is not marked landed — " +
					"the claim precedes its evidence")
			}
		}
	}
}

// collectTestFuncs returns a map of test/benchmark/fuzz func name -> its source body,
// scanned across every *_test.go in the root module package directory.
func collectTestFuncs(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	funcRe := regexp.MustCompile(`(?m)^func ((?:Test|Benchmark|Fuzz)[A-Za-z0-9_]+)\(`)
	out := map[string]string{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(data)
		locs := funcRe.FindAllStringSubmatchIndex(src, -1)
		for i, loc := range locs {
			name := src[loc[2]:loc[3]]
			start := loc[0]
			end := len(src)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			out[name] = src[start:end]
		}
	}
	return out
}

// isUnconditionallySkipped reports whether a test body contains a Skip call that is
// NOT guarded by a preceding `if` (a conditional/environment/-short skip still runs
// in some environment and may back a claim; an unconditional one never runs).
func isUnconditionallySkipped(body string) bool {
	idx := strings.Index(body, ".Skip(")
	for idx >= 0 {
		preceding := body[:idx]
		if !strings.Contains(preceding, "if ") {
			return true // a Skip with no `if` guard before it — unconditional
		}
		next := strings.Index(body[idx+len(".Skip("):], ".Skip(")
		if next < 0 {
			break
		}
		idx = idx + len(".Skip(") + next
	}
	return false
}

// roadmapM12T1Done reports whether the ROADMAP marks M12-T1 as landed/complete.
func roadmapM12T1Done(roadmap string) bool {
	lower := strings.ToLower(roadmap)
	// Accept either an explicit checkbox or a "landed"/"done" marker near M12-T1.
	for _, marker := range []string{"m12-t1"} {
		idx := strings.Index(lower, marker)
		for idx >= 0 {
			// Look at the surrounding line.
			lineStart := strings.LastIndex(lower[:idx], "\n") + 1
			lineEnd := idx + strings.IndexByte(lower[idx:], '\n')
			if lineEnd < idx {
				lineEnd = len(lower)
			}
			line := lower[lineStart:lineEnd]
			if strings.Contains(line, "landed") || strings.Contains(line, "[x]") || strings.Contains(line, "done") || strings.Contains(line, "complete") {
				return true
			}
			next := strings.Index(lower[idx+len(marker):], marker)
			if next < 0 {
				break
			}
			idx = idx + len(marker) + next
		}
	}
	return false
}
