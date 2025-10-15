package gocurl

import (
	"testing"
)

func TestExpandVariablesRefactored(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		vars     Variables
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple variable",
			text:     "Hello $NAME",
			vars:     Variables{"NAME": "World"},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "Braced variable",
			text:     "Hello ${NAME}",
			vars:     Variables{"NAME": "World"},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "Multiple variables",
			text:     "$GREETING $NAME",
			vars:     Variables{"GREETING": "Hello", "NAME": "World"},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "Escaped dollar",
			text:     "Price is \\$100",
			vars:     Variables{},
			expected: "Price is $100",
			wantErr:  false,
		},
		{
			name:     "Undefined variable",
			text:     "Hello $UNDEFINED",
			vars:     Variables{"NAME": "World"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Unclosed braced variable",
			text:     "Hello ${UNCLOSED",
			vars:     Variables{"UNCLOSED": "World"},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Dollar with no variable",
			text:     "Cost is $",
			vars:     Variables{},
			expected: "Cost is $",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandVariables(tt.text, tt.vars)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q but got %q", tt.expected, result)
			}
		})
	}
}
