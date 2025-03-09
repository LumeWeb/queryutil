package queryutil

// Response represents a Refine-compatible response structure.
// It includes both the data payload and a total count for pagination.
type Response struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
}

// BuildResponse creates a Refine-compatible response object.
// Takes the data payload and total count as parameters.
// The total count is used for client-side pagination calculations.
func BuildResponse(data interface{}, total int64) *Response {
	return &Response{
		Data:  data,
		Total: total,
	}
}
