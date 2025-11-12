package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/termbus/termbus/pkg/types"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() []ToolParameter
	Execute(agent *Agent, params map[string]interface{}) (*ToolResult, error)
}

type ToolParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}

type ToolResult struct {
	Success bool
	Output  interface{}
	Error   string
	Data    map[string]interface{}
}

type Executor struct {
	tools    map[string]Tool
	eventBus types.EventBus
	mu       sync.RWMutex
}

type ExecutionResult struct {
	PlanID      string                 `json:"plan_id"`
	StepResults map[string]*StepResult `json:"step_results"`
	Status      string                 `json:"status"`
	Output      string                 `json:"output"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

type StepResult struct {
	StepID   string        `json:"step_id"`
	Status   string        `json:"status"`
	Output   string        `json:"output"`
	Error    error         `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
	Retries  int           `json:"retries"`
}

func NewExecutor(tools []Tool, eventBus types.EventBus) *Executor {
	executor := &Executor{
		tools:    make(map[string]Tool),
		eventBus: eventBus,
	}

	for _, tool := range tools {
		executor.tools[tool.Name()] = tool
	}

	return executor
}

func (e *Executor) Execute(agentID string, plan *ExecutionPlan) (*ExecutionResult, error) {
	startTime := time.Now()

	result := &ExecutionResult{
		PlanID:      plan.ID,
		StepResults: make(map[string]*StepResult),
		Status:      "success",
	}

	executed := make(map[string]bool)

	for _, step := range plan.Steps {
		stepResult := e.ExecuteStepByID(agentID, step)
		result.StepResults[step.ID] = stepResult

		executed[step.ID] = true

		if stepResult.Status == "failed" {
			result.Status = "failed"
			result.Error = stepResult.Error.Error()
			break
		}

		for _, dep := range step.DependsOn {
			if !executed[dep] {
				result.StepResults[step.ID] = &StepResult{
					StepID: step.ID,
					Status: "skipped",
				}
			}
		}
	}

	result.Duration = time.Since(startTime)

	return result, nil
}

func (e *Executor) ExecuteStepByID(agentID string, step *PlanStep) *StepResult {
	startTime := time.Now()

	e.eventBus.Publish("tool.called", step.Action, step.Parameters)

	tool, err := e.GetTool(step.Action)
	if err != nil {
		e.eventBus.Publish("tool.failed", step.Action, err.Error())
		return &StepResult{
			StepID:   step.ID,
			Status:   "failed",
			Error:    fmt.Errorf("tool not found: %s", step.Action),
			Duration: time.Since(startTime),
		}
	}

	dummyAgent := &Agent{ID: agentID}
	result, err := tool.Execute(dummyAgent, step.Parameters)

	if err != nil || !result.Success {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else if result.Error != "" {
			errMsg = result.Error
		}
		e.eventBus.Publish("tool.failed", step.Action, errMsg)
		return &StepResult{
			StepID:   step.ID,
			Status:   "failed",
			Output:   fmt.Sprintf("%v", result.Output),
			Error:    fmt.Errorf(errMsg),
			Duration: time.Since(startTime),
		}
	}

	e.eventBus.Publish("tool.succeeded", step.Action, result.Output)

	return &StepResult{
		StepID:   step.ID,
		Status:   "success",
		Output:   fmt.Sprintf("%v", result.Output),
		Duration: time.Since(startTime),
	}
}

func (e *Executor) RegisterTool(tool Tool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.tools[tool.Name()]; exists {
		return fmt.Errorf("tool already registered: %s", tool.Name())
	}

	e.tools[tool.Name()] = tool
	return nil
}

func (e *Executor) GetTool(name string) (Tool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tool, exists := e.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool, nil
}

func (e *Executor) ListTools() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.tools))
	for name := range e.tools {
		names = append(names, name)
	}

	return names
}
