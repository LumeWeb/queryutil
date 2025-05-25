package queryutil

import (
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
	"go.lumeweb.com/httputil"
	"go.lumeweb.com/queryutil/filter"
)

var _ httputil.SchemaProvider = (*JsonSchamaProvider)(nil)

// JsonSchamaProvider implements httputil.SchemaProvider using jsonschema reflection
type JsonSchamaProvider struct {
	reflector *jsonschema.Reflector
}

// NewSchemaProvider creates a new schema adapter with default configuration
func NewSchemaProvider() *JsonSchamaProvider {
	r := jsonschema.Reflector{
		ExpandedStruct:             true,   // Expand nested structs
		DoNotReference:             false,  // Allow $ref usage
		FieldNameTag:               "json", // Use JSON tags
		RequiredFromJSONSchemaTags: true,
	}

	return &JsonSchamaProvider{
		reflector: &r,
	}
}

// ForType returns a FieldSchema implementation for a given DTO type
func (a *JsonSchamaProvider) ForType(dto any) httputil.FieldSchema {
	t := reflect.TypeOf(dto)
	if t == nil {
		return &SchemaWrapper{
			schema:  &jsonschema.Schema{},
			dtoType: nil,
		}
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var schema *jsonschema.Schema
	if t.Kind() == reflect.Struct {
		schema = a.reflector.Reflect(dto)
	} else {
		schema = &jsonschema.Schema{}
	}
	return &SchemaWrapper{
		schema:  schema,
		dtoType: t,
	}
}

// SchemaWrapper implements httputil.FieldSchema
type SchemaWrapper struct {
	schema  *jsonschema.Schema
	dtoType reflect.Type
}

// SortableFields returns valid sort fields from the schema
func (sw *SchemaWrapper) SortableFields() []string {
	var fields []string

	for pair := sw.schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		fieldName := pair.Key
		prop := pair.Value

		// Skip fields without sort:"true" tag or with json:"-"
		if tag := sw.fieldTag(fieldName, "sort"); tag != "true" {
			continue
		}

		if isSortableType(prop) {
			fields = append(fields, fieldName)
		}
	}

	return fields
}

// FilterOperators returns allowed operators per field
func (sw *SchemaWrapper) FilterOperators() map[string][]string {
	operators := make(map[string][]string)

	for pair := sw.schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		fieldName := pair.Key
		prop := pair.Value

		// Skip fields without filter:"true" tag or with no json tag
		if tag := sw.fieldTag(fieldName, "filter"); tag != "true" {
			continue
		}

		ops := getOperatorsForType(prop)
		if len(ops) > 0 {
			operators[fieldName] = ops
		}
	}

	return operators
}

// Helper functions

func isSortableType(prop *jsonschema.Schema) bool {
	switch {
	case prop.Type == "string" && prop.Format != "date-time":
		return true
	case prop.Type == "number" || prop.Type == "integer":
		return true
	case prop.OneOf != nil: // Check nullable fields
		for _, s := range prop.OneOf {
			if isSortableType(s) {
				return true
			}
		}
	}
	return false
}

func getOperatorsForType(prop *jsonschema.Schema) []string {
	baseType := prop.Type
	isNullable := false

	if baseType == "" && len(prop.OneOf) > 0 {
		for _, s := range prop.OneOf {
			if s.Type == "null" {
				isNullable = true
			} else if s.Type != "" {
				baseType = s.Type
			}
		}
	}

	var ops []string
	switch baseType {
	case "string":
		if prop.Format == "date-time" {
			ops = []string{
				string(filter.OpEq), string(filter.OpNe),
				string(filter.OpGt), string(filter.OpLt),
				string(filter.OpGte), string(filter.OpLte),
				string(filter.OpBetween),
			}
		} else {
			ops = []string{
				string(filter.OpEq), string(filter.OpNe),
				string(filter.OpContains), string(filter.OpStartswith),
				string(filter.OpEndswith),
			}
		}
	case "number", "integer":
		ops = []string{
			string(filter.OpEq), string(filter.OpNe),
			string(filter.OpGt), string(filter.OpLt),
			string(filter.OpGte), string(filter.OpLte),
			string(filter.OpBetween),
		}
	case "boolean":
		ops = []string{string(filter.OpEq), string(filter.OpNe)}
	}

	// Only add null checks if the field is actually nullable
	if isNullable {
		ops = append(ops, string(filter.OpNull), string(filter.OpNnull))
	}

	return ops
}

// fieldTag gets struct tag value for a JSON field
func (sw *SchemaWrapper) fieldTag(jsonField, tagName string) string {
	field := sw.findStructField(jsonField)
	if field == nil {
		return ""
	}
	return field.Tag.Get(tagName)
}

// findStructField finds the struct field by JSON tag name
func (sw *SchemaWrapper) findStructField(jsonName string) *reflect.StructField {
	for i := 0; i < sw.dtoType.NumField(); i++ {
		field := sw.dtoType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			continue
		}

		// Handle json tag formats: "name,omitempty"
		parts := strings.Split(tag, ",")
		if parts[0] == jsonName {
			return &field
		}
	}
	return nil
}
