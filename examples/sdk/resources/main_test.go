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

func TestRunResources(t *testing.T) {
	ctx := t.Context()

	output, err := runResources(ctx)
	require.NoError(t, err, "runResources should not return an error")

	// Verify the output contains the expected resource
	assert.Contains(t, output, "file:///a", "output should contain static resource")

	// Verify the output contains the expected template
	assert.Contains(t, output, "file:///dir/{f}", "output should contain resource template")

	// Verify successful reads
	assert.Contains(t, output, "a", "output should contain content from file:///a")
	assert.Contains(t, output, "x", "output should contain content from file:///dir/x")

	// Verify error handling for non-existent resource
	assert.True(t, strings.Contains(output, "error") || strings.Contains(output, "not found"),
		"output should contain error for non-existent resource")
}

// Example demonstrates the mcp-io resource functionality.
// This example shows how to register both static resources and dynamic
// resource templates, and how to handle resource read requests.
func Example() {
	ctx := context.Background()

	output, err := runResources(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)
	// Output: Resources: [file:///a]
	// Templates: [file:///dir/{f}]
	// Contents: [a x error: calling "resources/read": Resource not found]
}
