// Package queryutil implements Refine-compatible query parsing utilities for filtering, sorting, and pagination.
//
// This package provides functionality to parse and handle query parameters in the format
// expected by Refine's Simple REST Data Provider specification. It includes support for:
//   - Filtering with various operators (equals, not equals, gte, lte, contains)
//   - Sorting with multiple fields and directions
//   - Pagination with server/client modes
//   - GORM integration helpers
package queryutil

import (
	"errors"
	"fmt"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"slices"
	"strings"
)

// FilterOperator represents supported filter operations.
// Available operators are:
//   - "" (equals, default)
//   - "ne" (not equals)
//   - "gte" (greater than or equal)
//   - "lte" (less than or equal)
//   - "like" (contains)
type FilterOperator string

const (
	OperatorEquals    FilterOperator = ""     // default, no suffix
	OperatorNotEquals FilterOperator = "ne"   // _ne
	OperatorGTE       FilterOperator = "gte"  // _gte
	OperatorLTE       FilterOperator = "lte"  // _lte
	OperatorContains  FilterOperator = "like" // _like
)

// ErrUnsupportedOperator indicates an unsupported filter operator was used
var ErrUnsupportedOperator = errors.New("unsupported operator")

// Filter represents a single filter condition with a field name,
// operator, and value. The Field can be a special "q" parameter for
// global search across multiple columns.
type Filter struct {
	Field    string
	Operator FilterOperator
	Value    interface{}
}

// ParseQueryFilters parses URL query parameters into Filter structs.
// It handles both simple equality filters (field=value) and operator-based
// filters (field_operator=value). The special "q" parameter is preserved
// for global search functionality.
//
// Supported operators are:
//   - _ne (not equals)
//   - _gte (greater than or equal)
//   - _lte (less than or equal)
//   - _like (contains)
//
// Returns an error if an unsupported operator is used.
func ParseQueryFilters(query map[string][]string) ([]Filter, error) {
	// Get sorted keys to ensure consistent order
	keys := lo.Keys(query)
	slices.Sort(keys)

	var err error
	filters := lo.FilterMap(keys, func(key string, _ int) (Filter, bool) {
		values := query[key]
		if len(values) == 0 || strings.HasPrefix(key, "_") {
			return Filter{}, false
		}

		// Special handling for global search parameter
		if key == "q" {
			return Filter{
				Field:    "q",
				Operator: OperatorEquals,
				Value:    values[0],
			}, true
		}

		// Handle operators
		if strings.Contains(key, "_") {
			parts := strings.Split(key, "_")
			if len(parts) != 2 {
				return Filter{}, false
			}

			field := parts[0]
			operator := parts[1]

			// Check for unsupported logical operators
			if operator == "or" || operator == "and" {
				err = fmt.Errorf("%w: %s", ErrUnsupportedOperator, operator)
				return Filter{}, false
			}

			// Validate operator is supported
			switch FilterOperator(operator) {
			case OperatorNotEquals, OperatorGTE, OperatorLTE, OperatorContains:
				return Filter{
					Field:    field,
					Operator: FilterOperator(operator),
					Value:    values[0],
				}, true
			default:
				err = fmt.Errorf("%w: %s", ErrUnsupportedOperator, operator)
				return Filter{}, false
			}
		}

		// Default to equals operator
		return Filter{
			Field:    key,
			Operator: OperatorEquals,
			Value:    values[0],
		}, true
	})

	if err != nil {
		return nil, err
	}

	return filters, nil
}

// GlobalSearchConfig defines configuration for handling the global search 'q' parameter.
// It specifies which database columns should be included in the LIKE query when
// performing a global search.
type GlobalSearchConfig struct {
	// Columns to search in with LIKE queries
	SearchableColumns []string
}

// ApplyFilters applies a slice of Filter structs to a GORM query.
// It handles both regular filters and global search if a SearchConfig is provided.
// For global search (q parameter), it creates a combined LIKE query across all
// configured searchable columns.
//
// Regular filters are applied with their respective operators:
//   - equals: field = value
//   - not equals: field <> value
//   - gte: field >= value
//   - lte: field <= value
//   - contains: field LIKE %value%
func ApplyFilters(tx *gorm.DB, filters []Filter, searchConfig *GlobalSearchConfig) *gorm.DB {
	for _, filter := range filters {
		// Special handling for global search
		if filter.Field == "q" && searchConfig != nil {
			conditions := make([]string, len(searchConfig.SearchableColumns))
			values := make([]interface{}, len(searchConfig.SearchableColumns))
			
			for i, column := range searchConfig.SearchableColumns {
				conditions[i] = column + " LIKE ?"
				values[i] = fmt.Sprintf("%%%v%%", filter.Value)
			}
			
			tx = tx.Where(strings.Join(conditions, " OR "), values...)
			continue
		}

		switch filter.Operator {
		case OperatorEquals:
			tx = tx.Where(filter.Field+" = ?", filter.Value)
		case OperatorNotEquals:
			tx = tx.Where(filter.Field+" <> ?", filter.Value)
		case OperatorGTE:
			tx = tx.Where(filter.Field+" >= ?", filter.Value)
		case OperatorLTE:
			tx = tx.Where(filter.Field+" <= ?", filter.Value)
		case OperatorContains:
			tx = tx.Where(filter.Field+" LIKE ?", fmt.Sprintf("%%%v%%", filter.Value))
		}
	}
	return tx
}
