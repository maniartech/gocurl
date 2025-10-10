package gocurl

import (
	"fmt"
	"strings"
)

// ExpandVariables replaces ${var} or $var with values from map
// Supports escaping: \${var} becomes literal ${var}
func ExpandVariables(text string, vars Variables) (string, error) {
	var result strings.Builder
	result.Grow(len(text))

	i := 0
	for i < len(text) {
		// Check for escape sequence \$
		if i < len(text)-1 && text[i] == '\\' && text[i+1] == '$' {
			result.WriteByte('$')
			i += 2
			continue
		}

		// Check for variable ${VAR} or $VAR
		if text[i] == '$' {
			i++

			// Check for braced variable ${VAR}
			if i < len(text) && text[i] == '{' {
				i++
				start := i

				// Find closing brace
				for i < len(text) && text[i] != '}' {
					i++
				}

				if i >= len(text) {
					return "", fmt.Errorf("unclosed variable: ${%s", text[start:])
				}

				varName := text[start:i]
				i++ // Skip closing brace

				// Look up variable (only error if vars is provided)
				if vars == nil || len(vars) == 0 {
					return "", fmt.Errorf("undefined variable: %s", varName)
				}

				value, ok := vars[varName]
				if !ok {
					return "", fmt.Errorf("undefined variable: %s", varName)
				}

				result.WriteString(value)
				continue
			}

			// Simple variable $VAR
			start := i
			for i < len(text) && (isAlphaNum(text[i]) || text[i] == '_') {
				i++
			}

			if start == i {
				// Just a $ with no variable name
				result.WriteByte('$')
				continue
			}

			varName := text[start:i]

			// Look up variable (only error if vars is provided)
			if vars == nil || len(vars) == 0 {
				return "", fmt.Errorf("undefined variable: %s", varName)
			}

			value, ok := vars[varName]
			if !ok {
				return "", fmt.Errorf("undefined variable: %s", varName)
			}

			result.WriteString(value)
			continue
		}

		// Regular character
		result.WriteByte(text[i])
		i++
	}

	return result.String(), nil
}

// isAlphaNum checks if a byte is alphanumeric
func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
