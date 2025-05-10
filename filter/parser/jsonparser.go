package parser

import (
	"encoding/json"
	"fmt"
	"go.lumeweb.com/queryutil/filter"
)

// JSONParser handles parsing of filters from JSON input.
// Supports complex nested structures with logical operators (AND/OR/NOT).
// Validates operator syntax and value types according to filter specifications.
type JSONParser struct {
	input  string         // Raw JSON input to parse
	config *ParserOptions // Parser configuration
}

// NewJSONParser creates a new parser instance for JSON input.
// Uses functional options to configure parsing behavior:
//   - WithSearchConfig() for global search settings
//   - WithInputFormat() to explicitly specify JSON format
func NewJSONParser(input string, opts ...ParserOption) *JSONParser {
	p := &JSONParser{
		input:  input,
		config: ApplyParserOptions(opts),
	}
	return p
}

// ParseFilters converts JSON input into filter clauses with validation.
// Handles both logical and conditional filters with nested structures.
// The JSON input should be an array of filter objects with either:
// - Logical filters: {"field", "operator", "value"} 
// - Conditional filters: {"operator": "and/or/not", "value": [nested filters]}
// Returns parsed filters or validation error if JSON is malformed
func (p *JSONParser) ParseFilters() ([]filter.CrudFilter, error) {
	rawJSON := p.input

	var filters []filter.CrudFilter
	var rawFilters []json.RawMessage

	// First parse into raw messages to preserve nested structure
	if err := json.Unmarshal([]byte(rawJSON), &rawFilters); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	for _, rf := range rawFilters {
		lf := &filter.LogicalFilter{}
		if err := json.Unmarshal(rf, lf); err == nil && lf.Field != "" {
			// Validate operator
			if _, err := filter.ParseOperator(string(lf.Operator)); err != nil {
				return nil, fmt.Errorf("invalid operator %q: %w", lf.Operator, err)
			}
			filters = append(filters, lf)
			continue
		}

		// Try conditional filter
		var cfStruct struct {
			Operator filter.LogicalOperator `json:"operator"`
			Value    []json.RawMessage      `json:"value"`
		}
		if err := json.Unmarshal(rf, &cfStruct); err == nil &&
			(cfStruct.Operator == filter.LogicalAnd || cfStruct.Operator == filter.LogicalOr || cfStruct.Operator == filter.LogicalNot) {
			// Validate nested filters
			if len(cfStruct.Value) == 0 {
				return nil, fmt.Errorf("conditional filter must have nested filters")
			}

			// Parse nested filters
			var nestedFilters []filter.CrudFilter
			for _, nestedRF := range cfStruct.Value {
				nf, err := parseFilter(nestedRF)
				if err != nil {
					return nil, err
				}
				nestedFilters = append(nestedFilters, nf)
			}

			cf := &filter.ConditionalFilter{
				Operator: cfStruct.Operator,
				Filters:  nestedFilters,
			}
			if err != nil {
				return nil, err
			}
			cf.Filters = nestedFilters
			filters = append(filters, cf)
			continue
		}

		return nil, fmt.Errorf("invalid filter structure: %v", string(rf))
	}

	return filters, nil
}

func validateValueForOperator(op filter.Operator, value interface{}) error {
	switch op {
	case filter.OpNull, filter.OpNnull:
		if value != nil {
			return fmt.Errorf("operator %q requires null value", op)
		}
	case filter.OpBetween, filter.OpNbetween:
		if vals, ok := value.([]interface{}); !ok || len(vals) != 2 {
			return fmt.Errorf("operator %q requires array with exactly 2 values", op)
		}
	case filter.OpIn, filter.OpNin, filter.OpIna, filter.OpNina:
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("operator %q requires array value", op)
		}
	}
	return nil
}

func parseFilter(rf json.RawMessage) (filter.CrudFilter, error) {
	// Try logical filter first
	lf := &filter.LogicalFilter{}
	if err := json.Unmarshal(rf, lf); err == nil && lf.Field != "" {
		if _, err := filter.ParseOperator(string(lf.Operator)); err != nil {
			return nil, fmt.Errorf("invalid operator %q: %w", lf.Operator, err)
		}
		
		// Validate value type for operator
		if err := validateValueForOperator(lf.Operator, lf.Value); err != nil {
			return nil, fmt.Errorf("invalid value for operator %q: %w", lf.Operator, err)
		}
		return lf, nil
	}

	// Try conditional filter
	var cfStruct struct {
		Operator filter.LogicalOperator `json:"operator"`
		Value    []json.RawMessage      `json:"value"`
	}
	if err := json.Unmarshal(rf, &cfStruct); err == nil &&
		(cfStruct.Operator == filter.LogicalAnd || cfStruct.Operator == filter.LogicalOr || cfStruct.Operator == filter.LogicalNot) {
		if len(cfStruct.Value) == 0 {
			return nil, fmt.Errorf("conditional filter must have nested filters")
		}

		var nestedFilters []filter.CrudFilter
		for _, nestedRF := range cfStruct.Value {
			nf, err := parseFilter(nestedRF)
			if err != nil {
				return nil, err
			}
			nestedFilters = append(nestedFilters, nf)
		}

		return &filter.ConditionalFilter{
			Operator: cfStruct.Operator,
			Filters:  nestedFilters,
		}, nil
	}

	return nil, fmt.Errorf("invalid filter structure: %v", string(rf))
}

func (p *JSONParser) ParseNested(rawFilters []filter.CrudFilter) ([]filter.CrudFilter, error) {
	var nested []filter.CrudFilter
	for _, f := range rawFilters {
		switch v := f.(type) {
		case *filter.LogicalFilter:
			nested = append(nested, v)
		case *filter.ConditionalFilter:
			cf := *v
			children, err := p.ParseNested(cf.Filters)
			if err != nil {
				return nil, err
			}
			cf.Filters = children
			nested = append(nested, &cf)
		default:
			return nil, fmt.Errorf("unknown filter type: %T", f)
		}
	}
	return nested, nil
}
