package queryutil

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

// MockRequestParser implements the RequestParser interface for testing
type MockRequestParser struct {
	filters    []Filter
	sorts      []Sort
	pagination Pagination
	filterErr  error
	sortErr    error
	pageErr    error
}

func (m *MockRequestParser) ParseFilters() ([]Filter, error) {
	return m.filters, m.filterErr
}

func (m *MockRequestParser) ParseSorts() ([]Sort, error) {
	return m.sorts, m.sortErr
}

func (m *MockRequestParser) ParsePagination() (Pagination, error) {
	return m.pagination, m.pageErr
}

func TestParseFromCustomSource(t *testing.T) {
	mockParser := &MockRequestParser{
		filters: []Filter{
			{Field: "name", Operator: OperatorEquals, Value: "test"},
		},
		sorts: []Sort{
			{Field: "name", Order: OrderAsc},
		},
		pagination: Pagination{
			Start:    0,
			End:      10,
			PageSize: 10,
			Mode:     "server",
		},
	}
	
	filters, sorts, pagination, err := ParseFromCustomSource(mockParser)
	
	assert.NoError(t, err)
	assert.Len(t, filters, 1)
	assert.Equal(t, "name", filters[0].Field)
	assert.Len(t, sorts, 1)
	assert.Equal(t, "name", sorts[0].Field)
	assert.Equal(t, 0, pagination.Start)
	assert.Equal(t, 10, pagination.End)
}
