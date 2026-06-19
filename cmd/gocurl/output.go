package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/maniartech/gocurl"
)

// OutputOptions controls how the response is formatted
type OutputOptions struct {
	Verbose       bool   // -v: Show headers and request details
	IncludeHeader bool   // -i: Include response headers in output
	Silent        bool   // -s: Silent mode, no progress or error output
	OutputFile    string // -o: Write output to file instead of stdout
	WriteOut      string // -w: Custom output format
}

// FormatAndPrintResponse handles output according to options. Stream discipline
// (Spec 13): the verbose trace (`-v`) goes to stderr so stdout stays a clean,
// pipeable body; the body (or `-i` headers+body) goes to stdout (or `-o` file).
// The body is written exactly once across every flag combination.
func FormatAndPrintResponse(resp *http.Response, body []byte, opts OutputOptions, stdout, stderr io.Writer) error {
	// Verbose trace → stderr (redacted), independent of the body destination, so
	// `gocurl -v url | jq` still pipes a clean body. Suppressed by -s.
	if opts.Verbose && !opts.Silent {
		fmt.Fprint(stderr, formatVerboseTrace(resp))
	}

	if opts.OutputFile != "" {
		// -o writes the body to FILE VERBATIM — no JSON pretty-print/key-reorder and
		// no added trailing newline — so the saved bytes match the server exactly
		// (checksums, signatures, binary payloads). With -i the status line +
		// headers precede the verbatim body. Pretty-printing is a stdout-only
		// convenience (Spec 13).
		var filePayload []byte
		if opts.IncludeHeader {
			filePayload = []byte(formatHeaderOutput(resp, body))
		} else {
			filePayload = body
		}
		if err := os.WriteFile(opts.OutputFile, filePayload, 0644); err != nil {
			return err
		}
	} else if !opts.Silent {
		// stdout: pretty-print JSON for human consumption (default mode only).
		var payload string
		if opts.IncludeHeader {
			payload = formatHeaderOutput(resp, body)
		} else {
			payload = formatBodyOutput(resp, body)
		}
		fmt.Fprint(stdout, payload)
	}

	// -w expansion is explicit, user-requested data: it always goes to stdout, even
	// under -s or alongside -o (the canonical `curl -s -o /dev/null -w '%{http_code}'`
	// idiom depends on this).
	if opts.WriteOut != "" {
		fmt.Fprint(stdout, formatWriteOut(resp, body, opts.WriteOut))
	}

	return nil
}

// formatVerboseTrace formats the request/response metadata like curl -v, with
// sensitive headers redacted. It does NOT include the body — the body is written
// to stdout separately so the two streams stay independent.
func formatVerboseTrace(resp *http.Response) string {
	var sb strings.Builder

	// Request line
	sb.WriteString(fmt.Sprintf("> %s %s %s\n", resp.Request.Method, resp.Request.URL.Path, resp.Request.Proto))

	// Request headers (sensitive values redacted, like the library's -v output).
	sb.WriteString("> Host: " + resp.Request.URL.Host + "\n")
	writeRedactedHeaders(&sb, "> ", resp.Request.Header)
	sb.WriteString(">\n")

	// Response line
	sb.WriteString(fmt.Sprintf("< %s %s\n", resp.Proto, resp.Status))

	// Response headers (Set-Cookie etc. redacted).
	writeRedactedHeaders(&sb, "< ", resp.Header)
	sb.WriteString("<\n")

	return sb.String()
}

// writeRedactedHeaders writes headers with sensitive values redacted, matching
// the library's verbose redaction (Authorization, Cookie, API keys, etc.).
func writeRedactedHeaders(sb *strings.Builder, prefix string, h http.Header) {
	for key, values := range h {
		for _, value := range values {
			if gocurl.IsSensitiveHeader(key) {
				value = "[REDACTED]"
			}
			sb.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, key, value))
		}
	}
}

// formatHeaderOutput formats output like curl -i
func formatHeaderOutput(resp *http.Response, body []byte) string {
	var sb strings.Builder

	// Status line
	sb.WriteString(fmt.Sprintf("%s %s\n", resp.Proto, resp.Status))

	// Headers
	for key, values := range resp.Header {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}
	sb.WriteString("\n")

	// Body
	sb.WriteString(string(body))

	return sb.String()
}

// formatBodyOutput formats just the body (default)
func formatBodyOutput(resp *http.Response, body []byte) string {
	// Check if response is JSON and pretty-print
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			formatted, err := json.MarshalIndent(data, "", "  ")
			if err == nil {
				return string(formatted) + "\n"
			}
		}
	}

	// Return raw body
	return string(body)
}

// formatWriteOut formats custom output using curl-style format strings.
// Supports: %{http_code}, %{content_type}, %{size_download}, etc.
// size_download is the number of body bytes actually downloaded (len(body)),
// matching curl — resp.ContentLength is unreliable (-1 for chunked/unknown-length
// and the compressed size for transparently decoded responses).
func formatWriteOut(resp *http.Response, body []byte, format string) string {
	replacer := strings.NewReplacer(
		"%{http_code}", fmt.Sprintf("%d", resp.StatusCode),
		"%{content_type}", resp.Header.Get("Content-Type"),
		"%{size_download}", fmt.Sprintf("%d", len(body)),
		"\\n", "\n",
		"\\t", "\t",
	)

	return replacer.Replace(format)
}
