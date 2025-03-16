package queryutil

// Response represents a standard response structure for list operations.
// It includes both the data payload and a total count for pagination.
// This structure is compatible with Refine's Simple REST Data Provider.
type Response struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
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
func BuildResponse(data interface{}, total int64) *Response {
	return &Response{
		Data:  data,
		Total: total,
	}
}
