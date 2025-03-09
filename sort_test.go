package queryutil

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestParseQuerySort(t *testing.T) {
	config := &SortConfig{
		SortableFields: []string{"name", "age", "email"},
	}

	tests := []struct {
		name     string
		query    map[string][]string
		config   *SortConfig
		expected []Sort
		wantErr  bool
	}{
		{
			name: "single field ascending",
			query: map[string][]string{
				"_sort":  {"name"},
				"_order": {"asc"},
			},
			config: config,
			expected: []Sort{{
				Field: "name",
				Order: OrderAsc,
			}},
			wantErr: false,
		},
		{
			name: "multiple fields",
			query: map[string][]string{
				"_sort":  {"name,age"},
				"_order": {"desc,asc"},
			},
			config: config,
			expected: []Sort{
				{Field: "name", Order: OrderDesc},
				{Field: "age", Order: OrderAsc},
			},
			wantErr: false,
		},
		{
			name: "default order",
			query: map[string][]string{
				"_sort": {"name"},
			},
			config: config,
			expected: []Sort{{
				Field: "name",
				Order: OrderAsc,
			}},
			wantErr: false,
		},
		{
			name: "invalid field",
			query: map[string][]string{
				"_sort": {"invalid_field"},
			},
			config:   config,
			wantErr: true,
		},
		{
			name: "no config validation",
			query: map[string][]string{
				"_sort": {"any_field"},
			},
			config: nil,
			expected: []Sort{{
				Field: "any_field",
				Order: OrderAsc,
			}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQuerySort(tt.query, tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
