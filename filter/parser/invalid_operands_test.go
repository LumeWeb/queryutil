package parser

import (
	"errors"
	"net/url"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestMalformedOperandsReturnInvalidQueryWithoutPanicking(t *testing.T) {
	for _, tc := range []struct {
		name  string
		parse func() ([]filter.CrudFilter, error)
	}{
		{"numeric", func() ([]filter.CrudFilter, error) {
			return NewQueryParamParser(url.Values{"filters[age][gt]": {"oops"}}).ParseFilters()
		}},
		{"date", func() ([]filter.CrudFilter, error) {
			return NewQueryParamParser(url.Values{"filters[created_at][gt]": {"not-a-date"}}).ParseFilters()
		}},
		{"pattern", func() ([]filter.CrudFilter, error) {
			return NewQueryParamParser(url.Values{"filters[name][contains]": {"7"}}).ParseFilters()
		}},
		{"nested", func() ([]filter.CrudFilter, error) {
			return NewQueryParamParser(url.Values{"filters[or][0][age][gt]": {"oops"}}).ParseFilters()
		}},
		{"json pattern", func() ([]filter.CrudFilter, error) {
			return NewJSONParser(`[{"field":"name","operator":"contains","value":7}]`).ParseFilters()
		}},
		{"json nested", func() ([]filter.CrudFilter, error) {
			return NewJSONParser(`[{"operator":"or","value":[{"field":"age","operator":"gt","value":"not-a-date"}]}]`).ParseFilters()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked: %v", r)
				}
			}()
			filters, err := tc.parse()
			if !errors.Is(err, filter.ErrInvalidQuery) || filters != nil {
				t.Fatalf("filters=%v, err=%v", filters, err)
			}
		})
	}
}
