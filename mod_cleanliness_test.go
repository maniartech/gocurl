package gocurl_test

import (
	"os"
	"strings"
	"testing"
)

// TestNoVendorDepsInRootModule is the dependency diff-guard (Spec 14 §B): the
// competitive benchmark clients (resty, req) and their heavy transitive graph
// (quic-go, utls, ginkgo, …) live in the SEPARATE benchcmp module. They must NEVER
// leak into the library's own require graph — a gocurl user importing the library
// should not transitively pull resty or a QUIC stack. This asserts the root go.mod
// stays free of those tokens; if a competitor import accidentally lands in a
// non-_test file of the root module, `go mod tidy` would add it here and this fails.
func TestNoVendorDepsInRootModule(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	gomod := string(data)

	forbidden := []string{
		"go-resty/resty",
		"imroc/req",
		"quic-go",
		"refraction-networking/utls",
		"onsi/ginkgo",
		"gocurl/benchcmp",
	}
	for _, tok := range forbidden {
		if strings.Contains(gomod, tok) {
			t.Errorf("root go.mod must not depend on %q — competitive benchmark deps belong "+
				"in the benchcmp module only (keep the library's dependency graph minimal)", tok)
		}
	}
}
