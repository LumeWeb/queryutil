package parser

import (
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
)

func TestQueryParamParser_ParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string][]string
		expected []filter.CrudFilter // This will now hold filters created via constructors
		wantErr  bool
	}{
		{
			name: "basic equality",
			query: map[string][]string{
				"filters[name]": {"john"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpEq, "john"),
			},
			wantErr: false,
		},
		{
			name: "not equals operator",
			query: map[string][]string{
				"filters[age][ne]": {"20"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpNe, 20),
			},
			wantErr: false,
		},
		{
			name: "contains operator",
			query: map[string][]string{
				"filters[title][contains]": {"test"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("title", filter.OpContains, "test"),
			},
			wantErr: false,
		},
		{
			name: "nested conditional filters",
			query: map[string][]string{
				"filters[or][0][and][0][name][eq]": {"john"},
				"filters[or][0][and][1][age][gte]": {"30"},
			},
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("name", filter.OpEq, "john"),
						filter.NewLogicalFilter("age", filter.OpGte, 30),
					}),
				}),
			},
			wantErr: false,
		},
		{
			name: "global search",
			query: map[string][]string{
				"filters[q]": {"searchterm"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("q", filter.OpEq, "searchterm"),
			},
			wantErr: false,
		},
		{
			name: "unsupported operator",
			query: map[string][]string{
				"filters[status][invalid]": {"value"},
			},
			wantErr:  true,
			expected: nil, // No expected filters on error
		},
		{
			name: "numeric value types",
			query: map[string][]string{
				"filters[age][eq]":    {"25"},
				"filters[price][gte]": {"19.99"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpEq, 25),
				filter.NewLogicalFilter("price", filter.OpGte, 19.99),
			},
			wantErr: false,
		},
		{
			name: "boolean value handling",
			query: map[string][]string{
				"filters[active][eq]": {"true"},
				"filters[vip][neq]":   {"false"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("active", filter.OpEq, true),
				filter.NewLogicalFilter("vip", filter.OpNe, false),
			},
			wantErr: false,
		},
		{
			name: "complex nested filters",
			query: map[string][]string{
				"filters[or][0][and][0][status][eq]":     {"active"},
				"filters[or][0][and][1][age][gte]":       {"18"},
				"filters[or][1][not][0][name][contains]": {"test"},
			},
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("status", filter.OpEq, "active"),
						filter.NewLogicalFilter("age", filter.OpGte, 18), // Expected as number
					}),
					filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
						filter.NewLogicalFilter("name", filter.OpContains, "test"),
					}),
				}),
			},
			wantErr: false,
		},
		{
			name: "deeply nested conditional filters",
			query: map[string][]string{
				"filters[and][0][or][0][category][eq]":       {"books"},
				"filters[and][0][or][1][category][eq]":       {"movies"},
				"filters[and][1][price][between][0]":         {"10"},
				"filters[and][1][price][between][1]":         {"50"},
				"filters[and][2][or][0][and][0][type][eq]":   {"digital"},
				"filters[and][2][or][0][and][1][stock][gte]": {"5"},
				"filters[and][2][or][1][and][0][type][eq]":   {"physical"},
				"filters[and][2][or][1][and][1][stock][gte]": {"10"},
			},
			expected: []filter.CrudFilter{
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
			wantErr: false,
		},
		{
			name: "multiple nested conditionals with array indices",
			query: map[string][]string{
				"filters[and][0][or][0][author][eq]":     {"john"},
				"filters[and][0][or][1][author][eq]":     {"jane"},
				"filters[and][1][or][0][year][gte]":      {"2020"},
				"filters[and][1][or][1][year][lte]":      {"1990"},
				"filters[and][2][and][0][available][eq]": {"true"},
				"filters[and][2][and][1][rating][gte]":   {"4.5"},
			},
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("author", filter.OpEq, "john"),
						filter.NewLogicalFilter("author", filter.OpEq, "jane"),
					}),
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("year", filter.OpGte, 2020),
						filter.NewLogicalFilter("year", filter.OpLte, 1990),
					}),
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("available", filter.OpEq, true),
						filter.NewLogicalFilter("rating", filter.OpGte, 4.5),
					}),
				}),
			},
			wantErr: false,
		},
		{
			name: "empty string value for eq operator",
			query: map[string][]string{
				"filters[name][eq]": {""},
			},
			wantErr: false,
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpEq, ""),
			},
		},
		{
			name: "invalid array index",
			query: map[string][]string{
				"filters[or][invalid][age][eq]": {"30"},
			},
			wantErr:  true,
			expected: nil, // No expected filters on error
		},
		{
			name: "null operator",
			query: map[string][]string{
				"filters[deleted_at][null]": {""},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("deleted_at", filter.OpNull, nil),
			},
			wantErr: false,
		},
		{
			name: "not null operator",
			query: map[string][]string{
				"filters[updated_at][nnull]": {""},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("updated_at", filter.OpNnull, nil),
			},
			wantErr: false,
		},
		{
			name: "between operator with numeric values",
			query: map[string][]string{
				"filters[price][between]": {"10", "20"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("price", filter.OpBetween, []any{10, 20}),
			},
			wantErr: false,
		},
		{
			name: "multiple values for in operator",
			query: map[string][]string{
				"filters[id][in]": {"1", "2", "3"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("id", filter.OpIn, []any{1, 2, 3}), // EXPECT NUMBERS
			},
			wantErr: false,
		},
		{
			name: "invalid boolean value",
			query: map[string][]string{
				"filters[active][eq]": {"notaboolean"},
			},
			wantErr: false, // Should not error, treats as string if not standard bool/number
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("active", filter.OpEq, "notaboolean"),
			},
		},
		{
			name: "case-sensitive operator name",
			query: map[string][]string{
				"filters[name][EQ]": {"john"},
			},
			wantErr:  true,
			expected: nil, // No expected filters on error
		},
		{
			name: "empty filter key",
			query: map[string][]string{
				"filters[][eq]": {"value"},
			},
			wantErr:  true, // Should error on empty key
			expected: nil,  // No expected filters on error
		},
		{
			name: "in-array operator with mixed values",
			query: map[string][]string{
				"filters[tags][ina]": {"go", "123"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("tags", filter.OpIna, []any{"go", 123}), // Expect type conversion
			},
			wantErr: false,
		},
		{
			name: "reject object-based query with multiple operators",
			query: map[string][]string{
				"filters[age][eq]": {"25"},
				"filters[age][gt]": {"20"},
			},
			wantErr:  true, // Should error on ambiguous structure
			expected: nil,  // No expected filters on error
		},
		{
			name: "reject nested object structure",
			query: map[string][]string{
				"filters[user][name][eq]": {"john"},
				"filters[user][age][gt]":  {"30"},
			},
			wantErr:  true, // Should error on non-flat field/operator structure
			expected: nil,  // No expected filters on error
		},
		{
			name: "mixed conditional operators",
			query: map[string][]string{
				"filters[or][0][and][0][status][eq]": {"active"},
				"filters[or][0][and][1][price][lt]":  {"100"},
				"filters[or][1][name][contains]":     {"special"},
			},
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
						filter.NewLogicalFilter("status", filter.OpEq, "active"),
						filter.NewLogicalFilter("price", filter.OpLt, 100),
					}),
					filter.NewLogicalFilter("name", filter.OpContains, "special"),
				}),
			},
			wantErr: false,
		},
		{
			name: "empty array value for in operator",
			query: map[string][]string{
				"filters[id][in]": {},
			},
			wantErr:  true, // Should error if 'in' has no values
			expected: nil,  // No expected filters on error
		},
		{
			name: "string with commas for in operator",
			query: map[string][]string{
				"filters[id][in]": {"1,2,3"},
			},
			// Assuming comma split and type conversion for 'in' operator
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("id", filter.OpIn, []any{1, 2, 3}),
			},
			wantErr: false,
		},
		{
			name: "zero value handling",
			query: map[string][]string{
				"filters[age][eq]": {"0"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpEq, 0), // Expected as number
			},
			wantErr: false,
		},
		{
			name: "multiple top-level filters",
			query: map[string][]string{
				"filters[name][contains]": {"test"},
				"filters[status][eq]":     {"active"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpContains, "test"),
				filter.NewLogicalFilter("status", filter.OpEq, "active"),
			},
			wantErr: false,
		},
		{
			name: "mixed type array values",
			query: map[string][]string{
				"filters[id][in]": {"1", "abc"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("id", filter.OpIn, []any{1, "abc"}), // Mixed types expected
			},
			wantErr: false,
		},
		{
			name: "invalid boolean casing",
			query: map[string][]string{
				"filters[active][eq]": {"True"},
			},
			wantErr:  true, // Expect error if only lowercase 'true'/'false' are supported
			expected: nil,  // No expected filters on error
		},
		{
			name: "non-numeric between values",
			query: map[string][]string{
				"filters[price][between]": {"ten", "twenty"},
			},
			wantErr:  true, // Expect error if conversion to number fails
			expected: nil,  // No expected filters on error
		},
		{
			name: "nnull operator with value",
			query: map[string][]string{
				"filters[updated_at][nnull]": {"123"},
			},
			wantErr:  true, // Expect error if nnull has a value
			expected: nil,  // No expected filters on error
		},
		{
			name: "single value in array operator",
			query: map[string][]string{
				"filters[category][in]": {"shoes"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("category", filter.OpIn, []any{"shoes"}),
			},
			wantErr: false,
		},
		{
			name: "multiple values for equality operator",
			query: map[string][]string{
				"filters[age][eq]": {"30", "31"},
			},
			wantErr:  true, // Expect error for multiple values for non-array operators
			expected: nil,  // No expected filters on error
		},
		{
			name: "decimal value handling",
			query: map[string][]string{
				"filters[rating][eq]": {"4.75"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("rating", filter.OpEq, 4.75), // Expected as float
			},
			wantErr: false,
		},
		{
			name: "negative number values",
			query: map[string][]string{
				"filters[temperature][gte]": {"-10.5"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("temperature", filter.OpGte, -10.5), // Expected as float
			},
			wantErr: false,
		},
		{
			name: "special characters in field name",
			query: map[string][]string{
				"filters[user_name][eq]": {"john_doe"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("user_name", filter.OpEq, "john_doe"),
			},
			wantErr: false,
		},
		{
			name: "empty array for in operator",
			query: map[string][]string{
				"filters[ids][in]": {},
			},
			wantErr:  true, // Expect error if 'in' is empty
			expected: nil,  // No expected filters on error
		},
		{
			name: "invalid numeric value",
			query: map[string][]string{
				"filters[age][eq]": {"twenty"},
			},
			wantErr: false, // Expect string if conversion fails
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpEq, "twenty"),
			},
		},
		{
			name: "neq operator with string",
			query: map[string][]string{
				"filters[status][neq]": {"inactive"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("status", filter.OpNe, "inactive"),
			},
			wantErr: false,
		},
		{
			name: "mixed types in in array",
			query: map[string][]string{
				"filters[codes][in]": {"123", "abc", "45.6"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("codes", filter.OpIn, []any{123, "abc", 45.6}), // Expect mixed types with conversion
			},
			wantErr: false,
		},
		{
			name: "case sensitive field names",
			query: map[string][]string{
				"filters[UserName][eq]": {"john"},
			},
			wantErr: false, // Field names are usually case-sensitive as provided
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("UserName", filter.OpEq, "john"),
			},
		},
		{
			name: "multiple values for between operator",
			query: map[string][]string{
				"filters[price][between]": {"10", "20"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("price", filter.OpBetween, []any{10, 20}),
			},
			wantErr: false,
		},
		{
			name: "special characters in value",
			query: map[string][]string{
				"filters[note][contains]": {"hello,world"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("note", filter.OpContains, "hello,world"),
			},
			wantErr: false,
		},
		{
			name: "empty value for contains operator",
			query: map[string][]string{
				"filters[search][contains]": {""},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("search", filter.OpContains, ""),
			},
			wantErr: false,
		},
		{
			name: "zero value for between operator",
			query: map[string][]string{
				"filters[offset][between]": {"0", "10"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("offset", filter.OpBetween, []any{0, 10}), // Expected as numbers
			},
			wantErr: false,
		},
		{
			name: "very large numeric values",
			query: map[string][]string{
				"filters[population][gte]": {"1000000000"},
				"filters[ratio][lte]":      {"3.141592653589793"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("population", filter.OpGte, 1000000000),
				filter.NewLogicalFilter("ratio", filter.OpLte, 3.141592653589793),
			},
			wantErr: false,
		},
		{
			name: "mixed boolean representations",
			query: map[string][]string{
				"filters[active][eq]": {"1"},
				"filters[vip][eq]":    {"0"},
			},
			// Assuming '1' and '0' convert to numbers, not bools
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("active", filter.OpEq, 1),
				filter.NewLogicalFilter("vip", filter.OpEq, 0),
			},
			wantErr: false,
		},
		{
			name: "case-insensitive boolean error",
			query: map[string][]string{
				"filters[flag][eq]": {"True"},
			},
			wantErr:  true, // Assuming only lowercase 'true'/'false' are supported as bools
			expected: nil,  // No expected filters on error
		},
		{
			name: "invalid between array size",
			query: map[string][]string{
				"filters[range][between]": {"10"},
			},
			wantErr:  true, // Expect error if 'between' does not have 2 values
			expected: nil,  // No expected filters on error
		},
		{
			name: "url encoded values",
			query: map[string][]string{
				"filters[message][contains]": {"hello%20world"},
			},
			// Assuming URL parameters are parsed *before* the FilterParser
			// and the values are already decoded by net/url.Values.
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("message", filter.OpContains, "hello world"), // Expected decoded value
			},
			wantErr: false,
		},
		{
			name: "duplicate filters rejection",
			query: map[string][]string{
				"filters[status][eq]": {"active", "inactive"},
			},
			wantErr:  true, // Expect error if a non-array operator gets multiple values
			expected: nil,  // No expected filters on error
		},
		{
			name: "empty filter key segments",
			query: map[string][]string{
				"filters[][eq]": {"value"},
			},
			wantErr:  true, // Expect error on empty key segment
			expected: nil,  // No expected filters on error
		},
		{
			name: "negative numbers in in operator",
			query: map[string][]string{
				"filters[temps][in]": {"-5", "-10", "0"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("temps", filter.OpIn, []any{-5, -10, 0}), // Expected as numbers
			},
			wantErr: false,
		},
		{
			name: "mixed string/number representations",
			query: map[string][]string{
				"filters[code][eq]":     {"123"},
				"filters[alt_code][eq]": {"456"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("code", filter.OpEq, 123),
				filter.NewLogicalFilter("alt_code", filter.OpEq, 456),
			},
			wantErr: false,
		},
		{
			name: "empty string value",
			query: map[string][]string{
				"filters[comment][eq]": {""},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("comment", filter.OpEq, ""),
			},
			wantErr: false,
		},
		{
			name: "invalid operator syntax",
			query: map[string][]string{
				"filters[rating][approx]": {"4.5"},
			},
			wantErr:  true, // Expect error on unknown operator
			expected: nil,  // No expected filters on error
		},
		{
			name: "multiple between values (duplicate test)",
			query: map[string][]string{
				"filters[price][between]": {"10", "20"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("price", filter.OpBetween, []any{10, 20}),
			},
			wantErr: false,
		},
		{
			name: "zero-prefixed numbers",
			query: map[string][]string{
				"filters[code][eq]": {"00123"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("code", filter.OpEq, 123), // Expected as number
			},
			wantErr: false,
		},
		{
			name: "scientific notation numbers",
			query: map[string][]string{
				"filters[value][eq]": {"1e3"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("value", filter.OpEq, 1000.0), // Expected as float
			},
			wantErr: false,
		},
		{
			name: "invalid uuid format",
			query: map[string][]string{
				"filters[id][eq]": {"not-a-uuid"},
			},
			// UUID validation is typically done *after* parsing.
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("id", filter.OpEq, "not-a-uuid"),
			},
			wantErr: false,
		},
		{
			name: "uppercase operator name (duplicate test)",
			query: map[string][]string{
				"filters[name][EQ]": {"john"},
			},
			wantErr:  true, // Expect error on unknown operator casing
			expected: nil,  // No expected filters on error
		},
		{
			name: "mixed boolean representations (duplicate test)",
			query: map[string][]string{
				"filters[active][eq]": {"1"},
				"filters[vip][eq]":    {"0"},
			},
			// Assuming '1' and '0' convert to numbers.
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("active", filter.OpEq, 1),
				filter.NewLogicalFilter("vip", filter.OpEq, 0),
			},
			wantErr: false,
		},
		{
			name: "exponential notation decimals",
			query: map[string][]string{
				"filters[measurement][eq]": {"2.5e3"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("measurement", filter.OpEq, 2500.0), // Expected as float
			},
			wantErr: false,
		},
		{
			name: "hyphen in field name",
			query: map[string][]string{
				"filters[user-id][eq]": {"42"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("user-id", filter.OpEq, 42), // Expected as number
			},
			wantErr: false,
		},
		{
			name: "nina operator with mixed types",
			query: map[string][]string{
				"filters[tags][nina]": {"123", "abc"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("tags", filter.OpNina, []any{123, "abc"}), // Mixed types expected
			},
			wantErr: false,
		},
		{
			name: "case-sensitive starts with",
			query: map[string][]string{
				"filters[code][startswiths]": {"ABC123"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("code", filter.OpStartswiths, "ABC123"),
			},
			wantErr: false,
		},
		{
			name: "multiple in values as separate params (duplicate test)",
			query: map[string][]string{
				"filters[id][in]": {"1", "2"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("id", filter.OpIn, []any{1, 2}), // Expected as numbers
			},
			wantErr: false,
		},
		{
			name: "between with zero and negative",
			query: map[string][]string{
				"filters[temp][between]": {"-10", "0"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("temp", filter.OpBetween, []any{-10, 0}), // Expected as numbers
			},
			wantErr: false,
		},
		{
			name: "mixed top-level operators",
			query: map[string][]string{
				"filters[name][contains]": {"test"},
				"filters[age][gte]":       {"25"},
				"filters[active][eq]":     {"true"},
			},
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpContains, "test"),
				filter.NewLogicalFilter("age", filter.OpGte, 25),
				filter.NewLogicalFilter("active", filter.OpEq, true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewQueryParamParser(url.Values(tt.query))
			filters, err := parser.ParseFilters()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// We no longer need a separate transformation step as expected is already built with constructors

			// Sort filters for consistent comparison
			// Sorting logic needs to access fields via getters if they are private
			sort.SliceStable(filters, func(i, j int) bool {
				f1 := filters[i]
				f2 := filters[j]
				var field1, field2 string
				// Prioritize LogicalFilter's Field for sorting
				if lf, ok := f1.(*filter.LogicalFilter); ok {
					field1 = lf.Field() // Use getter
				} else if cf, ok := f1.(*filter.ConditionalFilter); ok {
					// Fallback for ConditionalFilter: sort by operator string
					field1 = cf.GetOperator().String() // Use getter
					// Add first child field if available
					if len(cf.GetFilters()) > 0 { // Use getter
						if lfc, okc := cf.GetFilters()[0].(*filter.LogicalFilter); okc { // Use getter
							field1 += ":" + lfc.Field() // Use getter
						}
					}
				}

				if lf, ok := f2.(*filter.LogicalFilter); ok {
					field2 = lf.Field() // Use getter
				} else if cf, ok := f2.(*filter.ConditionalFilter); ok {
					field2 = cf.GetOperator().String() // Use getter
					if len(cf.GetFilters()) > 0 {      // Use getter
						if lfc, okc := cf.GetFilters()[0].(*filter.LogicalFilter); okc { // Use getter
							field2 += ":" + lfc.Field() // Use getter
						}
					}
				}
				return field1 < field2
			})

			sort.SliceStable(tt.expected, func(i, j int) bool {
				f1 := tt.expected[i]
				f2 := tt.expected[j]
				var field1, field2 string
				if lf, ok := f1.(*filter.LogicalFilter); ok {
					field1 = lf.Field() // Use getter
				} else if cf, ok := f1.(*filter.ConditionalFilter); ok {
					field1 = string(cf.GetOperator()) // Use getter
					if len(cf.GetFilters()) > 0 {     // Use getter
						if lfc, okc := cf.GetFilters()[0].(*filter.LogicalFilter); okc { // Use getter
							field1 += ":" + lfc.Field() // Use getter
						}
					}
				}
				if lf, ok := f2.(*filter.LogicalFilter); ok {
					field2 = lf.Field() // Use getter
				} else if cf, ok := f2.(*filter.ConditionalFilter); ok {
					field2 = string(cf.GetOperator()) // Use getter
					if len(cf.GetFilters()) > 0 {     // Use getter
						if lfc, okc := cf.GetFilters()[0].(*filter.LogicalFilter); okc { // Use getter
							field2 += ":" + lfc.Field() // Use getter
						}
					}
				}
				return field1 < field2
			})

			// Use assert.Equal for deep comparison. This works correctly
			// when both slices contain objects created via constructors
			// that populate the same (exported or unexported) fields.
			assert.Equal(t, tt.expected, filters, "Parsed filters do not match expected")

		})
	}
}
