package http

import (
	"go.lumeweb.com/queryutil"
	"net/http"
)

// ParseRequestHTTP parses query parameters from an http.Request.
// This is a convenience wrapper around queryutil.ParseQuery that
// extracts the URL query parameters from the request.
//
// Example:
//
//	r, _ := http.NewRequest("GET", "/?name=john&_sort=name", nil)
//	filters, sorts, pagination, err := http.ParseRequestHTTP(r)
func ParseRequestHTTP(r *http.Request) ([]queryutil.Filter, []queryutil.Sort, queryutil.Pagination, error) {
	return queryutil.ParseQuery(r.URL.Query())
}

// ParseRequestWithSearchHTTP parses query parameters with global search support.
// This function is similar to ParseRequestHTTP but also accepts a GlobalSearchConfig
// for handling the special 'q' parameter for searching across multiple columns.
//
// Example:
//
//	searchConfig := &queryutil.GlobalSearchConfig{
//	    SearchableColumns: []string{"name", "email", "bio"},
//	}
//	r, _ := http.NewRequest("GET", "/?q=john&_sort=name", nil)
//	filters, sorts, pagination, err := http.ParseRequestWithSearchHTTP(r, searchConfig)
func ParseRequestWithSearchHTTP(r *http.Request, searchConfig *queryutil.GlobalSearchConfig) ([]queryutil.Filter, []queryutil.Sort, queryutil.Pagination, error) {
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
	data interface{}, totalCount int64) {
	resultCount := queryutil.GetResultCount(data)
	contentRange := queryutil.FormatContentRange(entityName, pagination, resultCount, totalCount)
	w.Header().Set("Content-Range", contentRange)
}
