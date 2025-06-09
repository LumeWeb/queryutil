package queryutil

import (
	"github.com/samber/lo"
	"go.lumeweb.com/queryutil/filter"
)

// FindFilter extracts the first CrudFilter with a matching field from a slice of filters.
// It performs a shallow search, only examining the top-level filters.
// Returns nil if no matching filter is found.
func FindFilter(filters []filter.CrudFilter, field string) filter.CrudFilter {
	return FindFilterWithOperator(filters, field, "")
}

// FindFilterWithOperator extracts the first CrudFilter with matching field and operator from a slice of filters.
// It performs a shallow search, only examining the top-level filters.
// Returns nil if no matching filter is found.
func FindFilterWithOperator(filters []filter.CrudFilter, field string, operator filter.Operator) filter.CrudFilter {
	return lo.FindOrElse(filters, nil, func(f filter.CrudFilter) bool {
		if f.GetField() != field {
			return false
		}
		if operator != "" {
			return f.GetOperator() == operator
		}
		return true
	})
}

// FindFilters extracts all CrudFilters with a matching field from a slice of filters.
// It performs a shallow search, only examining the top-level filters.
// Returns an empty slice if no matching filters are found.
func FindFilters(filters []filter.CrudFilter, field string) []filter.CrudFilter {
	return FindFiltersWithOperator(filters, field, "")
}

// FindFiltersWithOperator extracts all CrudFilters with matching field and operator from a slice of filters.
// It performs a shallow search, only examining the top-level filters.
// Returns an empty slice if no matching filters are found.
func FindFiltersWithOperator(filters []filter.CrudFilter, field string, operator filter.Operator) []filter.CrudFilter {
	return lo.Filter(filters, func(f filter.CrudFilter, _ int) bool {
		if f.GetField() != field {
			return false
		}
		if operator != "" {
			return f.GetOperator() == operator
		}
		return true
	})
}

// DeepFindFilter recursively searches for the first CrudFilter with a matching field.
// It traverses ConditionalFilters (AND, OR, NOT) to find nested filters.
// Returns nil if no matching filter is found.
func DeepFindFilter(filters []filter.CrudFilter, field string) filter.CrudFilter {
	return DeepFindFilterWithOperator(filters, field, "")
}

// DeepFindFilterWithOperator recursively searches for the first CrudFilter with matching field and operator.
// It traverses ConditionalFilters (AND, OR, NOT) to find nested filters.
// Returns nil if no matching filter is found.
func DeepFindFilterWithOperator(filters []filter.CrudFilter, field string, operator filter.Operator) filter.CrudFilter {
	for _, f := range filters {
		if f.GetField() == field && (operator == "" || f.GetOperator() == operator) {
			return f
		}

		if cf, ok := f.(*filter.ConditionalFilter); ok {
			if found := DeepFindFilterWithOperator(cf.GetFilters(), field, operator); found != nil {
				return found
			}
		}
	}
	return nil
}

// DeepFindFilters recursively searches for all CrudFilters with a matching field.
// It traverses ConditionalFilters (AND, OR, NOT) to find nested filters.
// Returns an empty slice if no matching filters are found.
func DeepFindFilters(filters []filter.CrudFilter, field string) []filter.CrudFilter {
	return DeepFindFiltersWithOperator(filters, field, "")
}

// DeepFindFiltersWithOperator recursively searches for all CrudFilters with matching field and operator.
// It traverses ConditionalFilters (AND, OR, NOT) to find nested filters.
// Returns an empty slice if no matching filters are found.
func DeepFindFiltersWithOperator(filters []filter.CrudFilter, field string, operator filter.Operator) []filter.CrudFilter {
	var results []filter.CrudFilter

	for _, f := range filters {
		if f.GetField() == field && (operator == "" || f.GetOperator() == operator) {
			results = append(results, f)
		}

		if cf, ok := f.(*filter.ConditionalFilter); ok {
			results = append(results, DeepFindFiltersWithOperator(cf.GetFilters(), field, operator)...)
		}
	}

	return results
}
