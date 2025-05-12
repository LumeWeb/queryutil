package filter

import (
	"fmt"
	"reflect"
	"strconv"
)

// Pagination represents server-side pagination parameters
type Pagination struct {
	Start    int    // Starting record index (inclusive)
	End      int    // Ending record index (exclusive)
	PageSize int    // Calculated page size (End - Start)
	Mode     string // Pagination mode (typically "server")
}

// ParseQueryPagination extracts pagination parameters from URL query values.
// Handles validation of _start and _end parameters with sensible defaults:
// - Default page size: 10
// - Maximum page size: 100
// Returns PaginationError for invalid values
func ParseQueryPagination(query map[string][]string) (Pagination, error) {
	pagination := Pagination{
		Start:    0,
		End:      10,
		PageSize: 10,
		Mode:     "server",
	}

	// Set some reasonable defaults if no values provided
	if _, ok := query["_start"]; !ok {
		query["_start"] = []string{"0"}
	}
	if _, ok := query["_end"]; !ok {
		query["_end"] = []string{"10"}
	}

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

	if pagination.PageSize > 100 {
		return Pagination{}, NewPaginationError("pageSize", "cannot exceed 100")
	}

	return pagination, nil
}

// GetLimit returns the page size for GORM Limit() clause
func (p Pagination) GetLimit() int {
	return p.PageSize
}

// GetOffset returns the starting record index for GORM Offset() clause
func (p Pagination) GetOffset() int {
	return p.Start
}

func FormatContentRange(entityName string, p Pagination, resultCount int, totalCount int) string {
	return fmt.Sprintf("%s %d-%d/%d",
		entityName,
		p.Start,
		p.Start+resultCount-1,
		totalCount,
	)
}

func GetResultCount(data any) int {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return v.Len()
	}
	return 0
}
