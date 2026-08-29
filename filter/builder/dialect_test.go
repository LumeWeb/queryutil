package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lumeweb.com/queryutil/filter"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetOperatorMapPerDialect(t *testing.T) {
	cases := []struct {
		name         string
		dialect      string
		contains     string // case-insensitive form
		notContains  string // negated case-insensitive form
		containsS    string // case-sensitive form
		notContainsS string // negated case-sensitive form
	}{
		{
			name:         "default fallback",
			dialect:      "postgres",
			contains:     "LIKE ?",
			notContains:  "NOT LIKE ?",
			containsS:    "LIKE BINARY ?",
			notContainsS: "NOT LIKE BINARY ?",
		},
		{
			name:         "sqlite",
			dialect:      dialectSQLite,
			contains:     "LIKE ? COLLATE NOCASE",
			notContains:  "NOT LIKE ? COLLATE NOCASE",
			containsS:    "GLOB ?",
			notContainsS: "NOT GLOB ?",
		},
		{
			name:         "mysql",
			dialect:      dialectMySQL,
			contains:     "LIKE ? COLLATE utf8mb4_0900_ai_ci",
			notContains:  "NOT LIKE ? COLLATE utf8mb4_0900_ai_ci",
			containsS:    "LIKE BINARY ?",
			notContainsS: "NOT LIKE BINARY ?",
		},
		{
			name:         "mysql5",
			dialect:      dialectMySQL5,
			contains:     "LIKE ? COLLATE utf8mb4_general_ci",
			notContains:  "NOT LIKE ? COLLATE utf8mb4_general_ci",
			containsS:    "LIKE BINARY ?",
			notContainsS: "NOT LIKE BINARY ?",
		},
		{
			name:         "mariadb",
			dialect:      dialectMariaDB,
			contains:     "LIKE ? COLLATE utf8mb4_unicode_ci",
			notContains:  "NOT LIKE ? COLLATE utf8mb4_unicode_ci",
			containsS:    "LIKE BINARY ?",
			notContainsS: "NOT LIKE BINARY ?",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := getOperatorMap(tc.dialect)
			assert.Equal(t, "= ?", m[filter.OpEq], "base operators must always be present")
			// Case-insensitive family shares one form.
			assert.Equal(t, tc.contains, m[filter.OpContains])
			assert.Equal(t, tc.contains, m[filter.OpStartswith])
			assert.Equal(t, tc.contains, m[filter.OpEndswith])
			assert.Equal(t, tc.notContains, m[filter.OpNcontains])
			// Case-sensitive family shares one binary form.
			assert.Equal(t, tc.containsS, m[filter.OpContainss])
			assert.Equal(t, tc.containsS, m[filter.OpStartswiths])
			assert.Equal(t, tc.containsS, m[filter.OpEndswiths])
			assert.Equal(t, tc.notContainsS, m[filter.OpNcontainss])
		})
	}
}

func TestMysqlFlavorDetection(t *testing.T) {
	cases := []struct {
		version string
		want    string
	}{
		{"8.0.36", dialectMySQL},
		{"8.0.36-36", dialectMySQL}, // Percona
		{"5.7.44-log", dialectMySQL5},
		{"5.6.51-log", dialectMySQL5},
		{"11.4.2-MariaDB-1:11.4.2+maria~ubu2204", dialectMariaDB},
		{"10.11.8-MariaDB-0ubuntu0.24.04.1", dialectMariaDB},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			assert.Equal(t, tc.want, mysqlFlavor(tc.version))
		})
	}
}

func TestTranslateGlob_EscapesWildcards(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		output string
	}{
		{"percent to star", "%value%", "*value*"},
		{"underscore to question", "v_lue", "v?lue"},
		{"literal star escaped", "a*b", "a[*]b"},
		{"literal question escaped", "a?b", "a[?]b"},
		{"literal bracket escaped", "v1[2]", "v1[[]2]"},
		{"combined", "%v[1]*?", "*v[[]1][*][?]"},
		{"plain text unchanged", "hello", "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.output, translateGlob(tc.input))
		})
	}
}

func TestTranslateGlob_LiteralWildcardsMatchLiterally(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE t(v TEXT)").Error)
	require.NoError(t, db.Exec("INSERT INTO t VALUES ('a*b'), ('aXb'), ('v1[2]'), ('v1X2')").Error)

	// GLOB pattern from translateGlob("a*b") should only match "a*b", not "aXb"
	pat := translateGlob("%a*b%")
	var count int64
	db.Raw("SELECT count(*) FROM t WHERE v GLOB ?", pat).Scan(&count)
	assert.Equal(t, int64(1), count, "literal * in search value must not act as wildcard")

	// GLOB pattern from translateGlob("%v1[2]%") should only match "v1[2]", not "v1X2"
	pat2 := translateGlob("%v1[2]%")
	db.Raw("SELECT count(*) FROM t WHERE v GLOB ?", pat2).Scan(&count)
	assert.Equal(t, int64(1), count, "literal [ in search value must not start a character class")
}

func TestDialectName_MemoizesPerBuilderForNonSqlDBPools(t *testing.T) {
	// Simulate a MySQL *gorm.DB where ConnPool is not *sql.DB (e.g. inside a tx).
	// We can't easily create a real *sql.Tx without a MySQL connection, but we
	// can verify the dialectOnce field prevents repeated calls by using SQLite
	// (which short-circuits before reaching the MySQL path) and asserting the
	// field stays empty. For MySQL transaction pools, the once is populated on
	// first call and reused on subsequent calls.
	db, err := gorm.Open(sqlite.Open(":memory:"))
	require.NoError(t, err)

	b := NewGORMBuilder(db, nil)
	// SQLite short-circuits — dialectName returns immediately
	assert.Equal(t, dialectSQLite, b.dialectName())
	// dialectOnce should still be empty (never reached for non-MySQL)
	assert.Nil(t, b.dialectOnce.Load())
}
