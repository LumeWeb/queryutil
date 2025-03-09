package queryutil

import (
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestApplyFilters(t *testing.T) {
	type User struct {
		ID    uint
		Name  string
		Email string
		Bio   string
	}

	// Setup test DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	db.AutoMigrate(&User{})
	db.Create(&User{Name: "John Doe", Email: "john@example.com", Bio: "Developer"})
	db.Create(&User{Name: "Jane Smith", Email: "jane@example.com", Bio: "Designer"})

	tests := []struct {
		name         string
		filters      []Filter
		searchConfig *GlobalSearchConfig
		wantCount    int64
	}{
		{
			name: "global search across multiple columns",
			filters: []Filter{{
				Field:    "q",
				Operator: OperatorEquals,
				Value:    "john",
			}},
			searchConfig: &GlobalSearchConfig{
				SearchableColumns: []string{"name", "email", "bio"},
			},
			wantCount: 1,
		},
		{
			name: "global search with multiple matches",
			filters: []Filter{{
				Field:    "q",
				Operator: OperatorEquals,
				Value:    "er",
			}},
			searchConfig: &GlobalSearchConfig{
				SearchableColumns: []string{"bio"},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := db
			query = ApplyFilters(query, tt.filters, tt.searchConfig)

			var count int64
			query.Model(&User{}).Count(&count)

			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestParseQueryFilters(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string][]string
		expected []Filter
		wantErr  bool
	}{
		{
			name: "basic equality",
			query: map[string][]string{
				"name": {"john"},
			},
			expected: []Filter{{
				Field:    "name",
				Operator: OperatorEquals,
				Value:    "john",
			}},
			wantErr: false,
		},
		{
			name: "not equals operator",
			query: map[string][]string{
				"age_ne": {"20"},
			},
			expected: []Filter{{
				Field:    "age",
				Operator: OperatorNotEquals,
				Value:    "20",
			}},
			wantErr: false,
		},
		{
			name: "contains operator",
			query: map[string][]string{
				"title_like": {"test"},
			},
			expected: []Filter{{
				Field:    "title",
				Operator: OperatorContains,
				Value:    "test",
			}},
			wantErr: false,
		},
		{
			name: "global search",
			query: map[string][]string{
				"q": {"searchterm"},
			},
			expected: []Filter{{
				Field:    "q",
				Operator: OperatorEquals,
				Value:    "searchterm",
			}},
			wantErr: false,
		},
		{
			name: "unsupported or operator",
			query: map[string][]string{
				"status_or": {"active,inactive"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := ParseQueryFilters(tt.query)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, filters)
		})
	}
}
