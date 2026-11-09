package queryutil_test

import (
	"encoding/hex"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"go.lumeweb.com/queryutil"
)

func TestParseRequestWithOptionsPreservesSelectedFields(t *testing.T) {
	digits := strings.Repeat("12345678", 8)
	req := httptest.NewRequest("GET", "/?filters[fingerprint][eq]="+digits+"&filters[age][eq]=30", nil)
	filters, _, _, err := queryutil.ParseRequestWithOptions(req, queryutil.WithStringFields("fingerprint"))
	if err != nil {
		t.Fatal(err)
	}
	values := make(map[string]any)
	for _, f := range filters {
		values[f.GetField()] = f.GetValue()
	}
	if values["fingerprint"] != digits || values["age"] != 30 {
		t.Fatalf("parsed values: %#v", values)
	}
	policy := queryutil.QueryPolicy{Fields: map[string]queryutil.FieldRule{
		"fingerprint": {Operators: []queryutil.Operator{queryutil.OpEq}, Convert: func(_ queryutil.Operator, v any) (any, error) {
			s, ok := v.(string)
			if !ok || len(s) != 64 {
				return nil, errors.New("invalid fingerprint")
			}
			return hex.DecodeString(s)
		}},
		"age": {Operators: []queryutil.Operator{queryutil.OpEq}},
	}}
	safe, err := policy.Normalize(filters, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range safe {
		if f.GetField() == "fingerprint" {
			b, ok := f.GetValue().([]byte)
			if !ok || len(b) != 32 || b[0] != 0x12 || b[31] != 0x78 {
				t.Fatalf("converted fingerprint: %T %v", f.GetValue(), f.GetValue())
			}
		}
	}
}
