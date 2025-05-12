package filter

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConditionalFilter(t *testing.T) {
	t.Run("valid OR operator with multiple filters", func(t *testing.T) {
		f1 := &LogicalFilter{Field: "a", Operator: OpEq, Value: 1}
		f2 := &LogicalFilter{Field: "b", Operator: OpGt, Value: 5}
		
		cf := NewConditionalFilter(LogicalOr, []CrudFilter{f1, f2})
		
		assert.Equal(t, string(LogicalOr), cf.GetOperator())
		require.Len(t, cf.GetFilters(), 2)
		assert.IsType(t, &LogicalFilter{}, cf.GetFilters()[0])
		assert.IsType(t, &LogicalFilter{}, cf.GetFilters()[1])
	})

	t.Run("valid NOT operator with single filter", func(t *testing.T) {
		f := &LogicalFilter{Field: "x", Operator: OpNull, Value: nil}
		
		cf := NewConditionalFilter(LogicalNot, []CrudFilter{f})
		
		assert.Equal(t, string(LogicalNot), cf.GetOperator())
		require.Len(t, cf.GetFilters(), 1)
		assert.IsType(t, &LogicalFilter{}, cf.GetFilters()[0])
	})

	t.Run("panic on invalid NOT operator usage", func(t *testing.T) {
		f1 := &LogicalFilter{Field: "a", Operator: OpEq, Value: 1}
		f2 := &LogicalFilter{Field: "b", Operator: OpNe, Value: 2}

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid NOT operator usage")
			}
		}()

		// This should panic because NOT has multiple filters
		NewConditionalFilter(LogicalNot, []CrudFilter{f1, f2})
	})
}
