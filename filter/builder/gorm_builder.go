package builder

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
	"gorm.io/gorm"
	"log"
)

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
		log.Printf("Applying SQL clause - Field: %s, Query: %s, Params: %v", c.Field, c.Query, c.Params)
		if c.Query == "IS NULL" || c.Query == "IS NOT NULL" {
			return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query)), nil
		}
		if c.Field == "q" { // Handle global search
			if b.searchConfig == nil || len(b.searchConfig.SearchableColumns) == 0 {
				// No search config or no searchable columns, 'q' filter has no effect.
				// Return the empty conditionBuilderDB, which GORM's .Where() will ignore.
				return conditionBuilderDB, nil
			}

			// Global search requires building an OR group across searchable columns.
			// c.Params[0] should be the formatted search term (e.g., "%searchterm%")
			// as prepared by formatValue via buildCondition.
			searchTerm := c.Params[0]
			var globalSearchGroup *gorm.DB

			for i, col := range b.searchConfig.SearchableColumns {
				// Each part of the OR group is also built on a fresh session.
				columnCondition := b.baseTx.Session(&gorm.Session{NewDB: true}).Where(fmt.Sprintf("%s LIKE ?", col), searchTerm)
				if i == 0 {
					globalSearchGroup = columnCondition
				} else {
					globalSearchGroup = globalSearchGroup.Or(columnCondition)
				}
			}
			// Apply the entire OR group to the current clause's conditionBuilderDB.
			return conditionBuilderDB.Where(globalSearchGroup), nil
		}

		// Handle NULL/NOT NULL operators directly
		if c.Query == "IS NULL" || c.Query == "IS NOT NULL" {
			return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query)), nil
		}
		// Standard SQL condition
		return conditionBuilderDB.Where(fmt.Sprintf("%s %s", c.Field, c.Query), c.Params...), nil
	case *CompoundClause:
		log.Printf("Applying compound clause - Operator: %s, Sub-filters: %d", c.Operator, len(c.Filters))
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
				// An empty OR group results in a condition that doesn't restrict,
				// or if you prefer, a condition that's always false to return no results.
				// For simplicity, let's return an empty condition that GORM's Where will ignore.
				// If an empty OR should mean "match nothing", use: return conditionBuilderDB.Where("1 = 0"), nil
				return conditionBuilderDB, nil
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
			return conditionBuilderDB.Not(subConditionDB), nil
		}
	}
	return nil, fmt.Errorf("unknown clause type: %T", clause)
}

func (b *GORMBuilder) VisitLogical(f *filter.LogicalFilter) (filter.Clause, error) {
	// Special handling for global search
	if f.Field == "q" {
		condition, value := buildCondition(f.Field, filter.OpContains, f.Value)
		return NewSQLClause(condition, value, f.Field), nil
	}
	condition, value := buildCondition(f.Field, f.Operator, f.Value)
	return NewSQLClause(condition, value, f.Field), nil
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
	return &CompoundClause{
		Operator: f.Operator,
		Filters:  clauses,
	}, nil
}

func buildCondition(field string, op filter.Operator, value interface{}) (string, interface{}) {
	formattedVal := formatValue(op, value)

	opMap := map[filter.Operator]string{
		filter.OpEq:           "= ?",
		filter.OpNe:           "<> ?",
		filter.OpLt:           "< ?",
		filter.OpGt:           "> ?",
		filter.OpLte:          "<= ?",
		filter.OpGte:          ">= ?",
		filter.OpContains:     "LIKE ?",
		filter.OpContainss:    "LIKE ?",
		filter.OpNcontains:    "NOT LIKE ?",
		filter.OpNcontainss:   "NOT LIKE ?",
		filter.OpStartswith:   "LIKE ?",
		filter.OpStartswiths:  "LIKE ?",
		filter.OpNstartswith:  "NOT LIKE ?",
		filter.OpNstartswiths: "NOT LIKE ?",
		filter.OpEndswith:     "LIKE ?",
		filter.OpEndswiths:    "LIKE ?",
		filter.OpNendswith:    "NOT LIKE ?",
		filter.OpNendswiths:   "NOT LIKE ?",
		filter.OpNull:         "IS NULL",
		filter.OpNnull:        "IS NOT NULL",
		filter.OpIn:           "IN (?)",
		filter.OpNin:          "NOT IN (?)",
		filter.OpBetween:      "BETWEEN ? AND ?",
	}

	// Handle global search
	if field == "q" {
		return "", formattedVal // Will be handled in ApplyClauses
	}

	return opMap[op], formattedVal
}

func formatValue(op filter.Operator, value interface{}) interface{} {
	switch op {
	case filter.OpContains, filter.OpNcontains:
		return fmt.Sprintf("%%%v%%", value)
	case filter.OpContainss, filter.OpNcontainss:
		return value // Exact case match, no wrapping
	case filter.OpStartswith, filter.OpStartswiths, filter.OpNstartswith, filter.OpNstartswiths:
		return fmt.Sprintf("%v%%", value)
	case filter.OpEndswith, filter.OpEndswiths, filter.OpNendswith, filter.OpNendswiths:
		return fmt.Sprintf("%%%v", value)
	case filter.OpBetween:
		if vals, ok := value.([]interface{}); ok && len(vals) == 2 {
			return []interface{}{vals[0], vals[1]}
		}
	}
	return value
}
