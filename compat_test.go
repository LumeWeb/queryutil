package queryutil_test

import (
	"testing"

	"go.lumeweb.com/queryutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type compatRecord struct {
	ID      uint
	Payload string
}

func TestApplyFiltersRetainsBuilderErrorOnGORMQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&compatRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&compatRecord{ID: 1, Payload: `{"status":"active"}`}).Error; err != nil {
		t.Fatal(err)
	}

	filters := []queryutil.CrudFilter{
		queryutil.NewLogicalFilter("id", queryutil.OpEq, 1),
		queryutil.NewLogicalFilter("payload.status", queryutil.OpIn, []any{"active"}),
	}
	query := queryutil.ApplyFilters(db.Model(&compatRecord{}), filters, nil)
	if query.Error == nil {
		t.Fatal("builder error lost")
	}
	var count int64
	if err := query.Count(&count).Error; err == nil {
		t.Fatalf("query executed with missing filter: count %d", count)
	}
	if count != 0 {
		t.Fatalf("query returned rows despite error: %d", count)
	}
}

func TestApplyFiltersErrorDoesNotPoisonInputDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&compatRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&compatRecord{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}

	// The first filter fails before GORMBuilder has cloned the input DB.
	failed := queryutil.ApplyFilters(db, []queryutil.CrudFilter{
		queryutil.NewLogicalFilter("payload.status", queryutil.OpIn, []any{"active"}),
	}, nil)
	if failed.Error == nil {
		t.Fatal("builder error lost")
	}
	if db.Error != nil {
		t.Fatalf("shared DB was poisoned: %v", db.Error)
	}
	var count int64
	if err := db.Model(&compatRecord{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("subsequent query count = %d, err = %v", count, err)
	}
}

func TestApplyFiltersWithOptionsUsesCustomPredicate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&compatRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&compatRecord{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	policy := queryutil.QueryPolicy{Fields: map[string]queryutil.FieldRule{"virtual": {Operators: []queryutil.Operator{queryutil.OpEq}}}}
	filters, err := policy.Normalize([]queryutil.CrudFilter{queryutil.NewLogicalFilter("virtual", queryutil.OpEq, "1")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	query := queryutil.ApplyFiltersWithOptions(db.Model(&compatRecord{}), filters, nil,
		queryutil.WithFieldPredicate("virtual", func(_ queryutil.Operator, value any) (string, []any, error) {
			return "id = ?", []any{value}, nil
		}))
	var count int64
	if err := query.Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("count=%d, err=%v", count, err)
	}
}

func TestApplyFiltersStillExecutesValidQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&compatRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&compatRecord{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	query := queryutil.ApplyFilters(db.Model(&compatRecord{}), []queryutil.CrudFilter{queryutil.NewLogicalFilter("id", queryutil.OpEq, 1)}, nil)
	var count int64
	if err := query.Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("valid count = %d, err = %v", count, err)
	}
}
