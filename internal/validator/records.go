package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// attachKinds are the root kinds that every record (hotspot, evidence,
// decision) accepts in on. A field reference has the root kind of its element.
var attachKinds = []string{
	"actor", "bounded_context", "system", "team", "workflow",
	"event", "aggregate", "field_type",
	"command", "readmodel", "screen", "processor",
	"lifecycle", "state", "transition", "invalid_transition",
}

// recordAttachKinds returns the attach target kinds of one record kind.
func recordAttachKinds(record string) []string {
	switch record {
	case "hotspot":
		return append(slices.Clone(attachKinds), "decision")
	case "evidence":
		return append(slices.Clone(attachKinds), "decision", "hotspot")
	}
	return attachKinds
}

// validateAttachTarget checks the on attribute of a record. A target kind
// that the record does not accept, a malformed form, or an index step is
// EM101. An accepted target that is not declared is EM102.
func (v *modelValidator) validateAttachTarget(attribute *hcl.Attribute, record string) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok {
		return hcl.Diagnostics{invalidReferenceShape(attribute, "on names a target with names only. It has no index or row selector.")}
	}
	if !slices.Contains(recordAttachKinds(record), parts[0]) {
		return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("%s on cannot name a %s. You referenced: %s.", record, parts[0], strings.Join(parts, ".")))}
	}
	if element, path, isField := source.SplitFieldReference(parts); isField && len(element) == 3 {
		_, tree, declared := v.fieldTree(element, "")
		if !declared || !fieldPathExists(tree, path) {
			return hcl.Diagnostics{unresolvedReference(attribute, parts)}
		}
		return nil
	}
	switch parts[0] {
	case "lifecycle", "decision", "hotspot":
		if len(parts) != 2 {
			return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("a %s reference has the form %s.<id>.", parts[0], parts[0]))}
		}
		if !v.recordExists(parts[0], parts[1]) {
			return hcl.Diagnostics{unresolvedReference(attribute, parts)}
		}
		return nil
	case "state", "transition", "invalid_transition":
		if len(parts) != 3 {
			return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("outside its lifecycle, a %s reference has the form %s.<lifecycle>.<id>.", parts[0], parts[0]))}
		}
		if !v.lifecyclePartExists(parts[0], parts[1], parts[2]) {
			return hcl.Diagnostics{unresolvedReference(attribute, parts)}
		}
		return nil
	}
	if !v.referenceExists(parts, "") {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

func (v *modelValidator) recordExists(kind, id string) bool {
	switch kind {
	case "lifecycle":
		_, ok := v.index.lifecycles[id]
		return ok
	case "decision":
		return v.index.decisions[id]
	case "hotspot":
		return v.index.hotspots[id]
	case "evidence_source":
		return v.index.evidenceSources[id]
	}
	return false
}

func (v *modelValidator) lifecyclePartExists(kind, lifecycleID, id string) bool {
	lifecycle, ok := v.index.lifecycles[lifecycleID]
	if !ok {
		return false
	}
	switch kind {
	case "state":
		return lifecycle.states[id]
	case "transition":
		return lifecycle.transitions[id] && !lifecycle.invalid[id]
	case "invalid_transition":
		return lifecycle.invalid[id]
	}
	return false
}

// validateAggregateBlock checks an aggregate and its optional identity, a
// field_type traversal in the local or the qualified form.
func (v *contextValidator) validateAggregateBlock(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.AggregateSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, aggregateRule)...)
	identity := content.Attributes["identity"]
	if identity == nil {
		return diagnostics
	}
	if _, traversalDiagnostics := absoluteTraversal(identity); traversalDiagnostics.HasErrors() {
		return append(diagnostics, invalidReferenceShape(identity, "identity must be a field_type.<id> or field_type.<context>.<id> traversal."))
	}
	_, referenceDiagnostics := v.fieldTypeAddress(identity)
	return append(diagnostics, referenceDiagnostics...)
}

// validateIdentityFields reports each event that declares an aggregate with
// an identity and has no top-level field of the identity field type (EM018).
func (v *modelValidator) validateIdentityFields() hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, event := range v.index.events {
		identity, ok := v.index.identities[event.aggregate]
		if !ok || !v.index.aggregates[event.aggregate] {
			continue
		}
		if _, declared := v.index.fieldTypes[identity]; !declared || event.fieldTypes[identity] {
			continue
		}
		diagnostics = append(diagnostics, errorDiagnostic(codeMissingIdentityField, event.definition, "Missing identity field", fmt.Sprintf("event %s declares aggregate.%s, whose identity is field_type.%s, but has no top-level field of that type.", event.address, event.aggregate, identity)))
	}
	return diagnostics
}

// validateLifecycle checks one lifecycle: its references, its parts, and the
// consistency rules of EM017.
func (v *modelValidator) validateLifecycle(block *hcl.Block) hcl.Diagnostics {
	id := block.Labels[0]
	schema := syntax.LifecycleSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, lifecycleRule)...)
	aggregate := ""
	if attribute := content.Attributes["aggregate"]; attribute != nil {
		parts, referenceDiagnostics := v.qualifiedReference(attribute.Expr, "aggregate", "aggregate.<context>.<id>")
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if parts != nil {
			aggregate = parts[1] + "." + parts[2]
			if first, exists := v.lifecycleAggregates[aggregate]; exists {
				diagnostics = append(diagnostics, errorDiagnostic(codeInvalidLifecycle, attribute.Expr.Range(), "Invalid lifecycle", fmt.Sprintf("lifecycle.%s already states the lifecycle of aggregate.%s.", first, aggregate)))
			} else {
				v.lifecycleAggregates[aggregate] = id
			}
		}
	}
	states := 0
	valid := map[string]bool{}
	type invalidPair struct {
		key     string
		subject hcl.Range
	}
	var invalid []invalidPair
	for _, child := range content.Blocks {
		switch child.Type {
		case "state":
			states++
			diagnostics = append(diagnostics, validateSimpleBlock(child.Body, syntax.StateSchema(), stateRule)...)
		case "transition":
			childSchema := syntax.TransitionSchema()
			childContent, childDiagnostics := syntax.Content(child.Body, childSchema)
			diagnostics = append(diagnostics, childDiagnostics...)
			diagnostics = append(diagnostics, validateLiteralAttributes(childContent.Attributes, childSchema.Attributes, transitionRule)...)
			before, beforeDiagnostics := v.stateList(childContent.Attributes["before"], id)
			diagnostics = append(diagnostics, beforeDiagnostics...)
			if after := childContent.Attributes["after"]; after != nil {
				diagnostics = append(diagnostics, v.stateReference(after.Expr, id)...)
			}
			command, commandDiagnostics := v.lifecycleCommand(childContent.Attributes["command"])
			diagnostics = append(diagnostics, commandDiagnostics...)
			diagnostics = append(diagnostics, v.transitionEvents(childContent.Attributes["events"], aggregate)...)
			for _, state := range before {
				valid[state+"\x00"+command] = true
			}
		case "invalid_transition":
			childSchema := syntax.InvalidTransitionSchema()
			childContent, childDiagnostics := syntax.Content(child.Body, childSchema)
			diagnostics = append(diagnostics, childDiagnostics...)
			diagnostics = append(diagnostics, validateLiteralAttributes(childContent.Attributes, childSchema.Attributes, transitionRule)...)
			diagnostics = append(diagnostics, nonEmptyText(childContent.Attributes["reason"])...)
			beforeAttribute := childContent.Attributes["before"]
			before, beforeDiagnostics := v.stateList(beforeAttribute, id)
			diagnostics = append(diagnostics, beforeDiagnostics...)
			if beforeAttribute != nil && !beforeDiagnostics.HasErrors() && len(before) == 0 {
				diagnostics = append(diagnostics, errorDiagnostic(codeInvalidLifecycle, beforeAttribute.Expr.Range(), "Invalid lifecycle", fmt.Sprintf("invalid_transition %q must name at least one state in before.", child.Labels[0])))
			}
			command, commandDiagnostics := v.lifecycleCommand(childContent.Attributes["command"])
			diagnostics = append(diagnostics, commandDiagnostics...)
			for _, state := range before {
				invalid = append(invalid, invalidPair{key: state + "\x00" + command, subject: child.DefRange})
			}
		}
	}
	if states == 0 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidLifecycle, block.DefRange, "Invalid lifecycle", fmt.Sprintf("lifecycle %q must declare at least one state.", id)))
	}
	for _, pair := range invalid {
		if valid[pair.key] && !strings.HasSuffix(pair.key, "\x00") {
			state, command, _ := strings.Cut(pair.key, "\x00")
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidLifecycle, pair.subject, "Invalid lifecycle", fmt.Sprintf("state.%s and %s appear in a transition and in an invalid_transition.", state, command)))
		}
	}
	return diagnostics
}

// qualifiedReference resolves a three-segment catalog or workflow-qualified
// reference of kind. It returns nil parts with a diagnostic when the form is
// wrong (EM101) or the target is not declared (EM102).
func (v *modelValidator) qualifiedReference(expression hcl.Expression, kind, form string) ([]string, hcl.Diagnostics) {
	attribute := &hcl.Attribute{Expr: expression}
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) != 3 || parts[0] != kind {
		return nil, hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("reference must have the form %s.", form))}
	}
	if !v.referenceExists(parts, "") {
		return nil, hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return parts, nil
}

// lifecycleCommand resolves the command of a transition. It returns the
// qualified command reference, or "" when it does not resolve.
func (v *modelValidator) lifecycleCommand(attribute *hcl.Attribute) (string, hcl.Diagnostics) {
	if attribute == nil {
		return "", nil
	}
	parts, diagnostics := v.qualifiedReference(attribute.Expr, "command", "command.<workflow>.<id>")
	if parts == nil {
		return "", diagnostics
	}
	return strings.Join(parts, "."), nil
}

// stateList resolves a list of state.<id> references inside the lifecycle
// lifecycleID and returns the IDs that resolve.
func (v *modelValidator) stateList(attribute *hcl.Attribute, lifecycleID string) ([]string, hcl.Diagnostics) {
	if attribute == nil {
		return nil, nil
	}
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	states := make([]string, 0, len(expressions))
	for _, expression := range expressions {
		stateDiagnostics := v.stateReference(expression, lifecycleID)
		diagnostics = append(diagnostics, stateDiagnostics...)
		if !stateDiagnostics.HasErrors() {
			states = append(states, strings.TrimPrefix(traversalString(expression), "state."))
		}
	}
	return states, diagnostics
}

// stateReference resolves one state.<id> reference inside its lifecycle.
func (v *modelValidator) stateReference(expression hcl.Expression, lifecycleID string) hcl.Diagnostics {
	attribute := &hcl.Attribute{Expr: expression}
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) != 2 || parts[0] != "state" {
		return hcl.Diagnostics{invalidReferenceShape(attribute, "inside a lifecycle, a state reference has the form state.<id>.")}
	}
	if !v.index.lifecycles[lifecycleID].states[parts[1]] {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

// transitionEvents resolves the events of a transition. The list must not be
// empty, and an event that declares an aggregate must declare the aggregate
// of the lifecycle (EM017).
func (v *modelValidator) transitionEvents(attribute *hcl.Attribute, aggregate string) hcl.Diagnostics {
	if attribute == nil {
		return nil
	}
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	if len(expressions) == 0 {
		return hcl.Diagnostics{errorDiagnostic(codeInvalidLifecycle, attribute.Expr.Range(), "Invalid lifecycle", "a transition must name at least one event.")}
	}
	for _, expression := range expressions {
		parts, referenceDiagnostics := v.qualifiedReference(expression, "event", "event.<context>.<id>")
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if parts == nil || aggregate == "" {
			continue
		}
		address := parts[1] + "." + parts[2]
		for _, event := range v.index.events {
			if event.address == address && event.aggregate != "" && event.aggregate != aggregate {
				diagnostics = append(diagnostics, errorDiagnostic(codeInvalidLifecycle, expression.Range(), "Invalid lifecycle", fmt.Sprintf("event.%s declares aggregate.%s, not the aggregate of the lifecycle, aggregate.%s.", address, event.aggregate, aggregate)))
			}
		}
	}
	return diagnostics
}

func traversalString(expression hcl.Expression) string {
	traversal, diagnostics := hcl.AbsTraversalForExpr(expression)
	if diagnostics.HasErrors() {
		return ""
	}
	parts, _ := traversalParts(traversal)
	return strings.Join(parts, ".")
}

// validateEvidence checks one evidence record. A stated or implied record
// must name its source.
func (v *modelValidator) validateEvidence(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.EvidenceSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, evidenceRule)...)
	if on := content.Attributes["on"]; on != nil {
		diagnostics = append(diagnostics, v.validateAttachTarget(on, "evidence")...)
	}
	source := content.Attributes["source"]
	if source != nil {
		diagnostics = append(diagnostics, v.validateRecordReference(source, "evidence_source")...)
	}
	if certainty, ok := literalString(content.Attributes["certainty"]); ok && (certainty == "stated" || certainty == "implied") && source == nil {
		diagnostics = append(diagnostics, errorDiagnostic(codeUnclassified, content.MissingItemRange, "Missing required argument", fmt.Sprintf("evidence with certainty = %q must set source.", certainty)))
	}
	return diagnostics
}

// validateRecordReference resolves a <kind>.<id> reference to a top-level
// record.
func (v *modelValidator) validateRecordReference(attribute *hcl.Attribute, kind string) hcl.Diagnostics {
	traversal, diagnostics := absoluteTraversal(attribute)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	parts, ok := traversalParts(traversal)
	if !ok || len(parts) != 2 || parts[0] != kind {
		return hcl.Diagnostics{invalidReferenceShape(attribute, fmt.Sprintf("reference must target %s.<id>.", kind))}
	}
	if !v.recordExists(kind, parts[1]) {
		return hcl.Diagnostics{unresolvedReference(attribute, parts)}
	}
	return nil
}

// validateDecision checks one decision record. signal is allowed only on an
// accepted debt. The effective kind is "choice" when kind is omitted.
func (v *modelValidator) validateDecision(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.DecisionSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, decisionRule)...)
	diagnostics = append(diagnostics, nonEmptyText(content.Attributes["statement"])...)
	diagnostics = append(diagnostics, nonEmptyText(content.Attributes["reason"])...)
	if on := content.Attributes["on"]; on != nil {
		diagnostics = append(diagnostics, v.validateAttachTarget(on, "decision")...)
	}
	signal := content.Attributes["signal"]
	if signal == nil {
		return diagnostics
	}
	kind := "choice"
	if attribute := content.Attributes["kind"]; attribute != nil {
		value, ok := literalString(attribute)
		if !ok || !slices.Contains([]string{"choice", "accepted_debt"}, value) {
			return diagnostics
		}
		kind = value
	}
	if kind != "accepted_debt" {
		diagnostics = append(diagnostics, errorDiagnostic(codeUnclassified, signal.NameRange, "Unexpected argument", fmt.Sprintf("signal is allowed only on a decision with kind = \"accepted_debt\". The effective kind is %q.", kind)))
	}
	return diagnostics
}
