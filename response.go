package queryutil

// Response represents a standard response structure for list operations.
// It includes both the data payload and a total count for pagination.
// This structure is compatible with Refine's Simple REST Data Provider.
type Response[T any] struct {
	Data  T     `json:"data"`
	Total int64 `json:"total"`
}

// BuildResponse creates a standard response object for list operations.
// Takes the data payload and total count as parameters.
// The total count is used for client-side pagination calculations.
//
// Example:
//
//	users := []User{...}
//	totalCount := int64(100)
//	response := BuildResponse(users, totalCount)
//	// response can be encoded to JSON and sent to the client
func BuildResponse[T any](data T, total int64) *Response[T] {
	return &Response[T]{
		Data:  data,
		Total: total,
	}
}
