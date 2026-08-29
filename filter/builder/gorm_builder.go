package builder

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"go.lumeweb.com/queryutil/filter"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	sqlIsNull     = "IS NULL"
	sqlIsNotNull  = "IS NOT NULL"
	sqlNotBetween = "NOT BETWEEN ? AND ?"
	sqlIn         = "IN (?)"
	sqlNotIn      = "NOT IN (?)"
	sqlBetween    = "BETWEEN ? AND ?"

	dialectSQLite  = "sqlite"
	dialectMySQL   = "mysql"
	dialectMySQL5  = "mysql5" // MySQL 5.x lacks utf8mb4_0900_ai_ci
	dialectMariaDB = "mariadb"
)

// Base operators that are common across all databases
var baseOperators = map[filter.Operator]string{
	filter.OpEq:       "= ?",
	filter.OpNe:       "<> ?",
	filter.OpLt:       "< ?",
	filter.OpGt:       "> ?",
	filter.OpLte:      "<= ?",
	filter.OpGte:      ">= ?",
	filter.OpNull:     sqlIsNull,
	filter.OpNnull:    sqlIsNotNull,
	filter.OpIn:       sqlIn,
	filter.OpNin:      sqlNotIn,
	filter.OpNbetween: sqlNotBetween,
	filter.OpBetween:  sqlBetween,
}

// dialectPattern captures the four LIKE forms a dialect uses for pattern
// matching: case-insensitive (ci) and case-sensitive (cs), each in a positive
// and negated variant. Case-insensitive forms carry the dialect's collation;
// case-sensitive forms are binary. SQLite's LIKE ignores COLLATE for case
// sensitivity, so its case-sensitive forms use GLOB, which is always binary.
// Because GLOB uses different wildcards (* and ? instead of % and _),
// csTranslate must be true for the SQLite pattern to activate wildcard
// translation on the bound value.
type dialectPattern struct {
	ciLike, ciNotLike string
	csLike, csNotLike string
	csTranslate       bool // true when cs forms use GLOB wildcards (* ?) not LIKE (% _)
}

// dialectPatterns holds the pattern operator forms per dialect. The empty-string
// key is the fallback for unknown dialects (plain LIKE, no collation support).
var dialectPatterns = map[string]dialectPattern{
	"": {
		ciLike: "LIKE ?", ciNotLike: "NOT LIKE ?",
		csLike: "LIKE BINARY ?", csNotLike: "NOT LIKE BINARY ?",
	},
	dialectSQLite: {
		ciLike: "LIKE ? COLLATE NOCASE", ciNotLike: "NOT LIKE ? COLLATE NOCASE",
		csLike: "GLOB ?", csNotLike: "NOT GLOB ?",
		csTranslate: true,
	},
	dialectMySQL: {
		// utf8mb4_0900_ai_ci is a MySQL 8.0+ collation.
		ciLike: "LIKE ? COLLATE utf8mb4_0900_ai_ci", ciNotLike: "NOT LIKE ? COLLATE utf8mb4_0900_ai_ci",
		csLike: "LIKE BINARY ?", csNotLike: "NOT LIKE BINARY ?",
	},
	dialectMySQL5: {
		// MySQL 5.x does not support utf8mb4_0900_ai_ci; use general_ci.
		ciLike: "LIKE ? COLLATE utf8mb4_general_ci", ciNotLike: "NOT LIKE ? COLLATE utf8mb4_general_ci",
		csLike: "LIKE BINARY ?", csNotLike: "NOT LIKE BINARY ?",
	},
	dialectMariaDB: {
		// MariaDB does not support utf8mb4_0900_ai_ci; use a portable unicode collation.
		ciLike: "LIKE ? COLLATE utf8mb4_unicode_ci", ciNotLike: "NOT LIKE ? COLLATE utf8mb4_unicode_ci",
		csLike: "LIKE BINARY ?", csNotLike: "NOT LIKE BINARY ?",
	},
}

// patternOpDef describes one LIKE/NOT LIKE pattern operator together with
// whether it matches case-sensitively and whether it is negated. Its SQL form is
// resolved per dialect; its bound value pattern is handled by formatValue.
type patternOpDef struct {
	op        filter.Operator
	sensitive bool
	negative  bool
}

// patternOpDefs is the single source of truth for the pattern operator set; it
// drives both the per-dialect operator map and its size, so the count is never
// hardcoded.
var patternOpDefs = []patternOpDef{
	{filter.OpContains, false, false}, {filter.OpContainss, true, false},
	{filter.OpNcontains, false, true}, {filter.OpNcontainss, true, true},
	{filter.OpStartswith, false, false}, {filter.OpStartswiths, true, false},
	{filter.OpNstartswith, false, true}, {filter.OpNstartswiths, true, true},
	{filter.OpEndswith, false, false}, {filter.OpEndswiths, true, false},
	{filter.OpNendswith, false, true}, {filter.OpNendswiths, true, true},
}

// buildPatternOperators expands a dialect's four LIKE forms into the full set of
// pattern operators defined by patternOpDefs.
func buildPatternOperators(p dialectPattern) map[filter.Operator]string {
	m := make(map[filter.Operator]string, len(patternOpDefs))
	for _, def := range patternOpDefs {
		form := p.ciLike
		switch {
		case def.sensitive && !def.negative:
			form = p.csLike
		case !def.sensitive && def.negative:
			form = p.ciNotLike
		case def.sensitive && def.negative:
			form = p.csNotLike
		}
		m[def.op] = form
	}
	return m
}

// translateGlob converts a LIKE pattern (% and _ wildcards) to a GLOB pattern
// (* and ? wildcards). Literal * and ? in the value are escaped using SQLite
// GLOB's character class syntax ([*] and [?]), since GLOB does not support an
// ESCAPE clause.
func translateGlob(pattern string) string {
	var b strings.Builder
	for _, r := range pattern {
		switch r {
		case '%':
			b.WriteByte('*')
		case '_':
			b.WriteByte('?')
		case '*':
			b.WriteString("[*]")
		case '?':
			b.WriteString("[?]")
		case '[':
			b.WriteString("[[]")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// operatorMaps is the precomputed, immutable operator mapping per dialect. It
// combines the dialect-independent base operators with the dialect pattern
// forms, so lookups do not rebuild a map on every call.
var operatorMaps = func() map[string]map[filter.Operator]string {
	maps := make(map[string]map[filter.Operator]string, len(dialectPatterns))
	for dialect, p := range dialectPatterns {
		full := make(map[filter.Operator]string, len(baseOperators)+len(patternOpDefs))
		for k, v := range baseOperators {
			full[k] = v
		}
		for k, v := range buildPatternOperators(p) {
			full[k] = v
		}
		maps[dialect] = full
	}
	return maps
}()

// getOperatorMap returns the operator-to-SQL mapping for the given dialect.
// Patterns are precomputed once per dialect; unknown dialects fall back to plain
// LIKE matching.
func getOperatorMap(dialectName string) map[filter.Operator]string {
	if m, ok := operatorMaps[dialectName]; ok {
		return m
	}
	return operatorMaps[""]
}

// dialectCache memoizes dialect detection per root *sql.DB. The key is the
// physical connection pool, which is stable for the lifetime of a database —
// even across transactions — so the cache does not grow with connection churn.
var dialectCache sync.Map // map[*sql.DB]string

type GORMBuilder struct {
	baseTx       *gorm.DB // The original DB connection/transaction
	searchConfig *filter.GlobalSearchConfig

	// dialectOnce memoizes the resolved MySQL dialect per builder instance. This
	// covers transaction pools (*sql.Tx) and prepared-statement wrappers
	// (gorm.PreparedStmtDB) where the *sql.DB type assertion in dialectName()
	// fails, ensuring SELECT VERSION() runs at most once per builder lifetime.
	dialectOnce atomic.Value // string

	// knownTables holds the table/alias names present in the query's FROM and JOINs
	// (lowercased), used to disambiguate qualified columns from JSON paths in dotted
	// field names. It is per-query state carried on a per-Apply copy of the builder so
	// the shared receiver handed to NewGORMBuilder stays immutable and race-free.
	knownTables map[string]struct{}
}

func NewGORMBuilder(baseTx *gorm.DB, searchConfig *filter.GlobalSearchConfig) *GORMBuilder {
	return &GORMBuilder{
		baseTx:       baseTx,
		searchConfig: searchConfig,
	}
}

// dialectName returns the effective database dialect for this builder's
// underlying connection. Resolution is lazy: the first call for a given MySQL
// connection pool queries SELECT VERSION() and caches the result on the root
// *sql.DB, so subsequent builders sharing the same pool pay no round-trip.
// Non-MySQL dialects return immediately.
func (b *GORMBuilder) dialectName() string {
	driver := b.baseTx.Dialector.Name()
	if driver != dialectMySQL {
		return driver
	}
	if sqlDB, ok := b.baseTx.ConnPool.(*sql.DB); ok {
		if v, ok := dialectCache.Load(sqlDB); ok {
			return v.(string)
		}
		resolved := detectMySQLFlavor(b.baseTx)
		actual, _ := dialectCache.LoadOrStore(sqlDB, resolved)
		return actual.(string)
	}
	if v := b.dialectOnce.Load(); v != nil {
		return v.(string)
	}
	resolved := detectMySQLFlavor(b.baseTx)
	b.dialectOnce.Store(resolved)
	return resolved
}

// detectMySQLFlavor runs SELECT VERSION() and maps the banner to the effective
// dialect. If the query fails, it falls back to dialectMySQL5, whose
// utf8mb4_general_ci collation is portable across MySQL 5.x, 8.x, and MariaDB,
// rather than risking an unknown-collation error from the 8.0-only
// utf8mb4_0900_ai_ci.
func detectMySQLFlavor(db *gorm.DB) string {
	var version string
	if err := db.Raw("SELECT VERSION()").Scan(&version).Error; err == nil {
		return mysqlFlavor(version)
	}
	return dialectMySQL5
}

// mysqlFlavor maps a MySQL-family server version banner to the effective
// dialect. MariaDB must be distinguished because it lacks MySQL 8.0 collations.
// MySQL 5.x is distinguished because it lacks utf8mb4_0900_ai_ci.
func mysqlFlavor(version string) string {
	lower := strings.ToLower(version)
	if strings.Contains(lower, dialectMariaDB) {
		return dialectMariaDB
	}
	// Extract the leading numeric major version from the banner.
	if major := mysqlMajorVersion(version); major > 0 && major < 6 {
		return dialectMySQL5
	}
	return dialectMySQL
}

// mysqlMajorVersion parses the major version number from a MySQL VERSION()
// banner. Returns 0 if it cannot be determined.
func mysqlMajorVersion(version string) int {
	// The banner starts with digits, optionally followed by more dots/numbers.
	i := 0
	for i < len(version) && version[i] >= '0' && version[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0
	}
	n, err := strconv.Atoi(version[:i])
	if err != nil {
		return 0
	}
	return n
}

// ApplySorts applies sorting parameters to a GORM query.
// It takes a slice of Sort structs and adds ORDER BY clauses to the query.
// Example: []Sort{{Field: "name", Order: "asc"}} becomes "name asc"
func ApplySorts(query *gorm.DB, sorts []filter.Sort) *gorm.DB {
	for _, sort := range sorts {
		query = query.Order(fmt.Sprintf("%s %s", sort.Field, sort.Order))
	}
	return query
}

func (b *GORMBuilder) Apply(query *gorm.DB, filters []filter.CrudFilter) (*gorm.DB, error) {
	// Capture the tables/aliases present in the target query so that dotted fields
	// can be disambiguated between qualified columns and JSON paths. The table set
	// is per-query state, so it lives on a per-call copy of the builder: the shared
	// receiver must not be mutated (concurrent Apply on one builder stays race-free).
	work := *b
	work.knownTables = extractKnownTables(query)
	return work.apply(query, filters)
}

// apply runs the visitor pipeline for a single Apply call against the receiver's
// per-call knownTables snapshot.
func (b *GORMBuilder) apply(query *gorm.DB, filters []filter.CrudFilter) (*gorm.DB, error) {
	for _, f := range filters {
		// 1. Convert CrudFilter to Clause using the Visitor pattern
		clause, err := f.AcceptVisitor(b) // Calls b.VisitLogical or b.VisitConditional
		if err != nil {
			return query, fmt.Errorf("error accepting visitor: %w", err)
		}
		if clause == nil {
			// Visitor might return nil if the filter should be ignored (like 'q' with no config)
			continue
		}

		// 2. Build a GORM condition (*gorm.DB) from the Clause
		// This condition is isolated and built on a new session from b.baseTx.
		conditionDB, err := b.buildClauseCondition(clause)
		if err != nil {
			return query, fmt.Errorf("error building clause condition: %w", err)
		}

		// 3. Apply the isolated condition to the main query
		query = query.Where(conditionDB)
	}
	return query, nil
}

// buildClauseCondition translates a filter.Clause into a *gorm.DB object
// representing that specific condition. It uses b.baseTx to spawn new sessions
// and does NOT modify b.baseTx itself.
func (b *GORMBuilder) buildClauseCondition(clause filter.Clause) (*gorm.DB, error) {
	// Each clause's condition is built on a fresh session derived from the base.
	// This 'conditionBuilderDB' is a scratchpad for the current clause's logic.
	conditionBuilderDB := b.baseTx.Session(&gorm.Session{NewDB: true})

	switch c := clause.(type) {
	case *SQLClause:

		switch c.Query {
		case sqlIsNull, sqlIsNotNull:
			// These have no parameters
			return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query)), nil
		case sqlIn, sqlNotIn:
			// These expect the slice as a single argument to Where
			// c.Params is already the slice we want to pass.
			return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query), c.Params), nil
		case sqlBetween:
			// This expects two separate arguments. c.Params should be []any{min, max}
			if len(c.Params) != 2 {
				return nil, fmt.Errorf("BETWEEN operator requires exactly 2 parameters, got %d for field '%s'", len(c.Params), c.Field)
			}
			return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query), c.Params[0], c.Params[1]), nil // Pass slice elements separately
		default:
			// All other standard operators expect parameters spread variadically
			return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query), c.Params...), nil
		}
	case *GormConditionClause:
		// For pre-built GORM conditions, just return the condition as is
		return c.Condition, nil
	case *CompoundClause:
		switch c.Operator {
		case filter.LogicalAnd:
			// For AND, chain .Where() calls on the current conditionBuilderDB.
			// Each sub-condition is built recursively and then applied.
			currentAndGroup := conditionBuilderDB
			for _, subClause := range c.Filters {
				subConditionDB, err := b.buildClauseCondition(subClause)
				if err != nil {
					return nil, err
				}
				currentAndGroup = currentAndGroup.Where(subConditionDB)
			}
			return currentAndGroup, nil

		case filter.LogicalOr:
			if len(c.Filters) == 0 {
				// An empty OR group should match nothing.
				return conditionBuilderDB.Where("1 = 0"), nil
			}
			if len(c.Filters) == 1 {
				return b.buildClauseCondition(c.Filters[0])
			}
			// Build the first sub-condition for the OR group.
			firstSubConditionDB, err := b.buildClauseCondition(c.Filters[0])
			if err != nil {
				return nil, err
			}
			orGroupAccumulator := firstSubConditionDB // This is the start of our (A OR B OR ...) group.

			for _, subClause := range c.Filters[1:] {
				nextSubConditionDB, err := b.buildClauseCondition(subClause)
				if err != nil {
					return nil, err
				}
				orGroupAccumulator = orGroupAccumulator.Or(nextSubConditionDB)
			}
			// Apply the complete OR group to the current clause's conditionBuilderDB.
			return conditionBuilderDB.Where(orGroupAccumulator), nil

		case filter.LogicalNot:
			if len(c.Filters) != 1 {
				return nil, fmt.Errorf("NOT operator requires exactly one sub-filter, got %d", len(c.Filters))
			}
			subConditionDB, err := b.buildClauseCondition(c.Filters[0])
			if err != nil {
				return nil, err
			}
			if subConditionDB == nil {
				return nil, fmt.Errorf("sub-filter in NOT compound clause produced a nil condition")
			}
			return conditionBuilderDB.Not(subConditionDB), nil
		}
	}
	return nil, fmt.Errorf("unknown clause type: %T", clause)
}

func (b *GORMBuilder) VisitLogical(f *filter.LogicalFilter) (filter.Clause, error) {
	if f.Field() == "q" {
		// Global search 'q' field translates to an OR clause across searchable columns
		if b.searchConfig == nil || len(b.searchConfig.SearchableColumns) == 0 {
			// No search config or no searchable columns, 'q' filter has no effect
			return nil, nil
		}

		dialectName := b.dialectName()
		searchTerm := formatValue(filter.OpContains, f.Value())
		operatorMap := getOperatorMap(dialectName)
		sqlQueryTemplate := operatorMap[filter.OpContains]

		var clauses []filter.Clause
		for _, col := range b.searchConfig.SearchableColumns {
			clauses = append(clauses, NewSQLClause(sqlQueryTemplate, col, searchTerm))
		}

		return NewCompoundClause(filter.LogicalOr, clauses), nil
	}

	// Check if this is a JSON path field (contains dot notation and the leading
	// identifier does not name a table/alias present in the query).
	if b.isJSONPath(f.Field()) {
		return b.buildJSONClause(f)
	}

	// For all other logical filters, build a single SQL clause
	dialectName := b.dialectName()
	condition, params, err := buildCondition(f.Field(), f.Operator(), f.Value(), dialectName)
	if err != nil {
		return nil, fmt.Errorf("failed to build condition for field '%s' operator '%s': %w", f.Field(), f.Operator(), err)
	}
	return NewSQLClause(condition, f.Field(), params...), nil
}

func (b *GORMBuilder) VisitConditional(f *filter.ConditionalFilter) (filter.Clause, error) {
	var clauses []filter.Clause
	for _, sf := range f.Filters {
		clause, err := sf.AcceptVisitor(b)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause)
	}
	return NewCompoundClause(f.Operator, clauses), nil
}

// buildCondition determines the SQL query fragment and its parameters for a given field, operator, and value.
// It returns the query string, a slice of parameters ([]any), and an error.
func buildCondition(field string, op filter.Operator, value any, dialectName string) (string, []any, error) {
	operatorMap := getOperatorMap(dialectName)
	sqlQuery, ok := operatorMap[op]
	if !ok {
		return "", nil, fmt.Errorf("unsupported operator: %s", op)
	}

	// Handle NULL/NOT NULL operators which have no parameters
	if op == filter.OpNull || op == filter.OpNnull {
		return sqlQuery, nil, nil // No parameters
	}

	formattedVal := formatValue(op, value)

	// SQLite case-sensitive operators use GLOB, which has different wildcards.
	if dialectName == dialectSQLite {
		switch op {
		case filter.OpContainss, filter.OpNcontainss,
			filter.OpStartswiths, filter.OpNstartswiths,
			filter.OpEndswiths, filter.OpNendswiths:
			if s, ok := formattedVal.(string); ok {
				formattedVal = translateGlob(s)
			}
		}
	}

	// Special handling for BETWEEN as it expects exactly two parameters
	if op == filter.OpBetween {
		// formatValue for OpBetween should return []any{start, end} if valid
		if vals, ok := formattedVal.([]any); ok && len(vals) == 2 {
			return sqlQuery, vals, nil // Return the slice [start, end] directly as parameters
		}
		// If formatValue didn't return []any{start, end}, the input value was invalid
		return "", nil, fmt.Errorf("invalid value format for BETWEEN operator on field '%s': expected []any with 2 elements, got %T", field, value)
	}

	// If the operator requires an array (IN, NIN, BETWEEN, etc.), the formattedVal is already the []any slice.
	if op.RequiresArray() {
		// Ensure formattedVal is actually a slice before type assertion
		if vals, ok := formattedVal.([]any); ok {
			return sqlQuery, vals, nil // Return the slice directly
		}
		return "", nil, fmt.Errorf("operator '%s' requires an array value, but formatValue returned %T on field '%s'", op, formattedVal, field)
	}
	// Otherwise, it's a single value, wrap it.
	return sqlQuery, []any{formattedVal}, nil // Wrap the single value
}

func formatValue(op filter.Operator, value any) any {
	// Handle BETWEEN first, as it requires specific validation and structure
	if op == filter.OpBetween {
		if vals, ok := value.([]any); ok && len(vals) == 2 {
			return []any{vals[0], vals[1]}
		}
		return value
	}

	// Handle string pattern matching operators
	switch op {
	case filter.OpContains, filter.OpNcontains, filter.OpContainss, filter.OpNcontainss:
		return fmt.Sprintf("%%%v%%", value) // %value%
	case filter.OpStartswith, filter.OpStartswiths, filter.OpNstartswith, filter.OpNstartswiths:
		return fmt.Sprintf("%v%%", value) // value%
	case filter.OpEndswith, filter.OpEndswiths, filter.OpNendswith, filter.OpNendswiths:
		return fmt.Sprintf("%%%v", value) // %value
	}

	// For all other operators (Eq, Ne, Lt, Gt, Lte, Gte, In, Nin), return the value as is.
	return value
}

// isJSONPath reports whether a dotted field should be treated as a JSON path.
// A dotted field is a JSON path unless its leading identifier names a table or
// alias present in the query's FROM/JOINs, in which case it is a qualified column.
func (b *GORMBuilder) isJSONPath(field string) bool {
	if !strings.Contains(field, ".") {
		return false
	}
	leading := field[:strings.Index(field, ".")]
	return !b.isKnownTable(leading)
}

// isKnownTable reports whether name matches one of the tables/aliases known to
// the current query. Matching is case-insensitive per SQL identifier semantics.
func (b *GORMBuilder) isKnownTable(name string) bool {
	if len(b.knownTables) == 0 {
		return false
	}
	_, ok := b.knownTables[strings.ToLower(cleanIdentifier(name))]
	return ok
}

// extractKnownTables collects the table and alias names present in the query's
// FROM clause and JOINs. It returns nil when none can be determined, in which
// case dotted fields retain the legacy JSON-path behavior.
func extractKnownTables(query *gorm.DB) map[string]struct{} {
	if query == nil || query.Statement == nil {
		return nil
	}

	tables := make(map[string]struct{})
	add := func(id string) {
		if id = strings.ToLower(cleanIdentifier(id)); id != "" {
			tables[id] = struct{}{}
		}
	}

	stmt := query.Statement
	// For Model-based queries GORM defers Statement.Table resolution until SQL
	// build time, leaving it empty here. Resolve the model's table without
	// mutating the caller's statement by parsing into a throwaway statement; a
	// failed parse leaves the table empty and falls back to legacy behavior.
	if stmt.Table == "" && stmt.Model != nil {
		tmp := gorm.Statement{DB: stmt.DB, Model: stmt.Model}
		if err := tmp.Parse(stmt.Model); err == nil {
			add(tmp.Table)
		}
	}
	add(stmt.Table)
	for _, join := range stmt.Joins {
		table, alias := parseJoinName(join.Name)
		add(table)
		add(alias)
	}

	if len(tables) == 0 {
		return nil
	}
	return tables
}

// cleanIdentifier removes surrounding quotes and strips any schema qualifier,
// leaving the bare table/alias name.
func cleanIdentifier(id string) string {
	id = strings.Trim(strings.TrimSpace(id), "\"`'")
	if i := strings.LastIndex(id, "."); i >= 0 {
		id = id[i+1:]
	}
	return id
}

// joinKeyword is the set of words that may precede the joined table identifier
// in a JOIN clause string.
var joinKeywords = map[string]struct{}{
	"join": {}, "left": {}, "right": {}, "inner": {}, "outer": {},
	"full": {}, "cross": {}, "natural": {},
}

// parseJoinName extracts the table and optional alias from a raw JOIN clause
// string such as "LEFT JOIN directories ON ..." or "JOIN files AS f ON ...".
func parseJoinName(name string) (table, alias string) {
	fields := strings.Fields(name)
	i := 0
	// Skip leading JOIN-type keywords (LEFT, JOIN, OUTER, ...).
	for i < len(fields) {
		if _, ok := joinKeywords[strings.ToLower(fields[i])]; ok {
			i++
			continue
		}
		break
	}
	if i >= len(fields) {
		return "", ""
	}

	table = fields[i]
	i++
	if i >= len(fields) {
		return table, ""
	}
	// Joins may declare an alias as "<name> <alias>" or "<name> AS <alias>".
	if strings.EqualFold(fields[i], "AS") {
		i++
	}
	if i < len(fields) {
		switch strings.ToLower(fields[i]) {
		case "on", "using", "join":
			return table, ""
		default:
			return table, fields[i]
		}
	}
	return table, ""
}

// parseJSONPath splits a JSON path field into column name and path components
func parseJSONPath(field string) (jsonColumn, jsonPath string) {
	parts := strings.SplitN(field, ".", 2)
	return parts[0], parts[1]
}

// isValidJSONKey reports whether a key is a plain JSON path identifier that needs
// no quoting (letters, digits and underscore, not starting with a digit).
func isValidJSONKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		identifier := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(i > 0 && r >= '0' && r <= '9')
		if !identifier {
			return false
		}
	}
	return true
}

// quotePath returns a dot-joined JSON path with any non-identifier segment
// double-quoted. MySQL rejects paths such as "$.user-info" or "$.123" with an
// invalid-path error; quoting the segment produces a path both MySQL and SQLite
// accept.
func quotePath(jsonPath string) string {
	segments := strings.Split(jsonPath, ".")
	for i, seg := range segments {
		if !isValidJSONKey(seg) {
			segments[i] = "\"" + strings.ReplaceAll(seg, "\"", "\\\"") + "\""
		}
	}
	return strings.Join(segments, ".")
}

// jsonPathExpr returns a full "$.path" JSON path expression, quoting any segment
// that is not a plain identifier.
func jsonPathExpr(jsonPath string) string {
	return "$." + quotePath(jsonPath)
}

// buildJSONQuery creates a JSON query expression for the given operator and parameters
func (b *GORMBuilder) buildJSONQuery(operator, jsonColumn, jsonPath string, value any) (string, []any) {
	extractFunc := "json_extract"
	switch b.dialectName() {
	case dialectMySQL, dialectMySQL5, dialectMariaDB:
		extractFunc = "JSON_EXTRACT"
	}

	query := fmt.Sprintf("%s(%s, ?) %s ?", extractFunc, jsonColumn, operator)
	params := []any{jsonPathExpr(jsonPath), value}

	return query, params
}

// jsonExtractExpr returns a dialect-appropriate SQL expression that extracts
// the value at a JSON path (bound via "?") from a JSON column as unquoted
// text. MySQL's JSON_EXTRACT preserves the surrounding quotes of string
// values, which would otherwise break pattern matching (e.g. LIKE 'dar%'
// against `"dark"`), so it is wrapped in JSON_UNQUOTE to match SQLite's
// json_extract behavior. Unknown dialects fall back to the SQLite form since
// json_extract is the most widely implemented JSON function across embedded
// databases.
func (b *GORMBuilder) jsonExtractExpr(jsonColumn string) string {
	switch b.dialectName() {
	case dialectMySQL, dialectMySQL5, dialectMariaDB:
		return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(%s, ?))", jsonColumn)
	default:
		return fmt.Sprintf("json_extract(%s, ?)", jsonColumn)
	}
}

// buildJSONClause creates a GORM condition for JSON path fields
func (b *GORMBuilder) buildJSONClause(f *filter.LogicalFilter) (filter.Clause, error) {
	jsonColumn, jsonPath := parseJSONPath(f.Field())

	// Create a new session for building the JSON query
	conditionBuilderDB := b.baseTx.Session(&gorm.Session{NewDB: true})

	// Handle different operators using datatypes.JSONQuery methods
	switch f.Operator() {
	case filter.OpEq:
		condition := conditionBuilderDB.Where(datatypes.JSONQuery(jsonColumn).Equals(f.Value(), quotePath(jsonPath)))
		return NewGormConditionClause(condition, f.Field()), nil
	case filter.OpNe:
		condition := conditionBuilderDB.Where(b.buildJSONQuery("<> ?", jsonColumn, jsonPath, f.Value()))
		return NewGormConditionClause(condition, f.Field()), nil
	case filter.OpGt:
		query, params := b.buildJSONQuery(">", jsonColumn, jsonPath, f.Value())
		return NewSQLClause(query, "", params...), nil
	case filter.OpGte:
		query, params := b.buildJSONQuery(">=", jsonColumn, jsonPath, f.Value())
		return NewSQLClause(query, "", params...), nil
	case filter.OpLt:
		query, params := b.buildJSONQuery("<", jsonColumn, jsonPath, f.Value())
		return NewSQLClause(query, "", params...), nil
	case filter.OpLte:
		query, params := b.buildJSONQuery("<=", jsonColumn, jsonPath, f.Value())
		return NewSQLClause(query, "", params...), nil
	case filter.OpNull:
		condition := conditionBuilderDB.Where("? IS NULL", datatypes.JSONQuery(jsonColumn).Extract(quotePath(jsonPath)))
		return NewGormConditionClause(condition, f.Field()), nil
	case filter.OpNnull:
		condition := conditionBuilderDB.Where("? IS NOT NULL", datatypes.JSONQuery(jsonColumn).Extract(quotePath(jsonPath)))
		return NewGormConditionClause(condition, f.Field()), nil
	default:
		// For pattern matching operators, build explicit SQL using the dialect's
		// JSON extract expression and the appropriate LIKE/GLOB operator. The
		// forms come from the shared dialectPatterns structure, matching the
		// non-JSON buildCondition path.
		extract := b.jsonExtractExpr(jsonColumn)
		pat, ok := dialectPatterns[b.dialectName()]
		if !ok {
			pat = dialectPatterns[""]
		}
		likeOp, notLikeOp := pat.ciLike, pat.ciNotLike

		// fmtPattern formats a LIKE pattern with LIKE wildcards (% _).
		fmtPattern := func(v any, format string) string {
			return fmt.Sprintf(format, v)
		}

		// fmtGlobPattern formats a GLOB pattern, translating LIKE wildcards
		// (% _) to GLOB wildcards (* ?) when the dialect uses GLOB.
		fmtGlobPattern := func(v any, format string) string {
			s := fmt.Sprintf(format, v)
			if pat.csTranslate {
				return translateGlob(s)
			}
			return s
		}

		var pattern string
		var query string
		switch f.Operator() {
		case filter.OpContains:
			query, pattern = extract+" "+likeOp, fmtPattern(f.Value(), "%%%v%%")
		case filter.OpContainss:
			query, pattern = extract+" "+pat.csLike, fmtGlobPattern(f.Value(), "%%%v%%")
		case filter.OpNcontains:
			query, pattern = extract+" "+notLikeOp, fmtPattern(f.Value(), "%%%v%%")
		case filter.OpNcontainss:
			query, pattern = extract+" "+pat.csNotLike, fmtGlobPattern(f.Value(), "%%%v%%")
		case filter.OpStartswith:
			query, pattern = extract+" "+likeOp, fmtPattern(f.Value(), "%v%%")
		case filter.OpStartswiths:
			query, pattern = extract+" "+pat.csLike, fmtGlobPattern(f.Value(), "%v%%")
		case filter.OpNstartswith:
			query, pattern = extract+" "+notLikeOp, fmtPattern(f.Value(), "%v%%")
		case filter.OpNstartswiths:
			query, pattern = extract+" "+pat.csNotLike, fmtGlobPattern(f.Value(), "%v%%")
		case filter.OpEndswith:
			query, pattern = extract+" "+likeOp, fmtPattern(f.Value(), "%%%v")
		case filter.OpEndswiths:
			query, pattern = extract+" "+pat.csLike, fmtGlobPattern(f.Value(), "%%%v")
		case filter.OpNendswith:
			query, pattern = extract+" "+notLikeOp, fmtPattern(f.Value(), "%%%v")
		case filter.OpNendswiths:
			query, pattern = extract+" "+pat.csNotLike, fmtGlobPattern(f.Value(), "%%%v")
		default:
			return nil, fmt.Errorf("unsupported operator for JSON field: %s", f.Operator())
		}

		return NewSQLClause(query, "", jsonPathExpr(jsonPath), pattern), nil
	}
}
