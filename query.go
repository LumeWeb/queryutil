package queryutil

import (
	"net/url"
	"sort"
	"strings"
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
