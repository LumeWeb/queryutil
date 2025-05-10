package queryutil

import (
	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
	"testing"
)

func TestParseFromCustomSource(t *testing.T) {
	mockParser := parser.NewMockParser(t)
	mockParser.EXPECT().ParseFilters().Return([]filter.CrudFilter{
		&filter.LogicalFilter{Field: "name", Operator: filter.OpEq, Value: "test"},
	}, nil)
	mockParser.EXPECT().ParseSorts((*filter.SortConfig)(nil)).Return([]filter.Sort{
		{Field: "name", Order: filter.OrderAsc},
	}, nil)
	mockParser.EXPECT().ParsePagination().Return(filter.Pagination{
		Start:    0,
		End:      10,
		PageSize: 10,
		Mode:     "server",
	}, nil)

	filters, sorts, pagination, err := ParseFromCustomSource(mockParser)

	assert.NoError(t, err)
	assert.Len(t, filters, 1)
	assert.Equal(t, "name", filters[0].(*filter.LogicalFilter).Field)
	assert.Len(t, sorts, 1)
	assert.Equal(t, "name", sorts[0].Field)
	assert.Equal(t, 0, pagination.Start)
	assert.Equal(t, 10, pagination.End)
}
