package queryutil

// RequestParser defines an interface for parsing query parameters from any source.
// This interface allows queryutil to be extended to work with different request types
// beyond HTTP requests, such as gRPC, WebSockets, or custom protocols.
//
// Implementations must provide methods to parse filters, sorts, and pagination
// parameters from their specific request format.
type RequestParser interface {
	ParseFilters() ([]Filter, error)
	ParseSorts() ([]Sort, error)
	ParsePagination() (Pagination, error)
}

// EntityFunc is a generic function type for entity list operations.
// It takes filters, sorts, and pagination parameters and returns
// a slice of items, total count, and an error.
//
// This type is commonly used for service layer functions that retrieve
// entities from a data source with filtering, sorting, and pagination.
//
// Example implementation:
//
//	func ListUsers(filters []Filter, sorts []Sort, pagination Pagination) ([]User, int64, error) {
//	    db := database.GetDB()
//	    query := db.Model(&User{})
//	    
//	    // Apply filters and sorts
//	    query = ApplyFilters(query, filters, nil)
//	    query = ApplySort(query, sorts)
//	    
//	    // Count total before pagination
//	    var total int64
//	    query.Count(&total)
//	    
//	    // Apply pagination
//	    query = query.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())
//	    
//	    // Execute query
//	    var users []User
//	    err := query.Find(&users).Error
//	    return users, total, err
//	}
type EntityFunc[T any] func([]Filter, []Sort, Pagination) ([]T, int64, error)

// ParseFromCustomSource allows using queryutil with custom request sources
// by accepting any implementation of the RequestParser interface.
//
// This function provides a way to extend queryutil to work with different
// request types beyond HTTP requests.
//
// Example with a custom parser:
//
//	type MyCustomParser struct {
//	    // Custom fields
//	}
//	
//	func (p *MyCustomParser) ParseFilters() ([]Filter, error) {
//	    // Custom implementation
//	}
//	
//	func (p *MyCustomParser) ParseSorts() ([]Sort, error) {
//	    // Custom implementation
//	}
//	
//	func (p *MyCustomParser) ParsePagination() (Pagination, error) {
//	    // Custom implementation
//	}
//	
//	// Usage
//	parser := &MyCustomParser{}
//	filters, sorts, pagination, err := ParseFromCustomSource(parser)
func ParseFromCustomSource(parser RequestParser) ([]Filter, []Sort, Pagination, error) {
	filters, err := parser.ParseFilters()
	if err != nil {
		return nil, nil, Pagination{}, err
	}

	sorts, err := parser.ParseSorts()
	if err != nil {
		return filters, nil, Pagination{}, err
	}

	pagination, err := parser.ParsePagination()
	if err != nil {
		return filters, sorts, Pagination{}, err
	}

	return filters, sorts, pagination, nil
}
