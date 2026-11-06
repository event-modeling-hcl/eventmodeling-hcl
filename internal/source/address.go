package source

import (
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/hashicorp/hcl/v2"
)

// Address is the reference form of one declaration, as the section
// "References and Identity" of the specification defines it. Scope is the
// owning bounded context of an event, aggregate, or field type, or the
// workflow of a workflow element. A document-wide declaration has no scope.
type Address struct {
	Kind  string
	Scope string
	ID    string
}

// String returns the dotted reference: <kind>.<id> or <kind>.<scope>.<id>.
func (a Address) String() string {
	if a.Scope == "" {
		return a.Kind + "." + a.ID
	}
	return a.Kind + "." + a.Scope + "." + a.ID
}

// WorkflowAddress returns the document-wide address of the workflow id.
func WorkflowAddress(id string) Address {
	return Address{Kind: "workflow", ID: id}
}

// ElementHasAddress reports whether a workflow element of kind has the
// workflow-qualified reference form <kind>.<workflow>.<id>. Other children of
// a workflow (screen_image, table, scenario) have no reference form.
func ElementHasAddress(kind string) bool {
	switch kind {
	case "command", "readmodel", "screen", "processor":
		return true
	}
	return false
}

// AddressAt returns the address of the innermost declaration that contains
// the start of rng and has a reference form. Every child of a bounded context
// (event, aggregate, field_type) has the form <kind>.<context>.<id>, and every
// part of a lifecycle has the form <kind>.<lifecycle>.<id>. A declaration
// without a reference form, such as a scenario, gives the address of the
// declaration around it. It reports false when no declaration with a
// reference form contains rng: a chapter, an evidence record, the board, and
// the start of the model (Document.Start), which is the subject of a
// diagnostic about the model as a whole.
func (d *Document) AddressAt(rng hcl.Range) (Address, bool) {
	if rng.Filename == "" || rng == d.parsed.Start() {
		return Address{}, false
	}
	content, _, _ := syntax.PartialContent(d.parsed.Body(), syntax.ModelSchema())
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 || !contains(block, rng) {
			continue
		}
		id := block.Labels[0]
		switch block.Type {
		case "bounded_context":
			children, _, _ := syntax.PartialContent(block.Body, syntax.BoundedContextSchema())
			if child, ok := containingChild(children, rng, everyKind); ok {
				return Address{Kind: child.Type, Scope: id, ID: child.Labels[0]}, true
			}
			return Address{Kind: block.Type, ID: id}, true
		case "actor", "team", "system", "hotspot", "decision", "evidence_source":
			return Address{Kind: block.Type, ID: id}, true
		case "lifecycle":
			children, _, _ := syntax.PartialContent(block.Body, syntax.LifecycleSchema())
			if child, ok := containingChild(children, rng, everyKind); ok {
				return Address{Kind: child.Type, Scope: id, ID: child.Labels[0]}, true
			}
			return Address{Kind: block.Type, ID: id}, true
		case "state_change", "state_view", "automation":
			children, _, _ := syntax.PartialContent(block.Body, syntax.WorkflowSchema())
			if child, ok := containingChild(children, rng, ElementHasAddress); ok {
				return Address{Kind: child.Type, Scope: id, ID: child.Labels[0]}, true
			}
			return WorkflowAddress(id), true
		}
		return Address{}, false
	}
	return Address{}, false
}

func everyKind(string) bool { return true }

// containingChild returns the labeled child block of content whose kind has a
// reference form and whose range contains the start of rng.
func containingChild(content *hcl.BodyContent, rng hcl.Range, hasAddress func(string) bool) (*hcl.Block, bool) {
	for _, child := range content.Blocks {
		if len(child.Labels) > 0 && hasAddress(child.Type) && contains(child, rng) {
			return child, true
		}
	}
	return nil, false
}

func contains(block *hcl.Block, rng hcl.Range) bool {
	extent := syntax.BlockRange(block)
	return extent.Filename == rng.Filename && extent.ContainsOffset(rng.Start.Byte)
}
