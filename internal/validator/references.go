package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

func (v *modelValidator) validateOwnerReference(attribute *hcl.Attribute) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) != 2 || !slices.Contains([]string{"bounded_context", "team", "system"}, parts[0]) {
		return hcl.Diagnostics{invalidReferenceShape(attribute, "owner must reference bounded_context.<id>, team.<id>, or system.<id>.")}
	}
	if !v.referenceExists(parts, "") {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

func (v *modelValidator) validateGlobalReference(attribute *hcl.Attribute) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || !v.referenceExists(parts, "") {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

func (v *modelValidator) validateReference(attribute *hcl.Attribute, expectedKind, workflowID string) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) < 2 || parts[0] != expectedKind {
		return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("reference must target %s.<id>.", expectedKind))}
	}
	if !v.referenceExists(parts, workflowID) {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

func (v *modelValidator) validateReferenceList(attribute *hcl.Attribute, expectedKind, workflowID string) hcl.Diagnostics {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	for _, expression := range expressions {
		traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
		diagnostics = append(diagnostics, traversalDiagnostics...)
		if traversalDiagnostics.HasErrors() {
			continue
		}
		parts, ok := traversalParts(traversal)
		if !ok || len(parts) < 2 || parts[0] != expectedKind {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidReference, expression.Range(), "Invalid reference", fmt.Sprintf("reference must target %s.<id>.", expectedKind)))
			continue
		}
		if !v.referenceExists(parts, workflowID) {
			diagnostics = append(diagnostics, errorDiagnostic(codeUnresolvedReference, expression.Range(), "Unresolved reference", fmt.Sprintf("%s is not declared in the model.", strings.Join(parts, "."))))
		}
	}
	return diagnostics
}

func (v *modelValidator) validateChapterRange(attribute *hcl.Attribute) hcl.Diagnostics {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() || len(expressions) < 2 {
		return diagnostics
	}
	positions := make(map[string]int, len(v.index.workflowOrder))
	for index, workflow := range v.index.workflowOrder {
		positions[workflow] = index
	}
	previous := -1
	for _, expression := range expressions {
		traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
		if traversalDiagnostics.HasErrors() {
			continue
		}
		parts, ok := traversalParts(traversal)
		if !ok || len(parts) != 2 || parts[0] != "workflow" {
			continue
		}
		position, exists := positions[parts[1]]
		if !exists {
			continue
		}
		if previous >= 0 && position != previous+1 {
			return hcl.Diagnostics{errorDiagnostic(codeInvalidChapterRange, attribute.Expr.Range(), "Invalid chapter range", "chapter workflows must be contiguous and listed in source order.")}
		}
		previous = position
	}
	return nil
}

func (v *modelValidator) validateFlowReferences(attribute *hcl.Attribute, workflowType, workflowID, ownerKind, direction string) hcl.Diagnostics {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	allowedRoots := allowedFlowRoots(workflowType, ownerKind, direction)
	for _, expression := range expressions {
		traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
		diagnostics = append(diagnostics, traversalDiagnostics...)
		if traversalDiagnostics.HasErrors() {
			continue
		}
		parts, ok := traversalParts(traversal)
		if !ok || len(parts) < 2 || !slices.Contains(allowedRoots, parts[0]) {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidFlowReference, expression.Range(), "Invalid flow reference", flowReferenceDetail(ownerKind, direction, workflowType, allowedRoots, parts)))
			continue
		}
		if !v.referenceExists(parts, workflowID) {
			diagnostics = append(diagnostics, errorDiagnostic(codeUnresolvedReference, expression.Range(), "Unresolved reference", fmt.Sprintf("%s is not declared in the model.", strings.Join(parts, "."))))
		}
	}
	return diagnostics
}

func flowReferenceDetail(ownerKind, direction, workflowType string, allowedRoots, referenced []string) string {
	actual := strings.Join(referenced, ".")
	if len(allowedRoots) == 0 {
		return fmt.Sprintf("%s.%s has no canonical flow targets in a %s workflow. You referenced: %s.", ownerKind, direction, workflowType, actual)
	}
	expected := make([]string, 0, len(allowedRoots))
	for _, root := range allowedRoots {
		expected = append(expected, flowReferenceTargetShape(root))
	}
	return fmt.Sprintf("%s.%s may reference: %s. You referenced: %s.", ownerKind, direction, strings.Join(expected, " or "), actual)
}

func flowReferenceTargetShape(root string) string {
	if root == "event" {
		return "event.<context>.<id>"
	}
	return root + ".<id>"
}

func allowedFlowRoots(workflowType, ownerKind, direction string) []string {
	switch workflowType {
	case "state_change":
		switch {
		case ownerKind == "screen" && direction == "to":
			return []string{"command"}
		case ownerKind == "command" && direction == "to":
			return []string{"event"}
		}
	case "state_view":
		switch {
		case ownerKind == "readmodel" && direction == "from":
			return []string{"event"}
		case ownerKind == "readmodel" && direction == "to":
			return []string{"screen"}
		}
	case "automation", "translation":
		switch {
		case ownerKind == "readmodel" && direction == "from":
			return []string{"event"}
		case ownerKind == "readmodel" && direction == "to":
			return []string{"processor"}
		case ownerKind == "processor" && direction == "from":
			return []string{"event"}
		case ownerKind == "processor" && direction == "to":
			return []string{"command"}
		case ownerKind == "command" && direction == "to":
			return []string{"event"}
		}
	}
	return nil
}

func (v *modelValidator) referenceExists(parts []string, workflowID string) bool {
	if len(parts) == 2 {
		switch parts[0] {
		case "actor":
			return v.index.actors[parts[1]]
		case "bounded_context":
			return v.index.boundedContexts[parts[1]]
		case "system":
			return v.index.systems[parts[1]]
		case "team":
			return v.index.teams[parts[1]]
		case "workflow":
			_, ok := v.index.workflows[parts[1]]
			return ok
		case "command", "readmodel", "screen", "processor":
			workflow, ok := v.index.workflows[workflowID]
			return ok && workflow.elements[parts[0]][parts[1]]
		}
	}
	if len(parts) == 3 {
		address := parts[1] + "." + parts[2]
		switch parts[0] {
		case "event":
			return v.index.catalogEvents[address]
		case "aggregate":
			return v.index.aggregates[address]
		case "field_type":
			_, ok := v.index.fieldTypes[address]
			return ok
		case "command", "readmodel", "screen", "processor":
			workflow, ok := v.index.workflows[parts[1]]
			return ok && workflow.elements[parts[0]][parts[2]]
		}
	}
	return false
}

func (v *contextValidator) validateAggregate(attribute *hcl.Attribute, allowLocal bool) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || parts[0] != "aggregate" || !validLocalOrQualifiedLength(parts, allowLocal) {
		return hcl.Diagnostics{invalidReferenceShape(attribute, "aggregate must be an aggregate.<context>.<name> reference; context-owned events may use aggregate.<name>.")}
	}
	address := ""
	if len(parts) == 2 {
		address = v.contextID + "." + parts[1]
	} else {
		address = parts[1] + "." + parts[2]
	}
	if !v.model.index.aggregates[address] {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

func validLocalOrQualifiedLength(parts []string, allowLocal bool) bool {
	return len(parts) == 3 || allowLocal && len(parts) == 2
}

func (v *contextValidator) validateAggregateList(attribute *hcl.Attribute, allowLocal bool) hcl.Diagnostics {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	for _, expression := range expressions {
		diagnostics = append(diagnostics, v.validateAggregate(&hcl.Attribute{Expr: expression}, allowLocal)...)
	}
	return diagnostics
}

func (v *contextValidator) fieldTypeAddress(attribute *hcl.Attribute) (string, hcl.Diagnostics) {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return "", diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || parts[0] != "field_type" || len(parts) != 2 && len(parts) != 3 {
		return "", hcl.Diagnostics{invalidReferenceShape(attribute, "field type must be a field_type.<context>.<name> reference; context-owned fields may use field_type.<name>.")}
	}
	address := ""
	if len(parts) == 2 {
		if v.contextID == "" {
			return "", hcl.Diagnostics{invalidReferenceShape(attribute, "workflow fields must use field_type.<context>.<name>.")}
		}
		address = v.contextID + "." + parts[1]
	} else {
		address = parts[1] + "." + parts[2]
	}
	if _, exists := v.model.index.fieldTypes[address]; !exists {
		return address, hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return address, nil
}

func absoluteTraversal(attribute *hcl.Attribute) (hcl.Traversal, hcl.Diagnostics) {
	return hcl.AbsTraversalForExpr(attribute.Expr)
}

func traversalParts(traversal hcl.Traversal) ([]string, bool) {
	if len(traversal) == 0 {
		return nil, false
	}
	parts := []string{traversal.RootName()}
	for _, step := range traversal[1:] {
		attribute, ok := step.(hcl.TraverseAttr)
		if !ok {
			return nil, false
		}
		parts = append(parts, attribute.Name)
	}
	return parts, true
}

func invalidReferenceShape(attribute *hcl.Attribute, detail string) *hcl.Diagnostic {
	return errorDiagnostic(codeInvalidReference, attribute.Expr.Range(), "Invalid reference", detail)
}

func unresolvedReference(attribute *hcl.Attribute, parts []string) *hcl.Diagnostic {
	return errorDiagnostic(codeUnresolvedReference, attribute.Expr.Range(), "Unresolved reference", fmt.Sprintf("%s is not declared in the model.", strings.Join(parts, ".")))
}
