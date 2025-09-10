package mcpio

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name string `json:"name" jsonschema:"User name"`
	Age  int    `json:"age"  jsonschema:"User age"`
}

func TestGenerateSchema(t *testing.T) {
	schema, err := GenerateSchema[TestStruct]()
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)
}

func TestCreateDynamicSchema(t *testing.T) {
	tests := []struct {
		name     string
		fields   []FieldDef
		expected func(*testing.T, *jsonschema.Schema)
	}{
		{
			name:   "empty fields",
			fields: []FieldDef{},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "object", s.Type)
				assert.Empty(t, s.Properties)
				assert.Empty(t, s.Required)
			},
		},
		{
			name: "single string field",
			fields: []FieldDef{
				{Name: "name", Type: "string", Description: "User name", Required: true},
			},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "object", s.Type)
				assert.Len(t, s.Properties, 1)
				assert.Contains(t, s.Properties, "name")
				assert.Equal(t, []string{"name"}, s.Required)
			},
		},
		{
			name: "multiple fields with different types",
			fields: []FieldDef{
				{Name: "name", Type: "string", Description: "User name", Required: true},
				{Name: "age", Type: "number", Description: "User age", Required: false},
				{Name: "active", Type: "boolean", Description: "Is active", Required: true},
			},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "object", s.Type)
				assert.Len(t, s.Properties, 3)
				assert.Contains(t, s.Properties, "name")
				assert.Contains(t, s.Properties, "age")
				assert.Contains(t, s.Properties, "active")
				assert.ElementsMatch(t, []string{"name", "active"}, s.Required)
			},
		},
		{
			name: "field with enum values",
			fields: []FieldDef{
				{Name: "status", Type: "string", Description: "Status", Required: true, Enum: []string{"active", "inactive", "pending"}},
			},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				nameSchema := s.Properties["status"]
				assert.NotNil(t, nameSchema)
				assert.NotNil(t, nameSchema.Enum)
				assert.Len(t, nameSchema.Enum, 3)
				assert.Contains(t, nameSchema.Enum, "active")
				assert.Contains(t, nameSchema.Enum, "inactive")
				assert.Contains(t, nameSchema.Enum, "pending")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := CreateDynamicSchema(tt.fields)
			assert.NotNil(t, schema)
			tt.expected(t, schema)
		})
	}
}

func TestCreateStringSchema(t *testing.T) {
	tests := []struct {
		name        string
		description string
		enum        []string
		expected    func(*testing.T, *jsonschema.Schema)
	}{
		{
			name:        "simple string schema",
			description: "A simple string",
			enum:        nil,
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "string", s.Type)
				assert.Equal(t, "A simple string", s.Description)
				assert.Nil(t, s.Enum)
			},
		},
		{
			name:        "string schema with enum",
			description: "String with choices",
			enum:        []string{"option1", "option2", "option3"},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "string", s.Type)
				assert.Equal(t, "String with choices", s.Description)
				assert.NotNil(t, s.Enum)
				assert.Len(t, s.Enum, 3)
				assert.Contains(t, s.Enum, "option1")
				assert.Contains(t, s.Enum, "option2")
				assert.Contains(t, s.Enum, "option3")
			},
		},
		{
			name:        "string schema with empty enum",
			description: "String with empty enum",
			enum:        []string{},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "string", s.Type)
				assert.Nil(t, s.Enum)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := CreateStringSchema(tt.description, tt.enum)
			assert.NotNil(t, schema)
			tt.expected(t, schema)
		})
	}
}

func TestCreateObjectSchema(t *testing.T) {
	tests := []struct {
		name        string
		description string
		properties  map[string]string
		required    []string
		expected    func(*testing.T, *jsonschema.Schema)
	}{
		{
			name:        "empty object schema",
			description: "Empty object",
			properties:  map[string]string{},
			required:    []string{},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "object", s.Type)
				assert.Equal(t, "Empty object", s.Description)
				assert.Empty(t, s.Properties)
				assert.Empty(t, s.Required)
			},
		},
		{
			name:        "object with properties",
			description: "User object",
			properties:  map[string]string{"name": "User name", "email": "User email"},
			required:    []string{"name"},
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "object", s.Type)
				assert.Equal(t, "User object", s.Description)
				assert.Len(t, s.Properties, 2)
				assert.Contains(t, s.Properties, "name")
				assert.Contains(t, s.Properties, "email")
				assert.Equal(t, []string{"name"}, s.Required)

				nameSchema := s.Properties["name"]
				assert.Equal(t, "string", nameSchema.Type)
				assert.Equal(t, "User name", nameSchema.Description)

				emailSchema := s.Properties["email"]
				assert.Equal(t, "string", emailSchema.Type)
				assert.Equal(t, "User email", emailSchema.Description)
			},
		},
		{
			name:        "object with no required fields",
			description: "Optional fields object",
			properties:  map[string]string{"optional1": "First optional", "optional2": "Second optional"},
			required:    nil,
			expected: func(t *testing.T, s *jsonschema.Schema) {
				t.Helper()
				assert.Equal(t, "object", s.Type)
				assert.Len(t, s.Properties, 2)
				assert.Nil(t, s.Required)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := CreateObjectSchema(tt.description, tt.properties, tt.required)
			assert.NotNil(t, schema)
			tt.expected(t, schema)
		})
	}
}
