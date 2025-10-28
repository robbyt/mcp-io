package mcpio_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	toolOption "github.com/robbyt/mcp-io/primitives/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Example input and output types
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to transform"`
}

type TextOutput struct {
	Result string `json:"result" jsonschema:"Transformed text"`
}

type DivideInput struct {
	Numerator   float64 `json:"numerator"   jsonschema:"Number to be divided"`
	Denominator float64 `json:"denominator" jsonschema:"Number to divide by (cannot be zero)"`
	Precision   int     `json:"precision"   jsonschema:"Decimal places for rounding,minimum=0,maximum=10,default=2"`
}

type DivideOutput struct {
	Result float64 `json:"result" jsonschema:"Division result rounded to specified precision"`
}

// Example tool functions
func toUpper(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func divide(ctx context.Context, input DivideInput) (DivideOutput, error) {
	if input.Denominator == 0 {
		return DivideOutput{}, mcpio.NewToolError("division by zero")
	}

	result := input.Numerator / input.Denominator

	// Round to specified precision
	multiplier := math.Pow(10, float64(input.Precision))
	rounded := math.Round(result*multiplier) / multiplier

	return DivideOutput{Result: rounded}, nil
}

// Helper function for raw JSON tool usage - from README Advanced Features section
func createRawToolHandler() (*mcpio.Handler, error) {
	// Example: A tool that validates and reformats any JSON input
	validateJSON := func(ctx context.Context, input []byte) ([]byte, error) {
		// Unmarshal to confirm it's valid JSON
		var jsonData any
		if err := json.Unmarshal(input, &jsonData); err != nil {
			return nil, mcpio.ValidationError("Invalid JSON: " + err.Error())
		}

		// Re-marshal back to JSON with indentation for pretty formatting
		formatted, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return nil, mcpio.ProcessingError("Failed to format JSON: " + err.Error())
		}

		// Return the formatted JSON wrapped in a result object
		result := map[string]any{
			"formatted_json": string(formatted),
			"valid":          true,
			"size_bytes":     len(input),
		}

		return json.Marshal(result)
	}

	// Define input schema (required) - tells LLM what to send
	inputSchema := `{
    "type": "object",
    "properties": {
        "json_data": {
            "type": "object",
            "description": "Any JSON object or array to validate and format"
        }
    },
    "required": ["json_data"]
}`

	// Define output schema (optional) - tells LLM what to expect back
	outputSchema := `{
    "type": "object",
    "properties": {
        "formatted_json": {"type": "string", "description": "Pretty-printed JSON"},
        "valid": {"type": "boolean", "description": "Whether input was valid JSON"},
        "size_bytes": {"type": "integer", "description": "Size of input in bytes"}
    }
}`

	return mcpio.NewHandler(
		mcpio.WithName("json-processor"),
		mcpio.WithRawTool("validate_json", "Validate and format any JSON input", inputSchema, validateJSON,
			toolOption.WithOutputSchema(outputSchema), // Optional but recommended for LLM understanding
		),
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

	t.Run("CoreConcepts_Instantiation", func(t *testing.T) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("my-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("to_upper", "Convert text", toUpper),
			mcpio.WithTool("divide", "Divide numbers", divide),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.GetServer())
	})

	t.Run("InputOutputSchema_Divide", func(t *testing.T) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("calculator"),
			mcpio.WithTool("divide", "Divide two numbers with configurable precision", divide,
				toolOption.WithReadOnly(),
				toolOption.WithIdempotent(),
			),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test division with precision
		result, err := divide(context.Background(), DivideInput{
			Numerator:   10.0,
			Denominator: 3.0,
			Precision:   2,
		})
		require.NoError(t, err)
		assert.InDelta(t, 3.33, result.Result, 0.001)

		// Test division by zero
		_, err = divide(context.Background(), DivideInput{
			Numerator:   5.0,
			Denominator: 0.0,
			Precision:   2,
		})
		require.Error(t, err)
		var toolErr *mcpio.ToolError
		require.ErrorAs(t, err, &toolErr)
	})

	t.Run("AdvancedFeatures_RawJSON", func(t *testing.T) {
		handler, err := createRawToolHandler()
		require.NoError(t, err)
		assert.NotNil(t, handler)
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

	t.Run("DivideTool", func(t *testing.T) {
		// Test precision rounding
		result, err := divide(context.Background(), DivideInput{
			Numerator:   10.0,
			Denominator: 3.0,
			Precision:   2,
		})
		require.NoError(t, err)
		assert.InDelta(t, 3.33, result.Result, 0.001)

		// Test division by zero
		_, err = divide(context.Background(), DivideInput{
			Numerator:   5.0,
			Denominator: 0.0,
			Precision:   2,
		})
		require.Error(t, err)
		var toolErr *mcpio.ToolError
		require.ErrorAs(t, err, &toolErr)
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

	t.Run("MCPSDKPanics", func(t *testing.T) {
		assert.Panics(t, func() {
			mcp.NewServer(nil, nil)
		}, "MCP SDK should panic when given nil Implementation")
	})
}

// Test Session Capabilities examples from README
func TestSessionCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("ElicitationExample", func(t *testing.T) {
		// Types from README
		type DeleteRecordsInput struct {
			UserID int `json:"userId" jsonschema:"User ID whose records will be deleted"`
		}

		type ConfirmDeletion struct {
			Confirm string `json:"confirm" jsonschema:"Type DELETE to confirm deletion"`
		}

		type Record struct {
			ID   int
			Name string
		}

		// Mock helper functions
		getRecords := func(_ int) []Record {
			return []Record{
				{ID: 1, Name: "record1"},
				{ID: 2, Name: "record2"},
			}
		}

		formatRecordList := func(records []Record) string {
			var list string
			for _, r := range records {
				list += fmt.Sprintf("- Record %d: %s\n", r.ID, r.Name)
			}
			return list
		}

		performDeletion := func(records []Record) int {
			return len(records)
		}

		// Tool from README
		deleteRecords := func(ctx context.Context, input DeleteRecordsInput) (map[string]any, error) {
			// Preview what will be deleted
			records := getRecords(input.UserID)

			if len(records) == 0 {
				return map[string]any{"deleted": 0, "message": "No records found for user"}, nil
			}

			// Build confirmation message showing what will be deleted
			message := fmt.Sprintf(
				"You are about to delete %d records for user ID %d:\n\n%s\n\nType DELETE to confirm.",
				len(records), input.UserID, formatRecordList(records),
			)

			// Ask user to confirm with elicitation
			result, err := mcpio.ElicitTyped[ConfirmDeletion](ctx, message)
			if err != nil {
				// Handle clients that don't support elicitation
				if errors.Is(err, mcpio.ErrElicitationNotSupported) {
					return nil, fmt.Errorf("cannot delete records: client does not support confirmation")
				}
				return nil, err
			}

			if result.IsAccepted() {
				var conf ConfirmDeletion
				if err := result.DecodeContent(&conf); err != nil {
					return nil, err
				}

				if conf.Confirm == "DELETE" {
					// Perform the actual deletion
					deleted := performDeletion(records)
					return map[string]any{
						"deleted": deleted,
						"message": fmt.Sprintf("Deleted %d records", deleted),
					}, nil
				}
			}

			return map[string]any{"status": "cancelled", "deleted": 0}, nil
		}

		// Register tool - session is automatically injected
		handler, err := mcpio.NewHandler(
			mcpio.WithName("database-manager"),
			mcpio.WithTool("delete_records", "Delete database records with confirmation", deleteRecords,
				toolOption.WithDestructive(),
			),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("ProgressNotifications", func(t *testing.T) {
		// Tool from README
		processDataTool := func(ctx context.Context, input struct{ Files []string }) (map[string]any, error) {
			session := mcpio.GetSession(ctx)
			if session == nil {
				return nil, fmt.Errorf("no session available")
			}
			total := float64(len(input.Files))
			for i := range input.Files {
				session.NotifyProgress(ctx, float64(i), total) //nolint:errcheck
				// Process file...
			}
			session.NotifyProgress(ctx, total, total) //nolint:errcheck // Mark complete
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
			session := mcpio.GetSession(ctx)
			if session == nil {
				return nil, fmt.Errorf("no session available")
			}
			result, err := session.CreateMessage(ctx, []*capabilities.Message{{
				Role:    "user",
				Content: "Analyze this code and suggest improvements:\n" + input.Code,
			}}, 2000)
			if err != nil {
				return nil, err
			}

			return map[string]any{"analysis": result.Content.Text}, nil
		}

		// Create a mock session that supports sampling
		mockSession := testutil.NewMockSession()
		mockSession.SetupSampling()
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(&mcp.CreateMessageResult{
			Role:    "assistant",
			Content: &mcp.TextContent{Text: "This code looks good. No issues found."},
		}, nil)

		ctx := testutil.WithSession(t.Context(), mockSession.Session)

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
			session := mcpio.GetSession(ctx)
			if session == nil {
				return nil, fmt.Errorf("no session available")
			}

			// Delegate storytelling to the client's LLM
			prompt := "You are a dungeon master. The player: \"" + input.Action +
				"\". Narrate what happens next in 2 sentences. Be dramatic!"

			result, err := session.CreateMessage(ctx, []*capabilities.Message{{
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
		mockSession := testutil.NewMockSession()
		mockSession.SetupSampling()
		mockSession.On("CreateMessage", mock.Anything, mock.Anything).Return(&mcp.CreateMessageResult{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "The ancient door creaks open, revealing a dimly lit corridor filled with mysterious glowing runes. A cold wind rushes past you, carrying whispers of forgotten secrets!",
			},
		}, nil)

		ctx := testutil.WithSession(t.Context(), mockSession.Session)
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

	t.Run("SessionInterfaceAccess", func(t *testing.T) {
		// Tool from README
		advancedTool := func(ctx context.Context, _ struct{}) (map[string]any, error) {
			session := mcpio.GetSession(ctx)
			if session == nil {
				return nil, errors.New("no session available")
			}

			// Check capabilities before using features
			if session.SupportsElicitation() { //nolint:staticcheck
				// Use elicitation...
			}
			if session.SupportsSampling() { //nolint:staticcheck
				// Use sampling...
			}

			// Check client capabilities
			if session.SupportsRoots() {
				// Client supports filesystem roots
				roots, err := session.ListRoots(ctx)
				if err == nil {
					// Use roots...
					assert.NotNil(t, roots)
				}
			}

			// Access session methods directly
			sessionID := session.SessionID()

			return map[string]any{"sessionID": sessionID}, nil
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("advanced-server"),
			mcpio.WithTool("info", "Get session info", advancedTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})
}
