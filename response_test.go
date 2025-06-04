package queryutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildResponse(t *testing.T) {
	type TestData struct {
		ID   int
		Name string
	}

	data := []TestData{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
	}

	response := BuildResponse(data, 10)

	assert.Equal(t, data, response.Data)
	assert.Equal(t, int64(10), response.Total)
}
