package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
		err      error
	}{
		{
			name:     "simple command",
			command:  "echo hello world",
			expected: []string{"echo", "hello", "world"},
			err:      nil,
		},
		{
			name:     "command with quotes",
			command:  "echo \"hello world\"",
			expected: []string{"echo", "hello world"},
			err:      nil,
		},
		{
			name:     "command with single quotes",
			command:  "echo 'hello world'",
			expected: []string{"echo", "hello world"},
			err:      nil,
		},
		{
			name:     "command with escaped characters",
			command:  "echo hello\\ world",
			expected: []string{"echo", "hello world"},
			err:      nil,
		},
		{
			name:     "multi-line command with line continuation",
			command:  "echo hello \\\nworld",
			expected: []string{"echo", "hello", "world"},
			err:      nil,
		},
		{
			name:     "unclosed single quote",
			command:  "echo 'hello",
			expected: nil,
			err:      errors.New("unclosed quote in command"),
		},
		{
			name:     "unclosed double quote",
			command:  "echo \"hello",
			expected: nil,
			err:      errors.New("unclosed quote in command"),
		},
		{
			name:     "unfinished escape sequence",
			command:  "echo hello\\",
			expected: nil,
			err:      errors.New("unfinished escape sequence at end of command"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCommand(tt.command)
			assert.Equal(t, tt.expected, result, "expected result to match")
			assert.Equal(t, tt.err, err, "expected error to match")
		})
	}
}

func TestPreprocessCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
		err      error
	}{
		{
			name:     "line continuation",
			command:  "echo hello \\\nworld",
			expected: "echo hello world",
			err:      nil,
		},
		{
			name:     "unfinished escape",
			command:  "echo hello\\",
			expected: "",
			err:      errors.New("unfinished escape sequence at end of command"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := preprocessCommand(tt.command)
			assert.Equal(t, tt.expected, result, "expected result to match")
			assert.Equal(t, tt.err, err, "expected error to match")
		})
	}
}

func BenchmarkParseCommand(b *testing.B) {
	command := "echo \"hello world\" -e 'another pattern' --include=\"*.go\" --exclude='test_*.go' -r /path/to/search | sort | uniq -c > results.txt"
	for i := 0; i < b.N; i++ {
		_, _ = ParseCommand(command)
	}
}

func BenchmarkPreprocessCommand(b *testing.B) {
	command := "echo hello \\\nworld"
	for i := 0; i < b.N; i++ {
		_, _ = preprocessCommand(command)
	}
}
