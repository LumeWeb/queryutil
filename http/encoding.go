package http

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// EncodeJSON encodes a value as JSON and writes it to the http.ResponseWriter.
// It handles empty slices and maps specially, encoding them as "[]" and "{}" respectively
// instead of "null". This provides a more consistent API response format.
//
// Example:
//
//	w := httptest.NewRecorder()
//	var emptySlice []string
//	EncodeJSON(w, emptySlice) // Writes "[]" instead of "null"
func EncodeJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")

	// Handle empty slices and maps specially
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice && val.Len() == 0 {
		_, err := w.Write([]byte("[]\n"))
		return err
	} else if val.Kind() == reflect.Map && val.Len() == 0 {
		_, err := w.Write([]byte("{}\n"))
		return err
	}

	// Use standard JSON encoding for other values
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	return enc.Encode(v)
}
