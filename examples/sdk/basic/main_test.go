package main

import (
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
