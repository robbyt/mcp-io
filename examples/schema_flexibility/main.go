package main

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"os"
	"strings"

	mcpio "github.com/robbyt/mcp-io"
)

// Traditional struct-based tool
type TextInput struct {
	Text string `json:"text" jsonschema:"Text to process"`
}

type TextOutput struct {
	Result string `json:"result" jsonschema:"Processed text"`
}

func toUpper(ctx context.Context, input TextInput) (TextOutput, error) {
	return TextOutput{Result: strings.ToUpper(input.Text)}, nil
}

// Generic tool function that works with any JSON - raw version for WithRawTool
func processJSON(ctx context.Context, input []byte) ([]byte, error) {
	var params map[string]any
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, err
	}

	// Add a "processed" flag to any input
	output := make(map[string]any)
	maps.Copy(output, params)
	output["processed"] = true
	return json.Marshal(output)
}

func main() {
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
	calculator := func(ctx context.Context, input []byte) ([]byte, error) {
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

	// Schema using map[string]any for dynamic construction
	dynamicSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Message to echo",
			},
			"repeat": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"maximum":     10,
				"default":     1,
				"description": "Number of times to repeat",
			},
		},
		"required": []string{"message"},
	}

	echo := func(ctx context.Context, input []byte) ([]byte, error) {
		var params map[string]any
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, err
		}

		message := params["message"].(string)
		repeat := 1
		if r, ok := params["repeat"]; ok {
			repeat = int(r.(float64))
		}

		var result []string
		for i := 0; i < repeat; i++ {
			result = append(result, message)
		}

		return json.Marshal(map[string]any{
			"echoed":  result,
			"count":   len(result),
			"message": message,
		})
	}

	handler, err := mcpio.NewHandler(
		mcpio.WithName("schema-flexibility-demo"),
		mcpio.WithVersion("1.0.0"),

		// Traditional struct-based tool (no schema options needed)
		mcpio.WithTool("to_upper", "Convert text to uppercase", toUpper),

		// Tool with JSON string schemas (optimal performance with json.RawMessage conversion)
		mcpio.WithRawTool("calculator", "Perform arithmetic operations", calculatorInputSchema, calculator),

		// Tool with dynamic map[string]any schema
		mcpio.WithRawTool("echo", "Echo a message with optional repetition", dynamicSchema, echo),

		// Tool with json.RawMessage schemas (maximum performance)
		mcpio.WithRawTool("process_json", "Add processed flag to any JSON object",
			json.RawMessage(`{"type":"object","additionalProperties":true}`), processJSON),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	log.Println("Schema Flexibility Demo Server")
	log.Println("Available tools:")
	log.Println("  - to_upper: WithTool with struct-based schema generation")
	log.Println("  - calculator: WithRawTool with JSON string schema (converted to json.RawMessage)")
	log.Println("  - echo: WithRawTool with map[string]any schema construction")
	log.Println("  - process_json: WithRawTool with json.RawMessage schema (maximum performance)")
	log.Println()

	if err := handler.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
