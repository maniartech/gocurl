// Package gocurl is a curl-ergonomic HTTP client for Go, built on net/http.
//
// It lets you paste a curl command from any API's documentation straight into
// Go code (or run the identical command with the gocurl CLI), removing the
// translation tax of rewriting curl snippets as net/http requests.
//
// # Quick start
//
// The primary entry point accepts a curl command as a single string or as
// separate arguments and returns a standard *http.Response:
//
//	resp, err := gocurl.Curl(ctx,
//		"-H", "Accept: application/vnd.github+json",
//		"https://api.github.com/repos/golang/go")
//	if err != nil {
//		return err
//	}
//	defer resp.Body.Close()
//
// Convenience helpers read the body for you:
//
//	body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/repos/golang/go")
//	_, err = gocurl.CurlJSON(ctx, &v, "https://api.github.com/repos/golang/go")
//	n, resp, err := gocurl.CurlDownload(ctx, "go.tar.gz", "https://go.dev/dl/...")
//
// # Variables
//
// By default, environment variables ($VAR and ${VAR}) are expanded. For
// explicit, testable control that does not read the process environment, pass a
// Variables map to the WithVars entry points:
//
//	vars := gocurl.Variables{"token": myToken}
//	resp, err := gocurl.CurlWithVars(ctx, vars,
//		"-H", "Authorization: Bearer ${token}",
//		"https://api.github.com/user")
//
// # Scope
//
// gocurl targets curl's HTTP/HTTPS behavior for the flags that appear in real
// API documentation. It is a convenience layer on top of net/http, not a
// replacement for it, and it does not implement curl's non-HTTP protocols.
package gocurl
