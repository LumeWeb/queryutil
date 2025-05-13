package filter

import (
	"errors"
	"fmt"
	"strings"

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
	GetOperator() Operator
	// GetValue returns the filter's value
	GetValue() any
	// GetField returns the field name (empty string for conditional filters)
	GetField() string
	// GetFilters returns nested filters for conditional types (empty for logical filters)
	GetFilters() []CrudFilter
	// String returns a human-readable representation of the filter
	String() string
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
	field    string   // Field name to filter on
	operator Operator // Comparison operator
	value    any      // Value to compare against
}

// NewLogicalFilter creates a validated LogicalFilter with immutable values
func NewLogicalFilter(field string, operator Operator, value any) *LogicalFilter {
	validateValue(operator, field, value)
	return &LogicalFilter{
		field:    field,
		operator: operator,
		value:    value,
	}
}

// Value accessor provides read-only access to the value
func (f *LogicalFilter) Value() any {
	return f.value
}

// Field accessor provides read-only access to the field name
func (f *LogicalFilter) Field() string {
	return f.field
}

// Operator accessor provides read-only access to the operator
func (f *LogicalFilter) Operator() Operator {
	return f.operator
}

func (f *LogicalFilter) AcceptVisitor(v Visitor) (Clause, error) {
	return v.VisitLogical(f)
}

func (f *LogicalFilter) GetOperator() Operator {
	return f.operator
}

// GetValue returns the filter's comparison value
func (f *LogicalFilter) GetValue() any {
	return f.value
}

func (f *LogicalFilter) GetField() string {
	return f.Field()
}

func (f *LogicalFilter) GetFilters() []CrudFilter {
	return nil
}

func (f *LogicalFilter) String() string {
	return fmt.Sprintf("%s %s %v", f.Field(), f.Operator(), f.Value())
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

func (f *ConditionalFilter) GetOperator() Operator {
	return Operator(f.Operator)
}

// GetValue returns the nested filters for conditional operators
func (f *ConditionalFilter) GetValue() any {
	return f.Filters
}

func (f *ConditionalFilter) GetField() string {
	return "" // Conditional filters don't have a field
}

func (f *ConditionalFilter) GetFilters() []CrudFilter {
	return f.Filters
}

func (f *ConditionalFilter) String() string {
	subs := make([]string, len(f.Filters))
	for i, sf := range f.Filters {
		subs[i] = sf.String()
	}
	return fmt.Sprintf("%s( %s )", strings.ToUpper(string(f.Operator)), strings.Join(subs, ", "))
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

func (o Operator) String() string {
    return string(o)
}

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
