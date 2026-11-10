package parser

import (
	"errors"
	"net/url"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestQueryParamFilterNodeLimitCountsGroupsAndLeaves(t *testing.T) {
	q := url.Values{"filters[or][0][id][eq]": {"1"}, "filters[or][1][id][eq]": {"2"}}
	for _, tc := range []struct {
		limit  int
		reject bool
	}{{2, true}, {3, false}} {
		_, err := NewQueryParamParser(q, WithMaxFilterNodes(tc.limit)).ParseFilters()
		if tc.reject && !errors.Is(err, filter.ErrInvalidQuery) || !tc.reject && err != nil {
			t.Errorf("limit %d: %v", tc.limit, err)
		}
	}
}

func TestJSONFilterNodeLimitCountsGroupsAndLeaves(t *testing.T) {
	input := `[{"operator":"or","value":[{"field":"id","operator":"eq","value":1},{"field":"id","operator":"eq","value":2}]}]`
	for _, tc := range []struct {
		limit  int
		reject bool
	}{{2, true}, {3, false}} {
		_, err := NewJSONParser(input, WithMaxFilterNodes(tc.limit)).ParseFilters()
		if tc.reject && !errors.Is(err, filter.ErrInvalidQuery) || !tc.reject && err != nil {
			t.Errorf("limit %d: %v", tc.limit, err)
		}
	}
}
