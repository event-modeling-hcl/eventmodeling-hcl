package validator

// The advisory Event-Modeling practice-smell pass (diagnostic codes
// EM401-EM406) lives in this file. Unlike the structural and reference
// checks elsewhere in this package, these diagnostics never report a hard
// error: they flag patterns worth a second look (fan-out anti-patterns,
// commands without a stated reason, and workflows that hoard all of a
// model's scenarios) and are escalated or suppressed per Profile in
// diagnostics.go. validateHotspot (structure.go) reports the unresolved-
// hotspot smell (EM406) directly since it is a simple per-block check.

import (
	"fmt"
	"slices"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// workflowSmells is the single entry point structural validation calls into
// this pass for one workflow block. It preserves the historical diagnostic
// order: externality (translation/automation misuse) first, then the
// per-element fan-out and command-reason warnings.
func (v *modelValidator) workflowSmells(workflowType string, subject hcl.Range, blocks hcl.Blocks) hcl.Diagnostics {
	diagnostics := v.validateWorkflowExternality(workflowType, subject, blocks)
	return append(diagnostics, v.workflowWarnings(blocks)...)
}

func (v *modelValidator) workflowWarnings(blocks hcl.Blocks) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	incomingCommands := incomingReferences(blocks, "command")
	for _, block := range blocks {
		if !slices.Contains([]string{"screen", "command", "readmodel"}, block.Type) {
			continue
		}
		content, _, _ := syntax.PartialContent(block.Body, syntax.ElementSchema(block.Type))
		var attribute *hcl.Attribute
		var smell, expectedRoot string
		switch block.Type {
		case "screen":
			attribute, smell, expectedRoot = content.Attributes["to"], "Bed anti-pattern", "command"
		case "command":
			attribute, smell, expectedRoot = content.Attributes["to"], "Left chair anti-pattern", "event"
		case "readmodel":
			attribute, smell, expectedRoot = content.Attributes["from"], "Right chair anti-pattern", "event"
		}
		if attribute != nil && countReferencesWithRoot(attribute, expectedRoot) > 1 {
			diagnostics = append(diagnostics, warningDiagnostic(smellCode(smell), attribute.Expr.Range(), smell, fmt.Sprintf("%s %q connects to several %ss; review whether the workflow contains more than one business capability.", block.Type, block.Labels[0], expectedRoot)))
		}
		if block.Type == "command" {
			content, _, _ := syntax.PartialContent(block.Body, syntax.ElementSchema(block.Type))
			_, hasAPI := content.Attributes["api_endpoint"]
			externalTrigger, _ := literalBool(content.Attributes["external_trigger"])
			if !hasAPI && !externalTrigger && !incomingCommands[block.Labels[0]] {
				diagnostics = append(diagnostics, warningDiagnostic(codeCommandWithoutReason, block.DefRange, "Every command has a reason", fmt.Sprintf("command %q has no incoming flow, api_endpoint, or external_trigger.", block.Labels[0])))
			}
		}
	}
	return diagnostics
}

func smellCode(summary string) diagnosticCode {
	switch summary {
	case "Bed anti-pattern":
		return codeBedAntiPattern
	case "Left chair anti-pattern":
		return codeLeftChairAntiPattern
	case "Right chair anti-pattern":
		return codeRightChairAntiPattern
	default:
		return codeUnclassified
	}
}

func incomingReferences(blocks hcl.Blocks, root string) map[string]bool {
	result := map[string]bool{}
	for _, block := range blocks {
		content, _, _ := syntax.PartialContent(block.Body, syntax.ElementSchema(block.Type))
		attribute := content.Attributes["to"]
		if attribute == nil {
			continue
		}
		expressions, diagnostics := hcl.ExprList(attribute.Expr)
		if diagnostics.HasErrors() {
			continue
		}
		for _, expression := range expressions {
			traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
			parts, ok := traversalParts(traversal)
			if !traversalDiagnostics.HasErrors() && ok && len(parts) == 2 && parts[0] == root {
				result[parts[1]] = true
			}
		}
	}
	return result
}

func (v *modelValidator) validateWorkflowExternality(workflowType string, subject hcl.Range, blocks hcl.Blocks) hcl.Diagnostics {
	hasExternalInput := false
	for _, block := range blocks {
		content, _, _ := syntax.PartialContent(block.Body, syntax.ElementSchema(block.Type))
		attribute := content.Attributes["from"]
		if attribute == nil {
			continue
		}
		expressions, diagnostics := hcl.ExprList(attribute.Expr)
		if diagnostics.HasErrors() {
			continue
		}
		for _, expression := range expressions {
			traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
			parts, ok := traversalParts(traversal)
			if !traversalDiagnostics.HasErrors() && ok && len(parts) == 3 && parts[0] == "event" && v.index.externalContexts[parts[1]] {
				hasExternalInput = true
			}
		}
	}
	if workflowType == "translation" && !hasExternalInput {
		return hcl.Diagnostics{errorDiagnostic(codeInvalidTranslation, subject, "Invalid translation", "a translation must consume at least one event from an external bounded_context.")}
	}
	if workflowType == "automation" && hasExternalInput {
		return hcl.Diagnostics{errorDiagnostic(codeInvalidAutomation, subject, "Invalid automation", "an automation consumes an external event; use a translation workflow for this pattern.")}
	}
	return nil
}

func (v *modelValidator) validateShelfSmell() hcl.Diagnostics {
	if len(v.index.workflows) < 2 {
		return nil
	}
	total, owner, owners := 0, "", 0
	for workflow, count := range v.index.scenarioCounts {
		total += count
		if count > 0 {
			owner = workflow
			owners++
		}
	}
	if total < 2 || owners != 1 {
		return nil
	}
	return hcl.Diagnostics{warningDiagnostic(codeShelfAntiPattern, v.index.definitionRanges[owner], "Shelf anti-pattern", fmt.Sprintf("workflow %q contains every scenario while the other workflows contain none; consider keeping scenarios with the workflow they specify.", owner))}
}

func countReferencesWithRoot(attribute *hcl.Attribute, root string) int {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return 0
	}
	count := 0
	for _, expression := range expressions {
		traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
		if !traversalDiagnostics.HasErrors() && len(traversal) > 0 && traversal.RootName() == root {
			count++
		}
	}
	return count
}
