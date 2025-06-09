package generator

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/samber/lo"
	"go.lumeweb.com/queryutil/filter"
)

// QueryParamGenerator converts CrudFilters into URL query parameters
type QueryParamGenerator interface {
	Generate(filters []filter.CrudFilter) (map[string][]string, error)
}

// DefaultQueryParamGenerator is the standard implementation of QueryParamGenerator
type DefaultQueryParamGenerator struct{}

// NewDefaultQueryParamGenerator creates a new DefaultQueryParamGenerator
func NewDefaultQueryParamGenerator() QueryParamGenerator {
	return &DefaultQueryParamGenerator{}
}

// Generate converts filters into URL query parameters
func (g *DefaultQueryParamGenerator) Generate(filters []filter.CrudFilter) (map[string][]string, error) {
	query := make(map[string][]string)
	generator := &queryGenerator{
		query: query,
	}

	if err := generator.processFilters(filters, []string{"filters"}); err != nil {
		return nil, err
	}
	return query, nil
}

// queryGenerator holds state for generating query parameters
type queryGenerator struct {
	query map[string][]string
}

// processFilters recursively processes filters into query parameters
func (g *queryGenerator) processFilters(filters []filter.CrudFilter, path []string) error {
	for i, f := range filters {
		switch v := f.(type) {
		case *filter.LogicalFilter:
			if err := g.processLogicalFilter(v, path); err != nil {
				return err
			}
		case *filter.ConditionalFilter:
			if err := g.processConditionalFilter(v, path, i); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported filter type: %T", f)
		}
	}
	return nil
}

// processLogicalFilter handles a single logical filter
func (g *queryGenerator) processLogicalFilter(f *filter.LogicalFilter, path []string) error {
	// Handle global search (q) specially
	if f.Field() == "q" {
		g.query["q"] = []string{url.QueryEscape(fmt.Sprint(f.Value()))}
		return nil
	}

	// Build path segments
	segments := append(path, f.Field())
	if f.Operator() != filter.OpEq {
		opStrs, ok := filter.GetOperatorReverseMap()[f.Operator()]
		if !ok {
			return fmt.Errorf("unsupported operator: %s", f.Operator())
		}
		// Use the first alias (primary operator name) from the reverse map
		segments = append(segments, opStrs[0])
	}

	key := buildQueryKey(segments)

	// Handle values
	values, err := g.getFilterValues(f)
	if err != nil {
		return err
	}

	g.query[key] = values
	return nil
}

// processConditionalFilter handles AND/OR/NOT conditional filters
func (g *queryGenerator) processConditionalFilter(f *filter.ConditionalFilter, path []string, index int) error {
	// For conditional filters, we need to maintain the original index structure
	// from the test expectations
	newPath := append(path, string(f.Operator))
	for i, subFilter := range f.Filters {
		subPath := append(newPath, strconv.Itoa(i))
		switch sf := subFilter.(type) {
		case *filter.LogicalFilter:
			if err := g.processLogicalFilter(sf, subPath); err != nil {
				return err
			}
		case *filter.ConditionalFilter:
			if err := g.processConditionalFilter(sf, subPath, i); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported filter type: %T", subFilter)
		}
	}
	return nil
}

// getFilterValues converts filter values to URL-encoded strings
func (g *queryGenerator) getFilterValues(f *filter.LogicalFilter) ([]string, error) {
	formatValue := func(v any) string {
		if t, ok := v.(time.Time); ok {
			return t.Format(time.RFC3339)
		}
		return fmt.Sprint(v)
	}

	switch val := f.Value().(type) {
	case []any: // For "in" and "between"
		if f.Operator() == filter.OpBetween || f.Operator() == filter.OpNbetween {
			if len(val) != 2 {
				return nil, fmt.Errorf("between operator requires exactly 2 values")
			}
			// Skip numeric validation if values are time.Time
			_, isTime1 := val[0].(time.Time)
			_, isTime2 := val[1].(time.Time)
			if !isTime1 && !isTime2 {
				// Validate between values are numeric if not time.Time
				if _, err := strconv.ParseFloat(formatValue(val[0]), 64); err != nil {
					return nil, fmt.Errorf("between operator requires numeric or time values")
				}
				if _, err := strconv.ParseFloat(formatValue(val[1]), 64); err != nil {
					return nil, fmt.Errorf("between operator requires numeric or time values")
				}
			}
		}
		values := lo.Map(val, func(item any, _ int) string {
			return url.QueryEscape(formatValue(item))
		})
		return values, nil
	case nil: // For null/nnull operators
		return []string{""}, nil
	default:
		return []string{url.QueryEscape(formatValue(f.Value()))}, nil
	}
}

// buildQueryKey constructs a query parameter key from path segments
func buildQueryKey(segments []string) string {
	if len(segments) == 0 {
		return ""
	}

	key := segments[0]
	for _, seg := range segments[1:] {
		key += "[" + seg + "]"
	}
	return key
}
