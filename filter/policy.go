package filter

import (
	"errors"
	"fmt"
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
			rule, ok := p.Fields[f.Field()]
			if !ok || !supportedPolicyOperator(f.Operator()) || !containsOperator(rule.Operators, f.Operator()) {
				return nil, fmt.Errorf("%w: field %q operator %q", ErrInvalidQuery, f.Field(), f.Operator())
			}
			value := f.Value()
			if rule.Convert != nil {
				var err error
				value, err = rule.Convert(f.Operator(), value)
				if err != nil {
					return nil, fmt.Errorf("%w: field %q: %v", ErrInvalidQuery, f.Field(), err)
				}
			}
			if f.Operator().RequiresArray() {
				values, ok := value.([]any)
				if !ok || len(values) == 0 || ((f.Operator() == OpBetween || f.Operator() == OpNbetween) && len(values) != 2) {
					return nil, fmt.Errorf("%w: field %q: invalid array value", ErrInvalidQuery, f.Field())
				}
			}
			// The constructor validates values by panicking; contain that legacy API
			// at this boundary so direct callers receive an error instead.
			node, err := checkedLogicalFilter(f.Field(), f.Operator(), value)
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

func checkedLogicalFilter(field string, op Operator, value any) (node *LogicalFilter, err error) {
	defer func() {
		if r := recover(); r != nil {
			node = nil
			err = fmt.Errorf("%w: field %q: invalid value: %v", ErrInvalidQuery, field, r)
		}
	}()
	return NewLogicalFilter(field, op, value), nil
}

func containsOperator(operators []Operator, op Operator) bool {
	for _, allowed := range operators {
		if allowed == op {
			return true
		}
	}
	return false
}

// Only operators implemented by the query builder are valid even if a policy
// accidentally lists an unsupported operator.
func supportedPolicyOperator(op Operator) bool {
	switch op {
	case OpEq, OpNe, OpLt, OpGt, OpLte, OpGte, OpIn, OpNin,
		OpBetween, OpNbetween, OpNull, OpNnull,
		OpContains, OpContainss, OpNcontains, OpNcontainss,
		OpStartswith, OpStartswiths, OpNstartswith, OpNstartswiths,
		OpEndswith, OpEndswiths, OpNendswith, OpNendswiths:
		return true
	}
	return false
}
