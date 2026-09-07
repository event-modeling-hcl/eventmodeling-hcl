// Package validator implements the strict validator for the native HCL Event
// Modeling Specification defined in eventmodeling.hclspec.md.
package validator

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

var modelIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateFile parses and validates one native HCL Event Modeling document.
func ValidateFile(path string) hcl.Diagnostics {
	return ValidateFileWithProfile(path, Valid)
}

// ValidateFileWithProfile parses and validates one native HCL Event Modeling
// document using the requested validation profile.
func ValidateFileWithProfile(path string, profile Profile) hcl.Diagnostics {
	source, err := os.ReadFile(path)
	if err != nil {
		return hcl.Diagnostics{&hcl.Diagnostic{Severity: hcl.DiagError, Summary: "Failed to read file", Detail: fmt.Sprintf("The configuration file %q could not be read.", path), Extra: codeReadFile}}
	}
	return ValidateSourceWithProfile(path, source, profile)
}

// ValidateSource parses and validates one in-memory Event Modeling document.
func ValidateSource(filename string, source []byte) hcl.Diagnostics {
	return ValidateSourceWithProfile(filename, source, Valid)
}

// ValidateSourceWithProfile parses and validates one in-memory Event Modeling
// document using the requested validation profile.
func ValidateSourceWithProfile(filename string, source []byte, profile Profile) hcl.Diagnostics {
	parser := hclparse.NewParser()
	file, diagnostics := parser.ParseHCL(source, filename)
	if diagnostics.HasErrors() {
		return applyProfile(diagnostics, profile)
	}
	return applyProfile(validateBody(file.Body), profile)
}

type fieldTypeRef struct {
	fieldType   string
	cardinality string
}

type workflowIndex struct {
	elements map[string]map[string]bool
}

type modelIndex struct {
	actors           map[string]bool
	aggregates       map[string]bool
	boundedContexts  map[string]bool
	catalogEvents    map[string]bool
	fieldTypes       map[string]fieldTypeRef
	fieldTypeNames   map[string][]string
	chapters         map[string]bool
	hotspots         map[string]bool
	systems          map[string]bool
	teams            map[string]bool
	workflows        map[string]workflowIndex
	workflowOrder    []string
	scenarioCounts   map[string]int
	definitionRanges map[string]hcl.Range
	externalContexts map[string]bool
}

type modelValidator struct {
	index modelIndex
}

type contextValidator struct {
	model     *modelValidator
	contextID string
}

func validateBody(body hcl.Body) hcl.Diagnostics {
	index, diagnostics := buildIndex(body)
	validator := &modelValidator{index: index}
	schema := modelSchema()
	content, contentDiagnostics := body.Content(&schema)
	diagnostics = append(diagnostics, contentDiagnostics...)
	for _, block := range content.Blocks {
		switch block.Type {
		case "bounded_context":
			diagnostics = append(diagnostics, validator.validateBoundedContext(block)...)
		case "actor":
			diagnostics = append(diagnostics, validator.validateActor(block)...)
		case "team", "system":
			diagnostics = append(diagnostics, validator.validateOwner(block)...)
		case "chapter":
			diagnostics = append(diagnostics, validator.validateChapter(block)...)
		case "hotspot":
			diagnostics = append(diagnostics, validator.validateHotspot(block)...)
		case "state_change", "state_view", "automation", "translation":
			diagnostics = append(diagnostics, validator.validateWorkflow(block)...)
		}
	}
	diagnostics = append(diagnostics, validator.validateShelfSmell()...)
	return diagnostics
}

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
	schema := modelSchema()
	content, _, _ := body.PartialContent(&schema)
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
	contextSchema := boundedContextSchema()
	contextContent, _, _ := block.Body.PartialContent(&contextSchema)
	if external, ok := literalBool(contextContent.Attributes["external"]); ok && external {
		i.externalContexts[contextID] = true
	}
	schema := hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{
		{Type: "aggregate", LabelNames: []string{"name"}},
		{Type: "field_type", LabelNames: []string{"name"}},
		{Type: "event", LabelNames: []string{"id"}},
	}}
	content, _, _ := block.Body.PartialContent(&schema)
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
	schema := workflowSchema()
	content, _, _ := block.Body.PartialContent(&schema)
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
	schema := hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "type"}, {Name: "cardinality"}}}
	content, _, _ := body.PartialContent(&schema)
	fieldType, _ := literalString(content.Attributes["type"])
	cardinality, _ := literalString(content.Attributes["cardinality"])
	return fieldTypeRef{fieldType: fieldType, cardinality: cardinality}
}

func (v *modelValidator) validateBoundedContext(block *hcl.Block) hcl.Diagnostics {
	schema := boundedContextSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, boundedContextRule)...)
	context := contextValidator{model: v, contextID: block.Labels[0]}
	if owner := content.Attributes["owner"]; owner != nil {
		diagnostics = append(diagnostics, v.validateOwnerReference(owner)...)
	}
	for _, child := range content.Blocks {
		switch child.Type {
		case "aggregate":
			diagnostics = append(diagnostics, validateSimpleBlock(child.Body, aggregateSchema(), aggregateRule)...)
		case "field_type":
			diagnostics = append(diagnostics, context.validateField(child, true)...)
		case "event":
			diagnostics = append(diagnostics, context.validateEvent(child)...)
		}
	}
	return diagnostics
}

func (v *modelValidator) validateActor(block *hcl.Block) hcl.Diagnostics {
	return validateSimpleBlock(block.Body, actorSchema(), actorRule)
}

func (v *modelValidator) validateOwner(block *hcl.Block) hcl.Diagnostics {
	schema := ownerSchema(block.Type)
	return validateSimpleBlock(block.Body, schema, ownerRule)
}

func (v *modelValidator) validateChapter(block *hcl.Block) hcl.Diagnostics {
	schema := chapterSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, chapterRule)...)
	if workflows := content.Attributes["workflows"]; workflows != nil {
		diagnostics = append(diagnostics, v.validateReferenceList(workflows, "workflow", "")...)
		diagnostics = append(diagnostics, v.validateChapterRange(workflows)...)
		if references, referenceDiagnostics := hcl.ExprList(workflows.Expr); !referenceDiagnostics.HasErrors() && len(references) == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidChapter, workflows.Expr.Range(), "Invalid chapter", "chapter must contain at least one workflow."))
		}
	}
	return diagnostics
}

func (v *modelValidator) validateHotspot(block *hcl.Block) hcl.Diagnostics {
	schema := hotspotSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, hotspotRule)...)
	if on := content.Attributes["on"]; on != nil {
		diagnostics = append(diagnostics, v.validateGlobalReference(on)...)
	}
	if status, ok := literalString(content.Attributes["status"]); !ok || status != "resolved" {
		diagnostics = append(diagnostics, warningDiagnostic(codeOpenHotspot, block.DefRange, "Open hotspot", fmt.Sprintf("hotspot %q is unresolved.", block.Labels[0])))
	}
	return diagnostics
}

func (v *modelValidator) validateWorkflow(block *hcl.Block) hcl.Diagnostics {
	schema := workflowSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, workflowRule)...)
	if owner := content.Attributes["owner"]; owner != nil {
		diagnostics = append(diagnostics, v.validateOwnerReference(owner)...)
	}
	for _, child := range content.Blocks {
		if !workflowAllows(block.Type, child.Type) {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidWorkflowChild, child.DefRange, "Invalid workflow child", fmt.Sprintf("%s blocks are not allowed in %s workflows.", child.Type, block.Type)))
			continue
		}
		switch child.Type {
		case "command", "readmodel", "screen", "processor":
			diagnostics = append(diagnostics, v.validateElement(block.Type, block.Labels[0], child)...)
		case "screen_image":
			diagnostics = append(diagnostics, validateSimpleBlock(child.Body, screenImageSchema(), screenImageRule)...)
		case "table":
			diagnostics = append(diagnostics, v.validateTable(child)...)
		case "scenario":
			diagnostics = append(diagnostics, v.validateScenario(block.Type, block.Labels[0], child)...)
		}
	}
	diagnostics = append(diagnostics, v.validateWorkflowExternality(block.Type, block.DefRange, content.Blocks)...)
	diagnostics = append(diagnostics, v.workflowWarnings(content.Blocks)...)
	return diagnostics
}

func workflowAllows(workflowType, childType string) bool {
	if childType == "table" || childType == "scenario" {
		return true
	}
	if childType == "screen_image" {
		return workflowType == "state_change" || workflowType == "state_view"
	}
	switch workflowType {
	case "state_change":
		return childType == "screen" || childType == "command"
	case "state_view":
		return childType == "readmodel" || childType == "screen"
	case "automation", "translation":
		return childType == "readmodel" || childType == "processor" || childType == "command"
	default:
		return false
	}
}

func (v *modelValidator) validateElement(workflowType, workflowID string, block *hcl.Block) hcl.Diagnostics {
	schema := elementSchema(block.Type)
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, elementRule)...)
	context := contextValidator{model: v}
	if aggregate := content.Attributes["aggregate"]; aggregate != nil {
		diagnostics = append(diagnostics, context.validateAggregate(aggregate, false)...)
	}
	if dependencies := content.Attributes["aggregate_dependencies"]; dependencies != nil {
		diagnostics = append(diagnostics, context.validateAggregateList(dependencies, false)...)
	}
	if actor := content.Attributes["actor"]; actor != nil {
		diagnostics = append(diagnostics, v.validateReference(actor, "actor", workflowID)...)
	}
	for _, direction := range []string{"from", "to"} {
		if attribute := content.Attributes[direction]; attribute != nil {
			diagnostics = append(diagnostics, v.validateFlowReferences(attribute, workflowType, workflowID, block.Type, direction)...)
		}
	}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

func (v *modelValidator) validateTable(block *hcl.Block) hcl.Diagnostics {
	schema := tableSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, tableRule)...)
	context := contextValidator{model: v}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

func (v *contextValidator) validateEvent(block *hcl.Block) hcl.Diagnostics {
	schema := eventSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, elementRule)...)
	if aggregate := content.Attributes["aggregate"]; aggregate != nil {
		diagnostics = append(diagnostics, v.validateAggregate(aggregate, true)...)
	}
	if dependencies := content.Attributes["aggregate_dependencies"]; dependencies != nil {
		diagnostics = append(diagnostics, v.validateAggregateList(dependencies, true)...)
	}
	diagnostics = append(diagnostics, v.validateElementFields(content)...)
	return diagnostics
}

// validateElementFields validates the optional shorthand fields = [...] list
// attribute together with the field blocks that share its namespace.
func (v *contextValidator) validateElementFields(content *hcl.BodyContent) hcl.Diagnostics {
	seen := map[string]bool{}
	var diagnostics hcl.Diagnostics
	if attribute := content.Attributes["fields"]; attribute != nil {
		diagnostics = append(diagnostics, v.validateFieldList(attribute, seen)...)
	}
	return append(diagnostics, v.validateFields(content.Blocks, seen)...)
}

func (v *contextValidator) validateFields(fields hcl.Blocks, seen map[string]bool) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, field := range fields {
		if len(field.Labels) == 0 {
			continue
		}
		name := field.Labels[0]
		diagnostics = append(diagnostics, validateLabel(field, name)...)
		if seen[name] {
			diagnostics = append(diagnostics, duplicateDiagnostic(field, field.Type, name))
			continue
		}
		seen[name] = true
		diagnostics = append(diagnostics, v.validateField(field, false)...)
	}
	return diagnostics
}

// validateFieldList resolves each entry of a fields = [...] attribute as a
// field_type reference and records its synthesized field name in seen.
func (v *contextValidator) validateFieldList(attribute *hcl.Attribute, seen map[string]bool) hcl.Diagnostics {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	for _, expression := range expressions {
		entry := &hcl.Attribute{Name: "fields", Expr: expression, Range: attribute.Range, NameRange: attribute.NameRange}
		address, referenceDiagnostics := v.fieldTypeAddress(entry)
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if referenceDiagnostics.HasErrors() {
			continue
		}
		name := address[strings.LastIndex(address, ".")+1:]
		if seen[name] {
			diagnostics = append(diagnostics, errorDiagnostic(codeDuplicateID, expression.Range(), "Duplicate field name", fmt.Sprintf("field %q is declared more than once in its namespace.", name)))
			continue
		}
		seen[name] = true
	}
	return diagnostics
}

func (v *contextValidator) validateField(block *hcl.Block, fieldTypeDeclaration bool) hcl.Diagnostics {
	schema := fieldSchema(block.Type)
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, fieldRule)...)
	typeAttribute := content.Attributes["type"]
	if typeAttribute == nil {
		if fieldTypeDeclaration {
			return append(diagnostics, errorDiagnostic(codeInvalidFieldType, block.DefRange, "Invalid field type", "a field_type declaration must set an explicit built-in type."))
		}
		address, inferenceDiagnostics := v.inferredFieldTypeAddress(block.Labels[0], block.DefRange)
		diagnostics = append(diagnostics, inferenceDiagnostics...)
		if !inferenceDiagnostics.HasErrors() {
			if reference, exists := v.model.index.fieldTypes[address]; exists {
				diagnostics = append(diagnostics, validateReferencedExample(content.Attributes, reference)...)
			}
		}
		return append(diagnostics, v.validateFields(content.Blocks, map[string]bool{})...)
	}
	if fieldTypeDeclaration {
		fieldType, ok := literalString(typeAttribute)
		if !ok {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidFieldType, typeAttribute.Expr.Range(), "Invalid field type", "field_type type must be a built-in type string."))
		} else {
			diagnostics = append(diagnostics, validateFieldTypeAndExample(content.Attributes, typeAttribute, fieldType)...)
		}
	} else if fieldType, ok := literalString(typeAttribute); ok {
		diagnostics = append(diagnostics, validateFieldTypeAndExample(content.Attributes, typeAttribute, fieldType)...)
	} else {
		address, referenceDiagnostics := v.fieldTypeAddress(typeAttribute)
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if reference, exists := v.model.index.fieldTypes[address]; exists {
			diagnostics = append(diagnostics, validateReferencedExample(content.Attributes, reference)...)
		}
	}
	diagnostics = append(diagnostics, v.validateFields(content.Blocks, map[string]bool{})...)
	return diagnostics
}

func validateFieldTypeAndExample(attributes hcl.Attributes, attribute *hcl.Attribute, fieldType string) hcl.Diagnostics {
	if !slices.Contains(fieldTypes, fieldType) {
		return hcl.Diagnostics{invalidEnum(attribute, "type", fieldTypes)}
	}
	return validateExample(attributes, fieldType, "field")
}

func (v *modelValidator) validateScenario(workflowType, workflowID string, block *hcl.Block) hcl.Diagnostics {
	schema := scenarioSchema()
	content, diagnostics := block.Body.Content(&schema)
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
		case "comment":
			diagnostics = append(diagnostics, validateSimpleBlock(step.Body, commentSchema(), commentRule)...)
			continue
		}
		diagnostics = append(diagnostics, v.validateScenarioStep(workflowType, workflowID, step)...)
	}
	if workflowType == "state_view" {
		if whenCount != 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios may not contain when steps. Found: when."))
		}
		if givenCount == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "state_view scenarios must contain at least one given step."))
		}
	} else if whenCount != 1 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario must contain exactly one when step."))
	}
	if thenCount == 0 {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenario, block.DefRange, "Invalid scenario", "a scenario must contain at least one then step."))
	}
	return diagnostics
}

func (v *modelValidator) validateScenarioStep(workflowType, workflowID string, block *hcl.Block) hcl.Diagnostics {
	schema := scenarioStepSchema()
	content, diagnostics := block.Body.Content(&schema)
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
	if !scenarioTargetAllowed(workflowType, block.Type, target) {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidScenarioTarget, content.Attributes[target].Expr.Range(), "Invalid scenario target", fmt.Sprintf("%s scenario %s steps may use: %s. You referenced: %s.", workflowType, block.Type, allowedScenarioTargets(workflowType, block.Type), target)))
	}
	if target != "error" {
		diagnostics = append(diagnostics, v.validateReference(content.Attributes[target], target, workflowID)...)
	}
	context := contextValidator{model: v}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

func scenarioTargetAllowed(workflowType, stepType, target string) bool {
	switch workflowType {
	case "state_change":
		return stepType == "given" && target == "event" || stepType == "when" && target == "command" || stepType == "then" && (target == "event" || target == "error")
	case "state_view":
		return stepType == "given" && target == "event" || stepType == "then" && (target == "readmodel" || target == "error")
	case "automation", "translation":
		return stepType == "given" && (target == "event" || target == "readmodel") || stepType == "when" && (target == "processor" || target == "command") || stepType == "then" && (target == "event" || target == "error")
	}
	return false
}

func allowedScenarioTargets(workflowType, stepType string) string {
	allowed := make([]string, 0, 2)
	for _, target := range []string{"event", "command", "readmodel", "processor", "error"} {
		if scenarioTargetAllowed(workflowType, stepType, target) {
			allowed = append(allowed, target)
		}
	}
	return strings.Join(allowed, " or ")
}

func (v *modelValidator) workflowWarnings(blocks hcl.Blocks) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	incomingCommands := incomingReferences(blocks, "command")
	for _, block := range blocks {
		if !slices.Contains([]string{"screen", "command", "readmodel"}, block.Type) {
			continue
		}
		schema := elementSchema(block.Type)
		content, _, _ := block.Body.PartialContent(&schema)
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
			schema := elementSchema(block.Type)
			content, _, _ := block.Body.PartialContent(&schema)
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
		schema := elementSchema(block.Type)
		content, _, _ := block.Body.PartialContent(&schema)
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
		schema := elementSchema(block.Type)
		content, _, _ := block.Body.PartialContent(&schema)
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

func literalString(attribute *hcl.Attribute) (string, bool) {
	if attribute == nil {
		return "", false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !value.Type().Equals(cty.String) {
		return "", false
	}
	return value.AsString(), true
}

func literalBool(attribute *hcl.Attribute) (bool, bool) {
	if attribute == nil {
		return false, false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !value.Type().Equals(cty.Bool) {
		return false, false
	}
	return value.True(), true
}
