package parser

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

var invalidBoolStrings = []string{"t", "f", "yes", "no", "on", "off"}

var allowedEmptyStringOps = lo.Associate([]filter.Operator{
	filter.OpContains,
	filter.OpNcontains,
	filter.OpStartswith,
	filter.OpNstartswith,
	filter.OpEndswith,
	filter.OpNendswith,
	filter.OpContainss,
	filter.OpNcontainss,
	filter.OpStartswiths,
	filter.OpNstartswiths,
	filter.OpEndswiths,
	filter.OpNendswiths,
	filter.OpEq,
	filter.OpNe,
}, func(op filter.Operator) (filter.Operator, struct{}) {
	return op, struct{}{}
})

const (
	dotPlaceholder = ":DOT:"
	filterPrefix   = "filters"
	indexPattern   = `^\d+$`
)

// QueryParamParser handles parsing of filter parameters from URL query strings.
// Supports complex nested filters using bracket notation and various operators.
type QueryParamParser struct {
	query  url.Values     // Source query parameters
	config *ParserOptions // Parser configuration
}

// NewQueryParamParser creates a new parser instance for URL query parameters.
// Uses functional options to configure parsing behavior:
//   - WithSearchConfig() for global search settings
//   - WithInputFormat() to specify format (though querystring is default)
func NewQueryParamParser(query url.Values, opts ...ParserOption) *QueryParamParser {
	p := &QueryParamParser{
		query:  query,
		config: ApplyParserOptions(opts),
	}
	return p
}

func (p *QueryParamParser) ParseSorts(config *filter.SortConfig) ([]filter.Sort, error) {
	// Combine parser config with parameter
	if config == nil {
		config = p.config.SortConfig
	}
	return filter.ParseQuerySort(p.query, config)
}

// ParseFilters converts URL query parameters into filter expressions.
// Supports nested AND/OR/NOT operators using bracket notation.
// Example: filters[or][0][age][gte]=30&filters[or][1][name][contains]=john
// Returns combined filters or validation error if parameters are malformed
func (p *QueryParamParser) ParseFilters() ([]filter.CrudFilter, error) {
	filterMap := make(map[string]any)

	// Build filter structure from query parameters
	for key, values := range p.query {
		// Skip special parameters that start with _
		if strings.HasPrefix(key, "_") {
			continue
		}

		// Handle both prefixed and non-prefixed filter formats
		if !strings.HasPrefix(key, filterPrefix) && !strings.Contains(key, "[") {
			// Split field and operator from suffix
			if strings.Contains(key, "_") {
				parts := strings.SplitN(key, "_", 2)
				field := parts[0]
				operator := parts[1]
				key = fmt.Sprintf("%s[%s][%s]", filterPrefix, field, operator)
			} else {
				// Default to equality operator
				key = fmt.Sprintf("%s[%s]", filterPrefix, key)
			}
		} else if !strings.HasPrefix(key, filterPrefix) {
			continue
		}

		path := strings.TrimPrefix(key, filterPrefix)
		segments := parseSegments(path)
		if len(segments) == 0 {
			if path != "" && path != "[]" {
				return nil, fmt.Errorf("invalid filter key: %s resulted in empty segments", key)
			}
			continue
		}

		currentNestedMap := filterMap
		for _, segment := range segments[:len(segments)-1] {
			if _, ok := currentNestedMap[segment]; !ok {
				currentNestedMap[segment] = make(map[string]any)
			}

			if nextMap, ok := currentNestedMap[segment].(map[string]any); ok {
				currentNestedMap = nextMap
			} else {
				return nil, fmt.Errorf("path conflict at segment '%s' for key '%s'", segment, key)
			}
		}

		finalKey := segments[len(segments)-1]
		if _, exists := currentNestedMap[finalKey]; exists {
			if _, isMap := currentNestedMap[finalKey].(map[string]any); isMap {
				return nil, fmt.Errorf("path conflict at final key '%s' for query key '%s'", finalKey, key)
			}
		}

		currentNestedMap[finalKey] = values
	}

	filters, err := p.buildFilters(filterMap)
	return filters, err
}

var segmentRegex = regexp.MustCompile(`\[(.*?)\]`) // Matches content inside brackets

// parseSegments extracts bracket-enclosed segments from a query param key.
// Converts nested key syntax like "[or][0][age][gte]" into ["or","0","age","gte"]
// Used to build hierarchical filter structures from flat query parameters
func parseSegments(path string) []string {
	if path == "" {
		return []string{}
	}

	var result []string
	// Find all occurrences of "[content]"
	matches := segmentRegex.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		// match[0] is the full "[content]", match[1] is "content"
		result = append(result, match[1])
	}
	return result
}

func (p *QueryParamParser) buildFilters(data any) ([]filter.CrudFilter, error) {
	switch v := data.(type) {
	case map[string]any:
		return p.buildFromMap(v)
	case []any:
		return p.buildFromSlice(v)
	default:
		return nil, fmt.Errorf("unexpected filter structure type: %T", data)
	}
}

func (p *QueryParamParser) buildFromMap(m map[string]any) ([]filter.CrudFilter, error) {
	var filters []filter.CrudFilter

	for key, value := range m {
		switch key {
		case "and", "or", "not":
			// Handle nested conditional filters that might be wrapped in arrays
			conditionalFilters, err := p.buildConditionalFilter(key, value)
			if err != nil {
				return nil, err
			}
			filters = append(filters, conditionalFilters...)
		default:
			logicalFilter, err := p.buildLogicalFilter(key, value)
			if err != nil {
			} else {
			}
			if err != nil {
				return nil, err
			}
			filters = append(filters, logicalFilter)
		}
	}

	return filters, nil
}

func (p *QueryParamParser) buildConditionalFilter(operator string, value any) ([]filter.CrudFilter, error) {
	var nestedFilters []filter.CrudFilter

	// Handle both array and single map values for nested conditionals
	if arr, ok := value.([]any); ok {
		for _, item := range arr {
			filters, err := p.buildFilters(item)
			if err != nil {
				return nil, err
			}
			nestedFilters = append(nestedFilters, filters...)
		}
	} else if m, ok := value.(map[string]any); ok {
		var intKeys []int
		for kStr := range m {
			if kInt, err := strconv.Atoi(kStr); err == nil {
				intKeys = append(intKeys, kInt)
			} else {
				return nil, fmt.Errorf("invalid key '%s' under conditional operator '%s'", kStr, operator)
			}
		}
		sort.Ints(intKeys)

		if len(intKeys) == 0 && (operator == string(filter.LogicalAnd) || operator == string(filter.LogicalOr)) {
			return nil, fmt.Errorf("empty %s group", operator)
		}

		for _, kInt := range intKeys {
			item := m[strconv.Itoa(kInt)]
			filters, err := p.buildFilters(item)
			if err != nil {
				return nil, err
			}
			nestedFilters = append(nestedFilters, filters...)
		}
	} else {
		return nil, fmt.Errorf("unexpected conditional value type: %T", value)
	}

	if operator == string(filter.LogicalNot) && len(nestedFilters) == 0 {
		return nil, fmt.Errorf("NOT operator requires a sub-filter")
	}

	return []filter.CrudFilter{&filter.ConditionalFilter{
		Operator: filter.LogicalOperator(operator),
		Filters:  nestedFilters,
	}}, nil
}

func convertStringValue(value any) (any, error) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return "", nil
		}
		// Try int first
		if intVal, err := strconv.Atoi(v); err == nil { // "1" -> int(1), "0" -> int(0)
			return intVal, nil
		}
		// Try float
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return floatVal, nil
		}
		// Strict actual boolean strings "true" / "false" (case-sensitive)
		if v == "true" {
			return true, nil
		}
		if v == "false" {
			return false, nil
		}

		// Check for other common boolean representations that are NOT strictly "true" or "false"
		lowerV := strings.ToLower(v)
		// invalidBoolStrings = []string{"t", "f", "yes", "no", "on", "off"}
		if (lowerV == "true" && v != "true") || (lowerV == "false" && v != "false") || // Catches "True", "FALSE"
			lo.Contains(invalidBoolStrings, lowerV) { // Catches "t", "f", "yes", "no", "on", "off"
			return nil, fmt.Errorf("invalid boolean value %q; use 'true' or 'false' (case-sensitive)", v)
		}

		// Decode any URL-encoded string value
		decoded, err := url.QueryUnescape(v)
		if err != nil {
			return v, nil // Return original if unescaping fails
		}
		return decoded, nil // Return decoded value

	case []string:
		if len(v) == 0 {
			return nil, fmt.Errorf("empty array value")
		}
		var converted []any
		for _, s := range v {
			c, err := convertStringValue(s)
			if err != nil {
				return nil, fmt.Errorf("error converting item '%s' in array: %w", s, err)
			}
			converted = append(converted, c)
		}
		return converted, nil

	case []any:
		var convertedInternal []any
		for _, item := range v {
			c, err := convertStringValue(item)
			if err != nil {
				return nil, err
			}
			convertedInternal = append(convertedInternal, c)
		}
		return convertedInternal, nil
	default:
		return value, nil
	}
}

// isNumericType checks if a value is an integer or float type.
func isNumericType(val any) (isNumeric bool, typeName string) {
	switch val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true, "integer"
	case float32, float64:
		return true, "float"
	default:
		return false, ""
	}
}

// buildLogicalFilter constructs a LogicalFilter from query parameter values.
// Handles value conversion and validation for different operator types.
// Validates operator-value compatibility and performs type conversion for:
// - Numeric values (int/float)
// - Boolean values (strict "true"/"false" only)
// - Array values for IN/NIN/BETWEEN operators
// Returns formatted LogicalFilter or error if validation fails
func (p *QueryParamParser) buildLogicalFilter(field string, value any) (filter.CrudFilter, error) {
	if field == "" {
		return nil, fmt.Errorf("field name cannot be empty")
	}

	var operator filter.Operator
	var rawValForOpProcessing any

	if subMap, ok := value.(map[string]any); ok {
		if len(subMap) > 1 {
			return nil, fmt.Errorf("multiple operators not allowed for field %q in query params", field)
		}
		if len(subMap) == 0 {
			return nil, fmt.Errorf("empty operator map for field %q", field)
		}
		for opStr, val := range subMap {
			if _, known := filter.OperatorMap[strings.ToLower(opStr)]; known && opStr != strings.ToLower(opStr) {
				return nil, fmt.Errorf("operator %q for field %q must be lowercase", opStr, field)
			}
			mappedOp, found := filter.OperatorMap[opStr]
			if !found {
				return nil, fmt.Errorf("unsupported operator: %s for field %s", opStr, field)
			}
			operator = mappedOp
			rawValForOpProcessing = val
		}
	} else {
		operator = filter.OpEq
		rawValForOpProcessing = value
	}

	if opIsMultiValue(operator) {
		if rvs, ok := rawValForOpProcessing.([]string); ok && len(rvs) == 1 && strings.Contains(rvs[0], ",") {
			rawValForOpProcessing = strings.Split(rvs[0], ",")
		}
	}

	var parsedVal any
	var err error
	isProcessedAsNullOp := false

	if operator == filter.OpNull || operator == filter.OpNnull {
		isProcessedAsNullOp = true
		if rvs, ok := rawValForOpProcessing.([]string); ok && (len(rvs) == 0 || (len(rvs) == 1 && rvs[0] == "")) {
			parsedVal = nil
		} else if rawValForOpProcessing == nil {
			parsedVal = nil
		} else {
			tempVal, _ := convertStringValue(rawValForOpProcessing)
			return nil, fmt.Errorf("operator %q on field %q does not accept values, got: %v", operator, field, tempVal)
		}
	} else if operator.RequiresArray() {
		if valMap, ok := rawValForOpProcessing.(map[string]any); ok {
			var indexedStrValues []string
			var intKeys []int
			for kStr := range valMap {
				if kInt, e := strconv.Atoi(kStr); e == nil {
					intKeys = append(intKeys, kInt)
				} else {
					return nil, fmt.Errorf("invalid non-numeric index %q for array operator %q on field %q", kStr, operator, field)
				}
			}
			sort.Ints(intKeys)
			if (operator == filter.OpBetween || operator == filter.OpNbetween) && len(intKeys) != 2 {
				return nil, fmt.Errorf("operator %q on field %q requires exactly 2 indexed values, got %d", operator, field, len(intKeys))
			}
			for _, kInt := range intKeys {
				idxRawValSlice, sliceOk := valMap[strconv.Itoa(kInt)].([]string)
				if !sliceOk || len(idxRawValSlice) == 0 {
					return nil, fmt.Errorf("missing or invalid value for index %d of operator %q on field %q", kInt, operator, field)
				}
				indexedStrValues = append(indexedStrValues, idxRawValSlice[0])
			}
			parsedVal, err = convertStringValue(indexedStrValues)
		} else {
			parsedVal, err = convertStringValue(rawValForOpProcessing)
		}
	} else {
		rvs, ok := rawValForOpProcessing.([]string)
		if !ok {
			return nil, fmt.Errorf("internal error: expected []string for single value op, got %T for field %s", rawValForOpProcessing, field)
		}
		if len(rvs) > 1 {
			return nil, fmt.Errorf("operator %q on field %q received multiple values %v but expected one", operator, field, rvs)
		}
		if len(rvs) == 0 {
			parsedVal, err = convertStringValue("")
		} else {
			parsedVal, err = convertStringValue(rvs[0])
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "empty array value provided for operator") && operator.RequiresArray() {
			return nil, fmt.Errorf("operator %q on field %q requires non-empty array values: %w", operator, field, err)
		}
		return nil, fmt.Errorf("error parsing value for field %q, operator %q: %w", field, operator, err)
	}

	if operator.RequiresArray() {
		sliceVal, ok := parsedVal.([]any)
		if parsedVal == nil || !ok || len(sliceVal) == 0 {
			return nil, fmt.Errorf("operator %q on field %q requires non-empty array values", operator, field)
		}
		if (operator == filter.OpBetween || operator == filter.OpNbetween) && len(sliceVal) != 2 {
			return nil, fmt.Errorf("operator %q on field %q requires exactly 2 values, got %d", operator, field, len(sliceVal))
		}

		// Additional validation for OpBetween and OpNbetween values
		if operator == filter.OpBetween || operator == filter.OpNbetween {
			val1 := sliceVal[0]
			val2 := sliceVal[1]

			isNum1, _ := isNumericType(val1)
			isNum2, _ := isNumericType(val2)

			if !isNum1 || !isNum2 {
				_, val1IsStr := val1.(string)
				_, val2IsStr := val2.(string)
				if val1IsStr && val2IsStr { // Both are strings that were not parseable as numbers
					return nil, fmt.Errorf("operator %q on field %q received non-numeric string values [%v, %v]; numeric values are required for comparison", operator, field, val1, val2)
				}
				return nil, fmt.Errorf("operator %q on field %q requires numeric values for comparison, but received %T (%v) and %T (%v)", operator, field, val1, val1, val2, val2)
			}
		}
	} else if operator == filter.OpNull || operator == filter.OpNnull {
		if parsedVal != nil {
			return nil, fmt.Errorf("operator %q on field %q does not accept values, got: %v", operator, field, parsedVal)
		}
	} else if sliceVal, ok := parsedVal.([]any); ok {
		if len(sliceVal) > 1 {
			return nil, fmt.Errorf("operator %q on field %q does not support multiple values", operator, field)
		}
		if len(sliceVal) == 1 {
			parsedVal = sliceVal[0]
		} else {
			parsedVal = nil
		}
	}

	if !isProcessedAsNullOp {
		// This block handles empty string validation for non-null/nnull operators
		if strVal, ok := parsedVal.(string); ok && strVal == "" {
			_, operatorAllowsEmpty := allowedEmptyStringOps[operator]
			isEqOnQField := (operator == filter.OpEq && field == "q")

			if !operatorAllowsEmpty && !isEqOnQField {
				return nil, fmt.Errorf("empty string value not allowed for operator %q on field %q", operator, field)
			}
		}
	}

	return filter.NewLogicalFilter(field, operator, parsedVal), nil
}

func (p *QueryParamParser) ParsePagination() (filter.Pagination, error) {
	return filter.ParseQueryPagination(p.query)
}

func (p *QueryParamParser) buildFromSlice(s []any) ([]filter.CrudFilter, error) {
	var filters []filter.CrudFilter

	for _, item := range s {
		if m, ok := item.(map[string]any); ok {
			subFilters, err := p.buildFromMap(m)
			if err != nil {
				return nil, err
			}
			filters = append(filters, subFilters...)
		} else if arr, ok := item.([]any); ok {
			subFilters, err := p.buildFromSlice(arr)
			if err != nil {
				return nil, err
			}
			filters = append(filters, subFilters...)
		} else {
			return nil, fmt.Errorf("unexpected slice element type: %T", item)
		}
	}

	return filters, nil
}

// Helper for comma splitting decision
func opIsMultiValue(op filter.Operator) bool {
	return op == filter.OpIn || op == filter.OpNin || op == filter.OpIna || op == filter.OpNina
}
