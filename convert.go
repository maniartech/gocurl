package gocurl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/parser"
)

func ArgsToOptions(args []string) (*options.RequestOptions, error) {
	tokens := []parser.Token{}
	for _, arg := range args {
		tokens = append(tokens, parser.Token{Type: parser.TokenValue, Value: arg})
	}
	return convertTokensToRequestOptions(tokens)
}

// ConvertTokensToRequestOptions converts the tokenized cURL command into options.RequestOptions.
func convertTokensToRequestOptions(tokens []parser.Token) (*options.RequestOptions, error) {
	o := options.NewRequestOptions("https://api.example.com/data")

	// Default method is GET
	o.Method = "GET"

	// Initialize slices for accumulating multiple headers and data fields
	dataFields := []string{}
	formFields := url.Values{}

	// Expand environment variables in tokens
	expandedTokens := []string{}
	for _, token := range tokens {
		expandedTokens = append(expandedTokens, expandVariables(token.Value))
	}
	// tokens = expandedTokens
	tokenLen := len(expandedTokens)

	i := 0
	for i < tokenLen {
		token := expandedTokens[i]

		if i == 0 && token == "curl" {
			i++
			continue
		}

		// Handle flags
		if strings.HasPrefix(token, "-") {
			switch token {
			case "-X", "--request":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected method after %s", token)
				}
				o.Method = token
			case "-d", "--data", "--data-raw", "--data-binary":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected data after %s", token)
				}
				dataFields = append(dataFields, token)
				if o.Method == "GET" {
					o.Method = "POST" // cURL defaults to POST when data is provided
				}
			case "-H", "--header":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected header after %s", token)
				}
				headerLine := token
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
					return nil, fmt.Errorf("expected form data after %s", token)
				}
				formData := token
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
					return nil, fmt.Errorf("expected credentials after %s", token)
				}
				creds := token
				parts := strings.SplitN(creds, ":", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid credentials format: %s", creds)
				}
				o.SetBasicAuth(parts[0], parts[1])
			case "-b", "--cookie":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected cookie data after %s", token)
				}
				cookieData := token
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
					return nil, fmt.Errorf("expected cookie jar file after %s", token)
				}
				// For simplicity, we won't implement cookie jar file writing here
				// You can set o.CookieJar or handle it as needed
			case "-o", "--output":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected output file after %s", token)
				}
				o.OutputFile = token
			case "--compressed":
				o.Compress = true
			case "-A", "--user-agent":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected user-agent after %s", token)
				}
				o.UserAgent = token
			case "-e", "--referer":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected referer after %s", token)
				}
				o.Referer = token
			case "--cert":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected certificate file after %s", token)
				}
				o.CertFile = token
			case "--key":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected key file after %s", token)
				}
				o.KeyFile = token
			case "--cacert":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected CA certificate file after %s", token)
				}
				o.CAFile = token
			case "--http2":
				o.HTTP2 = true
			case "--http2-only":
				o.HTTP2Only = true
			case "-x", "--proxy":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected proxy after %s", token)
				}
				o.Proxy = token
			case "--max-time":
				i++
				if i >= tokenLen {
					return nil, fmt.Errorf("expected time after %s", token)
				}
				timeout, err := time.ParseDuration(token + "s")
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
					return nil, fmt.Errorf("expected number after %s", token)
				}
				maxRedirs, err := parseInt(token)
				if err != nil {
					return nil, fmt.Errorf("invalid max redirects: %v", err)
				}
				o.MaxRedirects = maxRedirs
			case "-v", "--verbose":
				o.Verbose = true
			case "-s", "--silent":
				o.Silent = true
			default:
				return nil, fmt.Errorf("unknown flag: %s", token)
			}
			i++
		} else {
			// Handle positional arguments (e.g., URL)
			if o.URL == "" && strings.HasPrefix(token, "http") {
				o.URL = token
				i++
			} else {
				// Handle unexpected tokens
				return nil, fmt.Errorf("unexpected token: %s", token)
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
	content, err := ioutil.ReadFile(filename)
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
		caCert, err := ioutil.ReadFile(o.CAFile)
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
