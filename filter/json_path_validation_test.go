package filter_test

import (
	"errors"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestPolicyRejectsInvalidJSONPathBytesButAllowsEmptyKeys(t *testing.T) {
	for _, field := range []string{"meta.a\x00b", "meta.\xff", "meta.a\nb", "meta.a\x7fb"} {
		policy := filter.QueryPolicy{Fields: map[string]filter.FieldRule{field: {Operators: []filter.Operator{filter.OpEq}}}}
		_, err := policy.Normalize([]filter.CrudFilter{filter.NewLogicalFilter(field, filter.OpEq, "value")}, nil)
		if !errors.Is(err, filter.ErrInvalidQuery) {
			t.Errorf("field %q: got %v", field, err)
		}
	}
	for _, field := range []string{"meta.", "meta.a..b"} {
		policy := filter.QueryPolicy{Fields: map[string]filter.FieldRule{field: {Operators: []filter.Operator{filter.OpEq}}}}
		if _, err := policy.Normalize([]filter.CrudFilter{filter.NewLogicalFilter(field, filter.OpEq, "value")}, nil); err != nil {
			t.Errorf("valid empty key %q: %v", field, err)
		}
	}
}
