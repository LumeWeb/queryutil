// Package serializer provides components for serializing filters, sorts, and pagination parameters
// to URL query parameters. It supports complex nested filter structures and converts them
// back to query string format. This is the opposite functionality of the parser package.
//
// Key components:
// - QuerySerializer: Main interface for serializing to query parameters
// - QueryParamSerializer: Implementation for URL query parameter format
// - SerializerOptions: Configuration for serialization behavior
package serializer

import (
	"net/url"

	"go.lumeweb.com/queryutil/filter"
)

// SerializerOption configures serializer options through functional options pattern
type SerializerOption func(*SerializerOptions)

// QuerySerializer is the main interface for serializing filters, sorts and pagination parameters
// to URL query parameters. Implementations should handle nested structures and proper formatting.
type QuerySerializer interface {
	SerializeFilters([]filter.CrudFilter) (url.Values, error)
	SerializeSorts([]filter.Sort) (url.Values, error)
	SerializePagination(filter.Pagination) (url.Values, error)
}

// SerializerOptions contains configuration for serialization behavior
type SerializerOptions struct {
	FilterPrefix string // Prefix for filter parameters (default: "filters")
	SortPrefix   string // Prefix for sort parameters (default: "sort")
}

// DefaultSerializerOptions returns the default serializer configuration:
// - FilterPrefix: "filters"
// - SortPrefix: "sort"
func DefaultSerializerOptions() SerializerOptions {
	return SerializerOptions{
		FilterPrefix: "filters",
		SortPrefix:   "sort",
	}
}

// WithFilterPrefix sets the prefix for filter parameters
func WithFilterPrefix(prefix string) SerializerOption {
	return func(o *SerializerOptions) {
		o.FilterPrefix = prefix
	}
}

// WithSortPrefix sets the prefix for sort parameters
func WithSortPrefix(prefix string) SerializerOption {
	return func(o *SerializerOptions) {
		o.SortPrefix = prefix
	}
}

// ApplySerializerOptions processes serializer options by merging provided options with defaults.
// Uses functional options pattern to configure serialization behavior.
func ApplySerializerOptions(opts ...SerializerOption) *SerializerOptions {
	config := DefaultSerializerOptions()
	for _, opt := range opts {
		opt(&config)
	}
	return &config
}
