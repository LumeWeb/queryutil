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
		expected []filter.CrudFilter
		wantErr  bool
	}{
		{
			name: "basic equality",
			query: map[string][]string{
				"filters[name]": {"john"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "name",
					Operator: filter.OpEq,
					Value:    "john",
				},
			},
			wantErr: false,
		},
		{
			name: "not equals operator",
			query: map[string][]string{
				"filters[age][ne]": {"20"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "age",
					Operator: filter.OpNe,
					Value:    20,
				},
			},
			wantErr: false,
		},
		{
			name: "contains operator",
			query: map[string][]string{
				"filters[title][contains]": {"test"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "title",
					Operator: filter.OpContains,
					Value:    "test",
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalOr,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalAnd,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "name",
									Operator: filter.OpEq,
									Value:    "john",
								},
								&filter.LogicalFilter{
									Field:    "age",
									Operator: filter.OpGte,
									Value:    30,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "global search",
			query: map[string][]string{
				"filters[q]": {"searchterm"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "q",
					Operator: filter.OpEq,
					Value:    "searchterm",
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported operator",
			query: map[string][]string{
				"filters[status][invalid]": {"value"},
			},
			wantErr:  true,
			expected: []filter.CrudFilter{},
		},
		{
			name: "numeric value types",
			query: map[string][]string{
				"filters[age][eq]":    {"25"},
				"filters[price][gte]": {"19.99"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "age",
					Operator: filter.OpEq,
					Value:    25,
				},
				&filter.LogicalFilter{
					Field:    "price",
					Operator: filter.OpGte,
					Value:    19.99,
				},
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
				&filter.LogicalFilter{
					Field:    "active",
					Operator: filter.OpEq,
					Value:    true,
				},
				&filter.LogicalFilter{
					Field:    "vip",
					Operator: filter.OpNe,
					Value:    false,
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalOr,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalAnd,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "status",
									Operator: filter.OpEq,
									Value:    "active",
								},
								&filter.LogicalFilter{
									Field:    "age",
									Operator: filter.OpGte,
									Value:    "18",
								},
							},
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalNot,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "name",
									Operator: filter.OpContains,
									Value:    "test",
								},
							},
						},
					},
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalAnd,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "category",
									Operator: filter.OpEq,
									Value:    "books",
								},
								&filter.LogicalFilter{
									Field:    "category",
									Operator: filter.OpEq,
									Value:    "movies",
								},
							},
						},
						&filter.LogicalFilter{
							Field:    "price",
							Operator: filter.OpBetween,
							Value:    []any{10, 50},
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.ConditionalFilter{
									Operator: filter.LogicalAnd,
									Filters: []filter.CrudFilter{
										&filter.LogicalFilter{
											Field:    "type",
											Operator: filter.OpEq,
											Value:    "digital",
										},
										&filter.LogicalFilter{
											Field:    "stock",
											Operator: filter.OpGte,
											Value:    5,
										},
									},
								},
								&filter.ConditionalFilter{
									Operator: filter.LogicalAnd,
									Filters: []filter.CrudFilter{
										&filter.LogicalFilter{
											Field:    "type",
											Operator: filter.OpEq,
											Value:    "physical",
										},
										&filter.LogicalFilter{
											Field:    "stock",
											Operator: filter.OpGte,
											Value:    10,
										},
									},
								},
							},
						},
					},
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalAnd,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "author",
									Operator: filter.OpEq,
									Value:    "john",
								},
								&filter.LogicalFilter{
									Field:    "author",
									Operator: filter.OpEq,
									Value:    "jane",
								},
							},
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "year",
									Operator: filter.OpGte,
									Value:    2020,
								},
								&filter.LogicalFilter{
									Field:    "year",
									Operator: filter.OpLte,
									Value:    1990,
								},
							},
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalAnd,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "available",
									Operator: filter.OpEq,
									Value:    true,
								},
								&filter.LogicalFilter{
									Field:    "rating",
									Operator: filter.OpGte,
									Value:    4.5,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty string value for eq operator",
			query: map[string][]string{
				"filters[name][eq]": {""}, // Note the $eq
			},
			wantErr: false, // If $eq allows empty strings
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "name",
					Operator: filter.OpEq,
					Value:    "",
				},
			},
		},
		{
			name: "invalid array index",
			query: map[string][]string{
				"filters[or][invalid][age][eq]": {"30"},
			},
			wantErr: true,
		},
		{
			name: "null operator",
			query: map[string][]string{
				"filters[deleted_at][null]": {""},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "deleted_at",
					Operator: filter.OpNull,
					Value:    nil,
				},
			},
			wantErr: false,
		},
		{
			name: "not null operator",
			query: map[string][]string{
				"filters[updated_at][nnull]": {""},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "updated_at",
					Operator: filter.OpNnull,
					Value:    nil,
				},
			},
			wantErr: false,
		},
		{
			name: "between operator with numeric values",
			query: map[string][]string{
				"filters[price][between]": {"10", "20"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "price",
					Operator: filter.OpBetween,
					Value:    []any{10, 20},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple values for in operator",
			query: map[string][]string{
				"filters[id][in]": {"1", "2", "3"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "id",
					Operator: filter.OpIn,
					Value:    []any{1, 2, 3}, // EXPECT NUMBERS
				},
			},
			wantErr: false,
		},
		{
			name: "invalid boolean value",
			query: map[string][]string{
				"filters[active][eq]": {"notaboolean"},
			},
			wantErr: false,
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "active",
					Operator: filter.OpEq,
					Value:    "notaboolean",
				},
			},
		},
		{
			name: "case-sensitive operator name",
			query: map[string][]string{
				"filters[name][EQ]": {"john"},
			},
			wantErr: true,
		},
		{
			name: "empty filter key",
			query: map[string][]string{
				"filters[][eq]": {"value"},
			},
			wantErr: true,
		},
		{
			name: "in-array operator with mixed values",
			query: map[string][]string{
				"filters[tags][ina]": {"go", "123"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "tags",
					Operator: filter.OpIna,
					Value:    []any{"go", 123},
				},
			},
			wantErr: false,
		},
		{
			name: "reject object-based query with multiple operators",
			query: map[string][]string{
				"filters[age][eq]": {"25"},
				"filters[age][gt]": {"20"},
			},
			wantErr: true,
		},
		{
			name: "reject nested object structure",
			query: map[string][]string{
				"filters[user][name][eq]": {"john"},
				"filters[user][age][gt]":  {"30"},
			},
			wantErr: true,
		},
		{
			name: "mixed conditional operators",
			query: map[string][]string{
				"filters[or][0][and][0][status][eq]": {"active"},
				"filters[or][0][and][1][price][lt]":  {"100"},
				"filters[or][1][name][contains]":     {"special"},
			},
			expected: []filter.CrudFilter{
				&filter.ConditionalFilter{
					Operator: filter.LogicalOr,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalAnd,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "status",
									Operator: filter.OpEq,
									Value:    "active",
								},
								&filter.LogicalFilter{
									Field:    "price",
									Operator: filter.OpLt,
									Value:    100,
								},
							},
						},
						&filter.LogicalFilter{
							Field:    "name",
							Operator: filter.OpContains,
							Value:    "special",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty array value for in operator",
			query: map[string][]string{
				"filters[id][in]": {},
			},
			wantErr: true,
		},
		{
			name: "string with commas for in operator",
			query: map[string][]string{
				"filters[id][in]": {"1,2,3"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "id",
					Operator: filter.OpIn,
					Value:    []any{1, 2, 3},
				},
			},
			wantErr: false,
		},
		{
			name: "zero value handling",
			query: map[string][]string{
				"filters[age][eq]": {"0"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "age",
					Operator: filter.OpEq,
					Value:    0,
				},
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
				&filter.LogicalFilter{
					Field:    "name",
					Operator: filter.OpContains,
					Value:    "test",
				},
				&filter.LogicalFilter{
					Field:    "status",
					Operator: filter.OpEq,
					Value:    "active",
				},
			},
			wantErr: false,
		},
		{
			name: "mixed type array values",
			query: map[string][]string{
				"filters[id][in]": {"1", "abc"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "id",
					Operator: filter.OpIn,
					Value:    []any{1, "abc"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid boolean casing",
			query: map[string][]string{
				"filters[active][eq]": {"True"},
			},
			wantErr: true,
		},
		{
			name: "non-numeric between values",
			query: map[string][]string{
				"filters[price][between]": {"ten", "twenty"},
			},
			wantErr: true,
		},
		{
			name: "nnull operator with value",
			query: map[string][]string{
				"filters[updated_at][nnull]": {"123"},
			},
			wantErr: true,
		},
		{
			name: "single value in array operator",
			query: map[string][]string{
				"filters[category][in]": {"shoes"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "category",
					Operator: filter.OpIn,
					Value:    []any{"shoes"},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple values for equality operator",
			query: map[string][]string{
				"filters[age][eq]": {"30", "31"},
			},
			wantErr: true,
		},
		{
			name: "decimal value handling",
			query: map[string][]string{
				"filters[rating][eq]": {"4.75"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "rating",
					Operator: filter.OpEq,
					Value:    4.75,
				},
			},
			wantErr: false,
		},
		{
			name: "negative number values",
			query: map[string][]string{
				"filters[temperature][gte]": {"-10.5"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "temperature",
					Operator: filter.OpGte,
					Value:    -10.5,
				},
			},
			wantErr: false,
		},
		{
			name: "special characters in field name",
			query: map[string][]string{
				"filters[user_name][eq]": {"john_doe"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "user_name",
					Operator: filter.OpEq,
					Value:    "john_doe",
				},
			},
			wantErr: false,
		},
		{
			name: "empty array for in operator",
			query: map[string][]string{
				"filters[ids][in]": {},
			},
			wantErr: true,
		},
		{
			name: "invalid numeric value",
			query: map[string][]string{
				"filters[age][eq]": {"twenty"},
			},
			wantErr: false,
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "age",
					Operator: filter.OpEq,
					Value:    "twenty",
				},
			},
		},
		{
			name: "neq operator with string",
			query: map[string][]string{
				"filters[status][neq]": {"inactive"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "status",
					Operator: filter.OpNe,
					Value:    "inactive",
				},
			},
			wantErr: false,
		},
		{
			name: "mixed types in in array",
			query: map[string][]string{
				"filters[codes][in]": {"123", "abc", "45.6"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "codes",
					Operator: filter.OpIn,
					Value:    []any{123, "abc", 45.6},
				},
			},
			wantErr: false,
		},
		{
			name: "case sensitive field names",
			query: map[string][]string{
				"filters[UserName][eq]": {"john"},
			},
			wantErr: false,
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "UserName",
					Operator: filter.OpEq,
					Value:    "john",
				},
			},
		},
		{
			name: "multiple values for between operator",
			query: map[string][]string{
				"filters[price][between]": {"10", "20"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "price",
					Operator: filter.OpBetween,
					Value:    []any{10, 20},
				},
			},
			wantErr: false,
		},
		{
			name: "special characters in value",
			query: map[string][]string{
				"filters[note][contains]": {"hello,world"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "note",
					Operator: filter.OpContains,
					Value:    "hello,world",
				},
			},
			wantErr: false,
		},
		{
			name: "empty value for contains operator",
			query: map[string][]string{
				"filters[search][contains]": {""},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "search",
					Operator: filter.OpContains,
					Value:    "",
				},
			},
			wantErr: false,
		},
		{
			name: "zero value for between operator",
			query: map[string][]string{
				"filters[offset][between]": {"0", "10"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "offset",
					Operator: filter.OpBetween,
					Value:    []any{0, 10},
				},
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
				&filter.LogicalFilter{
					Field:    "population",
					Operator: filter.OpGte,
					Value:    1000000000,
				},
				&filter.LogicalFilter{
					Field:    "ratio",
					Operator: filter.OpLte,
					Value:    3.141592653589793,
				},
			},
			wantErr: false,
		},
		{
			name: "mixed boolean representations",
			query: map[string][]string{
				"filters[active][eq]": {"1"},
				"filters[vip][eq]":    {"0"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "active",
					Operator: filter.OpEq,
					Value:    1,
				},
				&filter.LogicalFilter{
					Field:    "vip",
					Operator: filter.OpEq,
					Value:    0,
				},
			},
			wantErr: false,
		},
		{
			name: "case-insensitive boolean error",
			query: map[string][]string{
				"filters[flag][eq]": {"True"},
			},
			wantErr: true,
		},
		{
			name: "invalid between array size",
			query: map[string][]string{
				"filters[range][between]": {"10"},
			},
			wantErr: true,
		},
		{
			name: "url encoded values",
			query: map[string][]string{
				"filters[message][contains]": {"hello%20world"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "message",
					Operator: filter.OpContains,
					Value:    "hello%20world",
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate filters rejection",
			query: map[string][]string{
				"filters[status][eq]": {"active", "inactive"},
			},
			wantErr: true,
		},
		{
			name: "empty filter key segments",
			query: map[string][]string{
				"filters[][eq]": {"value"},
			},
			wantErr: true,
		},
		{
			name: "negative numbers in in operator",
			query: map[string][]string{
				"filters[temps][in]": {"-5", "-10", "0"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "temps",
					Operator: filter.OpIn,
					Value:    []any{-5, -10, 0},
				},
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
				&filter.LogicalFilter{
					Field:    "code",
					Operator: filter.OpEq,
					Value:    123,
				},
				&filter.LogicalFilter{
					Field:    "alt_code",
					Operator: filter.OpEq,
					Value:    456,
				},
			},
			wantErr: false,
		},
		{
			name: "empty string value",
			query: map[string][]string{
				"filters[comment][eq]": {""},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "comment",
					Operator: filter.OpEq,
					Value:    "",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid operator syntax",
			query: map[string][]string{
				"filters[rating][approx]": {"4.5"},
			},
			wantErr: true,
		},
		{
			name: "multiple between values",
			query: map[string][]string{
				"filters[price][between]": {"10", "20"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "price",
					Operator: filter.OpBetween,
					Value:    []any{10, 20},
				},
			},
			wantErr: false,
		},
		{
			name: "zero-prefixed numbers",
			query: map[string][]string{
				"filters[code][eq]": {"00123"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "code",
					Operator: filter.OpEq,
					Value:    123,
				},
			},
			wantErr: false,
		},
		{
			name: "scientific notation numbers",
			query: map[string][]string{
				"filters[value][eq]": {"1e3"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "value",
					Operator: filter.OpEq,
					Value:    1000.0,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid uuid format",
			query: map[string][]string{
				"filters[id][eq]": {"not-a-uuid"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "id",
					Operator: filter.OpEq,
					Value:    "not-a-uuid",
				},
			},
			wantErr: false,
		},
		{
			name: "uppercase operator name",
			query: map[string][]string{
				"filters[name][EQ]": {"john"},
			},
			wantErr: true,
		},
		{
			name: "mixed boolean representations",
			query: map[string][]string{
				"filters[active][eq]": {"1"},
				"filters[vip][eq]":    {"0"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "active",
					Operator: filter.OpEq,
					Value:    1,
				},
				&filter.LogicalFilter{
					Field:    "vip",
					Operator: filter.OpEq,
					Value:    0,
				},
			},
			wantErr: false,
		},
		{
			name: "exponential notation decimals",
			query: map[string][]string{
				"filters[measurement][eq]": {"2.5e3"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "measurement",
					Operator: filter.OpEq,
					Value:    2500.0,
				},
			},
			wantErr: false,
		},
		{
			name: "hyphen in field name",
			query: map[string][]string{
				"filters[user-id][eq]": {"42"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "user-id",
					Operator: filter.OpEq,
					Value:    42,
				},
			},
			wantErr: false,
		},
		{
			name: "url encoded value handling",
			query: map[string][]string{
				"filters[message][contains]": {"hello%20world"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "message",
					Operator: filter.OpContains,
					Value:    "hello%20world",
				},
			},
			wantErr: false,
		},
		{
			name: "nina operator with mixed types",
			query: map[string][]string{
				"filters[tags][nina]": {"123", "abc"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "tags",
					Operator: filter.OpNina,
					Value:    []any{123, "abc"},
				},
			},
			wantErr: false,
		},
		{
			name: "case-sensitive starts with",
			query: map[string][]string{
				"filters[code][startswiths]": {"ABC123"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "code",
					Operator: filter.OpStartswiths,
					Value:    "ABC123",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple in values as separate params",
			query: map[string][]string{
				"filters[id][in]": {"1", "2"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "id",
					Operator: filter.OpIn,
					Value:    []any{1, 2},
				},
			},
			wantErr: false,
		},
		{
			name: "between with zero and negative",
			query: map[string][]string{
				"filters[temp][between]": {"-10", "0"},
			},
			expected: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "temp",
					Operator: filter.OpBetween,
					Value:    []any{-10, 0},
				},
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
				&filter.LogicalFilter{
					Field:    "name",
					Operator: filter.OpContains,
					Value:    "test",
				},
				&filter.LogicalFilter{
					Field:    "age",
					Operator: filter.OpGte,
					Value:    25,
				},
				&filter.LogicalFilter{
					Field:    "active",
					Operator: filter.OpEq,
					Value:    true,
				},
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

			// Sort filters for consistent comparison
			sort.SliceStable(filters, func(i, j int) bool {
				f1 := filters[i]
				f2 := filters[j]
				var field1, field2 string
				// Prioritize LogicalFilter's Field for sorting
				if lf, ok := f1.(*filter.LogicalFilter); ok {
					field1 = lf.Field
				} else if cf, ok := f1.(*filter.ConditionalFilter); ok {
					// Fallback for ConditionalFilter: sort by operator string
					field1 = string(cf.Operator)
					// Add first child field if available
					if len(cf.Filters) > 0 {
						if lfc, okc := cf.Filters[0].(*filter.LogicalFilter); okc {
							field1 += ":" + lfc.Field
						}
					}
				}

				if lf, ok := f2.(*filter.LogicalFilter); ok {
					field2 = lf.Field
				} else if cf, ok := f2.(*filter.ConditionalFilter); ok {
					field2 = string(cf.Operator)
					if len(cf.Filters) > 0 {
						if lfc, okc := cf.Filters[0].(*filter.LogicalFilter); okc {
							field2 += ":" + lfc.Field
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
					field1 = lf.Field
				} else if cf, ok := f1.(*filter.ConditionalFilter); ok {
					field1 = string(cf.Operator)
					if len(cf.Filters) > 0 {
						if lfc, okc := cf.Filters[0].(*filter.LogicalFilter); okc {
							field1 += ":" + lfc.Field
						}
					}
				}
				if lf, ok := f2.(*filter.LogicalFilter); ok {
					field2 = lf.Field
				} else if cf, ok := f2.(*filter.ConditionalFilter); ok {
					field2 = string(cf.Operator)
					if len(cf.Filters) > 0 {
						if lfc, okc := cf.Filters[0].(*filter.LogicalFilter); okc {
							field2 += ":" + lfc.Field
						}
					}
				}
				return field1 < field2
			})

			assert.Equal(t, len(tt.expected), len(filters), "Number of filters mismatch")

			for i, expected := range tt.expected {
				switch exp := expected.(type) {
				case *filter.LogicalFilter:
					act, ok := filters[i].(*filter.LogicalFilter)
					assert.True(t, ok, "Expected LogicalFilter at position %d", i)
					assert.Equal(t, exp.Field, act.Field, "Field mismatch")
					assert.Equal(t, exp.Operator, act.Operator, "Operator mismatch")
					assert.Equal(t, exp.Value, act.Value, "Value mismatch")

				case *filter.ConditionalFilter:
					act, ok := filters[i].(*filter.ConditionalFilter)
					assert.True(t, ok, "Expected ConditionalFilter at position %d", i)
					assert.Equal(t, exp.Operator, act.Operator, "Operator mismatch")
					assert.Equal(t, len(exp.Filters), len(act.Filters), "Nested filters count mismatch")
				}
			}
		})
	}
}
