package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/maniartech/gocurl"
)

func main() {
	// Exit with proper code
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point (Spec 13): it takes the raw argument slice and
// the stdout/stderr writers explicitly so the whole CLI can be unit-tested
// in-process with buffers, with no subprocess. stdout receives the response body
// (or -o/-w formatted output); stderr receives the verbose trace, warnings, and
// errors.
func run(argv []string, stdout, stderr io.Writer) int {
	// Define flags on a LOCAL FlagSet (never the global flag.CommandLine) so
	// run() is reentrant — it can be invoked repeatedly in-process by unit tests
	// without "flag redefined" panics.
	var (
		verbose, verbose2             bool
		includeHeader, includeHeader2 bool
		silent, silent2               bool
		outputFile, outputFile2       string
		writeOut, writeOut2           string
	)

	// Custom flag parsing to handle curl-style flags mixed with arguments
	// We need to separate flags from curl args
	args, curlArgs, sepErr := separateFlags(argv)
	if sepErr != nil {
		fmt.Fprintf(stderr, "gocurl: %v\n", sepErr)
		return 2
	}

	// Parse flags
	fs := flag.NewFlagSet("gocurl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&verbose, "v", false, "")
	fs.BoolVar(&verbose2, "verbose", false, "")
	fs.BoolVar(&includeHeader, "i", false, "")
	fs.BoolVar(&includeHeader2, "include", false, "")
	fs.BoolVar(&silent, "s", false, "")
	fs.BoolVar(&silent2, "silent", false, "")
	fs.StringVar(&outputFile, "o", "", "")
	fs.StringVar(&outputFile2, "output", "", "")
	fs.StringVar(&writeOut, "w", "", "")
	fs.StringVar(&writeOut2, "write-out", "", "")

	if err := fs.Parse(args); err != nil {
		// flag already wrote the parse error to stderr (fs.SetOutput). An
		// unusable invocation is a curl-style "failed to initialize" → exit 2.
		return 2
	}

	// If no curl args, show usage on stderr (a misuse diagnostic, not data).
	if len(curlArgs) == 0 {
		printUsage(stderr)
		return 2
	}

	ctx := context.Background()

	// Build output options
	opts := OutputOptions{
		Verbose:       verbose || verbose2,
		IncludeHeader: includeHeader || includeHeader2,
		Silent:        silent || silent2,
		OutputFile:    getFirstNonEmpty(outputFile, outputFile2),
		WriteOut:      getFirstNonEmpty(writeOut, writeOut2),
	}

	// Execute using SHARED code path (ZERO DIVERGENCE!)
	err := executeCLI(ctx, curlArgs, opts, stdout, stderr)
	if err != nil {
		// Error output to stderr (like curl)
		if !opts.Silent {
			fmt.Fprintf(stderr, "gocurl: %v\n", err)
		}
		return getExitCode(err)
	}

	return 0
}

// separateFlags splits the LEADING run of gocurl presentation flags from the curl
// args. Presentation flags (-v/-i/-s and the value-taking -o/-w, plus long forms)
// are recognized only before the first curl token — matching the documented
// "gocurl [gocurl options] [curl options] <URL>" usage. Once any curl token
// appears, every remaining token is passed VERBATIM to the library, so a curl flag
// value that happens to look like a presentation flag (e.g. `-d -s url`) is no
// longer stolen. A literal "--" ends presentation-flag scanning explicitly.
// A value-taking presentation flag with no following token is a usage error.
func separateFlags(args []string) (flags []string, curlArgs []string, err error) {
	boolFlags := map[string]bool{
		"-v": true, "--verbose": true,
		"-i": true, "--include": true,
		"-s": true, "--silent": true,
	}
	valueFlags := map[string]bool{
		"-o": true, "--output": true,
		"-w": true, "--write-out": true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			i++ // drop the separator; everything after is curl args
			break
		}
		if boolFlags[arg] {
			flags = append(flags, arg)
			i++
			continue
		}
		if valueFlags[arg] {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("option %s requires an argument", arg)
			}
			flags = append(flags, arg, args[i+1])
			i += 2
			continue
		}
		// First non-presentation token: the rest are curl args (verbatim).
		break
	}

	curlArgs = append(curlArgs, args[i:]...)
	return flags, curlArgs, nil
}

// getFirstNonEmpty returns the first non-empty string
func getFirstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// executeCLI uses EXACT same code path as library
func executeCLI(ctx context.Context, args []string, opts OutputOptions, stdout, stderr io.Writer) error {
	// Use the library's CurlArgs function directly
	resp, err := gocurl.CurlArgs(ctx, args...)
	if err != nil {
		// With -f/--fail a >=400 response returns both a response and an error;
		// curl -f discards the body, so close it and surface the error.
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Format and print output
	return FormatAndPrintResponse(resp, body, opts, stdout, stderr)
}

// getExitCode returns an appropriate process exit code for an error, preferring
// gocurl's typed Kind classification (curl-compatible codes) and falling back to
// string matching for errors that are not typed GocurlErrors.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	// Redirect-cap is matched by sentinel (it carries no distinct Kind) and maps to
	// curl's exit 47. Checked before the Kind switch.
	if errors.Is(err, gocurl.ErrTooManyRedirects) {
		return 47 // Number of redirects hit maximum
	}

	switch gocurl.KindOf(err) {
	case gocurl.KindServerStatus:
		return 22 // HTTP page not retrieved (-f/--fail)
	case gocurl.KindTimeout:
		return 28 // Operation timeout
	case gocurl.KindConnect:
		return 7 // Failed to connect to host
	case gocurl.KindTLS:
		return 35 // SSL/TLS connect error
	case gocurl.KindParse:
		return 2 // Failed to initialize / parse
	case gocurl.KindValidation:
		return 3 // URL malformed
	}

	// Fallback: legacy string matching for non-typed errors.
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "no URL"):
		return 3 // URL malformed
	case strings.Contains(errStr, "timeout"):
		return 28 // Operation timeout
	case strings.Contains(errStr, "connection refused"):
		return 7 // Failed to connect
	default:
		return 1 // Generic error
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "gocurl - Execute curl commands in Go")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  gocurl [gocurl options] [curl options] <URL>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GoCurl Options:")
	fmt.Fprintln(w, "  -v, --verbose        Verbose output (show headers and request)")
	fmt.Fprintln(w, "  -i, --include        Include response headers in output")
	fmt.Fprintln(w, "  -s, --silent         Silent mode (no output)")
	fmt.Fprintln(w, "  -o, --output FILE    Write output to file")
	fmt.Fprintln(w, "  -w, --write-out FMT  Custom output format")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Curl Options: (All standard curl options supported)")
	fmt.Fprintln(w, "  -X, --request        HTTP method (GET, POST, PUT, DELETE, etc.)")
	fmt.Fprintln(w, "  -H, --header         Add header")
	fmt.Fprintln(w, "  -d, --data           Request body data")
	fmt.Fprintln(w, "  -u, --user           Basic auth credentials")
	fmt.Fprintln(w, "  -f, --fail           Fail (non-zero exit) on HTTP errors (>= 400)")
	fmt.Fprintln(w, "  And many more...")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  gocurl https://api.example.com/data")
	fmt.Fprintln(w, "  gocurl -v https://api.example.com/data")
	fmt.Fprintln(w, "  gocurl -X POST -d 'key=value' https://api.example.com/data")
	fmt.Fprintln(w, "  gocurl -H 'Authorization: Bearer $TOKEN' https://api.example.com/data")
	fmt.Fprintln(w, "  gocurl -o response.json https://api.example.com/data")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Environment variables:")
	fmt.Fprintln(w, "  Use $VAR or ${VAR} syntax - automatically expanded from environment")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Multi-line commands:")
	fmt.Fprintln(w, "  Backslash (\\) for line continuation")
	fmt.Fprintln(w, "  # for comments")
	fmt.Fprintln(w, "  'curl' prefix optional")
}
