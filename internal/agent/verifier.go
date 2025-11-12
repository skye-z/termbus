package agent

import (
	"fmt"

	"github.com/termbus/termbus/pkg/types"
)

type Verifier struct {
	llmClient LLMClient
	eventBus  types.EventBus
}

type VerificationResult struct {
	PlanID      string
	Status      string
	Issues      []string
	Suggestions []string
	RetryNeeded bool
	Confidence  float64
}

func NewVerifier(llmClient LLMClient, eventBus types.EventBus) *Verifier {
	return &Verifier{
		llmClient: llmClient,
		eventBus:  eventBus,
	}
}

func (v *Verifier) Verify(plan *ExecutionPlan, result *ExecutionResult) (*VerificationResult, error) {
	verification := &VerificationResult{
		PlanID:      plan.ID,
		Status:      "passed",
		Issues:      []string{},
		Suggestions: []string{},
		RetryNeeded: false,
		Confidence:  1.0,
	}

	failedSteps := 0
	totalSteps := len(plan.Steps)

	for _, stepResult := range result.StepResults {
		if stepResult.Status == "failed" {
			failedSteps++
			verification.Issues = append(verification.Issues, fmt.Sprintf("Step %s failed: %v", stepResult.StepID, stepResult.Error))
		}
	}

	if failedSteps > 0 {
		verification.Status = "failed"
		verification.RetryNeeded = true
		verification.Confidence = float64(totalSteps-failedSteps) / float64(totalSteps)
		verification.Suggestions = append(verification.Suggestions, "Please retry the failed steps")
	} else if failedSteps > totalSteps/2 {
		verification.Status = "warning"
		verification.Confidence = 0.5
	}

	v.eventBus.Publish("agent.verified", verification)

	return verification, nil
}

func (v *Verifier) VerifyOutput(expected, actual string) (*VerificationResult, error) {
	result := &VerificationResult{
		Status:     "passed",
		Confidence: 1.0,
	}

	if expected != actual {
		result.Status = "failed"
		result.Confidence = 0.0
		result.Issues = append(result.Issues, "Output does not match expected")
	}

	return result, nil
}

func (v *Verifier) GenerateFeedback(ctx *AgentContext, result *VerificationResult) string {
	if result.Status == "passed" {
		return "Task completed successfully!"
	}

	if result.RetryNeeded {
		return fmt.Sprintf("Task completed with issues: %s. Please review and retry.", join(result.Issues, "; "))
	}

	return fmt.Sprintf("Task completed. Issues found: %s", join(result.Issues, "; "))
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
