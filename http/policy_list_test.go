package http_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.lumeweb.com/queryutil"
	queryhttp "go.lumeweb.com/queryutil/http"
)

func TestPolicyListRejectsInvalidQueriesBeforeService(t *testing.T) {
	policy := queryutil.QueryPolicy{Fields: map[string]queryutil.FieldRule{"name": {Operators: []queryutil.Operator{queryutil.OpEq}}}}
	for _, q := range []url.Values{{"filters[secret][eq]": {"x"}}, {"filters[name][contains]": {"x"}}, {"filters[name][gt]": {"oops"}}, {"_sort": {"secret"}}} {
		w := httptest.NewRecorder()
		called := false
		err := queryhttp.ProcessListRequestWithPolicy(w, httptest.NewRequest("GET", "/?"+q.Encode(), nil), "items", policy, func([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination) ([]string, int64, error) {
			called = true
			return nil, 0, nil
		}, func(s string) string { return s })
		if err == nil || w.Code != http.StatusBadRequest || called {
			t.Fatalf("%v: status=%d called=%v err=%v", q, w.Code, called, err)
		}
	}
}

func TestPolicyListPreservesRawTextAndResponse(t *testing.T) {
	policy := queryutil.QueryPolicy{Fields: map[string]queryutil.FieldRule{"name": {Operators: []queryutil.Operator{queryutil.OpEq}}}}
	q := url.Values{"filters[name][eq]": {"a%2Bb"}}
	w := httptest.NewRecorder()
	err := queryhttp.ProcessListRequestWithPolicy(w, httptest.NewRequest("GET", "/?"+q.Encode(), nil), "items", policy, func(filters []queryutil.CrudFilter, _ []queryutil.Sort, _ queryutil.Pagination) ([]string, int64, error) {
		if len(filters) != 1 || filters[0].GetValue() != "a%2Bb" {
			t.Errorf("double-decoded: %v", filters)
		}
		return []string{"ok"}, 1, nil
	}, func(s string) string { return s })
	if err != nil || w.Code != http.StatusOK || w.Header().Get("Content-Range") != "items 0-0/1" {
		t.Fatalf("status=%d range=%q err=%v", w.Code, w.Header().Get("Content-Range"), err)
	}
}

func TestPolicyListParserOptionsAndServiceRecheck(t *testing.T) {
	policy := queryutil.QueryPolicy{Fields: map[string]queryutil.FieldRule{
		"external_id": {Operators: []queryutil.Operator{queryutil.OpEq}, Convert: func(_ queryutil.Operator, v any) (any, error) {
			switch value := v.(type) {
			case string:
				return []byte(value), nil
			case []byte:
				return value, nil // Service rechecks an already normalized filter.
			default:
				return nil, fmt.Errorf("text required: %T", v)
			}
		}},
	}}
	q := url.Values{"filters[external_id][eq]": {"007"}}
	w := httptest.NewRecorder()
	err := queryhttp.ProcessListRequestWithPolicy(w, httptest.NewRequest("GET", "/?"+q.Encode(), nil), "items", policy,
		func(filters []queryutil.CrudFilter, sorts []queryutil.Sort, _ queryutil.Pagination) ([]int, int64, error) {
			checked, err := policy.Normalize(filters, sorts)
			if err != nil {
				return nil, 0, err
			}
			if got := string(checked[0].GetValue().([]byte)); got != "007" {
				t.Errorf("converted value = %q", got)
			}
			return []int{1}, 1, nil
		}, func(n int) int { return n }, queryutil.WithStringFields("external_id"))
	if err != nil || w.Code != http.StatusOK {
		t.Fatalf("status=%d err=%v", w.Code, err)
	}
}

func TestPolicyListMapsServiceInvalidQueryToBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	err := queryhttp.ProcessListRequestWithPolicy(w, httptest.NewRequest("GET", "/", nil), "items", queryutil.QueryPolicy{}, func([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination) ([]int, int64, error) {
		return nil, 0, queryutil.ErrInvalidQuery
	}, func(n int) int { return n })
	if !errors.Is(err, queryutil.ErrInvalidQuery) || w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d err=%v", w.Code, err)
	}
}
