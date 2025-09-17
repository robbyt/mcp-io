package main

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// MockElicitationCapability for testing elicitation functionality
type MockElicitationCapability struct {
	Responses []*mcp.ElicitResult
	CallIndex int
}

func (m *MockElicitationCapability) Elicit(ctx context.Context, message string, requestedSchema *jsonschema.Schema) (*mcp.ElicitResult, error) {
	if m.CallIndex >= len(m.Responses) {
		return &mcp.ElicitResult{Action: "cancel"}, nil
	}
	result := m.Responses[m.CallIndex]
	m.CallIndex++
	return result, nil
}

// DatabaseTestSuite tests the database elicitation example
type DatabaseTestSuite struct {
	testutil.ExampleTestSuite
}

func (s *DatabaseTestSuite) SetupSuite() {
	// Get project root - we're in examples/cli_elicitation
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "cli_elicitation"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()
}

func (s *DatabaseTestSuite) SetupTest() {
	// Clear database before each test
	dbMutex.Lock()
	database = make(map[string]*Record)
	dbMutex.Unlock()
}

func TestDatabaseSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

// Test standard database operations (no elicitation)

func (s *DatabaseTestSuite) TestReadRecord() {
	ctx := s.T().Context()

	// Add a test record
	testRecord := &Record{
		ID:       "test1",
		Name:     "Test User",
		Email:    "test@example.com",
		Category: "personal",
		Created:  time.Now(),
		Updated:  time.Now(),
	}
	database["test1"] = testRecord

	s.Run("RecordExists", func() {
		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to retrieve"`
		}{ID: "test1"}

		result, err := readRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("found", result["status"])
		record := result["record"].(*Record)
		s.Equal("test1", record.ID)
		s.Equal("Test User", record.Name)
	})

	s.Run("RecordNotFound", func() {
		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to retrieve"`
		}{ID: "nonexistent"}

		result, err := readRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("not_found", result["status"])
		s.Equal("nonexistent", result["id"])
	})
}

func (s *DatabaseTestSuite) TestListRecords() {
	ctx := s.T().Context()

	// Add test records
	records := []*Record{
		{ID: "personal1", Name: "John", Category: "personal", Created: time.Now().Add(-2 * time.Hour)},
		{ID: "business1", Name: "Company A", Category: "business", Created: time.Now().Add(-1 * time.Hour)},
		{ID: "personal2", Name: "Jane", Category: "personal", Created: time.Now()},
	}
	for _, record := range records {
		record.Updated = record.Created
		database[record.ID] = record
	}

	s.Run("AllRecords", func() {
		input := struct {
			Category string `json:"category,omitempty" jsonschema:"description:Optional category filter,enum:,enum:personal,enum:business,enum:academic"`
		}{}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(3, result["count"])
		s.Empty(result["filter"])

		returnedRecords := result["records"].([]*Record)
		s.Len(returnedRecords, 3)
		// Should be sorted by creation time
		s.Equal("personal1", returnedRecords[0].ID)
		s.Equal("business1", returnedRecords[1].ID)
		s.Equal("personal2", returnedRecords[2].ID)
	})

	s.Run("FilteredByCategory", func() {
		input := struct {
			Category string `json:"category,omitempty" jsonschema:"description:Optional category filter,enum:,enum:personal,enum:business,enum:academic"`
		}{Category: "personal"}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(2, result["count"])
		s.Equal("personal", result["filter"])

		returnedRecords := result["records"].([]*Record)
		s.Len(returnedRecords, 2)
		s.Equal("personal1", returnedRecords[0].ID)
		s.Equal("personal2", returnedRecords[1].ID)
	})
}

// Test elicitation-based operations

func (s *DatabaseTestSuite) TestCreateRecord() {
	ctx := s.T().Context()

	s.Run("AcceptRecordData", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"id":       "user1",
						"name":     "John Doe",
						"email":    "john@example.com",
						"category": "personal",
					},
				},
			},
		}

		result, err := createRecord(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("created", result["status"])
		record := result["record"].(*Record)
		s.Equal("user1", record.ID)
		s.Equal("John Doe", record.Name)
		s.Equal("john@example.com", record.Email)
		s.Equal("personal", record.Category)

		// Verify record was added to database
		dbRecord, exists := database["user1"]
		s.True(exists)
		s.Equal("John Doe", dbRecord.Name)
	})

	s.Run("DeclineRecordData", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		result, err := createRecord(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "decline")
	})

	s.Run("InvalidIDWithSpaces", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"id":       "user 1", // Invalid ID with space
						"name":     "John Doe",
						"email":    "john@example.com",
						"category": "personal",
					},
				},
			},
		}

		result, err := createRecord(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("error", result["status"])
		s.Contains(result["reason"], "spaces")
	})

	s.Run("DuplicateID", func() {
		// Add existing record
		database["existing"] = &Record{ID: "existing", Name: "Existing User"}

		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"id":       "existing",
						"name":     "New User",
						"email":    "new@example.com",
						"category": "business",
					},
				},
			},
		}

		result, err := createRecord(ctx, mockCapability, struct{}{})
		s.Require().NoError(err)

		s.Equal("error", result["status"])
		s.Contains(result["reason"], "already exists")
	})
}

func (s *DatabaseTestSuite) TestUpdateRecord() {
	ctx := s.T().Context()

	s.Run("ConfirmUpdate", func() {
		// Setup test record for this specific test
		testRecord := &Record{
			ID:       "update1",
			Name:     "Original Name",
			Email:    "original@example.com",
			Category: "personal",
			Created:  time.Now().Add(-1 * time.Hour),
			Updated:  time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "UPDATE",
					},
				},
			},
		}

		input := struct {
			ID       string `json:"id"                 jsonschema:"description:Record ID to update"`
			Name     string `json:"name,omitempty"     jsonschema:"description:New name (optional)"`
			Email    string `json:"email,omitempty"    jsonschema:"format:email,description:New email (optional)"`
			Category string `json:"category,omitempty" jsonschema:"description:New category (optional),enum:,enum:personal,enum:business,enum:academic"`
		}{
			ID:    "update1",
			Name:  "Updated Name",
			Email: "updated@example.com",
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("updated", result["status"])
		record := result["record"].(*Record)
		s.Equal("Updated Name", record.Name)
		s.Equal("updated@example.com", record.Email)
		s.Equal("personal", record.Category) // Unchanged

		changes := result["changes"].([]string)
		s.Len(changes, 2)
		s.Contains(changes[0], "Original Name")
		s.Contains(changes[1], "updated@example.com")
	})

	s.Run("DeclineUpdate", func() {
		// Setup test record for this specific test
		testRecord := &Record{
			ID:       "update1",
			Name:     "Original Name",
			Email:    "original@example.com",
			Category: "personal",
			Created:  time.Now().Add(-1 * time.Hour),
			Updated:  time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord

		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		input := struct {
			ID       string `json:"id"                 jsonschema:"description:Record ID to update"`
			Name     string `json:"name,omitempty"     jsonschema:"description:New name (optional)"`
			Email    string `json:"email,omitempty"    jsonschema:"format:email,description:New email (optional)"`
			Category string `json:"category,omitempty" jsonschema:"description:New category (optional),enum:,enum:personal,enum:business,enum:academic"`
		}{
			ID:   "update1",
			Name: "Should Not Change",
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "decline")

		// Verify record wasn't changed
		record := result["record"].(*Record)
		s.Equal("Original Name", record.Name) // Should be unchanged
	})

	s.Run("InvalidConfirmation", func() {
		// Setup test record for this specific test
		testRecord := &Record{
			ID:       "update1",
			Name:     "Original Name",
			Email:    "original@example.com",
			Category: "personal",
			Created:  time.Now().Add(-1 * time.Hour),
			Updated:  time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord

		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "YES", // Wrong confirmation
					},
				},
			},
		}

		input := struct {
			ID       string `json:"id"                 jsonschema:"description:Record ID to update"`
			Name     string `json:"name,omitempty"     jsonschema:"description:New name (optional)"`
			Email    string `json:"email,omitempty"    jsonschema:"format:email,description:New email (optional)"`
			Category string `json:"category,omitempty" jsonschema:"description:New category (optional),enum:,enum:personal,enum:business,enum:academic"`
		}{
			ID:   "update1",
			Name: "Should Not Change",
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "Invalid confirmation")
		s.Contains(result["reason"], "Expected 'UPDATE', got 'YES'")
	})

	s.Run("NoChanges", func() {
		// Setup test record for this specific test
		testRecord := &Record{
			ID:       "update1",
			Name:     "Original Name",
			Email:    "original@example.com",
			Category: "personal",
			Created:  time.Now().Add(-1 * time.Hour),
			Updated:  time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord

		mockCapability := &MockElicitationCapability{}
		input := struct {
			ID       string `json:"id"                 jsonschema:"description:Record ID to update"`
			Name     string `json:"name,omitempty"     jsonschema:"description:New name (optional)"`
			Email    string `json:"email,omitempty"    jsonschema:"format:email,description:New email (optional)"`
			Category string `json:"category,omitempty" jsonschema:"description:New category (optional),enum:,enum:personal,enum:business,enum:academic"`
		}{
			ID: "update1",
			// No changes provided
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("no_changes", result["status"])
	})

	s.Run("RecordNotFound", func() {
		mockCapability := &MockElicitationCapability{}
		input := struct {
			ID       string `json:"id"                 jsonschema:"description:Record ID to update"`
			Name     string `json:"name,omitempty"     jsonschema:"description:New name (optional)"`
			Email    string `json:"email,omitempty"    jsonschema:"format:email,description:New email (optional)"`
			Category string `json:"category,omitempty" jsonschema:"description:New category (optional),enum:,enum:personal,enum:business,enum:academic"`
		}{
			ID:   "nonexistent",
			Name: "New Name",
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("not_found", result["status"])
		s.Equal("nonexistent", result["id"])
	})
}

func (s *DatabaseTestSuite) TestDeleteRecord() {
	ctx := s.T().Context()

	// Setup test record
	testRecord := &Record{
		ID:       "delete1",
		Name:     "To Be Deleted",
		Email:    "delete@example.com",
		Category: "business",
		Created:  time.Now(),
		Updated:  time.Now(),
	}
	database["delete1"] = testRecord

	s.Run("ConfirmDeletion", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "delete1", // Correct ID
					},
				},
			},
		}

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "delete1"}

		result, err := deleteRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("deleted", result["status"])
		deletedRecord := result["deleted_record"].(*Record)
		s.Equal("delete1", deletedRecord.ID)
		s.Equal("To Be Deleted", deletedRecord.Name)

		// Verify record was removed from database
		_, exists := database["delete1"]
		s.False(exists)
	})

	s.Run("DeclineDeletion", func() {
		// Re-add the record for this test
		database["delete1"] = testRecord

		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "delete1"}

		result, err := deleteRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "decline")

		// Verify record still exists
		_, exists := database["delete1"]
		s.True(exists)
	})

	s.Run("InvalidConfirmation", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "wrong_id", // Wrong ID
					},
				},
			},
		}

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "delete1"}

		result, err := deleteRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "Invalid confirmation")
		s.Contains(result["reason"], "Expected 'delete1', got 'wrong_id'")

		// Verify record still exists
		_, exists := database["delete1"]
		s.True(exists)
	})

	s.Run("RecordNotFound", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "accept", Content: map[string]any{"confirm": "nonexistent"}},
			},
		}

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "nonexistent"}

		result, err := deleteRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("not_found", result["status"])
		s.Equal("nonexistent", result["id"])
	})
}

func (s *DatabaseTestSuite) TestDatabaseReport() {
	ctx := s.T().Context()

	// Add some test data
	records := []*Record{
		{ID: "p1", Name: "Personal 1", Category: "personal", Created: time.Now().Add(-2 * time.Hour)},
		{ID: "b1", Name: "Business 1", Category: "business", Created: time.Now().Add(-1 * time.Hour)},
		{ID: "p2", Name: "Personal 2", Category: "personal", Created: time.Now()},
	}
	for _, record := range records {
		record.Updated = record.Created
		database[record.ID] = record
	}

	s.Run("AcceptReportPreferences", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"format":       "detailed",
						"category":     "personal",
						"sortBy":       "name",
						"includeStats": true,
					},
				},
			},
		}

		result, err := databaseReport(ctx, mockCapability, map[string]any{})
		s.Require().NoError(err)

		s.Contains(result.Description, "detailed")
		s.Contains(result.Description, "report")
		s.Require().Len(result.Messages, 2)

		systemMsg := result.Messages[0]
		s.Equal("system", systemMsg.Role)
		s.Contains(systemMsg.Content, "detailed")
		s.Contains(systemMsg.Content, "Total records: 3")
		s.Contains(systemMsg.Content, "personal")
		s.Contains(systemMsg.Content, "Sort results by: name")

		userMsg := result.Messages[1]
		s.Equal("user", userMsg.Role)
		s.Contains(userMsg.Content, "detailed database report")
		s.Contains(userMsg.Content, "personal records")
	})

	s.Run("DeclineReportPreferences", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		result, err := databaseReport(ctx, mockCapability, map[string]any{})
		s.Require().NoError(err)

		s.Contains(result.Description, "summary") // Default format
		s.Require().Len(result.Messages, 2)

		systemMsg := result.Messages[0]
		s.Contains(systemMsg.Content, "summary")                  // Default format
		s.Contains(systemMsg.Content, "Sort results by: created") // Default sort
	})
}

func (s *DatabaseTestSuite) TestServerCreation() {
	// Test that we can create the server with all our tools
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("database-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("read_record", "Get a record by ID", readRecord),
			mcpio.WithTool("list_records", "List all records with optional category filter", listRecords),
			mcpio.WithSessionTool("create_record", "Create a new record with elicited data", createRecord),
			mcpio.WithSessionTool("update_record", "Update a record with change confirmation", updateRecord),
			mcpio.WithSessionTool("delete_record", "Delete a record with confirmation", deleteRecord),
			mcpio.WithSessionPrompt("database_report", "Generate database reports with custom preferences", databaseReport),
		)
		if err != nil {
			return nil, err
		}
		return handler.GetServer(), nil
	}

	// Verify server creation works
	server, err := serverBuilder()
	s.Require().NoError(err)
	s.NotNil(server)
}

func (s *DatabaseTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}
