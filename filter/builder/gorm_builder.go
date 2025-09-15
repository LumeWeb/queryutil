package builder

import (
	"fmt"
	"strings"

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
)

var operatorMap = map[filter.Operator]string{
	filter.OpEq:           "= ?",
	filter.OpNe:           "<> ?",
	filter.OpLt:           "< ?",
	filter.OpGt:           "> ?",
	filter.OpLte:          "<= ?",
	filter.OpGte:          ">= ?",
	filter.OpContains:     "LIKE ? COLLATE NOCASE",
	filter.OpContainss:    "LIKE ? COLLATE BINARY",
	filter.OpNcontains:    "NOT LIKE ? COLLATE NOCASE",
	filter.OpNcontainss:   "NOT LIKE ? COLLATE BINARY",
	filter.OpStartswith:   "LIKE ? COLLATE NOCASE",
	filter.OpStartswiths:  "LIKE ? COLLATE BINARY",
	filter.OpNstartswith:  "NOT LIKE ? COLLATE NOCASE",
	filter.OpNstartswiths: "NOT LIKE ? COLLATE BINARY",
	filter.OpEndswith:     "LIKE ? COLLATE NOCASE",
	filter.OpEndswiths:    "LIKE ? COLLATE BINARY",
	filter.OpNendswith:    "NOT LIKE ? COLLATE NOCASE",
	filter.OpNendswiths:   "NOT LIKE ? COLLATE BINARY",
	filter.OpNull:         sqlIsNull,
	filter.OpNnull:        sqlIsNotNull,
	filter.OpIn:           sqlIn,
	filter.OpNin:          sqlNotIn,
	filter.OpNbetween:     sqlNotBetween,
	filter.OpBetween:      sqlBetween,
}

type GORMBuilder struct {
	baseTx       *gorm.DB // The original DB connection/transaction
	searchConfig *filter.GlobalSearchConfig
}

func NewGORMBuilder(baseTx *gorm.DB, searchConfig *filter.GlobalSearchConfig) *GORMBuilder {
	return &GORMBuilder{baseTx: baseTx, searchConfig: searchConfig}
}

// ApplySorts applies sorting parameters to a GORM query.
// It takes a slice of Sort structs and adds ORDER BY clauses to the query.
// Example: []Sort{{Field: "name", Order: "asc"}} becomes "name asc"
func (b *GORMBuilder) ApplySorts(query *gorm.DB, sorts []filter.Sort) *gorm.DB {
	for _, sort := range sorts {
		query = query.Order(fmt.Sprintf("%s %s", sort.Field, sort.Order))
	}
	return query
}

func (b *GORMBuilder) Apply(query *gorm.DB, filters []filter.CrudFilter) (*gorm.DB, error) {
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

		searchTerm := formatValue(filter.OpContains, f.Value())
		sqlQueryTemplate := operatorMap[filter.OpContains]

		var clauses []filter.Clause
		for _, col := range b.searchConfig.SearchableColumns {
			clauses = append(clauses, NewSQLClause(sqlQueryTemplate, col, searchTerm))
		}

		return NewCompoundClause(filter.LogicalOr, clauses), nil
	}

	// Check if this is a JSON path field (contains dot notation)
	if isJSONPath(f.Field()) {
		return b.buildJSONClause(f)
	}

	// For all other logical filters, build a single SQL clause
	condition, params, err := buildCondition(f.Field(), f.Operator(), f.Value())
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
func buildCondition(field string, op filter.Operator, value any) (string, []any, error) {
	sqlQuery, ok := operatorMap[op]
	if !ok {
		return "", nil, fmt.Errorf("unsupported operator: %s", op)
	}

	// Handle NULL/NOT NULL operators which have no parameters
	if op == filter.OpNull || op == filter.OpNnull {
		return sqlQuery, nil, nil // No parameters
	}

	formattedVal := formatValue(op, value)

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

// isJSONPath checks if a field name represents a JSON path (contains dot notation)
func isJSONPath(field string) bool {
	return strings.Contains(field, ".")
}

// parseJSONPath splits a JSON path field into column name and path components
func parseJSONPath(field string) (jsonColumn, jsonPath string) {
	parts := strings.SplitN(field, ".", 2)
	return parts[0], parts[1]
}

// buildJSONQuery creates a JSON query expression for the given operator and parameters
func (b *GORMBuilder) buildJSONQuery(operator, jsonColumn, jsonPath string, value any) (string, []any) {
	// For SQLite, use json_extract()
	// For MySQL, use JSON_EXTRACT()
	extractFunc := "JSON_EXTRACT"
	if b.baseTx.Dialector.Name() == "sqlite" {
		extractFunc = "json_extract"
	}

	query := fmt.Sprintf("%s(%s, ?) %s ?", extractFunc, jsonColumn, operator)
	params := []any{"$." + jsonPath, value}

	return query, params
}

// buildJSONClause creates a GORM condition for JSON path fields
func (b *GORMBuilder) buildJSONClause(f *filter.LogicalFilter) (filter.Clause, error) {
	jsonColumn, jsonPath := parseJSONPath(f.Field())

	// Create a new session for building the JSON query
	conditionBuilderDB := b.baseTx.Session(&gorm.Session{NewDB: true})

	// Handle different operators using datatypes.JSONQuery methods
	switch f.Operator() {
	case filter.OpEq:
		condition := conditionBuilderDB.Where(datatypes.JSONQuery(jsonColumn).Equals(f.Value(), jsonPath))
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
		condition := conditionBuilderDB.Where("? IS NULL", datatypes.JSONQuery(jsonColumn).Extract(jsonPath))
		return NewGormConditionClause(condition, f.Field()), nil
	case filter.OpNnull:
		condition := conditionBuilderDB.Where("? IS NOT NULL", datatypes.JSONQuery(jsonColumn).Extract(jsonPath))
		return NewGormConditionClause(condition, f.Field()), nil
	default:
		// For pattern matching operators, we need to handle them specially
		switch f.Operator() {
		case filter.OpContains, filter.OpContainss:
			condition := conditionBuilderDB.Where(datatypes.JSONQuery(jsonColumn).Likes(fmt.Sprintf("%%%v%%", f.Value()), jsonPath))
			return NewGormConditionClause(condition, f.Field()), nil
		case filter.OpNcontains, filter.OpNcontainss:
			// For NOT LIKE operations, we need to negate the LIKE condition
			jsonQuery := datatypes.JSONQuery(jsonColumn).Likes(fmt.Sprintf("%%%v%%", f.Value()), jsonPath)
			condition := conditionBuilderDB.Where("NOT (?)", jsonQuery)
			return NewGormConditionClause(condition, f.Field()), nil
		case filter.OpStartswith, filter.OpStartswiths, filter.OpNstartswith, filter.OpNstartswiths:
			// Handle startswith by using LIKE with appropriate pattern
			condition := conditionBuilderDB.Where(datatypes.JSONQuery(jsonColumn).Likes(fmt.Sprintf("%v%%", f.Value()), jsonPath))
			return NewGormConditionClause(condition, f.Field()), nil
		case filter.OpEndswith, filter.OpEndswiths, filter.OpNendswith, filter.OpNendswiths:
			// Handle endswith by using LIKE with appropriate pattern
			condition := conditionBuilderDB.Where(datatypes.JSONQuery(jsonColumn).Likes(fmt.Sprintf("%%%v", f.Value()), jsonPath))
			return NewGormConditionClause(condition, f.Field()), nil
		default:
			// If we get here, the operator is not supported for JSON fields
			return nil, fmt.Errorf("unsupported operator for JSON field: %s", f.Operator())
		}
	}
}
