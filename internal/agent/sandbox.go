package agent

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/termbus/termbus/pkg/types"
)

type Sandbox struct {
	rules       []SandboxRule
	permissions map[string]map[string]bool
	mu          sync.RWMutex
}

type SandboxRule struct {
	ID      string
	Name    string
	Pattern string
	Action  string
	Tools   []string
}

type EvaluationResult struct {
	Allowed    bool
	Reason     string
	RuleID     string
	Suggestion string
}

func NewSandbox() *Sandbox {
	s := &Sandbox{
		rules: []SandboxRule{
			{ID: "block_rm", Name: "Block rm -rf", Pattern: `rm\s+-rf\s+[/\*]`, Action: "block", Tools: []string{"ssh_exec"}},
			{ID: "block_shutdown", Name: "Block shutdown", Pattern: `shutdown|reboot|halt`, Action: "block", Tools: []string{"ssh_exec"}},
			{ID: "warn_dangerous", Name: "Warn dangerous commands", Pattern: `dd|cat\s+/dev`, Action: "warn", Tools: []string{"ssh_exec"}},
		},
		permissions: make(map[string]map[string]bool),
	}
	return s
}

func (s *Sandbox) AddRule(rule *SandboxRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule already exists: %s", rule.ID)
		}
	}

	s.rules = append(s.rules, *rule)
	return nil
}

func (s *Sandbox) RemoveRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.rules {
		if r.ID == ruleID {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", ruleID)
}

func (s *Sandbox) Evaluate(tool string, params map[string]interface{}) (*EvaluationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	command, hasCommand := params["command"].(string)

	if hasCommand {
		for _, rule := range s.rules {
			if !contains(rule.Tools, tool) {
				continue
			}

			matched, err := regexp.MatchString(rule.Pattern, command)
			if err != nil {
				continue
			}

			if matched {
				if rule.Action == "block" {
					return &EvaluationResult{
						Allowed:    false,
						Reason:     fmt.Sprintf("Blocked by rule: %s", rule.Name),
						RuleID:     rule.ID,
						Suggestion: "This operation is not allowed",
					}, nil
				} else if rule.Action == "warn" {
					return &EvaluationResult{
						Allowed:    true,
						Reason:     fmt.Sprintf("Warning: %s", rule.Name),
						RuleID:     rule.ID,
						Suggestion: "Please be careful with this operation",
					}, nil
				}
			}
		}
	}

	return &EvaluationResult{Allowed: true, Reason: "Allowed"}, nil
}

func (s *Sandbox) RequestPermission(tool string, params map[string]interface{}) (bool, error) {
	eval, err := s.Evaluate(tool, params)
	if err != nil {
		return false, err
	}

	return eval.Allowed, nil
}

func (s *Sandbox) GrantPermission(agentID, tool string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.permissions[agentID] == nil {
		s.permissions[agentID] = make(map[string]bool)
	}
	s.permissions[agentID][tool] = true
}

func (s *Sandbox) RevokePermission(agentID, tool string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.permissions[agentID] != nil {
		delete(s.permissions[agentID], tool)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
