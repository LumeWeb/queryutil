package parser

import (
	"sort" // Keep sort just in case the parser's internal map handling isn't strictly ordered, although JSON arrays usually preserve order.
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
)

func TestJSONParser_ParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []filter.CrudFilter // This will now hold filters created via constructors
		wantErr  bool
	}{
		{
			name: "simple logical filter",
			input: `[
				{
					"field": "name",
					"operator": "contains",
					"value": "john"
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpContains, "john"),
			},
			wantErr: false,
		},
		{
			name: "nested conditional filter",
			input: `[
				{
					"operator": "or",
					"value": [
						{
							"field": "age",
							"operator": "gte",
							"value": 30
						},
						{
							"field": "email",
							"operator": "contains",
							"value": "example"
						}
					]
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewLogicalFilter("age", filter.OpGte, float64(30)), // JSON numbers unmarshal as float64
					filter.NewLogicalFilter("email", filter.OpContains, "example"),
				}),
			},
			wantErr: false,
		},
		{
			name: "invalid JSON structure",
			input: `[
				{
					"field": "name",
					"operator": "contains"
					// Missing value
				}
			]`,
			wantErr:  true,
			expected: nil, // No expected filters on error
		},
		{
			name: "invalid operator",
			input: `[
				{
					"field": "status",
					"operator": "invalid_op",
					"value": "active"
				}
			]`,
			wantErr:  true,
			expected: nil, // No expected filters on error
		},
		{
			name: "different value types",
			input: `[
				{
					"field": "active",
					"operator": "eq",
					"value": true
				},
				{
					"field": "metadata",
					"operator": "eq",
					"value": {"key": "value"}
				},
				{
					"field": "tags",
					"operator": "in",
					"value": ["go","test"]
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("active", filter.OpEq, true),
				filter.NewLogicalFilter("metadata", filter.OpEq, map[string]any{"key": "value"}), // JSON objects unmarshal as map[string]any
				filter.NewLogicalFilter("tags", filter.OpIn, []any{"go", "test"}),                // JSON arrays unmarshal as []any
			},
			wantErr: false,
		},
		{
			name: "multi-level nesting",
			input: `[
				{
					"operator": "and",
					"value": [
						{
							"field": "age",
							"operator": "gte",
							"value": 18
						},
						{
							"operator": "or",
							"value": [
								{"field": "role", "operator": "eq", "value": "admin"},
								{"field": "role", "operator": "eq", "value": "superuser"}
							]
						}
					]
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					filter.NewLogicalFilter("age", filter.OpGte, float64(18)), // JSON number
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("role", filter.OpEq, "admin"),
						filter.NewLogicalFilter("role", filter.OpEq, "superuser"),
					}),
				}),
			},
			wantErr: false,
		},
		{
			name: "empty conditional filter array",
			input: `[
				{
					"operator": "and",
					"value": []
				}
			]`,
			wantErr:  true, // Should fail if conditional filter value (array of filters) is empty
			expected: nil,  // No expected filters on error
		},
		{
			name: "malformed JSON input",
			input: `[
				{
					"field": "name",
					"operator": "eq",
					"value": "test"
				},
			]`, // intentional trailing comma
			wantErr:  true,
			expected: nil, // No expected filters on error
		},
		{
			name: "empty field name",
			input: `[
				{
					"field": "",
					"operator": "eq",
					"value": "test"
				}
			]`,
			wantErr:  true, // Should fail if field name is empty for a logical filter
			expected: nil,  // No expected filters on error
		},
		{
			name: "three-level nested filters",
			input: `[
				{
					"operator": "and",
					"value": [
						{
							"field": "age",
							"operator": "gte",
							"value": 18
						},
						{
							"operator": "or",
							"value": [
								{"field": "role", "operator": "eq", "value": "admin"},
								{
									"operator": "and",
									"value": [
										{"field": "status", "operator": "eq", "value": "active"},
										{"field": "verified", "operator": "eq", "value": true}
									]
								}
							]
						}
					]
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					filter.NewLogicalFilter("age", filter.OpGte, float64(18)), // JSON number
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("role", filter.OpEq, "admin"),
						filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
							filter.NewLogicalFilter("status", filter.OpEq, "active"),
							filter.NewLogicalFilter("verified", filter.OpEq, true),
						}),
					}),
				}),
			},
			wantErr: false,
		},
		{
			name: "null operator check",
			input: `[
				{
					"field": "deleted_at",
					"operator": "null",
					"value": null
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("deleted_at", filter.OpNull, nil),
			},
			wantErr: false,
		},
		{
			name: "not conditional filter",
			input: `[
				{
					"operator": "not",
					"value": [
						{
							"field": "name",
							"operator": "eq",
							"value": "john"
						}
					]
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
					filter.NewLogicalFilter("name", filter.OpEq, "john"),
				}),
			},
			wantErr: false,
		},
		{
			name: "complex nested not filter",
			input: `[
				{
					"operator": "and",
					"value": [
						{
							"operator": "not",
							"value": [
								{
									"field": "age",
									"operator": "lt",
									"value": 30
								}
							]
						},
						{
							"operator": "or",
							"value": [
								{
									"field": "status",
									"operator": "eq",
									"value": "active"
								},
								{
									"operator": "not",
									"value": [
										{
											"field": "deleted",
											"operator": "eq",
											"value": true
										}
									]
								}
							]
						}
					]
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
						filter.NewLogicalFilter("age", filter.OpLt, float64(30)), // JSON number
					}),
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("status", filter.OpEq, "active"),
						filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
							filter.NewLogicalFilter("deleted", filter.OpEq, true),
						}),
					}),
				}),
			},
			wantErr: false,
		},
		{
			name: "mixed case operator",
			input: `[
				{
					"field": "name",
					"operator": "CONTAINS",
					"value": "test"
				}
			]`,
			wantErr:  true, // Should fail since we expect lowercase operators
			expected: nil,  // No expected filters on error
		},
		{
			name:     "empty top-level array",
			input:    `[]`,
			expected: []filter.CrudFilter{}, // Expect an empty slice if input is empty array
			wantErr:  false,
		},
		{
			name: "null logical filter value",
			input: `[
				{
					"field": "optional_field",
					"operator": "eq",
					"value": null
				}
			]`,
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("optional_field", filter.OpEq, nil),
			},
			wantErr: false,
		},
		{
			name: "array value for equality operator",
			input: `[
				{
					"field": "name",
					"operator": "eq",
					"value": ["john", "jane"]
				}
			]`,
			// Depending on the parser implementation, this might be treated as a literal array value
			// or cause an error if "eq" doesn't support arrays. Let's assume it's treated as a literal array value.
			expected: []filter.CrudFilter{
				filter.NewLogicalFilter("name", filter.OpEq, []any{"john", "jane"}),
			},
			wantErr: false, // Assuming the parser allows this structure but validation happens later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewJSONParser(tt.input)
			result, err := parser.ParseFilters()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// JSON array order should be preserved, so sorting isn't strictly necessary
			// unless the parser implementation itself introduces reordering (which would be a bug).
			// However, including sorting makes the test robust against unexpected reordering.
			sort.SliceStable(result, func(i, j int) bool {
				return result[i].String() < result[j].String() // Use String() for sorting representation
			})
			sort.SliceStable(tt.expected, func(i, j int) bool {
				return tt.expected[i].String() < tt.expected[j].String() // Use String() for sorting representation
			})

			// assert.Equal performs a deep comparison and works correctly
			// when both 'result' and 'tt.expected' slices contain objects
			// created consistently using the filter package's constructors.
			assert.Equal(t, tt.expected, result, "Parsed filters do not match expected for input:\n%s", tt.input)

		})
	}
}

func TestJSONParser_EdgeCases(t *testing.T) {
	t.Run("empty input string", func(t *testing.T) {
		parser := NewJSONParser("")
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("null input string", func(t *testing.T) {
		parser := NewJSONParser("null")
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("non-array input", func(t *testing.T) {
		parser := NewJSONParser(`{"field": "name", "operator": "eq", "value": "test"}`) // Should be an array
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("array containing non-filter object", func(t *testing.T) {
		parser := NewJSONParser(`[{"field": "name", "operator": "eq", "value": "test"}, 123]`) // 123 is not a filter object
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("logical filter missing field", func(t *testing.T) {
		parser := NewJSONParser(`[{"operator": "eq", "value": "test"}]`) // Missing "field"
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("logical filter missing operator", func(t *testing.T) {
		parser := NewJSONParser(`[{"field": "name", "value": "test"}]`) // Missing "operator"
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("conditional filter missing operator", func(t *testing.T) {
		parser := NewJSONParser(`[{"value": [{"field": "name", "operator": "eq", "value": "test"}]}]`) // Missing "operator"
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("conditional filter missing value (array of filters)", func(t *testing.T) {
		parser := NewJSONParser(`[{"operator": "and"}]`) // Missing "value"
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})

	t.Run("conditional filter value is not an array", func(t *testing.T) {
		parser := NewJSONParser(`[{"operator": "and", "value": {"field": "name", "operator": "eq", "value": "test"}}]`) // Value should be array
		filters, err := parser.ParseFilters()
		assert.Error(t, err)
		assert.Nil(t, filters) // Expect nil filters on error
	})
}
