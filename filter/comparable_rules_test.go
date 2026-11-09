package filter_test

import (
	"reflect"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestComparableOperatorsExcludesPatternAndNullOperators(t *testing.T) {
	want := []filter.Operator{
		filter.OpEq, filter.OpNe, filter.OpLt, filter.OpGt, filter.OpLte, filter.OpGte,
		filter.OpIn, filter.OpNin, filter.OpBetween, filter.OpNbetween,
	}
	if got := filter.ComparableOperators(); !reflect.DeepEqual(got, want) {
		t.Fatalf("comparable operators = %v", got)
	}
	if got := filter.AllScalarFieldRules("id")["id"].Operators; len(got) != 24 {
		t.Fatalf("scalar operator count = %d, want 24", len(got))
	}
}

func TestComparableFieldRulesKeepOperatorSlicesIndependent(t *testing.T) {
	first := filter.ComparableFieldRules("id", "created_at")
	second := filter.ComparableFieldRules("id")
	if len(first) != 2 || first["id"].Convert != nil {
		t.Fatalf("field rules = %v", first)
	}
	if !reflect.DeepEqual(first["id"].Operators, filter.ComparableOperators()) {
		t.Fatalf("wrong operators for id: %v", first["id"].Operators)
	}
	first["id"].Operators[0] = filter.Operator("unsafe")
	if first["created_at"].Operators[0] != filter.OpEq || second["id"].Operators[0] != filter.OpEq {
		t.Fatal("mutating one field changed another")
	}
	operators := filter.ComparableOperators()
	operators[0] = filter.Operator("unsafe")
	if filter.ComparableOperators()[0] != filter.OpEq {
		t.Fatal("mutating returned slice changed preset")
	}
}
