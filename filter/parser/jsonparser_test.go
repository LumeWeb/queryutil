package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
)

func TestJSONParser_ParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []filter.CrudFilter
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
				&filter.LogicalFilter{
					Field:    "name",
					Operator: filter.OpContains,
					Value:    "john",
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalOr,
					Filters: []filter.CrudFilter{
						&filter.LogicalFilter{
							Field:    "age",
							Operator: filter.OpGte,
							Value:    float64(30), // JSON numbers decode as float64
						},
						&filter.LogicalFilter{
							Field:    "email",
							Operator: filter.OpContains,
							Value:    "example",
						},
					},
				},
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
			wantErr: true,
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
			wantErr: true,
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
				&filter.LogicalFilter{
					Field:    "active",
					Operator: filter.OpEq,
					Value:    true,
				},
				&filter.LogicalFilter{
					Field:    "metadata",
					Operator: filter.OpEq,
					Value:    map[string]any{"key": "value"},
				},
				&filter.LogicalFilter{
					Field:    "tags",
					Operator: filter.OpIn,
					Value:    []any{"go", "test"},
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalAnd,
					Filters: []filter.CrudFilter{
						&filter.LogicalFilter{
							Field:    "age",
							Operator: filter.OpGte,
							Value:    float64(18),
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "role",
									Operator: filter.OpEq,
									Value:    "admin",
								},
								&filter.LogicalFilter{
									Field:    "role",
									Operator: filter.OpEq,
									Value:    "superuser",
								},
							},
						},
					},
				},
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
			wantErr: true,
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
			wantErr: true,
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
			wantErr: true,
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalAnd,
					Filters: []filter.CrudFilter{
						&filter.LogicalFilter{
							Field:    "age",
							Operator: filter.OpGte,
							Value:    float64(18),
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "role",
									Operator: filter.OpEq,
									Value:    "admin",
								},
								&filter.ConditionalFilter{
									Operator: filter.LogicalAnd,
									Filters: []filter.CrudFilter{
										&filter.LogicalFilter{
											Field:    "status",
											Operator: filter.OpEq,
											Value:    "active",
										},
										&filter.LogicalFilter{
											Field:    "verified",
											Operator: filter.OpEq,
											Value:    true,
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
			name: "null operator check",
			input: `[
				{
					"field": "deleted_at",
					"operator": "null",
					"value": null
				}
			]`,
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalNot,
					Filters: []filter.CrudFilter{
						&filter.LogicalFilter{
							Field:    "name",
							Operator: filter.OpEq,
							Value:    "john",
						},
					},
				},
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
				&filter.ConditionalFilter{
					Operator: filter.LogicalAnd,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalNot,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "age",
									Operator: filter.OpLt,
									Value:    float64(30),
								},
							},
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "status",
									Operator: filter.OpEq,
									Value:    "active",
								},
								&filter.ConditionalFilter{
									Operator: filter.LogicalNot,
									Filters: []filter.CrudFilter{
										&filter.LogicalFilter{
											Field:    "deleted",
											Operator: filter.OpEq,
											Value:    true,
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
			name: "mixed case operator",
			input: `[
				{
					"field": "name",
					"operator": "CONTAINS",
					"value": "test"
				}
			]`,
			wantErr: true, // Should fail since we expect lowercase operators
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
			assert.Equal(t, len(tt.expected), len(result))

			// Compare each filter in detail
			for i, expectedFilter := range tt.expected {
				switch ef := expectedFilter.(type) {
				case *filter.LogicalFilter:
					actual, ok := result[i].(*filter.LogicalFilter)
					assert.True(t, ok, "Expected LogicalFilter at position %d", i)
					assert.Equal(t, ef.Field, actual.Field)
					assert.Equal(t, ef.Operator, actual.Operator)
					assert.Equal(t, ef.Value, actual.Value)

				case *filter.ConditionalFilter:
					actual, ok := result[i].(*filter.ConditionalFilter)
					assert.True(t, ok, "Expected ConditionalFilter at position %d", i)
					assert.Equal(t, ef.Operator, actual.Operator)
					assert.Equal(t, len(ef.Filters), len(actual.Filters))
				}
			}
		})
	}
}

func TestJSONParser_EdgeCases(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		parser := NewJSONParser("")
		_, err := parser.ParseFilters()
		assert.Error(t, err)
	})

	t.Run("non-array input", func(t *testing.T) {
		parser := NewJSONParser(`{"field": "name"}`)
		_, err := parser.ParseFilters()
		assert.Error(t, err)
	})
}
