package queryutil

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
)

type (
	// RequestParser re-exports the parser.Parser interface
	RequestParser      = parser.Parser
	Sort               = filter.Sort
	SortConfig         = filter.SortConfig
	SortOrder          = filter.SortOrder
	CrudFilter         = filter.CrudFilter
	GlobalSearchConfig = filter.GlobalSearchConfig
	Pagination         = filter.Pagination
	PaginationError    = filter.PaginationError
	Filter             = filter.LogicalFilter
	Operator           = filter.Operator
	LogicalOperator    = filter.LogicalOperator
)

// Re-export filter operators as typed constants
const (
	OpEq           = filter.OpEq
	OpNe           = filter.OpNe
	OpLt           = filter.OpLt
	OpGt           = filter.OpGt
	OpLte          = filter.OpLte
	OpGte          = filter.OpGte
	OpContains     = filter.OpContains
	OpContainss    = filter.OpContainss
	OpNcontains    = filter.OpNcontains
	OpNcontainss   = filter.OpNcontainss
	OpIn           = filter.OpIn
	OpNin          = filter.OpNin
	OpIna          = filter.OpIna
	OpNina         = filter.OpNina
	OpBetween      = filter.OpBetween
	OpNbetween     = filter.OpNbetween
	OpNull         = filter.OpNull
	OpNnull        = filter.OpNnull
	OpStartswith   = filter.OpStartswith
	OpStartswiths  = filter.OpStartswiths
	OpNstartswith  = filter.OpNstartswith
	OpNstartswiths = filter.OpNstartswiths
	OpEndswith     = filter.OpEndswith
	OpEndswiths    = filter.OpEndswiths
	OpNendswith    = filter.OpNendswith
	OpNendswiths   = filter.OpNendswiths
)

// Re-export operator map as a variable
var OperatorMap = filter.OperatorMap

// QueryBuilder defines the interface for building query clauses
type QueryBuilder interface {
	Apply(tx any, filters []filter.CrudFilter) (any, error)
}

// Query Builder API provides a fluent, type-safe interface for constructing filters:
// 
// Core Predicates:
// - Equal(field, value)        // Field equals value
// - NotEqual(field, value)     // Field does not equal value
// - GreaterThan(field, value)  // Field > value
// - GreaterOrEqual(field, val) // Field >= value
// - LessThan(field, value)     // Field < value
// - LessOrEqual(field, value)  // Field <= value
// - Contains(field, substring) // Field contains substring (case-insensitive)
// - In(field, values...)       // Field exists in values list
// - Between(field, min, max)   // Field between min/max (inclusive)
// - IsNull(field)              // Field is NULL
// - IsNotNull(field)           // Field is NOT NULL
//
// Logical Combinators:
// - And(filters...)            // Combine with AND logic
// - Or(filters...)             // Combine with OR logic
// - Not(filter)                // Negate filter with NOT
// - MergeFilters(strategy)     // Combine filter sets with merge strategies
//
// Type-Safe Field Helpers:
// - StringField(field).Eq(val)         // Chainable string field operations
// - NumberField[T](field).Between(...) // Type-checked numeric operations
// - BoolField(field).Eq(true/false)    // Boolean-specific operations
// - TimeField(field).After(time.Time)  // Temporal operations
//
// Merge Strategies:
// - MergeStrategyOverride  // Replace existing field operators
// - MergeStrategyDeep      // Deep merge nested conditionals
// - MergeStrategyAppend    // Simply append all filters
//
// Example:
// filters := And(
//     Or(
//         Equal("status", "active"),
//         GreaterThan("login_count", 10)
//     ),
//     Not(Equal("deleted", true)),
//     StringField("email").Contains("@company.com")
// )

// ParseFromSource is the unified parsing entry point using the RequestParser interface
func ParseFromSource(parser RequestParser) ([]CrudFilter, []Sort, Pagination, error) {
	filters, err := parser.ParseFilters()
	if err != nil {
		return nil, nil, Pagination{}, fmt.Errorf("error parsing filters: %w", err)
	}

	sorts, err := parser.ParseSorts(nil) // TODO: Pass appropriate SortConfig
	if err != nil {
		return filters, nil, Pagination{}, fmt.Errorf("error parsing sorts: %w", err)
	}

	pagination, err := parser.ParsePagination()
	if err != nil {
		return filters, sorts, Pagination{}, fmt.Errorf("error parsing pagination: %w", err)
	}

	return filters, sorts, pagination, nil
}

// Core predicate functions
var (
	ParseQuerySort        = filter.ParseQuerySort
	GetResultCount        = filter.GetResultCount
	FormatContentRange    = filter.FormatContentRange
	NewPaginationError    = filter.NewPaginationError
	NewSortError          = filter.NewSortError
	NewFilterError        = filter.NewFilterError
	ParseQueryPagination  = filter.ParseQueryPagination
	NewLogicalFilter      = filter.NewLogicalFilter
	NewConditionalFilter  = filter.NewConditionalFilter
	
	// Core predicates
	Equal          = filter.Equal
	NotEqual       = filter.NotEqual
	GreaterThan    = filter.GreaterThan
	GreaterOrEqual = filter.GreaterOrEqual 
	LessThan       = filter.LessThan
	LessOrEqual    = filter.LessOrEqual
	Contains       = filter.Contains
	In             = filter.In
	Between        = filter.Between
	FieldNotBetween = filter.FieldNotBetween
	FieldNotIn     = filter.FieldNotIn
	FieldIsNull    = filter.FieldIsNull
	FieldIsNotNull = filter.FieldIsNotNull
	
	// Logical combinators
	And = filter.And
	Or  = filter.Or
	Not = filter.Not
	
	// Merge strategies
	MergeStrategyOverride = filter.MergeStrategyOverride
	MergeStrategyDeep     = filter.MergeStrategyDeep
	MergeStrategyAppend   = filter.MergeStrategyAppend
	MergeFilters          = filter.MergeFilters
	
	// Type-safe field helpers
	StringField  = filter.StringField
	NumberField  = filter.NumberField
	BoolField    = filter.BoolField
	TimeField    = filter.TimeField
	
	// Field-specific helpers
	FieldIn        = filter.FieldIn
	FieldEqual     = filter.FieldEqual
	FieldNotEqual  = filter.FieldNotEqual
	FieldGt        = filter.FieldGt
	FieldLt        = filter.FieldLt
	FieldGte       = filter.FieldGte
	FieldLte       = filter.FieldLte
	FieldContains  = filter.FieldContains
	FieldBetween   = filter.FieldBetween
)
