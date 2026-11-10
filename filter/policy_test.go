package filter_test

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func fingerprintPolicy() filter.QueryPolicy {
	return filter.QueryPolicy{
		Fields: map[string]filter.FieldRule{
			"id": {Operators: []filter.Operator{filter.OpEq, filter.OpIn}},
			"fingerprint": {Operators: []filter.Operator{filter.OpEq}, Convert: func(_ filter.Operator, v any) (any, error) {
				s, ok := v.(string)
				if !ok || len(s) != 64 {
					return nil, errors.New("invalid fingerprint")
				}
				return hex.DecodeString(s)
			}},
		},
		SortFields: []string{"id", "fingerprint"},
	}
}

func TestPolicyConvertsNestedFingerprintWithoutMutatingInput(t *testing.T) {
	fingerprint := strings.Repeat("ab", 32)
	leaf := filter.NewLogicalFilter("fingerprint", filter.OpEq, fingerprint)
	input := []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
		filter.NewLogicalFilter("id", filter.OpEq, 1),
		filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{leaf}),
	})}
	safe, err := fingerprintPolicy().Normalize(input, []filter.Sort{{Field: "id", Order: filter.OrderDesc}})
	if err != nil {
		t.Fatal(err)
	}
	got := safe[0].GetFilters()[1].GetFilters()[0].GetValue()
	bytes, ok := got.([]byte)
	if !ok || len(bytes) != 32 || bytes[0] != 0xab {
		t.Fatalf("binary fingerprint = %T %v", got, got)
	}
	if leaf.GetValue() != fingerprint {
		t.Fatal("modified caller's fingerprint")
	}
}

func TestPolicyRejectsOperatorsBuilderCannotApply(t *testing.T) {
	policy := filter.QueryPolicy{Fields: map[string]filter.FieldRule{
		"metadata.status": {Operators: []filter.Operator{filter.OpEq, filter.OpIn, filter.OpNin, filter.OpBetween, filter.OpNbetween}},
		"q":               {Operators: []filter.Operator{filter.OpEq, filter.OpNe, filter.OpContains}},
	}}
	for _, tc := range []struct {
		name string
		item filter.CrudFilter
	}{
		{"json in", filter.NewLogicalFilter("metadata.status", filter.OpIn, []any{"active"})},
		{"json not in", filter.NewLogicalFilter("metadata.status", filter.OpNin, []any{"active"})},
		{"json between", filter.NewLogicalFilter("metadata.status", filter.OpBetween, []any{1, 2})},
		{"json not between", filter.NewLogicalFilter("metadata.status", filter.OpNbetween, []any{1, 2})},
		{"q ne", filter.NewLogicalFilter("q", filter.OpNe, "active")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := policy.Normalize([]filter.CrudFilter{tc.item}, nil); !errors.Is(err, filter.ErrInvalidQuery) {
				t.Fatalf("expected ErrInvalidQuery, got %v", err)
			}
		})
	}
	for _, item := range []filter.CrudFilter{
		filter.NewLogicalFilter("metadata.status", filter.OpEq, "active"),
		filter.NewLogicalFilter("q", filter.OpContains, "active"),
	} {
		if _, err := policy.Normalize([]filter.CrudFilter{item}, nil); err != nil {
			t.Fatalf("supported operator rejected: %v", err)
		}
	}
}

type customFilter struct{ *filter.LogicalFilter }

func TestPolicyRejectsUnsafeOrMalformedInput(t *testing.T) {
	cycle := &filter.ConditionalFilter{Operator: filter.LogicalAnd}
	cycle.Filters = []filter.CrudFilter{cycle}
	cases := []struct {
		name    string
		filters []filter.CrudFilter
		sorts   []filter.Sort
	}{
		{"injected field", []filter.CrudFilter{filter.NewLogicalFilter("id) OR 1=1 --", filter.OpEq, 1)}, nil},
		{"nested unknown field", []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{filter.NewLogicalFilter("tenant_id", filter.OpEq, 2)})}, nil},
		{"disallowed operator", []filter.CrudFilter{filter.NewLogicalFilter("fingerprint", filter.OpNe, "x")}, nil},
		{"bad hex", []filter.CrudFilter{filter.NewLogicalFilter("fingerprint", filter.OpEq, strings.Repeat("zz", 32))}, nil},
		{"bad hex nested", []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{filter.NewLogicalFilter("fingerprint", filter.OpEq, strings.Repeat("zz", 32))})})}, nil},
		{"empty in", []filter.CrudFilter{filter.NewLogicalFilter("id", filter.OpIn, []any{})}, nil},
		{"unknown operator", []filter.CrudFilter{filter.NewLogicalFilter("id", filter.Operator("evil"), 1)}, nil},
		{"short hex", []filter.CrudFilter{filter.NewLogicalFilter("fingerprint", filter.OpEq, "abcd")}, nil},
		{"custom visitor", []filter.CrudFilter{&customFilter{filter.NewLogicalFilter("id", filter.OpEq, 1)}}, nil},
		{"nil", []filter.CrudFilter{nil}, nil},
		{"typed nil", []filter.CrudFilter{(*filter.LogicalFilter)(nil)}, nil},
		{"empty group", []filter.CrudFilter{&filter.ConditionalFilter{Operator: filter.LogicalAnd}}, nil},
		{"bad not", []filter.CrudFilter{&filter.ConditionalFilter{Operator: filter.LogicalNot, Filters: []filter.CrudFilter{filter.NewLogicalFilter("id", filter.OpEq, 1), filter.NewLogicalFilter("id", filter.OpEq, 2)}}}, nil},
		{"cycle", []filter.CrudFilter{cycle}, nil},
		{"bad sort field", nil, []filter.Sort{{Field: "id; DROP TABLE records", Order: filter.OrderAsc}}},
		{"bad sort direction", nil, []filter.Sort{{Field: "id", Order: "asc; SELECT 1"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := fingerprintPolicy().Normalize(tc.filters, tc.sorts); !errors.Is(err, filter.ErrInvalidQuery) {
				t.Fatalf("expected ErrInvalidQuery, got %v", err)
			}
		})
	}
}
