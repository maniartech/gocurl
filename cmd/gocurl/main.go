package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/maniartech/gocurl"
)

func main() {
	// Exit with proper code
	os.Exit(run())
}

func run() int {
	// Define flags
	var (
		verbose        = flag.Bool("v", false, "Verbose output (show headers and request)")
		verbose2       = flag.Bool("verbose", false, "Verbose output (show headers and request)")
		includeHeader  = flag.Bool("i", false, "Include response headers in output")
		includeHeader2 = flag.Bool("include", false, "Include response headers in output")
		silent         = flag.Bool("s", false, "Silent mode")
		silent2        = flag.Bool("silent", false, "Silent mode")
		outputFile     = flag.String("o", "", "Write output to file")
		outputFile2    = flag.String("output", "", "Write output to file")
		writeOut       = flag.String("w", "", "Custom output format")
		writeOut2      = flag.String("write-out", "", "Custom output format")
	)

	// Custom flag parsing to handle curl-style flags mixed with arguments
	// We need to separate flags from curl args
	args, curlArgs := separateFlags(os.Args[1:])

	// Parse flags
	fs := flag.NewFlagSet("gocurl", flag.ContinueOnError)
	fs.BoolVar(verbose, "v", false, "")
	fs.BoolVar(verbose2, "verbose", false, "")
	fs.BoolVar(includeHeader, "i", false, "")
	fs.BoolVar(includeHeader2, "include", false, "")
	fs.BoolVar(silent, "s", false, "")
	fs.BoolVar(silent2, "silent", false, "")
	fs.StringVar(outputFile, "o", "", "")
	fs.StringVar(outputFile2, "output", "", "")
	fs.StringVar(writeOut, "w", "", "")
	fs.StringVar(writeOut2, "write-out", "", "")

	if err := fs.Parse(args); err != nil {
		if !*silent && !*silent2 {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		}
		return 1
	}

	// If no curl args, show usage
	if len(curlArgs) == 0 {
		printUsage()
		return 1
	}

	ctx := context.Background()

	// Build output options
	opts := OutputOptions{
		Verbose:       *verbose || *verbose2,
		IncludeHeader: *includeHeader || *includeHeader2,
		Silent:        *silent || *silent2,
		OutputFile:    getFirstNonEmpty(*outputFile, *outputFile2),
		WriteOut:      getFirstNonEmpty(*writeOut, *writeOut2),
	}

	// Execute using SHARED code path (ZERO DIVERGENCE!)
	err := executeCLI(ctx, curlArgs, opts)
	if err != nil {
		// Error output to stderr (like curl)
		if !opts.Silent {
			fmt.Fprintf(os.Stderr, "gocurl: %v\n", err)
		}
		return getExitCode(err)
	}

	return 0
}

// separateFlags separates gocurl-specific flags from curl command args
func separateFlags(args []string) (flags []string, curlArgs []string) {
	gocurlFlags := map[string]bool{
		"-v": true, "--verbose": true,
		"-i": true, "--include": true,
		"-s": true, "--silent": true,
		"-o": true, "--output": true,
		"-w": true, "--write-out": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if this is a gocurl flag
		if gocurlFlags[arg] {
			flags = append(flags, arg)

			// Check if this flag takes a value
			if arg == "-o" || arg == "--output" || arg == "-w" || arg == "--write-out" {
				if i+1 < len(args) {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			// This is a curl arg
			curlArgs = append(curlArgs, arg)
		}
	}

	return flags, curlArgs
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
func executeCLI(ctx context.Context, args []string, opts OutputOptions) error {
	// Use the library's CurlArgs function directly
	resp, err := gocurl.CurlArgs(ctx, args...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Format and print output
	return FormatAndPrintResponse(resp, body, opts)
}

// getExitCode returns appropriate exit code based on error
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	// Match curl exit codes where possible
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

func printUsage() {
	fmt.Println("gocurl - Execute curl commands in Go")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gocurl [gocurl options] [curl options] <URL>")
	fmt.Println()
	fmt.Println("GoCurl Options:")
	fmt.Println("  -v, --verbose        Verbose output (show headers and request)")
	fmt.Println("  -i, --include        Include response headers in output")
	fmt.Println("  -s, --silent         Silent mode (no output)")
	fmt.Println("  -o, --output FILE    Write output to file")
	fmt.Println("  -w, --write-out FMT  Custom output format")
	fmt.Println()
	fmt.Println("Curl Options: (All standard curl options supported)")
	fmt.Println("  -X, --request        HTTP method (GET, POST, PUT, DELETE, etc.)")
	fmt.Println("  -H, --header         Add header")
	fmt.Println("  -d, --data           Request body data")
	fmt.Println("  -u, --user           Basic auth credentials")
	fmt.Println("  And many more...")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gocurl https://api.example.com/data")
	fmt.Println("  gocurl -v https://api.example.com/data")
	fmt.Println("  gocurl -X POST -d 'key=value' https://api.example.com/data")
	fmt.Println("  gocurl -H 'Authorization: Bearer $TOKEN' https://api.example.com/data")
	fmt.Println("  gocurl -o response.json https://api.example.com/data")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  Use $VAR or ${VAR} syntax - automatically expanded from environment")
	fmt.Println()
	fmt.Println("Multi-line commands:")
	fmt.Println("  Backslash (\\) for line continuation")
	fmt.Println("  # for comments")
	fmt.Println("  'curl' prefix optional")
}
