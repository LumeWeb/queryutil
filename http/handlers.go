package http

import (
	"errors"
	"fmt"
	"go.lumeweb.com/queryutil"
	"go.lumeweb.com/queryutil/filter"
	"gorm.io/gorm"
	"net/http"
)

// ProcessListRequest handles the common pattern for list endpoints.
// This function encapsulates the standard flow for list operations:
// 1. Parse query parameters (filters, sorts, pagination)
// 2. Call the service function to get data
// 3. Convert domain entities to DTOs
// 4. Set appropriate headers
// 5. Encode the response as JSON
//
// Type parameters:
//   - T: The domain entity type returned by the service
//   - D: The DTO (Data Transfer Object) type to be sent to the client
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request
//   - entityName: The name of the entity (used for Content-Range header)
//   - service: A function that retrieves entities with filtering, sorting, and pagination
//   - converter: A function that converts domain entities to DTOs
//
// Example:
//
//	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
//	    err := http.ProcessListRequest(
//	        w, r,
//	        "users",
//	        userService.ListUsers,
//	        func(user User) UserDTO {
//	            return UserDTO{
//	                ID:   user.ID,
//	                Name: user.Name,
//	            }
//	        },
//	    )
//	    if err != nil {
//	        // Error already handled by ProcessListRequest
//	        return
//	    }
//	})
type ProcessOptions struct {
	SearchConfig *filter.GlobalSearchConfig
	SortConfig   *filter.SortConfig
}

type ProcessOption func(*ProcessOptions)

func WithSearchConfig(cfg *filter.GlobalSearchConfig) ProcessOption {
	return func(o *ProcessOptions) {
		o.SearchConfig = cfg
	}
}

func WithSortConfig(cfg *filter.SortConfig) ProcessOption {
	return func(o *ProcessOptions) {
		o.SortConfig = cfg
	}
}

func ProcessListRequest[T any, D any](
	w http.ResponseWriter,
	r *http.Request,
	entityName string,
	service queryutil.EntityFunc[T],
	converter func(T) D,
	opts ...ProcessOption,
) error {

	// Parse query parameters
	// Process options
	options := &ProcessOptions{}
	for _, opt := range opts {
		opt(options)
	}

	parser := queryutil.NewHTTPRequestParser(r, options.SearchConfig, options.SortConfig)
	filters, sorts, pagination, err := queryutil.ParseFromSource(parser)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query parameters: %v", err), http.StatusBadRequest)
		return err
	}

	// Get data from service
	items, total, err := service(filters, sorts, pagination)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, fmt.Sprintf("Entity %s not found", entityName), http.StatusNotFound)
			return err
		}
		http.Error(w, fmt.Sprintf("Failed to list %s: %v", entityName, err), http.StatusInternalServerError)
		return err
	}

	// Convert to DTOs
	responses := make([]D, len(items))
	for i, item := range items {
		responses[i] = converter(item)
	}

	// Set Content-Range header
	SetContentRangeHeader(w, entityName, pagination, responses, total)

	// Encode response using our enhanced JSON encoder
	response := queryutil.BuildResponse(responses, total)
	return EncodeJSON(w, response)
}
