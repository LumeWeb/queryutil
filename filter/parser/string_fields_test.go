package parser_test

import (
	"net/url"
	"strings"
	"testing"

	"go.lumeweb.com/queryutil/filter/parser"
)

func TestWithStringFieldsPreservesNestedAndArrayValues(t *testing.T) {
	digits := strings.Repeat("12345678", 8)
	query := url.Values{
		"filters[or][0][fingerprint][eq]": {digits},
		"filters[or][1][fingerprint][in]": {"123,456"},
		"filters[or][2][age][eq]":         {"30"},
	}
	filters, err := parser.NewQueryParamParser(query, parser.WithStringFields("fingerprint")).ParseFilters()
	if err != nil {
		t.Fatal(err)
	}
	children := filters[0].GetFilters()
	if got := children[0].GetValue(); got != digits {
		t.Fatalf("equality value = %T %v", got, got)
	}
	values, ok := children[1].GetValue().([]any)
	if !ok || len(values) != 2 || values[0] != "123" || values[1] != "456" {
		t.Fatalf("array value = %T %v", children[1].GetValue(), children[1].GetValue())
	}
	if got := children[2].GetValue(); got != 30 {
		t.Fatalf("other field value = %T %v", got, got)
	}
}

func TestWithStringFieldsRejectsNumericOperatorsWithoutPanicking(t *testing.T) {
	for _, operator := range []string{"gt", "gte", "lt", "lte", "between", "nbetween"} {
		t.Run(operator, func(t *testing.T) {
			values := []string{"123"}
			if operator == "between" || operator == "nbetween" {
				values = []string{"123", "456"}
			}
			query := url.Values{"filters[fingerprint][" + operator + "]": values}
			_, err := parser.NewQueryParamParser(query, parser.WithStringFields("fingerprint")).ParseFilters()
			if err == nil {
				t.Fatal("expected error for numeric operator on string-preserved field")
			}
		})
	}
}

func TestWithoutStringFieldsStillCoercesNumbers(t *testing.T) {
	filters, err := parser.NewQueryParamParser(url.Values{"filters[fingerprint][eq]": {"12345"}}).ParseFilters()
	if err != nil {
		t.Fatal(err)
	}
	if got := filters[0].GetValue(); got != 12345 {
		t.Fatalf("default value = %T %v", got, got)
	}
}
