package mcpio

import (
	"context"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for typed prompt tests
type DocumentPromptArgs struct {
	DocumentType string `json:"documentType" jsonschema:"description:Type of document to generate"`
	Topic        string `json:"topic"        jsonschema:"description:Main topic or subject"`
	Tone         string `json:"tone"         jsonschema:"description:Writing tone (formal, casual, etc)"`
}

type SimplePromptArgs struct {
	Message string `json:"message" jsonschema:"description:Message to include"`
}

func TestWithTypedPrompt_Basic(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args DocumentPromptArgs) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: "Write a " + args.DocumentType + " about " + args.Topic + " with " + args.Tone + " tone",
			}},
			Description: "Generated document prompt",
		}, nil
	}

	handler, err := NewHandler(
		WithName("document-generator"),
		WithTypedPrompt("document_writer", "Generate document prompts", promptFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestWithTypedPrompt_EmptyName(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args SimplePromptArgs) (*PromptResult, error) {
		return &PromptResult{Messages: []PromptMessage{{Role: "user", Content: args.Message}}}, nil
	}

	_, err := NewHandler(
		WithName("test"),
		WithTypedPrompt("", "Empty name test", promptFunc),
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyValue)
}

func TestWithTypedPrompt_SchemaGeneration(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args DocumentPromptArgs) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: "Generated prompt",
			}},
		}, nil
	}

	// Test that schema generation works without errors
	handler, err := NewHandler(
		WithName("schema-test"),
		WithTypedPrompt("document_test", "Test schema generation", promptFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestCreateTypedPromptHandler_Success(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args SimplePromptArgs) (*PromptResult, error) {
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: "Message: " + args.Message,
			}},
			Description: "Simple prompt",
		}, nil
	}

	handler := createTypedPromptHandler(promptFunc)
	require.NotNil(t, handler)

	// Test handler execution
	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "test",
			Arguments: map[string]string{
				"message": "Hello World",
			},
		},
	}

	result, err := handler(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, "user", string(result.Messages[0].Role))
	assert.Equal(t, "Message: Hello World", result.Messages[0].Content.(*mcp.TextContent).Text)
	assert.Equal(t, "Simple prompt", result.Description)
}

func TestCreateTypedPromptHandler_EmptyArguments(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args SimplePromptArgs) (*PromptResult, error) {
		message := args.Message
		if message == "" {
			message = "default message"
		}
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: message,
			}},
		}, nil
	}

	handler := createTypedPromptHandler(promptFunc)

	// Test with nil arguments
	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "test",
			Arguments: nil,
		},
	}

	result, err := handler(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "default message", result.Messages[0].Content.(*mcp.TextContent).Text)
}

func TestCreateTypedPromptHandler_InvalidJSON(t *testing.T) {
	t.Parallel()

	type InvalidArgs struct {
		Count int `json:"count"`
	}

	promptFunc := func(ctx context.Context, args InvalidArgs) (*PromptResult, error) {
		return &PromptResult{Messages: []PromptMessage{{Role: "user", Content: "test"}}}, nil
	}

	handler := createTypedPromptHandler(promptFunc)

	// Test with invalid argument type that can't be unmarshaled
	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "test",
			Arguments: map[string]string{
				"count": "not a number", // This should fail to unmarshal to int
			},
		},
	}

	_, err := handler(ctx, req)
	require.ErrorIs(t, err, ErrUnmarshalContent)
}

func TestCreateTypedPromptHandler_PromptError(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args SimplePromptArgs) (*PromptResult, error) {
		return nil, assert.AnError // Simulate prompt function error
	}

	handler := createTypedPromptHandler(promptFunc)

	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "test",
			Arguments: map[string]string{
				"message": "test",
			},
		},
	}

	_, err := handler(ctx, req)
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestSchemaToPromptArguments(t *testing.T) {
	t.Parallel()

	// Test with nil schema
	args, err := schemaToPromptArguments(nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNilValue)
	assert.Nil(t, args)

	// Test with schema that has no properties
	schema := &jsonschema.Schema{
		Type: "object",
	}
	args, err = schemaToPromptArguments(schema)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoSchemaProperties)
	assert.Nil(t, args)

	// Test with valid schema
	schema = &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"name": {
				Type:        "string",
				Description: "The name field",
			},
			"age": {
				Type:        "integer",
				Description: "The age field",
			},
		},
		Required: []string{"name"},
	}

	args, err = schemaToPromptArguments(schema)
	require.NoError(t, err)
	require.Len(t, args, 2)

	// Check that arguments are created correctly
	argMap := make(map[string]*mcp.PromptArgument)
	for _, arg := range args {
		argMap[arg.Name] = arg
	}

	nameArg := argMap["name"]
	require.NotNil(t, nameArg)
	assert.Equal(t, "name", nameArg.Name)
	assert.Equal(t, "The name field", nameArg.Description)
	assert.True(t, nameArg.Required)

	ageArg := argMap["age"]
	require.NotNil(t, ageArg)
	assert.Equal(t, "age", ageArg.Name)
	assert.Equal(t, "The age field", ageArg.Description)
	assert.False(t, ageArg.Required)
}

func TestSchemaToPromptArguments_DeterministicOrdering(t *testing.T) {
	t.Parallel()

	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"zebra": {
				Type:        "string",
				Description: "Last alphabetically",
			},
			"apple": {
				Type:        "string",
				Description: "First alphabetically",
			},
			"middle": {
				Type:        "string",
				Description: "Middle alphabetically",
			},
			"banana": {
				Type:        "string",
				Description: "Second alphabetically",
			},
		},
		Required: []string{"apple", "zebra"},
	}

	// Call the function multiple times to verify consistent ordering
	runs := 5
	var previousOrder []string

	for i := 0; i < runs; i++ {
		args, err := schemaToPromptArguments(schema)
		require.NoError(t, err)
		require.Len(t, args, 4)

		// Extract the order of names
		currentOrder := make([]string, len(args))
		for j, arg := range args {
			currentOrder[j] = arg.Name
		}

		// Verify alphabetical ordering
		assert.Equal(t, []string{"apple", "banana", "middle", "zebra"}, currentOrder,
			"Arguments should be in alphabetical order")

		// Verify consistency across runs
		if i > 0 {
			assert.Equal(t, previousOrder, currentOrder,
				"Argument order should be consistent across multiple calls")
		}

		previousOrder = currentOrder
	}
}

// Tests for createPromptHandler (regular map-based prompts)

func TestCreatePromptHandler_Success(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		name, _ := args["name"].(string)
		if name == "" {
			name = "World"
		}
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: "Hello " + name,
			}},
			Description: "Greeting prompt",
		}, nil
	}

	handler := createPromptHandler(promptFunc)
	require.NotNil(t, handler)

	// Test handler execution with arguments
	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "greeting",
			Arguments: map[string]string{
				"name": "Alice",
			},
		},
	}

	result, err := handler(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, "user", string(result.Messages[0].Role))
	assert.Equal(t, "Hello Alice", result.Messages[0].Content.(*mcp.TextContent).Text)
	assert.Equal(t, "Greeting prompt", result.Description)
}

func TestCreatePromptHandler_EmptyArguments(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		name, _ := args["name"].(string)
		if name == "" {
			name = "default"
		}
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: "Hello " + name,
			}},
		}, nil
	}

	handler := createPromptHandler(promptFunc)

	// Test with nil arguments
	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name:      "greeting",
			Arguments: nil,
		},
	}

	result, err := handler(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Hello default", result.Messages[0].Content.(*mcp.TextContent).Text)

	// Test with empty arguments map
	req.Params.Arguments = map[string]string{}
	result, err = handler(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Hello default", result.Messages[0].Content.(*mcp.TextContent).Text)
}

func TestCreatePromptHandler_MultipleMessages(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		topic, _ := args["topic"].(string)
		if topic == "" {
			topic = "general"
		}
		return &PromptResult{
			Messages: []PromptMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant.",
				},
				{
					Role:    "user",
					Content: "Tell me about " + topic,
				},
				{
					Role:    "assistant",
					Content: "I'd be happy to help with " + topic,
				},
			},
			Description: "Multi-message prompt",
		}, nil
	}

	handler := createPromptHandler(promptFunc)

	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "conversation",
			Arguments: map[string]string{
				"topic": "science",
			},
		},
	}

	result, err := handler(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Messages, 3)

	// Verify system message
	assert.Equal(t, "system", string(result.Messages[0].Role))
	assert.Equal(t, "You are a helpful assistant.", result.Messages[0].Content.(*mcp.TextContent).Text)

	// Verify user message
	assert.Equal(t, "user", string(result.Messages[1].Role))
	assert.Equal(t, "Tell me about science", result.Messages[1].Content.(*mcp.TextContent).Text)

	// Verify assistant message
	assert.Equal(t, "assistant", string(result.Messages[2].Role))
	assert.Equal(t, "I'd be happy to help with science", result.Messages[2].Content.(*mcp.TextContent).Text)

	// Verify description
	assert.Equal(t, "Multi-message prompt", result.Description)
}

func TestCreatePromptHandler_PromptError(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		return nil, assert.AnError // Simulate prompt function error
	}

	handler := createPromptHandler(promptFunc)

	ctx := t.Context()
	req := &mcp.GetPromptRequest{
		Params: &mcp.GetPromptParams{
			Name: "failing",
			Arguments: map[string]string{
				"test": "value",
			},
		},
	}

	_, err := handler(ctx, req)
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestWithPrompt_Integration(t *testing.T) {
	t.Parallel()

	promptFunc := func(ctx context.Context, args map[string]any) (*PromptResult, error) {
		greeting, _ := args["greeting"].(string)
		if greeting == "" {
			greeting = "Hello"
		}
		return &PromptResult{
			Messages: []PromptMessage{{
				Role:    "user",
				Content: greeting + " there!",
			}},
			Description: "Integration test prompt",
		}, nil
	}

	// Test that WithPrompt correctly wires createPromptHandler
	handler, err := NewHandler(
		WithName("integration-test"),
		WithPrompt("greeting", "Simple greeting prompt", promptFunc),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)
}
