package serializer

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
)

func TestQueryParamSerializer_SerializeFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  []filter.CrudFilter
		expected url.Values
		wantErr  bool
	}{
		{
			name: "basic equality",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpEq, "john"),
			},
			expected: url.Values{
				"filters[name]": {"john"},
			},
			wantErr: false,
		},
		{
			name: "not equals operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpNe, 20),
			},
			expected: url.Values{
				"filters[age][ne]": {"20"},
			},
			wantErr: false,
		},
		{
			name: "contains operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("title", filter.OpContains, "test"),
			},
			expected: url.Values{
				"filters[title][contains]": {"test"},
			},
			wantErr: false,
		},
		{
			name: "nested conditional filters",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("name", filter.OpEq, "john"),
						filter.NewLogicalFilter("age", filter.OpGte, 30),
					}),
				}),
			},
			expected: url.Values{
				"filters[or][0][and][0][name]":     {"john"},
				"filters[or][0][and][1][age][gte]": {"30"},
			},
			wantErr: false,
		},
		{
			name: "global search",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("q", filter.OpEq, "searchterm"),
			},
			expected: url.Values{
				"filters[q]": {"searchterm"},
			},
			wantErr: false,
		},
		{
			name: "numeric value types",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpEq, 25),
				filter.NewLogicalFilter("price", filter.OpGte, 19.99),
			},
			expected: url.Values{
				"filters[age]":        {"25"},
				"filters[price][gte]": {"19.99"},
			},
			wantErr: false,
		},
		{
			name: "boolean value handling",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("active", filter.OpEq, true),
				filter.NewLogicalFilter("vip", filter.OpNe, false),
			},
			expected: url.Values{
				"filters[active]":  {"true"},
				"filters[vip][ne]": {"false"},
			},
			wantErr: false,
		},
		{
			name: "complex nested filters",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("status", filter.OpEq, "active"),
						filter.NewLogicalFilter("age", filter.OpGte, 18),
					}),
					filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
						filter.NewLogicalFilter("name", filter.OpContains, "test"),
					}),
				}),
			},
			expected: url.Values{
				"filters[or][0][and][0][status]":         {"active"},
				"filters[or][0][and][1][age][gte]":       {"18"},
				"filters[or][1][not][0][name][contains]": {"test"},
			},
			wantErr: false,
		},
		{
			name: "deeply nested conditional filters",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("category", filter.OpEq, "books"),
						filter.NewLogicalFilter("category", filter.OpEq, "movies"),
					}),
					filter.NewLogicalFilter("price", filter.OpBetween, []any{10, 50}),
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
							filter.NewLogicalFilter("type", filter.OpEq, "digital"),
							filter.NewLogicalFilter("stock", filter.OpGte, 5),
						}),
						filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
							filter.NewLogicalFilter("type", filter.OpEq, "physical"),
							filter.NewLogicalFilter("stock", filter.OpGte, 10),
						}),
					}),
				}),
			},
			expected: url.Values{
				"filters[and][0][or][0][category]":           {"books"},
				"filters[and][0][or][1][category]":           {"movies"},
				"filters[and][1][price][between][0]":         {"10"},
				"filters[and][1][price][between][1]":         {"50"},
				"filters[and][2][or][0][and][0][type]":       {"digital"},
				"filters[and][2][or][0][and][1][stock][gte]": {"5"},
				"filters[and][2][or][1][and][0][type]":       {"physical"},
				"filters[and][2][or][1][and][1][stock][gte]": {"10"},
			},
			wantErr: false,
		},
		{
			name: "empty string value for eq operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpEq, ""),
			},
			expected: url.Values{
				"filters[name]": {""},
			},
			wantErr: false,
		},
		{
			name: "null operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("deleted_at", filter.OpNull, nil),
			},
			expected: url.Values{
				"filters[deleted_at][null]": {""},
			},
			wantErr: false,
		},
		{
			name: "not null operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("updated_at", filter.OpNnull, nil),
			},
			expected: url.Values{
				"filters[updated_at][nnull]": {""},
			},
			wantErr: false,
		},
		{
			name: "between operator with numeric values",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("price", filter.OpBetween, []any{10, 20}),
			},
			expected: url.Values{
				"filters[price][between][0]": {"10"},
				"filters[price][between][1]": {"20"},
			},
			wantErr: false,
		},
		{
			name: "multiple values for in operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("id", filter.OpIn, []any{1, 2, 3}),
			},
			expected: url.Values{
				"filters[id][in][0]": {"1"},
				"filters[id][in][1]": {"2"},
				"filters[id][in][2]": {"3"},
			},
			wantErr: false,
		},
		{
			name: "in-array operator with mixed values",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("tags", filter.OpIna, []any{"go", 123}),
			},
			expected: url.Values{
				"filters[tags][ina][0]": {"go"},
				"filters[tags][ina][1]": {"123"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewQueryParamSerializer()
			result, err := serializer.SerializeFilters(tt.filters)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Compare url.Values directly (map equality is order-independent)
			assert.Equal(t, tt.expected, result)

			// Interop test: serialize -> re-parse -> compare
			t.Run("interop", func(t *testing.T) {
				// Create parser from serialized result
				queryParser := parser.NewQueryParamParser(result)

				// Parse the filters back
				parsedFilters, err := queryParser.ParseFilters()
				assert.NoError(t, err, "Failed to parse serialized filters")

				// Compare original and parsed filters
				assertFiltersEqual(t, tt.filters, parsedFilters)
			})
		})
	}
}

func TestQueryParamSerializer_SerializeSorts(t *testing.T) {
	tests := []struct {
		name     string
		sorts    []filter.Sort
		expected url.Values
		wantErr  bool
	}{
		{
			name: "single sort ascending",
			sorts: []filter.Sort{
				{Field: "name", Order: filter.OrderAsc},
			},
			expected: url.Values{
				"_sort":  {"name"},
				"_order": {"asc"},
			},
			wantErr: false,
		},
		{
			name: "single sort descending",
			sorts: []filter.Sort{
				{Field: "age", Order: filter.OrderDesc},
			},
			expected: url.Values{
				"_sort":  {"age"},
				"_order": {"desc"},
			},
			wantErr: false,
		},
		{
			name: "multiple sorts",
			sorts: []filter.Sort{
				{Field: "name", Order: filter.OrderAsc},
				{Field: "age", Order: filter.OrderDesc},
				{Field: "created_at", Order: filter.OrderAsc},
			},
			expected: url.Values{
				"_sort":  {"name,age,created_at"},
				"_order": {"asc,desc,asc"},
			},
			wantErr: false,
		},
		{
			name:     "empty sorts",
			sorts:    []filter.Sort{},
			expected: url.Values{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewQueryParamSerializer()
			result, err := serializer.SerializeSorts(tt.sorts)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Interop test: serialize -> re-parse -> compare
			t.Run("interop", func(t *testing.T) {
				// Create parser from serialized result
				queryParser := parser.NewQueryParamParser(result)

				// Parse the sorts back
				parsedSorts, err := queryParser.ParseSorts(nil)
				assert.NoError(t, err, "Failed to parse serialized sorts")

				// Compare original and parsed sorts
				assertSortsEqual(t, tt.sorts, parsedSorts)
			})
		})
	}
}

func TestQueryParamSerializer_SerializePagination(t *testing.T) {
	tests := []struct {
		name       string
		pagination filter.Pagination
		expected   url.Values
		wantErr    bool
	}{
		{
			name: "default pagination",
			pagination: filter.Pagination{
				Start:    0,
				End:      10,
				PageSize: 10,
				Mode:     "server",
			},
			expected: url.Values{
				"_start": {"0"},
				"_end":   {"10"},
			},
			wantErr: false,
		},
		{
			name: "custom pagination",
			pagination: filter.Pagination{
				Start:    20,
				End:      50,
				PageSize: 30,
				Mode:     "server",
			},
			expected: url.Values{
				"_start": {"20"},
				"_end":   {"50"},
			},
			wantErr: false,
		},
		{
			name: "zero start",
			pagination: filter.Pagination{
				Start:    0,
				End:      100,
				PageSize: 100,
				Mode:     "server",
			},
			expected: url.Values{
				"_start": {"0"},
				"_end":   {"100"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := NewQueryParamSerializer()
			result, err := serializer.SerializePagination(tt.pagination)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			// Interop test: serialize -> re-parse -> compare
			t.Run("interop", func(t *testing.T) {
				// Create parser from serialized result
				queryParser := parser.NewQueryParamParser(result)

				// Parse the pagination back
				parsedPagination, err := queryParser.ParsePagination()
				assert.NoError(t, err, "Failed to parse serialized pagination")

				// Compare original and parsed pagination
				assertPaginationEqual(t, tt.pagination, parsedPagination)
			})
		})
	}
}

func TestQueryParamSerializer_Options(t *testing.T) {
	t.Run("custom filter prefix", func(t *testing.T) {
		serializer := NewQueryParamSerializer(WithFilterPrefix("custom"))
		filters := []filter.CrudFilter{
			filter.NewLogicalFilter("name", filter.OpEq, "john"),
		}

		result, err := serializer.SerializeFilters(filters)
		assert.NoError(t, err)

		expected := url.Values{
			"custom[name]": {"john"},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("custom sort prefix", func(t *testing.T) {
		serializer := NewQueryParamSerializer(WithSortPrefix("ordering"))
		sorts := []filter.Sort{
			{Field: "name", Order: filter.OrderAsc},
		}

		result, err := serializer.SerializeSorts(sorts)
		assert.NoError(t, err)

		expected := url.Values{
			"ordering[0]": {"name:asc"},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("both custom prefixes", func(t *testing.T) {
		serializer := NewQueryParamSerializer(
			WithFilterPrefix("f"),
			WithSortPrefix("s"),
		)

		filters := []filter.CrudFilter{
			filter.NewLogicalFilter("name", filter.OpEq, "john"),
		}
		sorts := []filter.Sort{
			{Field: "age", Order: filter.OrderDesc},
		}

		filterResult, err := serializer.SerializeFilters(filters)
		assert.NoError(t, err)

		sortResult, err := serializer.SerializeSorts(sorts)
		assert.NoError(t, err)

		expectedFilters := url.Values{
			"f[name]": {"john"},
		}
		expectedSorts := url.Values{
			"s[0]": {"age:desc"},
		}

		assert.Equal(t, expectedFilters, filterResult)
		assert.Equal(t, expectedSorts, sortResult)
	})
}

// assertFiltersEqual compares two slices of CrudFilter for equality
func assertFiltersEqual(t *testing.T, expected, actual []filter.CrudFilter) {
	assert.Equal(t, len(expected), len(actual), "Filter count mismatch")

	for i, expectedFilter := range expected {
		actualFilter := actual[i]
		assertFilterEqual(t, expectedFilter, actualFilter)
	}
}

// assertFilterEqual compares two individual CrudFilter instances
func assertFilterEqual(t *testing.T, expected, actual filter.CrudFilter) {
	// Check types by checking the concrete types
	expectedType := ""
	actualType := ""

	switch expected.(type) {
	case *filter.LogicalFilter:
		expectedType = "LogicalFilter"
	case *filter.ConditionalFilter:
		expectedType = "ConditionalFilter"
	}

	switch actual.(type) {
	case *filter.LogicalFilter:
		actualType = "LogicalFilter"
	case *filter.ConditionalFilter:
		actualType = "ConditionalFilter"
	}

	assert.Equal(t, expectedType, actualType,
		"Filter type mismatch: expected %T, got %T", expected, actual)

	switch e := expected.(type) {
	case *filter.LogicalFilter:
		a, ok := actual.(*filter.LogicalFilter)
		assert.True(t, ok, "Expected LogicalFilter, got %T", actual)
		if ok {
			assert.Equal(t, e.GetField(), a.GetField(), "Field mismatch")
			assert.Equal(t, e.GetOperator(), a.GetOperator(), "Operator mismatch")
			assertValuesEqual(t, e.GetValue(), a.GetValue(), "Value mismatch")
		}

	case *filter.ConditionalFilter:
		a, ok := actual.(*filter.ConditionalFilter)
		assert.True(t, ok, "Expected ConditionalFilter, got %T", actual)
		if ok {
			assert.Equal(t, e.Operator, a.Operator, "Logical operator mismatch")
			assertFiltersEqual(t, e.Filters, a.Filters)
		}

	default:
		t.Errorf("Unknown filter type: %T", expected)
	}
}

// assertPaginationEqual compares two Pagination structs for equality
func assertPaginationEqual(t *testing.T, expected, actual filter.Pagination) {
	assert.Equal(t, expected.Start, actual.Start, "Pagination Start mismatch")
	assert.Equal(t, expected.End, actual.End, "Pagination End mismatch")
	assert.Equal(t, expected.PageSize, actual.PageSize, "Pagination PageSize mismatch")
	assert.Equal(t, expected.Mode, actual.Mode, "Pagination Mode mismatch")
}

// assertSortsEqual compares two slices of Sort for equality
func assertSortsEqual(t *testing.T, expected, actual []filter.Sort) {
	assert.Equal(t, len(expected), len(actual), "Sort count mismatch")

	for i, expectedSort := range expected {
		actualSort := actual[i]
		assert.Equal(t, expectedSort.Field, actualSort.Field, "Sort field mismatch at index %d", i)
		assert.Equal(t, expectedSort.Order, actualSort.Order, "Sort order mismatch at index %d", i)
	}
}

// assertValuesEqual compares two values, handling special cases like slices
func assertValuesEqual(t *testing.T, expected, actual any, msg string) {
	switch e := expected.(type) {
	case []any:
		a, ok := actual.([]any)
		assert.True(t, ok, "%s: Expected []any, got %T", msg, actual)
		if ok {
			assert.Equal(t, len(e), len(a), "%s: Slice length mismatch", msg)
			for i, ev := range e {
				assertValuesEqual(t, ev, a[i], fmt.Sprintf("%s: Slice element %d mismatch", msg, i))
			}
		}
	default:
		assert.Equal(t, expected, actual, msg)
	}
}
