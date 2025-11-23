package queryutil

import (
	"net/url"
	"sort"
	"strings"

	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/serializer"
)

// ToQueryString converts a map of query parameters to a URL-encoded query string.
// The keys are sorted alphabetically to ensure consistent output.
// Example:
//
//	params := map[string][]string{
//	    "name": {"john"},
//	    "age":  {"30"},
//	}
//
// query := queryutil.ToQueryString(params) // "age=30&name=john"
func ToQueryString(params map[string][]string) string {
	if len(params) == 0 {
		return ""
	}

	// Get sorted keys for consistent output
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, key := range keys {
		values := params[key]
		keyEscaped := url.QueryEscape(key)

		for _, value := range values {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			if value != "" { // Only write =value if value is not empty
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(value))
			}

		}
	}

	return buf.String()
}

// BuildURL creates a complete URL with query parameters for filters, sorts, and pagination.
// It parses the base URL using url.Parse, then uses the QueryParamSerializer to convert
// the structures to query parameters and merges them with any existing query parameters.
// Later sources win on key clashes.
//
// Parameters:
//   - baseURL: The base URL (e.g., "https://api.example.com/users" or "https://api.example.com/users?existing=param")
//   - sorts: Slice of Sort structures for ordering
//   - pagination: Pagination structure for limiting results (nil to use DefaultPagination)
//   - filters: Variadic CrudFilter structures for filtering
//
// Returns:
//   - Complete URL with query string
//   - Error if URL parsing or serialization fails
//
// Example:
//
//	baseURL := "https://api.example.com/users"
//	sorts := []Sort{{Field: "name", Order: OrderAsc}}
//	pagination := nil // Uses DefaultPagination
//	filters := []CrudFilter{NewLogicalFilter("age", OpGte, 18)}
//
//	url, err := BuildURL(baseURL, sorts, pagination, filters...)
//	// Result: "https://api.example.com/users?_end=10&_order=asc&_sort=name&_start=0&filters[age][gte]=18"
func BuildURL(baseURL string, sorts []Sort, pagination *filter.Pagination, filters ...CrudFilter) (string, error) {
	// Parse the base URL to handle existing query parameters
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Create serializer with default options
	_serializer := serializer.NewQueryParamSerializer()

	// Start with existing query parameters from the base URL
	allParams := make(map[string][]string)
	if parsedURL.Query() != nil {
		mergeParams(allParams, parsedURL.Query())
	}

	// Serialize filters if provided
	if len(filters) > 0 {
		filterValues, err := _serializer.SerializeFilters(filters)
		if err != nil {
			return "", err
		}
		mergeParams(allParams, filterValues)
	}

	// Serialize sorts if provided
	if len(sorts) > 0 {
		sortValues, err := _serializer.SerializeSorts(sorts)
		if err != nil {
			return "", err
		}
		mergeParams(allParams, sortValues)
	}

	// Serialize pagination (use default if nil)
	paginationToUse := pagination
	if paginationToUse == nil {
		paginationToUse = &filter.DefaultPagination
	}
	paginationValues, err := _serializer.SerializePagination(*paginationToUse)
	if err != nil {
		return "", err
	}
	mergeParams(allParams, paginationValues)

	// Build query string
	queryString := ToQueryString(allParams)

	// Set the new query parameters on the parsed URL
	parsedURL.RawQuery = queryString

	// Return the complete URL
	return parsedURL.String(), nil
}

// mergeParams merges url.Values into a map[string][]string
func mergeParams(target map[string][]string, source url.Values) {
	for key, values := range source {
		target[key] = values
	}
}
