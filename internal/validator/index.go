package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

var modelIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func buildIndex(document *source.Document) (modelIndex, hcl.Diagnostics) {
	index := modelIndex{
		actors:           map[string]bool{},
		aggregates:       map[string]bool{},
		boundedContexts:  map[string]bool{},
		catalogEvents:    map[string]bool{},
		fieldTypes:       map[string]fieldTypeRef{},
		chapters:         map[string]bool{},
		hotspots:         map[string]bool{},
		systems:          map[string]bool{},
		teams:            map[string]bool{},
		workflows:        map[string]workflowIndex{},
		scenarioCounts:   map[string]int{},
		definitionRanges: map[string]hcl.Range{},
		declarations:     map[string]hcl.Range{},
		lifecycles:       map[string]lifecycleIndex{},
		evidenceSources:  map[string]bool{},
		evidence:         map[string]bool{},
		decisions:        map[string]bool{},
		identities:       map[string]string{},
		eventFields:      map[string]fieldNode{},
	}
	var diagnostics hcl.Diagnostics
	content, _, _ := syntax.PartialContent(document.Parsed().Body(), syntax.ModelSchema())
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 {
			continue
		}
		diagnostics = append(diagnostics, validateLabel(block, block.Labels[0])...)
		switch block.Type {
		case "bounded_context":
			diagnostics = append(diagnostics, index.addBoundedContext(block)...)
		case "actor":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.actors, block, "actor")...)
		case "team":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.teams, block, "team")...)
		case "system":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.systems, block, "system")...)
		case "chapter":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.chapters, block, "chapter")...)
		case "hotspot":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.hotspots, block, "hotspot")...)
		case "lifecycle":
			diagnostics = append(diagnostics, index.addLifecycle(block)...)
		case "evidence_source":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.evidenceSources, block, "evidence_source")...)
		case "evidence":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.evidence, block, "evidence")...)
		case "decision":
			diagnostics = append(diagnostics, index.addNamedSymbol(index.decisions, block, "decision")...)
		case "state_change", "state_view", "automation":
			diagnostics = append(diagnostics, index.addWorkflow(block)...)
		}
	}
	return index, diagnostics
}

func (i *modelIndex) addBoundedContext(block *hcl.Block) hcl.Diagnostics {
	contextID := block.Labels[0]
	diagnostics := i.addNamedSymbol(i.boundedContexts, block, "bounded_context")
	content, _, _ := syntax.PartialContent(block.Body, syntax.BoundedContextSchema())
	for _, child := range content.Blocks {
		if len(child.Labels) == 0 {
			continue
		}
		diagnostics = append(diagnostics, validateLabel(child, child.Labels[0])...)
		address := contextID + "." + child.Labels[0]
		switch child.Type {
		case "aggregate":
			diagnostics = append(diagnostics, i.addAddress(i.aggregates, child, "aggregate", address)...)
			i.addIdentity(contextID, address, child)
		case "event":
			diagnostics = append(diagnostics, i.addAddress(i.catalogEvents, child, "event", address)...)
			i.addEventFacts(contextID, address, child)
		case "field_type":
			if first, exists := i.declarations[declarationKey("field_type", address)]; exists {
				diagnostics = append(diagnostics, duplicateDeclarationDiagnostic(child, "field_type", address, first))
				continue
			}
			i.declarations[declarationKey("field_type", address)] = child.DefRange
			i.fieldTypes[address] = readFieldType(child.Body)
		}
	}
	return diagnostics
}

func (i *modelIndex) addWorkflow(block *hcl.Block) hcl.Diagnostics {
	id := block.Labels[0]
	if first, exists := i.definitionRanges[id]; exists {
		return hcl.Diagnostics{duplicateDeclarationDiagnostic(block, "workflow", id, first)}
	}
	workflow := workflowIndex{elements: map[string]map[string]bool{
		"command": {}, "readmodel": {}, "screen": {}, "processor": {},
		"screen_image": {}, "table": {}, "scenario": {},
	}, fields: map[string]fieldNode{}}
	content, _, _ := syntax.PartialContent(block.Body, syntax.WorkflowSchema())
	var diagnostics hcl.Diagnostics
	for _, child := range content.Blocks {
		if child.Type == "scenario" {
			i.scenarioCounts[id]++
		}
		kind, ok := workflow.elements[child.Type]
		if !ok || len(child.Labels) == 0 {
			continue
		}
		diagnostics = append(diagnostics, validateLabel(child, child.Labels[0])...)
		if kind[child.Labels[0]] {
			diagnostics = append(diagnostics, duplicateDiagnostic(child, child.Type, child.Labels[0]))
			continue
		}
		kind[child.Labels[0]] = true
		if source.ElementHasAddress(child.Type) {
			elementContent, _, _ := syntax.PartialContent(child.Body, syntax.ElementSchema(child.Type))
			workflow.fields[child.Type+"."+child.Labels[0]] = buildFieldTree(elementContent)
		}
	}
	i.workflows[id] = workflow
	i.definitionRanges[id] = block.DefRange
	return diagnostics
}

func (i *modelIndex) addNamedSymbol(symbols map[string]bool, block *hcl.Block, kind string) hcl.Diagnostics {
	return i.addAddress(symbols, block, kind, block.Labels[0])
}

func (i *modelIndex) addAddress(symbols map[string]bool, block *hcl.Block, kind, address string) hcl.Diagnostics {
	key := declarationKey(kind, address)
	if first, exists := i.declarations[key]; exists {
		return hcl.Diagnostics{duplicateDeclarationDiagnostic(block, kind, address, first)}
	}
	i.declarations[key] = block.DefRange
	symbols[address] = true
	return nil
}

func declarationKey(kind, id string) string {
	return kind + " " + id
}

// duplicateDeclarationDiagnostic reports an ID declared again after its first
// declaration at first, so that a model split across files names both places.
func duplicateDeclarationDiagnostic(block *hcl.Block, kind, id string, first hcl.Range) *hcl.Diagnostic {
	detail := fmt.Sprintf("%s %q is declared more than once in its namespace. First declared at %s:%d:%d.", kind, id, first.Filename, first.Start.Line, first.Start.Column)
	return errorDiagnostic(codeDuplicateID, block.DefRange, "Duplicate "+kind+" id", detail)
}

func duplicateDiagnostic(block *hcl.Block, kind, id string) *hcl.Diagnostic {
	return errorDiagnostic(codeDuplicateID, block.DefRange, "Duplicate "+kind+" id", fmt.Sprintf("%s %q is declared more than once in its namespace.", kind, id))
}

func validateLabel(block *hcl.Block, label string) hcl.Diagnostics {
	if modelIdentifier.MatchString(label) {
		return nil
	}
	return hcl.Diagnostics{errorDiagnostic(codeInvalidBlockLabel, block.DefRange, "Invalid block label", fmt.Sprintf("%s label %q must use lower_snake_case so it can be referenced as an HCL traversal.", block.Type, label))}
}

func readFieldType(body hcl.Body) fieldTypeRef {
	content, _, _ := syntax.PartialContent(body, syntax.FieldSchema())
	fieldType, _ := literalString(content.Attributes["type"])
	cardinality, _ := literalString(content.Attributes["cardinality"])
	return fieldTypeRef{fieldType: fieldType, cardinality: cardinality}
}

// addLifecycle records a lifecycle and the IDs of its parts. State IDs have
// one space in each lifecycle. Transition and invalid transition IDs share
// another space in each lifecycle.
func (i *modelIndex) addLifecycle(block *hcl.Block) hcl.Diagnostics {
	id := block.Labels[0]
	key := declarationKey("lifecycle", id)
	if first, exists := i.declarations[key]; exists {
		return hcl.Diagnostics{duplicateDeclarationDiagnostic(block, "lifecycle", id, first)}
	}
	i.declarations[key] = block.DefRange
	lifecycle := lifecycleIndex{states: map[string]bool{}, transitions: map[string]bool{}, invalid: map[string]bool{}}
	content, _, _ := syntax.PartialContent(block.Body, syntax.LifecycleSchema())
	var diagnostics hcl.Diagnostics
	for _, child := range content.Blocks {
		if len(child.Labels) == 0 {
			continue
		}
		label := child.Labels[0]
		diagnostics = append(diagnostics, validateLabel(child, label)...)
		space, kind := lifecycle.states, "state"
		if child.Type != "state" {
			space, kind = lifecycle.transitions, "transition"
		}
		if space[label] {
			diagnostics = append(diagnostics, duplicateDiagnostic(child, kind, label))
			continue
		}
		space[label] = true
		if child.Type == "invalid_transition" {
			lifecycle.invalid[label] = true
		}
	}
	i.lifecycles[id] = lifecycle
	return diagnostics
}

// addIdentity records the identity field type of an aggregate when the
// identity is a field_type traversal. The validator reports other values.
func (i *modelIndex) addIdentity(contextID, address string, block *hcl.Block) {
	content, _, _ := syntax.PartialContent(block.Body, syntax.AggregateSchema())
	identity := content.Attributes["identity"]
	if identity == nil {
		return
	}
	parts, diagnostics := source.ReferenceParts(identity.Expr)
	if diagnostics.HasErrors() || len(parts) < 2 || parts[0] != "field_type" {
		return
	}
	if fieldType := qualifiedAddress(contextID, parts); fieldType != "" {
		i.identities[address] = fieldType
	}
}

// addEventFacts records the declared aggregate, the top-level field types,
// and the field tree of one catalog event.
func (i *modelIndex) addEventFacts(contextID, address string, block *hcl.Block) {
	content, _, _ := syntax.PartialContent(block.Body, syntax.EventSchema())
	facts := eventFacts{address: address, definition: block.DefRange, fieldTypes: map[string]bool{}}
	if aggregate := content.Attributes["aggregate"]; aggregate != nil {
		if parts, diagnostics := source.ReferenceParts(aggregate.Expr); !diagnostics.HasErrors() && len(parts) > 1 && parts[0] == "aggregate" {
			facts.aggregate = qualifiedAddress(contextID, parts)
		}
	}
	if fields := content.Attributes["fields"]; fields != nil {
		for _, reference := range source.ReferenceList(fields.Expr) {
			facts.fieldTypes[fieldTypeKey(contextID, reference)] = true
		}
	}
	for _, field := range content.Blocks {
		fieldContent, _, _ := syntax.PartialContent(field.Body, syntax.FieldSchema())
		if typeAttribute := fieldContent.Attributes["type"]; typeAttribute != nil {
			if reference := source.ReferenceString(typeAttribute.Expr); reference != "" {
				facts.fieldTypes[fieldTypeKey(contextID, reference)] = true
			}
		}
	}
	i.events = append(i.events, facts)
	i.eventFields[address] = buildFieldTree(content)
}

// qualifiedAddress returns <context>.<id> for a local (<kind>.<id>) or
// qualified (<kind>.<context>.<id>) catalog reference, or "" for another
// shape.
func qualifiedAddress(contextID string, parts []string) string {
	switch len(parts) {
	case 2:
		return contextID + "." + parts[1]
	case 3:
		return parts[1] + "." + parts[2]
	}
	return ""
}

// fieldTypeKey returns the <context>.<id> address that a field_type reference
// names, or "" when reference is not a field_type traversal.
func fieldTypeKey(contextID, reference string) string {
	parts := strings.Split(reference, ".")
	if parts[0] != "field_type" {
		return ""
	}
	return qualifiedAddress(contextID, parts)
}

// buildFieldTree returns the declared fields of an element body: the names of
// its fields list entries and its field blocks, with the subfield blocks of
// each field block.
func buildFieldTree(content *hcl.BodyContent) fieldNode {
	tree := fieldNode{}
	if attribute := content.Attributes["fields"]; attribute != nil {
		for _, reference := range source.ReferenceList(attribute.Expr) {
			tree[reference[strings.LastIndex(reference, ".")+1:]] = fieldNode{}
		}
	}
	for _, block := range content.Blocks {
		if (block.Type != "field" && block.Type != "subfield") || len(block.Labels) == 0 {
			continue
		}
		if _, exists := tree[block.Labels[0]]; exists {
			continue
		}
		fieldContent, _, _ := syntax.PartialContent(block.Body, syntax.FieldSchema())
		tree[block.Labels[0]] = buildFieldTree(fieldContent)
	}
	return tree
}
