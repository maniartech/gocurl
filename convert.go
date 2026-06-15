package gocurl

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/tokenizer"
)

// ArgsToOptions converts raw curl-style arguments into RequestOptions.
//
// As a convenience for direct callers, it expands environment variables
// ($VAR / ${VAR}) in each argument. The higher-level Curl* entry points perform
// their own (single) expansion step and call convertTokensToRequestOptions
// directly, so expansion is never applied twice.
func ArgsToOptions(args []string) (*options.RequestOptions, error) {
	tokens := make([]tokenizer.Token, 0, len(args))
	for _, arg := range args {
		tokens = append(tokens, tokenizer.Token{Type: tokenizer.TokenValue, Value: expandVariables(arg)})
	}
	return convertTokensToRequestOptions(tokens)
}

// convertTokensToRequestOptions converts already-expanded tokens into
// RequestOptions. It performs NO variable expansion of its own; callers are
// responsible for expanding (env or explicit map) beforehand. This keeps the
// explicit-variables-only paths (CurlWithVars) free of environment leakage.
func convertTokensToRequestOptions(tokens []tokenizer.Token) (*options.RequestOptions, error) {
	st := newParseState()
	if err := parseTokens(tokens, st); err != nil {
		return nil, err
	}
	if err := finalizeRequestOptions(st); err != nil {
		return nil, err
	}
	return st.o, nil
}

// parseState holds the mutable state accumulated while parsing a curl command.
type parseState struct {
	o          *options.RequestOptions
	data       []string   // -d/--data values (joined with &)
	formFields url.Values // -F values
	getMode    bool       // -G: send data as query string
	remoteName bool       // -O: derive output filename from URL
}

func newParseState() *parseState {
	o := options.NewRequestOptions("")
	o.Headers = make(http.Header)
	o.Form = make(url.Values)
	o.Method = "GET"
	return &parseState{o: o, formFields: url.Values{}}
}

// parseTokens processes all tokens and populates the parse state.
func parseTokens(tokens []tokenizer.Token, st *parseState) error {
	i, n := 0, len(tokens)
	for i < n {
		v := tokens[i].Value

		// Skip a leading "curl" command word.
		if i == 0 && v == "curl" {
			i++
			continue
		}

		if len(v) > 1 && strings.HasPrefix(v, "-") {
			next, err := processFlag(tokens, i, st)
			if err != nil {
				return err
			}
			i = next
		} else {
			next, err := processPositionalArg(tokens, i, st)
			if err != nil {
				return err
			}
			i = next
		}
	}
	return nil
}

// processFlag dispatches a single flag (and consumes its argument if any).
func processFlag(tokens []tokenizer.Token, i int, st *parseState) (int, error) {
	flag := tokens[i].Value

	if idx, ok := processSimpleFlag(flag, i, st); ok {
		return idx, nil
	}
	if idx, ok, err := processArgFlag(tokens, i, flag, st); ok {
		return idx, err
	}
	return 0, fmt.Errorf("unknown flag: %s", flag)
}

// processSimpleFlag handles flags that take no argument.
func processSimpleFlag(flag string, i int, st *parseState) (int, bool) {
	o := st.o
	switch flag {
	case "--compressed":
		o.Compress = true
	case "--http2":
		o.HTTP2 = true
	case "--http2-only", "--http2-prior-knowledge":
		o.HTTP2Only = true
	case "-k", "--insecure":
		o.Insecure = true
	case "-L", "--location":
		o.FollowRedirects = true
	case "-f", "--fail":
		o.FailOnError = true
	case "-v", "--verbose":
		o.Verbose = true
	case "-s", "--silent":
		o.Silent = true
	case "-G", "--get":
		st.getMode = true
	case "-I", "--head":
		o.Method = "HEAD"
	case "-O", "--remote-name":
		st.remoteName = true
	case "--proxy-insecure":
		o.ProxyInsecure = true
	default:
		return 0, false
	}
	return i + 1, true
}

// processArgFlag handles flags that take an argument. It returns ok=true if the
// flag was recognized (err may still be non-nil, e.g. a missing value).
func processArgFlag(tokens []tokenizer.Token, i int, flag string, st *parseState) (int, bool, error) {
	if idx, ok, err := processDataFlags(tokens, i, flag, st); ok {
		return idx, ok, err
	}
	if idx, ok, err := processHeaderAuthFlags(tokens, i, flag, st); ok {
		return idx, ok, err
	}
	if idx, ok, err := processTLSFlags(tokens, i, flag, st.o); ok {
		return idx, ok, err
	}
	if idx, ok, err := processNetworkFlags(tokens, i, flag, st.o); ok {
		return idx, ok, err
	}
	return 0, false, nil
}

// nextArg returns the argument following the flag at index i.
func nextArg(tokens []tokenizer.Token, i int, flag string) (string, int, error) {
	if i+1 >= len(tokens) {
		return "", 0, fmt.Errorf("missing value for %s", flag)
	}
	return tokens[i+1].Value, i + 2, nil
}

// processDataFlags handles request method and body data flags.
func processDataFlags(tokens []tokenizer.Token, i int, flag string, st *parseState) (int, bool, error) {
	switch flag {
	case "-X", "--request":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		st.o.Method = v
		return next, true, nil
	case "-d", "--data", "--data-raw", "--data-binary":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		data, err := readDataValue(flag, v)
		if err != nil {
			return 0, true, err
		}
		st.data = append(st.data, data)
		setPostIfDefault(st.o)
		return next, true, nil
	case "--data-urlencode":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		data, err := dataURLEncode(v)
		if err != nil {
			return 0, true, err
		}
		st.data = append(st.data, data)
		setPostIfDefault(st.o)
		return next, true, nil
	case "-T", "--upload-file":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		// Stream the upload from disk (rewindable) rather than buffering it.
		st.o.BodyStream = FileBody(v)
		if st.o.Method == "GET" {
			st.o.Method = "PUT"
		}
		return next, true, nil
	default:
		return 0, false, nil
	}
}

// readDataValue resolves a -d/--data value, reading @file references for all
// forms except --data-raw. For --data, CR/LF are stripped (curl behavior); for
// --data-binary the file content is preserved verbatim.
func readDataValue(flag, value string) (string, error) {
	if flag == "--data-raw" || !strings.HasPrefix(value, "@") {
		return value, nil
	}
	content, err := os.ReadFile(value[1:])
	if err != nil {
		return "", fmt.Errorf("failed to read data file: %w", err)
	}
	s := string(content)
	if flag != "--data-binary" {
		s = strings.NewReplacer("\r", "", "\n", "").Replace(s)
	}
	return s, nil
}

// dataURLEncode implements curl's --data-urlencode semantics.
func dataURLEncode(value string) (string, error) {
	if idx := strings.IndexAny(value, "=@"); idx >= 0 {
		name, sep, rest := value[:idx], value[idx], value[idx+1:]
		content := rest
		if sep == '@' {
			b, err := os.ReadFile(rest)
			if err != nil {
				return "", fmt.Errorf("failed to read data file: %w", err)
			}
			content = string(b)
		}
		enc := url.QueryEscape(content)
		if name == "" {
			return enc, nil
		}
		return name + "=" + enc, nil
	}
	return url.QueryEscape(value), nil
}

// processHeaderAuthFlags handles headers, forms, auth, cookies, and identity.
func processHeaderAuthFlags(tokens []tokenizer.Token, i int, flag string, st *parseState) (int, bool, error) {
	o := st.o
	switch flag {
	case "-H", "--header":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		return next, true, applyHeaderArg(o, v)
	case "-F", "--form":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		return next, true, applyFormArg(o, st.formFields, v)
	case "-u", "--user":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		user, pass, _ := strings.Cut(v, ":")
		o.SetBasicAuth(user, pass)
		return next, true, nil
	case "-b", "--cookie":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		return next, true, applyCookieArg(o, v)
	case "-c", "--cookie-jar":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		o.CookieFile = v
		return next, true, nil
	case "-A", "--user-agent":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		o.UserAgent = v
		return next, true, nil
	case "-e", "--referer":
		v, next, err := nextArg(tokens, i, flag)
		if err != nil {
			return 0, true, err
		}
		o.Referer = v
		return next, true, nil
	default:
		return 0, false, nil
	}
}

func applyHeaderArg(o *options.RequestOptions, headerLine string) error {
	idx := strings.Index(headerLine, ":")
	if idx <= 0 {
		return fmt.Errorf("invalid header format: %s", headerLine)
	}
	key := strings.TrimSpace(headerLine[:idx])
	value := strings.TrimSpace(headerLine[idx+1:])
	o.Headers.Add(key, value)
	return nil
}

func applyFormArg(o *options.RequestOptions, formFields url.Values, formData string) error {
	idx := strings.Index(formData, "=")
	if idx <= 0 {
		return fmt.Errorf("invalid form data: %s", formData)
	}
	key, value := formData[:idx], formData[idx+1:]
	if strings.HasPrefix(value, "@") {
		filePath := value[1:]
		o.FileUpload = &options.FileUpload{
			FieldName: key,
			FileName:  path.Base(filePath),
			FilePath:  filePath,
		}
	} else {
		formFields.Add(key, value)
	}
	if o.Method == "GET" {
		o.Method = "POST"
	}
	return nil
}

func applyCookieArg(o *options.RequestOptions, cookieData string) error {
	if strings.Contains(cookieData, "=") {
		o.Cookies = append(o.Cookies, parseCookies(cookieData)...)
		return nil
	}
	fileCookies, err := readCookiesFromFile(cookieData)
	if err != nil {
		return fmt.Errorf("error reading cookies from file: %v", err)
	}
	o.Cookies = append(o.Cookies, fileCookies...)
	return nil
}

// processTLSFlags handles TLS certificate, version, and cipher flags.
func processTLSFlags(tokens []tokenizer.Token, i int, flag string, o *options.RequestOptions) (int, bool, error) {
	switch flag {
	case "--cert":
		return argSetter(tokens, i, flag, func(v string) error { o.CertFile = v; return nil })
	case "--key":
		return argSetter(tokens, i, flag, func(v string) error { o.KeyFile = v; return nil })
	case "--cacert":
		return argSetter(tokens, i, flag, func(v string) error { o.CAFile = v; return nil })
	case "--tlsv1", "--tlsv1.0":
		o.TLSMinVersion = tls.VersionTLS10
		return i + 1, true, nil
	case "--tlsv1.1":
		o.TLSMinVersion = tls.VersionTLS11
		return i + 1, true, nil
	case "--tlsv1.2":
		o.TLSMinVersion = tls.VersionTLS12
		return i + 1, true, nil
	case "--tlsv1.3":
		o.TLSMinVersion = tls.VersionTLS13
		return i + 1, true, nil
	case "--tls-max":
		return argSetter(tokens, i, flag, func(v string) error {
			version, err := ParseTLSVersion(v)
			if err != nil {
				return err
			}
			o.TLSMaxVersion = version
			return nil
		})
	case "--ciphers":
		return argSetter(tokens, i, flag, func(v string) error {
			suites, err := ParseCipherSuites(v)
			if err != nil {
				return err
			}
			o.CipherSuites = suites
			return nil
		})
	case "--tls13-ciphers":
		return argSetter(tokens, i, flag, func(v string) error {
			suites, err := ParseTLS13CipherSuites(v)
			if err != nil {
				return err
			}
			o.TLS13CipherSuites = suites
			return nil
		})
	default:
		return 0, false, nil
	}
}

// processNetworkFlags handles proxy, timeout, redirect, retry, and output flags.
func processNetworkFlags(tokens []tokenizer.Token, i int, flag string, o *options.RequestOptions) (int, bool, error) {
	switch flag {
	case "-x", "--proxy":
		return argSetter(tokens, i, flag, func(v string) error { o.Proxy = v; return nil })
	case "--proxy-cert":
		return argSetter(tokens, i, flag, func(v string) error { o.ProxyCert = v; return nil })
	case "--proxy-key":
		return argSetter(tokens, i, flag, func(v string) error { o.ProxyKey = v; return nil })
	case "--proxy-cacert":
		return argSetter(tokens, i, flag, func(v string) error { o.ProxyCACert = v; return nil })
	case "--noproxy":
		return argSetter(tokens, i, flag, func(v string) error {
			o.ProxyNoProxy = splitList(v)
			return nil
		})
	case "-o", "--output":
		return argSetter(tokens, i, flag, func(v string) error { o.OutputFile = v; return nil })
	case "--url":
		return argSetter(tokens, i, flag, func(v string) error { o.URL = v; return nil })
	case "--max-time":
		return argSetter(tokens, i, flag, func(v string) error {
			d, err := parseSeconds(v)
			if err != nil {
				return err
			}
			o.Timeout = d
			return nil
		})
	case "--connect-timeout":
		return argSetter(tokens, i, flag, func(v string) error {
			d, err := parseSeconds(v)
			if err != nil {
				return err
			}
			o.ConnectTimeout = d
			return nil
		})
	case "--max-redirs":
		return argSetter(tokens, i, flag, func(v string) error {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("invalid max redirects: %v", err)
			}
			o.MaxRedirects = n
			return nil
		})
	case "--retry":
		return argSetter(tokens, i, flag, func(v string) error {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("invalid retry count: %v", err)
			}
			if o.RetryConfig == nil {
				o.RetryConfig = &options.RetryConfig{}
			}
			o.RetryConfig.MaxRetries = n
			return nil
		})
	default:
		return 0, false, nil
	}
}

// argSetter consumes the next token as the flag's argument and applies setter.
func argSetter(tokens []tokenizer.Token, i int, flag string, setter func(string) error) (int, bool, error) {
	v, next, err := nextArg(tokens, i, flag)
	if err != nil {
		return 0, true, err
	}
	if err := setter(v); err != nil {
		return 0, true, err
	}
	return next, true, nil
}

// processPositionalArg handles positional arguments (the URL).
func processPositionalArg(tokens []tokenizer.Token, i int, st *parseState) (int, error) {
	v := tokens[i].Value
	if st.o.URL == "" {
		st.o.URL = v
		return i + 1, nil
	}
	return 0, fmt.Errorf("unexpected argument: %s", v)
}

// finalizeRequestOptions performs post-processing once all tokens are parsed.
func finalizeRequestOptions(st *parseState) error {
	o := st.o

	if err := normalizeURL(o); err != nil {
		return err
	}

	applyData(st)

	if len(st.formFields) > 0 {
		o.Form = st.formFields
	}

	if st.remoteName && o.OutputFile == "" {
		o.OutputFile = remoteFilename(o.URL)
	}

	// curl follows up to 50 redirects with -L; default to a sane bound when the
	// user enabled following but did not set --max-redirs.
	if o.FollowRedirects && o.MaxRedirects == 0 {
		o.MaxRedirects = 30
	}

	// TLS configuration (certs, CA, version, ciphers, insecure) is built lazily
	// and authoritatively in LoadTLSConfig from these option fields — see
	// security.go. We intentionally do NOT pre-build o.TLSConfig here so there is
	// a single source of truth.

	return nil
}

// applyData routes accumulated -d data either to the query string (-G) or the
// request body.
func applyData(st *parseState) {
	o := st.o
	if len(st.data) == 0 {
		return
	}
	joined := strings.Join(st.data, "&")
	if st.getMode {
		sep := "?"
		if strings.Contains(o.URL, "?") {
			sep = "&"
		}
		o.URL += sep + joined
		o.Method = "GET"
		return
	}
	o.Body = joined
	if o.Headers.Get("Content-Type") == "" {
		o.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

// normalizeURL validates the URL and, when no scheme is present, defaults to
// http:// (matching curl). It preserves userinfo, port, path, query, and
// fragment rather than reconstructing the URL lossily.
func normalizeURL(o *options.RequestOptions) error {
	raw := strings.TrimSpace(o.URL)
	if raw == "" {
		return fmt.Errorf("no URL provided")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL: missing host in %q", o.URL)
	}
	o.URL = u.String()
	return nil
}

// remoteFilename derives an output filename from the URL path for -O.
func remoteFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "curl_response"
	}
	name := path.Base(u.Path)
	if name == "" || name == "/" || name == "." {
		return "curl_response"
	}
	return name
}

func setPostIfDefault(o *options.RequestOptions) {
	if o.Method == "GET" {
		o.Method = "POST"
	}
}

func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseSeconds parses a curl duration expressed in (possibly fractional) seconds.
func parseSeconds(v string) (time.Duration, error) {
	secs, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %v", v, err)
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// expandVariables expands environment variables within a string (used by
// ArgsToOptions only; the Curl* entry points expand before conversion).
func expandVariables(s string) string {
	return os.ExpandEnv(s)
}

// parseCookies parses cookies from a "name=value; name2=value2" string.
func parseCookies(cookieStr string) []*http.Cookie {
	cookies := []*http.Cookie{}
	for _, part := range strings.Split(cookieStr, ";") {
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(part[:idx]),
			Value: strings.TrimSpace(part[idx+1:]),
		})
	}
	return cookies
}

// readCookiesFromFile reads cookies from a file in "name=value;" form.
func readCookiesFromFile(filename string) ([]*http.Cookie, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return parseCookies(string(content)), nil
}
