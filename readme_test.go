package mcpio_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcpio "github.com/robbyt/go-mcpio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example input and output types
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to transform"`
}

type TextOutput struct {
	Result string `json:"result" jsonschema:"Transformed text"`
}

type CalculateInput struct {
	Operation string  `json:"operation" jsonschema:"Arithmetic operation to perform"`
	A         float64 `json:"a"         jsonschema:"First number"`
	B         float64 `json:"b"         jsonschema:"Second number"`
}

type CalculateOutput struct {
	Result float64 `json:"result" jsonschema:"Calculation result"`
}

// Example tool functions
func toUpper(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func calculate(ctx context.Context, input CalculateInput) (CalculateOutput, error) {
	var result float64
	switch input.Operation {
	case "add":
		result = input.A + input.B
	case "subtract":
		result = input.A - input.B
	case "multiply":
		result = input.A * input.B
	case "divide":
		if input.B == 0 {
			return CalculateOutput{}, mcpio.NewToolError("division by zero")
		}
		result = input.A / input.B
	default:
		return CalculateOutput{}, mcpio.ValidationError("unsupported operation: " + input.Operation)
	}
	return CalculateOutput{Result: result}, nil
}

// Helper function for raw JSON tool usage
func createRawToolHandler() (*mcpio.Handler, error) {
	// Raw function that works with JSON bytes
	processJSON := func(ctx context.Context, input []byte) ([]byte, error) {
		// Custom JSON processing logic here
		return []byte(`{"processed": true, "input_length": ` + fmt.Sprintf("%d", len(input)) + `}`), nil
	}

	// Define input schema for the raw tool
	inputSchema := mcpio.CreateObjectSchema(
		"Raw processing input",
		map[string]string{
			"data": "Raw data to process",
		},
		[]string{"data"},
	)

	return mcpio.New(
		mcpio.WithName("raw-processor"),
		mcpio.WithRawTool("process_raw", "Process raw JSON data", inputSchema, processJSON),
	)
}

// Helper function for script tool usage
func createScriptToolHandler() (*mcpio.Handler, error) {
	// Mock script evaluator for demonstration
	mockEvaluator := &mockScriptEvaluator{
		script: "return {result: input.data * 2}",
	}

	return mcpio.New(
		mcpio.WithName("script-server"),
		mcpio.WithScriptTool("lua_double", "Double the input using Lua", mockEvaluator),
	)
}

// Mock script evaluator for the example
type mockScriptEvaluator struct {
	script string
}

func (m *mockScriptEvaluator) Execute(ctx context.Context, input []byte) ([]byte, error) {
	// In a real implementation, this would execute the script
	return []byte(`{"result": "script executed", "input_received": true}`), nil
}

func (m *mockScriptEvaluator) GetTimeout() time.Duration {
	return 5 * time.Second
}

// Helper function for schema generation testing
func createSchemaGenerationHandler() (*mcpio.Handler, error) {
	type MyInput struct {
		Name string `json:"name" jsonschema:"User's full name"`
		Age  int    `json:"age"  jsonschema:"User's age in years"`
	}

	type MyOutput struct {
		Greeting string `json:"greeting" jsonschema:"Personalized greeting message"`
	}

	greetFunc := func(ctx context.Context, input MyInput) (MyOutput, error) {
		return MyOutput{Greeting: fmt.Sprintf("Hello %s, age %d!", input.Name, input.Age)}, nil
	}

	return mcpio.New(
		mcpio.WithName("schema-example"),
		mcpio.WithTool("greet", "Greet user with name and age", greetFunc),
	)
}

// Test README examples with comprehensive assertions
func TestReadmeExamples(t *testing.T) {
	t.Run("QuickStart", func(t *testing.T) {
		// Test exact Quick Start example from README
		handler, err := mcpio.New(
			mcpio.WithName("example-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test with HTTP server as shown in README
		server := httptest.NewServer(handler)
		defer server.Close()

		assert.NotNil(t, server)
		assert.NotEmpty(t, server.URL)
		assert.Contains(t, server.URL, "http://")
	})

	t.Run("FunctionalOptionsAPI", func(t *testing.T) {
		// Test multi-tool example from README Functional Options section
		handler, err := mcpio.New(
			mcpio.WithName("my-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text", toUpper),
			mcpio.WithTool("calculate", "Do math", calculate),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.GetServer())
	})

	t.Run("CalculatorExample", func(t *testing.T) {
		// Test calculator example from README Type-Safe Tools section
		handler, err := mcpio.New(
			mcpio.WithName("calculator"),
			mcpio.WithTool("calculate", "Perform arithmetic operations", calculate),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test that calculate function works correctly
		result, err := calculate(context.Background(), CalculateInput{
			Operation: "add",
			A:         5.0,
			B:         3.0,
		})
		require.NoError(t, err)
		assert.InDelta(t, 8.0, result.Result, 0.001)

		// Test division by zero error
		_, err = calculate(context.Background(), CalculateInput{
			Operation: "divide",
			A:         5.0,
			B:         0.0,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "division by zero")
	})

	t.Run("RawJSONTool", func(t *testing.T) {
		handler, err := createRawToolHandler()
		require.NoError(t, err)
		assert.NotNil(t, handler)

		server := httptest.NewServer(handler)
		defer server.Close()

		assert.NotNil(t, server)
	})

	t.Run("ScriptIntegration", func(t *testing.T) {
		handler, err := createScriptToolHandler()
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test the mock script evaluator
		mockEval := &mockScriptEvaluator{script: "test"}
		result, err := mockEval.Execute(context.Background(), []byte(`{"test": "data"}`))
		require.NoError(t, err)
		assert.Contains(t, string(result), "script executed")
		assert.Equal(t, 5*time.Second, mockEval.GetTimeout())
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test configuration error - empty name
		_, err := mcpio.New(mcpio.WithName(""))
		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrEmptyName)

		// Test configuration error - empty version
		_, err = mcpio.New(mcpio.WithVersion(""))
		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrEmptyVersion)

		// Test tool error types
		validationErr := mcpio.ValidationError("test validation error")
		assert.Contains(t, validationErr.Error(), "test validation error")

		processingErr := mcpio.ProcessingError("test processing error")
		assert.Contains(t, processingErr.Error(), "test processing error")

		toolErr := mcpio.NewToolError("test tool error")
		assert.Contains(t, toolErr.Error(), "test tool error")
	})

	t.Run("JSONSchemaGeneration", func(t *testing.T) {
		handler, err := createSchemaGenerationHandler()
		require.NoError(t, err)
		assert.NotNil(t, handler)

		server := httptest.NewServer(handler)
		defer server.Close()

		assert.NotNil(t, server)
	})
}

// Test Dynamic Schema Creation (from README Schema Generation section)
func TestDynamicSchemaCreation(t *testing.T) {
	t.Run("CreateObjectSchema", func(t *testing.T) {
		// Test CreateObjectSchema example from README
		schema := mcpio.CreateObjectSchema(
			"Dynamic input",
			map[string]string{
				"field1": "First field",
				"field2": "Second field",
			},
			[]string{"field1"}, // required fields
		)

		require.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "Dynamic input", schema.Description)
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Required, "field1")
		assert.NotContains(t, schema.Required, "field2")
	})

	t.Run("CreateDynamicSchema", func(t *testing.T) {
		// Test CreateDynamicSchema example from README
		fields := []mcpio.FieldDef{
			{Name: "status", Type: "string", Required: true, Enum: []string{"active", "inactive"}},
			{Name: "count", Type: "number", Required: false},
		}
		schema := mcpio.CreateDynamicSchema(fields)

		require.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Required, "status")
		assert.NotContains(t, schema.Required, "count")
	})
}

// Test Transport Options (from README Transport Options section)
func TestTransportOptions(t *testing.T) {
	handler, err := mcpio.New(
		mcpio.WithName("transport-test"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text", toUpper),
	)
	require.NoError(t, err)
	require.NotNil(t, handler)

	t.Run("HTTPTransport", func(t *testing.T) {
		// Test HTTP transport as shown in README
		server := httptest.NewServer(handler)
		defer server.Close()

		assert.NotNil(t, server)
		assert.NotEmpty(t, server.URL)

		// Verify handler implements http.Handler interface
		assert.Implements(t, (*http.Handler)(nil), handler)
	})

	t.Run("SSETransport", func(t *testing.T) {
		// Test SSE transport method exists and is callable
		assert.NotNil(t, handler.ServeSSE)

		// Test SSE handler can be wrapped in http.HandlerFunc
		sseHandler := http.HandlerFunc(handler.ServeSSE)
		assert.NotNil(t, sseHandler)

		server := httptest.NewServer(sseHandler)
		defer server.Close()

		assert.NotNil(t, server)
	})

	t.Run("StdioTransport", func(t *testing.T) {
		// Test ServeStdio method exists and signature is correct
		assert.NotNil(t, handler.ServeStdio)

		// Note: We can't easily test stdio in unit tests without complex mocking,
		// but we can verify the method signature and that it doesn't panic with nil inputs
		// In real usage, this would be: handler.ServeStdio(os.Stdin, os.Stdout)
	})
}

// Test Tool Functionality (actual execution of tools)
func TestToolExecution(t *testing.T) {
	t.Run("ToUpperTool", func(t *testing.T) {
		// Test the toUpper function directly
		result, err := toUpper(context.Background(), TextInput{Text: "hello world"})
		require.NoError(t, err)
		assert.Equal(t, "HELLO WORLD", result.Result)
	})

	t.Run("CalculateTool", func(t *testing.T) {
		testCases := []struct {
			name      string
			input     CalculateInput
			expected  float64
			expectErr bool
			errMsg    string
		}{
			{"Add", CalculateInput{"add", 10, 5}, 15, false, ""},
			{"Subtract", CalculateInput{"subtract", 10, 5}, 5, false, ""},
			{"Multiply", CalculateInput{"multiply", 10, 5}, 50, false, ""},
			{"Divide", CalculateInput{"divide", 10, 5}, 2, false, ""},
			{"DivideByZero", CalculateInput{"divide", 10, 0}, 0, true, "division by zero"},
			{"InvalidOperation", CalculateInput{"invalid", 10, 5}, 0, true, "unsupported operation"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := calculate(context.Background(), tc.input)

				if tc.expectErr {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tc.errMsg)
				} else {
					require.NoError(t, err)
					assert.InDelta(t, tc.expected, result.Result, 0.001)
				}
			})
		}
	})
}

// Test error types from README Error Handling section
func TestErrorTypes(t *testing.T) {
	t.Run("ToolError", func(t *testing.T) {
		toolErr := mcpio.NewToolError("test message")
		assert.Contains(t, toolErr.Error(), "test message")
	})

	t.Run("ValidationError", func(t *testing.T) {
		validErr := mcpio.ValidationError("validation failed")
		assert.Contains(t, validErr.Error(), "validation failed")
	})

	t.Run("ProcessingError", func(t *testing.T) {
		procErr := mcpio.ProcessingError("processing failed")
		assert.Contains(t, procErr.Error(), "processing failed")
	})

	t.Run("ConfigurationErrors", func(t *testing.T) {
		// Test various configuration errors as shown in README
		tests := []struct {
			name     string
			opts     []mcpio.Option
			expected error
		}{
			{"EmptyName", []mcpio.Option{mcpio.WithName("")}, mcpio.ErrEmptyName},
			{"EmptyVersion", []mcpio.Option{mcpio.WithVersion("")}, mcpio.ErrEmptyVersion},
			{"NilServer", []mcpio.Option{mcpio.WithServer(nil)}, mcpio.ErrNilServer},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := mcpio.New(tt.opts...)
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expected)
			})
		}
	})
}
