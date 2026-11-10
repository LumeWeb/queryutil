package http_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.lumeweb.com/queryutil"
	queryhttp "go.lumeweb.com/queryutil/http"
)

func TestPolicyListDoesNotExposeServiceErrorsInServerResponse(t *testing.T) {
	internal := errors.New("SQLSTATE 42P01: relation internal_accounts does not exist")
	w := httptest.NewRecorder()
	err := queryhttp.ProcessListRequestWithPolicy(w, httptest.NewRequest(http.MethodGet, "/", nil), "items",
		queryutil.QueryPolicy{}, func([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination) ([]int, int64, error) {
			return nil, 0, internal
		}, func(n int) int { return n })
	if !errors.Is(err, internal) || w.Code != http.StatusInternalServerError || w.Body.String() != "Internal Server Error\n" {
		t.Fatalf("status=%d body=%q err=%v", w.Code, w.Body.String(), err)
	}
}

func TestLegacyListPreservesServiceErrorResponse(t *testing.T) {
	internal := errors.New("database unavailable")
	w := httptest.NewRecorder()
	err := queryhttp.ProcessListRequest(w, httptest.NewRequest(http.MethodGet, "/", nil), "items",
		func([]queryutil.CrudFilter, []queryutil.Sort, queryutil.Pagination) ([]int, int64, error) {
			return nil, 0, internal
		}, func(n int) int { return n })
	if !errors.Is(err, internal) || w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "database unavailable") {
		t.Fatalf("status=%d body=%q err=%v", w.Code, w.Body.String(), err)
	}
}
