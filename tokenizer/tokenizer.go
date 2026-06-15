// Package tokenizer splits a curl command string into shell-style tokens,
// respecting single/double quotes and backslash escapes. Surrounding quotes are
// stripped (their job is grouping, not content), and variable references such as
// $VAR / ${VAR} are left intact within a token so callers can expand them in a
// single, well-defined step.
package tokenizer

import (
	"fmt"
	"strings"
)

type TokenType int

const (
	TokenFlag TokenType = iota
	TokenValue
	// TokenVariable is retained for backward compatibility. The tokenizer no
	// longer splits variables into separate tokens; expansion is performed on
	// whole tokens by the caller.
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

// Tokenize splits command into tokens. Each shell word becomes exactly one
// token; surrounding quotes are removed and escapes are resolved.
func (t *Tokenizer) Tokenize(command string) error {
	fields, err := splitRespectQuotes(command)
	if err != nil {
		return err
	}

	for _, f := range fields {
		if !f.quoted && len(f.value) > 1 && strings.HasPrefix(f.value, "-") {
			t.tokens = append(t.tokens, Token{Type: TokenFlag, Value: f.value})
		} else {
			t.tokens = append(t.tokens, Token{Type: TokenValue, Value: f.value})
		}
	}
	return nil
}

// tokField is a single split field plus whether it began inside a quote (so a
// leading '-' is treated as a literal value rather than a flag).
type tokField struct {
	value  string
	quoted bool
}

// splitRespectQuotes splits s on unquoted whitespace, stripping quote
// delimiters and resolving backslash escapes. Single quotes preserve their
// contents literally; double quotes allow backslash escapes.
func splitRespectQuotes(s string) ([]tokField, error) {
	var result []tokField
	var current strings.Builder
	started := false
	startedQuoted := false
	inQuotes := false
	quoteChar := rune(0)
	escaped := false

	flush := func() {
		if started {
			result = append(result, tokField{value: current.String(), quoted: startedQuoted})
			current.Reset()
			started = false
			startedQuoted = false
		}
	}

	for _, char := range s {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case inQuotes && quoteChar == '\'':
			// Single quotes: literal until the closing single quote.
			if char == '\'' {
				inQuotes, quoteChar = false, 0
			} else {
				current.WriteRune(char)
			}
		case char == '\\' && (!inQuotes || quoteChar == '"'):
			// Backslash escapes the next rune outside quotes and in double quotes.
			escaped = true
		case char == '\'' || char == '"':
			if !inQuotes {
				inQuotes, quoteChar = true, char
				if !started {
					startedQuoted = true
				}
				started = true
			} else if char == quoteChar {
				inQuotes, quoteChar = false, 0
			} else {
				current.WriteRune(char)
			}
		case (char == ' ' || char == '\t') && !inQuotes:
			flush()
		default:
			current.WriteRune(char)
			started = true
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unmatched %c quote", quoteChar)
	}
	if escaped {
		// Trailing backslash: keep it literal.
		current.WriteRune('\\')
		started = true
	}
	flush()
	return result, nil
}

func (t *Tokenizer) GetTokens() []Token {
	return t.tokens
}
