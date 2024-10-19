package main

import (
	"errors"
	"fmt"
	"sync"
	"unicode"
)

var (
	byteSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 64)
		},
	}
)

// ParseCommand parses a complex shell command into a slice of arguments.
// It handles quotes, escaped characters, line continuations, and nested quotes.
func ParseCommand(command string) ([]string, error) {
	// Step 1: Preprocess the command to handle line continuations
	preprocessed, err := preprocessCommand(command)
	if err != nil {
		return nil, err
	}

	// Step 2: Parse the preprocessed command
	return tokenize(preprocessed)
}

// preprocessCommand removes backslash-newline pairs to handle line continuations.
func preprocessCommand(command string) (string, error) {
	result := byteSlicePool.Get().([]byte)
	defer byteSlicePool.Put(result[:0])

	inBackslash := false

	for i := 0; i < len(command); i++ {
		ch := command[i]
		if ch == '\\' && !inBackslash {
			inBackslash = true
			continue
		}

		if inBackslash {
			if ch == '\n' {
				// Skip the newline after backslash for line continuation
				inBackslash = false
				continue
			}
			result = append(result, '\\')
			inBackslash = false
		}
		result = append(result, ch)
	}

	if inBackslash {
		return "", errors.New("unfinished escape sequence at end of command")
	}

	return string(result), nil
}

// tokenize splits the preprocessed command into arguments.
func tokenize(command string) ([]string, error) {
	args := make([]string, 0, 16)
	current := byteSlicePool.Get().([]byte)
	defer byteSlicePool.Put(current[:0])

	inSingleQuote := false
	inDoubleQuote := false
	escapeNext := false

	for i := 0; i < len(command); i++ {
		ch := command[i]
		switch {
		case escapeNext:
			current = append(current, ch)
			escapeNext = false

		case ch == '\\' && !inSingleQuote:
			escapeNext = true

		case ch == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote

		case ch == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote

		case unicode.IsSpace(rune(ch)) && !inSingleQuote && !inDoubleQuote:
			if len(current) > 0 {
				args = append(args, string(current))
				current = current[:0]
			}

		default:
			current = append(current, ch)
		}
	}

	if escapeNext {
		return nil, errors.New("unfinished escape sequence at end of command")
	}

	if inSingleQuote || inDoubleQuote {
		return nil, errors.New("unclosed quote in command")
	}

	if len(current) > 0 {
		args = append(args, string(current))
	}

	return args, nil
}

func main2() {
	// Example complex multi-line command with various permutations
	command := `
	grep "pattern with spaces" \
	  -e 'another pattern' \
	  --include="*.go" \
	  --exclude='test_*.go' \
	  -r \
	  /path/to/search \
	  | sort \
	  | uniq -c \
	  > results.txt
	`

	args, err := ParseCommand(command)
	if err != nil {
		fmt.Printf("Error parsing command: %v\n", err)
		return
	}

	fmt.Println("Parsed arguments:")
	for i, arg := range args {
		fmt.Printf("[%d]: %s\n", i, arg)
	}
}
