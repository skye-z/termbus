package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/termbus/termbus/pkg/types"
)

type RiskAssessor struct {
	rules    []RiskRule
	eventBus types.EventBus
	mu       sync.RWMutex
}

type RiskRule struct {
	ID              string
	Name            string
	Tool            string
	Keywords        []string
	RiskLevel       string
	RequiresConfirm bool
}

type RiskAssessment struct {
	Level           string
	Score           int
	Issues          []string
	RequiresConfirm bool
	Suggestions     []string
}

func NewRiskAssessor(eventBus types.EventBus) *RiskAssessor {
	return &RiskAssessor{
		rules: []RiskRule{
			{ID: "rm_root", Name: "Remove root files", Tool: "ssh_exec", Keywords: []string{"rm -rf /", "rm -rf /*"}, RiskLevel: "critical", RequiresConfirm: true},
			{ID: "systemctl", Name: "System control", Tool: "ssh_exec", Keywords: []string{"systemctl stop", "systemctl restart"}, RiskLevel: "high", RequiresConfirm: true},
			{ID: "iptables", Name: "Firewall changes", Tool: "ssh_exec", Keywords: []string{"iptables", "firewall-cmd"}, RiskLevel: "high", RequiresConfirm: true},
			{ID: "kill", Name: "Kill processes", Tool: "ssh_exec", Keywords: []string{"kill -9", "pkill"}, RiskLevel: "medium", RequiresConfirm: true},
			{ID: "write_config", Name: "Write config files", Tool: "sftp_write", Keywords: []string{".conf", ".cfg", "nginx", "apache"}, RiskLevel: "medium", RequiresConfirm: true},
		},
		eventBus: eventBus,
	}
}

func (r *RiskAssessor) Assess(tool string, params map[string]interface{}) (*RiskAssessment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	assessment := &RiskAssessment{
		Level:       "low",
		Score:       0,
		Issues:      []string{},
		Suggestions: []string{},
	}

	command, hasCommand := params["command"].(string)

	if hasCommand {
		command = strings.ToLower(command)
		for _, rule := range r.rules {
			if rule.Tool != tool {
				continue
			}

			for _, kw := range rule.Keywords {
				if strings.Contains(command, strings.ToLower(kw)) {
					assessment.Issues = append(assessment.Issues, rule.Name)

					switch rule.RiskLevel {
					case "critical":
						assessment.Score += 100
						assessment.Level = "critical"
						assessment.RequiresConfirm = true
						assessment.Suggestions = append(assessment.Suggestions, "This operation is very dangerous!")
					case "high":
						assessment.Score += 75
						if assessment.Level != "critical" {
							assessment.Level = "high"
						}
						assessment.RequiresConfirm = true
						assessment.Suggestions = append(assessment.Suggestions, "Please confirm this operation")
					case "medium":
						assessment.Score += 50
						if assessment.Level == "low" {
							assessment.Level = "medium"
						}
						if !assessment.RequiresConfirm {
							assessment.RequiresConfirm = rule.RequiresConfirm
						}
					}
				}
			}
		}
	}

	if assessment.Score == 0 {
		assessment.Level = "low"
	}

	if assessment.RequiresConfirm && r.eventBus != nil {
		r.eventBus.Publish("agent.confirm_required", assessment)
	}

	return assessment, nil
}

func (r *RiskAssessor) GetRiskLevel(score int) string {
	switch {
	case score >= 100:
		return "critical"
	case score >= 75:
		return "high"
	case score >= 50:
		return "medium"
	default:
		return "low"
	}
}

type RollbackManager struct {
	operations map[string]*Operation
	mu         sync.RWMutex
	eventBus   types.EventBus
}

type Operation struct {
	ID         string
	Type       string
	Tool       string
	Params     map[string]interface{}
	Result     *ToolResult
	Snapshot   interface{}
	Timestamp  time.Time
	Reversible bool
}

func NewRollbackManager(eventBus types.EventBus) *RollbackManager {
	return &RollbackManager{
		operations: make(map[string]*Operation),
		eventBus:   eventBus,
	}
}

func (r *RollbackManager) RecordOperation(op *Operation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	op.Timestamp = time.Now()
	r.operations[op.ID] = op

	if r.eventBus != nil {
		r.eventBus.Publish("tool.rollback_ready", op.ID)
	}

	return nil
}

func (r *RollbackManager) GetOperation(id string) (*Operation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	op, exists := r.operations[id]
	if !exists {
		return nil, fmt.Errorf("operation not found: %s", id)
	}

	return op, nil
}

func (r *RollbackManager) Rollback(id string) (*ToolResult, error) {
	r.mu.RLock()
	op, exists := r.operations[id]
	r.mu.RUnlock()

	if !exists {
		return &ToolResult{Success: false, Error: fmt.Sprintf("operation not found: %s", id)}, nil
	}

	if !op.Reversible {
		return &ToolResult{Success: false, Error: "operation is not reversible"}, nil
	}

	if r.eventBus != nil {
		r.eventBus.Publish("agent.rollback_started", id)
	}

	switch op.Type {
	case "sftp_write":
		return &ToolResult{Success: true, Output: "Rollback not yet implemented for file operations"}, nil
	default:
		return &ToolResult{Success: false, Error: "unsupported rollback type"}, nil
	}
}

func (r *RollbackManager) CanRollback(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	op, exists := r.operations[id]
	if !exists {
		return false
	}

	return op.Reversible && time.Since(op.Timestamp) < 24*time.Hour
}
