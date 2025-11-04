package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunElicitationExample(t *testing.T) {
	ctx := t.Context()

	output, err := runElicitationExample(ctx)
	require.NoError(t, err, "runElicitationExample should not return an error")

	// Verify the output contains the expected configuration values
	assert.Contains(t, output, "https://api.example.com", "output should contain server endpoint")
	assert.Contains(t, output, "3", "output should contain max retries")
	assert.Contains(t, output, "true", "output should contain logs enabled flag")

	// Verify the output format matches expected pattern
	assert.True(t, strings.HasPrefix(output, "Configuration received:"), "output should start with expected prefix")
}

// Example demonstrates the mcp-io elicitation functionality.
// This example shows how to use ElicitTyped to gather structured configuration
// data from the client with automatic schema generation.
func Example() {
	ctx := context.Background()

	output, err := runElicitationExample(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)
	// Output:
	// Server requests: Please provide your configuration settings
	// Configuration received: Endpoint: https://api.example.com, Max Retries: 3, Logs: true
}
