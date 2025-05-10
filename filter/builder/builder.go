// Package builder provides database-agnostic query building capabilities for filters.
// It translates filter expressions into database-specific query constructs using a visitor pattern.
// Currently supports GORM implementations with extensibility for other ORMs/database drivers.
package builder

import (
	"go.lumeweb.com/queryutil/filter"
)

// BuilderType defines supported database driver types
type BuilderType int
// BuilderOption configures builder options through functional options pattern
type BuilderOption func(options *BuilderOptions)

const (
	GORM BuilderType = iota // GORM builder for SQL databases via gorm.io
)

// BuilderOptions contains configuration for query builder implementations
type BuilderOptions struct {
	BuilderType BuilderType // Database driver type to build for
}

// DefaultBuilderOptions returns the default configuration with GORM as the builder type
func DefaultBuilderOptions() BuilderOptions {
	return BuilderOptions{
		BuilderType: GORM,
	}
}

// QueryBuilder is the core interface for converting filter expressions to database-specific clauses.
// Implementations should handle both logical (field comparisons) and conditional (AND/OR/NOT) filters.
type QueryBuilder interface {
	VisitLogical(*filter.LogicalFilter) (filter.Clause, error)
	VisitConditional(*filter.ConditionalFilter) (filter.Clause, error)
}

// ApplyBuilderOptions processes driver options using default values as base configuration.
// Merges provided options with defaults to create final builder configuration.
func ApplyBuilderOptions(opts []BuilderOption) *BuilderOptions {
	config := filter.ApplyOptions[BuilderOptions, BuilderOption](DefaultBuilderOptions(), opts)
	return config
}
