package queryutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToQueryString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected string
	}{
		{
			name:     "empty map",
			input:    map[string][]string{},
			expected: "",
		},
		{
			name: "single key-value pair",
			input: map[string][]string{
				"name": {"john"},
			},
			expected: "name=john",
		},
		{
			name: "multiple values for same key",
			input: map[string][]string{
				"id": {"1", "2", "3"},
			},
			expected: "id=1&id=2&id=3",
		},
		{
			name: "multiple key-value pairs",
			input: map[string][]string{
				"name":  {"john"},
				"age":   {"30"},
				"email": {"john@example.com"},
			},
			expected: "age=30&email=john%40example.com&name=john",
		},
		{
			name: "empty value",
			input: map[string][]string{
				"flag": {""},
				"name": {"john"},
			},
			expected: "flag&name=john",
		},
		{
			name: "special characters in values",
			input: map[string][]string{
				"query": {"hello world"},
				"email": {"user@example.com"},
			},
			expected: "email=user%40example.com&query=hello+world",
		},
		{
			name: "special characters in keys",
			input: map[string][]string{
				"user name": {"john doe"},
				"age":       {"30"},
			},
			expected: "age=30&user+name=john+doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToQueryString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
