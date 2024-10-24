package parser_test

import (
	"reflect"
	"testing"

	"github.com/maniartech/gocurl/parser"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []parser.Token
	}{
		{
			name:    "Simple GET request",
			command: "curl https://api.example.com/data",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenValue, Value: "https://api.example.com/data"},
			},
		},
		{
			name:    "POST request with data",
			command: "curl -X POST -d '{\"key\":\"value\"}' https://api.example.com/data",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenFlag, Value: "-X"},
				{Type: parser.TokenValue, Value: "POST"},
				{Type: parser.TokenFlag, Value: "-d"},
				{Type: parser.TokenValue, Value: "'{\"key\":\"value\"}'"},
				{Type: parser.TokenValue, Value: "https://api.example.com/data"},
			},
		},
		{
			name:    "Request with headers and variables",
			command: "curl -H 'Content-Type: $CONTENT_TYPE' -H 'Authorization: Bearer $TOKEN' $API_URL/data",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenFlag, Value: "-H"},
				{Type: parser.TokenValue, Value: "'Content-Type: "},
				{Type: parser.TokenVariable, Value: "CONTENT_TYPE"},
				{Type: parser.TokenValue, Value: "'"},
				{Type: parser.TokenFlag, Value: "-H"},
				{Type: parser.TokenValue, Value: "'Authorization: Bearer "},
				{Type: parser.TokenVariable, Value: "TOKEN"},
				{Type: parser.TokenValue, Value: "'"},
				{Type: parser.TokenVariable, Value: "API_URL"},
				{Type: parser.TokenValue, Value: "/data"},
			},
		},
		{
			name:    "Request with escaped quotes and variables",
			command: `curl -H "Authorization: Bearer $TOKEN" -d "{\"key\":\"value with $VARIABLE\"}" $API_URL`,
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenFlag, Value: "-H"},
				{Type: parser.TokenValue, Value: `"Authorization: Bearer `},
				{Type: parser.TokenVariable, Value: "TOKEN"},
				{Type: parser.TokenValue, Value: `"`},
				{Type: parser.TokenFlag, Value: "-d"},
				{Type: parser.TokenValue, Value: `"{\"key\":\"value with `},
				{Type: parser.TokenVariable, Value: "VARIABLE"},
				{Type: parser.TokenValue, Value: `\"}"`},
				{Type: parser.TokenVariable, Value: "API_URL"},
			},
		},
		{
			name:    "Request with multiple variables in one value",
			command: "curl $SCHEME://$HOST:$PORT/$PATH",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenVariable, Value: "SCHEME"},
				{Type: parser.TokenValue, Value: "://"},
				{Type: parser.TokenVariable, Value: "HOST"},
				{Type: parser.TokenValue, Value: ":"},
				{Type: parser.TokenVariable, Value: "PORT"},
				{Type: parser.TokenValue, Value: "/"},
				{Type: parser.TokenVariable, Value: "PATH"},
			},
		},
		{
			name:    "Request with variables using braces",
			command: "curl ${SCHEME}://${HOST}:${PORT}/${PATH}",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenVariable, Value: "SCHEME"},
				{Type: parser.TokenValue, Value: "://"},
				{Type: parser.TokenVariable, Value: "HOST"},
				{Type: parser.TokenValue, Value: ":"},
				{Type: parser.TokenVariable, Value: "PORT"},
				{Type: parser.TokenValue, Value: "/"},
				{Type: parser.TokenVariable, Value: "PATH"},
			},
		},
		{
			name:    "Request with single quotes",
			command: "curl -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' https://api.example.com",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenFlag, Value: "-H"},
				{Type: parser.TokenValue, Value: "'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'"},
				{Type: parser.TokenValue, Value: "https://api.example.com"},
			},
		},
		{
			name:    "Request with escaped characters",
			command: "curl -d '{\"key\":\"value with \\\"quotes\\\" and \\\\backslashes\\\\\"}' https://api.example.com",
			expected: []parser.Token{
				{Type: parser.TokenValue, Value: "curl"},
				{Type: parser.TokenFlag, Value: "-d"},
				{Type: parser.TokenValue, Value: "'{\"key\":\"value with \\\"quotes\\\" and \\\\backslashes\\\\\"}'"},
				{Type: parser.TokenValue, Value: "https://api.example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := parser.NewTokenizer()
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
			tokenizer := parser.NewTokenizer()
			err := tokenizer.Tokenize(tt.command)
			if err == nil {
				t.Errorf("Tokenize() error = nil, expected an error for unmatched quote")
			}
		})
	}
}
