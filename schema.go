package mcpio

import (
	"github.com/google/jsonschema-go/jsonschema"
)

// FieldDef defines a field for dynamic schema construction
type FieldDef struct {
	Name        string
	Type        string // "string", "number", "boolean", "object", "array"
	Description string
	Required    bool
	Enum        []string // Optional enum values
}

// GenerateSchema is a thin wrapper around jsonschema.For[T]() for convenience
func GenerateSchema[T any]() (*jsonschema.Schema, error) {
	return jsonschema.For[T](nil)
}

// CreateDynamicSchema constructs a JSON schema from field definitions
// This is useful for runtime-determined schemas (e.g., from Lua script inspection)
func CreateDynamicSchema(fields []FieldDef) *jsonschema.Schema {
	properties := make(map[string]*jsonschema.Schema)
	var required []string

	for _, field := range fields {
		schema := &jsonschema.Schema{
			Type:        field.Type,
			Description: field.Description,
		}

		if len(field.Enum) > 0 {
			enum := make([]any, len(field.Enum))
			for i, v := range field.Enum {
				enum[i] = v
			}
			schema.Enum = enum
		}

		properties[field.Name] = schema

		if field.Required {
			required = append(required, field.Name)
		}
	}

	return &jsonschema.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// CreateStringSchema creates a simple string schema with optional constraints
func CreateStringSchema(description string, enum []string) *jsonschema.Schema {
	schema := &jsonschema.Schema{
		Type:        "string",
		Description: description,
	}

	if len(enum) > 0 {
		enumAny := make([]any, len(enum))
		for i, v := range enum {
			enumAny[i] = v
		}
		schema.Enum = enumAny
	}

	return schema
}

// CreateObjectSchema creates a simple object schema with string properties
func CreateObjectSchema(description string, properties map[string]string, required []string) *jsonschema.Schema {
	props := make(map[string]*jsonschema.Schema)

	for name, desc := range properties {
		props[name] = &jsonschema.Schema{
			Type:        "string",
			Description: desc,
		}
	}

	return &jsonschema.Schema{
		Type:        "object",
		Description: description,
		Properties:  props,
		Required:    required,
	}
}
