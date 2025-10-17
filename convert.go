package gocurl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/tokenizer"
)

func ArgsToOptions(args []string) (*options.RequestOptions, error) {
	tokens := []tokenizer.Token{}
	for _, arg := range args {
		tokens = append(tokens, tokenizer.Token{Type: tokenizer.TokenValue, Value: arg})
	}
	return convertTokensToRequestOptions(tokens)
}

// ConvertTokensToRequestOptions converts the tokenized cURL command into options.RequestOptions.
func convertTokensToRequestOptions(tokens []tokenizer.Token) (*options.RequestOptions, error) {
	o := initializeRequestOptions()
	dataFields := []string{}
	formFields := url.Values{}

	// Expand environment variables in tokens
	expandedTokens := expandTokenVariables(tokens)

	// Parse all flags and arguments
	err := parseTokens(expandedTokens, o, &dataFields, &formFields)
	if err != nil {
		return nil, err
	}

	// Finalize the request options
	err = finalizeRequestOptions(o, dataFields, formFields)
	if err != nil {
		return nil, err
	}

	return o, nil
}

// initializeRequestOptions creates and initializes a new RequestOptions
func initializeRequestOptions() *options.RequestOptions {
	o := options.NewRequestOptions("")
	o.Headers = make(http.Header)
	o.Form = make(url.Values)
	o.QueryParams = make(url.Values)
	o.Method = "GET"
	return o
}

// expandTokenVariables expands environment variables in all tokens
func expandTokenVariables(tokens []tokenizer.Token) []tokenizer.Token {
	expandedTokens := make([]tokenizer.Token, 0, len(tokens))
	for _, token := range tokens {
		expandedTokens = append(expandedTokens, tokenizer.Token{
			Type:  token.Type,
			Value: expandVariables(token.Value),
		})
	}
	return expandedTokens
}

// parseTokens processes all tokens and populates the request options
func parseTokens(tokens []tokenizer.Token, o *options.RequestOptions, dataFields *[]string, formFields *url.Values) error {
	tokenLen := len(tokens)
	i := 0

	for i < tokenLen {
		token := tokens[i]

		// Skip leading "curl" command
		if i == 0 && token.Value == "curl" {
			i++
			continue
		}

		// Handle flags vs positional arguments
		if strings.HasPrefix(token.Value, "-") {
			newIdx, err := processFlag(tokens, i, o, dataFields, formFields)
			if err != nil {
				return err
			}
			i = newIdx
		} else {
			newIdx, err := processPositionalArg(tokens, i, o)
			if err != nil {
				return err
			}
			i = newIdx
		}
	}

	return nil
}

// processFlag handles a single flag and its arguments
func processFlag(tokens []tokenizer.Token, i int, o *options.RequestOptions, dataFields *[]string, formFields *url.Values) (int, error) {
	flagName := tokens[i].Value
	tokenLen := len(tokens)

	// Try simple flags first (no arguments)
	if newIdx, handled := processSimpleFlag(flagName, i, o); handled {
		return newIdx, nil
	}

	// Try flags with arguments
	if newIdx, err := processFlagWithArgument(tokens, i, tokenLen, flagName, o, dataFields, formFields); err == nil {
		return newIdx, nil
	} else if err.Error() != "not handled" {
		return 0, err
	}

	return 0, fmt.Errorf("unknown flag: %s", flagName)
}

// processSimpleFlag handles flags that don't require arguments
func processSimpleFlag(flagName string, i int, o *options.RequestOptions) (int, bool) {
	switch flagName {
	case "--compressed":
		o.Compress = true
		return i + 1, true
	case "--http2":
		o.HTTP2 = true
		return i + 1, true
	case "--http2-only":
		o.HTTP2Only = true
		return i + 1, true
	case "-k", "--insecure":
		o.Insecure = true
		return i + 1, true
	case "-L", "--location":
		o.FollowRedirects = true
		return i + 1, true
	case "-v", "--verbose":
		o.Verbose = true
		return i + 1, true
	case "-s", "--silent":
		o.Silent = true
		return i + 1, true
	case "-c", "--cookie-jar":
		// Skip cookie jar (not implemented)
		return i + 2, true
	default:
		return 0, false
	}
}

// processFlagWithArgument handles flags that require arguments
func processFlagWithArgument(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions, dataFields *[]string, formFields *url.Values) (int, error) {
	// Try request/data flags first
	if idx, err := processRequestDataFlags(tokens, i, tokenLen, flagName, o, dataFields); err == nil || err.Error() != "not handled" {
		return idx, err
	}

	// Try header/form/auth flags
	if idx, err := processHeaderFormAuthFlags(tokens, i, tokenLen, flagName, o, formFields); err == nil || err.Error() != "not handled" {
		return idx, err
	}

	// Try TLS/security flags
	if idx, err := processTLSSecurityFlags(tokens, i, tokenLen, flagName, o); err == nil || err.Error() != "not handled" {
		return idx, err
	}

	// Try network/output flags
	if idx, err := processNetworkOutputFlags(tokens, i, tokenLen, flagName, o); err == nil || err.Error() != "not handled" {
		return idx, err
	}

	return 0, fmt.Errorf("not handled")
}

// processRequestDataFlags handles request method and data flags
func processRequestDataFlags(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions, dataFields *[]string) (int, error) {
	switch flagName {
	case "-X", "--request":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.Method = value
			return nil
		})
	case "-d", "--data", "--data-raw", "--data-binary":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			*dataFields = append(*dataFields, value)
			if o.Method == "GET" {
				o.Method = "POST"
			}
			return nil
		})
	default:
		return 0, fmt.Errorf("not handled")
	}
}

// processHeaderFormAuthFlags handles headers, forms, and authentication flags
func processHeaderFormAuthFlags(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions, formFields *url.Values) (int, error) {
	switch flagName {
	case "-H", "--header":
		return processHeaderFlag(tokens, i, tokenLen, flagName, o)
	case "-F", "--form":
		return processFormFlag(tokens, i, tokenLen, flagName, o, formFields)
	case "-u", "--user":
		return processUserFlag(tokens, i, tokenLen, flagName, o)
	case "-b", "--cookie":
		return processCookieFlag(tokens, i, tokenLen, flagName, o)
	case "-A", "--user-agent":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.UserAgent = value
			return nil
		})
	case "-e", "--referer":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.Referer = value
			return nil
		})
	default:
		return 0, fmt.Errorf("not handled")
	}
}

// processTLSSecurityFlags handles TLS certificate and CA flags
func processTLSSecurityFlags(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	switch flagName {
	case "--cert":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.CertFile = value
			return nil
		})
	case "--key":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.KeyFile = value
			return nil
		})
	case "--cacert":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.CAFile = value
			return nil
		})
	case "--tlsv1", "--tlsv1.0":
		o.TLSMinVersion = 0x0301 // tls.VersionTLS10
		return i + 1, nil
	case "--tlsv1.1":
		o.TLSMinVersion = 0x0302 // tls.VersionTLS11
		return i + 1, nil
	case "--tlsv1.2":
		o.TLSMinVersion = 0x0303 // tls.VersionTLS12
		return i + 1, nil
	case "--tlsv1.3":
		o.TLSMinVersion = 0x0304 // tls.VersionTLS13
		return i + 1, nil
	case "--tls-max":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			version, err := ParseTLSVersion(value)
			if err != nil {
				return err
			}
			o.TLSMaxVersion = version
			return nil
		})
	case "--ciphers":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			suites, err := ParseCipherSuites(value)
			if err != nil {
				return err
			}
			o.CipherSuites = suites
			return nil
		})
	case "--tls13-ciphers":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			suites, err := ParseTLS13CipherSuites(value)
			if err != nil {
				return err
			}
			o.TLS13CipherSuites = suites
			return nil
		})
	default:
		return 0, fmt.Errorf("not handled")
	}
}

// processNetworkOutputFlags handles proxy, timeout, redirect, and output flags
func processNetworkOutputFlags(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	switch flagName {
	case "-x", "--proxy":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.Proxy = value
			return nil
		})
	case "--proxy-cert":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.ProxyCert = value
			return nil
		})
	case "--proxy-key":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.ProxyKey = value
			return nil
		})
	case "--proxy-cacert":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.ProxyCACert = value
			return nil
		})
	case "--proxy-insecure":
		o.ProxyInsecure = true
		return i + 1, nil
	case "-o", "--output":
		return processFlagWithArg(tokens, i, tokenLen, flagName, func(value string) error {
			o.OutputFile = value
			return nil
		})
	case "--max-time":
		return processMaxTimeFlag(tokens, i, tokenLen, flagName, o)
	case "--max-redirs":
		return processMaxRedirsFlag(tokens, i, tokenLen, flagName, o)
	default:
		return 0, fmt.Errorf("not handled")
	}
}

// processFlagWithArg handles flags that require a single argument
func processFlagWithArg(tokens []tokenizer.Token, i int, tokenLen int, flagName string, setter func(string) error) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected value after %s", flagName)
	}
	err := setter(tokens[i].Value)
	if err != nil {
		return 0, err
	}
	return i + 1, nil
}

// processHeaderFlag handles -H/--header flag
func processHeaderFlag(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected header after %s", flagName)
	}
	headerLine := tokens[i].Value
	idx := strings.Index(headerLine, ":")
	if idx <= 0 {
		return 0, fmt.Errorf("invalid header format: %s", headerLine)
	}
	key := strings.TrimSpace(headerLine[:idx])
	value := strings.TrimSpace(headerLine[idx+1:])
	o.Headers.Add(key, value)
	return i + 1, nil
}

// processFormFlag handles -F/--form flag
func processFormFlag(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions, formFields *url.Values) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected form data after %s", flagName)
	}
	formData := tokens[i].Value
	idx := strings.Index(formData, "=")
	if idx <= 0 {
		return 0, fmt.Errorf("invalid form data: %s", formData)
	}
	key := formData[:idx]
	value := formData[idx+1:]

	// Check if value starts with '@' indicating a file upload
	if strings.HasPrefix(value, "@") {
		filePath := value[1:]
		fileName := filePath
		if lastSlash := strings.LastIndex(filePath, "/"); lastSlash != -1 {
			fileName = filePath[lastSlash+1:]
		}
		o.FileUpload = &options.FileUpload{
			FieldName: key,
			FileName:  fileName,
			FilePath:  filePath,
		}
	} else {
		formFields.Add(key, value)
	}

	if o.Method == "GET" {
		o.Method = "POST"
	}
	return i + 1, nil
}

// processUserFlag handles -u/--user flag
func processUserFlag(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected credentials after %s", flagName)
	}
	creds := tokens[i].Value
	parts := strings.SplitN(creds, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid credentials format: %s", creds)
	}
	o.SetBasicAuth(parts[0], parts[1])
	return i + 1, nil
}

// processCookieFlag handles -b/--cookie flag
func processCookieFlag(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected cookie data after %s", flagName)
	}
	cookieData := tokens[i].Value
	if strings.Contains(cookieData, "=") {
		// Inline cookies
		cookies := parseCookies(cookieData)
		o.Cookies = append(o.Cookies, cookies...)
	} else {
		// Cookie file
		fileCookies, err := readCookiesFromFile(cookieData)
		if err != nil {
			return 0, fmt.Errorf("error reading cookies from file: %v", err)
		}
		o.Cookies = append(o.Cookies, fileCookies...)
	}
	return i + 1, nil
}

// processMaxTimeFlag handles --max-time flag
func processMaxTimeFlag(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected time after %s", flagName)
	}
	timeout, err := time.ParseDuration(tokens[i].Value + "s")
	if err != nil {
		return 0, err
	}
	o.Timeout = timeout
	return i + 1, nil
}

// processMaxRedirsFlag handles --max-redirs flag
func processMaxRedirsFlag(tokens []tokenizer.Token, i int, tokenLen int, flagName string, o *options.RequestOptions) (int, error) {
	i++
	if i >= tokenLen {
		return 0, fmt.Errorf("expected number after %s", flagName)
	}
	maxRedirs, err := parseInt(tokens[i].Value)
	if err != nil {
		return 0, fmt.Errorf("invalid max redirects: %v", err)
	}
	o.MaxRedirects = maxRedirs
	return i + 1, nil
}

// processPositionalArg handles positional arguments (mainly URL)
func processPositionalArg(tokens []tokenizer.Token, i int, o *options.RequestOptions) (int, error) {
	token := tokens[i]
	if o.URL == "" && strings.HasPrefix(token.Value, "http") {
		o.URL = token.Value
		return i + 1, nil
	}
	return 0, fmt.Errorf("unexpected token: %s", token.Value)
}

// finalizeRequestOptions performs post-processing on request options
func finalizeRequestOptions(o *options.RequestOptions, dataFields []string, formFields url.Values) error {
	// Ensure URL is provided
	if o.URL == "" {
		return fmt.Errorf("no URL provided")
	}

	// Parse URL and extract query parameters
	err := parseAndSetURL(o)
	if err != nil {
		return err
	}

	// Set body data
	if len(dataFields) > 0 {
		o.Body = strings.Join(dataFields, "&")
		if o.Headers.Get("Content-Type") == "" {
			o.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	// Set form data
	if len(formFields) > 0 {
		o.Form = formFields
	}

	// Apply compression headers
	if o.Compress {
		o.Headers.Set("Accept-Encoding", "deflate, gzip")
	}

	// Apply user agent
	if o.UserAgent != "" {
		o.Headers.Set("User-Agent", o.UserAgent)
	}

	// Apply referer
	if o.Referer != "" {
		o.Headers.Set("Referer", o.Referer)
	}

	// Setup TLS config if needed
	if o.CertFile != "" || o.KeyFile != "" || o.CAFile != "" || o.Insecure {
		tlsConfig, err := createTLSConfig(o)
		if err != nil {
			return fmt.Errorf("error creating TLS config: %v", err)
		}
		o.TLSConfig = tlsConfig
	}

	return nil
}

// parseAndSetURL parses the URL and extracts query parameters
func parseAndSetURL(o *options.RequestOptions) error {
	parsedURL, err := url.Parse(o.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}
	o.QueryParams = parsedURL.Query()
	o.URL = parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	return nil
}

// Helper function to expand environment variables within a string
func expandVariables(s string) string {
	return os.ExpandEnv(s)
}

// Helper function to parse cookies from a string
func parseCookies(cookieStr string) []*http.Cookie {
	cookies := []*http.Cookie{}
	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		name := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		cookies = append(cookies, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	return cookies
}

// Helper function to read cookies from a file
func readCookiesFromFile(filename string) ([]*http.Cookie, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return parseCookies(string(content)), nil
}

// Helper function to create TLS configuration
func createTLSConfig(o *options.RequestOptions) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: o.Insecure,
	}

	// Apply TLS version constraints (curl-compatible)
	if o.TLSMinVersion != 0 {
		tlsConfig.MinVersion = o.TLSMinVersion
	}
	if o.TLSMaxVersion != 0 {
		tlsConfig.MaxVersion = o.TLSMaxVersion
	}

	// Apply cipher suites (curl-compatible)
	if len(o.CipherSuites) > 0 {
		tlsConfig.CipherSuites = o.CipherSuites
	}

	// Apply TLS 1.3 cipher suites (Go 1.21+)
	// Note: In older Go versions, TLS 1.3 cipher suites are not configurable
	// and this field will be ignored by the TLS library
	if len(o.TLS13CipherSuites) > 0 {
		// This is a no-op in Go < 1.21, but safe to set
		// In Go 1.21+, this would be: tlsConfig.CipherSuites = append(tlsConfig.CipherSuites, o.TLS13CipherSuites...)
		// For now, we just document that TLS 1.3 ciphers are handled by Go's defaults
	}

	if o.CertFile != "" && o.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if o.CAFile != "" {
		caCert, err := os.ReadFile(o.CAFile)
		if err != nil {
			return nil, err
		}
		caCertPool := tlsConfig.RootCAs
		if caCertPool == nil {
			caCertPool = x509.NewCertPool()
			tlsConfig.RootCAs = caCertPool
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	return tlsConfig, nil
}

// Helper function to parse integer values
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
