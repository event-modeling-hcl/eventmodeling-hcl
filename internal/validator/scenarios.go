package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
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
	processorWhen := false
	for _, step := range content.Blocks {
		switch step.Type {
		case "given":
			givenCount++
		case "when":
			whenCount++
			stepContent, _, _ := syntax.PartialContent(step.Body, syntax.ScenarioStepSchema())
			processorWhen = stepContent.Attributes["processor"] != nil && len(stepTargets(stepContent)) == 1
		case "then":
			thenCount++
		}
	}
	processorWhen = processorWhen && whenCount == 1
	rules := scenarioRules(workflowType, whenCount)
	attributeDiagnostics, emptyHistory := scenarioAttributes(content.Attributes, rules)
	diagnostics = append(diagnostics, attributeDiagnostics...)
	dispatches := map[string]map[bool]bool{}
	for _, step := range content.Blocks {
		if step.Type == "comment" {
			diagnostics = append(diagnostics, validateSimpleBlock(step.Body, syntax.CommentSchema(), commentRule)...)
			continue
		}
		stepDiagnostics, dispatch := v.validateScenarioStep(rules, workflowID, step, processorWhen)
		diagnostics = append(diagnostics, stepDiagnostics...)
		if dispatch.command != "" {
			if dispatches[dispatch.command] == nil {
				dispatches[dispatch.command] = map[bool]bool{}
			}
			dispatches[dispatch.command][dispatch.negative] = true
		}
	}
	for _, command := range sortedKeys(dispatches) {
		if dispatches[command][true] && dispatches[command][false] {
			diagnostics = append(diagnostics, errorDiagnostic(codeContradictoryDispatch, block.DefRange, "Contradictory dispatch", fmt.Sprintf("scenario %q states that the processor issues command.%s and also that it does not issue it.", block.Labels[0], command)))
		}
	}
	switch rules {
	case "state_view", automationViewForm:
		if rules == "state_view" && whenCount != 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios may not contain when steps. Found: when."))
		}
		switch {
		case emptyHistory && givenCount > 0:
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario with empty_history = true states the read model before any event and may not contain given steps."))
		case !emptyHistory && givenCount == 0 && rules == "state_view":
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios must contain at least one given step, or set empty_history = true."))
		case !emptyHistory && givenCount == 0:
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "an automation scenario without a when step is a view form scenario and must contain at least one given event step, or set empty_history = true. An action form scenario needs exactly one when step."))
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

// scenarioAttributes checks case and empty_history against the form of the
// scenario (EM304). It reports whether the scenario states an empty starting
// history. Neither attribute selects the form, and case never changes the
// step rules.
func scenarioAttributes(attributes hcl.Attributes, rules string) (hcl.Diagnostics, bool) {
	var diagnostics hcl.Diagnostics
	viewForm := rules == "state_view" || rules == automationViewForm
	if attribute := attributes["case"]; attribute != nil {
		value, ok := literalString(attribute)
		group, groupName := commandCases, "command"
		if viewForm {
			group, groupName = readModelCases, "read model"
		}
		known := slices.Contains(commandCases, value) || slices.Contains(readModelCases, value)
		if ok && known && !slices.Contains(group, value) {
			diagnostics = append(diagnostics, errorDiagnostic(codeScenarioAttributeMisfit, attribute.Expr.Range(), "Invalid scenario case", fmt.Sprintf("a %s scenario accepts the %s cases: %s. You wrote: %q.", rules, groupName, strings.Join(group, ", "), value)))
		}
	}
	attribute := attributes["empty_history"]
	if attribute == nil {
		return diagnostics, false
	}
	if !viewForm {
		return append(diagnostics, errorDiagnostic(codeScenarioAttributeMisfit, attribute.NameRange, "Invalid scenario attribute", fmt.Sprintf("empty_history is allowed only on a view scenario, not on a %s scenario, which already accepts zero given steps.", rules))), false
	}
	emptyHistory, _ := literalBool(attribute)
	return diagnostics, emptyHistory
}

// dispatchAssertion is the command that one then step states as issued or,
// with negative, as not issued. command is empty for another step.
type dispatchAssertion struct {
	command  string
	negative bool
}

func stepTargets(content *hcl.BodyContent) []string {
	targets := make([]string, 0, 1)
	for _, target := range []string{"event", "command", "readmodel", "processor", "error"} {
		if content.Attributes[target] != nil {
			targets = append(targets, target)
		}
	}
	return targets
}

func (v *modelValidator) validateScenarioStep(rules, workflowID string, block *hcl.Block, processorWhen bool) (hcl.Diagnostics, dispatchAssertion) {
	schema := syntax.ScenarioStepSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, scenarioStepRule)...)
	setTargets := stepTargets(content)
	if len(setTargets) != 1 {
		return append(diagnostics, errorDiagnostic(codeInvalidScenarioStep, block.DefRange, "Invalid scenario step", "a scenario step must set exactly one target: event, command, readmodel, processor, or error.")), dispatchAssertion{}
	}
	target := setTargets[0]
	attribute := content.Attributes[target]
	allowed := scenarioTargetAllowed(rules, block.Type, target, processorWhen)
	if !allowed {
		detail := fmt.Sprintf("%s scenario %s steps may use: %s. You referenced: %s.", rules, block.Type, allowedScenarioTargets(rules, block.Type, processorWhen), target)
		if rules == automationViewForm {
			detail += " An automation scenario without a when step is a view form scenario. An action form scenario needs exactly one when step."
		}
		if rules == "automation" && block.Type == "then" && target == "command" {
			detail += " A then step can name a command only when the when step names a processor."
		}
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenarioTarget, attribute.Expr.Range(), "Invalid scenario target", detail))
	}
	dispatchCommand := rules == "automation" && block.Type == "then" && target == "command" && allowed
	var dispatch dispatchAssertion
	if target != "error" {
		referenceDiagnostics := v.validateReference(attribute, target, workflowID)
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if !referenceDiagnostics.HasErrors() && block.Type == "then" && (rules == automationViewForm && target == "readmodel" || dispatchCommand) {
			ownDiagnostics := ownElementOnly(attribute, workflowID)
			diagnostics = append(diagnostics, ownDiagnostics...)
			if dispatchCommand && !ownDiagnostics.HasErrors() {
				parts, _ := source.ReferenceParts(attribute.Expr)
				dispatch.command = parts[len(parts)-1]
			}
		}
	}
	if noDispatch := content.Attributes["expect_no_dispatch"]; noDispatch != nil {
		switch {
		case block.Type != "then" || target != "command":
			diagnostics = append(diagnostics, errorDiagnostic(codeScenarioAttributeMisfit, noDispatch.NameRange, "Invalid step attribute", "expect_no_dispatch is allowed only on a then step that names a command."))
		case content.Attributes["fields"] != nil || len(content.Blocks) > 0:
			diagnostics = append(diagnostics, errorDiagnostic(codeScenarioAttributeMisfit, noDispatch.NameRange, "Invalid step attribute", "a step with expect_no_dispatch states that no command is issued, so it has no field blocks and no fields list."))
		default:
			dispatch.negative, _ = literalBool(noDispatch)
		}
	}
	context := contextValidator{model: v}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics, dispatch
}

// ownElementOnly reports a then step target that names an element of another
// workflow through the workflow-qualified form: the read model of a view form
// scenario, or the command of a dispatch assertion.
func ownElementOnly(attribute *hcl.Attribute, workflowID string) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return nil
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) != 3 || parts[1] == workflowID {
		return nil
	}
	if parts[0] == "readmodel" {
		return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("a view form scenario may reference only a read model of its own automation %q. You referenced: %s.", workflowID, strings.Join(parts, ".")))}
	}
	return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("a dispatch assertion may reference only a command of its own automation %q. You referenced: %s.", workflowID, strings.Join(parts, ".")))}
}

func scenarioTargetAllowed(rules, stepType, target string, processorWhen bool) bool {
	switch rules {
	case "state_change":
		return stepType == "given" && target == "event" || stepType == "when" && target == "command" || stepType == "then" && (target == "event" || target == "error")
	case "state_view", automationViewForm:
		return stepType == "given" && target == "event" || stepType == "then" && (target == "readmodel" || target == "error")
	case "automation":
		return stepType == "given" && (target == "event" || target == "readmodel") || stepType == "when" && (target == "processor" || target == "command") || stepType == "then" && (target == "event" || target == "error" || target == "command" && processorWhen)
	}
	return false
}

func allowedScenarioTargets(rules, stepType string, processorWhen bool) string {
	allowed := make([]string, 0, 2)
	for _, target := range []string{"event", "command", "readmodel", "processor", "error"} {
		if scenarioTargetAllowed(rules, stepType, target, processorWhen) {
			allowed = append(allowed, target)
		}
	}
	return strings.Join(allowed, " or ")
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
