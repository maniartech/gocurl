# Exercise 4: Build a CLI Tool

**Difficulty:** ⭐⭐ Intermediate
**Time:** 30-45 minutes
**Concepts:** CLI development, flag parsing, output formatting, user experience

## Objective

Create a command-line tool called `httpcat` that makes HTTP requests and displays responses in a user-friendly format, similar to `httpie` or `curlie`.

## Requirements

### Functional Requirements

1. Accept URL as command-line argument
2. Support HTTP methods (GET, POST, PUT, DELETE, PATCH)
3. Allow custom headers via flags
4. Support JSON request body
5. Format output with syntax highlighting (colors)
6. Display response headers, status, and body
7. Save response to file (optional flag)

### Technical Requirements

1. Use Go's `flag` package or `cobra` for CLI
2. Use `gocurl` for HTTP requests
3. Colorize output using ANSI codes
4. Pretty-print JSON responses
5. Handle errors gracefully with helpful messages
6. Provide --help documentation

## Example Usage

```bash
# Simple GET request
httpcat https://api.github.com/users/octocat

# POST with JSON data
httpcat -X POST -d '{"name":"test"}' https://api.example.com/users

# Custom headers
httpcat -H "Authorization: Bearer token123" https://api.example.com/data

# Save response to file
httpcat -o response.json https://api.example.com/data

# Verbose output (show request headers)
httpcat -v https://api.example.com/data
```

## Expected Output

```
GET https://api.github.com/users/octocat
Status: 200 OK
Time: 245ms

Headers:
  content-type: application/json
  cache-control: public, max-age=60

Response:
{
  "login": "octocat",
  "id": 583231,
  "name": "The Octocat",
  "public_repos": 8,
  "followers": 9762
}
```

## Implementation Guide

### 1. Project Setup

```bash
mkdir exercise4
cd exercise4
touch main.go
go mod init httpcat
go get github.com/maniartech/gocurl
go get github.com/fatih/color  # For colored output
```

### 2. CLI Flags Structure

```go
type Config struct {
    Method      string
    Headers     []string
    Data        string
    OutputFile  string
    Verbose     bool
    NoColor     bool
}

func parseFlags() (*Config, string, error) {
    config := &Config{}

    flag.StringVar(&config.Method, "X", "GET", "HTTP method")
    flag.StringVar(&config.Data, "d", "", "Request body")
    flag.StringVar(&config.OutputFile, "o", "", "Output file")
    flag.BoolVar(&config.Verbose, "v", false, "Verbose output")
    // ... more flags

    flag.Parse()

    // Get URL from remaining args
    args := flag.Args()
    if len(args) == 0 {
        return nil, "", errors.New("URL required")
    }

    return config, args[0], nil
}
```

### 3. Making the Request

```go
func makeRequest(ctx context.Context, url string, config *Config) (*Response, error) {
    args := []string{"-X", config.Method}

    // Add headers
    for _, header := range config.Headers {
        args = append(args, "-H", header)
    }

    // Add data
    if config.Data != "" {
        args = append(args, "-d", config.Data)
    }

    args = append(args, url)

    // Make request
    body, resp, err := gocurl.CurlString(ctx, args...)
    // ...
}
```

### 4. Formatting Output

```go
func formatResponse(resp *http.Response, body string, config *Config) {
    // Print status line
    color.Green("Status: %d %s\n", resp.StatusCode, resp.Status)

    // Print headers
    if config.Verbose {
        color.Cyan("\nHeaders:\n")
        for key, values := range resp.Header {
            fmt.Printf("  %s: %s\n", key, strings.Join(values, ", "))
        }
    }

    // Print body (pretty-print JSON)
    if isJSON(resp) {
        printPrettyJSON(body)
    } else {
        fmt.Println(body)
    }
}
```

## Checklist

- [ ] Parse command-line flags
- [ ] Validate input (URL, method)
- [ ] Build gocurl arguments from flags
- [ ] Make HTTP request with context timeout
- [ ] Handle errors with helpful messages
- [ ] Format response with colors
- [ ] Pretty-print JSON responses
- [ ] Implement --help text
- [ ] Save to file if -o flag provided
- [ ] Add verbose mode

## Bonus Challenges

1. **Follow redirects**: Add -L flag to follow redirects
2. **Authentication shortcuts**: Add -u flag for basic auth
3. **Request timing**: Show detailed timing breakdown
4. **Session support**: Save/restore cookies between requests
5. **Query parameters**: Parse key=value syntax for query params
6. **Form data**: Support -F flag for multipart form uploads
7. **Syntax highlighting**: Colorize JSON keys vs values
8. **Streaming**: Stream large responses instead of buffering
9. **Configuration file**: Load defaults from ~/.httpcatrc
10. **Plugin system**: Allow custom formatters

## Color Scheme Suggestion

```go
// Use github.com/fatih/color package
var (
    colorStatus   = color.New(color.FgGreen, color.Bold)
    colorHeader   = color.New(color.FgCyan)
    colorError    = color.New(color.FgRed)
    colorKey      = color.New(color.FgBlue)
    colorValue    = color.New(color.FgWhite)
)
```

## Testing Your Tool

```bash
# Build the tool
go build -o httpcat

# Test basic GET
./httpcat https://httpbin.org/get

# Test POST
./httpcat -X POST -d '{"test":"data"}' https://httpbin.org/post

# Test headers
./httpcat -H "User-Agent: MyTool/1.0" https://httpbin.org/headers

# Test verbose
./httpcat -v https://httpbin.org/get

# Test file output
./httpcat -o response.json https://httpbin.org/json

# Test error handling
./httpcat https://invalid-url-that-does-not-exist.com
./httpcat not-a-url
```

## Distribution

Once working, create a release:

```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o httpcat-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o httpcat-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o httpcat-windows-amd64.exe

# Create archive
tar -czf httpcat-1.0.0-linux-amd64.tar.gz httpcat-linux-amd64
```

## Example Implementation Hints

<details>
<summary>Hint: Pretty-Print JSON</summary>

```go
func printPrettyJSON(raw string) error {
    var data interface{}
    if err := json.Unmarshal([]byte(raw), &data); err != nil {
        // Not valid JSON, print as-is
        fmt.Println(raw)
        return nil
    }

    pretty, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }

    fmt.Println(string(pretty))
    return nil
}
```
</details>

<details>
<summary>Hint: Multiple Headers</summary>

```go
// Use flag.Var for repeated flags
type arrayFlags []string

func (a *arrayFlags) String() string {
    return strings.Join(*a, ", ")
}

func (a *arrayFlags) Set(value string) error {
    *a = append(*a, value)
    return nil
}

// Usage:
var headers arrayFlags
flag.Var(&headers, "H", "Header (can be used multiple times)")
```
</details>

## Production Features

For a production tool, consider:

1. **Progress bars** for large responses
2. **Request history** saved to file
3. **Auto-completion** for shells
4. **Man page** documentation
5. **Update checker** to notify of new versions
6. **Proxy support** via environment variables
7. **Certificate validation** options
8. **HTTP/2** and HTTP/3 support indicators

## Success Criteria

Your tool should:
- ✅ Make requests successfully
- ✅ Display colored output
- ✅ Handle errors gracefully
- ✅ Pretty-print JSON
- ✅ Support basic flags
- ✅ Provide helpful --help text
- ✅ Work cross-platform

## Next Steps

1. Polish the user experience
2. Add automated tests
3. Create documentation
4. Publish to GitHub
5. Consider adding to Homebrew (macOS) or apt (Linux)

## Solution

The complete solution is available in `solutions/exercise4/`.

---

**Congratulations!** After completing all 4 exercises, you have:
- Built real API clients
- Implemented resilient retry logic
- Created a production CLI tool
- Gained practical GoCurl experience

You're now ready to move on to Chapter 2!
