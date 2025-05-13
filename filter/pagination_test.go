package filter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseQueryPagination(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string][]string
		expected Pagination
		wantErr  bool
	}{
		{
			name:  "default values",
			query: map[string][]string{},
			expected: Pagination{
				Start:    0,
				End:      10,
				PageSize: 10,
				Mode:     "server",
			},
		},
		{
			name: "custom start and end",
			query: map[string][]string{
				"_start": {"20"},
				"_end":   {"30"},
			},
			expected: Pagination{
				Start:    20,
				End:      30,
				PageSize: 10,
				Mode:     "server",
			},
		},
		{
			name: "invalid start parameter",
			query: map[string][]string{
				"_start": {"invalid"},
			},
			wantErr: true,
		},
		{
			name: "negative start",
			query: map[string][]string{
				"_start": {"-1"},
			},
			wantErr: true,
		},
		{
			name: "end less than start",
			query: map[string][]string{
				"_start": {"10"},
				"_end":   {"5"},
			},
			wantErr: true,
		},
		{
			name: "page size too large",
			query: map[string][]string{
				"_start": {"0"},
				"_end":   {"101"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQueryPagination(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewPagination(t *testing.T) {
	t.Run("valid parameters", func(t *testing.T) {
		p, err := NewPagination(20, 50)
		assert.NoError(t, err)
		assert.Equal(t, Pagination{
			Start:    20,
			End:      70,
			PageSize: 50,
			Mode:     "server",
		}, p)
	})

	t.Run("negative start", func(t *testing.T) {
		_, err := NewPagination(-5, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start: -5 - must be >= 0")
	})

	t.Run("zero page size", func(t *testing.T) {
		_, err := NewPagination(0, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pageSize: 0 - must be > 0")
	})

	t.Run("negative page size", func(t *testing.T) {
		_, err := NewPagination(0, -5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pageSize: -5 - must be > 0")
	})
}

func TestCreatePage(t *testing.T) {
	t.Run("valid page number", func(t *testing.T) {
		p, err := CreatePage(3, 25)
		assert.NoError(t, err)
		assert.Equal(t, Pagination{
			Start:    50, // (3-1)*25 = 50
			End:      75,
			PageSize: 25,
			Mode:     "server",
		}, p)
	})

	t.Run("page 0", func(t *testing.T) {
		_, err := CreatePage(0, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pageNum: 0 - must be >= 1")
	})

	t.Run("negative page number", func(t *testing.T) {
		_, err := CreatePage(-2, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pageNum: -2 - must be >= 1")
	})
}

func TestPaginationPresets(t *testing.T) {
	t.Run("default preset", func(t *testing.T) {
		assert.Equal(t, 10, DefaultPagination.PageSize)
		assert.Equal(t, 0, DefaultPagination.Start)
	})

	t.Run("large preset", func(t *testing.T) {
		assert.Equal(t, 100, LargePagination.PageSize)
		assert.Equal(t, 0, LargePagination.Start)
	})

	t.Run("xlarge preset", func(t *testing.T) {
		assert.Equal(t, 200, XLargePagination.PageSize)
		assert.Equal(t, 0, XLargePagination.Start)
	})

	t.Run("xxlarge preset", func(t *testing.T) {
		assert.Equal(t, 500, XXLargePagination.PageSize)
		assert.Equal(t, 0, XXLargePagination.Start)
	})
}

func TestPaginationHelpers(t *testing.T) {
	p := Pagination{
		Start:    20,
		End:      50,
		PageSize: 30,
		Mode:     "server",
	}

	t.Run("GetLimit", func(t *testing.T) {
		assert.Equal(t, 30, p.GetLimit())
	})

	t.Run("GetOffset", func(t *testing.T) {
		assert.Equal(t, 20, p.GetOffset())
	})
}

func TestFormatContentRange(t *testing.T) {
	pagination := Pagination{
		Start:    10,
		End:      20,
		PageSize: 10,
		Mode:     "server",
	}

	result := FormatContentRange("users", pagination, 5, 100)

	assert.Equal(t, "users 10-14/100", result)
}

func TestGetResultCount(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected int
	}{
		{
			name:     "slice with items",
			data:     []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "empty slice",
			data:     []int{},
			expected: 0,
		},
		{
			name:     "non-slice",
			data:     "not a slice",
			expected: 0,
		},
		{
			name:     "nil",
			data:     nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := GetResultCount(tt.data)
			assert.Equal(t, tt.expected, count)
		})
	}
}
