// Package compare lists the declarations that differ between two canonical
// models. It reads only internal/model values: no parsing, no validation, and
// no I/O. Because the canonical model does not depend on file names or block
// positions, a block that moves to another member file, or a new layout, gives
// no change.
package compare

import (
	"reflect"
	"slices"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
)

// The three kinds of change.
const (
	Added   = "added"
	Removed = "removed"
	Changed = "changed"
)

// Change is one declaration that only the head has (added), only the base has
// (removed), or that both have with different canonical values (changed).
// Address is the reference form of the declaration, when the specification
// defines one. Children lists the changed elements and scenarios of a changed
// workflow.
type Change struct {
	Change   string   `json:"change"`
	Kind     string   `json:"kind"`
	Context  string   `json:"context,omitempty"`
	ID       string   `json:"id,omitempty"`
	Address  string   `json:"address,omitempty"`
	Children []Change `json:"children,omitempty"`
}

// Label names the declaration in text output: its address, or its kind and
// ID when it has no reference form, or only its kind for the board.
func (c Change) Label() string {
	switch {
	case c.Address != "":
		return c.Address
	case c.ID != "":
		return c.Kind + " " + c.ID
	default:
		return c.Kind
	}
}

// Models returns the changes from base to head. The order follows the unit
// kinds (actor, team, system, bounded_context, event, aggregate, field_type,
// workflow, chapter, hotspot, lifecycle, evidence_source, evidence, decision,
// board), then the byte order of the identity inside one kind. Identical
// models give an empty, non-nil slice.
func Models(base, head *model.Model) []Change {
	changes := []Change{}
	changes = append(changes, units(base.Actors, head.Actors, actorUnit, nil)...)
	for _, kind := range []string{"team", "system"} {
		changes = append(changes, units(owners(base, kind), owners(head, kind), ownerUnit, nil)...)
	}
	changes = append(changes, units(ownAttributes(base.Contexts), ownAttributes(head.Contexts), contextUnit, nil)...)
	changes = append(changes, units(events(base), events(head), catalogUnit[model.Event]("event"), nil)...)
	changes = append(changes, units(aggregates(base), aggregates(head), catalogUnit[model.Aggregate]("aggregate"), nil)...)
	changes = append(changes, units(fieldTypes(base), fieldTypes(head), catalogUnit[model.FieldType]("field_type"), nil)...)
	changes = append(changes, units(workflows(base), workflows(head), workflowIdentity, workflowChildren)...)
	changes = append(changes, units(base.Chapters, head.Chapters, chapterUnit, nil)...)
	changes = append(changes, units(base.Hotspots, head.Hotspots, hotspotUnit, nil)...)
	changes = append(changes, units(base.Lifecycles, head.Lifecycles, lifecycleUnit, nil)...)
	changes = append(changes, units(base.EvidenceSources, head.EvidenceSources, evidenceSourceUnit, nil)...)
	changes = append(changes, units(base.Evidence, head.Evidence, evidenceUnit, nil)...)
	changes = append(changes, units(base.Decisions, head.Decisions, decisionUnit, nil)...)
	if !slices.Equal(base.Board, head.Board) {
		changes = append(changes, Change{Change: Changed, Kind: "board"})
	}
	return changes
}

// identity returns the change template of a unit (kind, context, ID, address)
// and its key. Keys sort in the output order inside one kind.
type identity[T any] func(T) (Change, string)

// units compares two lists of one unit kind by key. children, when not nil,
// lists the child changes of a unit that both lists have.
func units[T any](base, head []T, identify identity[T], children func(base, head T) []Change) []Change {
	baseByKey := keyed(base, identify)
	headByKey := keyed(head, identify)
	keys := make([]string, 0, len(baseByKey)+len(headByKey))
	for key := range baseByKey {
		keys = append(keys, key)
	}
	for key := range headByKey {
		if _, ok := baseByKey[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	var changes []Change
	for _, key := range keys {
		before, inBase := baseByKey[key]
		after, inHead := headByKey[key]
		switch {
		case !inHead:
			change, _ := identify(before)
			change.Change = Removed
			changes = append(changes, change)
		case !inBase:
			change, _ := identify(after)
			change.Change = Added
			changes = append(changes, change)
		case !reflect.DeepEqual(before, after):
			change, _ := identify(after)
			change.Change = Changed
			if children != nil {
				change.Children = children(before, after)
			}
			changes = append(changes, change)
		}
	}
	return changes
}

func keyed[T any](items []T, identify identity[T]) map[string]T {
	byKey := make(map[string]T, len(items))
	for _, item := range items {
		_, key := identify(item)
		byKey[key] = item
	}
	return byKey
}

func actorUnit(actor model.Actor) (Change, string) {
	address := source.Address{Kind: "actor", ID: actor.ID}
	return Change{Kind: "actor", ID: actor.ID, Address: address.String()}, actor.ID
}

func ownerUnit(owner model.Owner) (Change, string) {
	address := source.Address{Kind: owner.Kind, ID: owner.ID}
	return Change{Kind: owner.Kind, ID: owner.ID, Address: address.String()}, owner.ID
}

func owners(m *model.Model, kind string) []model.Owner {
	var selected []model.Owner
	for _, owner := range m.Owners {
		if owner.Kind == kind {
			selected = append(selected, owner)
		}
	}
	return selected
}

// contextUnit compares only the attributes of the bounded context itself. Its
// events, aggregates, and field types are units of their own.
func contextUnit(context model.Context) (Change, string) {
	address := source.Address{Kind: "bounded_context", ID: context.ID}
	return Change{Kind: "bounded_context", ID: context.ID, Address: address.String()}, context.ID
}

func ownAttributes(contexts []model.Context) []model.Context {
	own := make([]model.Context, len(contexts))
	for index, context := range contexts {
		context.Aggregates, context.FieldTypes, context.Events = nil, nil, nil
		own[index] = context
	}
	return own
}

// scoped is one catalog declaration with the bounded context that owns it.
type scoped[T any] struct {
	Context string
	ID      string
	Value   T
}

func catalogUnit[T any](kind string) identity[scoped[T]] {
	return func(item scoped[T]) (Change, string) {
		address := source.Address{Kind: kind, Scope: item.Context, ID: item.ID}
		return Change{Kind: kind, Context: item.Context, ID: item.ID, Address: address.String()}, item.Context + "\x00" + item.ID
	}
}

func events(m *model.Model) []scoped[model.Event] {
	var items []scoped[model.Event]
	for _, context := range m.Contexts {
		for _, event := range context.Events {
			items = append(items, scoped[model.Event]{Context: context.ID, ID: event.ID, Value: event})
		}
	}
	return items
}

func aggregates(m *model.Model) []scoped[model.Aggregate] {
	var items []scoped[model.Aggregate]
	for _, context := range m.Contexts {
		for _, aggregate := range context.Aggregates {
			items = append(items, scoped[model.Aggregate]{Context: context.ID, ID: aggregate.ID, Value: aggregate})
		}
	}
	return items
}

func fieldTypes(m *model.Model) []scoped[model.FieldType] {
	var items []scoped[model.FieldType]
	for _, context := range m.Contexts {
		for _, fieldType := range context.FieldTypes {
			items = append(items, scoped[model.FieldType]{Context: context.ID, ID: fieldType.ID, Value: fieldType})
		}
	}
	return items
}

// workflowValue is the canonical value of one workflow: the workflow and the
// flow edges that belong to it.
type workflowValue struct {
	Workflow model.Workflow
	Edges    []model.Edge
}

func workflows(m *model.Model) []workflowValue {
	values := make([]workflowValue, 0, len(m.Workflows))
	for _, workflow := range m.Workflows {
		value := workflowValue{Workflow: workflow}
		for _, edge := range m.Edges {
			if edge.WorkflowID == workflow.ID {
				value.Edges = append(value.Edges, edge)
			}
		}
		values = append(values, value)
	}
	return values
}

func workflowIdentity(value workflowValue) (Change, string) {
	id := value.Workflow.ID
	return Change{Kind: "workflow", ID: id, Address: source.WorkflowAddress(id).String()}, id
}

// workflowChildren lists the changed elements, then the changed scenarios, of
// a workflow that both models have.
func workflowChildren(base, head workflowValue) []Change {
	workflowID := head.Workflow.ID
	elementUnit := func(element model.Element) (Change, string) {
		change := Change{Kind: string(element.Kind), ID: element.ID}
		if source.ElementHasAddress(string(element.Kind)) {
			change.Address = source.Address{Kind: string(element.Kind), Scope: workflowID, ID: element.ID}.String()
		}
		return change, string(element.Kind) + "\x00" + element.ID
	}
	scenarioUnit := func(scenario model.Scenario) (Change, string) {
		return Change{Kind: "scenario", ID: scenario.ID}, scenario.ID
	}
	children := units(base.Workflow.Elements, head.Workflow.Elements, elementUnit, nil)
	return append(children, units(base.Workflow.Scenarios, head.Workflow.Scenarios, scenarioUnit, nil)...)
}

func chapterUnit(chapter model.Chapter) (Change, string) {
	return Change{Kind: "chapter", ID: chapter.ID}, chapter.ID
}

func hotspotUnit(hotspot model.Hotspot) (Change, string) {
	address := source.Address{Kind: "hotspot", ID: hotspot.ID}
	return Change{Kind: "hotspot", ID: hotspot.ID, Address: address.String()}, hotspot.ID
}

func lifecycleUnit(lifecycle model.Lifecycle) (Change, string) {
	address := source.Address{Kind: "lifecycle", ID: lifecycle.ID}
	return Change{Kind: "lifecycle", ID: lifecycle.ID, Address: address.String()}, lifecycle.ID
}

func evidenceSourceUnit(evidenceSource model.EvidenceSource) (Change, string) {
	address := source.Address{Kind: "evidence_source", ID: evidenceSource.ID}
	return Change{Kind: "evidence_source", ID: evidenceSource.ID, Address: address.String()}, evidenceSource.ID
}

// evidenceUnit has no address: no reference names an evidence record.
func evidenceUnit(evidence model.Evidence) (Change, string) {
	return Change{Kind: "evidence", ID: evidence.ID}, evidence.ID
}

func decisionUnit(decision model.Decision) (Change, string) {
	address := source.Address{Kind: "decision", ID: decision.ID}
	return Change{Kind: "decision", ID: decision.ID, Address: address.String()}, decision.ID
}
