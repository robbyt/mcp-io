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
	"github.com/robbyt/mcp-io/capabilities/sampling"
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
func toUpper(ctx context.Context, toolCtx mcpio.RequestContext, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

func divide(ctx context.Context, toolCtx mcpio.RequestContext, input DivideInput) (DivideOutput, error) {
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
	validateJSON := func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
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
		mockToolCtx := testutil.NewMockRequestContext(nil)
		result, err := divide(t.Context(), mockToolCtx, DivideInput{
			Numerator:   10.0,
			Denominator: 3.0,
			Precision:   2,
		})
		require.NoError(t, err)
		assert.InDelta(t, 3.33, result.Result, 0.001)

		// Test division by zero
		_, err = divide(t.Context(), mockToolCtx, DivideInput{
			Numerator:   5.0,
			Denominator: 0.0,
			Precision:   2,
		})
		require.Error(t, err)
		var toolErr *mcpio.ToolError
		require.ErrorAs(t, err, &toolErr)
	})
}

func TestToolMetadata_MultipleTools(t *testing.T) {
	t.Parallel()

	// Define output types (MCP requires object schemas)
	type RecordsOutput struct {
		Records []string `json:"records" jsonschema:"List of retrieved records"`
	}

	type UpdateOutput struct {
		Success bool `json:"success" jsonschema:"Whether update succeeded"`
	}

	type DeleteOutput struct {
		Deleted int `json:"deleted" jsonschema:"Number of records deleted"`
	}

	type BackupOutput struct {
		BackupID string `json:"backupId" jsonschema:"Backup identifier"`
	}

	// Define tool functions
	getRecords := func(ctx context.Context, toolCtx mcpio.RequestContext, _ struct{}) (RecordsOutput, error) {
		return RecordsOutput{Records: []string{"record1", "record2"}}, nil
	}

	updateRecord := func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
		ID int `json:"id" jsonschema:"Record ID to update"`
	},
	) (UpdateOutput, error) {
		return UpdateOutput{Success: true}, nil
	}

	deleteRecords := func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
		IDs []int `json:"ids" jsonschema:"Record IDs to delete"`
	},
	) (DeleteOutput, error) {
		return DeleteOutput{Deleted: len(input.IDs)}, nil
	}

	backupDatabase := func(ctx context.Context, toolCtx mcpio.RequestContext, input struct {
		Location string `json:"location" jsonschema:"Backup storage location"`
	},
	) (BackupOutput, error) {
		return BackupOutput{BackupID: "backup-" + input.Location}, nil
	}

	// Create handler matching README example
	handler, err := mcpio.NewHandler(
		mcpio.WithName("database-manager"),
		mcpio.WithVersion("1.0.0"),

		mcpio.WithTool("get_records", "Retrieve records from database", getRecords,
			toolOption.WithReadOnly(),
			toolOption.WithIdempotent(),
		),

		mcpio.WithTool("update_record", "Update existing database record", updateRecord,
			toolOption.WithIdempotent(),
		),

		mcpio.WithTool("delete_records", "Permanently delete records", deleteRecords,
			toolOption.WithDestructive(),
		),

		mcpio.WithTool("backup_database", "Backup to cloud storage", backupDatabase,
			toolOption.WithReadOnly(),
			toolOption.WithOpenWorld(),
		),
	)

	require.NoError(t, err)
	assert.NotNil(t, handler)

	server := handler.GetServer()
	assert.NotNil(t, server)
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

	t.Run("StreamableHTTPTransport", func(t *testing.T) {
		assert.NotNil(t, handler.ServeHTTP)

		httpHandler := http.HandlerFunc(handler.ServeHTTP)
		assert.NotNil(t, httpHandler)

		server := httptest.NewServer(httpHandler)
		defer server.Close()

		assert.NotNil(t, server)
	})

	t.Run("StdioTransport", func(t *testing.T) {
		assert.NotNil(t, handler.ServeStdio)
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
		deleteRecords := func(ctx context.Context, toolCtx mcpio.RequestContext, input DeleteRecordsInput) (map[string]any, error) {
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
			elicitor := mcpio.NewElicitor(toolCtx)
			result, err := mcpio.ElicitTyped[ConfirmDeletion](ctx, elicitor, message)
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

	t.Run("DungeonMasterExample", func(t *testing.T) {
		// Test dungeon_master example from README Sampling section
		type AdventureInput struct {
			Action string `json:"action" jsonschema:"What the player does"`
		}

		dungeonMaster := func(ctx context.Context, toolCtx mcpio.RequestContext, input AdventureInput) (map[string]any, error) {
			session := toolCtx.GetSession()
			if session == nil {
				return nil, fmt.Errorf("no session available")
			}

			// Delegate storytelling to the client's LLM
			prompt := "You are a dungeon master. The player: \"" + input.Action +
				"\". Narrate what happens next in 2 sentences. Be dramatic!"

			result, err := session.CreateMessage(ctx, []*capabilities.Message{{
				Role:    "user",
				Content: prompt,
			}}, sampling.WithMaxTokens(300))
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
		mockSession.On("ID").Return("test-session-123")
		mockSession.On("CreateMessage", mock.Anything, mock.Anything, mock.Anything).Return(&mcp.CreateMessageResult{
			Role: "assistant",
			Content: &mcp.TextContent{
				Text: "The ancient door creaks open, revealing a dimly lit corridor filled with mysterious glowing runes. A cold wind rushes past you, carrying whispers of forgotten secrets!",
			},
		}, nil)

		mockToolCtx := testutil.NewMockRequestContext(mockSession.Session)
		result, err := dungeonMaster(t.Context(), mockToolCtx, AdventureInput{Action: "I open the mysterious door"})
		require.NoError(t, err)
		assert.Contains(t, result, "narrative")
		assert.Contains(t, result["narrative"], "door")
	})

	t.Run("SamplingExampleWithExecution", func(t *testing.T) {
		// Actually test the sampling functionality with a mock
		analyzeTool := func(ctx context.Context, toolCtx mcpio.RequestContext, input struct{ Code string }) (map[string]any, error) {
			session := toolCtx.GetSession()
			if session == nil {
				return nil, fmt.Errorf("no session available")
			}
			result, err := session.CreateMessage(ctx, []*capabilities.Message{{
				Role:    "user",
				Content: "Analyze this code and suggest improvements:\n" + input.Code,
			}}, sampling.WithMaxTokens(2000))
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

		mockToolCtx := testutil.NewMockRequestContext(mockSession.Session)

		// Execute the tool
		result, err := analyzeTool(t.Context(), mockToolCtx, struct{ Code string }{Code: "func test() {}"})
		require.NoError(t, err)
		assert.Contains(t, result["analysis"], "looks good")
	})

	t.Run("ProgressNotifications", func(t *testing.T) {
		// Tool from README - Basic Progress example
		type TaskInput struct {
			Items []string `json:"items"`
		}
		type TaskOutput struct {
			Status string `json:"status"`
		}

		backgroundTask := func(ctx context.Context, toolCtx mcpio.RequestContext, input TaskInput) (TaskOutput, error) {
			session := toolCtx.GetSession()
			total := len(input.Items)

			for i, item := range input.Items {
				session.NotifyProgress(ctx, float64(i+1), float64(total)) //nolint:errcheck
				// Process item...
				_ = item
			}
			return TaskOutput{Status: "done"}, nil
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("progress-server"),
			mcpio.WithTool("process", "Process items", backgroundTask),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test execution
		mockSession := testutil.NewMockSession()
		mockSession.On("NotifyProgress", mock.Anything, mock.Anything).Return(nil)
		mockToolCtx := testutil.NewMockRequestContext(mockSession.Session)
		result, err := backgroundTask(t.Context(), mockToolCtx, TaskInput{Items: []string{"a", "b", "c"}})
		require.NoError(t, err)
		assert.Equal(t, "done", result.Status)
	})

	t.Run("LoggingExample", func(t *testing.T) {
		// Tool from README
		type MyInput struct {
			Items []string `json:"items"`
		}
		type MyOutput struct {
			Processed int `json:"processed"`
		}

		myTool := func(ctx context.Context, toolCtx mcpio.RequestContext, input MyInput) (MyOutput, error) {
			toolCtx.GetSession().LogInfo(ctx, "Processing started", map[string]any{ //nolint:errcheck
				"itemCount": len(input.Items),
			})

			processed := 0
			var remaining int

			// Do work...
			processed = len(input.Items)
			remaining = 0

			toolCtx.GetSession().LogDebug(ctx, "Detailed state", map[string]any{ //nolint:errcheck
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

		// Test the tool function directly
		mockSession := testutil.NewMockSession()
		mockSession.On("Log", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockToolCtx := testutil.NewMockRequestContext(mockSession.Session)
		result, err := myTool(t.Context(), mockToolCtx, MyInput{Items: []string{"a", "b"}})
		require.NoError(t, err)
		assert.Equal(t, 2, result.Processed)
	})

	t.Run("RequestMetadataAccess", func(t *testing.T) {
		// Types for the tool
		type MyInput struct {
			Data string `json:"data"`
		}
		type MyOutput struct {
			Result string `json:"result"`
		}

		// Tool from README
		myTool := func(ctx context.Context, toolCtx mcpio.RequestContext, input MyInput) (MyOutput, error) {
			// Get the tool/prompt/resource identifier
			identifier := toolCtx.GetIdentifier()

			// Read custom HTTP headers from the client
			clientVersion := toolCtx.GetHeaders().Get("X-Client-Version")
			deploymentEnv := toolCtx.GetHeaders().Get("X-Deployment-Env")

			// Use backend logger for analytics/audit (NOT MCP logs which go to LLM)
			// Note: slog.Info would log to stderr/stdout, not to MCP client

			// Verify the context helpers work
			assert.NotNil(t, identifier, "identifier should not be nil")
			assert.NotNil(t, clientVersion, "header getter should not return nil")
			assert.NotNil(t, deploymentEnv, "header getter should not return nil")

			// Process request...
			output := MyOutput{Result: "processed: " + input.Data}
			return output, nil
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("metadata-server"),
			mcpio.WithTool("process", "Process with metadata", myTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test the tool function directly
		mockToolCtx := testutil.NewMockRequestContext(nil)
		result, err := myTool(t.Context(), mockToolCtx, MyInput{Data: "test"})
		require.NoError(t, err)
		assert.Equal(t, "processed: test", result.Result)
	})

	t.Run("SessionInterfaceAccess", func(t *testing.T) {
		// Tool from README
		advancedTool := func(ctx context.Context, toolCtx mcpio.RequestContext, _ struct{}) (map[string]any, error) {
			session := toolCtx.GetSession()
			if session == nil {
				return nil, errors.New("no session available")
			}

			// Check capabilities before using features
			if session.SupportsElicitation() {
				_ = session // Use elicitation...
			}
			if session.SupportsSampling() {
				_ = session // Use sampling...
			}

			// Access client capabilities for detailed information
			caps := session.ClientCapabilities()
			if caps.Roots != nil && caps.Roots.ListChanged {
				roots, err := session.ListRoots(ctx)
				if err == nil {
					_ = roots
				}
			}

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

func TestAdvancedFeatures(t *testing.T) {
	t.Parallel()

	t.Run("RawJSONTools", func(t *testing.T) {
		// Test the validateJSON raw tool example from README Advanced Features section
		handler, err := createRawToolHandler()
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})
}

func TestSchemaTypeOptions(t *testing.T) {
	t.Parallel()

	// Tool function for all three variations
	simpleTool := func(ctx context.Context, toolCtx mcpio.RequestContext, input []byte) ([]byte, error) {
		return []byte(`{"status":"processed"}`), nil
	}

	t.Run("JSONStringFormat", func(t *testing.T) {
		// Option 1: JSON string (for readability, shown in README)
		schemaJSON := `{
            "type": "object",
            "properties": {
                "operation": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"]},
                "a": {"type": "number"},
                "b": {"type": "number"}
            },
            "required": ["operation", "a", "b"]
        }`

		handler, err := mcpio.NewHandler(
			mcpio.WithName("schema-test-string"),
			mcpio.WithRawTool("calculator", "Arithmetic calculator", schemaJSON, simpleTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("MapFormat", func(t *testing.T) {
		// Option 2: map[string]any (for programmatic/dynamic schema construction)
		dynamicSchema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
				"repeat":  map[string]any{"type": "integer", "minimum": 1, "maximum": 10},
			},
			"required": []string{"message"},
		}

		handler, err := mcpio.NewHandler(
			mcpio.WithName("schema-test-map"),
			mcpio.WithRawTool("echo", "Echo with repetition", dynamicSchema, simpleTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})

	t.Run("RawMessageFormat", func(t *testing.T) {
		// Option 3: json.RawMessage (best performance)
		jRaw := json.RawMessage(`{
            "type": "object",
            "properties": {
                "data": {"type": "string"},
                "count": {"type": "integer", "minimum": 1}
            },
            "required": ["data"]
        }`)

		handler, err := mcpio.NewHandler(
			mcpio.WithName("schema-test-raw"),
			mcpio.WithRawTool("fast_processor", "High-performance processing", jRaw, simpleTool),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)
	})
}

func TestSDKComparison(t *testing.T) {
	t.Parallel()

	// Types from README SDK Comparison section
	type GreetInput struct {
		Name string `json:"name" jsonschema:"User's name"`
	}

	type GreetOutput struct {
		Greeting string `json:"greeting" jsonschema:"The greeting message"`
	}

	// mcp-io simplified signature (from README line 787-793)
	greet := func(ctx context.Context, toolCtx mcpio.RequestContext, input GreetInput) (GreetOutput, error) {
		// Session available via toolCtx, if needed
		_ = toolCtx.GetSession()
		return GreetOutput{Greeting: "Hello " + input.Name}, nil
	}

	t.Run("SimplifiedSignature", func(t *testing.T) {
		// Create handler with greet tool
		handler, err := mcpio.NewHandler(
			mcpio.WithName("greeter"),
			mcpio.WithTool("greet", "Greet user", greet),
		)
		require.NoError(t, err)
		assert.NotNil(t, handler)

		// Test the function directly
		mockToolCtx := testutil.NewMockRequestContext(nil)
		result, err := greet(t.Context(), mockToolCtx, GreetInput{Name: "World"})
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result.Greeting)
	})
}

func TestToolExecution(t *testing.T) {
	t.Parallel()
	t.Run("ToUpperTool", func(t *testing.T) {
		mockToolCtx := testutil.NewMockRequestContext(nil)
		result, err := toUpper(t.Context(), mockToolCtx, TextInput{Text: "hello world"})
		require.NoError(t, err)
		assert.Equal(t, "HELLO WORLD", result.Result)
	})

	t.Run("DivideTool", func(t *testing.T) {
		// Test precision rounding
		mockToolCtx := testutil.NewMockRequestContext(nil)
		result, err := divide(t.Context(), mockToolCtx, DivideInput{
			Numerator:   10.0,
			Denominator: 3.0,
			Precision:   2,
		})
		require.NoError(t, err)
		assert.InDelta(t, 3.33, result.Result, 0.001)

		// Test division by zero
		_, err = divide(t.Context(), mockToolCtx, DivideInput{
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

	t.Run("SDKComparison_ErrorHandling", func(t *testing.T) {
		// mcp-io returns errors instead of panicking
		_, err := mcpio.NewHandler(mcpio.WithName(""))
		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrEmptyValue)
		assert.Contains(t, err.Error(), "name cannot be empty")

		// Configuration errors provide meaningful context
		_, err = mcpio.NewHandler(
			mcpio.WithName("test"),
			mcpio.WithServer(nil),
		)
		require.Error(t, err)
		require.ErrorIs(t, err, mcpio.ErrNilValue)
		assert.Contains(t, err.Error(), "server cannot be nil")

		// Tool errors can be checked and provide context
		toolErr := mcpio.NewToolError("operation not supported")
		assert.Contains(t, toolErr.Error(), "operation not supported")

		// Validation errors are distinguishable
		validErr := mcpio.ValidationError("invalid input format")
		assert.NotEqual(t, toolErr.Error(), validErr.Error())
	})
}
