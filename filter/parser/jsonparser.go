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
		return nil, fmt.Errorf("invalid JSON format (must be a array): %w", err)
	}

	// Handle null input by checking if rawFilters is nil
	if rawFilters == nil {
		return nil, filter.NewFilterError("", "input must be a JSON array")
	}

	// Initialize the result slice as non-nil, potentially with capacity
	filters = make([]filter.CrudFilter, 0, len(rawFilters))

	for _, rf := range rawFilters {
		// Delegate parsing each top-level raw message to the helper function
		f, err := parseFilterFromRaw(rf)
		if err != nil {
			return nil, err // Propagate parsing errors
		}
		filters = append(filters, f)
	}

	return filters, nil
}



func parseFilterFromRaw(rf json.RawMessage) (filter.CrudFilter, error) {
	var raw map[string]any
	if err := json.Unmarshal(rf, &raw); err != nil {
		return nil, filter.NewFilterError("", fmt.Sprintf("invalid JSON object structure: %v", err))
	}

	// Check if it's a conditional filter first (operator and value keys)
	operatorVal, okOperator := raw["operator"]
	valueVal, okValue := raw["value"]
	fieldVal, okField := raw["field"]

	if okOperator && okValue && !okField {
		// Looks like a conditional filter
		opStr, ok := operatorVal.(string)
		if !ok {
			return nil, filter.NewFilterError("operator", "must be a string")
		}
		op := filter.LogicalOperator(opStr)
		if op != filter.LogicalAnd && op != filter.LogicalOr && op != filter.LogicalNot {
			return nil, filter.NewFilterError("operator", fmt.Sprintf("invalid logical operator %q", opStr))
		}

		// For conditional filter, 'value' is expected to be an array of filter objects
		valueArr, ok := valueVal.([]any)
		if !ok {
			return nil, filter.NewFilterError("value", "must be an array of filters")
		}

		if len(valueArr) == 0 && (op == filter.LogicalAnd || op == filter.LogicalOr) {
			return nil, filter.NewFilterError(string(op), "conditional filter must have nested filters")
		}

		if op == filter.LogicalNot && len(valueArr) != 1 {
			return nil, filter.NewFilterError(string(op), "NOT operator requires exactly one nested filter")
		}

		var nestedFilters []filter.CrudFilter
		for _, item := range valueArr {
			nestedRF, err := json.Marshal(item)
			if err != nil {
				return nil, filter.NewFilterError("", fmt.Sprintf("internal error: failed to re-marshal nested filter: %v", err))
			}
			nf, err := parseFilterFromRaw(nestedRF)
			if err != nil {
				return nil, err
			}
			nestedFilters = append(nestedFilters, nf)
		}

		return filter.NewConditionalFilter(op, nestedFilters), nil

	} else if okField && okOperator {
		// Looks like a logical filter (must have field and operator)
		fieldStr, ok := fieldVal.(string)
		if !ok || fieldStr == "" {
			return nil, filter.NewFilterError("field", "must be a non-empty string")
		}

		opStr, ok := operatorVal.(string)
		if !ok {
			return nil, filter.NewFilterError("operator", "must be a string")
		}
		op, err := filter.ParseOperator(opStr)
		if err != nil {
			return nil, filter.NewFilterError("operator", err.Error())
		}

		var lf filter.CrudFilter
		defer func() {
			if r := recover(); r != nil {
				err = filter.NewFilterError(fieldStr, fmt.Sprintf("value validation failed for operator '%s': %v", opStr, r))
				lf = nil
			}
		}()

		lf = filter.NewLogicalFilter(fieldStr, op, valueVal)

		if err != nil {
			return nil, err
		}

		return lf, nil
	}

	return nil, filter.NewFilterError("", fmt.Sprintf("invalid filter structure: %v", string(rf)))
}
