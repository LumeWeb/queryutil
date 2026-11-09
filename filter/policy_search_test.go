package filter_test

import (
	"errors"
	"net/url"
	"testing"

	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
)

func TestPolicyNormalizesQEqualityToContainsInsideGroups(t *testing.T) {
	leaf := filter.NewLogicalFilter("q", filter.OpEq, "needle")
	input := []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
		filter.NewLogicalFilter("id", filter.OpEq, 1),
		filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{leaf}),
	})}
	var convertedWith filter.Operator
	policy := filter.QueryPolicy{Fields: map[string]filter.FieldRule{
		"id": {Operators: []filter.Operator{filter.OpEq}},
		"q":  {Operators: []filter.Operator{filter.OpContains}, Convert: func(op filter.Operator, value any) (any, error) { convertedWith = op; return value, nil }},
	}}
	safe, err := policy.Normalize(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := safe[0].GetFilters()[1].GetFilters()[0]
	if got.GetOperator() != filter.OpContains || got.GetValue() != "needle" || convertedWith != filter.OpContains {
		t.Fatalf("normalized q = %s %v, conversion op = %s", got.GetOperator(), got.GetValue(), convertedWith)
	}
	if leaf.GetOperator() != filter.OpEq {
		t.Fatal("mutated caller's filter")
	}
}

func TestPolicyAcceptsParsedQEqualityOnlyWhenContainsAllowed(t *testing.T) {
	policy := filter.QueryPolicy{Fields: map[string]filter.FieldRule{"q": {Operators: []filter.Operator{filter.OpContains}}}}
	for _, query := range []url.Values{
		{"q": {"needle"}},
		{"filters[q][eq]": {"needle"}},
	} {
		filters, err := parser.NewQueryParamParser(query).ParseFilters()
		if err != nil {
			t.Fatal(err)
		}
		if filters[0].GetOperator() != filter.OpEq {
			t.Fatalf("parser op = %s", filters[0].GetOperator())
		}
		safe, err := policy.Normalize(filters, nil)
		if err != nil || safe[0].GetOperator() != filter.OpContains {
			t.Fatalf("normalized = %v, err = %v", safe, err)
		}
	}
	filters := []filter.CrudFilter{filter.NewLogicalFilter("q", filter.OpEq, "needle")}
	for _, rule := range []filter.FieldRule{
		{Operators: []filter.Operator{filter.OpEq}},
		{Operators: []filter.Operator{filter.OpNe}},
	} {
		policy.Fields["q"] = rule
		if _, err := policy.Normalize(filters, nil); !errors.Is(err, filter.ErrInvalidQuery) {
			t.Fatalf("rule %v accepted q eq: %v", rule.Operators, err)
		}
	}
	policy.Fields["q"] = filter.FieldRule{Operators: []filter.Operator{filter.OpContains, filter.OpNe}}
	if _, err := policy.Normalize([]filter.CrudFilter{filter.NewLogicalFilter("q", filter.OpNe, "needle")}, nil); !errors.Is(err, filter.ErrInvalidQuery) {
		t.Fatalf("q ne accepted: %v", err)
	}
	if _, err := policy.Normalize([]filter.CrudFilter{filter.NewLogicalFilter("q", filter.OpEq, 42)}, nil); !errors.Is(err, filter.ErrInvalidQuery) {
		t.Fatalf("non-string q accepted: %v", err)
	}
}
