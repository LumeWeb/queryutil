package queryutil

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
	"golang.org/x/exp/constraints"
)

// Re-exported types
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

var (
	NewPagination     = filter.NewPagination
	CreatePage        = filter.CreatePage
	DefaultPagination = filter.DefaultPagination 
	LargePagination   = filter.LargePagination
	XLargePagination  = filter.XLargePagination
	XXLargePagination = filter.XXLargePagination
)

// Re-exported constants

// Filter operators
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

// Sort directions
const (
	OrderAsc  = filter.OrderAsc
	OrderDesc = filter.OrderDesc
)

// Re-exported variables
var (
	OperatorMap = filter.OperatorMap
)

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

// Core functions
var (
	ParseQuerySort       = filter.ParseQuerySort
	GetResultCount       = filter.GetResultCount
	FormatContentRange   = filter.FormatContentRange
	NewPaginationError   = filter.NewPaginationError
	NewSortError         = filter.NewSortError
	NewFilterError       = filter.NewFilterError
	ParseQueryPagination = filter.ParseQueryPagination
)

// Filter constructors
var (
	NewLogicalFilter     = filter.NewLogicalFilter
	NewConditionalFilter = filter.NewConditionalFilter
)

// Core predicates
var (
	Equal          = filter.Equal
	NotEqual       = filter.NotEqual
	GreaterThan    = filter.GreaterThan
	GreaterOrEqual = filter.GreaterOrEqual
	LessThan       = filter.LessThan
	LessOrEqual    = filter.LessOrEqual
	Search         = filter.Search
	Contains       = filter.Contains
	In             = filter.In
	Between        = filter.Between
)

// Field operations
var (
	FieldNotBetween = filter.FieldNotBetween
	FieldNotIn      = filter.FieldNotIn
	FieldIsNull     = filter.FieldIsNull
	FieldIsNotNull  = filter.FieldIsNotNull
	FieldIn         = filter.FieldIn
	FieldEqual      = filter.FieldEqual
	FieldNotEqual   = filter.FieldNotEqual
	FieldGt         = filter.FieldGt
	FieldLt         = filter.FieldLt
	FieldGte        = filter.FieldGte
	FieldLte        = filter.FieldLte
	FieldContains   = filter.FieldContains
	FieldBetween    = filter.FieldBetween
)

// Logical combinators
var (
	And = filter.And
	Or  = filter.Or
	Not = filter.Not
)

// Merge strategies
var (
	MergeStrategyOverride = filter.MergeStrategyOverride
	MergeStrategyDeep     = filter.MergeStrategyDeep
	MergeStrategyAppend   = filter.MergeStrategyAppend
	MergeFilters          = filter.MergeFilters
)

// Type-safe field helpers
var (
	StringField = filter.StringField
	BoolField   = filter.BoolField
	TimeField   = filter.TimeField
)

// NumberField creates a type-safe query builder helper for numeric fields.
// Example: queryutil.NumberField[int]("age").Gt(18)
func NumberField[T constraints.Integer | constraints.Float](field string) filter.NumberFieldHelper[T] {
	return filter.NumberField[T](field)
}
