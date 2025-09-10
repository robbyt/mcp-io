package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	mcpio "github.com/robbyt/go-mcpio"
)

// Example input/output types for the calculator tool
type CalculateInput struct {
	Operation string  `json:"operation" jsonschema:"Arithmetic operation (add, subtract, multiply, divide)"`
	A         float64 `json:"a"         jsonschema:"First number"`
	B         float64 `json:"b"         jsonschema:"Second number"`
}

type CalculateOutput struct {
	Result float64 `json:"result" jsonschema:"Calculation result"`
}

// Example input/output types for the echo tool
type EchoInput struct {
	Message string `json:"message" jsonschema:"Message to echo"`
}

type EchoOutput struct {
	Echo string `json:"echo" jsonschema:"Echoed message"`
}

// Calculator tool implementation
func calculator(ctx context.Context, input CalculateInput) (CalculateOutput, error) {
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

// Echo tool implementation
func echo(ctx context.Context, input EchoInput) (EchoOutput, error) {
	return EchoOutput{Echo: input.Message}, nil
}

func main() {
	// Create MCP handler with multiple tools
	handler, err := mcpio.New(
		mcpio.WithName("example-calculator"),
		mcpio.WithVersion("1.0.0"),
		mcpio.WithTool("calculate", "Perform arithmetic operations", calculator),
		mcpio.WithTool("echo", "Echo a message", echo),
	)
	if err != nil {
		log.Fatal("Failed to create MCP handler:", err)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	// Add a simple health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintln(w, "OK"); err != nil {
			// Log the error but don't fail the health check
			log.Printf("Failed to write health check response: %v", err)
		}
	})

	fmt.Println("MCP HTTP Server starting on :8080")
	fmt.Println("MCP endpoint: http://localhost:8080/mcp")
	fmt.Println("Health check: http://localhost:8080/health")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
