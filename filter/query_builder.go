package filter

import (
	"fmt"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
	"time"
)

// Core predicate functions

// Equal creates a 'field equals value' filter
// Equal creates an equality filter for a field.
// Uses the OpEq operator to match values exactly.
// Example: Equal("age", 25) -> "age = 25"
func Equal(field string, value any) CrudFilter {
	return NewLogicalFilter(field, OpEq, value)
}

// NotEqual creates a 'field not equal to value' filter
// NotEqual creates an inequality filter for a field.
// Uses the OpNe operator to match values not equal to the provided value.
// Example: NotEqual("status", "active") -> "status != 'active'"
func NotEqual(field string, value any) CrudFilter {
	return NewLogicalFilter(field, OpNe, value)
}

// GreaterThan creates a 'field > value' filter
// GreaterThan creates a filter for values greater than the provided value.
// Uses the OpGt operator. Works with numeric and temporal values.
// Example: GreaterThan("age", 21) -> "age > 21"
func GreaterThan(field string, value any) CrudFilter {
	return NewLogicalFilter(field, OpGt, value)
}

// GreaterOrEqual creates a 'field >= value' filter
// GreaterOrEqual creates a filter for values greater than or equal to the provided value.
// Uses the OpGte operator. Works with numeric and temporal values.
// Example: GreaterOrEqual("score", 90) -> "score >= 90"
func GreaterOrEqual(field string, value any) CrudFilter {
	return NewLogicalFilter(field, OpGte, value)
}

// LessThan creates a 'field < value' filter
// LessThan creates a filter for values less than the provided value.
// Uses the OpLt operator. Works with numeric and temporal values.
// Example: LessThan("created_at", time.Now()) -> "created_at < NOW()"
func LessThan(field string, value any) CrudFilter {
	return NewLogicalFilter(field, OpLt, value)
}

// LessOrEqual creates a 'field <= value' filter
// LessOrEqual creates a filter for values less than or equal to the provided value.
// Uses the OpLte operator. Works with numeric and temporal values.
// Example: LessOrEqual("inventory", 0) -> "inventory <= 0"
func LessOrEqual(field string, value any) CrudFilter {
	return NewLogicalFilter(field, OpLte, value)
}

// In creates a 'field IN (values...)' filter
func In(field string, values ...any) CrudFilter {
	validateNotEmpty(values, "In")
	return NewLogicalFilter(field, OpIn, values)
}

// Contains creates a 'field CONTAINS value' filter
// Search creates a global search filter for the 'q' parameter.
// Uses the OpContains operator on the special 'q' field.
// Example: Search("test") -> "q contains 'test'"
func Search(query string) CrudFilter {
	return NewLogicalFilter("q", OpContains, query)
}

// Contains creates a case-insensitive substring filter.
// Uses the OpContains operator. Value must be non-empty.
// Example: Contains("name", "smith") -> "name ILIKE '%smith%'"
func Contains(field string, value string) CrudFilter {
	return NewLogicalFilter(field, OpContains, value)
}

// Between creates a 'field BETWEEN min AND max' filter
func Between(field string, min, max any) CrudFilter {
	validateBetweenValues([]any{min, max})
	return NewLogicalFilter(field, OpBetween, []any{min, max})
}

func FieldNotBetween(field string, min, max any) CrudFilter {
	validateBetweenValues([]any{min, max})
	return NewLogicalFilter(field, OpNbetween, []any{min, max})
}

func FieldNotIn(field string, values ...any) CrudFilter {
	validateNotEmpty(values, "NotIn")
	return NewLogicalFilter(field, OpNin, values)
}

func FieldIsNull(field string) CrudFilter {
	return NewLogicalFilter(field, OpNull, nil)
}

func FieldIsNotNull(field string) CrudFilter {
	return NewLogicalFilter(field, OpNnull, nil)
}

// Filters creates a filter group from multiple conditions
func Filters(filters ...CrudFilter) []CrudFilter {
	return filters
}

// AndF combines filters with AND logic (explicit version)
func AndF(filters ...CrudFilter) CrudFilter {
	return NewConditionalFilter(LogicalAnd, filters)
}

// OrF combines filters with OR logic (explicit version)
func OrF(filters ...CrudFilter) CrudFilter {
	return NewConditionalFilter(LogicalOr, filters)
}

// Logical combinators

// And combines filters with AND logic
func And(filters ...CrudFilter) CrudFilter {
	validateNotEmpty(filters, "And")
	return NewConditionalFilter(LogicalAnd, filters)
}

// Or combines filters with OR logic
func Or(filters ...CrudFilter) CrudFilter {
	validateNotEmpty(filters, "Or")
	return NewConditionalFilter(LogicalOr, filters)
}

// Not negates a filter with NOT logic
func Not(filter CrudFilter) CrudFilter {
	return NewConditionalFilter(LogicalNot, []CrudFilter{filter})
}

// MergeFilters combines multiple filter sets with different strategies
func MergeFilters(base []CrudFilter, overrides []CrudFilter, strategy MergeStrategy) []CrudFilter {
	switch strategy {
	case MergeStrategyOverride:
		return mergeOverride(base, overrides)
	case MergeStrategyDeep:
		return mergeDeep(base, overrides)
	case MergeStrategyAppend:
		return mergeAppend(base, overrides)
	default:
		panic(fmt.Sprintf("unknown merge strategy: %v", strategy))
	}
}

// MergeStrategy defines how multiple filter sets should be combined
type MergeStrategy int

const (
	MergeStrategyOverride MergeStrategy = iota // Replace existing field operators
	MergeStrategyDeep                          // Deep merge nested conditional filters
	MergeStrategyAppend                        // Append all filters without deduplication
)

// MergeFilters combines multiple filter sets using the specified strategy.
// This is useful for merging base filters with request-specific filters.
// Example:
// baseFilters := []CrudFilter{Equal("status", "active")}
// userFilters := []CrudFilter{GreaterThan("age", 18)}
// merged := MergeFilters(baseFilters, userFilters, MergeStrategyAppend)

func mergeOverride(base, overrides []CrudFilter) []CrudFilter {
	overrideFields := make(map[string]bool)
	for _, f := range overrides {
		if field := f.GetField(); field != "" {
			overrideFields[field] = true
		}
	}

	merged := make([]CrudFilter, 0, len(overrides)+len(base))
	merged = append(merged, overrides...)

	for _, f := range base {
		field := f.GetField()
		if field == "" {
			merged = append(merged, f)
		} else if !overrideFields[field] {
			merged = append(merged, f)
		}
	}

	return merged
}

func mergeDeep(base, overrides []CrudFilter) []CrudFilter {
	// Implement deep merge logic for conditional filters
	// This would recursively merge nested filter structures
	// (omitted for brevity but would handle nested AND/OR groups)
	return append(base, overrides...)
}

func mergeAppend(base, overrides []CrudFilter) []CrudFilter {
	return append(base, overrides...)
}

// Field-specific helpers

// FieldIn creates an 'IN' filter for a specific field
func FieldIn(field string, values ...any) CrudFilter {
	return In(field, values...)
}

// FieldEqual creates an equality filter for a specific field
func FieldEqual(field string, value any) CrudFilter {
	return Equal(field, value)
}

// FieldNotEqual creates an inequality filter for a specific field
func FieldNotEqual(field string, value any) CrudFilter {
	return NotEqual(field, value)
}

// FieldGt creates a 'field > value' filter
func FieldGt(field string, value any) CrudFilter {
	return GreaterThan(field, value)
}

// FieldLt creates a 'field < value' filter
func FieldLt(field string, value any) CrudFilter {
	return LessThan(field, value)
}

// FieldGte creates a 'field >= value' filter
func FieldGte(field string, value any) CrudFilter {
	return GreaterOrEqual(field, value)
}

// FieldLte creates a 'field <= value' filter
func FieldLte(field string, value any) CrudFilter {
	return LessOrEqual(field, value)
}

// FieldContains creates a 'field CONTAINS value' filter
func FieldContains(field string, value string) CrudFilter {
	if value == "" {
		panic("Contains requires a non-empty value")
	}
	return Contains(field, value)
}

// FieldBetween creates a 'field BETWEEN min AND max' filter
func FieldBetween(field string, min, max any) CrudFilter {
	return Between(field, min, max)
}

// Type-safe field helpers

// stringFieldHelper provides type-safe filtering methods for string fields
type stringFieldHelper struct {
	field string
}

// Eq creates an equality filter for the string field.
// Uses OpEq operator. Performs exact match.
// Example: StringField("name").Eq("john")
func (s stringFieldHelper) Eq(v string) CrudFilter {
	return FieldEqual(s.field, v)
}

// Contains creates a case-insensitive substring filter.
// Uses OpContains operator. Value must be non-empty.
// Example: StringField("bio").Contains("golang")
func (s stringFieldHelper) Contains(v string) CrudFilter {
	return FieldContains(s.field, v)
}

// StartsWith creates a case-insensitive starts-with filter.
// Uses OpContains operator with wildcard suffix.
// Example: StringField("title").StartsWith("chapter")
func (s stringFieldHelper) StartsWith(v string) CrudFilter {
	return FieldContains(s.field, v+"%")
}

// EndsWith creates a case-insensitive ends-with filter.
// Uses OpContains operator with wildcard prefix.
// Example: StringField("filename").EndsWith(".pdf")
func (s stringFieldHelper) EndsWith(v string) CrudFilter {
	return FieldContains(s.field, "%"+v)
}

// In creates a filter matching any of the provided values.
// Uses OpIn operator. Requires at least one value.
// Example: StringField("status").In("active", "pending")
func (s stringFieldHelper) In(values ...string) CrudFilter {
	return FieldIn(s.field, lo.ToAnySlice(values)...)
}

// NotIn creates a filter excluding the provided values.
// Uses OpNin operator. Requires at least one value.
// Example: StringField("role").NotIn("admin", "superuser")
func (s stringFieldHelper) NotIn(values ...string) CrudFilter {
	return FieldNotIn(s.field, lo.ToAnySlice(values)...)
}

func (s stringFieldHelper) IsNull() CrudFilter {
	return FieldIsNull(s.field)
}

func (s stringFieldHelper) IsNotNull() CrudFilter {
	return FieldIsNotNull(s.field)
}

// StringField creates a type-safe query builder helper for string fields.
// Provides chainable methods for building string-specific filters.
// Example:
// queryutil.StringField("name").Eq("john").Contains("doe")
func StringField(field string) stringFieldHelper {
	return stringFieldHelper{field: field}
}

type NumberFieldHelper[T constraints.Integer | constraints.Float] struct {
	field string
}

func (n NumberFieldHelper[T]) Eq(v T) CrudFilter {
	return FieldEqual(n.field, v)
}

func (n NumberFieldHelper[T]) Gt(v T) CrudFilter {
	return FieldGt(n.field, v)
}

func (n NumberFieldHelper[T]) Lt(v T) CrudFilter {
	return FieldLt(n.field, v)
}

func (n NumberFieldHelper[T]) Gte(v T) CrudFilter {
	return FieldGte(n.field, v)
}

func (n NumberFieldHelper[T]) Lte(v T) CrudFilter {
	return FieldLte(n.field, v)
}

func (n NumberFieldHelper[T]) Between(min, max T) CrudFilter {
	return FieldBetween(n.field, min, max)
}

func (n NumberFieldHelper[T]) NotBetween(min, max T) CrudFilter {
	return FieldNotBetween(n.field, min, max)
}

func (n NumberFieldHelper[T]) In(values ...T) CrudFilter {
	return FieldIn(n.field, lo.ToAnySlice(values)...)
}

func (n NumberFieldHelper[T]) NotIn(values ...T) CrudFilter {
	return FieldNotIn(n.field, lo.ToAnySlice(values)...)
}

func (n NumberFieldHelper[T]) IsNull() CrudFilter {
	return FieldIsNull(n.field)
}

func (n NumberFieldHelper[T]) IsNotNull() CrudFilter {
	return FieldIsNotNull(n.field)
}

// NumberField creates a type-safe query builder helper for numeric fields.
// Supports generic numeric types (integers and floats).
// Example:
// queryutil.NumberField[int]("age").Gte(18)
// queryutil.NumberField[float64]("price").Between(10.5, 20.0)
func NumberField[T constraints.Integer | constraints.Float](field string) NumberFieldHelper[T] {
	return NumberFieldHelper[T]{field: field}
}

// boolFieldHelper provides type-safe filtering methods for boolean fields
type boolFieldHelper struct {
	field string
}

func (b boolFieldHelper) Eq(v bool) CrudFilter {
	return FieldEqual(b.field, v)
}

func (b boolFieldHelper) Neq(v bool) CrudFilter {
	return FieldNotEqual(b.field, v)
}

func (b boolFieldHelper) IsNull() CrudFilter {
	return FieldIsNull(b.field)
}

func (b boolFieldHelper) IsNotNull() CrudFilter {
	return FieldIsNotNull(b.field)
}

// BoolField creates a type-safe query builder helper for boolean fields.
// Example:
// queryutil.BoolField("verified").Eq(true)
// queryutil.BoolField("deleted").IsNotNull()
func BoolField(field string) boolFieldHelper {
	return boolFieldHelper{field: field}
}

// timeFieldHelper provides type-safe filtering methods for time fields
type timeFieldHelper struct {
	field string
}

func (t timeFieldHelper) Eq(v time.Time) CrudFilter {
	return FieldEqual(t.field, v)
}

func (t timeFieldHelper) Before(v time.Time) CrudFilter {
	return FieldLt(t.field, v)
}

func (t timeFieldHelper) After(v time.Time) CrudFilter {
	return FieldGt(t.field, v)
}

func (t timeFieldHelper) Between(start, end time.Time) CrudFilter {
	return FieldBetween(t.field, start, end)
}

func (t timeFieldHelper) NotBetween(start, end time.Time) CrudFilter {
	return &LogicalFilter{
		field:    t.field,
		operator: OpNbetween,
		value:    []any{start, end},
	}
}

func (t timeFieldHelper) IsNull() CrudFilter {
	return &LogicalFilter{
		field:    t.field,
		operator: OpNull,
		value:    nil,
	}
}

func (t timeFieldHelper) IsNotNull() CrudFilter {
	return &LogicalFilter{
		field:    t.field,
		operator: OpNnull,
		value:    nil,
	}
}

// TimeField creates a type-safe query builder helper for time fields.
// Supports time.Time values and temporal comparisons.
// Example:
// queryutil.TimeField("created_at").After(time.Now().Add(-24*time.Hour))
func TimeField(field string) timeFieldHelper {
	return timeFieldHelper{field: field}
}

// Validation checks
var DefaultValidators = map[Operator][]func(field string, value any){
	OpContains:     {validateStringType, validateNonNumericField},
	OpContainss:    {validateStringType, validateNonNumericField},
	OpNcontains:    {validateStringType, validateNonNumericField},
	OpNcontainss:   {validateStringType, validateNonNumericField},
	OpStartswith:   {validateStringType, validateNonNumericField},
	OpStartswiths:  {validateStringType, validateNonNumericField},
	OpNstartswith:  {validateStringType, validateNonNumericField},
	OpNstartswiths: {validateStringType, validateNonNumericField},
	OpEndswith:     {validateStringType, validateNonNumericField},
	OpEndswiths:    {validateStringType, validateNonNumericField},
	OpNendswith:    {validateStringType, validateNonNumericField},
	OpNendswiths:   {validateStringType, validateNonNumericField},
	OpEq:           {},
	OpGt:           {validateNumericOrTimeType},
	OpLt:           {validateNumericOrTimeType},
	OpGte:          {validateNumericOrTimeType},
	OpLte:          {validateNumericOrTimeType},
	OpBetween:      {validateNumericPair},
	OpNbetween:     {validateNumericPair},
	OpIn:           {validateArrayType},
	OpNin:          {validateArrayType},
	OpNull:         {validateNullValue},
	OpNnull:        {validateNullValue},
}

func validateNullValue(field string, value any) {
	if value != nil {
		panic(fmt.Sprintf("%s: null operator requires nil value, got %T (%v)", field, value, value))
	}
}

func validateNonNumericField(field string, value any) {
	if _, ok := value.(string); !ok {
		panic(fmt.Sprintf("%s: string value required for string operator, got %T (%v)", field, value, value))
	}
}

func validateNumericOrTimeTypeNoPanic(value any) bool {
	if isNumeric(value) {
		return true
	}
	if _, ok := value.(time.Time); ok {
		return true
	}
	return false
}

func validateValue(operator Operator, field string, value any) {
	if validators, ok := DefaultValidators[operator]; ok {
		for _, validate := range validators {
			validate(field, value)
		}
	}
}

func validateStringType(field string, value any) {
	if _, ok := value.(string); !ok {
		panic(fmt.Sprintf("%s: string value required for operator, got %T", field, value))
	}
}

func validateNumericOrTimeType(field string, value any) {
	if isNumeric(value) {
		return
	}
	if _, ok := value.(time.Time); ok {
		return
	}
	panic(fmt.Sprintf("%s: value must be numeric or time.Time, got %T", field, value))
}

func validateNumericPair(field string, value any) {
	vals, ok := value.([]any)
	if !ok || len(vals) != 2 {
		panic(fmt.Sprintf("%s: between operator requires exactly 2 numeric values", field))
	}
	for _, v := range vals {
		validateNumericOrTimeType(field, v)
	}
}

func validateArrayType(field string, value any) {
	if _, ok := value.([]any); !ok {
		panic(fmt.Sprintf("%s: array value required for operator", field))
	}
}

func isNumeric(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}

func validateBetweenValues(values []any) {
	if len(values) != 2 {
		panic("Between requires exactly 2 values")
	}

	min := values[0]
	max := values[1]

	// Handle time.Time values specifically
	if minTime, ok := min.(time.Time); ok {
		if maxTime, ok := max.(time.Time); ok {
			if minTime.After(maxTime) {
				panic("Between min must be less than or equal to max")
			}
			return // Valid time range
		}
	}

	// Try to convert both values to float64 for comparison
	minVal, err1 := convertToFloat(min)
	maxVal, err2 := convertToFloat(max)

	if err1 != nil || err2 != nil {
		panic("Between values must be numeric or time.Time")
	}

	if minVal > maxVal {
		panic("Between min must be less than or equal to max")
	}
}

// convertToFloat tries to convert numeric types to float64
func convertToFloat(v any) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("non-numeric type: %T", v)
	}
}

func validateNotEmpty[T any](values []T, operator string) {
	if len(values) == 0 {
		panic(operator + " requires at least one value")
	}
}
