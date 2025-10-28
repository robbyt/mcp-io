package main

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/capabilities"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

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

	s.Run("EmptyDatabase", func() {
		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(0, result["count"])
	})

	s.Run("WithRecords", func() {
		// Add test records
		database["active1"] = &Record{ID: "active1", Status: "active", Created: time.Now()}
		database["inactive1"] = &Record{ID: "inactive1", Status: "inactive", Created: time.Now().Add(time.Second)}

		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(2, result["count"])
	})

	s.Run("FilterByStatus", func() {
		// Add test records
		database["active1"] = &Record{ID: "active1", Status: "active", Created: time.Now()}
		database["active2"] = &Record{ID: "active2", Status: "active", Created: time.Now().Add(time.Second)}
		database["inactive1"] = &Record{ID: "inactive1", Status: "inactive", Created: time.Now().Add(2 * time.Second)}

		input := struct {
			Status string `json:"status,omitempty" jsonschema:"description:Optional status filter,enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
		}{Status: "active"}

		result, err := listRecords(ctx, input)
		s.Require().NoError(err)

		s.Equal("success", result["status"])
		s.Equal(2, result["count"])
		s.Equal("active", result["filter"])

		returnedRecords := result["records"].([]*Record)
		s.Len(returnedRecords, 2)
		s.Equal("active1", returnedRecords[0].ID)
	})
}

// Test elicitation-based operations

func (s *DatabaseTestSuite) TestCreateRecord() {
	s.Run("AcceptRecordData", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"id":     "user1",
				"name":   "John Doe",
				"email":  "john@example.com",
				"status": "active",
				"age":    30,
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := createRecord(ctx, struct{}{})
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

		mockSession.AssertExpectations(s.T())
	})

	s.Run("DeclineRecordData", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "decline",
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := createRecord(ctx, struct{}{})
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "decline")

		mockSession.AssertExpectations(s.T())
	})

	s.Run("InvalidIDWithSpaces", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"id":     "user 1",
				"name":   "John Doe",
				"email":  "john@example.com",
				"status": "active",
				"age":    25,
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := createRecord(ctx, struct{}{})
		s.Require().NoError(err)

		s.Equal("error", result["status"])
		s.Contains(result["reason"], "spaces")

		mockSession.AssertExpectations(s.T())
	})

	s.Run("DuplicateID", func() {
		// Add existing record
		database["existing"] = &Record{ID: "existing", Name: "Existing User"}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"id":     "existing",
				"name":   "New User",
				"email":  "new@example.com",
				"status": "inactive",
				"age":    25,
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := createRecord(ctx, struct{}{})
		s.Require().NoError(err)

		s.Equal("error", result["status"])
		s.Contains(result["reason"], "already exists")

		mockSession.AssertExpectations(s.T())
	})
}

func (s *DatabaseTestSuite) TestUpdateRecord() {
	s.Run("UpdateName", func() {
		// Add existing record
		database["user1"] = &Record{
			ID:    "user1",
			Name:  "Old Name",
			Email: "old@example.com",
			Age:   25,
		}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "UPDATE",
			},
		}, nil).Once()

		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
		}{
			ID:   "user1",
			Name: "New Name",
		}

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := updateRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("updated", result["status"])
		record := result["record"].(*Record)
		s.Equal("New Name", record.Name)
		s.Equal("old@example.com", record.Email)

		mockSession.AssertExpectations(s.T())
	})

	s.Run("RecordNotFound", func() {
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

		ctx := s.T().Context()
		result, err := updateRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("not_found", result["status"])
	})

	s.Run("UserDeclinesUpdate", func() {
		// Add existing record
		database["user1"] = &Record{
			ID:    "user1",
			Name:  "Old Name",
			Email: "old@example.com",
		}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "decline",
		}, nil).Once()

		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
		}{
			ID:   "user1",
			Name: "New Name",
		}

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := updateRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])

		mockSession.AssertExpectations(s.T())
	})

	s.Run("InvalidConfirmation", func() {
		// Add existing record
		database["user1"] = &Record{
			ID:    "user1",
			Name:  "Old Name",
			Email: "old@example.com",
		}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "wrong",
			},
		}, nil).Once()

		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
		}{
			ID:   "user1",
			Name: "New Name",
		}

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := updateRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "Expected 'UPDATE'")

		mockSession.AssertExpectations(s.T())
	})

	s.Run("NoChanges", func() {
		// Add existing record
		database["user1"] = &Record{
			ID:    "user1",
			Name:  "Same Name",
			Email: "same@example.com",
		}

		input := struct {
			ID     string `json:"id"               jsonschema:"description:Record ID to update"`
			Name   string `json:"name,omitempty"   jsonschema:"description:New name (optional),minLength:1,maxLength:100"`
			Email  string `json:"email,omitempty"  jsonschema:"format:email,description:New email (optional),maxLength:255"`
			Status string `json:"status,omitempty" jsonschema:"description:New status (optional),enum:,enum:active,enum:inactive,enum:pending,enum:archived"`
			Age    *int   `json:"age,omitempty"    jsonschema:"description:New age (optional),minimum:18,maximum:120"`
		}{
			ID: "user1",
		}

		ctx := s.T().Context()
		result, err := updateRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("no_changes", result["status"])
	})
}

func (s *DatabaseTestSuite) TestDeleteRecord() {
	s.Run("DeleteExistingRecord", func() {
		// Add a record to delete
		database["delete1"] = &Record{
			ID:    "delete1",
			Name:  "To Delete",
			Email: "delete@example.com",
			Age:   30,
		}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "delete1",
			},
		}, nil).Once()

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "delete1"}

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := deleteRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("deleted", result["status"])
		deletedRecord := result["deleted_record"].(*Record)
		s.Equal("delete1", deletedRecord.ID)

		// Verify record was deleted from database
		_, exists := database["delete1"]
		s.False(exists)

		mockSession.AssertExpectations(s.T())
	})

	s.Run("RecordNotFound", func() {
		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "nonexistent"}

		ctx := s.T().Context()
		result, err := deleteRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("not_found", result["status"])
	})

	s.Run("UserDeclinesDelete", func() {
		// Add a record
		database["keep1"] = &Record{
			ID:   "keep1",
			Name: "Keep This",
		}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "decline",
		}, nil).Once()

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "keep1"}

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := deleteRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])

		// Verify record still exists
		_, exists := database["keep1"]
		s.True(exists)

		mockSession.AssertExpectations(s.T())
	})

	s.Run("InvalidConfirmation", func() {
		// Add a record
		database["keep2"] = &Record{
			ID:   "keep2",
			Name: "Keep This Too",
		}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"confirm": "wrong_id",
			},
		}, nil).Once()

		input := struct {
			ID string `json:"id" jsonschema:"description:Record ID to delete"`
		}{ID: "keep2"}

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := deleteRecord(ctx, input)
		s.Require().NoError(err)

		s.Equal("cancelled", result["status"])
		s.Contains(result["reason"], "Expected 'keep2'")

		// Verify record still exists
		_, exists := database["keep2"]
		s.True(exists)

		mockSession.AssertExpectations(s.T())
	})
}

func (s *DatabaseTestSuite) TestDatabaseReport() {
	s.Run("AcceptReportPreferences", func() {
		// Add some records
		database["rec1"] = &Record{ID: "rec1", Status: "active"}
		database["rec2"] = &Record{ID: "rec2", Status: "inactive"}

		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "accept",
			Content: map[string]any{
				"format":       "detailed",
				"status":       "active",
				"sortBy":       "created",
				"includeStats": true,
			},
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := databaseReport(ctx, map[string]any{})
		s.Require().NoError(err)

		s.NotNil(result)
		s.NotEmpty(result.Description)
		s.NotEmpty(result.Messages)
		s.Len(result.Messages, 2)

		mockSession.AssertExpectations(s.T())
	})

	s.Run("DeclineReportPreferences", func() {
		mockSession := testutil.NewMockSession()
		mockSession.SetupElicitation()
		mockSession.On("Elicit", mock.Anything, mock.Anything).Return(&mcp.ElicitResult{
			Action: "decline",
		}, nil).Once()

		ctx := capabilities.WithSession(s.T().Context(), mockSession.Session)
		result, err := databaseReport(ctx, map[string]any{})
		s.Require().NoError(err)

		// Should use default preferences
		s.NotNil(result)
		s.NotEmpty(result.Description)

		mockSession.AssertExpectations(s.T())
	})
}

func (s *DatabaseTestSuite) TestServerCreation() {
	// Test that we can create the server
	serverBuilder := func() (*mcp.Server, error) {
		handler, err := mcpio.NewHandler(
			mcpio.WithName("database-server"),
			mcpio.WithVersion("1.0.0"),
			mcpio.WithTool("read_record", "Get a record by ID", readRecord),
			mcpio.WithTool("list_records", "List all records with optional status filter", listRecords),
			mcpio.WithTool("create_record", "Create a new record with elicited data", createRecord),
			mcpio.WithTool("update_record", "Update a record with change confirmation", updateRecord),
			mcpio.WithTool("delete_record", "Delete a record with confirmation", deleteRecord),
			mcpio.WithPrompt("database_report", "Generate database reports with custom preferences", databaseReport),
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
