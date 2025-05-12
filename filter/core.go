package filter

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
)

// Operator defines comparison operations for field values (eq, gt, contains, etc)
type Operator string

// LogicalOperator defines boolean logic combinators (AND/OR/NOT)
type LogicalOperator string

// Visitor implements the visitor pattern for filter expression evaluation
type Visitor interface {
	VisitLogical(*LogicalFilter) (Clause, error)
	VisitConditional(*ConditionalFilter) (Clause, error)
}

// CrudFilter is the core interface for all filter expressions
type CrudFilter interface {
	// AcceptVisitor enables the visitor pattern for filter evaluation
	AcceptVisitor(Visitor) (Clause, error)
	// GetOperator returns the filter's operator (for both logical and conditional filters)
	GetOperator() string
	// GetField returns the field name (empty string for conditional filters)
	GetField() string
	// GetFilters returns nested filters for conditional types (empty for logical filters)
	GetFilters() []CrudFilter
}

// Clause represents a compiled filter expression ready for execution
type Clause interface {
	// Type indicates the clause category (WHERE condition or compound group)
	Type() ClauseType
}

// ClauseType distinguishes between fundamental clause types
type ClauseType int

const (
	WhereClauseType    ClauseType = iota // Direct field comparison clause
	CompoundClauseType                   // Group of clauses combined with AND/OR/NOT
)

// LogicalFilter represents a direct field comparison condition
type LogicalFilter struct {
	Field    string   `json:"field"`    // Field name to filter on
	Operator Operator `json:"operator"` // Comparison operator
	Value    any      `json:"value"`    // Value to compare against
}

func (f *LogicalFilter) AcceptVisitor(v Visitor) (Clause, error) {
	return v.VisitLogical(f)
}

func (f *LogicalFilter) GetOperator() string {
	return string(f.Operator)
}

func (f *LogicalFilter) GetField() string {
	return f.Field
}

func (f *LogicalFilter) GetFilters() []CrudFilter {
	return nil
}

// ConditionalFilter groups multiple filters with boolean logic
type ConditionalFilter struct {
	Operator LogicalOperator `json:"operator"` // AND/OR/NOT combinator
	Filters  []CrudFilter    `json:"value"`    // Nested filter expressions
}

// NewConditionalFilter creates a new ConditionalFilter with validation
func NewConditionalFilter(op LogicalOperator, filters []CrudFilter) CrudFilter {
	if op == LogicalNot && len(filters) != 1 {
		panic("NOT operator requires exactly one filter")
	}
	return &ConditionalFilter{Operator: op, Filters: filters}
}

func (f *ConditionalFilter) AcceptVisitor(v Visitor) (Clause, error) {
	return v.VisitConditional(f)
}

func (f *ConditionalFilter) GetOperator() string {
	return string(f.Operator)
}

func (f *ConditionalFilter) GetField() string {
	return "" // Conditional filters don't have a field
}

func (f *ConditionalFilter) GetFilters() []CrudFilter {
	return f.Filters
}

// GlobalSearchConfig defines settings for full-text search across multiple columns
type GlobalSearchConfig struct {
	SearchableColumns []string // List of columns to search with 'q' parameter
}

// Supported comparison operators
const (
	OpEq           Operator = "eq" // Equal
	OpNe           Operator = "ne" // Not equal
	OpLt           Operator = "lt"
	OpGt           Operator = "gt"
	OpLte          Operator = "lte"
	OpGte          Operator = "gte"
	OpIn           Operator = "in"
	OpNin          Operator = "nin"
	OpContains     Operator = "contains"
	OpContainss    Operator = "containss"
	OpNcontains    Operator = "ncontains"
	OpNcontainss   Operator = "ncontainss"
	OpIna          Operator = "ina"
	OpNina         Operator = "nina"
	OpBetween      Operator = "between"
	OpNbetween     Operator = "nbetween"
	OpNull         Operator = "null"
	OpNnull        Operator = "nnull"
	OpStartswith   Operator = "startswith"
	OpStartswiths  Operator = "startswiths"
	OpNstartswith  Operator = "nstartswith"
	OpNstartswiths Operator = "nstartswiths"
	OpEndswith     Operator = "endswith"
	OpEndswiths    Operator = "endswiths"
	OpNendswith    Operator = "nendswith"
	OpNendswiths   Operator = "nendswiths"
)

var OperatorMap = map[string]Operator{
	"eq":           OpEq,
	"ne":           OpNe,
	"neq":          OpNe, // Alias for ne
	"lt":           OpLt,
	"gt":           OpGt,
	"lte":          OpLte,
	"gte":          OpGte,
	"in":           OpIn,
	"nin":          OpNin,
	"contains":     OpContains,
	"containss":    OpContainss,
	"ncontains":    OpNcontains,
	"ncontainss":   OpNcontainss,
	"between":      OpBetween,
	"nbetween":     OpNbetween,
	"null":         OpNull,
	"nnull":        OpNnull,
	"startswith":   OpStartswith,
	"startswiths":  OpStartswiths,
	"nstartswith":  OpNstartswith,
	"nstartswiths": OpNstartswiths,
	"endswith":     OpEndswith,
	"endswiths":    OpEndswiths,
	"nendswith":    OpNendswith,
	"nendswiths":   OpNendswiths,
	"ina":          OpIna,
	"nina":         OpNina,
	"like":         OpContains, // Alias for contains
}

// OperatorReverseMap provides reverse lookup from Operator to its string representation
var OperatorReverseMap = lo.Invert(OperatorMap)

const (
	LogicalAnd LogicalOperator = "and"
	LogicalOr  LogicalOperator = "or"
	LogicalNot LogicalOperator = "not"
)

var ErrUnsupportedOperator = errors.New("unsupported operator")

// RequiresArray checks if the operator requires an array value
func (o Operator) RequiresArray() bool {
	return o == OpIn || o == OpNin || o == OpIna || o == OpNina ||
		o == OpBetween || o == OpNbetween
}

func ParseOperator(op string) (Operator, error) {
	if operator, ok := OperatorMap[op]; ok {
		return operator, nil
	}
	return "", fmt.Errorf("%w: %s", ErrUnsupportedOperator, op)
}

// ApplyOptions applies functional options to a configuration struct.
// Used internally to merge default configs with user-provided options.
// Example: Merging parser/builder defaults with runtime configurations
func ApplyOptions[T any, O ~func(*T)](defaults T, opts []O) *T {
	result := defaults
	for _, opt := range opts {
		opt(&result)
	}
	return &result
}
