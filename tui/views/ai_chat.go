package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/termbus/termbus/internal/agent"
	"github.com/termbus/termbus/tui/styles"
)

type AIChatModel struct {
	agentID    string
	sessionID  string
	runtime    *agent.AgentRuntime
	messages   []*agent.ChatMessage
	plan       *agent.ExecutionPlan
	result     *agent.ExecutionResult
	inputValue string
	width      int
	height     int
	loading    bool
	viewState  string
	err        error
}

func NewAIChat(sessionID string, runtime *agent.AgentRuntime) *AIChatModel {
	return &AIChatModel{
		sessionID: sessionID,
		runtime:   runtime,
		messages:  make([]*agent.ChatMessage, 0),
		viewState: "chat",
		width:     80,
		height:    24,
	}
}

func (m *AIChatModel) Init() tea.Cmd {
	return nil
}

func (m *AIChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.inputValue != "" && !m.loading {
				return m, m.sendMessage(m.inputValue)
			}
		case "backspace":
			if len(m.inputValue) > 0 {
				m.inputValue = m.inputValue[:len(m.inputValue)-1]
			}
		default:
			m.inputValue += msg.String()
		}

	case agentMessageMsg:
		m.messages = append(m.messages, &agent.ChatMessage{
			Role:      "assistant",
			Content:   msg.content,
			Timestamp: time.Now(),
		})
		m.loading = false

	case planMsg:
		m.plan = msg.plan
		m.viewState = "plan"
		m.loading = false

	case resultMsg:
		m.result = msg.result
		m.viewState = "result"
		m.loading = false
	}

	return m, nil
}

func (m *AIChatModel) sendMessage(message string) tea.Cmd {
	m.messages = append(m.messages, &agent.ChatMessage{
		Role:      "user",
		Content:   message,
		Timestamp: time.Now(),
	})

	m.loading = true
	m.inputValue = ""

	llmClient := &mockLLMClient{}
	tools := []agent.Tool{}

	agent, err := m.runtime.Create(m.sessionID, "gpt-4", llmClient, tools)
	if err != nil {
		m.err = err
		m.loading = false
		return nil
	}

	m.agentID = agent.ID

	go func() {
		resp, err := m.runtime.Process(agent.ID, message)
		if err != nil {
			m.err = err
			m.loading = false
			return
		}

		if resp.Plan != nil {
			m.plan = resp.Plan
		}
		if resp.Result != nil {
			m.result = resp.Result
		}
		m.loading = false
	}()

	return nil
}

func (m *AIChatModel) View() string {
	var content strings.Builder

	header := styles.GetEditorHeaderStyle().
		Width(m.width - 2).
		Render(fmt.Sprintf("AI Assistant (Agent: %s)", m.agentID))

	content.WriteString(header + "\n\n")

	if m.viewState == "plan" && m.plan != nil {
		content.WriteString(renderPlan(m.plan, m.width-4))
	} else if m.viewState == "result" && m.result != nil {
		content.WriteString(renderResult(m.result, m.width-4))
	} else {
		for _, msg := range m.messages {
			content.WriteString(renderMessage(msg, m.width-4))
		}

		if m.loading {
			content.WriteString(styles.Muted.Render("  Thinking...") + "\n")
		}
	}

	inputLine := fmt.Sprintf("> %s", m.inputValue)
	if m.loading {
		inputLine = styles.Muted.Render(inputLine)
	}

	content.WriteString("\n" + strings.Repeat("─", m.width-2) + "\n")
	content.WriteString(inputLine)

	footer := styles.GetEditorFooterStyle().
		Width(m.width - 2).
		Render("Enter: Send | Ctrl+C: Quit | Esc: Clear")

	container := styles.GetEditorContainerStyle().Width(m.width).Height(m.height)
	return container.Render(content.String() + "\n" + footer)
}

func (m *AIChatModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *AIChatModel) ConfirmPlan() tea.Cmd {
	if m.agentID == "" || m.plan == nil {
		return nil
	}

	m.loading = true
	m.viewState = "executing"

	go func() {
		result, err := m.runtime.ExecutePlan(m.agentID, m.plan)
		if err != nil {
			m.err = err
			m.loading = false
			return
		}
		m.result = result
		m.viewState = "result"
		m.loading = false
	}()

	return nil
}

func (m *AIChatModel) CancelPlan() {
	m.plan = nil
	m.viewState = "chat"
}

type agentMessageMsg struct{ content string }
type planMsg struct{ plan *agent.ExecutionPlan }
type resultMsg struct{ result *agent.ExecutionResult }

func renderMessage(msg *agent.ChatMessage, width int) string {
	role := msg.Role
	if role == "assistant" {
		role = "AI"
	}
	return fmt.Sprintf("%s: %s\n", role, msg.Content)
}

func renderPlan(plan *agent.ExecutionPlan, width int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  📋 %s\n\n", plan.Description))

	riskLevel := "low"
	for _, step := range plan.Steps {
		if step.RiskLevel == "high" || step.RiskLevel == "critical" {
			riskLevel = step.RiskLevel
		}
	}
	b.WriteString(fmt.Sprintf("  🎯 Risk Level: %s\n\n", riskLevel))

	b.WriteString("  📝 Steps:\n")
	for i, step := range plan.Steps {
		b.WriteString(fmt.Sprintf("    %d. %s (%s)\n", i+1, step.Description, step.Action))
	}

	b.WriteString("\n  [Confirm] [Cancel]\n")

	return b.String()
}

func renderResult(result *agent.ExecutionResult, width int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  📊 Status: %s\n", result.Status))
	b.WriteString(fmt.Sprintf("  ⏱️ Duration: %v\n\n", result.Duration))

	successCount := 0
	failCount := 0

	for _, step := range result.StepResults {
		if step.Status == "success" {
			successCount++
		} else if step.Status == "failed" {
			failCount++
		}
	}

	b.WriteString(fmt.Sprintf("  ✅ Success: %d | ❌ Failed: %d\n\n", successCount, failCount))

	b.WriteString("  📝 Results:\n")
	for _, step := range result.StepResults {
		icon := "✅"
		if step.Status == "failed" {
			icon = "❌"
		}
		b.WriteString(fmt.Sprintf("    %s Step %s: %s\n", icon, step.StepID, step.Status))
	}

	return b.String()
}

type mockLLMClient struct{}

func (m *mockLLMClient) Chat(msgs []*agent.ChatMessage, opts *agent.ChatOptions) (*agent.ChatResponse, error) {
	return &agent.ChatResponse{
		Message: &agent.ChatMessage{
			Role:    "assistant",
			Content: "I understand your request. Let me help you with that.",
		},
	}, nil
}

func (m *mockLLMClient) ChatStream(msgs []*agent.ChatMessage, opts *agent.ChatOptions, handler func(*agent.ChatResponse) error) error {
	return nil
}

func (m *mockLLMClient) FunctionCall(msgs []*agent.ChatMessage, tools []*agent.ToolDefinition) (*agent.FunctionCallResponse, error) {
	return nil, nil
}

func (m *mockLLMClient) GetModelInfo() *agent.ModelInfo {
	return &agent.ModelInfo{Name: "mock", Provider: "mock", MaxTokens: 4096, SupportsTool: false}
}
