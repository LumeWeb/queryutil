package builder

import (
	"errors"
	"testing"

	"go.lumeweb.com/queryutil/filter"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestJSONPathRejectsInvalidBytesBeforeDatabaseExecution(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"meta.a\x00b", "meta.\xff", "meta.a\nb", "meta.a\x7fb"} {
		_, err := NewGORMBuilder(db, nil).Apply(db, []filter.CrudFilter{filter.NewLogicalFilter(field, filter.OpEq, "v")})
		if !errors.Is(err, filter.ErrInvalidQuery) {
			t.Errorf("field %q: got %v", field, err)
		}
	}
}

func TestJSONPathEmptySegmentsMatchEmptyObjectKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE samples (meta text)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO samples (meta) VALUES (?)", `{"":"root","a":{"":{"b":"nested"}}}`).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ field, value string }{{"meta.", "root"}, {"meta.a..b", "nested"}} {
		q, err := NewGORMBuilder(db, nil).Apply(db.Table("samples"), []filter.CrudFilter{filter.NewLogicalFilter(tc.field, filter.OpEq, tc.value)})
		if err != nil {
			t.Fatal(err)
		}
		var count int64
		if err := q.Count(&count).Error; err != nil || count != 1 {
			t.Errorf("field %q: count=%d err=%v", tc.field, count, err)
		}
	}
}
