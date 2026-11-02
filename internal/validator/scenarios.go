package validator

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// automationViewForm names the rules of an automation scenario without a when
// step. Such a scenario states the content of a read model of its automation
// after the given events, with the step rules of a state_view scenario.
const automationViewForm = "automation view form"

// scenarioRules selects the rule set of one scenario. The count of when steps
// selects the form of an automation scenario: zero gives the view form, any
// other count gives the action form of the automation keyword.
func scenarioRules(workflowType string, whenCount int) string {
	if workflowType == "automation" && whenCount == 0 {
		return automationViewForm
	}
	return workflowType
}

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
		}
	}
	rules := scenarioRules(workflowType, whenCount)
	for _, step := range content.Blocks {
		if step.Type == "comment" {
			diagnostics = append(diagnostics, validateSimpleBlock(step.Body, syntax.CommentSchema(), commentRule)...)
			continue
		}
		diagnostics = append(diagnostics, v.validateScenarioStep(rules, workflowID, step)...)
	}
	switch rules {
	case "state_view":
		if whenCount != 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios may not contain when steps. Found: when."))
		}
		if givenCount == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios must contain at least one given step."))
		}
	case automationViewForm:
		if givenCount == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "an automation scenario without a when step is a view form scenario and must contain at least one given event step. An action form scenario needs exactly one when step."))
		}
	default:
		if whenCount != 1 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario must contain exactly one when step."))
		}
	}
	if thenCount == 0 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario must contain at least one then step."))
	}
	return diagnostics
}

func (v *modelValidator) validateScenarioStep(rules, workflowID string, block *hcl.Block) hcl.Diagnostics {
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
	attribute := content.Attributes[target]
	if !scenarioTargetAllowed(rules, block.Type, target) {
		detail := fmt.Sprintf("%s scenario %s steps may use: %s. You referenced: %s.", rules, block.Type, allowedScenarioTargets(rules, block.Type), target)
		if rules == automationViewForm {
			detail += " An automation scenario without a when step is a view form scenario. An action form scenario needs exactly one when step."
		}
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenarioTarget, attribute.Expr.Range(), "Invalid scenario target", detail))
	}
	if target != "error" {
		referenceDiagnostics := v.validateReference(attribute, target, workflowID)
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if rules == automationViewForm && block.Type == "then" && target == "readmodel" && !referenceDiagnostics.HasErrors() {
			diagnostics = append(diagnostics, ownReadModelOnly(attribute, workflowID)...)
		}
	}
	context := contextValidator{model: v}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

// ownReadModelOnly reports a view form read model target that names a read
// model of another workflow through the workflow-qualified form.
func ownReadModelOnly(attribute *hcl.Attribute, workflowID string) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return nil
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) != 3 || parts[1] == workflowID {
		return nil
	}
	return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("a view form scenario may reference only a read model of its own automation %q. You referenced: %s.", workflowID, strings.Join(parts, ".")))}
}

func scenarioTargetAllowed(rules, stepType, target string) bool {
	switch rules {
	case "state_change":
		return stepType == "given" && target == "event" || stepType == "when" && target == "command" || stepType == "then" && (target == "event" || target == "error")
	case "state_view", automationViewForm:
		return stepType == "given" && target == "event" || stepType == "then" && (target == "readmodel" || target == "error")
	case "automation":
		return stepType == "given" && (target == "event" || target == "readmodel") || stepType == "when" && (target == "processor" || target == "command") || stepType == "then" && (target == "event" || target == "error")
	}
	return false
}

func allowedScenarioTargets(rules, stepType string) string {
	allowed := make([]string, 0, 2)
	for _, target := range []string{"event", "command", "readmodel", "processor", "error"} {
		if scenarioTargetAllowed(rules, stepType, target) {
			allowed = append(allowed, target)
		}
	}
	return strings.Join(allowed, " or ")
}
