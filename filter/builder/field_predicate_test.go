package builder

import (
	"errors"
	"testing"

	"go.lumeweb.com/queryutil/filter"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCustomFieldPredicateComposesWithScopeAndBooleanGroups(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		"CREATE TABLE ipfs_websites (id integer PRIMARY KEY, user_id integer, name text)",
		"CREATE TABLE website_domains (website_id integer, domain text, deleted_at datetime)",
		"INSERT INTO ipfs_websites VALUES (1,1,'one'),(2,2,'two'),(3,1,'other'),(4,1,'four')",
		"INSERT INTO website_domains VALUES (1,'example.com',NULL),(2,'example.com',NULL),(3,'other.com',NULL),(4,'example.com','2020-01-01')",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	domain := func(op filter.Operator, value any) (string, []any, error) {
		if op != filter.OpEq {
			return "", nil, filter.ErrInvalidQuery
		}
		return "EXISTS (SELECT 1 FROM website_domains wd WHERE wd.website_id = ipfs_websites.id AND wd.domain = ? AND wd.deleted_at IS NULL)", []any{value}, nil
	}
	policy := filter.QueryPolicy{Fields: map[string]filter.FieldRule{
		"domain": {Operators: []filter.Operator{filter.OpEq}},
		"name":   {Operators: []filter.Operator{filter.OpEq}},
	}}
	b := NewGORMBuilderWithOptions(db, nil, WithFieldPredicate("domain", domain))
	filters := []filter.CrudFilter{filter.NewConditionalFilter(filter.LogicalAnd, []filter.CrudFilter{
		filter.NewConditionalFilter(filter.LogicalOr, []filter.CrudFilter{
			filter.NewLogicalFilter("domain", filter.OpEq, "example.com"), filter.NewLogicalFilter("name", filter.OpEq, "other"),
		}),
		filter.NewConditionalFilter(filter.LogicalNot, []filter.CrudFilter{filter.NewLogicalFilter("domain", filter.OpEq, "blocked.com")}),
	})}
	safe, err := policy.Normalize(filters, nil)
	if err != nil {
		t.Fatal(err)
	}
	q, err := b.Apply(db.Table("ipfs_websites").Where("user_id = ?", 1), safe)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	if err := q.Order("id").Pluck("id", &ids).Error; err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 3 {
		t.Fatalf("got %v, want [1 3]", ids)
	}

	// A value that looks like SQL must remain a bound parameter.
	safe, err = policy.Normalize([]filter.CrudFilter{filter.NewLogicalFilter("domain", filter.OpEq, "example.com' OR 1=1 --")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	q, err = b.Apply(db.Table("ipfs_websites").Where("user_id = ?", 1), safe)
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("injected term matched %d, err=%v", count, err)
	}
}

func TestCustomPredicateErrorsAbortQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	expected := errors.New("bad domain")
	b := NewGORMBuilderWithOptions(db, nil, WithFieldPredicate("domain", func(filter.Operator, any) (string, []any, error) { return "", nil, expected }))
	_, err = b.Apply(db, []filter.CrudFilter{filter.NewLogicalFilter("domain", filter.OpEq, "x")})
	if !errors.Is(err, expected) {
		t.Fatalf("got %v, want predicate error", err)
	}
}
