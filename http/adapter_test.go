package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil"
)

func TestParseRequestHTTP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/?name=john&age_gte=20&_sort=name,age&_order=desc,asc&_start=0&_end=10", nil)

	filters, sorts, pagination, err := ParseRequestHTTP(r)

	assert.NoError(t, err)
	assert.Len(t, filters, 2)
	assert.Len(t, sorts, 2)
	assert.Equal(t, 0, pagination.Start)
	assert.Equal(t, 10, pagination.End)
}

func TestSetContentRangeHeader(t *testing.T) {
	w := httptest.NewRecorder()
	data := []string{"item1", "item2", "item3"}
	pagination := queryutil.Pagination{
		Start:    0,
		End:      10,
		PageSize: 10,
		Mode:     "server",
	}

	SetContentRangeHeader(w, "items", pagination, data, 100)

	assert.Equal(t, "items 0-2/100", w.Header().Get("Content-Range"))
}
