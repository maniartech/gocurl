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
			varName, newPos, err := extractVariable(text, i+1)
			if err != nil {
				return "", err
			}

			if varName == "" {
				// Just a $ with no variable name
				result.WriteByte('$')
				i++
				continue
			}

			// Look up variable value
			value, err := lookupVariable(varName, vars)
			if err != nil {
				return "", err
			}

			result.WriteString(value)
			i = newPos
			continue
		}

		// Regular character
		result.WriteByte(text[i])
		i++
	}

	return result.String(), nil
}

// extractVariable extracts variable name from position i in text
// Returns variable name, new position, and error if any
func extractVariable(text string, i int) (string, int, error) {
	// Check for braced variable ${VAR}
	if i < len(text) && text[i] == '{' {
		return extractBracedVariable(text, i+1)
	}

	// Simple variable $VAR
	return extractSimpleVariable(text, i)
}

// extractBracedVariable extracts ${VAR} style variable
func extractBracedVariable(text string, start int) (string, int, error) {
	i := start
	// Find closing brace
	for i < len(text) && text[i] != '}' {
		i++
	}

	if i >= len(text) {
		return "", i, fmt.Errorf("unclosed variable: ${%s", text[start:])
	}

	varName := text[start:i]
	return varName, i + 1, nil // +1 to skip closing brace
}

// extractSimpleVariable extracts $VAR style variable
func extractSimpleVariable(text string, start int) (string, int, error) {
	i := start
	for i < len(text) && (isAlphaNum(text[i]) || text[i] == '_') {
		i++
	}

	if start == i {
		// No variable name found
		return "", start, nil
	}

	varName := text[start:i]
	return varName, i, nil
}

// lookupVariable looks up variable in vars map
func lookupVariable(varName string, vars Variables) (string, error) {
	if vars == nil || len(vars) == 0 {
		return "", fmt.Errorf("undefined variable: %s", varName)
	}

	value, ok := vars[varName]
	if !ok {
		return "", fmt.Errorf("undefined variable: %s", varName)
	}

	return value, nil
}

// isAlphaNum checks if a byte is alphanumeric
func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
