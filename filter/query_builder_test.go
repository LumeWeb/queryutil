package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNumericOrTimeType(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{
			name:    "valid numeric int",
			value:   42,
			wantErr: false,
		},
		{
			name:    "valid numeric float",
			value:   3.14,
			wantErr: false,
		},
		{
			name:    "valid time.Time",
			value:   time.Now(),
			wantErr: false,
		},
		{
			name:    "valid RFC3339 time string",
			value:   "2025-06-09T12:34:56Z",
			wantErr: false,
		},
		{
			name:    "valid MySQL datetime string",
			value:   "2025-06-09 12:34:56.789",
			wantErr: false,
		},
		{
			name:    "valid date string",
			value:   "2025-06-09",
			wantErr: false,
		},
		{
			name:    "valid Go debug time string",
			value:   time.Now().String(), // Generates current time in Go debug format
			wantErr: false,
		},
		{
			name:    "valid timezone-aware string",
			value:   "2025-06-09T12:34:56+02:00",
			wantErr: false,
		},
		{
			name:    "invalid string",
			value:   "not a time",
			wantErr: true,
		},
		{
			name:    "invalid type",
			value:   []int{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantErr {
						t.Errorf("validateNumericOrTimeType() panicked unexpectedly: %v", r)
					}
				}
			}()

			validateNumericOrTimeType("test_field", tt.value)

			if tt.wantErr {
				t.Error("validateNumericOrTimeType() should have panicked but didn't")
			}
		})
	}
}

func TestQueryBuilderFunctions(t *testing.T) {
	t.Run("EqualityFilters", func(t *testing.T) {
		tests := []struct {
			name     string
			fn       func() CrudFilter
			expected *LogicalFilter
		}{
			{
				name: "Equal",
				fn:   func() CrudFilter { return Equal("age", 25) },
				expected: &LogicalFilter{
					field:    "age",
					operator: OpEq,
					value:    25,
				},
			},
			{
				name: "Search",
				fn:   func() CrudFilter { return Search("test") },
				expected: &LogicalFilter{
					field:    "q",
					operator: OpContains,
					value:    "test",
				},
			},
			{
				name: "NotEqual",
				fn:   func() CrudFilter { return NotEqual("status", "active") },
				expected: &LogicalFilter{
					field:    "status",
					operator: OpNe,
					value:    "active",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.fn()
				lf, ok := result.(*LogicalFilter)
				require.True(t, ok)
				assert.Equal(t, tt.expected, lf)
			})
		}
	})

	t.Run("ComparisonFilters", func(t *testing.T) {
		lf := GreaterThan("price", 100.50).(*LogicalFilter)
		assert.Equal(t, OpGt, lf.Operator())
		assert.Equal(t, 100.50, lf.Value())

		lf = LessOrEqual("quantity", 10).(*LogicalFilter)
		assert.Equal(t, OpLte, lf.Operator())
		assert.Equal(t, 10, lf.Value())
	})

	t.Run("LogicalCombinators", func(t *testing.T) {
		t.Run("And", func(t *testing.T) {
			f1 := Equal("a", 1)
			f2 := NotEqual("b", 2)
			cf := And(f1, f2).(*ConditionalFilter)

			assert.Equal(t, LogicalAnd, cf.Operator)
			require.Len(t, cf.Filters, 2)
			assert.Equal(t, f1, cf.Filters[0])
			assert.Equal(t, f2, cf.Filters[1])
		})

		t.Run("Or", func(t *testing.T) {
			f1 := Contains("name", "john")
			f2 := GreaterThan("age", 30)
			cf := Or(f1, f2).(*ConditionalFilter)

			assert.Equal(t, LogicalOr, cf.Operator)
			require.Len(t, cf.Filters, 2)
			assert.Equal(t, f1, cf.Filters[0])
			assert.Equal(t, f2, cf.Filters[1])
		})

		t.Run("Not", func(t *testing.T) {
			f := Equal("deleted", true)
			cf := Not(f).(*ConditionalFilter)

			assert.Equal(t, LogicalNot, cf.Operator)
			require.Len(t, cf.Filters, 1)
			assert.Equal(t, f, cf.Filters[0])
		})

	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("EmptyIn", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for empty In values")
				}
			}()
			In("tags")
		})

		t.Run("InvalidBetween", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for invalid Between values")
				}
			}()
			Between("price", 30, 10) // Missing min/max
		})

		t.Run("BetweenValidNumerics", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("Unexpected panic for valid Between values")
				}
			}()
			Between("age", 18, 65)
			Between("temperature", -10.5, 25.3)
			Between("rating", 4.0, 5.0)
		})

		t.Run("BetweenValidTimes", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("Unexpected panic for valid time ranges")
				}
			}()
			start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
			Between("created_at", start, end)
		})

		t.Run("BetweenInvalidTypes", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for non-numeric Between values")
				}
			}()
			Between("name", "a", "z")
		})

		t.Run("NotBetweenValidNumerics", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("Unexpected panic for valid NotBetween values")
				}
			}()
			FieldNotBetween("weight", 50, 100)
			FieldNotBetween("altitude", -100, 5000)
		})

		t.Run("NotBetweenPanic", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for invalid NotBetween values")
				}
			}()
			FieldNotBetween("height", 30, 10) // Missing min/max
		})

		t.Run("NotBetweenInvalidTypes", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for non-numeric NotBetween values")
				}
			}()
			FieldNotBetween("category", "a", "z")
		})

		t.Run("NotBetweenZeroValues", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("Unexpected panic for zero values")
				}
			}()
			FieldNotBetween("offset", 0, 0)
		})

		t.Run("InvalidStringOperatorOnNumber", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for using string operator with non-string value")
				}
			}()
			GreaterThan("name", "123") // Using numeric operator with string value
		})

		t.Run("NullOperatorWithValue", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for null operator with value")
				}
			}()
			NewLogicalFilter("deleted_at", OpNull, "invalid")
		})

		t.Run("InvalidOperatorOnBool", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for invalid operator on bool field")
				}
			}()
			GreaterThan("active", true)
		})

		t.Run("DeepNestedConditionals", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Error("Unexpected panic for valid deep nesting")
				}
			}()
			And(
				Or(
					Equal("a", 1),
					Not(
						And(
							Equal("b", 2),
							Equal("c", 3),
						),
					),
				),
				Not(
					Or(
						Equal("d", 4),
						Equal("e", 5),
					),
				),
			)
		})

		t.Run("MalformedMergeDeep", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for malformed deep merge")
				}
			}()
			MergeFilters(nil, nil, 999) // invalid merge strategy
		})

		t.Run("LogicalAndStringRepresentation", func(t *testing.T) {
			f := And(Equal("a", 1), Equal("b", 2)).String()
			assert.Contains(t, f, "AND( a eq 1, b eq 2 )")
		})
	})

	t.Run("TypeSafeBuilders", func(t *testing.T) {
		t.Run("StringField", func(t *testing.T) {
			sf := StringField("email")

			tests := []struct {
				name     string
				fn       func() CrudFilter
				expected *LogicalFilter
			}{
				{
					name: "Eq",
					fn:   func() CrudFilter { return sf.Eq("test@example.com") },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpEq,
						value:    "test@example.com",
					},
				},
				{
					name: "Contains",
					fn:   func() CrudFilter { return sf.Contains("domain") },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpContains,
						value:    "domain",
					},
				},
				{
					name: "StartsWith",
					fn:   func() CrudFilter { return sf.StartsWith("test") },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpContains,
						value:    "test%",
					},
				},
				{
					name: "EndsWith",
					fn:   func() CrudFilter { return sf.EndsWith(".com") },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpContains,
						value:    "%.com",
					},
				},
				{
					name: "NotIn",
					fn:   func() CrudFilter { return sf.NotIn("invalid", "spam") },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpNin,
						value:    []any{"invalid", "spam"},
					},
				},
				{
					name: "IsNull",
					fn:   func() CrudFilter { return sf.IsNull() },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpNull,
						value:    nil,
					},
				},
				{
					name: "IsNotNull",
					fn:   func() CrudFilter { return sf.IsNotNull() },
					expected: &LogicalFilter{
						field:    "email",
						operator: OpNnull,
						value:    nil,
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := tt.fn()
					lf, ok := result.(*LogicalFilter)
					require.True(t, ok)
					assert.Equal(t, tt.expected, lf)
				})
			}
		})

		t.Run("NumberField", func(t *testing.T) {
			nf := NumberField[int]("age")

			tests := []struct {
				name     string
				fn       func() CrudFilter
				expected *LogicalFilter
			}{
				{
					name: "Gt",
					fn:   func() CrudFilter { return nf.Gt(18) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpGt,
						value:    18,
					},
				},
				{
					name: "Lt",
					fn:   func() CrudFilter { return nf.Lt(30) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpLt,
						value:    30,
					},
				},
				{
					name: "Gte",
					fn:   func() CrudFilter { return nf.Gte(21) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpGte,
						value:    21,
					},
				},
				{
					name: "Lte",
					fn:   func() CrudFilter { return nf.Lte(65) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpLte,
						value:    65,
					},
				},
				{
					name: "Between",
					fn:   func() CrudFilter { return nf.Between(20, 30) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpBetween,
						value:    []any{20, 30},
					},
				},
				{
					name: "NotBetween",
					fn:   func() CrudFilter { return nf.NotBetween(10, 20) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpNbetween,
						value:    []any{10, 20},
					},
				},
				{
					name: "NotIn",
					fn:   func() CrudFilter { return nf.NotIn(99, 100) },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpNin,
						value:    []any{99, 100},
					},
				},
				{
					name: "IsNull",
					fn:   func() CrudFilter { return nf.IsNull() },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpNull,
						value:    nil,
					},
				},
				{
					name: "IsNotNull",
					fn:   func() CrudFilter { return nf.IsNotNull() },
					expected: &LogicalFilter{
						field:    "age",
						operator: OpNnull,
						value:    nil,
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := tt.fn()
					lf, ok := result.(*LogicalFilter)
					require.True(t, ok)
					assert.Equal(t, tt.expected, lf)
				})
			}
		})

		t.Run("BoolField", func(t *testing.T) {
			bf := BoolField("active")

			tests := []struct {
				name     string
				fn       func() CrudFilter
				expected *LogicalFilter
			}{
				{
					name: "Eq",
					fn:   func() CrudFilter { return bf.Eq(true) },
					expected: &LogicalFilter{
						field:    "active",
						operator: OpEq,
						value:    true,
					},
				},
				{
					name: "IsNull",
					fn:   func() CrudFilter { return bf.IsNull() },
					expected: &LogicalFilter{
						field:    "active",
						operator: OpNull,
						value:    nil,
					},
				},
				{
					name: "IsNotNull",
					fn:   func() CrudFilter { return bf.IsNotNull() },
					expected: &LogicalFilter{
						field:    "active",
						operator: OpNnull,
						value:    nil,
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := tt.fn()
					lf, ok := result.(*LogicalFilter)
					require.True(t, ok)
					assert.Equal(t, tt.expected, lf)
				})
			}
		})

		t.Run("TimeField", func(t *testing.T) {
			tf := TimeField("created_at")
			testTime := time.Date(2023, 10, 5, 14, 30, 0, 0, time.UTC)
			testTime2 := time.Date(2023, 10, 6, 0, 0, 0, 0, time.UTC)

			tests := []struct {
				name     string
				fn       func() CrudFilter
				expected *LogicalFilter
			}{
				{
					name: "Eq",
					fn:   func() CrudFilter { return tf.Eq(testTime) },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpEq,
						value:    testTime,
					},
				},
				{
					name: "Before",
					fn:   func() CrudFilter { return tf.Before(testTime) },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpLt,
						value:    testTime,
					},
				},
				{
					name: "After",
					fn:   func() CrudFilter { return tf.After(testTime) },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpGt,
						value:    testTime,
					},
				},
				{
					name: "Between",
					fn:   func() CrudFilter { return tf.Between(testTime, testTime2) },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpBetween,
						value:    []any{testTime, testTime2},
					},
				},
				{
					name: "IsNull",
					fn:   func() CrudFilter { return tf.IsNull() },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpNull,
						value:    nil,
					},
				},
				{
					name: "IsNotNull",
					fn:   func() CrudFilter { return tf.IsNotNull() },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpNnull,
						value:    nil,
					},
				},
				{
					name: "NotBetween",
					fn:   func() CrudFilter { return tf.NotBetween(testTime, testTime2) },
					expected: &LogicalFilter{
						field:    "created_at",
						operator: OpNbetween,
						value:    []any{testTime, testTime2},
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := tt.fn()
					lf, ok := result.(*LogicalFilter)
					require.True(t, ok)
					assert.Equal(t, tt.expected, lf)
				})
			}
		})
	})

	t.Run("MergeStrategies", func(t *testing.T) {
		base := []CrudFilter{Equal("a", 1), Equal("b", 2)}
		overrides := []CrudFilter{Equal("b", 3), Equal("c", 4)}

		t.Run("Override", func(t *testing.T) {
			merged := MergeFilters(base, overrides, MergeStrategyOverride)
			assert.Len(t, merged, 3)
			assert.Equal(t, 3, merged[0].(*LogicalFilter).Value()) // b=3 from overrides
			assert.Equal(t, 4, merged[1].(*LogicalFilter).Value()) // c=4 from overrides
			assert.Equal(t, 1, merged[2].(*LogicalFilter).Value()) // a=1 from base
		})

		t.Run("Append", func(t *testing.T) {
			merged := MergeFilters(base, overrides, MergeStrategyAppend)
			assert.Len(t, merged, 4)
		})
	})
}
