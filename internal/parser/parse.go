package parser

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func ParseCurlCommand(command string, vars ...Variables) (*RequestOptions, error) {
	var variables Variables
	if len(vars) > 0 {
		variables = vars[0]
	}

	parser := NewParser()
	parsedOptions, err := parser.Parse(command, variables)
	if err != nil {
		return nil, err
	}

	options := &RequestOptions{
		Method:          "GET", // Default method
		Headers:         make(http.Header),
		QueryParams:     make(url.Values),
		Form:            make(url.Values),
		FollowRedirects: true,
		MaxRedirects:    10,
		Compress:        true,
	}

	for flag, value := range parsedOptions {
		switch flag {
		case "-X", "--request":
			options.Method = value
		case "-H", "--header":
			parts := strings.SplitN(value, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				options.Headers.Add(key, val)
				if strings.ToLower(key) == "authorization" {
					if strings.HasPrefix(strings.ToLower(val), "bearer ") {
						options.BearerToken = strings.TrimPrefix(val, "Bearer ")
					}
				}
			}
		case "-d", "--data", "--data-ascii", "--data-binary":
			options.Body = value
		case "-F", "--form":
			parts := strings.SplitN(value, "=", 2)
			if len(parts) == 2 {
				options.Form.Add(parts[0], parts[1])
			}
		case "-u", "--user":
			parts := strings.SplitN(value, ":", 2)
			if len(parts) == 2 {
				options.BasicAuth = &BasicAuth{
					Username: parts[0],
					Password: parts[1],
				}
			}
		case "-x", "--proxy":
			options.Proxy = value
		case "--max-time":
			if timeout, err := time.ParseDuration(value + "s"); err == nil {
				options.Timeout = timeout
			}
		case "-L", "--location":
			options.FollowRedirects = true
		case "--max-redirs":
			if maxRedirs, err := strconv.Atoi(value); err == nil {
				options.MaxRedirects = maxRedirs
			}
		case "--compressed":
			options.Compress = true
		case "--cert":
			options.CertFile = value
		case "--key":
			options.KeyFile = value
		case "-k", "--insecure":
			options.Insecure = true
		case "-A", "--user-agent":
			options.UserAgent = value
		case "--http2":
			options.HTTP2 = true
		case "URL":
			options.URL = value
			if u, err := url.Parse(value); err == nil {
				options.QueryParams = u.Query()
			}
		}
	}

	return options, nil
}
