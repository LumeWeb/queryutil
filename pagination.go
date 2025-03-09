package queryutil

import (
	"strconv"
)

// Pagination represents pagination parameters used for limiting query results.
// It supports both offset/limit style pagination through Start/End parameters
// and includes support for client/server-side pagination modes.
type Pagination struct {
	Start    int    // _start param
	End      int    // _end param
	PageSize int    // calculated
	Mode     string // "server" or "client"
}

// ParseQueryPagination parses and validates pagination parameters from query parameters.
// It handles _start and _end parameters, calculating the appropriate page size.
//
// Validation rules:
//   - _start must be non-negative
//   - _end must be greater than _start
//   - Maximum page size is 100
//
// Returns a PaginationError if validation fails.
func ParseQueryPagination(query map[string][]string) (Pagination, error) {
	pagination := Pagination{
		Start:    0,
		End:      10,
		PageSize: 10,
		Mode:     "server",
	}

	// Parse _start
	if starts, ok := query["_start"]; ok && len(starts) > 0 {
		start, err := strconv.Atoi(starts[0])
		if err != nil {
			return Pagination{}, NewPaginationError("_start", "must be a valid integer")
		}
		if start < 0 {
			return Pagination{}, NewPaginationError("_start", "must be non-negative")
		}
		pagination.Start = start
	}

	// Parse _end
	if ends, ok := query["_end"]; ok && len(ends) > 0 {
		end, err := strconv.Atoi(ends[0])
		if err != nil {
			return Pagination{}, NewPaginationError("_end", "must be a valid integer")
		}
		if end <= pagination.Start {
			return Pagination{}, NewPaginationError("_end", "must be greater than _start")
		}
		pagination.End = end
		pagination.PageSize = pagination.End - pagination.Start
	}

	// Validate page size
	if pagination.PageSize > 100 {
		return Pagination{}, NewPaginationError("pageSize", "cannot exceed 100")
	}

	return pagination, nil
}

// GetLimit returns the number of records to fetch
func (p Pagination) GetLimit() int {
	return p.PageSize
}

// GetOffset calculates the offset for SQL queries
func (p Pagination) GetOffset() int {
	return p.Start
}
