package main

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func (m *MockElicitationCapability) Elicit(ctx context.Context, message string, requestedSchema any) (*mcp.ElicitResult, error) {
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
		ID:      "test1",
		Name:    "Test User",
		Email:   "test@example.com",
		Status:  "active",
		Age:     25,
		Created: time.Now(),
		Updated: time.Now(),
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
		{ID: "active1", Name: "John", Status: "active", Age: 30, Created: time.Now().Add(-2 * time.Hour)},
		{ID: "inactive1", Name: "Company A", Status: "inactive", Age: 45, Created: time.Now().Add(-1 * time.Hour)},
		{ID: "pending1", Name: "Jane", Status: "pending", Age: 28, Created: time.Now()},
	}
	for _, record := range records {
		record.Updated = record.Created
		database[record.ID] = record
	}

	s.Run("AllRecords", func() {
		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(3, result["count"])
		s.Empty(result["filter"])

		returnedRecords := result["records"].([]*Record)
		s.Len(returnedRecords, 3)
		// Should be sorted by creation time
		s.Equal("active1", returnedRecords[0].ID)
		s.Equal("inactive1", returnedRecords[1].ID)
		s.Equal("pending1", returnedRecords[2].ID)
	})

	s.Run("FilteredByStatus", func() {
		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{Status: "active"}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(1, result["count"])
		s.Equal("active", result["filter"])

		returnedRecords := result["records"].([]*Record)
		s.Len(returnedRecords, 1)
		s.Equal("active1", returnedRecords[0].ID)
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
						"id":     "user1",
						"name":   "John Doe",
						"email":  "john@example.com",
						"status": "active",
						"age":    30,
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
		s.Equal("active", record.Status)
		s.Equal(30, record.Age)

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
						"id":     "user 1",
						"name":   "John Doe",
						"email":  "john@example.com",
						"status": "active",
						"age":    25,
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
						"id":     "existing",
						"name":   "New User",
						"email":  "new@example.com",
						"status": "inactive",
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
			ID:      "update1",
			Name:    "Original Name",
			Email:   "original@example.com",
			Status:  "active",
			Age:     25,
			Created: time.Now().Add(-1 * time.Hour),
			Updated: time.Now().Add(-1 * time.Hour),
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
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
		}{
			ID:     "update1",
			Name:   "Updated Name",
			Email:  "updated@example.com",
			Status: "pending",
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("updated", result["status"])
		record := result["record"].(*Record)
		s.Equal("Updated Name", record.Name)
		s.Equal("updated@example.com", record.Email)
		s.Equal("pending", record.Status)

		changes := result["changes"].([]string)
		s.Len(changes, 3)
		s.Contains(changes[0], "Original Name")
		s.Contains(changes[1], "updated@example.com")
		s.Contains(changes[2], "pending")
	})

	s.Run("DeclineUpdate", func() {
		// Setup test record for this specific test
		testRecord := &Record{
			ID:      "update1",
			Name:    "Original Name",
			Email:   "original@example.com",
			Status:  "active",
			Age:     25,
			Created: time.Now().Add(-1 * time.Hour),
			Updated: time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord

		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
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
		s.Equal("Original Name", record.Name)
	})

	s.Run("InvalidConfirmation", func() {
		// Setup test record for this specific test
		testRecord := &Record{
			ID:      "update1",
			Name:    "Original Name",
			Email:   "original@example.com",
			Status:  "active",
			Age:     25,
			Created: time.Now().Add(-1 * time.Hour),
			Updated: time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord

		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "YES",
					},
				},
			},
		}

		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
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
			ID:      "update1",
			Name:    "Original Name",
			Email:   "original@example.com",
			Status:  "active",
			Age:     25,
			Created: time.Now().Add(-1 * time.Hour),
			Updated: time.Now().Add(-1 * time.Hour),
		}
		database["update1"] = testRecord

		mockCapability := &MockElicitationCapability{}
		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
		}{
			ID: "update1",
		}

		result, err := updateRecord(ctx, mockCapability, input)
		s.Require().NoError(err)

		s.Equal("no_changes", result["status"])
	})

	s.Run("RecordNotFound", func() {
		mockCapability := &MockElicitationCapability{}
		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
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
		ID:      "delete1",
		Name:    "To Be Deleted",
		Email:   "delete@example.com",
		Status:  "inactive",
		Age:     40,
		Created: time.Now(),
		Updated: time.Now(),
	}
	database["delete1"] = testRecord

	s.Run("ConfirmDeletion", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{
					Action: "accept",
					Content: map[string]any{
						"confirm": "delete1",
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
						"confirm": "wrong_id",
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
		{ID: "a1", Name: "Active 1", Status: "active", Age: 25, Created: time.Now().Add(-2 * time.Hour)},
		{ID: "i1", Name: "Inactive 1", Status: "inactive", Age: 35, Created: time.Now().Add(-1 * time.Hour)},
		{ID: "p1", Name: "Pending 1", Status: "pending", Age: 45, Created: time.Now()},
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
						"status":       "active",
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
		s.Contains(systemMsg.Content, "active")
		s.Contains(systemMsg.Content, "Sort results by: name")

		userMsg := result.Messages[1]
		s.Equal("user", userMsg.Role)
		s.Contains(userMsg.Content, "detailed database report")
		s.Contains(userMsg.Content, "active records")
	})

	s.Run("DeclineReportPreferences", func() {
		mockCapability := &MockElicitationCapability{
			Responses: []*mcp.ElicitResult{
				{Action: "decline"},
			},
		}

		result, err := databaseReport(ctx, mockCapability, map[string]any{})
		s.Require().NoError(err)

		s.Contains(result.Description, "summary")
		s.Require().Len(result.Messages, 2)

		systemMsg := result.Messages[0]
		s.Contains(systemMsg.Content, "summary")
		s.Contains(systemMsg.Content, "Sort results by: created")
	})
}

func (s *DatabaseTestSuite) TestServerCreation() {
	// Test that we can create the server with all our tools
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("database-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("read_record", "Get a record by ID", readRecord),
			mcpio.WithTool("list_records", "List all records with optional status filter", listRecords),
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
