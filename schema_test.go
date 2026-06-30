package queryutil

import (
	"strings"
	"testing"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/queryutil/filter"
)

const (
	stringOps = string(filter.OpEq) + "," + string(filter.OpNe) + "," +
		string(filter.OpContains) + "," + string(filter.OpStartswith) + "," +
		string(filter.OpEndswith)
	numberOps = string(filter.OpEq) + "," + string(filter.OpNe) + "," +
		string(filter.OpGt) + "," + string(filter.OpLt) + "," +
		string(filter.OpGte) + "," + string(filter.OpLte) + "," +
		string(filter.OpBetween)
	boolOps = string(filter.OpEq) + "," + string(filter.OpNe)
	dateOps = string(filter.OpEq) + "," + string(filter.OpNe) + "," +
		string(filter.OpGt) + "," + string(filter.OpLt) + "," +
		string(filter.OpGte) + "," + string(filter.OpLte) + "," +
		string(filter.OpBetween)
	nullableOps = string(filter.OpNull) + "," + string(filter.OpNnull)
)

func stringOperators() []string {
	return strings.Split(stringOps, ",")
}

func numberOperators() []string {
	return strings.Split(numberOps, ",")
}

func boolOperators() []string {
	return strings.Split(boolOps, ",")
}

func dateOperators() []string {
	return strings.Split(dateOps, ",")
}

type TestDTO struct {
	ID          int       `json:"id" sort:"true" filter:"true"`
	Name        string    `json:"name" sort:"true"`
	Email       string    `json:"email" filter:"true"`
	CreatedAt   time.Time `json:"created_at" sort:"true" filter:"true"`
	IsActive    bool      `json:"is_active" filter:"true"`
	Rating      float64   `json:"rating" sort:"false" filter:"true"`
	NullField   *string   `json:"null_field" filter:"true"`
	NoTagField  string
	HiddenField string `json:"-" sort:"true"`
}

type EnumDTO struct {
	Status string `json:"status" filter:"true" jsonschema:"enum=pending,enum=processing,enum=completed,enum=failed"`
	Level  string `json:"level" filter:"true" jsonschema:"enum=info,enum=warn,enum=error"`
	Name   string `json:"name" filter:"true"`
}

func TestNewSchemaProvider(t *testing.T) {
	provider := NewSchemaProvider()
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.reflector)
}

func TestForType(t *testing.T) {
	provider := NewSchemaProvider()
	schema := provider.ForType(TestDTO{})
	assert.NotNil(t, schema)

	// Test with pointer type
	schemaPtr := provider.ForType(&TestDTO{})
	assert.NotNil(t, schemaPtr)
}

func TestSortableFields(t *testing.T) {
	provider := NewSchemaProvider()
	schema := provider.ForType(TestDTO{})

	fields := schema.SortableFields()
	expected := []string{"id", "name"}
	assert.ElementsMatch(t, expected, fields)

	// Verify excluded fields
	assert.NotContains(t, fields, "email")       // no sort tag
	assert.NotContains(t, fields, "rating")      // sort:"false"
	assert.NotContains(t, fields, "NoTagField")  // no json tag
	assert.NotContains(t, fields, "HiddenField") // json:"-"
}

func TestFilterOperators(t *testing.T) {
	provider := NewSchemaProvider()
	schema := provider.ForType(TestDTO{})

	operators := schema.FilterOperators()
	assert.NotEmpty(t, operators)

	// Test specific field operators
	assert.ElementsMatch(t, boolOperators(), operators["is_active"])
	assert.ElementsMatch(t, dateOperators(), operators["created_at"])
	assert.ElementsMatch(t, stringOperators(), operators["email"])

	// email is non-nullable string - should have base string operators
	assert.ElementsMatch(t, stringOperators(), operators["email"])

	// null_field is nullable string - should have base string operators
	assert.ElementsMatch(t, stringOperators(), operators["null_field"])

	// Verify excluded fields
	assert.NotContains(t, operators, "name")        // no filter tag
	assert.NotContains(t, operators, "NoTagField")  // no json tag
	assert.NotContains(t, operators, "HiddenField") // json:"-"
}

func TestFieldTagHelpers(t *testing.T) {
	provider := NewSchemaProvider()
	schema := provider.ForType(TestDTO{}).(*SchemaWrapper)

	// Test field tag lookup
	assert.Equal(t, "true", schema.fieldTag("id", "sort"))
	assert.Equal(t, "true", schema.fieldTag("email", "filter"))
	assert.Equal(t, "", schema.fieldTag("nonexistent", "sort"))

	// Test struct field finding
	field := schema.findStructField("name")
	assert.NotNil(t, field)
	assert.Equal(t, "Name", field.Name)

	field = schema.findStructField("invalid")
	assert.Nil(t, field)
}

func TestTypeHelpers(t *testing.T) {
	// Test isSortableType
	assert.True(t, isSortableType(&jsonschema.Schema{Type: "string"}))
	assert.True(t, isSortableType(&jsonschema.Schema{Type: "number"}))
	assert.False(t, isSortableType(&jsonschema.Schema{Type: "boolean"}))
	assert.False(t, isSortableType(&jsonschema.Schema{Type: "string", Format: "date-time"}))

	// Nullable field check
	assert.True(t, isSortableType(&jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "null"},
		},
	}))

	// Test getOperatorsForType
	assert.ElementsMatch(t, stringOperators(),
		getOperatorsForType(&jsonschema.Schema{Type: "string"}))
	assert.ElementsMatch(t, numberOperators(),
		getOperatorsForType(&jsonschema.Schema{Type: "number"}))
	assert.ElementsMatch(t, boolOperators(),
		getOperatorsForType(&jsonschema.Schema{Type: "boolean"}))
	assert.ElementsMatch(t, dateOperators(),
		getOperatorsForType(&jsonschema.Schema{Type: "string", Format: "date-time"}))

	// Test nullable field
	nullableSchema := &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "null"},
		},
	}
	expectedNullableOps := append(stringOperators(), strings.Split(nullableOps, ",")...)
	assert.ElementsMatch(t, expectedNullableOps, getOperatorsForType(nullableSchema))
}

func TestEdgeCases(t *testing.T) {
	// Test empty struct
	type Empty struct{}
	provider := NewSchemaProvider()
	schema := provider.ForType(Empty{})

	assert.Empty(t, schema.SortableFields())
	assert.Empty(t, schema.FilterOperators())

	// Test non-struct type
	assert.NotPanics(t, func() {
		schema := provider.ForType("string")
		assert.NotNil(t, schema)
	})
}

func TestFieldEnums(t *testing.T) {
	provider := NewSchemaProvider()
	schema := provider.ForType(EnumDTO{})

	enums := schema.FieldEnums()
	assert.NotEmpty(t, enums)

	// Status field should have enum values
	assert.ElementsMatch(t,
		[]string{"pending", "processing", "completed", "failed"},
		enums["status"])

	// Level field should have enum values
	assert.ElementsMatch(t,
		[]string{"info", "warn", "error"},
		enums["level"])

	// Name field has no enum
	assert.NotContains(t, enums, "name")
}

func TestFieldEnums_NoEnums(t *testing.T) {
	provider := NewSchemaProvider()
	schema := provider.ForType(TestDTO{})

	enums := schema.FieldEnums()
	assert.Empty(t, enums)
}
