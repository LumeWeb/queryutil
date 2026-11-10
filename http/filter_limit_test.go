package http_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.lumeweb.com/queryutil"
	queryhttp "go.lumeweb.com/queryutil/http"
)

func TestPolicyListRejectsFilterWidthBeforeService(t *testing.T) {
	policy := queryutil.QueryPolicy{MaxFilterNodes: 2, Fields: map[string]queryutil.FieldRule{"id": {Operators: []queryutil.Operator{queryutil.OpEq}}}}
	q := url.Values{"filters[or][0][id][eq]": {"1"}, "filters[or][1][id][eq]": {"2"}}
	w := httptest.NewRecorder()
	called := false
	err := queryhttp.ProcessListRequestWithPolicy(w, httptest.NewRequest(http.MethodGet, "/?"+q.Encode(), nil), "items", policy,
		func([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination) ([]int, int64, error) {
			called = true
			return nil, 0, nil
		}, func(n int) int { return n })
	if !errors.Is(err, queryutil.ErrInvalidQuery) || w.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v err=%v", w.Code, called, err)
	}
}
