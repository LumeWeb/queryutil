package filter

import "fmt"

// ValidationError is the base error type for all validation errors.
// It includes the field name that failed validation and a descriptive message.
// This is the parent type for more specific validation errors like
// PaginationError, SortError, and FilterError.
// ValidationError is returned for invalid filter/sort/pagination parameters.
// Embedded in specific error types to preserve field context.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// PaginationError represents pagination-specific validation errors.
// Used when pagination parameters (_start, _end) are invalid.
// PaginationError indicates invalid _start/_end parameter values
type PaginationError struct {
	ValidationError  // Embeds field name and message
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
