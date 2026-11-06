package validator

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// fieldReference is one resolved field reference: the qualified key of the
// element (event.<context>.<id> or <kind>.<workflow>.<id>) and the field
// names from the top-level field to the last subfield.
type fieldReference struct {
	element string
	path    []string
}

func (r fieldReference) key() string {
	return r.element + "\x00" + strings.Join(r.path, "\x00")
}

// fieldTree returns the qualified key, the field tree, and the presence of the
// element that element names. workflowID resolves the local form.
func (v *modelValidator) fieldTree(element []string, workflowID string) (string, fieldNode, bool) {
	if element[0] == "event" {
		address := element[1] + "." + element[2]
		return "event." + address, v.index.eventFields[address], v.index.catalogEvents[address]
	}
	kind, id := element[0], element[len(element)-1]
	if len(element) == 3 {
		workflowID = element[1]
	}
	workflow, ok := v.index.workflows[workflowID]
	key := kind + "." + workflowID + "." + id
	if !ok || !workflow.elements[kind][id] {
		return key, nil, false
	}
	return key, workflow.fields[kind+"."+id], true
}

// fieldPathExists reports whether each name of path is a declared field, then
// a subfield of the preceding field or subfield. The lookup is exact.
func fieldPathExists(tree fieldNode, path []string) bool {
	for _, name := range path {
		next, ok := tree[name]
		if !ok {
			return false
		}
		tree = next
	}
	return true
}

// workflowFieldReference resolves a field reference of a field_source in the
// workflow workflowID. An element of the workflow uses the local form, or the
// qualified form of the same workflow. An event uses its qualified form.
func (v *modelValidator) workflowFieldReference(expression hcl.Expression, workflowID string) (*fieldReference, hcl.Diagnostics) {
	invalid := func(detail string) (*fieldReference, hcl.Diagnostics) {
		return nil, hcl.Diagnostics{errorDiagnostic(codeInvalidReference, expression.Range(), "Invalid reference", detail)}
	}
	traversal, diagnostics := source.AbsoluteTraversal(expression)
	if diagnostics.HasErrors() {
		return invalid("a field reference must be a traversal such as command.<id>.field.<name>.")
	}
	parts, ok := traversalParts(traversal)
	if !ok {
		return invalid("a field reference names fields only. It has no index or row selector.")
	}
	element, path, ok := source.SplitFieldReference(parts)
	if !ok {
		return invalid(fmt.Sprintf("a field reference must have the form event.<context>.<id>.field.<name> or <kind>.<id>.field.<name>, with one .field.<name> for each subfield. You referenced: %s.", strings.Join(parts, ".")))
	}
	if element[0] != "event" && len(element) == 3 && element[1] != workflowID {
		return invalid(fmt.Sprintf("a field_source may reference only elements of its own workflow %q. You referenced: %s.", workflowID, strings.Join(parts, ".")))
	}
	key, tree, declared := v.fieldTree(element, workflowID)
	if !declared || !fieldPathExists(tree, path) {
		return nil, hcl.Diagnostics{errorDiagnostic(codeUnresolvedReference, expression.Range(), "Unresolved reference", fmt.Sprintf("%s is not declared in the model.", strings.Join(parts, ".")))}
	}
	return &fieldReference{element: key, path: path}, nil
}

// flowEdge is one flow edge of a workflow between qualified element keys.
type flowEdge struct {
	from, to string
}

// workflowEdges returns the flow edges that the elements of one workflow
// declare, with qualified keys at both ends.
func (v *modelValidator) workflowEdges(workflowID string, blocks hcl.Blocks) map[flowEdge]bool {
	edges := map[flowEdge]bool{}
	for _, block := range blocks {
		if !source.ElementHasAddress(block.Type) || len(block.Labels) == 0 {
			continue
		}
		owner := block.Type + "." + workflowID + "." + block.Labels[0]
		content, _, _ := syntax.PartialContent(block.Body, syntax.ElementSchema(block.Type))
		for _, direction := range []string{"to", "from"} {
			attribute := content.Attributes[direction]
			if attribute == nil {
				continue
			}
			for _, reference := range source.ReferenceList(attribute.Expr) {
				other := nodeKey(strings.Split(reference, "."), workflowID)
				if direction == "to" {
					edges[flowEdge{from: owner, to: other}] = true
				} else {
					edges[flowEdge{from: other, to: owner}] = true
				}
			}
		}
	}
	return edges
}

// nodeKey returns the qualified key of a flow reference: event.<context>.<id>
// for an event, and <kind>.<workflow>.<id> for a workflow element.
func nodeKey(parts []string, workflowID string) string {
	if len(parts) == 2 {
		return parts[0] + "." + workflowID + "." + parts[1]
	}
	return strings.Join(parts, ".")
}

// fieldSourceTargets holds the resolved targets of the field sources of one
// workflow.
type fieldSourceTargets map[string]bool

// validateFieldSource checks one field_source block of the workflow
// workflowID against the flow edges of that workflow (EM202).
func (v *modelValidator) validateFieldSource(workflowID string, block *hcl.Block, edges map[flowEdge]bool, targets fieldSourceTargets) hcl.Diagnostics {
	schema := syntax.FieldSourceSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, fieldSourceRule)...)
	var target *fieldReference
	if attribute := content.Attributes["target"]; attribute != nil {
		var targetDiagnostics hcl.Diagnostics
		target, targetDiagnostics = v.workflowFieldReference(attribute.Expr, workflowID)
		diagnostics = append(diagnostics, targetDiagnostics...)
		if target != nil {
			if targets[target.key()] {
				diagnostics = append(diagnostics, errorDiagnostic(codeInvalidFieldSource, attribute.Expr.Range(), "Invalid field source", fmt.Sprintf("another field_source of workflow %q already names this target.", workflowID)))
			}
			targets[target.key()] = true
		}
	}
	attribute := content.Attributes["inputs"]
	if attribute == nil {
		return diagnostics
	}
	expressions, listDiagnostics := hcl.ExprList(attribute.Expr)
	if listDiagnostics.HasErrors() {
		return append(diagnostics, listDiagnostics...)
	}
	inputs := make([]*fieldReference, len(expressions))
	resolved := true
	for index, expression := range expressions {
		input, inputDiagnostics := v.workflowFieldReference(expression, workflowID)
		diagnostics = append(diagnostics, inputDiagnostics...)
		inputs[index] = input
		resolved = resolved && input != nil
	}
	if target == nil || !resolved {
		return diagnostics
	}
	if len(expressions) == 0 {
		return append(diagnostics, errorDiagnostic(codeInvalidFieldSource, attribute.Expr.Range(), "Invalid field source", "inputs must name at least one field."))
	}
	if kind := target.element[:strings.Index(target.element, ".")]; (kind == "event" || kind == "command") && len(expressions) > 1 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidFieldSource, attribute.Expr.Range(), "Invalid field source", fmt.Sprintf("a field of a %s has exactly one input. Found: %d.", kind, len(expressions))))
	}
	for index, input := range inputs {
		if !edges[flowEdge{from: input.element, to: target.element}] {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidFieldSource, expressions[index].Range(), "Invalid field source", fmt.Sprintf("%s does not connect to %s by one canonical edge of workflow %q.", input.element, target.element, workflowID)))
		}
	}
	return diagnostics
}
