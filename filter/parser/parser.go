// Package parser provides components for parsing filters, sorts, and pagination parameters
// from different input formats. It supports both URL query parameters and JSON input formats.
// The package handles validation, type conversion, and complex nested filter structures.
//
// Key components:
// - QueryParamParser: Parses filters from URL query parameters
// - JSONParser: Parses filters from JSON input
// - ParserOptions: Configuration for parsing behavior
package parser

import (
	"go.lumeweb.com/queryutil/filter"
)

// InputFormat defines supported input formats for parsing
type InputFormat int

const (
	FormatJSON InputFormat = iota // JSON format for complex nested filters
	FormatQueryParams             // URL query parameters format
)

// ParserOption configures parser options through functional options pattern
type ParserOption func(*ParserOptions)

// Parser is the main interface for parsing filters, sorts and pagination parameters
// from different input formats. Implementations should handle validation and type
// conversion according to the specified configuration.
type Parser interface {
	ParseFilters() ([]filter.CrudFilter, error)
	ParseSorts(*filter.SortConfig) ([]filter.Sort, error)
	ParsePagination() (filter.Pagination, error)
}

// ApplyParserOptions processes parser options by merging provided options with defaults.
// Uses functional options pattern to configure parsing behavior.
func ApplyParserOptions(opts []ParserOption) *ParserOptions {
	config := filter.ApplyOptions[ParserOptions, ParserOption](DefaultParserOptions(), opts)
	return config
}

// ParserOptions contains configuration for parsing behavior
type ParserOptions struct {
	SearchConfig *filter.GlobalSearchConfig // Configuration for global search parameters
	SortConfig   *filter.SortConfig         // Valid sort fields configuration
	InputFormat  InputFormat                // Preferred input format (JSON or query params)
}

// DefaultParserOptions returns the default parser configuration:
// - InputFormat: FormatQueryParams
func DefaultParserOptions() ParserOptions {
	return ParserOptions{
		InputFormat: FormatQueryParams,
	}
}

// WithSearchConfig sets the global search configuration for the parser
func WithSearchConfig(cfg *filter.GlobalSearchConfig) ParserOption {
	return func(o *ParserOptions) {
		o.SearchConfig = cfg
	}
}

// WithInputFormat specifies the input format to parse from (JSON or query params)
func WithInputFormat(fmt InputFormat) ParserOption {
	return func(o *ParserOptions) {
		o.InputFormat = fmt
	}
}
