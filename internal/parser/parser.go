// internal/parser/parser.go

package parser

import (
	"fmt"
	"strings"
)

type Parser struct {
	tokenizer *Tokenizer
}

func NewParser() *Parser {
	return &Parser{
		tokenizer: NewTokenizer(),
	}
}

func (p *Parser) Parse(command string, variables Variables) (map[string]string, error) {
	err := p.tokenizer.Tokenize(command)
	if err != nil {
		return nil, err
	}

	tokens := p.tokenizer.GetTokens()
	result := make(map[string]string)
	var currentFlag string
	var currentValue strings.Builder

	for i, token := range tokens {
		if i == 0 && token.Value == "curl" {
			continue // Skip the 'curl' command itself
		}

		switch token.Type {
		case TokenFlag:
			if currentFlag != "" {
				result[currentFlag] = currentValue.String()
				currentValue.Reset()
			}
			currentFlag = token.Value
		case TokenValue:
			currentValue.WriteString(token.Value)
		case TokenVariable:
			value, ok := variables[token.Value]
			if !ok {
				return nil, fmt.Errorf("undefined variable: %s", token.Value)
			}
			currentValue.WriteString(value)
		}
	}

	if currentFlag != "" {
		result[currentFlag] = currentValue.String()
	} else if currentValue.Len() > 0 {
		result["URL"] = currentValue.String()
	}

	// Clean up values
	for k, v := range result {
		result[k] = strings.Trim(v, "'\"")
	}

	return result, nil
}
