package queryutil

import "fmt"

// ValidationError is the base error type for all validation errors.
// It includes the field name that failed validation and a descriptive message.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// PaginationError represents pagination-specific validation errors.
// Used when pagination parameters (_start, _end) are invalid.
type PaginationError struct {
	ValidationError
}

// NewPaginationError creates a new PaginationError
func NewPaginationError(field, message string) *PaginationError {
	return &PaginationError{
		ValidationError: ValidationError{
			Field:   field,
			Message: message,
		},
	}
}

// SortError represents sort-specific validation errors.
// Used when sort parameters (_sort, _order) are invalid.
type SortError struct {
	ValidationError
}

// NewSortError creates a new SortError
func NewSortError(field, message string) *SortError {
	return &SortError{
		ValidationError: ValidationError{
			Field:   field,
			Message: message,
		},
	}
}

// FilterError represents filter-specific validation errors.
// Used when filter operators or values are invalid.
type FilterError struct {
	ValidationError
}

// NewFilterError creates a new FilterError
func NewFilterError(field, message string) *FilterError {
	return &FilterError{
		ValidationError: ValidationError{
			Field:   field,
			Message: message,
		},
	}
}
