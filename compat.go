package queryutil

import (
	"go.lumeweb.com/queryutil/filter/builder"
	"gorm.io/gorm"
)

// ApplyFilters applies filters to a GORM query using the global search configuration.
// If building a filter fails, the returned query carries the error in GORM's
// Error field so later operations cannot run with only part of the filter set.
func ApplyFilters(tx *gorm.DB, filters []CrudFilter, searchConfig *GlobalSearchConfig) *gorm.DB {
	b := builder.NewGORMBuilder(tx, searchConfig)
	result, err := b.Apply(tx, filters)
	if err != nil {
		// Apply may return tx itself when the first filter fails. Keep errors
		// off the caller's handle, which may be shared by unrelated queries.
		result = result.Session(&gorm.Session{})
		result.AddError(err)
	}
	return result
}

// ApplySort applies sort parameters to a GORM query
func ApplySort(tx *gorm.DB, sorts []Sort) *gorm.DB {
	return builder.ApplySorts(tx, sorts)
}

// ApplyPagination applies pagination parameters to a GORM query
// Only applies offset/limit if either value is non-zero
func ApplyPagination(tx *gorm.DB, pagination Pagination) *gorm.DB {
	pagingNotSet := pagination.GetOffset() == 0 && pagination.GetLimit() == 0

	if !pagingNotSet {
		tx = tx.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())
	}
	return tx
}
