package queryutil

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.lumeweb.com/queryutil/filter"
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

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		sorts       []Sort
		pagination  *filter.Pagination
		filters     []CrudFilter
		expectedURL string
		expectError bool
	}{
		{
			name:        "base URL only",
			baseURL:     "https://api.example.com/users",
			sorts:       nil,
			pagination:  nil,
			filters:     nil,
			expectedURL: "https://api.example.com/users?_end=10&_start=0",
			expectError: false,
		},
		{
			name:    "with sorts only",
			baseURL: "https://api.example.com/users",
			sorts: []Sort{
				{Field: "name", Order: OrderAsc},
				{Field: "age", Order: OrderDesc},
			},
			pagination:  nil,
			filters:     nil,
			expectedURL: "https://api.example.com/users?_end=10&_order=asc%2Cdesc&_sort=name%2Cage&_start=0",
			expectError: false,
		},
		{
			name:       "with filters only",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: nil,
			filters: []CrudFilter{
				NewLogicalFilter("age", OpGte, 18),
				NewLogicalFilter("status", OpEq, "active"),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bage%5D%5Bgte%5D=18&filters%5Bstatus%5D=active",
			expectError: false,
		},
		{
			name:    "with all components",
			baseURL: "https://api.example.com/users",
			sorts: []Sort{
				{Field: "name", Order: OrderAsc},
			},
			pagination: &filter.Pagination{Start: 10, End: 20, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("age", OpGte, 18),
			},
			expectedURL: "https://api.example.com/users?_end=20&_order=asc&_sort=name&_start=10&filters%5Bage%5D%5Bgte%5D=18",
			expectError: false,
		},
		{
			name:        "custom pagination",
			baseURL:     "https://api.example.com/users",
			sorts:       nil,
			pagination:  &filter.Pagination{Start: 5, End: 15, PageSize: 10, Mode: "server"},
			filters:     nil,
			expectedURL: "https://api.example.com/users?_end=15&_start=5",
			expectError: false,
		},
		{
			name:       "nested conditional filters",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewConditionalFilter(filter.LogicalOr, []CrudFilter{
					NewConditionalFilter(filter.LogicalAnd, []CrudFilter{
						NewLogicalFilter("name", OpEq, "john"),
						NewLogicalFilter("age", OpGte, 30),
					}),
				}),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bor%5D%5B0%5D%5Band%5D%5B0%5D%5Bname%5D=john&filters%5Bor%5D%5B0%5D%5Band%5D%5B1%5D%5Bage%5D%5Bgte%5D=30",
			expectError: false,
		},
		{
			name:       "complex nested filters",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewConditionalFilter(filter.LogicalOr, []CrudFilter{
					NewConditionalFilter(filter.LogicalAnd, []CrudFilter{
						NewLogicalFilter("status", OpEq, "active"),
						NewLogicalFilter("age", OpGte, 18),
					}),
					NewConditionalFilter(filter.LogicalNot, []CrudFilter{
						NewLogicalFilter("name", OpContains, "test"),
					}),
				}),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bor%5D%5B0%5D%5Band%5D%5B0%5D%5Bstatus%5D=active&filters%5Bor%5D%5B0%5D%5Band%5D%5B1%5D%5Bage%5D%5Bgte%5D=18&filters%5Bor%5D%5B1%5D%5Bnot%5D%5B0%5D%5Bname%5D%5Bcontains%5D=test",
			expectError: false,
		},
		{
			name:       "deeply nested conditional filters",
			baseURL:    "https://api.example.com/products",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewConditionalFilter(filter.LogicalAnd, []CrudFilter{
					NewConditionalFilter(filter.LogicalOr, []CrudFilter{
						NewLogicalFilter("category", OpEq, "books"),
						NewLogicalFilter("category", OpEq, "movies"),
					}),
					NewLogicalFilter("price", OpBetween, []any{10, 50}),
					NewConditionalFilter(filter.LogicalOr, []CrudFilter{
						NewConditionalFilter(filter.LogicalAnd, []CrudFilter{
							NewLogicalFilter("type", OpEq, "digital"),
							NewLogicalFilter("stock", OpGte, 5),
						}),
						NewConditionalFilter(filter.LogicalAnd, []CrudFilter{
							NewLogicalFilter("type", OpEq, "physical"),
							NewLogicalFilter("stock", OpGte, 10),
						}),
					}),
				}),
			},
			expectedURL: "https://api.example.com/products?_end=10&_start=0&filters%5Band%5D%5B0%5D%5Bor%5D%5B0%5D%5Bcategory%5D=books&filters%5Band%5D%5B0%5D%5Bor%5D%5B1%5D%5Bcategory%5D=movies&filters%5Band%5D%5B1%5D%5Bprice%5D%5Bbetween%5D%5B0%5D=10&filters%5Band%5D%5B1%5D%5Bprice%5D%5Bbetween%5D%5B1%5D=50&filters%5Band%5D%5B2%5D%5Bor%5D%5B0%5D%5Band%5D%5B0%5D%5Btype%5D=digital&filters%5Band%5D%5B2%5D%5Bor%5D%5B0%5D%5Band%5D%5B1%5D%5Bstock%5D%5Bgte%5D=5&filters%5Band%5D%5B2%5D%5Bor%5D%5B1%5D%5Band%5D%5B0%5D%5Btype%5D=physical&filters%5Band%5D%5B2%5D%5Bor%5D%5B1%5D%5Band%5D%5B1%5D%5Bstock%5D%5Bgte%5D=10",
			expectError: false,
		},
		{
			name:       "boolean value handling",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("active", OpEq, true),
				NewLogicalFilter("vip", OpNe, false),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bactive%5D=true&filters%5Bvip%5D%5Bne%5D=false",
			expectError: false,
		},
		{
			name:       "null and not null operators",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("deleted_at", OpNull, nil),
				NewLogicalFilter("updated_at", OpNnull, nil),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bdeleted_at%5D%5Bnull%5D&filters%5Bupdated_at%5D%5Bnnull%5D",
			expectError: false,
		},
		{
			name:       "between operator with numeric values",
			baseURL:    "https://api.example.com/products",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("price", OpBetween, []any{10, 20}),
			},
			expectedURL: "https://api.example.com/products?_end=10&_start=0&filters%5Bprice%5D%5Bbetween%5D%5B0%5D=10&filters%5Bprice%5D%5Bbetween%5D%5B1%5D=20",
			expectError: false,
		},
		{
			name:       "in operator with multiple values",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("id", OpIn, []any{1, 2, 3}),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bid%5D%5Bin%5D%5B0%5D=1&filters%5Bid%5D%5Bin%5D%5B1%5D=2&filters%5Bid%5D%5Bin%5D%5B2%5D=3",
			expectError: false,
		},
		{
			name:       "in-array operator with mixed values",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("tags", OpIna, []any{"go", 123}),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Btags%5D%5Bina%5D%5B0%5D=go&filters%5Btags%5D%5Bina%5D%5B1%5D=123",
			expectError: false,
		},
		{
			name:       "contains operator",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("title", OpContains, "test"),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Btitle%5D%5Bcontains%5D=test",
			expectError: false,
		},
		{
			name:       "not equals operator",
			baseURL:    "https://api.example.com/users",
			sorts:      nil,
			pagination: &filter.Pagination{Start: 0, End: 10, PageSize: 10, Mode: "server"},
			filters: []CrudFilter{
				NewLogicalFilter("age", OpNe, 20),
			},
			expectedURL: "https://api.example.com/users?_end=10&_start=0&filters%5Bage%5D%5Bne%5D=20",
			expectError: false,
		},
		{
			name:        "base URL with existing query parameters",
			baseURL:     "https://api.example.com/users?foo=bar&baz=qux",
			sorts:       []Sort{{Field: "name", Order: OrderAsc}},
			pagination:  nil,
			filters:     []CrudFilter{NewLogicalFilter("age", OpGte, 18)},
			expectedURL: "https://api.example.com/users?_end=10&_order=asc&_sort=name&_start=0&baz=qux&filters%5Bage%5D%5Bgte%5D=18&foo=bar",
			expectError: false,
		},
		{
			name:        "invalid baseURL",
			baseURL:     "://bad",
			sorts:       nil,
			pagination:  nil,
			filters:     nil,
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildURL(tt.baseURL, tt.sorts, tt.pagination, tt.filters...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, result)
			}
		})
	}
}
