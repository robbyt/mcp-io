package main

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/crypt"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/dice"
	"github.com/robbyt/mcp-io/examples/simple_dungeon_master/narrative"
	"github.com/robbyt/mcp-io/internal/mcpwrapper"
	"github.com/robbyt/mcp-io/internal/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// DungeonMasterTestSuite tests the dungeon_master example
type DungeonMasterTestSuite struct {
	testutil.ExampleTestSuite
	mockLLM    *MockLLM
	session    *mcp.ClientSession
	cancelFunc context.CancelFunc
}

// MockLLM is a mock LLM for testing
type MockLLM struct {
	mock.Mock
}

// CreateMessage mocks the LLM message creation
func (m *MockLLM) CreateMessage(ctx context.Context, toolCtx mcpio.ToolContext, prompt string, maxTokens int, preferredModel string) (string, error) {
	args := m.Called(ctx, toolCtx, prompt, maxTokens, preferredModel)
	return args.String(0), args.Error(1)
}

func (s *DungeonMasterTestSuite) SetupSuite() {
	// Get project root - we're in examples/simple_dungeon_master
	_, b, _, _ := runtime.Caller(0)
	exampleDir := filepath.Dir(b)
	s.ProjectRoot = filepath.Join(exampleDir, "..", "..")
	s.ExampleName = "simple_dungeon_master"

	// Call parent SetupSuite
	s.ExampleTestSuite.SetupSuite()

	// Setup shared mock LLM
	s.mockLLM = new(MockLLM)
	s.mockLLM.On("CreateMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test narrative", nil)
}

func (s *DungeonMasterTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(s.T().Context())
	s.cancelFunc = cancel

	// Build server
	cryptState, err := crypt.NewState()
	s.Require().NoError(err)

	gameState := &GameState{
		narrative: narrative.NewState(),
		dice:      dice.NewState(&dice.Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 1.0}),
		crypt:     cryptState,
		llmFunc:   s.mockLLM.CreateMessage,
		config: &Config{
			summarizationThreshold: 10,
			summarizeMaxTokens:     100,
			narrativeMaxTokens:     100,
		},
	}
	handler, err := buildHandler(gameState)
	s.Require().NoError(err)

	// Create in-memory server with transport
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	s.Require().True(ok, "failed to unwrap SDK server")
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Start server in background
	go func() {
		if err := wrappedServer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.T().Errorf("server run error: %v", err)
		}
	}()

	// Create client and session
	testImpl := &mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
	client := mcp.NewClient(testImpl, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	s.Require().NoError(err)
	s.session = session
}

func (s *DungeonMasterTestSuite) TearDownTest() {
	if s.session != nil {
		if err := s.session.Close(); err != nil && !errors.Is(err, context.Canceled) {
			s.T().Errorf("session close error: %v", err)
		}
		s.session = nil
	}
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}
}

func TestDungeonMasterSuite(t *testing.T) {
	suite.Run(t, new(DungeonMasterTestSuite))
}

// Helper to trigger skill check
func (s *DungeonMasterTestSuite) triggerSkillCheck(ctx context.Context, action string) {
	result, err := s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": action},
	})
	s.NotNil(result)
	s.Require().NoError(err)
	s.Require().Len(result.Content, 1)

	content := result.Content[0].(*mcp.TextContent)
	var resp narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &resp))
	s.Require().Positive(resp.SkillCheckRequired)
}

func (s *DungeonMasterTestSuite) TestToolListing() {
	ctx := s.T().Context()

	cryptState, err := crypt.NewState()
	s.Require().NoError(err)

	gameState := &GameState{
		narrative: narrative.NewState(),
		dice:      dice.NewState(&dice.Config{GracePeriodMin: 0, GracePeriodMax: 0, SkillCheckFrequency: 1.0}),
		crypt:     cryptState,
		config: &Config{
			summarizationThreshold: 4,
			summarizeMaxTokens:     100,
			narrativeMaxTokens:     100,
		},
	}
	handler, err := buildHandler(gameState)
	s.Require().NoError(err)

	s.WithMCPSession(handler, func(session *mcp.ClientSession) {
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		s.Require().NoError(err)
		s.Require().Len(result.Tools, 3)

		// Tools are returned in alphabetical order
		// Check for debug_gameState tool
		debugTool := result.Tools[0]
		s.Equal("debug_gameState", debugTool.Name)
		s.NotNil(debugTool.InputSchema, "Tool should have input schema")

		// Check for dungeon_master tool
		dungeonMasterTool := result.Tools[1]
		s.Equal("dungeon_master", dungeonMasterTool.Name)
		s.NotNil(dungeonMasterTool.InputSchema, "Tool should have input schema")

		// Check for roll_d20 tool
		diceTool := result.Tools[2]
		s.Equal("roll_d20", diceTool.Name)
		s.NotNil(diceTool.InputSchema, "Tool should have input schema")
	})
}

func (s *DungeonMasterTestSuite) TestBinaryBuild() {
	binaryPath := s.BuildBinary()

	// Verify binary was created
	s.FileExists(binaryPath)
	s.T().Log("Binary built successfully at", binaryPath)
}

func (s *DungeonMasterTestSuite) TestFakeEncryptedRollRejection() {
	ctx := s.T().Context()

	s.triggerSkillCheck(ctx, "test action")

	// Roll dice to get a valid encrypted result
	res, err := s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "roll_d20",
		Arguments: map[string]any{},
	})
	s.NotNil(res)
	s.Require().NoError(err)

	// Tamper with the encrypted roll
	fakeEncryptedRoll := "fake_encrypted_data"

	// Attempt to submit tampered encrypted roll (should fail - anti-cheat)
	result, err := s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "second action", "encryptedData": fakeEncryptedRoll},
	})
	s.NotNil(result)
	s.Require().NoError(err)
	s.Require().True(result.IsError)
	content := result.Content[0].(*mcp.TextContent)
	s.NotEmpty(content.Text)

	s.mockLLM.AssertExpectations(s.T())
}

func (s *DungeonMasterTestSuite) TestMissingEncryptedRollRejection() {
	ctx := s.T().Context()

	s.triggerSkillCheck(ctx, "test action")

	// Attempt to submit action without rolling (should fail - anti-cheat)
	result, err := s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "second action"},
	})
	s.NotNil(result)
	s.Require().NoError(err)

	content := result.Content[0].(*mcp.TextContent)
	var resp narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &resp))
	s.NotEmpty(resp.ErrorMessage)
	s.Contains(resp.ErrorMessage, "Skill check required")
	s.Contains(resp.ErrorMessage, "roll_d20")

	s.mockLLM.AssertExpectations(s.T())
}

func (s *DungeonMasterTestSuite) TestValidEncryptedRollFlow() {
	ctx := s.T().Context()

	s.triggerSkillCheck(ctx, "pet the cat")

	// Roll dice
	result, err := s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "roll_d20",
		Arguments: map[string]any{},
	})
	s.NotNil(result)
	s.Require().NoError(err)

	content := result.Content[0].(*mcp.TextContent)
	var roll dice.Roll
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &roll))
	s.Require().NotEmpty(roll.EncryptedData)
	s.Require().GreaterOrEqual(roll.Result, 1)
	s.Require().LessOrEqual(roll.Result, 20)

	// Submit second action with encrypted roll (should succeed)
	result, err = s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "continue petting the cat", "encryptedData": roll.EncryptedData},
	})
	s.NotNil(result)
	s.Require().NoError(err)

	content = result.Content[0].(*mcp.TextContent)
	var resp narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &resp))
	s.Empty(resp.ErrorMessage)
	s.NotEmpty(resp.Narrative)

	s.mockLLM.AssertExpectations(s.T())
}

func (s *DungeonMasterTestSuite) TestMultipleTurnsWithoutSkillChecks() {
	ctx := s.T().Context()

	// Create a game state with NO skill checks (frequency = 0)
	cryptState, err := crypt.NewState()
	s.Require().NoError(err)

	gameState := &GameState{
		narrative: narrative.NewState(),
		dice:      dice.NewState(&dice.Config{GracePeriodMin: 100, GracePeriodMax: 100, SkillCheckFrequency: 0.0}),
		crypt:     cryptState,
		llmFunc:   s.mockLLM.CreateMessage,
		config: &Config{
			summarizationThreshold: 10,
			summarizeMaxTokens:     100,
			narrativeMaxTokens:     100,
		},
	}
	handler, err := buildHandler(gameState)
	s.Require().NoError(err)

	// Create in-memory server with transport
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	s.Require().True(ok, "failed to unwrap SDK server")
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Start server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()
	go func() {
		if err := wrappedServer.Run(serverCtx); err != nil && !errors.Is(err, context.Canceled) {
			s.T().Errorf("server run error: %v", err)
		}
	}()

	// Create client
	testImpl := &mcp.Implementation{Name: "test-client", Version: "1.0.0"}
	client := mcp.NewClient(testImpl, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	s.Require().NoError(err)
	defer session.Close() //nolint:errcheck

	// Perform multiple actions without skill checks
	actions := []string{"look around", "walk forward", "examine door"}
	for i, action := range actions {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "dungeon_master",
			Arguments: map[string]any{"action": action},
		})
		s.NotNil(result)
		s.Require().NoError(err)
		s.Require().Len(result.Content, 1)

		content := result.Content[0].(*mcp.TextContent)
		var resp narrative.Response
		s.Require().NoError(json.Unmarshal([]byte(content.Text), &resp))
		s.Empty(resp.ErrorMessage, "Turn %d should not have error", i)
		s.NotEmpty(resp.Narrative, "Turn %d should have narrative", i)
		s.Equal(i+1, resp.TurnNumber, "Turn %d should have correct turn number", i)
		s.Zero(resp.SkillCheckRequired, "Turn %d should not require skill check", i)
	}

	s.mockLLM.AssertExpectations(s.T())
}

func (s *DungeonMasterTestSuite) TestSequentialSkillChecks() {
	ctx := s.T().Context()

	// Perform first skill check cycle
	s.triggerSkillCheck(ctx, "climb the mountain")

	// Roll for first check
	result, err := s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "roll_d20",
		Arguments: map[string]any{},
	})
	s.Require().NoError(err)
	content := result.Content[0].(*mcp.TextContent)
	var roll1 dice.Roll
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &roll1))

	// Submit action with first roll (this will trigger ANOTHER skill check due to frequency=1.0)
	result, err = s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "continue climbing", "encryptedData": roll1.EncryptedData},
	})
	s.Require().NoError(err)
	content = result.Content[0].(*mcp.TextContent)
	var resp1 narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &resp1))
	s.Empty(resp1.ErrorMessage)
	s.NotEmpty(resp1.Narrative)
	s.Equal(2, resp1.TurnNumber, "triggerSkillCheck advances turn to 1, first action with roll advances to 2")
	s.Positive(resp1.SkillCheckRequired, "Second action should also require skill check")

	// Roll for second check
	result, err = s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "roll_d20",
		Arguments: map[string]any{},
	})
	s.Require().NoError(err)
	content = result.Content[0].(*mcp.TextContent)
	var roll2 dice.Roll
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &roll2))
	s.NotEmpty(roll2.EncryptedData)

	// Submit action with second roll
	result, err = s.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "reach the summit", "encryptedData": roll2.EncryptedData},
	})
	s.Require().NoError(err)
	content = result.Content[0].(*mcp.TextContent)
	var resp2 narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content.Text), &resp2))
	s.Empty(resp2.ErrorMessage)
	s.NotEmpty(resp2.Narrative)
	s.Equal(3, resp2.TurnNumber, "After two actions with rolls, turn should be 3")

	s.mockLLM.AssertExpectations(s.T())
}

func (s *DungeonMasterTestSuite) TestDuplicateActionSubmission() {
	ctx := s.T().Context()

	// Create a game state with NO skill checks (frequency = 0) to test pure duplicate behavior
	cryptState, err := crypt.NewState()
	s.Require().NoError(err)

	gameState := &GameState{
		narrative: narrative.NewState(),
		dice:      dice.NewState(&dice.Config{GracePeriodMin: 100, GracePeriodMax: 100, SkillCheckFrequency: 0.0}),
		crypt:     cryptState,
		llmFunc:   s.mockLLM.CreateMessage,
		config: &Config{
			summarizationThreshold: 10,
			summarizeMaxTokens:     100,
			narrativeMaxTokens:     100,
		},
	}
	handler, err := buildHandler(gameState)
	s.Require().NoError(err)

	// Create in-memory server with transport
	sdkServer, ok := handler.GetServer().Unwrap().(*mcp.Server)
	s.Require().True(ok, "failed to unwrap SDK server")
	wrappedServer, clientTransport := mcpwrapper.NewInMemoryServer(sdkServer)

	// Start server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()
	go func() {
		if err := wrappedServer.Run(serverCtx); err != nil && !errors.Is(err, context.Canceled) {
			s.T().Errorf("server run error: %v", err)
		}
	}()

	// Create client
	testImpl := &mcp.Implementation{Name: "test-client", Version: "1.0.0"}
	client := mcp.NewClient(testImpl, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	s.Require().NoError(err)
	defer session.Close() //nolint:errcheck

	// Submit first action
	result1, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "look around"},
	})
	s.Require().NoError(err)
	content1 := result1.Content[0].(*mcp.TextContent)
	var resp1 narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content1.Text), &resp1))

	// Submit same action again (simulating LLM retry/confusion)
	result2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dungeon_master",
		Arguments: map[string]any{"action": "look around"},
	})
	s.Require().NoError(err)
	content2 := result2.Content[0].(*mcp.TextContent)
	var resp2 narrative.Response
	s.Require().NoError(json.Unmarshal([]byte(content2.Text), &resp2))

	// Both should succeed (no duplicate detection yet)
	s.Empty(resp1.ErrorMessage)
	s.Empty(resp2.ErrorMessage)
	s.Equal(1, resp1.TurnNumber)
	s.Equal(2, resp2.TurnNumber, "Duplicate action advances turn counter")

	// This test documents current behavior: duplicates ARE processed
	// Future enhancement: add duplicate detection to prevent this

	s.mockLLM.AssertExpectations(s.T())
}
