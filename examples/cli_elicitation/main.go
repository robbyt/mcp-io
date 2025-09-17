package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	mcpio "github.com/robbyt/mcp-io"
)

// Record represents a database record
type Record struct {
	ID       string    `json:"id"       jsonschema:"description:Unique identifier for the record"`
	Name     string    `json:"name"     jsonschema:"description:Display name"`
	Email    string    `json:"email"    jsonschema:"format:email,description:Email address"`
	Category string    `json:"category" jsonschema:"description:Record category,enum:personal,enum:business,enum:academic"`
	Created  time.Time `json:"created"  jsonschema:"description:Creation timestamp"`
	Updated  time.Time `json:"updated"  jsonschema:"description:Last update timestamp"`
}

// Global in-memory database with thread safety
var (
	database = make(map[string]*Record)
	dbMutex  sync.RWMutex
)

// Standard database operations (no elicitation needed)

// readRecord retrieves a record by ID - standard tool operation
func readRecord(ctx context.Context, input struct {
	ID string `json:"id" jsonschema:"description:Record ID to retrieve"`
},
) (map[string]any, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	fmt.Printf("[Server] Looking up record: %s\n", input.ID)

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

// listRecords returns all records, optionally filtered by category
func listRecords(ctx context.Context, input struct {
	Category string `json:"category,omitempty" jsonschema:"description:Optional category filter,enum:,enum:personal,enum:business,enum:academic"`
},
) (map[string]any, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	fmt.Printf("[Server] Listing records (category filter: %s)\n", input.Category)

	var records []*Record
	for _, record := range database {
		if input.Category == "" || record.Category == input.Category {
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
		"filter":  input.Category,
		"records": records,
	}, nil
}

// Elicitation-based operations

// createRecord demonstrates elicitation for gathering structured data
func createRecord(ctx context.Context, capability mcpio.ElicitationCapability, input struct{}) (map[string]any, error) {
	fmt.Println("[Server] Creating new record - need to gather record data")
	fmt.Println("[Server] Pausing to elicit record details from user")

	// Server pauses to elicit structured record data
	result, err := mcpio.ElicitTypedResult[struct {
		ID       string `json:"id"       jsonschema:"description:Unique identifier (no spaces)"`
		Name     string `json:"name"     jsonschema:"description:Display name"`
		Email    string `json:"email"    jsonschema:"format:email,description:Email address"`
		Category string `json:"category" jsonschema:"description:Record category,enum:personal,enum:business,enum:academic"`
	}](ctx, capability, "Please provide the details for the new record:")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit record data: %w", err)
	}

	fmt.Printf("[Server] Received record data: action=%s\n", result.Action)

	if !result.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("User %s the record creation", result.Action),
		}, nil
	}

	// Decode the elicited data
	var recordData struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Category string `json:"category"`
	}
	if err := result.DecodeContent(&recordData); err != nil {
		return nil, fmt.Errorf("failed to decode record data: %w", err)
	}

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
		ID:       recordData.ID,
		Name:     recordData.Name,
		Email:    recordData.Email,
		Category: recordData.Category,
		Created:  now,
		Updated:  now,
	}

	database[recordData.ID] = record
	fmt.Printf("[Server] Created record: %s (%s)\n", record.ID, record.Name)

	return map[string]any{
		"status": "created",
		"record": record,
	}, nil
}

// updateRecord demonstrates elicitation for confirming destructive changes
func updateRecord(ctx context.Context, capability mcpio.ElicitationCapability, input struct {
	ID       string `json:"id"                 jsonschema:"description:Record ID to update"`
	Name     string `json:"name,omitempty"     jsonschema:"description:New name (optional)"`
	Email    string `json:"email,omitempty"    jsonschema:"format:email,description:New email (optional)"`
	Category string `json:"category,omitempty" jsonschema:"description:New category (optional),enum:,enum:personal,enum:business,enum:academic"`
},
) (map[string]any, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	fmt.Printf("[Server] Updating record: %s\n", input.ID)

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
		changes = append(changes, fmt.Sprintf("Name: %s → %s", record.Name, input.Name))
	}
	if input.Email != "" && input.Email != record.Email {
		changes = append(changes, fmt.Sprintf("Email: %s → %s", record.Email, input.Email))
	}
	if input.Category != "" && input.Category != record.Category {
		changes = append(changes, fmt.Sprintf("Category: %s → %s", record.Category, input.Category))
	}

	if len(changes) == 0 {
		return map[string]any{
			"status": "no_changes",
			"record": record,
		}, nil
	}

	fmt.Println("[Server] Changes detected - pausing to confirm with user")

	// Server pauses to confirm the changes
	changesSummary := strings.Join(changes, "\n")
	confirmationMessage := fmt.Sprintf("Update record '%s'?\n\nChanges:\n%s\n\nThis will overwrite the existing data.", input.ID, changesSummary)

	result, err := mcpio.ElicitSimple(ctx, capability, confirmationMessage, "confirm", "Type 'UPDATE' to confirm these changes")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit confirmation: %w", err)
	}

	fmt.Printf("[Server] Received update confirmation: action=%s\n", result.Action)

	if !result.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("User %s the update", result.Action),
			"record": record,
		}, nil
	}

	// Check confirmation text
	content := result.GetContent()
	if content == nil {
		return map[string]any{
			"status": "cancelled",
			"reason": "No confirmation provided",
			"record": record,
		}, nil
	}

	confirmation, ok := content["confirm"].(string)
	if !ok || confirmation != "UPDATE" {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("Invalid confirmation. Expected 'UPDATE', got '%s'", confirmation),
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
	if input.Category != "" {
		record.Category = input.Category
	}
	record.Updated = time.Now()

	fmt.Printf("[Server] Updated record: %s\n", record.ID)

	return map[string]any{
		"status":  "updated",
		"record":  record,
		"changes": changes,
	}, nil
}

// deleteRecord demonstrates critical operation confirmation
func deleteRecord(ctx context.Context, capability mcpio.ElicitationCapability, input struct {
	ID string `json:"id" jsonschema:"description:Record ID to delete"`
},
) (map[string]any, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	fmt.Printf("[Server] Request to delete record: %s\n", input.ID)

	// Check if record exists
	record, exists := database[input.ID]
	if !exists {
		return map[string]any{
			"status": "not_found",
			"id":     input.ID,
		}, nil
	}

	fmt.Println("[Server] Record found - pausing to confirm deletion")

	// Server pauses to confirm critical operation
	confirmationMessage := fmt.Sprintf("Delete record '%s'?\n\nRecord details:\n- Name: %s\n- Email: %s\n- Category: %s\n\nThis action cannot be undone.",
		record.ID, record.Name, record.Email, record.Category)

	result, err := mcpio.ElicitSimple(ctx, capability, confirmationMessage, "confirm", fmt.Sprintf("Type '%s' to confirm deletion", record.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to elicit confirmation: %w", err)
	}

	fmt.Printf("[Server] Received deletion confirmation: action=%s\n", result.Action)

	if !result.IsAccepted() {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("User %s the deletion", result.Action),
			"record": record,
		}, nil
	}

	// Check confirmation text matches record ID
	content := result.GetContent()
	if content == nil {
		return map[string]any{
			"status": "cancelled",
			"reason": "No confirmation provided",
			"record": record,
		}, nil
	}

	confirmation, ok := content["confirm"].(string)
	if !ok || confirmation != record.ID {
		return map[string]any{
			"status": "cancelled",
			"reason": fmt.Sprintf("Invalid confirmation. Expected '%s', got '%s'", record.ID, confirmation),
			"record": record,
		}, nil
	}

	// Delete the record
	delete(database, input.ID)
	fmt.Printf("[Server] Deleted record: %s\n", input.ID)

	return map[string]any{
		"status":         "deleted",
		"deleted_record": record,
	}, nil
}

// databaseReport demonstrates elicitation within prompts
func databaseReport(ctx context.Context, capability mcpio.ElicitationCapability, args map[string]any) (*mcpio.PromptResult, error) {
	fmt.Println("[Server] Generating database report prompt...")
	fmt.Println("[Server] Need report preferences - pausing to elicit from user")

	// Server pauses to elicit report preferences
	result, err := mcpio.ElicitTypedResult[struct {
		Format       string `json:"format"             jsonschema:"description:Report format,enum:summary,enum:detailed,enum:analysis"`
		Category     string `json:"category,omitempty" jsonschema:"description:Filter by category (optional),enum:,enum:personal,enum:business,enum:academic"`
		SortBy       string `json:"sortBy"             jsonschema:"description:Sort order,enum:name,enum:created,enum:updated,enum:category"`
		IncludeStats bool   `json:"includeStats"       jsonschema:"description:Include database statistics"`
	}](ctx, capability, "Configure the database report:")
	if err != nil {
		return nil, fmt.Errorf("failed to elicit report preferences: %w", err)
	}

	fmt.Printf("[Server] Received report preferences: action=%s\n", result.Action)

	// Default preferences if user declines
	reportConfig := struct {
		Format       string `json:"format"`
		Category     string `json:"category,omitempty"`
		SortBy       string `json:"sortBy"`
		IncludeStats bool   `json:"includeStats"`
	}{
		Format:       "summary",
		Category:     "",
		SortBy:       "created",
		IncludeStats: true,
	}

	if result.IsAccepted() {
		if err := result.DecodeContent(&reportConfig); err != nil {
			fmt.Printf("[Server] Failed to decode preferences, using defaults: %v\n", err)
		} else {
			fmt.Printf("[Server] Using custom report preferences\n")
		}
	} else {
		fmt.Printf("[Server] User %s preferences, using defaults\n", result.Action)
	}

	// Generate database snapshot
	dbMutex.RLock()
	totalRecords := len(database)
	categories := make(map[string]int)
	for _, record := range database {
		categories[record.Category]++
	}
	dbMutex.RUnlock()

	// Build system message based on preferences
	var systemMessage strings.Builder
	systemMessage.WriteString("You are a database analyst. Generate a ")
	systemMessage.WriteString(reportConfig.Format)
	systemMessage.WriteString(" report for an in-memory database.\n\n")

	systemMessage.WriteString("Database Overview:\n")
	systemMessage.WriteString(fmt.Sprintf("- Total records: %d\n", totalRecords))

	if reportConfig.IncludeStats {
		systemMessage.WriteString("- Category breakdown:\n")
		for cat, count := range categories {
			if cat == "" {
				cat = "uncategorized"
			}
			systemMessage.WriteString(fmt.Sprintf("  - %s: %d\n", cat, count))
		}
	}

	if reportConfig.Category != "" {
		systemMessage.WriteString(fmt.Sprintf("\nFocus on '%s' category records only.\n", reportConfig.Category))
	}

	systemMessage.WriteString(fmt.Sprintf("\nSort results by: %s\n", reportConfig.SortBy))
	systemMessage.WriteString("\nProvide insights and recommendations based on the data patterns.")

	userMessage := fmt.Sprintf("Create a %s database report", reportConfig.Format)
	if reportConfig.Category != "" {
		userMessage += fmt.Sprintf(" for %s records", reportConfig.Category)
	}

	return &mcpio.PromptResult{
		Description: fmt.Sprintf("Database %s report with custom preferences", reportConfig.Format),
		Messages: []mcpio.PromptMessage{
			{Role: "system", Content: systemMessage.String()},
			{Role: "user", Content: userMessage},
		},
	}, nil
}

func main() {
	// Create an MCP server that demonstrates elicitation with a practical in-memory database.
	// This server provides both standard database operations and elicitation-enhanced operations.
	handler, err := mcpio.NewHandler(
		mcpio.WithName("database-server"),
		mcpio.WithVersion("1.0.0"),

		// Standard database operations (no elicitation)
		mcpio.WithTool("read_record", "Get a record by ID", readRecord),
		mcpio.WithTool("list_records", "List all records with optional category filter", listRecords),

		// Elicitation-enhanced operations
		// These tools pause execution to gather user input or confirmation
		mcpio.WithSessionTool("create_record", "Create a new record with elicited data", createRecord),
		mcpio.WithSessionTool("update_record", "Update a record with change confirmation", updateRecord),
		mcpio.WithSessionTool("delete_record", "Delete a record with confirmation", deleteRecord),

		// Elicitation-enhanced prompt
		mcpio.WithSessionPrompt("database_report", "Generate database reports with custom preferences", databaseReport),
	)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	fmt.Println("[Server] Starting MCP database server with elicitation capabilities...")
	fmt.Println("[Server] Standard operations: read_record, list_records")
	fmt.Println("[Server] Elicitation operations: create_record, update_record, delete_record")
	fmt.Println("[Server] Interactive prompts: database_report")
	fmt.Println("[Server] Connect with Claude Desktop to interact with the database")

	// Run the server on stdio transport
	if err := handler.ServeStdio(context.Background(), nil, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
