package validator

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

func (v *modelValidator) validateScenario(workflowType, workflowID string, block *hcl.Block) hcl.Diagnostics {
	schema := syntax.ScenarioSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, scenarioRule)...)
	givenCount := 0
	whenCount := 0
	thenCount := 0
	for _, step := range content.Blocks {
		switch step.Type {
		case "given":
			givenCount++
		case "when":
			whenCount++
		case "then":
			thenCount++
		case "comment":
			diagnostics = append(diagnostics, validateSimpleBlock(step.Body, syntax.CommentSchema(), commentRule)...)
			continue
		}
		diagnostics = append(diagnostics, v.validateScenarioStep(workflowType, workflowID, step)...)
	}
	if workflowType == "state_view" {
		if whenCount != 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios may not contain when steps. Found: when."))
		}
		if givenCount == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios must contain at least one given step."))
		}
	} else if whenCount != 1 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario must contain exactly one when step."))
	}
	if thenCount == 0 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario must contain at least one then step."))
	}
	return diagnostics
}

func (v *modelValidator) validateScenarioStep(workflowType, workflowID string, block *hcl.Block) hcl.Diagnostics {
	schema := syntax.ScenarioStepSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, scenarioStepRule)...)
	targets := []string{"event", "command", "readmodel", "processor", "error"}
	setTargets := make([]string, 0, 1)
	for _, target := range targets {
		if content.Attributes[target] != nil {
			setTargets = append(setTargets, target)
		}
	}
	if len(setTargets) != 1 {
		return append(diagnostics, errorDiagnostic(codeInvalidScenarioStep, block.DefRange, "Invalid scenario step", "a scenario step must set exactly one target: event, command, readmodel, processor, or error."))
	}
	target := setTargets[0]
	if !scenarioTargetAllowed(workflowType, block.Type, target) {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenarioTarget, content.Attributes[target].Expr.Range(), "Invalid scenario target", fmt.Sprintf("%s scenario %s steps may use: %s. You referenced: %s.", workflowType, block.Type, allowedScenarioTargets(workflowType, block.Type), target)))
	}
	if target != "error" {
		diagnostics = append(diagnostics, v.validateReference(content.Attributes[target], target, workflowID)...)
	}
	context := contextValidator{model: v}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

func scenarioTargetAllowed(workflowType, stepType, target string) bool {
	switch workflowType {
	case "state_change":
		return stepType == "given" && target == "event" || stepType == "when" && target == "command" || stepType == "then" && (target == "event" || target == "error")
	case "state_view":
		return stepType == "given" && target == "event" || stepType == "then" && (target == "readmodel" || target == "error")
	case "automation", "translation":
		return stepType == "given" && (target == "event" || target == "readmodel") || stepType == "when" && (target == "processor" || target == "command") || stepType == "then" && (target == "event" || target == "error")
	}
	return false
}

func allowedScenarioTargets(workflowType, stepType string) string {
	allowed := make([]string, 0, 2)
	for _, target := range []string{"event", "command", "readmodel", "processor", "error"} {
		if scenarioTargetAllowed(workflowType, stepType, target) {
			allowed = append(allowed, target)
		}
	}
	return strings.Join(allowed, " or ")
}
