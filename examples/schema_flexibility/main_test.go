package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// SchemaFlexibilityTestSuite tests the schema_flexibility example
type SchemaFlexibilityTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *SchemaFlexibilityTestSuite) SetupSuite() {
	// Get project root - we're in examples/schema_flexibility
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "schema_flexibility"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func TestSchemaFlexibilitySuite(t *testing.T) {
	suite.Run(t, new(SchemaFlexibilityTestSuite))
}

// createCalculator creates a handler with only the calculator tool
func createCalculator() (*mcpio.Handler, error) {
	calculatorInputSchema := `{
		"type": "object",
		"properties": {
			"operation": {
				"type": "string",
				"enum": ["add", "subtract", "multiply", "divide"],
				"description": "Arithmetic operation to perform"
			},
			"a": {"type": "number", "description": "First number"},
			"b": {"type": "number", "description": "Second number"}
		},
		"required": ["operation", "a", "b"]
	}`

	calculator := func(ctx context.Context, toolCtx mcpio.ToolContext, input []byte) ([]byte, error) {
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
		default:
			return nil, mcpio.ValidationError("unsupported operation: " + op)
		}

		return json.Marshal(map[string]any{
			"result":    result,
			"operation": op,
		})
	}

	return mcpio.NewHandler(
		mcpio.WithName("schema-flexibility-demo"),
		mcpio.WithRawTool("calculator", "Perform arithmetic operations", calculatorInputSchema, calculator),
	)
}

// createProcessJson creates a handler with only the process_json tool
func createProcessJson() (*mcpio.Handler, error) {
	return mcpio.NewHandler(
		mcpio.WithName("schema-flexibility-demo"),
		mcpio.WithRawTool("process_json", "Add processed flag to any JSON object",
			json.RawMessage(`{"type":"object","additionalProperties":true}`), processJSON),
	)
}

// createSimple creates a handler with only the to_upper tool
func createSimple() (*mcpio.Handler, error) {
	return mcpio.NewHandler(
		mcpio.WithName("schema-flexibility-demo"),
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),
	)
}

// createFull creates a handler with all tools
func createFull() (*mcpio.Handler, error) {
	// JSON schema string for calculator input (json.RawMessage is optimal)
	calculatorInputSchema := `{
		"type": "object",
		"properties": {
			"operation": {
				"type": "string",
				"enum": ["add", "subtract", "multiply", "divide"],
				"description": "Arithmetic operation to perform"
			},
			"a": {"type": "number", "description": "First number"},
			"b": {"type": "number", "description": "Second number"}
		},
		"required": ["operation", "a", "b"]
	}`

	// Calculator function - raw version for WithRawTool
	calculator := func(ctx context.Context, toolCtx mcpio.ToolContext, input []byte) ([]byte, error) {
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
		default:
			return nil, mcpio.ValidationError("unsupported operation: " + op)
		}

		return json.Marshal(map[string]any{
			"result":    result,
			"operation": op,
		})
	}

	return mcpio.NewHandler(
		mcpio.WithName("schema-flexibility-demo"),
		mcpio.WithVersion("1.0.0"),

		// Traditional struct-based tool (no schema options needed)
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),

		// Tool with JSON string schema (converted to json.RawMessage)
		mcpio.WithRawTool("calculator", "Perform arithmetic operations", calculatorInputSchema, calculator),

		// Tool with json.RawMessage schema (maximum performance)
		mcpio.WithRawTool("process_json", "Add processed flag to any JSON object",
			json.RawMessage(`{"type":"object","additionalProperties":true}`), processJSON),
	)
}

func (s *SchemaFlexibilityTestSuite) TestToUpperTool() {
	ctx := s.T().Context()

	handler, err := createFull()
	s.Require().NoError(err)

	s.WithMCPSession(handler, func(session *mcp.ClientSession) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"BasicToUpper", "hello world", "HELLO WORLD"},
			{"EmptyString", "", ""},
			{"SpecialCharacters", "hello, world! 123", "HELLO, WORLD! 123"},
			{"MixedCase", "Hello WoRLD", "HELLO WORLD"},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name:      "to_upper",
					Arguments: map[string]any{"text": tc.input},
				})
				s.Require().NoError(err)
				s.Require().False(result.IsError)

				// Use StructuredContent which should have the typed result directly
				s.Require().NotNil(result.StructuredContent, "StructuredContent should be populated")

				resultMap, ok := result.StructuredContent.(map[string]any)
				s.Require().True(ok, "StructuredContent should be a map")
				s.Equal(tc.expected, resultMap["result"])
			})
		}
	})
}

func (s *SchemaFlexibilityTestSuite) TestCalculatorTool() {
	ctx := s.T().Context()

	handler, err := createCalculator()
	s.Require().NoError(err)

	s.WithMCPSession(handler, func(session *mcp.ClientSession) {
		testCases := []struct {
			name      string
			operation string
			a         float64
			b         float64
			expected  float64
		}{
			{"Addition", "add", 5, 3, 8},
			{"Subtraction", "subtract", 10, 4, 6},
			{"Multiplication", "multiply", 6, 7, 42},
			{"Division", "divide", 15, 3, 5},
			{"DecimalDivision", "divide", 10, 3, 3.3333333333333335},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: "calculator",
					Arguments: map[string]any{
						"operation": tc.operation,
						"a":         tc.a,
						"b":         tc.b,
					},
				})
				s.Require().NoError(err)
				s.Require().False(result.IsError)

				// Raw tools return JSON in TextContent
				textContent, ok := result.Content[0].(*mcp.TextContent)
				s.Require().True(ok, "Content should be TextContent")

				var resultMap map[string]any
				err = json.Unmarshal([]byte(textContent.Text), &resultMap)
				s.Require().NoError(err)

				s.InEpsilon(tc.expected, resultMap["result"], 0.0001)
				s.Equal(tc.operation, resultMap["operation"])
			})
		}

		// Test division by zero error
		s.Run("DivisionByZero", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "calculator",
				Arguments: map[string]any{
					"operation": "divide",
					"a":         10,
					"b":         0,
				},
			})
			s.Require().NoError(err)
			s.Require().True(result.IsError)
			if textContent, ok := result.Content[0].(*mcp.TextContent); s.True(ok) {
				s.Contains(textContent.Text, "division by zero")
			}
		})

		// Test invalid operation - tool returns validation error
		s.Run("InvalidOperation", func() {
			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "calculator",
				Arguments: map[string]any{
					"operation": "invalid",
					"a":         5,
					"b":         3,
				},
			})
			s.Require().NoError(err)
			s.Require().True(result.IsError)
			if textContent, ok := result.Content[0].(*mcp.TextContent); s.True(ok) {
				s.Contains(textContent.Text, "unsupported operation")
			}
		})
	})
}

func (s *SchemaFlexibilityTestSuite) TestProcessJsonTool() {
	ctx := s.T().Context()

	handler, err := createProcessJson()
	s.Require().NoError(err)

	s.WithMCPSession(handler, func(session *mcp.ClientSession) {
		testCases := []struct {
			name     string
			input    map[string]any
			expected map[string]any
		}{
			{
				"EmptyObject",
				map[string]any{},
				map[string]any{"processed": true},
			},
			{
				"SimpleObject",
				map[string]any{"name": "test", "value": 42.0},
				map[string]any{"name": "test", "value": 42.0, "processed": true},
			},
			{
				"ComplexObject",
				map[string]any{
					"user":  map[string]any{"id": 1.0, "name": "Alice"},
					"items": []any{"item1", "item2"},
				},
				map[string]any{
					"user":      map[string]any{"id": 1.0, "name": "Alice"},
					"items":     []any{"item1", "item2"},
					"processed": true,
				},
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name:      "process_json",
					Arguments: tc.input,
				})
				s.Require().NoError(err)
				s.Require().False(result.IsError)

				// Raw tools return JSON in TextContent
				textContent, ok := result.Content[0].(*mcp.TextContent)
				s.Require().True(ok, "Content should be TextContent")

				var resultMap map[string]any
				err = json.Unmarshal([]byte(textContent.Text), &resultMap)
				s.Require().NoError(err)

				for key, expectedValue := range tc.expected {
					s.Equal(expectedValue, resultMap[key], "Key %s should match", key)
				}
				s.True(resultMap["processed"].(bool), "processed flag should be true")
			})
		}
	})
}

func (s *SchemaFlexibilityTestSuite) TestToolListing() {
	ctx := s.T().Context()

	handler, err := createFull()
	s.Require().NoError(err)

	s.WithMCPSession(handler, func(session *mcp.ClientSession) {
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Tools, 3)

		toolMap := make(map[string]mcp.Tool)
		for _, tool := range result.Tools {
			toolMap[tool.Name] = *tool
		}

		// Verify each tool
		expectedTools := map[string]string{
			"to_upper":     "Convert text to uppercase",
			"calculator":   "Perform arithmetic operations",
			"process_json": "Add processed flag to any JSON object",
		}

		for name, description := range expectedTools {
			tool, exists := toolMap[name]
			s.Require().True(exists, "Tool %s should exist", name)
			s.Equal(description, tool.Description)
			s.NotNil(tool.InputSchema, "Tool %s should have input schema", name)
		}
	})
}

func (s *SchemaFlexibilityTestSuite) TestErrorHandling() {
	ctx := s.T().Context()

	handler, err := createSimple()
	s.Require().NoError(err)

	s.WithMCPSession(handler, func(session *mcp.ClientSession) {
		// Test calling non-existent tool
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "non_existent",
			Arguments: map[string]any{},
		})
		s.Require().Error(err)
		s.Contains(err.Error(), "unknown tool")
	})
}

func (s *SchemaFlexibilityTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
