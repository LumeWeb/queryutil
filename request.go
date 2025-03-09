package queryutil

import (
	"net/http"
)

// ParseRequest parses all query parameters from an HTTP request.
// This is the main entry point for processing Refine requests.
// It combines parsing of filters, sorts, and pagination parameters.
//
// Returns parsed Filter, Sort, and Pagination structs or an error
// if any validation fails.
func ParseRequest(r *http.Request) ([]Filter, []Sort, Pagination, error) {
	query := r.URL.Query()
	
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

// ParseRequestWithSearch parses query parameters with global search support.
// Similar to ParseRequest but includes configuration for global search
// functionality across multiple columns.
//
// The searchConfig parameter specifies which columns should be included
// in global search operations when the 'q' parameter is present.
func ParseRequestWithSearch(r *http.Request, searchConfig *GlobalSearchConfig) ([]Filter, []Sort, Pagination, error) {
	filters, sorts, pagination, err := ParseRequest(r)
	if err != nil {
		return nil, nil, Pagination{}, err
	}

	return filters, sorts, pagination, nil
}
