package filter

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidQuery indicates that a filter or sort does not satisfy a QueryPolicy.
var ErrInvalidQuery = errors.New("invalid query")

// FieldRule permits only the listed comparison operators on a field.
// Convert may validate and replace a value before it is sent to the query builder.
// For array operators, Convert receives the whole array; callers decide how
// to convert its elements. Return an error for malformed values.
type FieldRule struct {
	Operators []Operator
	Convert   func(operator Operator, value any) (any, error)
}

// QueryPolicy is an opt-in boundary for untrusted filters and sorts. Call
// Normalize on every entry point before applying either to a database query.
// Fields and SortFields must be explicitly populated; empty lists allow nothing.
// Do not mutate a policy's maps or slices while it is in use.
type QueryPolicy struct {
	Fields     map[string]FieldRule
	SortFields []string
}

// Normalize validates the entire filter tree and sorts and returns a new tree.
// It accepts only built-in filter nodes so custom visitors cannot bypass checks.
// Original filters are not modified. An error means no query should be built.
func (p QueryPolicy) Normalize(filters []CrudFilter, sorts []Sort) ([]CrudFilter, error) {
	for _, s := range sorts {
		if !containsField(p.SortFields, s.Field) || (s.Order != OrderAsc && s.Order != OrderDesc) {
			return nil, fmt.Errorf("%w: sort %q %q", ErrInvalidQuery, s.Field, s.Order)
		}
	}
	return p.normalize(filters, 0)
}

func containsField(fields []string, field string) bool {
	for _, allowed := range fields {
		if allowed == field {
			return true
		}
	}
	return false
}

func (p QueryPolicy) normalize(filters []CrudFilter, depth int) ([]CrudFilter, error) {
	if depth > 32 {
		return nil, fmt.Errorf("%w: filter nesting too deep", ErrInvalidQuery)
	}
	result := make([]CrudFilter, 0, len(filters))
	for _, item := range filters {
		switch f := item.(type) {
		case *LogicalFilter:
			if f == nil {
				return nil, fmt.Errorf("%w: nil filter", ErrInvalidQuery)
			}
			operator := f.Operator()
			// The search builder interprets q[eq] as a contains search. Normalize
			// that legacy spelling before checking the policy's allowed operators.
			if f.Field() == "q" && operator == OpEq {
				operator = OpContains
			}
			if strings.Contains(f.Field(), ".") {
				if err := ValidateJSONPathField(f.Field()); err != nil {
					return nil, err
				}
			}
			rule, ok := p.Fields[f.Field()]
			if !ok || !supportedPolicyOperator(operator) || !containsOperator(rule.Operators, operator) ||
				(f.Field() == "q" && operator != OpContains) ||
				(strings.Contains(f.Field(), ".") && unsupportedJSONOperator(operator)) {
				return nil, fmt.Errorf("%w: field %q operator %q", ErrInvalidQuery, f.Field(), f.Operator())
			}
			value := f.Value()
			if rule.Convert != nil {
				var err error
				value, err = rule.Convert(operator, value)
				if err != nil {
					return nil, fmt.Errorf("%w: field %q: %v", ErrInvalidQuery, f.Field(), err)
				}
			}
			if operator.RequiresArray() {
				values, ok := value.([]any)
				if !ok || len(values) == 0 || ((operator == OpBetween || operator == OpNbetween) && len(values) != 2) {
					return nil, fmt.Errorf("%w: field %q: invalid array value", ErrInvalidQuery, f.Field())
				}
			}
			// The constructor validates values by panicking; contain that legacy API
			// at this boundary so direct callers receive an error instead.
			node, err := NewLogicalFilterChecked(f.Field(), operator, value)
			if err != nil {
				return nil, err
			}
			result = append(result, node)
		case *ConditionalFilter:
			if f == nil || len(f.Filters) == 0 {
				return nil, fmt.Errorf("%w: empty or nil group", ErrInvalidQuery)
			}
			if (f.Operator != LogicalAnd && f.Operator != LogicalOr && f.Operator != LogicalNot) || (f.Operator == LogicalNot && len(f.Filters) != 1) {
				return nil, fmt.Errorf("%w: logical operator %q", ErrInvalidQuery, f.Operator)
			}
			children, err := p.normalize(f.Filters, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, &ConditionalFilter{Operator: f.Operator, Filters: children})
		default:
			return nil, fmt.Errorf("%w: unsupported filter type %T", ErrInvalidQuery, item)
		}
	}
	return result, nil
}

// NewLogicalFilterChecked adapts the legacy panic-based constructor for
// untrusted operands. It returns ErrInvalidQuery on validation failures.
func NewLogicalFilterChecked(field string, op Operator, value any) (node *LogicalFilter, err error) {
	defer func() {
		if r := recover(); r != nil {
			node = nil
			err = fmt.Errorf("%w: field %q: invalid value: %v", ErrInvalidQuery, field, r)
		}
	}()
	return NewLogicalFilter(field, op, value), nil
}

// Dotted fields may be interpreted as JSON paths by the query builder. Reject
// operators its JSON path implementation cannot handle, even when allowlisted.
func unsupportedJSONOperator(op Operator) bool {
	switch op {
	case OpIn, OpNin, OpBetween, OpNbetween:
		return true
	}
	return false
}

func containsOperator(operators []Operator, op Operator) bool {
	for _, allowed := range operators {
		if allowed == op {
			return true
		}
	}
	return false
}

// scalarOperators is the set supported by the scalar query builder. Keep it
// private so callers cannot mutate the policy's supported-operator set.
var scalarOperators = [...]Operator{
	OpEq, OpNe, OpLt, OpGt, OpLte, OpGte,
	OpIn, OpNin, OpBetween, OpNbetween,
	OpNull, OpNnull, OpContains, OpContainss,
	OpNcontains, OpNcontainss, OpStartswith, OpStartswiths,
	OpNstartswith, OpNstartswiths, OpEndswith, OpEndswiths,
	OpNendswith, OpNendswiths,
}

// ScalarOperators returns a fresh copy of every operator supported for scalar
// fields. Use a narrower list when the field should permit fewer operations.
func ScalarOperators() []Operator {
	operators := make([]Operator, len(scalarOperators))
	copy(operators, scalarOperators[:])
	return operators
}

// AllScalarFieldRules returns a rule for each field with every scalar
// comparison operator enabled. Use it only when each field should permit all
// operators; declare a narrower FieldRule otherwise. Sorts remain opt-in.
// Each call and field gets its own operator slice to prevent accidental sharing.
func AllScalarFieldRules(fields ...string) map[string]FieldRule {
	rules := make(map[string]FieldRule, len(fields))
	for _, field := range fields {
		rules[field] = FieldRule{Operators: ScalarOperators()}
	}
	return rules
}

// comparableOperators excludes pattern matching and null checks.
var comparableOperators = [...]Operator{
	OpEq, OpNe, OpLt, OpGt, OpLte, OpGte,
	OpIn, OpNin, OpBetween, OpNbetween,
}

// ComparableOperators returns a fresh copy of the equality, range, IN, and
// BETWEEN operators. It excludes pattern and null operators.
func ComparableOperators() []Operator {
	operators := make([]Operator, len(comparableOperators))
	copy(operators, comparableOperators[:])
	return operators
}

// ComparableFieldRules returns independent comparable rules for each field.
// Use it only when every listed operator is appropriate for those fields.
// Sort fields must still be explicitly allowlisted on the QueryPolicy.
func ComparableFieldRules(fields ...string) map[string]FieldRule {
	rules := make(map[string]FieldRule, len(fields))
	for _, field := range fields {
		rules[field] = FieldRule{Operators: ComparableOperators()}
	}
	return rules
}

// Only operators implemented by the scalar query builder are valid even if a
// policy accidentally lists an unsupported operator.
func supportedPolicyOperator(op Operator) bool {
	for _, supported := range scalarOperators {
		if supported == op {
			return true
		}
	}
	return false
}
