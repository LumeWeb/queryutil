package queryutil

import (
	"github.com/samber/lo"
	"go.lumeweb.com/queryutil/filter"
)

// FindFilter extracts the first CrudFilter with a matching field from a slice of filters.
// It performs a shallow search, only examining the top-level filters.
// Returns nil if no matching filter is found.
func FindFilter(filters []filter.CrudFilter, field string) filter.CrudFilter {
	return lo.FindOrElse(filters, nil, func(f filter.CrudFilter) bool {
		return f.GetField() == field
	})
}

// FindFilters extracts all CrudFilters with a matching field from a slice of filters.
// It performs a shallow search, only examining the top-level filters.
// Returns an empty slice if no matching filters are found.
func FindFilters(filters []filter.CrudFilter, field string) []filter.CrudFilter {
	return lo.Filter(filters, func(f filter.CrudFilter, _ int) bool {
		return f.GetField() == field
	})
}

// DeepFindFilter recursively searches for the first CrudFilter with a matching field.
// It traverses ConditionalFilters (AND, OR, NOT) to find nested filters.
// Returns nil if no matching filter is found.
func DeepFindFilter(filters []filter.CrudFilter, field string) filter.CrudFilter {
	for _, f := range filters {
		if f.GetField() == field {
			return f
		}

		if cf, ok := f.(*filter.ConditionalFilter); ok {
			if found := DeepFindFilter(cf.GetFilters(), field); found != nil {
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
	var results []filter.CrudFilter

	for _, f := range filters {
		if f.GetField() == field {
			results = append(results, f)
		}

		if cf, ok := f.(*filter.ConditionalFilter); ok {
			results = append(results, DeepFindFilters(cf.GetFilters(), field)...)
		}
	}

	return results
}
