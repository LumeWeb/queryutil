package http

import (
	"go.lumeweb.com/queryutil"
	"go.lumeweb.com/queryutil/filter/parser"
	"net/http"
	"strconv"
)

// ParseRequestHTTP parses query parameters from an http.Request.
// Maintained for backward compatibility
func ParseRequestHTTP(r *http.Request) ([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination, error) {
	// Use the query param parser directly with the query values
	p := parser.NewQueryParamParser(r.URL.Query())
	return queryutil.ParseFromSource(p)
}

// ParseRequestWithSearchHTTP parses query parameters with global search support
// Deprecated: Use ParseFromSource with a custom parser instead
func ParseRequestWithSearchHTTP(r *http.Request, searchConfig *queryutil.GlobalSearchConfig) ([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination, error) {
	return queryutil.ParseQueryWithSearch(r.URL.Query(), searchConfig)
}

// SetContentRangeHeader sets the Content-Range header for pagination.
// This header is used by clients (like Refine) to understand the pagination
// state and total count of resources.
//
// The format is: "entityName start-end/total"
// For example: "users 0-9/100"
//
// Example:
//
//	w := httptest.NewRecorder()
//	users := []User{...} // 10 users
//	pagination := queryutil.Pagination{Start: 0, End: 10}
//	SetContentRangeHeader(w, "users", pagination, users, 100)
//	// Sets header: Content-Range: users 0-9/100
func SetContentRangeHeader(w http.ResponseWriter, entityName string, pagination queryutil.Pagination,
	data any, totalCount int64) {
	resultCount := queryutil.GetResultCount(data)
	contentRange := queryutil.FormatContentRange(entityName, pagination, resultCount, int(totalCount))
	w.Header().Set("Content-Range", contentRange)
	w.Header().Set("X-Total-Count", strconv.Itoa(resultCount))
}
