package queryutil

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
	"net/http"
	"net/url"
)

// ParseQuery parses all query parameters from a map.
// This is the main entry point for processing query parameters in an HTTP-agnostic way.
// It combines parsing of filters, sorts, and pagination parameters.
//
// Returns parsed Filter, Sort, and Pagination structs or an error
// if any validation fails.
//
// Example:
//
//	// With a map of query parameters
//	query := map[string][]string{
//	    "name": {"john"},
//	    "age_gte": {"18"},
//	    "_sort": {"name"},
//	    "_order": {"desc"},
//	    "_start": {"0"},
//	    "_end": {"10"},
//	}
//	filters, sorts, pagination, err := ParseQuery(query)
func ParseQuery(query map[string][]string) ([]CrudFilter, []Sort, Pagination, error) {
	p := parser.NewQueryParamParser(query)
	return ParseFromSource(p)
}

// ParseQueryWithSearch parses query parameters with global search support.
// Similar to ParseQuery but includes configuration for global search
// functionality across multiple columns.
//
// The searchConfig parameter specifies which columns should be included
// in global search operations when the 'q' parameter is present.
func ParseQueryWithSearch(query map[string][]string, searchConfig *filter.GlobalSearchConfig) ([]CrudFilter, []Sort, Pagination, error) {
	p := parser.NewQueryParamParser(query, parser.WithSearchConfig(searchConfig))
	return ParseFromSource(p)
}

// HTTPRequestParser implements the RequestParser interface for HTTP requests.
// It provides a way to parse query parameters from an HTTP request's URL query.
type HTTPRequestParser struct {
	Query        url.Values
	SearchConfig *filter.GlobalSearchConfig
	sortConfig   *filter.SortConfig
}

// NewHTTPRequestParser creates a new HTTPRequestParser from an http.Request
func NewHTTPRequestParser(r *http.Request, searchConfig *filter.GlobalSearchConfig, sortConfig *filter.SortConfig) *HTTPRequestParser {
	return &HTTPRequestParser{
		Query:        r.URL.Query(),
		SearchConfig: searchConfig,
		sortConfig:   sortConfig,
	}
}

// ParseFilters implements RequestParser.ParseFilters
func (p *HTTPRequestParser) ParseFilters() ([]filter.CrudFilter, error) {
	return parser.NewQueryParamParser(p.Query).ParseFilters()
}

// ParseSorts implements RequestParser.ParseSorts
func (p *HTTPRequestParser) ParseSorts(config *filter.SortConfig) ([]filter.Sort, error) {
	// Use provided config if available, fallback to our config
	if config == nil {
		config = p.sortConfig
	}
	if config == nil {
		config = &filter.SortConfig{} // Default empty config
	}
	return ParseQuerySort(p.Query, config)
}

// GetSortConfig returns the parser's sort configuration
func (p *HTTPRequestParser) GetSortConfig() *filter.SortConfig {
	return p.sortConfig
}

// ParsePagination implements RequestParser.ParsePagination
func (p *HTTPRequestParser) ParsePagination() (Pagination, error) {
	return ParseQueryPagination(p.Query)
}

// ParseRequest parses all query parameters from an HTTP request.
// This is maintained for backward compatibility with older code.
func ParseRequest(r interface{}) ([]CrudFilter, []Sort, Pagination, error) {
	if req, ok := r.(*http.Request); ok {
		filters, sorts, pagination, err := ParseFromSource(NewHTTPRequestParser(req, nil, nil))
		return filters, sorts, pagination, err
	}
	return nil, nil, Pagination{}, fmt.Errorf("unsupported request type")
}

// ParseRequestWithSearch parses query parameters with global search support.
// This is maintained for backward compatibility.
func ParseRequestWithSearch(r interface{}, searchConfig *GlobalSearchConfig) ([]CrudFilter, []Sort, Pagination, error) {
	if req, ok := r.(*http.Request); ok {
		filters, sorts, pagination, err := ParseFromSource(NewHTTPRequestParser(req, searchConfig, &filter.SortConfig{
			SortableFields: searchConfig.SearchableColumns,
		}))
		return filters, sorts, pagination, err
	}
	return nil, nil, Pagination{}, fmt.Errorf("unsupported request type")
}
