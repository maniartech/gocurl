package tokenizer_test

import (
	"reflect"
	"testing"

	"github.com/maniartech/gocurl/tokenizer"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []tokenizer.Token
	}{
		{
			name:    "Simple GET request",
			command: "curl https://api.example.com/data",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenValue, Value: "https://api.example.com/data"},
			},
		},
		{
			name:    "POST request with data",
			command: "curl -X POST -d '{\"key\":\"value\"}' https://api.example.com/data",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenFlag, Value: "-X"},
				{Type: tokenizer.TokenValue, Value: "POST"},
				{Type: tokenizer.TokenFlag, Value: "-d"},
				{Type: tokenizer.TokenValue, Value: "'{\"key\":\"value\"}'"},
				{Type: tokenizer.TokenValue, Value: "https://api.example.com/data"},
			},
		},
		{
			name:    "Request with headers and variables",
			command: "curl -H 'Content-Type: $CONTENT_TYPE' -H 'Authorization: Bearer $TOKEN' $API_URL/data",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenFlag, Value: "-H"},
				{Type: tokenizer.TokenValue, Value: "'Content-Type: "},
				{Type: tokenizer.TokenVariable, Value: "CONTENT_TYPE"},
				{Type: tokenizer.TokenValue, Value: "'"},
				{Type: tokenizer.TokenFlag, Value: "-H"},
				{Type: tokenizer.TokenValue, Value: "'Authorization: Bearer "},
				{Type: tokenizer.TokenVariable, Value: "TOKEN"},
				{Type: tokenizer.TokenValue, Value: "'"},
				{Type: tokenizer.TokenVariable, Value: "API_URL"},
				{Type: tokenizer.TokenValue, Value: "/data"},
			},
		},
		{
			name:    "Request with escaped quotes and variables",
			command: `curl -H "Authorization: Bearer $TOKEN" -d "{\"key\":\"value with $VARIABLE\"}" $API_URL`,
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenFlag, Value: "-H"},
				{Type: tokenizer.TokenValue, Value: `"Authorization: Bearer `},
				{Type: tokenizer.TokenVariable, Value: "TOKEN"},
				{Type: tokenizer.TokenValue, Value: `"`},
				{Type: tokenizer.TokenFlag, Value: "-d"},
				{Type: tokenizer.TokenValue, Value: `"{\"key\":\"value with `},
				{Type: tokenizer.TokenVariable, Value: "VARIABLE"},
				{Type: tokenizer.TokenValue, Value: `\"}"`},
				{Type: tokenizer.TokenVariable, Value: "API_URL"},
			},
		},
		{
			name:    "Request with multiple variables in one value",
			command: "curl $SCHEME://$HOST:$PORT/$PATH",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenVariable, Value: "SCHEME"},
				{Type: tokenizer.TokenValue, Value: "://"},
				{Type: tokenizer.TokenVariable, Value: "HOST"},
				{Type: tokenizer.TokenValue, Value: ":"},
				{Type: tokenizer.TokenVariable, Value: "PORT"},
				{Type: tokenizer.TokenValue, Value: "/"},
				{Type: tokenizer.TokenVariable, Value: "PATH"},
			},
		},
		{
			name:    "Request with variables using braces",
			command: "curl ${SCHEME}://${HOST}:${PORT}/${PATH}",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenVariable, Value: "SCHEME"},
				{Type: tokenizer.TokenValue, Value: "://"},
				{Type: tokenizer.TokenVariable, Value: "HOST"},
				{Type: tokenizer.TokenValue, Value: ":"},
				{Type: tokenizer.TokenVariable, Value: "PORT"},
				{Type: tokenizer.TokenValue, Value: "/"},
				{Type: tokenizer.TokenVariable, Value: "PATH"},
			},
		},
		{
			name:    "Request with single quotes",
			command: "curl -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' https://api.example.com",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenFlag, Value: "-H"},
				{Type: tokenizer.TokenValue, Value: "'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'"},
				{Type: tokenizer.TokenValue, Value: "https://api.example.com"},
			},
		},
		{
			name:    "Request with escaped characters",
			command: "curl -d '{\"key\":\"value with \\\"quotes\\\" and \\\\backslashes\\\\\"}' https://api.example.com",
			expected: []tokenizer.Token{
				{Type: tokenizer.TokenValue, Value: "curl"},
				{Type: tokenizer.TokenFlag, Value: "-d"},
				{Type: tokenizer.TokenValue, Value: "'{\"key\":\"value with \\\"quotes\\\" and \\\\backslashes\\\\\"}'"},
				{Type: tokenizer.TokenValue, Value: "https://api.example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := tokenizer.NewTokenizer()
			err := tokenizer.Tokenize(tt.command)
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			if !reflect.DeepEqual(tokenizer.GetTokens(), tt.expected) {
				t.Errorf("Tokenize() got = %v, want %v", tokenizer.GetTokens(), tt.expected)
			}
		})
	}
}

func TestTokenizer_TokenizeErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "Unmatched single quote",
			command: "curl -H 'Authorization: Bearer token https://api.example.com",
		},
		{
			name:    "Unmatched double quote",
			command: "curl -H \"Authorization: Bearer token https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := tokenizer.NewTokenizer()
			err := tokenizer.Tokenize(tt.command)
			if err == nil {
				t.Errorf("Tokenize() error = nil, expected an error for unmatched quote")
			}
		})
	}
}
