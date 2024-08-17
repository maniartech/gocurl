// internal/parser/tokenizer_test.go

package parser

import (
	"reflect"
	"testing"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []Token
	}{
		{
			name:    "Simple GET request",
			command: "curl https://api.example.com/data",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenValue, Value: "https://api.example.com/data"},
			},
		},
		{
			name:    "POST request with data",
			command: "curl -X POST -d '{\"key\":\"value\"}' https://api.example.com/data",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenFlag, Value: "-X"},
				{Type: TokenValue, Value: "POST"},
				{Type: TokenFlag, Value: "-d"},
				{Type: TokenValue, Value: "'{\"key\":\"value\"}'"},
				{Type: TokenValue, Value: "https://api.example.com/data"},
			},
		},
		{
			name:    "Request with headers and variables",
			command: "curl -H 'Content-Type: $CONTENT_TYPE' -H 'Authorization: Bearer $TOKEN' $API_URL/data",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenFlag, Value: "-H"},
				{Type: TokenValue, Value: "'Content-Type: "},
				{Type: TokenVariable, Value: "CONTENT_TYPE"},
				{Type: TokenValue, Value: "'"},
				{Type: TokenFlag, Value: "-H"},
				{Type: TokenValue, Value: "'Authorization: Bearer "},
				{Type: TokenVariable, Value: "TOKEN"},
				{Type: TokenValue, Value: "'"},
				{Type: TokenVariable, Value: "API_URL"},
				{Type: TokenValue, Value: "/data"},
			},
		},
		{
			name:    "Request with escaped quotes and variables",
			command: `curl -H "Authorization: Bearer $TOKEN" -d "{\"key\":\"value with $VARIABLE\"}" $API_URL`,
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenFlag, Value: "-H"},
				{Type: TokenValue, Value: `"Authorization: Bearer `},
				{Type: TokenVariable, Value: "TOKEN"},
				{Type: TokenValue, Value: `"`},
				{Type: TokenFlag, Value: "-d"},
				{Type: TokenValue, Value: `"{\"key\":\"value with `},
				{Type: TokenVariable, Value: "VARIABLE"},
				{Type: TokenValue, Value: `\"}"`},
				{Type: TokenVariable, Value: "API_URL"},
			},
		},
		{
			name:    "Request with multiple variables in one value",
			command: "curl $SCHEME://$HOST:$PORT/$PATH",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenVariable, Value: "SCHEME"},
				{Type: TokenValue, Value: "://"},
				{Type: TokenVariable, Value: "HOST"},
				{Type: TokenValue, Value: ":"},
				{Type: TokenVariable, Value: "PORT"},
				{Type: TokenValue, Value: "/"},
				{Type: TokenVariable, Value: "PATH"},
			},
		},
		{
			name:    "Request with variables using braces",
			command: "curl ${SCHEME}://${HOST}:${PORT}/${PATH}",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenVariable, Value: "SCHEME"},
				{Type: TokenValue, Value: "://"},
				{Type: TokenVariable, Value: "HOST"},
				{Type: TokenValue, Value: ":"},
				{Type: TokenVariable, Value: "PORT"},
				{Type: TokenValue, Value: "/"},
				{Type: TokenVariable, Value: "PATH"},
			},
		},
		{
			name:    "Request with single quotes",
			command: "curl -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' https://api.example.com",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenFlag, Value: "-H"},
				{Type: TokenValue, Value: "'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'"},
				{Type: TokenValue, Value: "https://api.example.com"},
			},
		},
		{
			name:    "Request with escaped characters",
			command: "curl -d '{\"key\":\"value with \\\"quotes\\\" and \\\\backslashes\\\\\"}' https://api.example.com",
			expected: []Token{
				{Type: TokenValue, Value: "curl"},
				{Type: TokenFlag, Value: "-d"},
				{Type: TokenValue, Value: "'{\"key\":\"value with \\\"quotes\\\" and \\\\backslashes\\\\\"}'"},
				{Type: TokenValue, Value: "https://api.example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer()
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
			tokenizer := NewTokenizer()
			err := tokenizer.Tokenize(tt.command)
			if err == nil {
				t.Errorf("Tokenize() error = nil, expected an error for unmatched quote")
			}
		})
	}
}
