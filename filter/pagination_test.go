package filter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseQueryPagination(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string][]string
		expected Pagination
		wantErr  bool
	}{
		{
			name:  "default values",
			query: map[string][]string{},
			expected: Pagination{
				Start:    0,
				End:      10,
				PageSize: 10,
				Mode:     "server",
			},
		},
		{
			name: "custom start and end",
			query: map[string][]string{
				"_start": {"20"},
				"_end":   {"30"},
			},
			expected: Pagination{
				Start:    20,
				End:      30,
				PageSize: 10,
				Mode:     "server",
			},
		},
		{
			name: "invalid start parameter",
			query: map[string][]string{
				"_start": {"invalid"},
			},
			wantErr: true,
		},
		{
			name: "negative start",
			query: map[string][]string{
				"_start": {"-1"},
			},
			wantErr: true,
		},
		{
			name: "end less than start",
			query: map[string][]string{
				"_start": {"10"},
				"_end":   {"5"},
			},
			wantErr: true,
		},
		{
			name: "page size too large",
			query: map[string][]string{
				"_start": {"0"},
				"_end":   {"101"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQueryPagination(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatContentRange(t *testing.T) {
	pagination := Pagination{
		Start:    10,
		End:      20,
		PageSize: 10,
		Mode:     "server",
	}

	result := FormatContentRange("users", pagination, 5, 100)

	assert.Equal(t, "users 10-14/100", result)
}

func TestGetResultCount(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected int
	}{
		{
			name:     "slice with items",
			data:     []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "empty slice",
			data:     []int{},
			expected: 0,
		},
		{
			name:     "non-slice",
			data:     "not a slice",
			expected: 0,
		},
		{
			name:     "nil",
			data:     nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := GetResultCount(tt.data)
			assert.Equal(t, tt.expected, count)
		})
	}
}
