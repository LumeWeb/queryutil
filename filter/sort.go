package filter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// SortOrder defines valid sorting directions
type SortOrder string

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

// Sort parameter names used in URL query strings
const (
	SortParamName  = "_sort"
	OrderParamName = "_order"
)

// Sort represents a field sorting configuration
type Sort struct {
	Field string    // Field name to sort by
	Order SortOrder // Sorting direction (asc/desc)
}

// SortConfig defines valid sort fields and order directions
type SortConfig struct {
	SortableFields []string // Whitelist of fields allowed for sorting
}

func (c *SortConfig) IsValidField(field string) bool {
	return lo.Contains(c.SortableFields, field)
}

var ErrInvalidSortField = errors.New("invalid sort field")

// ParseQuerySort extracts sorting parameters from URL query values.
// Validates against SortConfig if provided.
// Example URL: ?_sort=name,age&_order=asc,desc
func ParseQuerySort(query map[string][]string, config *SortConfig) ([]Sort, error) {
	sorts, ok := query["_sort"]
	if !ok || len(sorts) == 0 {
		return nil, nil
	}

	// Skip validation if no config provided
	if config != nil && config.SortableFields != nil {
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
