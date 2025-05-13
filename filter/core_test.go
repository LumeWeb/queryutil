package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConditionalFilter(t *testing.T) {
	t.Run("valid OR operator with multiple filters", func(t *testing.T) {
		f1 := &LogicalFilter{field: "a", operator: OpEq, value: 1}
		f2 := &LogicalFilter{field: "b", operator: OpGt, value: 5}

		cf := NewConditionalFilter(LogicalOr, []CrudFilter{f1, f2})

		assert.EqualValues(t, LogicalOr, cf.GetOperator())
		require.Len(t, cf.GetFilters(), 2)
		assert.IsType(t, &LogicalFilter{}, cf.GetFilters()[0])
		assert.IsType(t, &LogicalFilter{}, cf.GetFilters()[1])
	})

	t.Run("valid NOT operator with single filter", func(t *testing.T) {
		f := &LogicalFilter{field: "x", operator: OpNull, value: nil}

		cf := NewConditionalFilter(LogicalNot, []CrudFilter{f})

		assert.EqualValues(t, string(LogicalNot), cf.GetOperator())
		require.Len(t, cf.GetFilters(), 1)
		assert.IsType(t, &LogicalFilter{}, cf.GetFilters()[0])
	})

	t.Run("panic on invalid NOT operator usage", func(t *testing.T) {
		f1 := &LogicalFilter{field: "a", operator: OpEq, value: 1}
		f2 := &LogicalFilter{field: "b", operator: OpNe, value: 2}

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid NOT operator usage")
			}
		}()

		// This should panic because NOT has multiple filters
		NewConditionalFilter(LogicalNot, []CrudFilter{f1, f2})
	})
}

func TestFiltersFunction(t *testing.T) {
	t.Run("empty filters", func(t *testing.T) {
		result := Filters()
		assert.Empty(t, result)
	})

	t.Run("single filter", func(t *testing.T) {
		f := NewLogicalFilter("age", OpGte, 18)
		result := Filters(f)
		assert.Equal(t, []CrudFilter{f}, result)
	})

	t.Run("multiple filters", func(t *testing.T) {
		f1 := NewLogicalFilter("name", OpEq, "john")
		f2 := NewLogicalFilter("age", OpGte, 30)
		result := Filters(f1, f2)
		assert.Equal(t, []CrudFilter{f1, f2}, result)
	})
}

func TestAndFAndOrFFunctions(t *testing.T) {
	f1 := NewLogicalFilter("status", OpEq, "active")
	f2 := NewLogicalFilter("age", OpGte, 18)
	f3 := NewLogicalFilter("deleted", OpEq, false)

	t.Run("AndF combines filters with AND", func(t *testing.T) {
		cf := AndF(f1, f2)
		conditional, ok := cf.(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalAnd, conditional.Operator)
		assert.Equal(t, []CrudFilter{f1, f2}, conditional.Filters)
	})

	t.Run("OrF combines filters with OR", func(t *testing.T) {
		cf := OrF(f1, f3)
		conditional, ok := cf.(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalOr, conditional.Operator)
		assert.Equal(t, []CrudFilter{f1, f3}, conditional.Filters)
	})

	t.Run("nested combinations", func(t *testing.T) {
		complexFilter := AndF(
			OrF(f1, NewLogicalFilter("role", OpEq, "admin")),
			f2,
			OrF(f3, NewLogicalFilter("verified", OpEq, true)),
		)

		conditional, ok := complexFilter.(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalAnd, conditional.Operator)
		require.Len(t, conditional.Filters, 3)

		// Check first nested OR
		nestedOr, ok := conditional.Filters[0].(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalOr, nestedOr.Operator)
		assert.Equal(t, []CrudFilter{f1, NewLogicalFilter("role", OpEq, "admin")}, nestedOr.Filters)

		// Check second filter is f2
		assert.Equal(t, f2, conditional.Filters[1])

		// Check third nested OR
		nestedOr2, ok := conditional.Filters[2].(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalOr, nestedOr2.Operator)
		assert.Equal(t, []CrudFilter{f3, NewLogicalFilter("verified", OpEq, true)}, nestedOr2.Filters)
	})

	t.Run("empty AndF creates empty AND group", func(t *testing.T) {
		cf := AndF()
		conditional, ok := cf.(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalAnd, conditional.Operator)
		assert.Empty(t, conditional.Filters)
	})

	t.Run("empty OrF creates empty OR group", func(t *testing.T) {
		cf := OrF()
		conditional, ok := cf.(*ConditionalFilter)
		require.True(t, ok)
		assert.Equal(t, LogicalOr, conditional.Operator)
		assert.Empty(t, conditional.Filters)
	})
}
