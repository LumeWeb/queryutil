package queryutil

import (
	"go.lumeweb.com/queryutil/filter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindFilter(t *testing.T) {
	filters := []filter.CrudFilter{
		filter.NewLogicalFilter("status", filter.OpEq, "active"),
		filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
			filter.NewLogicalFilter("age", filter.OpGte, 18),
			filter.NewLogicalFilter("country", filter.OpEq, "USA"),
		}),
		filter.NewLogicalFilter("name", filter.OpContains, "john"),
	}

	// Test cases for FindFilter
	t.Run("shallow find - existing field", func(t *testing.T) {
		result := FindFilter(filters, "name")
		assert.NotNil(t, result)
		assert.Equal(t, "name", result.GetField())
	})

	t.Run("shallow find - non-existing field", func(t *testing.T) {
		result := FindFilter(filters, "city")
		assert.Nil(t, result)
	})

	t.Run("shallow find with operator - existing", func(t *testing.T) {
		result := FindFilterWithOperator(filters, "status", filter.OpEq)
		assert.NotNil(t, result)
		assert.Equal(t, "status", result.GetField())
		assert.Equal(t, filter.OpEq, result.GetOperator())
	})

	t.Run("shallow find with operator - non-existing operator", func(t *testing.T) {
		result := FindFilterWithOperator(filters, "status", filter.OpGt)
		assert.Nil(t, result)
	})

	t.Run("shallow find with operator - non-existing field", func(t *testing.T) {
		result := FindFilterWithOperator(filters, "city", filter.OpEq)
		assert.Nil(t, result)
	})

	// Test cases for FindFilters
	t.Run("shallow finds - existing field", func(t *testing.T) {
		results := FindFilters(filters, "status")
		assert.NotEmpty(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, "status", results[0].GetField())
	})

	t.Run("shallow finds - non-existing field", func(t *testing.T) {
		results := FindFilters(filters, "city")
		assert.Empty(t, results)
	})

	t.Run("shallow finds with operator - existing", func(t *testing.T) {
		results := FindFiltersWithOperator(filters, "status", filter.OpEq)
		assert.NotEmpty(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, "status", results[0].GetField())
		assert.Equal(t, filter.OpEq, results[0].GetOperator())
	})

	t.Run("shallow finds with operator - non-existing operator", func(t *testing.T) {
		results := FindFiltersWithOperator(filters, "status", filter.OpGt)
		assert.Empty(t, results)
	})

	t.Run("shallow finds with operator - non-existing field", func(t *testing.T) {
		results := FindFiltersWithOperator(filters, "city", filter.OpEq)
		assert.Empty(t, results)
	})

	// Test cases for DeepFindFilter
	t.Run("deep find - existing field", func(t *testing.T) {
		result := DeepFindFilter(filters, "age")
		assert.NotNil(t, result)
		assert.Equal(t, "age", result.GetField())
	})

	t.Run("deep find - non-existing field", func(t *testing.T) {
		result := DeepFindFilter(filters, "city")
		assert.Nil(t, result)
	})

	t.Run("deep find with operator - existing", func(t *testing.T) {
		result := DeepFindFilterWithOperator(filters, "age", filter.OpGte)
		assert.NotNil(t, result)
		assert.Equal(t, "age", result.GetField())
		assert.Equal(t, filter.OpGte, result.GetOperator())
	})

	t.Run("deep find with operator - non-existing operator", func(t *testing.T) {
		result := DeepFindFilterWithOperator(filters, "age", filter.OpLt)
		assert.Nil(t, result)
	})

	t.Run("deep find with operator - non-existing field", func(t *testing.T) {
		result := DeepFindFilterWithOperator(filters, "city", filter.OpEq)
		assert.Nil(t, result)
	})

	// Test cases for DeepFindFilters
	t.Run("deep finds - existing field", func(t *testing.T) {
		results := DeepFindFilters(filters, "country")
		assert.NotEmpty(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, "country", results[0].GetField())
	})

	t.Run("deep finds - non-existing field", func(t *testing.T) {
		results := DeepFindFilters(filters, "city")
		assert.Empty(t, results)
	})

	t.Run("deep finds with operator - existing", func(t *testing.T) {
		results := DeepFindFiltersWithOperator(filters, "age", filter.OpGte)
		assert.NotEmpty(t, results)
		assert.Len(t, results, 1)
		assert.Equal(t, "age", results[0].GetField())
		assert.Equal(t, filter.OpGte, results[0].GetOperator())
	})

	t.Run("deep finds with operator - non-existing operator", func(t *testing.T) {
		results := DeepFindFiltersWithOperator(filters, "age", filter.OpLt)
		assert.Empty(t, results)
	})

	t.Run("deep finds with operator - non-existing field", func(t *testing.T) {
		results := DeepFindFiltersWithOperator(filters, "city", filter.OpEq)
		assert.Empty(t, results)
	})

	t.Run("deep finds - multiple matches", func(t *testing.T) {
		// Create a fresh copy of filters to avoid modifying shared state
		testFilters := make([]filter.CrudFilter, len(filters))
		copy(testFilters, filters)
		
		// Add another nested filter with the same field
		testFilters[1].(*filter.ConditionalFilter).Filters = append(
			testFilters[1].(*filter.ConditionalFilter).Filters,
			filter.NewLogicalFilter("age", filter.OpLt, 30),
		)
		results := DeepFindFilters(testFilters, "age")
		assert.NotEmpty(t, results)
		assert.Len(t, results, 2)
		assert.Equal(t, "age", results[0].GetField())
		assert.Equal(t, "age", results[1].GetField())
	})

	t.Run("deep find - or conditional", func(t *testing.T) {
		filters := []filter.CrudFilter{
			filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpGte, 18),
				filter.NewLogicalFilter("country", filter.OpEq, "USA"),
			}),
		}

		result := DeepFindFilter(filters, "age")
		assert.NotNil(t, result)
		assert.Equal(t, "age", result.GetField())

		result = DeepFindFilter(filters, "country")
		assert.NotNil(t, result)
		assert.Equal(t, "country", result.GetField())
	})

	t.Run("deep find - and conditional", func(t *testing.T) {
		filters := []filter.CrudFilter{
			filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpGte, 18),
				filter.NewLogicalFilter("country", filter.OpEq, "USA"),
			}),
		}

		result := DeepFindFilter(filters, "age")
		assert.NotNil(t, result)
		assert.Equal(t, "age", result.GetField())

		result = DeepFindFilter(filters, "country")
		assert.NotNil(t, result)
		assert.Equal(t, "country", result.GetField())
	})

	t.Run("deep find - not conditional", func(t *testing.T) {
		filters := []filter.CrudFilter{
			filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
				filter.NewLogicalFilter("age", filter.OpGte, 18),
			}),
		}

		result := DeepFindFilter(filters, "age")
		assert.NotNil(t, result)
		assert.Equal(t, "age", result.GetField())
	})
}
