package filter_test

import (
	"errors"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestFilterNodeLimitCountsGroupsAndLeavesBeforeConversion(t *testing.T) {
	tree := []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
		filter.NewLogicalFilter("id", filter.OpEq, 1), filter.NewLogicalFilter("id", filter.OpEq, 2),
	})}
	converted := 0
	policy := filter.QueryPolicy{MaxFilterNodes: 2, Fields: map[string]filter.FieldRule{"id": {Operators: []filter.Operator{filter.OpEq}, Convert: func(_ filter.Operator, value any) (any, error) { converted++; return value, nil }}}}
	if err := filter.ValidateFilterNodeLimit(tree, 2); !errors.Is(err, filter.ErrInvalidQuery) {
		t.Fatalf("pre-policy limit: %v", err)
	}
	if _, err := policy.Normalize(tree, nil); !errors.Is(err, filter.ErrInvalidQuery) || converted != 0 {
		t.Fatalf("normalized above limit: err=%v converted=%d", err, converted)
	}
	policy.MaxFilterNodes = 3
	if _, err := policy.Normalize(tree, nil); err != nil || converted != 2 {
		t.Fatalf("boundary: err=%v converted=%d", err, converted)
	}
	policy.MaxFilterNodes = 0
	if _, err := policy.Normalize(tree, nil); err != nil {
		t.Fatalf("unlimited legacy policy: %v", err)
	}
}
