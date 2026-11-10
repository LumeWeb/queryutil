package generator

import (
	"net/url"
	"testing"

	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
)

func TestGeneratedSearchTextRoundTripsThroughQueryValues(t *testing.T) {
	for _, term := range []string{"hello world", "a+b", "50% done", "00123"} {
		t.Run(term, func(t *testing.T) {
			generated, err := NewDefaultQueryParamGenerator().Generate([]filter.CrudFilter{filter.NewLogicalFilter("q", filter.OpEq, term)})
			if err != nil {
				t.Fatal(err)
			}
			if got := generated["q"]; len(got) != 1 || got[0] != term {
				t.Fatalf("generated q = %v, want raw text %q", got, term)
			}
			values := url.Values(generated)
			filters, err := parser.NewQueryParamParser(values).ParseFilters()
			if err != nil {
				t.Fatal(err)
			}
			if got := filters[0].GetValue(); got != term {
				t.Fatalf("direct parse = %T %v, want %q", got, got, term)
			}
			encoded := values.Encode()
			decoded, err := url.ParseQuery(encoded)
			if err != nil {
				t.Fatal(err)
			}
			filters, err = parser.NewQueryParamParser(decoded).ParseFilters()
			if err != nil {
				t.Fatal(err)
			}
			if got := filters[0].GetValue(); got != term {
				t.Fatalf("URL round trip = %T %v, want %q", got, got, term)
			}
		})
	}
}
