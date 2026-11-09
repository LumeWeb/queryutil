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
