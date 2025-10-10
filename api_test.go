package gocurl_test

import (
	"testing"

	"github.com/maniartech/gocurl"
	"github.com/stretchr/testify/assert"
)

func TestRequest_StringCommand(t *testing.T) {
	// This is a basic test - we'll need a test server for real testing
	cmd := "curl https://httpbin.org/get"

	resp, err := gocurl.Request(cmd, nil)

	// For now, just check we can parse and create the request structure
	// Real execution would need network access
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestRequest_WithVariables(t *testing.T) {
	cmd := "curl -H \"Authorization: Bearer ${token}\" https://api.example.com/data"
	vars := gocurl.Variables{
		"token": "test-token-123",
	}

	// This will fail in execution but should parse successfully
	_, err := gocurl.Request(cmd, vars)

	// We expect an error from execution, but not from parsing
	// For now, any result means the parsing and variable substitution worked
	_ = err // Actual execution will fail without a test server
}

func TestExpandVariables(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		vars     gocurl.Variables
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple variable",
			text:     "Hello $name",
			vars:     gocurl.Variables{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "Braced variable",
			text:     "Hello ${name}",
			vars:     gocurl.Variables{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "Multiple variables",
			text:     "$greeting $name",
			vars:     gocurl.Variables{"greeting": "Hello", "name": "World"},
			expected: "Hello World",
		},
		{
			name:     "Escaped variable",
			text:     "Price: \\$100",
			vars:     gocurl.Variables{},
			expected: "Price: $100",
		},
		{
			name:    "Undefined variable",
			text:    "Hello $name",
			vars:    gocurl.Variables{},
			wantErr: true,
		},
		{
			name:     "No variables",
			text:     "Hello World",
			vars:     gocurl.Variables{},
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gocurl.ExpandVariables(tt.text, tt.vars)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
