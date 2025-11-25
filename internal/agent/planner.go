package agent

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Planner struct {
	llmClient LLMClient
	mu        sync.RWMutex
}

type ExecutionPlan struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Steps       []*PlanStep            `json:"steps"`
	Tools       []string               `json:"tools"`
	Context     map[string]interface{} `json:"context"`
	CreatedAt   time.Time              `json:"created_at"`
}

type PlanStep struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters"`
	DependsOn   []string               `json:"depends_on"`
	RiskLevel   string                 `json:"risk_level"`
}

func NewPlanner(llmClient LLMClient) *Planner {
	return &Planner{
		llmClient: llmClient,
	}
}

func (p *Planner) Plan(ctx *AgentContext, request string) (*ExecutionPlan, error) {
	systemPrompt := `你是一个任务规划器。根据用户请求，将其分解为可执行的步骤。
每个步骤需要指定：工具名称、参数、依赖的前置步骤、风险等级。
请以JSON格式返回执行计划。`

	userPrompt := fmt.Sprintf(`用户请求: %s

请生成执行计划，格式如下:
{
  "description": "任务描述",
  "steps": [
    {
      "id": "step_1",
      "description": "步骤描述",
      "action": "工具名称",
      "parameters": {"参数": "值"},
      "depends_on": [],
      "risk_level": "low/medium/high"
    }
  ],
  "tools": ["需要的工具列表"]
}`, request)

	messages := []*ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := p.llmClient.Chat(messages, &ChatOptions{
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get plan from LLM: %w", err)
	}

	plan, err := p.parsePlan(resp.Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	return plan, nil
}

func (p *Planner) parsePlan(content string) (*ExecutionPlan, error) {
	content = extractJSON(content)

	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	if plan.ID == "" {
		plan.ID = fmt.Sprintf("plan_%d", time.Now().UnixNano())
	}
	plan.CreatedAt = time.Now()

	return &plan, nil
}

func (p *Planner) RefinePlan(ctx *AgentContext, plan *ExecutionPlan, feedback string) (*ExecutionPlan, error) {
	systemPrompt := `你是一个任务规划器。根据反馈改进执行计划。`

	userPrompt := fmt.Sprintf(`当前计划: %s

反馈: %s

请生成改进后的执行计划（JSON格式）`, toJSON(plan), feedback)

	messages := []*ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := p.llmClient.Chat(messages, nil)
	if err != nil {
		return nil, err
	}

	return p.parsePlan(resp.Message.Content)
}

func (p *Planner) ExplainPlan(plan *ExecutionPlan) string {
	var steps []string
	for i, step := range plan.Steps {
		steps = append(steps, fmt.Sprintf("%d. %s (使用 %s)", i+1, step.Description, step.Action))
	}
	return fmt.Sprintf("计划: %s\n\n步骤:\n%s", plan.Description, join(steps, "\n"))
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func extractJSON(s string) string {
	start := 0
	end := len(s)

	for i := 0; i < len(s); i++ {
		if s[i] == '{' {
			start = i
			break
		}
	}

	stack := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			stack++
		} else if s[i] == '}' {
			stack--
			if stack == 0 {
				end = i + 1
				break
			}
		}
	}

	return s[start:end]
}

func join(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
