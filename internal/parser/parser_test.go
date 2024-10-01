// internal/parser/parser_test.go

package parser

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrial(t *testing.T) {
	opt, err := ParseCurlCommand("curl -X POST -d '{\"key\":\"value\"}' https://api.example.com/data")
	assert.NoError(t, err)

	assert.Equal(t, "POST", opt.Method)
	assert.Equal(t, "https://api.example.com/data", opt.URL)
	assert.Equal(t, "{\"key\":\"value\"}", opt.Body)

}

func TestParseCurlCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		variables Variables
		want      *RequestOptions
		wantErr   bool
	}{
		{
			name:    "Simple GET request",
			command: "curl https://api.example.com/data",
			want: &RequestOptions{
				Method:          "GET",
				URL:             "https://api.example.com/data",
				Headers:         make(http.Header),
				QueryParams:     make(url.Values),
				Form:            make(url.Values),
				FollowRedirects: true,
				MaxRedirects:    10,
				Compress:        true,
			},
		},
		{
			name:    "POST request with data and variables",
			command: "curl -X POST -d '$POST_DATA' $API_URL/data",
			variables: Variables{
				"POST_DATA": "{\"key\":\"value\"}",
				"API_URL":   "https://api.example.com",
			},
			want: &RequestOptions{
				Method:          "POST",
				URL:             "https://api.example.com/data",
				Headers:         make(http.Header),
				QueryParams:     make(url.Values),
				Form:            make(url.Values),
				Body:            "{\"key\":\"value\"}",
				FollowRedirects: true,
				MaxRedirects:    10,
				Compress:        true,
			},
		},
		{
			name:    "Request with headers, variables, and query parameters",
			command: "curl -H 'Content-Type: $CONTENT_TYPE' -H 'Authorization: Bearer $TOKEN' \"$API_URL/data?q=$QUERY&special=$SPECIAL\"",
			variables: Variables{
				"CONTENT_TYPE": "application/json",
				"TOKEN":        "abc123",
				"API_URL":      "https://api.example.com",
				"QUERY":        "test query",
				"SPECIAL":      "!@#$%^&*()",
			},
			want: &RequestOptions{
				Method: "GET",
				URL:    "https://api.example.com/data?q=test query&special=!@#$%^&*()",
				Headers: http.Header{
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer abc123"},
				},
				QueryParams: url.Values{
					"q":       []string{"test query"},
					"special": []string{"!@#$%^&*()"},
				},
				Form:            make(url.Values),
				BearerToken:     "abc123",
				FollowRedirects: true,
				MaxRedirects:    10,
				Compress:        true,
			},
		},
		{
			name:    "Request with all options",
			command: "curl -X POST -H 'Content-Type: application/json' -H 'Authorization: Bearer token123' -d '{\"key\":\"value\"}' -F 'file=@/path/to/file.txt' -u username:password -x http://proxy.example.com:8080 --max-time 30 -L --max-redirs 5 --compressed --cert /path/to/cert.pem --key /path/to/key.pem -k -A 'MyUserAgent/1.0' --http2 https://api.example.com/data?param1=value1&param2=value2",
			want: &RequestOptions{
				Method: "POST",
				URL:    "https://api.example.com/data?param1=value1&param2=value2",
				Headers: http.Header{
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer token123"},
				},
				QueryParams: url.Values{
					"param1": []string{"value1"},
					"param2": []string{"value2"},
				},
				Form: url.Values{
					"file": []string{"@/path/to/file.txt"},
				},
				Body: "{\"key\":\"value\"}",
				BasicAuth: &BasicAuth{
					Username: "username",
					Password: "password",
				},
				BearerToken:     "token123",
				Proxy:           "http://proxy.example.com:8080",
				Timeout:         30 * time.Second,
				FollowRedirects: true,
				MaxRedirects:    5,
				Compress:        true,
				CertFile:        "/path/to/cert.pem",
				KeyFile:         "/path/to/key.pem",
				Insecure:        true,
				UserAgent:       "MyUserAgent/1.0",
				HTTP2:           true,
			},
		},
		{
			name:    "Request with undefined variable",
			command: "curl -H 'Authorization: Bearer $UNDEFINED_TOKEN' https://api.example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurlCommand(tt.command, tt.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCurlCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCurlCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
