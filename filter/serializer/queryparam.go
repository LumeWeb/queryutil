package serializer

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"go.lumeweb.com/queryutil/filter"
)

// QueryParamSerializer implements QuerySerializer for URL query parameter format.
// Handles conversion of filter structures back to flat query parameter format.
type QueryParamSerializer struct {
	config *SerializerOptions
}

// NewQueryParamSerializer creates a new serializer instance for URL query parameters.
// Uses functional options to configure serialization behavior:
//   - WithFilterPrefix() to customize filter parameter prefix
//   - WithSortPrefix() to customize sort parameter prefix
func NewQueryParamSerializer(opts ...SerializerOption) *QueryParamSerializer {
	return &QueryParamSerializer{
		config: ApplySerializerOptions(opts...),
	}
}

// SerializeFilters converts filter structures to URL query parameters.
// Handles both logical and conditional filters with proper nesting.
// Returns url.Values that can be used to build query strings.
func (s *QueryParamSerializer) SerializeFilters(filters []filter.CrudFilter) (url.Values, error) {
	values := url.Values{}

	for _, f := range filters {
		// Never include top-level indices to match parser expectations
		// Indices are only used for nested filters inside logical operators
		if err := s.serializeFilter(f, values, s.config.FilterPrefix, -1); err != nil {
			return nil, err
		}
	}

	return values, nil
}

// SerializeSorts converts sort structures to URL query parameters.
// Formats sort parameters according to the configured prefix.
// Uses standard _sort and _order format when default prefix is used.
// Note: "sort" prefix enables _sort/_order style, while any other prefix uses legacy prefix[index]=field:order format.
func (s *QueryParamSerializer) SerializeSorts(sorts []filter.Sort) (url.Values, error) {
	values := url.Values{}

	if len(sorts) == 0 {
		return values, nil
	}

	// Check if using default sort prefix - if so, use _sort and _order format
	if s.config.SortPrefix == "sort" {
		// Build comma-separated field names
		var fields []string
		var orders []string

		for _, sort := range sorts {
			fields = append(fields, sort.Field)
			orders = append(orders, string(sort.Order))
		}

		// Use standard _sort and _order parameter names
		values.Add(filter.SortParamName, strings.Join(fields, ","))
		values.Add(filter.OrderParamName, strings.Join(orders, ","))
	} else {
		// Use custom prefix format for backward compatibility
		for i, sort := range sorts {
			key := s.config.SortPrefix + "[" + strconv.Itoa(i) + "]"
			values.Add(key, sort.Field+":"+string(sort.Order))
		}
	}

	return values, nil
}

// SerializePagination converts pagination structure to URL query parameters.
// Uses standard _start and _end parameter names.
func (s *QueryParamSerializer) SerializePagination(pagination filter.Pagination) (url.Values, error) {
	values := url.Values{}

	values.Add("_start", strconv.Itoa(pagination.Start))
	values.Add("_end", strconv.Itoa(pagination.End))

	return values, nil
}

// serializeFilter handles the serialization of individual filters with proper nesting
func (s *QueryParamSerializer) serializeFilter(f filter.CrudFilter, values url.Values, prefix string, index int) error {
	switch flt := f.(type) {
	case *filter.LogicalFilter:
		return s.serializeLogicalFilter(flt, values, prefix, index)
	case *filter.ConditionalFilter:
		return s.serializeConditionalFilter(flt, values, prefix, index)
	default:
		return fmt.Errorf("unsupported filter type: %T", f)
	}
}

// serializeLogicalFilter serializes a logical filter to query parameters
func (s *QueryParamSerializer) serializeLogicalFilter(logicalFilter *filter.LogicalFilter, values url.Values, prefix string, index int) error {
	key := prefix
	if index >= 0 {
		key += "[" + strconv.Itoa(index) + "]"
	}

	// For simple equality without operator, use field directly
	if logicalFilter.Operator() == filter.OpEq {
		values.Add(key+"["+logicalFilter.GetField()+"]", s.formatValue(logicalFilter.GetValue()))
	} else {
		operatorKey := key + "[" + logicalFilter.GetField() + "][" + string(logicalFilter.Operator()) + "]"

		// Handle array operators by creating indexed parameters
		if logicalFilter.Operator().RequiresArray() {
			if arrayValue, ok := logicalFilter.GetValue().([]any); ok {
				for i, item := range arrayValue {
					values.Add(operatorKey+"["+strconv.Itoa(i)+"]", s.formatValue(item))
				}
				return nil
			}
		}

		values.Add(operatorKey, s.formatValue(logicalFilter.GetValue()))
	}

	return nil
}

// serializeConditionalFilter serializes a conditional filter to query parameters
func (s *QueryParamSerializer) serializeConditionalFilter(conditionalFilter *filter.ConditionalFilter, values url.Values, prefix string, index int) error {
	key := prefix
	if index >= 0 {
		key += "[" + strconv.Itoa(index) + "]"
	}

	operatorKey := key + "[" + string(conditionalFilter.Operator) + "]"

	for i, nestedFilter := range conditionalFilter.Filters {
		if err := s.serializeFilter(nestedFilter, values, operatorKey, i); err != nil {
			return err
		}
	}

	return nil
}

// formatValue converts a value to string format for query parameters
func (s *QueryParamSerializer) formatValue(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return strconv.FormatBool(v)
	case []any:
		var result []string
		for _, item := range v {
			result = append(result, s.formatValue(item))
		}
		return fmt.Sprintf("%v", result) // This will be handled differently for array operators
	default:
		return fmt.Sprintf("%v", v)
	}
}
