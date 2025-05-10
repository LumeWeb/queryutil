package queryutil

import (
	"go.lumeweb.com/queryutil/filter"
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          url.Values
		wantFilters    []Filter
		wantSorts      []Sort
		wantPagination Pagination
		wantErr        bool
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
			wantFilters: []filter.CrudFilter{
				&filter.LogicalFilter{Field: "name", Operator: filter.OpEq, Value: "john"},
				&filter.LogicalFilter{Field: "age", Operator: filter.OpGte, Value: 20},
			},
			wantSorts: []filter.Sort{
				{Field: "name", Order: filter.OrderDesc},
				{Field: "age", Order: filter.OrderAsc},
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
			// Test ParseQuery directly
			filters, sorts, pagination, err := ParseQuery(tt.query)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			// Sort both slices by Field to make comparison order-independent
			sort.Slice(filters, func(i, j int) bool {
				return filters[i].(*filter.LogicalFilter).Field < filters[j].(*filter.LogicalFilter).Field
			})
			sort.Slice(tt.wantFilters, func(i, j int) bool {
				return tt.wantFilters[i].(*filter.LogicalFilter).Field < tt.wantFilters[j].(*filter.LogicalFilter).Field
			})
			assert.Equal(t, tt.wantFilters, filters)
			assert.Equal(t, tt.wantSorts, sorts)
			assert.Equal(t, tt.wantPagination, pagination)
		})
	}
}

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name           string
		query          url.Values
		wantFilters    []Filter
		wantSorts      []Sort
		wantPagination Pagination
		wantErr        bool
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
			wantFilters: []filter.CrudFilter{
				&filter.LogicalFilter{Field: "name", Operator: filter.OpEq, Value: "john"},
				&filter.LogicalFilter{Field: "age", Operator: filter.OpGte, Value: 20},
			},
			wantSorts: []filter.Sort{
				{Field: "name", Order: filter.OrderDesc},
				{Field: "age", Order: filter.OrderAsc},
			},
			wantPagination: Pagination{
				Start:    0,
				End:      10,
				PageSize: 10,
				Mode:     "server",
			},
			wantErr: false,
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
				return filters[i].(*filter.LogicalFilter).Field < filters[j].(*filter.LogicalFilter).Field
			})
			sort.Slice(tt.wantFilters, func(i, j int) bool {
				return tt.wantFilters[i].(*filter.LogicalFilter).Field < tt.wantFilters[j].(*filter.LogicalFilter).Field
			})
			assert.Equal(t, tt.wantFilters, filters)
			assert.Equal(t, tt.wantSorts, sorts)
			assert.Equal(t, tt.wantPagination, pagination)
		})
	}
}
