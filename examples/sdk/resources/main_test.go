package main

import (
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
