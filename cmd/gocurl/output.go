package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// OutputOptions controls how the response is formatted
type OutputOptions struct {
	Verbose       bool   // -v: Show headers and request details
	IncludeHeader bool   // -i: Include response headers in output
	Silent        bool   // -s: Silent mode, no progress or error output
	OutputFile    string // -o: Write output to file instead of stdout
	WriteOut      string // -w: Custom output format
}

// FormatAndPrintResponse handles output according to options
func FormatAndPrintResponse(resp *http.Response, body []byte, opts OutputOptions) error {
	var output string

	// Verbose mode: show connection info + headers + body
	if opts.Verbose {
		output = formatVerboseOutput(resp, body)
	} else if opts.IncludeHeader {
		// Include headers mode: headers + body
		output = formatHeaderOutput(resp, body)
	} else {
		// Default: body only
		output = formatBodyOutput(resp, body)
	}

	// Apply write-out format if specified
	if opts.WriteOut != "" {
		output += formatWriteOut(resp, opts.WriteOut)
	}

	// Write to file or stdout
	if opts.OutputFile != "" {
		return os.WriteFile(opts.OutputFile, []byte(output), 0644)
	}

	if !opts.Silent {
		fmt.Print(output)
	}

	return nil
}

// formatVerboseOutput formats output like curl -v
func formatVerboseOutput(resp *http.Response, body []byte) string {
	var sb strings.Builder

	// Request line
	sb.WriteString(fmt.Sprintf("> %s %s %s\n", resp.Request.Method, resp.Request.URL.Path, resp.Request.Proto))

	// Request headers
	sb.WriteString("> Host: " + resp.Request.URL.Host + "\n")
	for key, values := range resp.Request.Header {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("> %s: %s\n", key, value))
		}
	}
	sb.WriteString(">\n")

	// Response line
	sb.WriteString(fmt.Sprintf("< %s %s\n", resp.Proto, resp.Status))

	// Response headers
	for key, values := range resp.Header {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("< %s: %s\n", key, value))
		}
	}
	sb.WriteString("<\n")

	// Body
	sb.WriteString(string(body))

	return sb.String()
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

// formatWriteOut formats custom output using curl-style format strings
// Supports: %{http_code}, %{content_type}, %{size_download}, etc.
func formatWriteOut(resp *http.Response, format string) string {
	replacer := strings.NewReplacer(
		"%{http_code}", fmt.Sprintf("%d", resp.StatusCode),
		"%{content_type}", resp.Header.Get("Content-Type"),
		"%{size_download}", fmt.Sprintf("%d", resp.ContentLength),
		"\\n", "\n",
		"\\t", "\t",
	)

	return replacer.Replace(format)
}
