package generator

import (
	"fmt"
	"net/url"
	"strconv"

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
	switch val := f.Value().(type) {
	case []any: // For "in" and "between"
		if f.Operator() == filter.OpBetween || f.Operator() == filter.OpNbetween {
			if len(val) != 2 {
				return nil, fmt.Errorf("between operator requires exactly 2 values")
			}
			// Validate between values are numeric before encoding
			if _, err := strconv.ParseFloat(fmt.Sprint(val[0]), 64); err != nil {
				return nil, fmt.Errorf("between operator requires numeric values")
			}
			if _, err := strconv.ParseFloat(fmt.Sprint(val[1]), 64); err != nil {
				return nil, fmt.Errorf("between operator requires numeric values")
			}
		}
		values := lo.Map(val, func(item any, _ int) string {
			return url.QueryEscape(fmt.Sprint(item))
		})
		return values, nil
	case nil: // For null/nnull operators
		return []string{""}, nil
	default:
		return []string{url.QueryEscape(fmt.Sprint(f.Value()))}, nil
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
