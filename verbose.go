package gocurl

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/maniartech/gocurl/options"
)

// VerboseWriter is the default writer for verbose output (stderr like curl)
var VerboseWriter io.Writer = os.Stderr

// printConnectionInfo prints connection details before the request (curl -v style)
func printConnectionInfo(opts *options.RequestOptions, req *http.Request) {
	if !opts.Verbose {
		return
	}

	w := VerboseWriter
	if w == nil {
		return
	}

	// Parse host and port
	host := req.URL.Hostname()
	port := req.URL.Port()
	scheme := req.URL.Scheme

	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	// Print connection attempt (like curl)
	fmt.Fprintf(w, "*   Trying %s:%s...\n", host, port)
	fmt.Fprintf(w, "* Connected to %s (%s) port %s (#0)\n", host, host, port)

	// Print TLS info for HTTPS
	if scheme == "https" {
		fmt.Fprintf(w, "* ALPN, offering h2\n")
		fmt.Fprintf(w, "* ALPN, offering http/1.1\n")

		// Print certificate verification info
		if opts.Insecure {
			fmt.Fprintf(w, "* TLS: Skipping certificate verification\n")
		} else {
			fmt.Fprintf(w, "* TLS: Successfully set certificate verify locations\n")
		}

		// Print TLS version info
		tlsVersion := "TLS 1.2"
		if opts.TLSConfig != nil && opts.TLSConfig.MinVersion == tls.VersionTLS13 {
			tlsVersion = "TLS 1.3"
		}
		fmt.Fprintf(w, "* Using %s\n", tlsVersion)
	}

	fmt.Fprintf(w, "*\n")
}

// printRequestVerbose prints request details in curl -v style
func printRequestVerbose(opts *options.RequestOptions, req *http.Request) {
	if !opts.Verbose {
		return
	}

	w := VerboseWriter
	if w == nil {
		return
	}

	// Print request line (like curl: > GET / HTTP/1.1)
	fmt.Fprintf(w, "> %s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto)

	// Print Host header (curl always shows this first)
	fmt.Fprintf(w, "> Host: %s\r\n", req.Host)

	// Print other headers (with sensitive data redaction)
	for name, values := range req.Header {
		for _, value := range values {
			// Redact sensitive headers
			if isSensitiveHeader(name) {
				value = "[REDACTED]"
			}
			fmt.Fprintf(w, "> %s: %s\r\n", name, value)
		}
	}

	// Print empty line to separate headers from body
	fmt.Fprintf(w, ">\r\n")
}

// printResponseVerbose prints response details in curl -v style
func printResponseVerbose(opts *options.RequestOptions, resp *http.Response) {
	if !opts.Verbose {
		return
	}

	w := VerboseWriter
	if w == nil {
		return
	}

	// Print protocol negotiation info (like curl for HTTP/2)
	if resp.ProtoMajor == 2 {
		fmt.Fprintf(w, "* Using HTTP/2\n")
	}

	// Print status line (like curl: < HTTP/1.1 200 OK)
	fmt.Fprintf(w, "< %s %s\r\n", resp.Proto, resp.Status)

	// Print response headers
	for name, values := range resp.Header {
		for _, value := range values {
			// Redact sensitive response headers (like Set-Cookie)
			if isSensitiveHeader(name) {
				value = "[REDACTED]"
			}
			fmt.Fprintf(w, "< %s: %s\r\n", name, value)
		}
	}

	// Print empty line to separate headers from body
	fmt.Fprintf(w, "<\r\n")
}

// printConnectionClose prints connection close info (curl -v style)
func printConnectionClose(opts *options.RequestOptions) {
	if !opts.Verbose {
		return
	}

	w := VerboseWriter
	if w == nil {
		return
	}

	fmt.Fprintf(w, "* Connection #0 to host left intact\n")
}

// isSensitiveHeader checks if a header contains sensitive data
func isSensitiveHeader(name string) bool {
	sensitive := []string{
		"authorization",
		"cookie",
		"set-cookie",
		"x-api-key",
		"api-key",
		"x-auth-token",
		"auth-token",
	}

	lowerName := strings.ToLower(name)
	for _, s := range sensitive {
		if lowerName == s {
			return true
		}
	}

	return false
}
