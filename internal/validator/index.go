package validator

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

var modelIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func buildIndex(body hcl.Body) (modelIndex, hcl.Diagnostics) {
	index := modelIndex{
		actors:           map[string]bool{},
		aggregates:       map[string]bool{},
		boundedContexts:  map[string]bool{},
		catalogEvents:    map[string]bool{},
		fieldTypes:       map[string]fieldTypeRef{},
		fieldTypeNames:   map[string][]string{},
		chapters:         map[string]bool{},
		hotspots:         map[string]bool{},
		systems:          map[string]bool{},
		teams:            map[string]bool{},
		workflows:        map[string]workflowIndex{},
		scenarioCounts:   map[string]int{},
		definitionRanges: map[string]hcl.Range{},
		externalContexts: map[string]bool{},
	}
	var diagnostics hcl.Diagnostics
	content, _, _ := syntax.PartialContent(body, syntax.ModelSchema())
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 {
			continue
		}
		diagnostics = append(diagnostics, validateLabel(block, block.Labels[0])...)
		switch block.Type {
		case "bounded_context":
			diagnostics = append(diagnostics, index.addBoundedContext(block)...)
		case "actor":
			diagnostics = append(diagnostics, addNamedSymbol(index.actors, block, "actor")...)
		case "team":
			diagnostics = append(diagnostics, addNamedSymbol(index.teams, block, "team")...)
		case "system":
			diagnostics = append(diagnostics, addNamedSymbol(index.systems, block, "system")...)
		case "chapter":
			diagnostics = append(diagnostics, addNamedSymbol(index.chapters, block, "chapter")...)
		case "hotspot":
			diagnostics = append(diagnostics, addNamedSymbol(index.hotspots, block, "hotspot")...)
		case "state_change", "state_view", "automation", "translation":
			diagnostics = append(diagnostics, index.addWorkflow(block)...)
		}
	}
	return index, diagnostics
}

func (i *modelIndex) addBoundedContext(block *hcl.Block) hcl.Diagnostics {
	contextID := block.Labels[0]
	diagnostics := addNamedSymbol(i.boundedContexts, block, "bounded_context")
	content, _, _ := syntax.PartialContent(block.Body, syntax.BoundedContextSchema())
	if external, ok := literalBool(content.Attributes["external"]); ok && external {
		i.externalContexts[contextID] = true
	}
	for _, child := range content.Blocks {
		if len(child.Labels) == 0 {
			continue
		}
		diagnostics = append(diagnostics, validateLabel(child, child.Labels[0])...)
		address := contextID + "." + child.Labels[0]
		switch child.Type {
		case "aggregate":
			diagnostics = append(diagnostics, addAddress(i.aggregates, child, "aggregate", address)...)
		case "event":
			diagnostics = append(diagnostics, addAddress(i.catalogEvents, child, "event", address)...)
		case "field_type":
			if _, exists := i.fieldTypes[address]; exists {
				diagnostics = append(diagnostics, duplicateDiagnostic(child, "field_type", address))
				continue
			}
			i.fieldTypes[address] = readFieldType(child.Body)
			i.fieldTypeNames[child.Labels[0]] = append(i.fieldTypeNames[child.Labels[0]], contextID)
		}
	}
	return diagnostics
}

func (i *modelIndex) addWorkflow(block *hcl.Block) hcl.Diagnostics {
	id := block.Labels[0]
	if _, exists := i.workflows[id]; exists {
		return hcl.Diagnostics{duplicateDiagnostic(block, "workflow", id)}
	}
	workflow := workflowIndex{elements: map[string]map[string]bool{
		"command": {}, "readmodel": {}, "screen": {}, "processor": {},
		"screen_image": {}, "table": {}, "scenario": {},
	}}
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
	}
	i.workflows[id] = workflow
	i.workflowOrder = append(i.workflowOrder, id)
	i.definitionRanges[id] = block.DefRange
	return diagnostics
}

func addNamedSymbol(symbols map[string]bool, block *hcl.Block, kind string) hcl.Diagnostics {
	id := block.Labels[0]
	if symbols[id] {
		return hcl.Diagnostics{duplicateDiagnostic(block, kind, id)}
	}
	symbols[id] = true
	return nil
}

func addAddress(symbols map[string]bool, block *hcl.Block, kind, address string) hcl.Diagnostics {
	if symbols[address] {
		return hcl.Diagnostics{duplicateDiagnostic(block, kind, address)}
	}
	symbols[address] = true
	return nil
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
