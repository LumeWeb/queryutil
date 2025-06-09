package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
)

func TestDefaultQueryParamGenerator_Generate(t *testing.T) {
	tests := []struct {
		name     string
		filters  func() []filter.CrudFilter
		expected map[string][]string
		wantErr  bool
	}{
		{
			name: "basic equality",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("name", filter.OpEq, "john"),
				}
			},
			expected: map[string][]string{
				"filters[name]": {"john"},
			},
		},
		{
			name: "not equals operator",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("age", filter.OpNe, 20),
				}
			},
			expected: map[string][]string{
				"filters[age][ne]": {"20"},
			},
		},
		{
			name: "contains operator",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("title", filter.OpContains, "test"),
				}
			},
			expected: map[string][]string{
				"filters[title][contains]": {"test"},
			},
		},
		{
			name: "nested conditional filters",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("name", filter.OpEq, "john"),
						filter.NewLogicalFilter("age", filter.OpGte, 30),
					}),
				}
			},
			expected: map[string][]string{
				"filters[or][0][name]":     {"john"},
				"filters[or][1][age][gte]": {"30"},
			},
		},
		{
			name: "global search",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("q", filter.OpEq, "searchterm"),
				}
			},
			expected: map[string][]string{
				"q": {"searchterm"},
			},
		},
		{
			name: "numeric value types",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("age", filter.OpEq, 25),
					filter.NewLogicalFilter("price", filter.OpGte, 19.99),
				}
			},
			expected: map[string][]string{
				"filters[age]":        {"25"},
				"filters[price][gte]": {"19.99"},
			},
		},
		{
			name: "between operator with explicit float values",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.Between("price", 10.0, 20.0), // Explicit float values
				}
			},
			expected: map[string][]string{
				"filters[price][between]": {"10", "20"}, // Strings are ok here since they'll be parsed as numbers
			},
		},
		{
			name: "multiple values for in operator",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("id", filter.OpIn, []any{1, 2, 3}),
				}
			},
			expected: map[string][]string{
				"filters[id][in]": {"1", "2", "3"},
			},
		},
		{
			name: "null operator",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("deleted_at", filter.OpNull, nil),
				}
			},
			expected: map[string][]string{
				"filters[deleted_at][null]": {""},
			},
		},
		{
			name: "not null operator",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewLogicalFilter("updated_at", filter.OpNnull, nil),
				}
			},
			expected: map[string][]string{
				"filters[updated_at][nnull]": {""},
			},
		},
		{
			name: "complex nested filters",
			filters: func() []filter.CrudFilter {
				return []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("status", filter.OpEq, "active"),
						filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
							filter.NewLogicalFilter("age", filter.OpGte, 18),
							filter.NewLogicalFilter("vip", filter.OpEq, true),
						}),
					}),
				}
			},
			expected: map[string][]string{
				"filters[and][0][status]":          {"active"},
				"filters[and][1][or][0][age][gte]": {"18"},
				"filters[and][1][or][1][vip]":      {"true"},
			},
		},
		{
			name: "invalid between operator",
			filters: func() (filters []filter.CrudFilter) {
				filters = []filter.CrudFilter{
					filter.NewLogicalFilter("price", filter.OpBetween, []any{10}),
				}
				return filters
			},
			wantErr: true,
		},
	}

	generator := NewDefaultQueryParamGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filters []filter.CrudFilter
			var err error
			if tt.filters != nil {
				defer func() {
					if r := recover(); r != nil {
						if !tt.wantErr {
							assert.Fail(t, "Unexpected panic", "Panic: %v", r)
						}
					}
				}()
				filters = tt.filters()
			}
			result, err := generator.Generate(filters)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
