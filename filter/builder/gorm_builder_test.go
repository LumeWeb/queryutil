package builder

import (
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"

	"go.lumeweb.com/queryutil/filter"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGORMBuilder_ApplyFilters(t *testing.T) {
	type User struct {
		ID    uint
		Name    string
		Email   string
		Bio     string
		Age     int
		Status  string
		Deleted bool
	}

	// Setup test DB
	// Create new logger with configuration
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound
			ParameterizedQueries:      true,        // Don't include params
			Colorful:                  false,       // Disable color
		},
	)

	// Open database connection with logger
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		t.Error(t)
	}
	db.Create(&User{Name: "John Doe", Email: "john@example.com", Bio: "Developer", Age: 35, Status: "active", Deleted: false})
	db.Create(&User{Name: "Jane Smith", Email: "jane@example.com", Bio: "Designer", Age: 25, Status: "inactive", Deleted: false})

	tests := []struct {
		name         string
		filters      []filter.CrudFilter
		searchConfig *filter.GlobalSearchConfig
		wantCount    int64
	}{
		{
			name: "global search across multiple columns",
			filters: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "q",
					Operator: filter.OpContains,
					Value:    "john",
				},
			},
			searchConfig: &filter.GlobalSearchConfig{
				SearchableColumns: []string{"name", "email", "bio"},
			},
			wantCount: 1,
		},
		{
			name: "global search with multiple matches",
			filters: []filter.CrudFilter{
				&filter.LogicalFilter{
					Field:    "q",
					Operator: filter.OpContains,
					Value:    "er",
				},
			},
			searchConfig: &filter.GlobalSearchConfig{
				SearchableColumns: []string{"bio"},
			},
			wantCount: 2,
		},
		{
			name: "not operator filter",
			filters: []filter.CrudFilter{
				&filter.ConditionalFilter{
					Operator: filter.LogicalNot,
					Filters: []filter.CrudFilter{
						&filter.LogicalFilter{
							Field:    "name",
							Operator: filter.OpEq,
							Value:    "John Doe",
						},
					},
				},
			},
			searchConfig: nil,
			wantCount:    1,
		},
		{
			name: "complex not combination",
			filters: []filter.CrudFilter{
				&filter.ConditionalFilter{
					Operator: filter.LogicalAnd,
					Filters: []filter.CrudFilter{
						&filter.ConditionalFilter{
							Operator: filter.LogicalNot,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "age",
									Operator: filter.OpLt,
									Value:    30,
								},
							},
						},
						&filter.ConditionalFilter{
							Operator: filter.LogicalOr,
							Filters: []filter.CrudFilter{
								&filter.LogicalFilter{
									Field:    "status",
									Operator: filter.OpEq,
									Value:    "active",
								},
								&filter.ConditionalFilter{
									Operator: filter.LogicalNot,
									Filters: []filter.CrudFilter{
										&filter.LogicalFilter{
											Field:    "deleted",
											Operator: filter.OpEq,
											Value:    true,
										},
									},
								},
							},
						},
					},
				},
			},
			searchConfig: nil,
			wantCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewGORMBuilder(db, tt.searchConfig) // db is the baseTx
			query := db.Model(&User{})

			finalQuery, errApply := builder.Apply(query, tt.filters)
			assert.NoError(t, errApply)
			var users []User
			result := finalQuery.Find(&users)
			assert.NoError(t, result.Error)
			assert.Equal(t, tt.wantCount, int64(len(users)))

			// Verify actual records match expected filters
			for _, user := range users {
				switch tt.name {
				case "not operator filter":
					assert.Contains(t, user.Name, "Jane")
				case "complex not combination":
					assert.Contains(t, user.Name, "John")
				}
			}
		})
	}
}
