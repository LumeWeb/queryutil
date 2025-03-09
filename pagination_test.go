package queryutil

import (
	"testing"
	"github.com/stretchr/testify/assert"
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
