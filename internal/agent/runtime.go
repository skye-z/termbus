package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/pkg/types"
)

type AgentState int

const (
	StateIdle AgentState = iota
	StatePlanning
	StateExecuting
	StateVerifying
	StateFeedback
	StateError
)

func (s AgentState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StatePlanning:
		return "planning"
	case StateExecuting:
		return "executing"
	case StateVerifying:
		return "verifying"
	case StateFeedback:
		return "feedback"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

type AgentContext struct {
	SessionID   string
	Variables   map[string]interface{}
	History     []*ConversationMessage
	FileContext map[string]*FileContext
}

type ConversationMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type FileContext struct {
	Path    string
	Content string
	ModTime time.Time
}

type Agent struct {
	ID        string        `json:"id"`
	SessionID string        `json:"session_id"`
	Model     string        `json:"model"`
	Config    *AgentConfig  `json:"config"`
	Planner   *Planner      `json:"-"`
	Executor  *Executor     `json:"-"`
	Verifier  *Verifier     `json:"-"`
	Tools     []Tool        `json:"-"`
	Context   *AgentContext `json:"-"`
	State     AgentState    `json:"state"`
}

type AgentConfig struct {
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
}

type AgentResponse struct {
	Message string           `json:"message"`
	Plan    *ExecutionPlan   `json:"plan,omitempty"`
	Result  *ExecutionResult `json:"result,omitempty"`
	Error   string           `json:"error,omitempty"`
}

type AgentRuntime struct {
	agents   map[string]*Agent
	config   interface{}
	eventBus types.EventBus
	mu       sync.RWMutex
}

func NewAgentRuntime(eventBus types.EventBus) *AgentRuntime {
	return &AgentRuntime{
		agents:   make(map[string]*Agent),
		eventBus: eventBus,
	}
}

func (r *AgentRuntime) Create(sessionID string, model string, llmClient LLMClient, tools []Tool) (*Agent, error) {
	agent := &Agent{
		ID:        fmt.Sprintf("agent_%d", time.Now().UnixNano()),
		SessionID: sessionID,
		Model:     model,
		Config: &AgentConfig{
			Model:       model,
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		Context: &AgentContext{
			SessionID:   sessionID,
			Variables:   make(map[string]interface{}),
			History:     make([]*ConversationMessage, 0),
			FileContext: make(map[string]*FileContext),
		},
		State: StateIdle,
	}

	agent.Planner = NewPlanner(llmClient)
	agent.Executor = NewExecutor(tools, eventBus)
	agent.Verifier = NewVerifier(llmClient, eventBus)
	agent.Tools = tools

	r.mu.Lock()
	r.agents[agent.ID] = agent
	r.mu.Unlock()

	r.eventBus.Publish("agent.created", agent)

	return agent, nil
}

func (r *AgentRuntime) Process(agentID string, userMessage string) (*AgentResponse, error) {
	agent, err := r.Get(agentID)
	if err != nil {
		return nil, err
	}

	agent.Context.History = append(agent.Context.History, &ConversationMessage{
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	})

	r.eventBus.Publish("agent.planning", agentID)
	agent.State = StatePlanning

	plan, err := agent.Planner.Plan(agent.Context, userMessage)
	if err != nil {
		agent.State = StateError
		r.eventBus.Publish("agent.error", agentID, err.Error())
		return &AgentResponse{Error: err.Error()}, err
	}

	r.eventBus.Publish("agent.planned", agentID, plan)
	agent.State = StateExecuting

	result, err := agent.Executor.Execute(agentID, plan)
	if err != nil {
		agent.State = StateError
		r.eventBus.Publish("agent.error", agentID, err.Error())
		return &AgentResponse{Error: err.Error()}, err
	}

	r.eventBus.Publish("agent.executed", agentID, result)

	agent.State = StateVerifying
	verification, err := agent.Verifier.Verify(plan, result)
	if err != nil {
		r.eventBus.Publish("agent.verification_error", agentID, err.Error())
	} else {
		r.eventBus.Publish("agent.verified", agentID, verification)
	}

	agent.State = StateIdle

	response := &AgentResponse{
		Message: agent.Verifier.GenerateFeedback(agent.Context, &VerificationResult{
			PlanID:      plan.ID,
			Status:      "success",
			Issues:      []string{},
			RetryNeeded: false,
			Confidence:  0.9,
		}),
		Plan:   plan,
		Result: result,
	}

	agent.Context.History = append(agent.Context.History, &ConversationMessage{
		Role:      "assistant",
		Content:   response.Message,
		Timestamp: time.Now(),
	})

	return response, nil
}

func (r *AgentRuntime) ExecutePlan(agentID string, plan *ExecutionPlan) (*ExecutionResult, error) {
	agent, err := r.Get(agentID)
	if err != nil {
		return nil, err
	}

	agent.State = StateExecuting
	r.eventBus.Publish("agent.executing", agentID, plan)

	result, err := agent.Executor.Execute(agentID, plan)
	if err != nil {
		agent.State = StateError
		return result, err
	}

	agent.State = StateIdle
	r.eventBus.Publish("agent.executed", agentID, result)

	return result, nil
}

func (r *AgentRuntime) Stop(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found")
	}

	agent.State = StateIdle
	r.eventBus.Publish("agent.stopped", agentID)

	return nil
}

func (r *AgentRuntime) Get(agentID string) (*Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

func (r *AgentRuntime) List(sessionID string) []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Agent
	for _, agent := range r.agents {
		if sessionID == "" || agent.SessionID == sessionID {
			result = append(result, agent)
		}
	}

	return result
}

func (r *AgentRuntime) Delete(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentID]; !exists {
		return fmt.Errorf("agent not found")
	}

	delete(r.agents, agentID)
	r.eventBus.Publish("agent.deleted", agentID)

	return nil
}
