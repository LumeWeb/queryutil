package builder

import (
	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sort"
	"testing"
)

func TestGORMBuilder_ApplyFilters(t *testing.T) {
	type User struct {
		ID      uint
		Name    string
		Email   string
		Bio     string
		Age     int
		Status  string
		Deleted bool
	}

	// Open database connection with logger
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		t.Fatal(err)
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		t.Fatal(err) // Changed from t.Error to t.Fatal as migration failure is critical
	}

	// Seed data
	db.Create(&User{Name: "John Doe", Email: "john@example.com", Bio: "Developer", Age: 35, Status: "active", Deleted: false})
	db.Create(&User{Name: "Jane Smith", Email: "jane@example.com", Bio: "Designer", Age: 25, Status: "inactive", Deleted: false})
	db.Create(&User{Name: "Peter Jones", Email: "peter@test.com", Bio: "Consultant", Age: 40, Status: "active", Deleted: true}) // Added a third user for better testing

	tests := []struct {
		name         string
		filters      []filter.CrudFilter // Now defined using constructors
		searchConfig *filter.GlobalSearchConfig
		wantCount    int64
		// Optional: Add a function to verify specific results if needed beyond count
		verify func(*testing.T, []User)
	}{
		{
			name: "global search across multiple columns",
			filters: []filter.CrudFilter{
				// Use constructor
				filter.NewLogicalFilter("q", filter.OpContains, "john"),
			},
			searchConfig: &filter.GlobalSearchConfig{
				SearchableColumns: []string{"name", "email", "bio"},
			},
			wantCount: 1,
			verify: func(t *testing.T, users []User) {
				if assert.Len(t, users, 1) { // First check length
					assert.Equal(t, "John Doe", users[0].Name)
				}
			},
		},
		{
			name: "global search with multiple matches",
			filters: []filter.CrudFilter{
				// Use constructor
				filter.NewLogicalFilter("q", filter.OpContains, "er"),
			},
			searchConfig: &filter.GlobalSearchConfig{
				SearchableColumns: []string{"bio"}, // Matches "Developer", "Designer"
			},
			wantCount: 2,
			verify: func(t *testing.T, users []User) {
				// Sorting by Name for predictable assertion order
				sort.SliceStable(users, func(i, j int) bool {
					return users[i].Name < users[j].Name
				})
				assert.Len(t, users, 2)
				assert.Equal(t, "Jane Smith", users[0].Name)
				assert.Equal(t, "John Doe", users[1].Name)
			},
		},
		{
			name: "not operator filter (exclude John Doe)",
			filters: []filter.CrudFilter{
				// Use constructor
				filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
					filter.NewLogicalFilter("name", filter.OpEq, "John Doe"),
				}),
			},
			searchConfig: nil,
			wantCount:    2, // Should get Jane Smith and Peter Jones
			verify: func(t *testing.T, users []User) {
				// Sorting by Name for predictable assertion order
				sort.SliceStable(users, func(i, j int) bool {
					return users[i].Name < users[j].Name
				})
				assert.Len(t, users, 2)
				assert.Equal(t, "Jane Smith", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		{
			name: "complex not combination (Age >= 30 AND (Status = 'active' OR Deleted = false))",
			filters: []filter.CrudFilter{
				// Use constructor
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					// NOT (age < 30) is equivalent to age >= 30
					filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
						filter.NewLogicalFilter("age", filter.OpLt, 30),
					}),
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("status", filter.OpEq, "active"),
						// NOT (deleted = true) is equivalent to deleted = false
						filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{
							filter.NewLogicalFilter("deleted", filter.OpEq, true),
						}),
					}),
				}),
			},
			searchConfig: nil,
			// John Doe: Age 35 (>=30), Status active (active=true) -> Match
			// Jane Smith: Age 25 (<30), Status inactive, Deleted false -> No match (first part fails)
			// Peter Jones: Age 40 (>=30), Status active, Deleted true (deleted=false fails) -> Match (first part true, second part active=true is true)
			// Expected: John Doe, Peter Jones
			wantCount: 2,
			verify: func(t *testing.T, users []User) {
				// Sorting by Name for predictable assertion order
				sort.SliceStable(users, func(i, j int) bool {
					return users[i].Name < users[j].Name
				})
				assert.Len(t, users, 2)
				assert.Equal(t, "John Doe", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		{
			name: "basic equality filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpEq, 25), // Case-sensitive field name test
			},
			searchConfig: nil,
			wantCount:    1,
			verify: func(t *testing.T, users []User) {
				if assert.Len(t, users, 1) { // First check length
					assert.Equal(t, "Jane Smith", users[0].Name)
				}
			},
		},
		{
			name: "combined logical filters (AND implied)",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Status", filter.OpEq, "active"),
				filter.NewLogicalFilter("Age", filter.OpGte, 35),
			},
			searchConfig: nil,
			wantCount:    2, // John Doe (active, >=35) and Peter Jones (active, >=35)
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "John Doe", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		{
			name: "in operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Name", filter.OpIn, []any{"John Doe", "Peter Jones"}),
			},
			searchConfig: nil,
			wantCount:    2,
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "John Doe", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		{
			name: "between operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpBetween, []any{26, 39}),
			},
			searchConfig: nil,
			wantCount:    1, // Only John Doe (Age 35)
			verify: func(t *testing.T, users []User) {
				assert.Len(t, users, 1)
				assert.Equal(t, "John Doe", users[0].Name)
			},
		},
		{
			name: "null operator (for non-null column)",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpNull, nil),
			},
			searchConfig: nil,
			wantCount:    0, // Age is INT, non-nullable by default in GORM/SQLite
		},
		{
			name: "not null operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpNnull, nil),
			},
			searchConfig: nil,
			wantCount:    3, // All users have a non-null Age
		},
		{
			name: "deleted boolean filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Deleted", filter.OpEq, true),
			},
			searchConfig: nil,
			wantCount:    1, // Only Peter Jones
			verify: func(t *testing.T, users []User) {
				assert.Len(t, users, 1)
				assert.Equal(t, "Peter Jones", users[0].Name)
			},
		},
		{
			name: "not deleted boolean filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Deleted", filter.OpEq, false),
			},
			searchConfig: nil,
			wantCount:    2, // John Doe and Jane Smith
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "Jane Smith", users[0].Name)
				assert.Equal(t, "John Doe", users[1].Name)
			},
		},
		{
			name: "or conditional filter",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
					filter.NewLogicalFilter("Status", filter.OpEq, "inactive"),
					filter.NewLogicalFilter("Deleted", filter.OpEq, true),
				}),
			},
			searchConfig: nil,
			wantCount:    2, // Jane Smith (inactive) and Peter Jones (deleted)
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "Jane Smith", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		{
			name: "complex filter with AND and OR",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
					filter.NewLogicalFilter("Age", filter.OpGte, 30),
					filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
						filter.NewLogicalFilter("Status", filter.OpEq, "inactive"),
						filter.NewLogicalFilter("Name", filter.OpContains, "Doe"),
					}),
				}),
			},
			searchConfig: nil,
			// John Doe: Age 35 (>=30), (Status active (no), Name Doe (yes)) -> Match
			// Jane Smith: Age 25 (<30), ... -> No match
			// Peter Jones: Age 40 (>=30), (Status inactive (no), Name Contains Doe (no)) -> TRUE AND (FALSE OR FALSE) -> No match
			// Re-evaluation confirms only John Doe matches.
			wantCount: 1, // Only John Doe (Age 35 AND (Status inactive OR Name contains Doe))
			verify: func(t *testing.T, users []User) {
				assert.Len(t, users, 1)
				assert.Equal(t, "John Doe", users[0].Name)
			},
		},
		// Add more tests for other operators (gt, lt, gte, lte, nin, nbetween, etc.)
		// and more complex combinations.
		{
			name: "greater than filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpGt, 35),
			},
			searchConfig: nil,
			wantCount:    1, // Peter Jones (40)
			verify: func(t *testing.T, users []User) {
				assert.Len(t, users, 1)
				assert.Equal(t, "Peter Jones", users[0].Name)
			},
		},
		{
			name: "less than filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpLt, 30),
			},
			searchConfig: nil,
			wantCount:    1, // Jane Smith (25)
			verify: func(t *testing.T, users []User) {
				assert.Len(t, users, 1)
				assert.Equal(t, "Jane Smith", users[0].Name)
			},
		},
		{
			name: "greater than or equal filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpGte, 35),
			},
			searchConfig: nil,
			wantCount:    2, // John Doe (35), Peter Jones (40)
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "John Doe", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		{
			name: "less than or equal filter",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpLte, 35),
			},
			searchConfig: nil,
			wantCount:    2, // John Doe (35), Jane Smith (25)
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "Jane Smith", users[0].Name)
				assert.Equal(t, "John Doe", users[1].Name)
			},
		},
		{
			name: "not in operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Name", filter.OpNin, []any{"John Doe", "Jane Smith"}),
			},
			searchConfig: nil,
			wantCount:    1, // Peter Jones
			verify: func(t *testing.T, users []User) {
				assert.Len(t, users, 1)
				assert.Equal(t, "Peter Jones", users[0].Name)
			},
		},
		{
			name: "not between operator",
			filters: []filter.CrudFilter{
				filter.NewLogicalFilter("Age", filter.OpNbetween, []any{26, 39}), // Excludes 26-39 range (inclusive)
			},
			searchConfig: nil,
			wantCount:    2, // Jane Smith (25), Peter Jones (40)
			verify: func(t *testing.T, users []User) {
				sort.SliceStable(users, func(i, j int) bool { return users[i].Name < users[j].Name })
				assert.Len(t, users, 2)
				assert.Equal(t, "Jane Smith", users[0].Name)
				assert.Equal(t, "Peter Jones", users[1].Name)
			},
		},
		// Test cases involving nil slices or filters
		{
			name:      "nil filters slice",
			filters:   nil,
			wantCount: 3, // All users
		},
		{
			name:      "empty filters slice",
			filters:   []filter.CrudFilter{},
			wantCount: 3, // All users
		},
		{
			name: "conditional filter with nil filters",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, nil),
			},
			wantCount: 3, // Should likely apply no condition, returning all users, or potentially error depending on builder logic
			// Assuming it applies no condition if filter list is nil/empty for conditional filters.
			// If the builder returns an error for nil/empty filter list in a conditional, adjust wantErr=true.
		},
		{
			name: "conditional filter with empty filters",
			filters: []filter.CrudFilter{
				filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{}),
			},
			wantCount: 3, // Should likely apply no condition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewGORMBuilder(db, tt.searchConfig) // db is the baseTx
			query := db.Model(&User{})

			finalQuery, errApply := builder.Apply(query, tt.filters)

			// Note: GORM builder errors often come *after* the Apply call, during the database execution (Find/First/Count).
			// However, the builder itself might validate filter structures and return an error early.
			// Let's assume for now that structural errors are caught in Apply and database errors during Find.

			// Check for early errors from Apply if any were introduced
			if errApply != nil {
				if tt.wantCount > 0 { // If we expected results, an early error is unexpected
					assert.Fail(t, "Unexpected error from Apply", "Error: %v", errApply)
				} else {
					assert.Error(t, errApply)
					// No need to query the database if Apply failed as expected
				}
				return // Exit the test case if Apply failed
			}

			var users []User
			// Execute the query
			result := finalQuery.Find(&users)

			// Check for database errors during query execution
			assert.NoError(t, result.Error, "Database query failed")

			// Verify the count
			assert.Equal(t, tt.wantCount, int64(result.RowsAffected), "Result count mismatch")

			// Use the verification function if provided
			if tt.verify != nil {
				tt.verify(t, users)
			}
		})
	}
}
