package queryutil

import (
	"fmt"
	"net/http"
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
func ParseQuery(query map[string][]string) ([]Filter, []Sort, Pagination, error) {
	// Parse filters
	filters, err := ParseQueryFilters(query)
	if err != nil {
		return nil, nil, Pagination{}, err
	}

	// Parse sort
	sorts, err := ParseQuerySort(query, nil) // Pass config if needed
	if err != nil {
		return nil, nil, Pagination{}, err
	}

	// Parse pagination
	pagination, err := ParseQueryPagination(query)
	if err != nil {
		return nil, nil, Pagination{}, err
	}

	return filters, sorts, pagination, nil
}

// ParseQueryWithSearch parses query parameters with global search support.
// Similar to ParseQuery but includes configuration for global search
// functionality across multiple columns.
//
// The searchConfig parameter specifies which columns should be included
// in global search operations when the 'q' parameter is present.
func ParseQueryWithSearch(query map[string][]string, searchConfig *GlobalSearchConfig) ([]Filter, []Sort, Pagination, error) {
	filters, sorts, pagination, err := ParseQuery(query)
	if err != nil {
		return nil, nil, Pagination{}, err
	}

	return filters, sorts, pagination, nil
}

// HTTPRequestParser implements the RequestParser interface for HTTP requests.
// It provides a way to parse query parameters from an HTTP request's URL query.
type HTTPRequestParser struct {
	Query map[string][]string
	SearchConfig *GlobalSearchConfig
}

// NewHTTPRequestParser creates a new HTTPRequestParser from an http.Request
func NewHTTPRequestParser(r *http.Request, searchConfig *GlobalSearchConfig) *HTTPRequestParser {
	return &HTTPRequestParser{
		Query: r.URL.Query(),
		SearchConfig: searchConfig,
	}
}

// ParseFilters implements RequestParser.ParseFilters
func (p *HTTPRequestParser) ParseFilters() ([]Filter, error) {
	return ParseQueryFilters(p.Query)
}

// ParseSorts implements RequestParser.ParseSorts
func (p *HTTPRequestParser) ParseSorts() ([]Sort, error) {
	return ParseQuerySort(p.Query, nil)
}

// ParsePagination implements RequestParser.ParsePagination
func (p *HTTPRequestParser) ParsePagination() (Pagination, error) {
	return ParseQueryPagination(p.Query)
}

// ParseRequest parses all query parameters from an HTTP request.
// This is maintained for backward compatibility with older code.
//
// New code should use the http.ParseRequestHTTP function instead,
// which provides a more type-safe interface.
//
// Example:
//
//	// With an http.Request
//	r, _ := http.NewRequest("GET", "/?name=john&_sort=name", nil)
//	filters, sorts, pagination, err := ParseRequest(r)
func ParseRequest(r interface{}) ([]Filter, []Sort, Pagination, error) {
	// Check if r is an *http.Request
	if req, ok := r.(*http.Request); ok {
		parser := NewHTTPRequestParser(req, nil)
		return ParseFromCustomSource(parser)
	}
	
	return nil, nil, Pagination{}, fmt.Errorf("unsupported request type")
}

// ParseRequestWithSearch parses query parameters with global search support.
// This is maintained for backward compatibility.
func ParseRequestWithSearch(r interface{}, searchConfig *GlobalSearchConfig) ([]Filter, []Sort, Pagination, error) {
	// Check if r is an *http.Request
	if req, ok := r.(*http.Request); ok {
		parser := NewHTTPRequestParser(req, searchConfig)
		return ParseFromCustomSource(parser)
	}
	
	return nil, nil, Pagination{}, fmt.Errorf("unsupported request type")
}
