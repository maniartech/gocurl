// Package demo supports the book's runnable examples.
//
// Examples keep the REAL documentation URL in the source, because that is what a reader
// copies. Wrapping it in demo.URL lets the repository's test harness redirect the request
// to a local server so every example can actually be executed in CI, instead of only being
// compile-checked. When GOCURL_BOOK_SERVER is unset — which is always the case when you run
// an example yourself — demo.URL returns the URL unchanged.
package demo

import (
	"net/url"
	"os"
)

// ServerEnv is the environment variable the test harness sets to redirect examples.
const ServerEnv = "GOCURL_BOOK_SERVER"

// URL returns docURL unchanged, unless GOCURL_BOOK_SERVER is set, in which case the
// request is redirected to that base while preserving the path and query.
func URL(docURL string) string {
	base := os.Getenv(ServerEnv)
	if base == "" {
		return docURL
	}
	u, err := url.Parse(docURL)
	if err != nil {
		return base
	}
	redirected := base + u.EscapedPath()
	if u.RawQuery != "" {
		redirected += "?" + u.RawQuery
	}
	return redirected
}
