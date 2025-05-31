package queryutil

import (
	"go.lumeweb.com/queryutil/filter/builder"
	"gorm.io/gorm"
)

// ApplyFilters applies filters to a GORM query using the global search configuration
func ApplyFilters(tx *gorm.DB, filters []CrudFilter, searchConfig *GlobalSearchConfig) *gorm.DB {
	b := builder.NewGORMBuilder(tx, searchConfig)
	result, _ := b.Apply(tx, filters)
	return result
}

// ApplySort applies sort parameters to a GORM query
func ApplySort(tx *gorm.DB, sorts []Sort) *gorm.DB {
	b := builder.NewGORMBuilder(tx, nil)
	return b.ApplySorts(tx, sorts)
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
