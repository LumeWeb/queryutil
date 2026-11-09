package filter_test

import (
	"reflect"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestAllScalarFieldRulesIncludesBuilderOperators(t *testing.T) {
	expected := []filter.Operator{
		filter.OpEq, filter.OpNe, filter.OpLt, filter.OpGt, filter.OpLte, filter.OpGte,
		filter.OpIn, filter.OpNin, filter.OpBetween, filter.OpNbetween,
		filter.OpNull, filter.OpNnull, filter.OpContains, filter.OpContainss,
		filter.OpNcontains, filter.OpNcontainss, filter.OpStartswith, filter.OpStartswiths,
		filter.OpNstartswith, filter.OpNstartswiths, filter.OpEndswith, filter.OpEndswiths,
		filter.OpNendswith, filter.OpNendswiths,
	}
	rules := filter.AllScalarFieldRules("id", "created_at")
	if len(rules) != 2 {
		t.Fatalf("fields = %v", rules)
	}
	for _, field := range []string{"id", "created_at"} {
		if !reflect.DeepEqual(rules[field].Operators, expected) {
			t.Fatalf("operators for %s = %v", field, rules[field].Operators)
		}
		if rules[field].Convert != nil {
			t.Fatalf("unexpected converter on %s", field)
		}
	}
}

func TestScalarOperatorsReturnsIndependentCopy(t *testing.T) {
	first := filter.ScalarOperators()
	first[0] = filter.Operator("unsafe")
	if second := filter.ScalarOperators(); second[0] != filter.OpEq {
		t.Fatalf("shared scalar operator list: %v", second)
	}
}

func TestAllScalarFieldRulesDoesNotShareMutableOperators(t *testing.T) {
	first := filter.AllScalarFieldRules("id", "created_at")
	second := filter.AllScalarFieldRules("id")
	first["id"].Operators[0] = filter.Operator("unsafe")
	if first["created_at"].Operators[0] != filter.OpEq || second["id"].Operators[0] != filter.OpEq {
		t.Fatal("mutating one rule changed another")
	}
}
