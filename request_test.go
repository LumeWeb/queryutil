package queryutil

import (
	"net/http"
	"net/url"
	"sort"
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name           string
		query         url.Values
		wantFilters   []Filter
		wantSorts     []Sort
		wantPagination Pagination
		wantErr       bool
	}{
		{
			name: "complete request",
			query: url.Values{
				"name":    []string{"john"},
				"age_gte": []string{"20"},
				"_sort":   []string{"name,age"},
				"_order":  []string{"desc,asc"},
				"_start":  []string{"0"},
				"_end":    []string{"10"},
			},
			wantFilters: []Filter{
				{Field: "name", Operator: OperatorEquals, Value: "john"},
				{Field: "age", Operator: OperatorGTE, Value: "20"},
			},
			wantSorts: []Sort{
				{Field: "name", Order: OrderDesc},
				{Field: "age", Order: OrderAsc},
			},
			wantPagination: Pagination{
				Start:    0,
				End:      10,
				PageSize: 10,
				Mode:     "server",
			},
			wantErr: false,
		},
		{
			name: "invalid filter operator",
			query: url.Values{
				"status_or": []string{"active,inactive"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{URL: &url.URL{RawQuery: tt.query.Encode()}}
			
			filters, sorts, pagination, err := ParseRequest(r)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			assert.NoError(t, err)
			// Sort both slices by Field to make comparison order-independent
			sort.Slice(filters, func(i, j int) bool {
				return filters[i].Field < filters[j].Field
			})
			sort.Slice(tt.wantFilters, func(i, j int) bool {
				return tt.wantFilters[i].Field < tt.wantFilters[j].Field
			})
			assert.Equal(t, tt.wantFilters, filters)
			assert.Equal(t, tt.wantSorts, sorts)
			assert.Equal(t, tt.wantPagination, pagination)
		})
	}
}
