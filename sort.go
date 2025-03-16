package queryutil

import (
	"errors"
	"fmt"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"strings"
)

// SortOrder represents the direction of sorting (ascending or descending).
// Valid values are "asc" and "desc".
type SortOrder string

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

// SortConfig defines configuration for field sorting validation.
// It maintains a list of field names that are allowed to be used
// in sort operations, preventing sorting on unauthorized fields.
//
// This is useful for security to ensure users can only sort by
// fields that are intended to be sortable.
type SortConfig struct {
	// Fields that are allowed to be sorted
	SortableFields []string
}

// Sort represents a sort specification
type Sort struct {
	Field string
	Order SortOrder
}

// IsValidField checks if a field name is in the allowed sortable fields
func (c *SortConfig) IsValidField(field string) bool {
	return lo.Contains(c.SortableFields, field)
}

// ErrInvalidSortField indicates an invalid sort field was requested
var ErrInvalidSortField = errors.New("invalid sort field")

// ParseQuerySort parses _sort and _order parameters into Sort structs.
// It supports multiple sort fields and orders, separated by commas.
//
// The _sort parameter specifies fields to sort by.
// The _order parameter specifies corresponding sort directions (asc/desc).
// If _order is omitted, "asc" is used as the default.
//
// If a SortConfig is provided, validates that all sort fields are allowed.
// Returns an error if validation fails or if an invalid order value is provided.
func ParseQuerySort(query map[string][]string, config *SortConfig) ([]Sort, error) {
	sorts, ok := query["_sort"]
	if !ok || len(sorts) == 0 {
		return nil, nil
	}

	if config != nil {
		// Validate all fields
		for _, field := range strings.Split(sorts[0], ",") {
			if !config.IsValidField(field) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidSortField, field)
			}
		}
	}

	orders := query["_order"]
	fields := strings.Split(sorts[0], ",")

	var orderValues []string
	if len(orders) > 0 {
		orderValues = strings.Split(orders[0], ",")
	}

	result := make([]Sort, len(fields))
	for i, field := range fields {
		order := OrderAsc
		if i < len(orderValues) {
			orderStr := strings.ToLower(orderValues[i])
			if orderStr != string(OrderAsc) && orderStr != string(OrderDesc) {
				return nil, NewSortError("_order", fmt.Sprintf("invalid order value: %s", orderValues[i]))
			}
			order = SortOrder(orderStr)
		}

		result[i] = Sort{
			Field: field,
			Order: order,
		}
	}
	return result, nil
}

// ApplySort applies sort specifications to a GORM query.
// For each Sort struct, adds an ORDER BY clause to the query.
// Multiple sorts are applied in the order they appear in the slice.
//
// Example:
//
//	db := gorm.Open(...)
//	sorts := []Sort{
//	    {Field: "name", Order: OrderAsc},
//	    {Field: "created_at", Order: OrderDesc},
//	}
//	query := ApplySort(db, sorts)
//	var users []User
//	query.Find(&users)
func ApplySort(tx *gorm.DB, sorts []Sort) *gorm.DB {
	for _, sort := range sorts {
		tx = tx.Order(fmt.Sprintf("%s %s", sort.Field, sort.Order))
	}
	return tx
}
