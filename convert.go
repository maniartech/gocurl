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
	o := options.NewRequestOptions("")

	// Initialize all maps and slices to prevent nil pointer panics
	o.Headers = make(http.Header)
	o.Form = make(url.Values)
	o.QueryParams = make(url.Values)

	// Default method is GET
	o.Method = "GET"

	// Initialize slices for accumulating multiple headers and data fields
	dataFields := []string{}
	formFields := url.Values{}

	// Expand environment variables in tokens
	expandedTokens := []tokenizer.Token{}
	for _, token := range tokens {
		expandedTokens = append(expandedTokens, tokenizer.Token{
			Type:  token.Type,
			Value: expandVariables(token.Value),
		})
	}

	tokenLen := len(expandedTokens)

	i := 0
	for i < tokenLen {
		token := expandedTokens[i]

		if i == 0 && token.Value == "curl" {
			i++
			continue
		}

		// Handle flags
		if strings.HasPrefix(token.Value, "-") {
			flagName := token.Value
			switch flagName {
			case "-X", "--request":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected method after %s", flagName)
				}
				o.Method = expandedTokens[i].Value
			case "-d", "--data", "--data-raw", "--data-binary":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected data after %s", flagName)
				}
				dataFields = append(dataFields, expandedTokens[i].Value)
				if o.Method == "GET" {
					o.Method = "POST" // cURL defaults to POST when data is provided
				}
			case "-H", "--header":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected header after %s", flagName)
				}
				headerLine := expandedTokens[i].Value
				idx := strings.Index(headerLine, ":")
				if idx <= 0 {
					return nil, fmt.Errorf("invalid header format: %s", headerLine)
				}
				key := strings.TrimSpace(headerLine[:idx])
				value := strings.TrimSpace(headerLine[idx+1:])
				o.Headers.Add(key, value)
			case "-F", "--form":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected form data after %s", flagName)
				}
				formData := expandedTokens[i].Value
				idx := strings.Index(formData, "=")
				if idx <= 0 {
					return nil, fmt.Errorf("invalid form data: %s", formData)
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
			case "-u", "--user":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected credentials after %s", flagName)
				}
				creds := expandedTokens[i].Value
				parts := strings.SplitN(creds, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid credentials format: %s", creds)
				}
				o.SetBasicAuth(parts[0], parts[1])
			case "-b", "--cookie":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected cookie data after %s", flagName)
				}
				cookieData := expandedTokens[i].Value
				if strings.Contains(cookieData, "=") {
					// Inline cookies
					cookies := parseCookies(cookieData)
					o.Cookies = append(o.Cookies, cookies...)
				} else {
					// Cookie file
					fileCookies, err := readCookiesFromFile(cookieData)
					if err != nil {
						return nil, fmt.Errorf("error reading cookies from file: %v", err)
					}
					o.Cookies = append(o.Cookies, fileCookies...)
				}
			case "-c", "--cookie-jar":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected cookie jar file after %s", flagName)
				}
				// For simplicity, we won't implement cookie jar file writing here
				// You can set o.CookieJar or handle it as needed
			case "-o", "--output":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected output file after %s", flagName)
				}
				o.OutputFile = expandedTokens[i].Value
			case "--compressed":
				o.Compress = true
			case "-A", "--user-agent":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected user-agent after %s", flagName)
				}
				o.UserAgent = expandedTokens[i].Value
			case "-e", "--referer":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected referer after %s", flagName)
				}
				o.Referer = expandedTokens[i].Value
			case "--cert":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected certificate file after %s", flagName)
				}
				o.CertFile = expandedTokens[i].Value
			case "--key":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected key file after %s", flagName)
				}
				o.KeyFile = expandedTokens[i].Value
			case "--cacert":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected CA certificate file after %s", flagName)
				}
				o.CAFile = expandedTokens[i].Value
			case "--http2":
				o.HTTP2 = true
			case "--http2-only":
				o.HTTP2Only = true
			case "-x", "--proxy":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected proxy after %s", flagName)
				}
				o.Proxy = expandedTokens[i].Value
			case "--max-time":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected time after %s", flagName)
				}
				timeout, err := time.ParseDuration(expandedTokens[i].Value + "s")
				if err != nil {
					return nil, err
				}
				o.Timeout = timeout
			case "-k", "--insecure":
				o.Insecure = true
			case "-L", "--location":
				o.FollowRedirects = true
			case "--max-redirs":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected number after %s", flagName)
				}
				maxRedirs, err := parseInt(expandedTokens[i].Value)
				if err != nil {
					return nil, fmt.Errorf("invalid max redirects: %v", err)
				}
				o.MaxRedirects = maxRedirs
			case "-v", "--verbose":
				o.Verbose = true
			case "-s", "--silent":
				o.Silent = true
			default:
				return nil, fmt.Errorf("unknown flag: %s", flagName)
			}
			i++
		} else {
			// Handle positional arguments (e.g., URL)
			if o.URL == "" && strings.HasPrefix(token.Value, "http") {
				o.URL = token.Value
				i++
			} else {
				// Handle unexpected tokens
				return nil, fmt.Errorf("unexpected token: %s", token.Value)
			}
		}
	}

	// Ensure URL is provided
	if o.URL == "" {
		return nil, fmt.Errorf("no URL provided")
	}

	// Parse the URL and extract query parameters
	parsedURL, err := url.Parse(o.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	o.QueryParams = parsedURL.Query()
	o.URL = parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path

	// Combine data fields if any
	if len(dataFields) > 0 {
		o.Body = strings.Join(dataFields, "&")
		// Set Content-Type header if not already set (curl behavior)
		if o.Headers.Get("Content-Type") == "" {
			o.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	// Set form data if any
	if len(formFields) > 0 {
		o.Form = formFields
	}

	// Handle Compression
	if o.Compress {
		o.Headers.Set("Accept-Encoding", "deflate, gzip")
	}

	// Set User-Agent
	if o.UserAgent != "" {
		o.Headers.Set("User-Agent", o.UserAgent)
	}

	// Set Referer
	if o.Referer != "" {
		o.Headers.Set("Referer", o.Referer)
	}

	// Handle TLS Config if CertFile, KeyFile, or CAFile are provided
	if o.CertFile != "" || o.KeyFile != "" || o.CAFile != "" || o.Insecure {
		tlsConfig, err := createTLSConfig(o)
		if err != nil {
			return nil, fmt.Errorf("error creating TLS config: %v", err)
		}
		o.TLSConfig = tlsConfig
	}

	return o, nil
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
