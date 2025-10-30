package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	mcpio "github.com/robbyt/mcp-io"
)

// Record represents a database record
type Record struct {
	ID      string    `json:"id"      jsonschema:"description:Unique identifier for the record"`
	Name    string    `json:"name"    jsonschema:"description:Display name"`
	Email   string    `json:"email"   jsonschema:"format:email,description:Email address"`
	Status  string    `json:"status"  jsonschema:"description:Record status,enum:active,enum:inactive,enum:pending,enum:archived"`
	Age     int       `json:"age"     jsonschema:"description:Age in years,minimum:18,maximum:120"`
	Created time.Time `json:"created" jsonschema:"description:Creation timestamp"`
	Updated time.Time `json:"updated" jsonschema:"description:Last update timestamp"`
}

// CreateRecordInput represents the data needed to create a new record
type CreateRecordInput struct {
	ID     string `json:"id"     jsonschema:"description:Unique identifier (no spaces),minLength:1,maxLength:50"`
	Name   string `json:"name"   jsonschema:"description:Display name,minLength:1,maxLength:100"`
	Email  string `json:"email"  jsonschema:"format:email,description:Email address,maxLength:255"`
	Status string `json:"status" jsonschema:"description:Record status,enum:active,enum:inactive,enum:pending,enum:archived"`
	Age    int    `json:"age"    jsonschema:"description:Age in years,minimum:18,maximum:120"`
}

// Global in-memory database with thread safety
var (
	database = make(map[string]*Record)
	dbMutex  sync.RWMutex
)

// Helper functions for confirmation validation

// validateConfirmation checks if the confirmation text matches the expected value
func validateConfirmation(result *mcpio.ElicitationResult, expectedValue string) (bool, string) {
	if !result.IsAccepted() {
		return false, fmt.Sprintf("User %s the operation", result.Action)
	}

	content := result.GetContent()
	if content == nil {
		return false, "No confirmation provided"
	}

	confirmation, ok := content["confirm"].(string)
	if !ok || confirmation != expectedValue {
		return false, fmt.Sprintf("Invalid confirmation. Expected '%s', got '%s'", expectedValue, confirmation)
	}

	return true, ""
}

// Standard database operations (no elicitation needed)

// readRecord retrieves a record by ID - standard tool operation
func readRecord(ctx context.Context, toolCtx mcpio.ToolContext, input struct {
	ID string `json:"id" jsonschema:"description:Record ID to retrieve"`
},
) (map[string]any, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	record, exists := database[input.ID]
	if !exists {
		return map[string]any{
			"status": "not_found",
			"id":     input.ID,
		}, nil
	}

	return map[string]any{
		"status": "found",
		"record": record,
	}, nil
}

// listRecords returns all records, optionally filtered by status
func listRecords(ctx context.Context, toolCtx mcpio.ToolContext, input struct {
	Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
},
) (map[string]any, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	var records []*Record
	for _, record := range database {
		if input.Status == "" || record.Status == input.Status {
			records = append(records, record)
		}
	}

	// Sort by creation time
	sort.Slice(records, func(i, j int) bool {
		return records[i].Created.Before(records[j].Created)
	})

	return map[string]any{
		"status":  "success",
		"count":   len(records),
		"filter":  input.Status,
		"records": records,
	}, nil
}

// Elicitation-based operations

// createRecord demonstrates elicitation for gathering structured data
func createRecord(ctx context.Context, toolCtx mcpio.ToolContext, _ struct{}) (map[string]any, error) {
	slog.Debug("createRecord starting", "operation", "elicitation")

	// Server pauses to elicit structured record data
	elicitor := mcpio.NewElicitor(toolCtx)
	result, err := mcpio.ElicitTyped[CreateRecordInput](ctx, elicitor, "To create a new database record, please provide the following information. This data will be stored in the local database and can be updated or deleted later:")
	if err != nil {
		slog.Error("createRecord elicitation failed", "error", err)
		return nil, fmt.Errorf("failed to elicit record data: %w", err)
	}

	slog.Debug("createRecord elicitation completed", "action", result.Action)

	if !result.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("User %s the record creation", result.Action),
		}, nil
	}

	// Decode the elicited data
	var recordData CreateRecordInput
	if err := result.DecodeContent(&recordData); err != nil {
		slog.Error("createRecord decode failed", "error", err)
		return nil, fmt.Errorf("failed to decode record data: %w", err)
	}

	slog.Debug("createRecord decoded data", "id", recordData.ID, "name", recordData.Name, "email", recordData.Email, "status", recordData.Status, "age", recordData.Age)

	// Validate ID format (no spaces)
	if strings.Contains(recordData.ID, " ") {
		return map[string]any{
			"status": "error",
			"reason": "Record ID cannot contain spaces",
		}, nil
	}

	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check if record already exists
	if _, exists := database[recordData.ID]; exists {
		return map[string]any{
			"status": "error",
			"reason": fmt.Sprintf("Record with ID '%s' already exists", recordData.ID),
		}, nil
	}

	// Create the record
	now := time.Now()
	record := &Record{
		ID:      recordData.ID,
		Name:    recordData.Name,
		Email:   recordData.Email,
		Status:  recordData.Status,
		Age:     recordData.Age,
		Created: now,
		Updated: now,
	}

	database[recordData.ID] = record

	slog.Info("createRecord success", "id", record.ID, "name", record.Name)

	return map[string]any{
		"status": "created",
		"record": record,
	}, nil
}

// updateRecord demonstrates elicitation for confirming destructive changes
func updateRecord(ctx context.Context, toolCtx mcpio.ToolContext, input struct {
	ID     string `json:"id"               jsonschema:"description:Record ID to update"`
	Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
	Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
	Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
	Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
},
) (map[string]any, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check if record exists
	record, exists := database[input.ID]
	if !exists {
		return map[string]any{
			"status": "not_found",
			"id":     input.ID,
		}, nil
	}

	// Build change summary
	var changes []string
	if input.Name != "" && input.Name != record.Name {
		changes = append(changes, fmt.Sprintf("Name: %s -> %s", record.Name, input.Name))
	}
	if input.Email != "" && input.Email != record.Email {
		changes = append(changes, fmt.Sprintf("Email: %s -> %s", record.Email, input.Email))
	}
	if input.Status != "" && input.Status != record.Status {
		changes = append(changes, fmt.Sprintf("Status: %s -> %s", record.Status, input.Status))
	}
	if input.Age != nil && *input.Age != record.Age {
		changes = append(changes, fmt.Sprintf("Age: %d -> %d", record.Age, *input.Age))
	}

	if len(changes) == 0 {
		return map[string]any{
			"status": "no_changes",
			"record": record,
		}, nil
	}

	// Server pauses to confirm the changes
	changesSummary := strings.Join(changes, "\n")
	confirmationMessage := fmt.Sprintf("Update record '%s'?\n\nChanges:\n%s\n\nThis will overwrite the existing data.", input.ID, changesSummary)

	elicitor := mcpio.NewElicitor(toolCtx)
	result, err := elicitor.ElicitSimple(ctx, confirmationMessage, "confirm", "Type 'UPDATE' to confirm these changes")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit confirmation: %w", err)
	}

	// Validate confirmation
	if valid, reason := validateConfirmation(result, "UPDATE"); !valid {
		return map[string]any{
			"status": "cancelled",
			"reason": reason,
			"record": record,
		}, nil
	}

	// Apply the changes
	if input.Name != "" {
		record.Name = input.Name
	}
	if input.Email != "" {
		record.Email = input.Email
	}
	if input.Status != "" {
		record.Status = input.Status
	}
	if input.Age != nil {
		record.Age = *input.Age
	}
	record.Updated = time.Now()

	return map[string]any{
		"status":  "updated",
		"record":  record,
		"changes": changes,
	}, nil
}

// deleteRecord demonstrates critical operation confirmation
func deleteRecord(ctx context.Context, toolCtx mcpio.ToolContext, input struct {
	ID string `json:"id" jsonschema:"description:Record ID to delete"`
},
) (map[string]any, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Check if record exists
	record, exists := database[input.ID]
	if !exists {
		return map[string]any{
			"status": "not_found",
			"id":     input.ID,
		}, nil
	}

	// Server pauses to confirm critical operation
	confirmationMessage := fmt.Sprintf("Delete record '%s'?\n\nRecord details:\n- Name: %s\n- Email: %s\n- Status: %s\n- Age: %d\n\nThis action cannot be undone.",
		record.ID, record.Name, record.Email, record.Status, record.Age)

	elicitor := mcpio.NewElicitor(toolCtx)
	result, err := elicitor.ElicitSimple(ctx, confirmationMessage, "confirm", fmt.Sprintf("Type '%s' to confirm deletion", record.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to elicit confirmation: %w", err)
	}

	// Validate confirmation
	if valid, reason := validateConfirmation(result, record.ID); !valid {
		return map[string]any{
			"status": "cancelled",
			"reason": reason,
			"record": record,
		}, nil
	}

	// Delete the record
	delete(database, input.ID)

	return map[string]any{
		"status":         "deleted",
		"deleted_record": record,
	}, nil
}

func main() {
	// Setup structured logging to stderr at debug level
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// Create an MCP server that demonstrates elicitation with a practical in-memory database.
	// This server provides both standard database operations and elicitation-enhanced operations.
	handler, err := mcpio.NewHandler(
		mcpio.WithName("database-server"),
		mcpio.WithVersion("1.0.0"),

		// Standard database operations (no elicitation)
		mcpio.WithTool("read_record", "Get a record by ID", readRecord),
		mcpio.WithTool("list_records", "List all records with optional status filter", listRecords),

		// Elicitation-enhanced operations
		// These tools pause execution to gather user input or confirmation
		mcpio.WithTool("create_record", "Create a new record with elicited data", createRecord),
		mcpio.WithTool("update_record", "Update a record with change confirmation", updateRecord),
		mcpio.WithTool("delete_record", "Delete a record with confirmation", deleteRecord),

		// Elicitation-enhanced prompt
	)
	if err != nil {
		slog.Error("Failed to create handler", "error", err)
		os.Exit(1)
	}

	// Run the server on stdio transport
	if err := handler.ServeStdio(context.Background(), nil, nil); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
