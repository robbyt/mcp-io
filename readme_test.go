package mcpio_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/robbyt/mcp-io/schema"
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

var (
	errDivisionByZero = mcpio.NewToolError("division by zero")
	errUnsupportedOp  = mcpio.ValidationError("unsupported operation")
)

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
			return CalculateOutput{}, errDivisionByZero
		}
		result = input.A / input.B
	default:
		return CalculateOutput{}, fmt.Errorf("operation %q: %w", input.Operation, errUnsupportedOp)
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
	inputSchema := schema.NewObject(
		"Raw processing input",
		map[string]string{
			"data": "Raw data to process",
		},
		[]string{"data"},
	)

	return mcpio.NewHandler(
		mcpio.WithName("raw-processor"),
		mcpio.WithRawTool("process_raw", "Process raw JSON data", inputSchema, processJSON),
	)
}

func TestReadmeExamples(t *testing.T) {
	t.Parallel()
	t.Run("QuickStart", func(t *testing.T) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("example-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		server := httptest.NewServer(handler)
		defer server.Close()

		assert.NotNil(t, server)
		assert.NotEmpty(t, server.URL)
		assert.Contains(t, server.URL, "http://")
	})

	t.Run("SchemaFlexibility_AllOptions", func(t *testing.T) {
		// Traditional struct-based example
		handler1, err := mcpio.NewHandler(
			mcpio.WithName("traditional-example"),
			mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler1)

		// Calculator with custom schema using WithRawTool
		calcRawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			var params map[string]any
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			op := params["operation"].(string)
			a := params["a"].(float64)
			b := params["b"].(float64)

			var result float64
			switch op {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return nil, mcpio.NewToolError("division by zero")
				}
				result = a / b
			}
			return json.Marshal(map[string]any{"result": result})
		}

		handler2, err := mcpio.NewHandler(
			mcpio.WithName("calculator-example"),
			mcpio.WithRawTool("calculator", "Arithmetic calculator", `{
				"type": "object",
				"properties": {
					"operation": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"]},
					"a": {"type": "number"},
					"b": {"type": "number"}
				},
				"required": ["operation", "a", "b"]
			}`, calcRawFunc),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler2)

		// Maximum performance with json.RawMessage using WithRawTool
		processorRawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			var params map[string]any
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			output := make(map[string]any)
			for k, v := range params {
				output[k] = v
			}
			output["processed"] = true
			return json.Marshal(output)
		}

		handler3, err := mcpio.NewHandler(
			mcpio.WithName("fast-processor-example"),
			mcpio.WithRawTool("fast_processor", "High-performance processing",
				json.RawMessage(`{"type":"object","additionalProperties":true}`),
				processorRawFunc),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler3)

		// Dynamic schemas using map[string]any with WithRawTool
		echoRawFunc := func(ctx context.Context, input []byte) ([]byte, error) {
			var params map[string]any
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			message := params["message"].(string)
			repeat := 1
			if r, ok := params["repeat"]; ok {
				repeat = int(r.(float64))
			}

			var echoed []string
			for i := 0; i < repeat; i++ {
				echoed = append(echoed, message)
			}

			result := map[string]any{
				"echoed":  echoed,
				"count":   len(echoed),
				"message": message,
			}
			return json.Marshal(result)
		}

		dynamicSchema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
				"repeat":  map[string]any{"type": "integer", "minimum": 1, "maximum": 10},
			},
			"required": []string{"message"},
		}

		handler4, err := mcpio.NewHandler(
			mcpio.WithName("echo-example"),
			mcpio.WithRawTool("echo", "Echo with repetition", dynamicSchema, echoRawFunc),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler4)
	})

	t.Run("CoreConcepts_Instantiation", func(t *testing.T) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("my-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text", toUpper),
			mcpio.WithTool("calculate", "Do math", calculate),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.GetServer())
	})

	t.Run("InputOutputSchema_Calculator", func(t *testing.T) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("calculator"),
			mcpio.WithTool("calculate", "Perform arithmetic operations", calculate),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		result, err := calculate(context.Background(), CalculateInput{
			Operation: "add",
			A:         5.0,
			B:         3.0,
		})
		require.NoError(t, err)
		assert.InDelta(t, 8.0, result.Result, 0.001)

		_, err = calculate(context.Background(), CalculateInput{
			Operation: "divide",
			A:         5.0,
			B:         0.0,
		})
		require.ErrorIs(t, err, errDivisionByZero)
	})

	t.Run("AdvancedFeatures_RawJSON", func(t *testing.T) {
		handler, err := createRawToolHandler()
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("ComparisonSection_ErrorHandling", func(t *testing.T) {
		_, err := mcpio.NewHandler(mcpio.WithName(""))
		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrEmptyValue)

		_, err = mcpio.NewHandler(mcpio.WithVersion(""))
		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrEmptyValue)

		validationErr := mcpio.ValidationError("test validation error")
		assert.Contains(t, validationErr.Error(), "test validation error")

		processingErr := mcpio.ProcessingError("test processing error")
		assert.Contains(t, processingErr.Error(), "test processing error")

		toolErr := mcpio.NewToolError("test tool error")
		assert.Contains(t, toolErr.Error(), "test tool error")
	})
}

func TestDynamicSchemaCreation(t *testing.T) {
	t.Parallel()
	t.Run("NewObject", func(t *testing.T) {
		schema := schema.NewObject(
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

	t.Run("NewDynamic", func(t *testing.T) {
		fields := []schema.FieldDef{
			{Name: "status", Type: "string", Required: true, Enum: []string{"active", "inactive"}},
			{Name: "count", Type: "number", Required: false},
		}
		schema := schema.NewDynamic(fields)

		require.NotNil(t, schema)
		assert.Equal(t, "object", schema.Type)
		assert.NotNil(t, schema.Properties)
		assert.Contains(t, schema.Required, "status")
		assert.NotContains(t, schema.Required, "count")
	})
}

func TestTransportOptions(t *testing.T) {
	t.Parallel()
	handler, err := mcpio.NewHandler(
		mcpio.WithName("transport-test"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("to_upper", "Convert text", toUpper),
	)
	require.NoError(t, err)
	require.NotNil(t, handler)

	t.Run("HTTPTransport", func(t *testing.T) {
		server := httptest.NewServer(handler)
		defer server.Close()

		assert.NotNil(t, server)
		assert.NotEmpty(t, server.URL)
		assert.Implements(t, (*http.Handler)(nil), handler)
	})

	t.Run("SSETransport", func(t *testing.T) {
		assert.NotNil(t, handler.ServeSSE)

		sseHandler := http.HandlerFunc(handler.ServeSSE)
		assert.NotNil(t, sseHandler)

		server := httptest.NewServer(sseHandler)
		defer server.Close()

		assert.NotNil(t, server)
	})

	t.Run("StdioTransport", func(t *testing.T) {
		assert.NotNil(t, handler.ServeStdio)
	})
}

func TestToolExecution(t *testing.T) {
	t.Parallel()
	t.Run("ToUpperTool", func(t *testing.T) {
		result, err := toUpper(context.Background(), TextInput{Text: "hello world"})
		require.NoError(t, err)
		assert.Equal(t, "HELLO WORLD", result.Result)
	})

	t.Run("CalculateTool", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    CalculateInput
			expected float64
			wantErr  error
		}{
			{"Add", CalculateInput{"add", 10, 5}, 15, nil},
			{"Subtract", CalculateInput{"subtract", 10, 5}, 5, nil},
			{"Multiply", CalculateInput{"multiply", 10, 5}, 50, nil},
			{"Divide", CalculateInput{"divide", 10, 5}, 2, nil},
			{"DivideByZero", CalculateInput{"divide", 10, 0}, 0, errDivisionByZero},
			{"InvalidOperation", CalculateInput{"invalid", 10, 5}, 0, errUnsupportedOp},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := calculate(context.Background(), tc.input)

				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.InDelta(t, tc.expected, result.Result, 0.001)
				}
			})
		}
	})
}

func TestErrorTypes(t *testing.T) {
	t.Parallel()
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
		tests := []struct {
			name     string
			opts     []mcpio.Option
			expected error
		}{
			{"EmptyName", []mcpio.Option{mcpio.WithName("")}, mcpio.ErrEmptyValue},
			{"EmptyVersion", []mcpio.Option{mcpio.WithVersion("")}, mcpio.ErrEmptyValue},
			{"NilServer", []mcpio.Option{mcpio.WithServer(nil)}, mcpio.ErrNilValue},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := mcpio.NewHandler(tt.opts...)
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expected)
			})
		}
	})
}

// Test Session Capabilities examples from README
func TestSessionCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("ElicitationExample", func(t *testing.T) {
		// Type from README
		type UserConfig struct {
			Name        string `json:"name"        jsonschema:"Your full name"`
			Environment string `json:"environment" jsonschema:"enum=dev,enum=prod"`
			Port        int    `json:"port"        jsonschema:"minimum=1024,maximum=65535"`
		}

		// Tool from README
		setupTool := func(ctx context.Context, _ struct{}) (map[string]any, error) {
			// Session is automatically available via context
			result, err := mcpio.ElicitTyped[UserConfig](ctx, "Enter configuration:")
			if err != nil {
				return nil, err
			}

			if result.IsAccepted() {
				var config UserConfig
				if err := result.DecodeContent(&config); err != nil {
					return nil, err
				}
				return map[string]any{"status": "configured", "config": config}, nil
			}

			return map[string]any{"status": "cancelled"}, nil
		}

		// Register tool - session is automatically injected
		handler, err := mcpio.NewHandler(
			mcpio.WithName("interactive-server"),
			mcpio.WithTool("setup", "Interactive setup", setupTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("ProgressNotifications", func(t *testing.T) {
		// Tool from README
		processDataTool := func(ctx context.Context, input struct{ Files []string }) (map[string]any, error) {
			total := float64(len(input.Files))
			for i := range input.Files {
				mcpio.NotifyProgress(ctx, float64(i), total) //nolint:errcheck
				// Process file...
			}
			mcpio.NotifyProgress(ctx, total, total) //nolint:errcheck // Mark complete
			return map[string]any{"status": "done"}, nil
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-server"),
			mcpio.WithTool("process", "Process files", processDataTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("SamplingExampleWithExecution", func(t *testing.T) {
		// Actually test the sampling functionality with a mock
		analyzeTool := func(ctx context.Context, input struct{ Code string }) (map[string]any, error) {
			result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
				Role:    "user",
				Content: "Analyze this code and suggest improvements:\n" + input.Code,
			}}, 2000)
			if err != nil {
				return nil, err
			}

			return map[string]any{"analysis": result.Content.Text}, nil
		}

		// Create a mock session that supports sampling
		mockSession := testutil.NewMockSamplingSession(true, &capabilities.MessageResult{
			Role:    "assistant",
			Content: capabilities.TextContent{Text: "This code looks good. No issues found."},
		})

		ctx := mcpio.InjectSessionForTesting(context.Background(), mockSession)

		// Execute the tool
		result, err := analyzeTool(ctx, struct{ Code string }{Code: "func test() {}"})
		require.NoError(t, err)
		assert.Contains(t, result["analysis"], "looks good")
	})

	t.Run("DungeonMasterExample", func(t *testing.T) {
		// Test dungeon_master example from README Sampling section
		type AdventureInput struct {
			Action string `json:"action" jsonschema:"What the player does"`
		}

		dungeonMaster := func(ctx context.Context, input AdventureInput) (map[string]any, error) {
			// Delegate storytelling to the client's LLM
			prompt := "You are a dungeon master. The player: \"" + input.Action +
				"\". Narrate what happens next in 2 sentences. Be dramatic!"

			result, err := mcpio.CreateMessage(ctx, []*capabilities.Message{{
				Role:    "user",
				Content: prompt,
			}}, 300)
			if err != nil {
				return nil, err
			}

			// Use the LLM's narrative in our response
			return map[string]any{"narrative": result.Content.Text}, nil
		}

		// Test handler creation
		handler, err := mcpio.NewHandler(
			mcpio.WithName("dungeon-master"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("dungeon_master", "Narrate adventures using AI", dungeonMaster),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test execution with mock
		mockSession := testutil.NewMockSamplingSession(true, &capabilities.MessageResult{
			Role: "assistant",
			Content: capabilities.TextContent{
				Text: "The ancient door creaks open, revealing a dimly lit corridor filled with mysterious glowing runes. A cold wind rushes past you, carrying whispers of forgotten secrets!",
			},
		})

		ctx := mcpio.InjectSessionForTesting(context.Background(), mockSession)
		result, err := dungeonMaster(ctx, AdventureInput{Action: "I open the mysterious door"})
		require.NoError(t, err)
		assert.Contains(t, result, "narrative")
		assert.Contains(t, result["narrative"], "door")
	})

	t.Run("LoggingExample", func(t *testing.T) {
		// Tool from README
		type MyInput struct {
			Items []string `json:"items"`
		}
		type MyOutput struct {
			Processed int `json:"processed"`
		}

		myTool := func(ctx context.Context, input MyInput) (MyOutput, error) {
			mcpio.LogInfo(ctx, "Processing started", map[string]any{ //nolint:errcheck
				"itemCount": len(input.Items),
			})

			processed := 0
			var remaining int

			// Do work...
			processed = len(input.Items)
			remaining = 0

			mcpio.LogDebug(ctx, "Detailed state", map[string]any{ //nolint:errcheck
				"processed": processed,
				"remaining": remaining,
			})

			return MyOutput{Processed: processed}, nil
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("logging-server"),
			mcpio.WithTool("process", "Process items", myTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("DirectSessionAccess", func(t *testing.T) {
		// Tool from README
		advancedTool := func(ctx context.Context, _ struct{}) (map[string]any, error) {
			session := mcpio.GetSession(ctx)
			if session == nil {
				return map[string]any{"error": "no session available"}, nil
			}

			// Check capabilities
			var features []string
			if session.SupportsElicitation() {
				features = append(features, "elicitation")
			}
			if session.SupportsSampling() {
				features = append(features, "sampling")
			}

			// Access session info
			sessionID := session.SessionID()

			return map[string]any{"sessionID": sessionID, "features": features}, nil
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("advanced-server"),
			mcpio.WithTool("info", "Get session info", advancedTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})
}
