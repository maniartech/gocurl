// internal/parser/tokenizer.go

package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type TokenType int

const (
	TokenFlag TokenType = iota
	TokenValue
	TokenVariable
)

type Token struct {
	Type  TokenType
	Value string
}

type Tokenizer struct {
	tokens []Token
}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{}
}

func (t *Tokenizer) Tokenize(command string) error {
	// Updated regex to match variables like $VAR, ${VAR}, $VAR_NAME, ${VAR_NAME}, etc.
	// This pattern allows alphanumeric characters, underscores, and hyphens in variable names
	varRegex := regexp.MustCompile(`\$(\w+)|\$\{(\w+)\}`)

	// Split the command, respecting quotes
	fields, err := splitRespectQuotes(command)
	if err != nil {
		return err
	}

	for _, field := range fields {
		if strings.HasPrefix(field, "-") {
			t.tokens = append(t.tokens, Token{Type: TokenFlag, Value: field})
		} else {
			// Check for variables in the field
			vars := varRegex.FindAllStringSubmatchIndex(field, -1)
			if len(vars) > 0 {
				// If variables are found, split the field and tokenize each part
				lastIndex := 0
				for _, match := range vars {
					if match[0] > lastIndex {
						// Add non-variable part as TokenValue
						t.tokens = append(t.tokens, Token{Type: TokenValue, Value: field[lastIndex:match[0]]})
					}
					// Add variable as TokenVariable
					var varName string
					if match[2] != -1 {
						varName = field[match[2]:match[3]]
					} else {
						varName = field[match[4]:match[5]]
					}
					t.tokens = append(t.tokens, Token{Type: TokenVariable, Value: varName})
					lastIndex = match[1]
				}
				if lastIndex < len(field) {
					// Add remaining part as TokenValue
					t.tokens = append(t.tokens, Token{Type: TokenValue, Value: field[lastIndex:]})
				}
			} else {
				// If no variables, add entire field as TokenValue
				t.tokens = append(t.tokens, Token{Type: TokenValue, Value: field})
			}
		}
	}
	return nil
}

// Helper function to split the command respecting quotes
func splitRespectQuotes(s string) ([]string, error) {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	escaped := false

	for i, char := range s {
		if escaped {
			current.WriteRune(char)
			escaped = false
		} else if char == '\\' {
			current.WriteRune(char)
			escaped = true
		} else if char == '\'' || char == '"' {
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = rune(0)
			}
			current.WriteRune(char)
		} else if char == ' ' && !inQuotes {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(char)
		}

		// Check for unmatched quote at the end of the string
		if i == len(s)-1 && inQuotes {
			return nil, fmt.Errorf("unmatched %c quote", quoteChar)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result, nil
}

func (t *Tokenizer) GetTokens() []Token {
	return t.tokens
}
