package http

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeJSON(t *testing.T) {
	tests := []struct {
		name           string
		value          interface{}
		expectedBody   string
		expectedHeader string
	}{
		{
			name:           "empty slice",
			value:          []string{},
			expectedBody:   "[]\n",
			expectedHeader: "application/json",
		},
		{
			name:           "empty map",
			value:          map[string]string{},
			expectedBody:   "{}\n",
			expectedHeader: "application/json",
		},
		{
			name:           "string value",
			value:          "test",
			expectedBody:   "\"test\"\n",
			expectedHeader: "application/json",
		},
		{
			name: "struct value",
			value: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{
				Name: "John",
				Age:  30,
			},
			expectedBody:   "{\n\t\"name\": \"John\",\n\t\"age\": 30\n}\n",
			expectedHeader: "application/json",
		},
		{
			name: "slice with values",
			value: []string{
				"one",
				"two",
			},
			expectedBody:   "[\n\t\"one\",\n\t\"two\"\n]\n",
			expectedHeader: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			
			err := EncodeJSON(w, tt.value)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedHeader, w.Header().Get("Content-Type"))
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}
