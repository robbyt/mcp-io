package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunGreeter(t *testing.T) {
	ctx := t.Context()

	output, err := runGreeter(ctx)
	require.NoError(t, err, "runGreeter should not return an error")
	assert.JSONEq(t, `{"greeting":"Hi user"}`, output, "greeting output should match expected JSON")
}

// Example demonstrates the basic mcp-io tool functionality.
// This example shows how to create a simple MCP server with a greeting tool.
func Example() {
	ctx := context.Background()

	output, err := runGreeter(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)
	// Output: {"greeting":"Hi user"}
}
