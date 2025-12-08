package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/agent"
	"github.com/termbus/termbus/internal/eventbus"
)

func TestE2E_AgentSandbox(t *testing.T) {
	sandbox := agent.NewSandbox()

	rule := &agent.SandboxRule{
		ID:      "test-rule",
		Name:    "Test Rule",
		Pattern: "test.*",
		Action:  "allow",
		Tools:   []string{"ssh"},
	}
	err := sandbox.AddRule(rule)
	require.NoError(t, err)

	result, err := sandbox.Evaluate("ssh", map[string]interface{}{
		"command": "test command",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	hasPermission, err := sandbox.RequestPermission("ssh", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, hasPermission)

	sandbox.GrantPermission("test-agent", "ssh")
	sandbox.RevokePermission("test-agent", "ssh")

	err = sandbox.RemoveRule("test-rule")
	require.NoError(t, err)
}

func TestE2E_AgentRiskAssessor(t *testing.T) {
	eventBus := eventbus.New()
	riskAssessor := agent.NewRiskAssessor(eventBus)

	assessment, err := riskAssessor.Assess("ssh", map[string]interface{}{
		"command": "ls",
	})
	require.NoError(t, err)
	assert.NotNil(t, assessment)

	riskLevel := riskAssessor.GetRiskLevel(30)
	assert.Equal(t, "low", riskLevel)

	riskLevel = riskAssessor.GetRiskLevel(70)
	assert.Equal(t, "medium", riskLevel)

	riskLevel = riskAssessor.GetRiskLevel(10)
	assert.Equal(t, "low", riskLevel)
}

func TestE2E_AgentRollbackManager(t *testing.T) {
	eventBus := eventbus.New()
	rollbackMgr := agent.NewRollbackManager(eventBus)

	op := &agent.Operation{
		ID:         "op-1",
		Type:       "ssh",
		Tool:       "ssh",
		Params:     map[string]interface{}{},
		Timestamp:  time.Now(),
		Reversible: true,
	}

	err := rollbackMgr.RecordOperation(op)
	require.NoError(t, err)

	retrieved, err := rollbackMgr.GetOperation("op-1")
	require.NoError(t, err)
	assert.Equal(t, "op-1", retrieved.ID)

	canRollback := rollbackMgr.CanRollback("op-1")
	assert.True(t, canRollback)

	canRollback = rollbackMgr.CanRollback("nonexistent")
	assert.False(t, canRollback)
}

func TestE2E_AgentRuntime(t *testing.T) {
	eventBus := eventbus.New()
	runtime := agent.NewAgentRuntime(eventBus)

	agents := runtime.List("session-1")
	assert.Equal(t, 0, len(agents))

	_, err := runtime.Get("nonexistent-agent")
	assert.Error(t, err)

	err = runtime.Delete("nonexistent-agent")
	assert.Error(t, err)
}

func TestE2E_AgentVerifier(t *testing.T) {
	eventBus := eventbus.New()
	verifier := agent.NewVerifier(nil, eventBus)

	verification, err := verifier.VerifyOutput("expected", "actual")
	require.NoError(t, err)
	assert.NotNil(t, verification)
}

func TestE2E_AgentToolDefinitions(t *testing.T) {
	tool := &agent.SSHTool{}

	assert.Equal(t, "ssh_exec", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotEmpty(t, tool.Parameters())

	sftpReadTool := &agent.SFTPReadTool{}
	assert.Equal(t, "sftp_read", sftpReadTool.Name())

	sftpWriteTool := &agent.SFTPWriteTool{}
	assert.Equal(t, "sftp_write", sftpWriteTool.Name())

	fileListTool := &agent.FileListTool{}
	assert.Equal(t, "file_list", fileListTool.Name())
}

func TestE2E_AgentExecutor(t *testing.T) {
	eventBus := eventbus.New()
	executor := agent.NewExecutor(nil, eventBus)

	tools := executor.ListTools()
	assert.NotNil(t, tools)

	_, err := executor.GetTool("nonexistent")
	assert.Error(t, err)
}

func TestE2E_AgentToolResult(t *testing.T) {
	result := &agent.ToolResult{
		Success: true,
		Output:  "test output",
		Error:   "",
	}

	assert.True(t, result.Success)
	assert.Equal(t, "test output", result.Output)

	result = &agent.ToolResult{
		Success: false,
		Output:  "",
		Error:   "test error",
	}

	assert.False(t, result.Success)
	assert.Equal(t, "test error", result.Error)
}

func TestE2E_LLMToolDefinition(t *testing.T) {
	toolDef := &agent.ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]string{"type": "string"},
			},
		},
	}

	assert.Equal(t, "test_tool", toolDef.Name)
	assert.Equal(t, "A test tool", toolDef.Description)
}

func TestE2E_AgentContext(t *testing.T) {
	ctx := &agent.AgentContext{
		SessionID: "session-1",
	}

	assert.Equal(t, "session-1", ctx.SessionID)
}

func TestE2E_AgentExecutionPlan(t *testing.T) {
	plan := &agent.ExecutionPlan{
		Steps: []*agent.PlanStep{
			{ID: "step-1", Description: "test step"},
		},
	}

	assert.Equal(t, 1, len(plan.Steps))
}

func TestE2E_AgentExecutionResult(t *testing.T) {
	result := &agent.ExecutionResult{
		StepResults: map[string]*agent.StepResult{
			"step-1": {StepID: "step-1", Output: "test"},
		},
	}

	assert.NotNil(t, result.StepResults)
}

func TestE2E_AgentVerificationResult(t *testing.T) {
	verification := &agent.VerificationResult{
		PlanID:      "plan-1",
		Status:      "completed",
		RetryNeeded: false,
		Confidence:  0.9,
	}

	assert.Equal(t, "plan-1", verification.PlanID)
	assert.Equal(t, "completed", verification.Status)
}

func TestE2E_AgentRiskAssessment(t *testing.T) {
	assessment := &agent.RiskAssessment{
		Level:           "medium",
		Score:           50,
		Issues:          []string{"command execution"},
		RequiresConfirm: false,
		Suggestions:     []string{"approve before execution"},
	}

	assert.Equal(t, "medium", assessment.Level)
	assert.Equal(t, 50, assessment.Score)
}

func TestE2E_AgentRiskRule(t *testing.T) {
	rule := agent.RiskRule{
		ID:              "test-rule",
		Name:            "Test Rule",
		Tool:            "ssh",
		Keywords:        []string{"rm", "delete"},
		RiskLevel:       "medium",
		RequiresConfirm: true,
	}

	assert.Equal(t, "test-rule", rule.ID)
	assert.Equal(t, "ssh", rule.Tool)
}

func TestE2E_AgentStepResult(t *testing.T) {
	result := &agent.StepResult{
		StepID:   "step-1",
		Output:   "test output",
		Status:   "completed",
		Duration: 100 * time.Millisecond,
		Retries:  0,
	}

	assert.Equal(t, "step-1", result.StepID)
	assert.Equal(t, "completed", result.Status)
}

func TestE2E_AgentConversationMessage(t *testing.T) {
	msg := &agent.ConversationMessage{
		Role:    "user",
		Content: "Hello",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello", msg.Content)
}

func TestE2E_AgentFileContext(t *testing.T) {
	file := &agent.FileContext{
		Path:    "/tmp/test.txt",
		Content: "test content",
	}

	assert.Equal(t, "/tmp/test.txt", file.Path)
	assert.Equal(t, "test content", file.Content)
}
