// Package model loads valid Event Modeling HCL into a canonical typed model.
package model

import (
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// Profile is the validator profile used by Load.
type Profile = validator.Profile

const (
	Workshop = validator.Workshop
	Valid    = validator.Valid
	Strict   = validator.Strict
)

type WorkflowKind string

const (
	StateChange WorkflowKind = "state_change"
	StateView   WorkflowKind = "state_view"
	Automation  WorkflowKind = "automation"
	Translation WorkflowKind = "translation"
)

type ElementKind string

const (
	Command     ElementKind = "command"
	ReadModel   ElementKind = "readmodel"
	Screen      ElementKind = "screen"
	Processor   ElementKind = "processor"
	ScreenImage ElementKind = "screen_image"
	Table       ElementKind = "table"
)

type StepKind string

const (
	Given StepKind = "given"
	When  StepKind = "when"
	Then  StepKind = "then"
)

type Model struct {
	Title     string     `json:"title,omitempty"`
	Version   string     `json:"version,omitempty"`
	Actors    []Actor    `json:"actors,omitempty"`
	Owners    []Owner    `json:"owners,omitempty"`
	Contexts  []Context  `json:"contexts,omitempty"`
	Workflows []Workflow `json:"workflows,omitempty"`
	Chapters  []Chapter  `json:"chapters,omitempty"`
	Hotspots  []Hotspot  `json:"hotspots,omitempty"`
	Edges     []Edge     `json:"edges,omitempty"`
}

type Actor struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	TitleExplicit bool   `json:"title_explicit"`
	AuthRequired  bool   `json:"auth_required"`
	Description   string `json:"description,omitempty"`
}

type Owner struct {
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	Title         string `json:"title"`
	TitleExplicit bool   `json:"title_explicit"`
	Description   string `json:"description,omitempty"`
	External      bool   `json:"external,omitempty"`
}

type Context struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	TitleExplicit bool        `json:"title_explicit"`
	Description   string      `json:"description,omitempty"`
	External      bool        `json:"external,omitempty"`
	Owner         string      `json:"owner,omitempty"`
	Aggregates    []Aggregate `json:"aggregates,omitempty"`
	FieldTypes    []FieldType `json:"field_types,omitempty"`
	Events        []Event     `json:"events,omitempty"`
}

type Aggregate struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	TitleExplicit bool   `json:"title_explicit"`
	Description   string `json:"description,omitempty"`
}

type FieldType struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Cardinality string  `json:"cardinality,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
}

type Event struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	TitleExplicit bool         `json:"title_explicit"`
	Semantic      Semantic     `json:"semantic"`
	Presentation  Presentation `json:"presentation"`
	Fields        []Field      `json:"fields,omitempty"`
}

type Workflow struct {
	Kind          WorkflowKind `json:"kind"`
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	TitleExplicit bool         `json:"title_explicit"`
	Status        string       `json:"status,omitempty"`
	Owner         string       `json:"owner,omitempty"`
	Description   string       `json:"description,omitempty"`
	Elements      []Element    `json:"elements,omitempty"`
	Scenarios     []Scenario   `json:"scenarios,omitempty"`
}

type Element struct {
	Kind          ElementKind  `json:"kind"`
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	TitleExplicit bool         `json:"title_explicit"`
	Semantic      Semantic     `json:"semantic"`
	Presentation  Presentation `json:"presentation"`
	Fields        []Field      `json:"fields,omitempty"`
	From          []string     `json:"from,omitempty"`
	To            []string     `json:"to,omitempty"`
}

type Semantic struct {
	Description           string   `json:"description,omitempty"`
	Aggregate             string   `json:"aggregate,omitempty"`
	AggregateDependencies []string `json:"aggregate_dependencies,omitempty"`
	APIEndpoint           string   `json:"api_endpoint,omitempty"`
	Service               string   `json:"service,omitempty"`
	CreatesAggregate      bool     `json:"creates_aggregate,omitempty"`
	ExternalTrigger       bool     `json:"external_trigger,omitempty"`
	Triggers              []string `json:"triggers,omitempty"`
	Question              string   `json:"question,omitempty"`
	Actor                 string   `json:"actor,omitempty"`
}

type Presentation struct {
	GroupID     string   `json:"group_id,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Sketched    bool     `json:"sketched,omitempty"`
	Prototype   bool     `json:"prototype,omitempty"`
	ListElement bool     `json:"list_element,omitempty"`
	URL         string   `json:"url,omitempty"`
}

type Field struct {
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	Cardinality        string  `json:"cardinality,omitempty"`
	Mapping            string  `json:"mapping,omitempty"`
	Optional           bool    `json:"optional,omitempty"`
	TechnicalAttribute bool    `json:"technical_attribute,omitempty"`
	Generated          bool    `json:"generated,omitempty"`
	IDAttribute        bool    `json:"id_attribute,omitempty"`
	PII                bool    `json:"pii,omitempty"`
	Schema             string  `json:"schema,omitempty"`
	Fields             []Field `json:"fields,omitempty"`
}

type Scenario struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	TitleExplicit bool     `json:"title_explicit"`
	Description   string   `json:"description,omitempty"`
	Steps         []Step   `json:"steps,omitempty"`
	Comments      []string `json:"comments,omitempty"`
}

type Step struct {
	Kind            StepKind `json:"kind"`
	Title           string   `json:"title,omitempty"`
	Target          string   `json:"target"`
	Ref             string   `json:"ref,omitempty"`
	Error           string   `json:"error,omitempty"`
	ExpectEmptyList bool     `json:"expect_empty_list,omitempty"`
	Fields          []Field  `json:"fields,omitempty"`
}

type Chapter struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	TitleExplicit bool     `json:"title_explicit"`
	Description   string   `json:"description,omitempty"`
	Workflows     []string `json:"workflows,omitempty"`
}

type Hotspot struct {
	ID          string `json:"id"`
	Question    string `json:"question"`
	Description string `json:"description,omitempty"`
	On          string `json:"on,omitempty"`
	Status      string `json:"status,omitempty"`
}

type Edge struct {
	WorkflowID string `json:"workflow_id"`
	From       string `json:"from"`
	To         string `json:"to"`
}

// Load validates source and, only when it is valid, decodes it into a
// normalized model. Validation warnings remain available in diagnostics.
func Load(filename string, source []byte, profile Profile) (*Model, hcl.Diagnostics) {
	diagnostics := validator.ValidateSourceWithProfile(filename, source, profile)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	parser := hclparse.NewParser()
	file, parseDiagnostics := parser.ParseHCL(source, filename)
	if parseDiagnostics.HasErrors() {
		return nil, append(diagnostics, parseDiagnostics...)
	}
	decoded, decodeDiagnostics := decode(file.Body)
	diagnostics = append(diagnostics, decodeDiagnostics...)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	return decoded, diagnostics
}

func decode(body hcl.Body) (*Model, hcl.Diagnostics) {
	content, diagnostics := contentOf(body, rootSchema())
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	model := &Model{}
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 {
			continue
		}
		switch block.Type {
		case "actor":
			model.Actors = append(model.Actors, decodeActor(block))
		case "team", "system":
			model.Owners = append(model.Owners, decodeOwner(block))
		case "bounded_context":
			model.Contexts = append(model.Contexts, decodeContext(block))
		case "chapter":
			model.Chapters = append(model.Chapters, decodeChapter(block))
		case "hotspot":
			model.Hotspots = append(model.Hotspots, decodeHotspot(block))
		case "state_change", "state_view", "automation", "translation":
			workflow, edges := decodeWorkflow(block)
			model.Workflows = append(model.Workflows, workflow)
			model.Edges = append(model.Edges, edges...)
		}
	}
	return model, diagnostics
}

func decodeActor(block *hcl.Block) Actor {
	content, _ := contentOf(block.Body, actorSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	authRequired, _ := boolValue(content.Attributes["auth_required"])
	return Actor{ID: block.Labels[0], Title: title, TitleExplicit: explicit, AuthRequired: authRequired, Description: stringValue(content.Attributes["description"])}
}

func decodeOwner(block *hcl.Block) Owner {
	content, _ := contentOf(block.Body, ownerSchema(block.Type))
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	external, _ := boolValue(content.Attributes["external"])
	return Owner{Kind: block.Type, ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"]), External: external}
}

func decodeContext(block *hcl.Block) Context {
	content, _ := contentOf(block.Body, contextSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	external, _ := boolValue(content.Attributes["external"])
	context := Context{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"]), External: external, Owner: traversalValue(content.Attributes["owner"])}
	for _, child := range content.Blocks {
		switch child.Type {
		case "aggregate":
			context.Aggregates = append(context.Aggregates, decodeAggregate(child))
		case "field_type":
			context.FieldTypes = append(context.FieldTypes, decodeFieldType(child))
		case "event":
			context.Events = append(context.Events, decodeEvent(child))
		}
	}
	return context
}

func decodeAggregate(block *hcl.Block) Aggregate {
	content, _ := contentOf(block.Body, aggregateSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	return Aggregate{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"])}
}

func decodeFieldType(block *hcl.Block) FieldType {
	content, _ := contentOf(block.Body, fieldSchema())
	return FieldType{ID: block.Labels[0], Type: stringValue(content.Attributes["type"]), Cardinality: stringValue(content.Attributes["cardinality"]), Fields: decodeFields(content.Blocks)}
}

func decodeEvent(block *hcl.Block) Event {
	content, _ := contentOf(block.Body, eventSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	return Event{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Semantic: semanticFrom(content.Attributes), Presentation: presentationFrom(content.Attributes), Fields: decodeFields(content.Blocks)}
}

func decodeWorkflow(block *hcl.Block) (Workflow, []Edge) {
	content, _ := contentOf(block.Body, workflowSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	workflow := Workflow{Kind: mustWorkflowKind(block.Type), ID: block.Labels[0], Title: title, TitleExplicit: explicit, Status: stringValue(content.Attributes["status"]), Owner: traversalValue(content.Attributes["owner"]), Description: stringValue(content.Attributes["description"])}
	var edges []Edge
	for _, child := range content.Blocks {
		switch child.Type {
		case "command", "readmodel", "screen", "processor", "screen_image", "table":
			element, elementEdges := decodeElement(workflow.ID, child)
			workflow.Elements = append(workflow.Elements, element)
			edges = append(edges, elementEdges...)
		case "scenario":
			workflow.Scenarios = append(workflow.Scenarios, decodeScenario(child))
		}
	}
	return workflow, edges
}

func decodeElement(workflowID string, block *hcl.Block) (Element, []Edge) {
	content, _ := contentOf(block.Body, elementSchema(block.Type))
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	from := traversalList(content.Attributes["from"])
	to := traversalList(content.Attributes["to"])
	element := Element{Kind: mustElementKind(block.Type), ID: block.Labels[0], Title: title, TitleExplicit: explicit, Semantic: semanticFrom(content.Attributes), Presentation: presentationFrom(content.Attributes), Fields: decodeFields(content.Blocks), From: from, To: to}
	owner := block.Type + "." + block.Labels[0]
	edges := make([]Edge, 0, len(from)+len(to))
	for _, target := range to {
		edges = append(edges, Edge{WorkflowID: workflowID, From: owner, To: target})
	}
	for _, source := range from {
		edges = append(edges, Edge{WorkflowID: workflowID, From: source, To: owner})
	}
	return element, edges
}

func decodeScenario(block *hcl.Block) Scenario {
	content, _ := contentOf(block.Body, scenarioSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	scenario := Scenario{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"])}
	for _, child := range content.Blocks {
		if child.Type == "comment" {
			comment, _ := contentOf(child.Body, commentSchema())
			scenario.Comments = append(scenario.Comments, stringValue(comment.Attributes["description"]))
			continue
		}
		step := decodeStep(child)
		scenario.Steps = append(scenario.Steps, step)
	}
	return scenario
}

func decodeStep(block *hcl.Block) Step {
	content, _ := contentOf(block.Body, stepSchema())
	step := Step{Kind: mustStepKind(block.Type), Title: stringValue(content.Attributes["title"]), ExpectEmptyList: boolValueOrFalse(content.Attributes["expect_empty_list"]), Fields: decodeFields(content.Blocks)}
	for _, target := range []string{"event", "command", "readmodel", "processor", "error"} {
		attribute := content.Attributes[target]
		if attribute == nil {
			continue
		}
		step.Target = target
		if target == "error" {
			step.Error = stringValue(attribute)
		} else {
			step.Ref = traversalValue(attribute)
		}
		break
	}
	return step
}

func decodeChapter(block *hcl.Block) Chapter {
	content, _ := contentOf(block.Body, chapterSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	return Chapter{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"]), Workflows: traversalList(content.Attributes["workflows"])}
}

func decodeHotspot(block *hcl.Block) Hotspot {
	content, _ := contentOf(block.Body, hotspotSchema())
	return Hotspot{ID: block.Labels[0], Question: stringValue(content.Attributes["question"]), Description: stringValue(content.Attributes["description"]), On: traversalValue(content.Attributes["on"]), Status: stringValue(content.Attributes["status"])}
}

func semanticFrom(attributes hcl.Attributes) Semantic {
	createsAggregate, _ := boolValue(attributes["creates_aggregate"])
	externalTrigger, _ := boolValue(attributes["external_trigger"])
	return Semantic{Description: stringValue(attributes["description"]), Aggregate: traversalValue(attributes["aggregate"]), AggregateDependencies: traversalList(attributes["aggregate_dependencies"]), APIEndpoint: stringValue(attributes["api_endpoint"]), Service: stringValue(attributes["service"]), CreatesAggregate: createsAggregate, ExternalTrigger: externalTrigger, Triggers: stringList(attributes["triggers"]), Question: stringValue(attributes["question"]), Actor: traversalValue(attributes["actor"])}
}

func presentationFrom(attributes hcl.Attributes) Presentation {
	sketch, _ := boolValue(attributes["sketched"])
	listElement, _ := boolValue(attributes["list_element"])
	return Presentation{GroupID: stringValue(attributes["group_id"]), Tags: stringList(attributes["tags"]), Sketched: sketch, Prototype: attributes["prototype"] != nil, ListElement: listElement, URL: stringValue(attributes["url"])}
}

func decodeFields(blocks hcl.Blocks) []Field {
	fields := make([]Field, 0, len(blocks))
	for _, block := range blocks {
		if block.Type != "field" && block.Type != "subfield" {
			continue
		}
		content, _ := contentOf(block.Body, fieldSchema())
		optional, _ := boolValue(content.Attributes["optional"])
		technical, _ := boolValue(content.Attributes["technical_attribute"])
		generated, _ := boolValue(content.Attributes["generated"])
		idAttribute, _ := boolValue(content.Attributes["id_attribute"])
		pii, _ := boolValue(content.Attributes["pii"])
		fields = append(fields, Field{Name: block.Labels[0], Type: typeValue(content.Attributes["type"]), Cardinality: stringValue(content.Attributes["cardinality"]), Mapping: stringValue(content.Attributes["mapping"]), Optional: optional, TechnicalAttribute: technical, Generated: generated, IDAttribute: idAttribute, PII: pii, Schema: stringValue(content.Attributes["schema"]), Fields: decodeFields(content.Blocks)})
	}
	return fields
}

func effectiveTitle(label string, attribute *hcl.Attribute) (string, bool) {
	if title, ok := stringValueOK(attribute); ok {
		return title, true
	}
	return humanize(label), false
}

func humanize(label string) string {
	words := strings.Split(label, "_")
	for index, word := range words {
		if word == "" {
			continue
		}
		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func stringValue(attribute *hcl.Attribute) string {
	value, _ := stringValueOK(attribute)
	return value
}

func stringValueOK(attribute *hcl.Attribute) (string, bool) {
	if attribute == nil {
		return "", false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !value.Type().Equals(cty.String) {
		return "", false
	}
	return value.AsString(), true
}

func boolValue(attribute *hcl.Attribute) (bool, bool) {
	if attribute == nil {
		return false, false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !value.Type().Equals(cty.Bool) {
		return false, false
	}
	return value.True(), true
}

func boolValueOrFalse(attribute *hcl.Attribute) bool {
	value, _ := boolValue(attribute)
	return value
}

func typeValue(attribute *hcl.Attribute) string {
	if value, ok := stringValueOK(attribute); ok {
		return value
	}
	return traversalValue(attribute)
}

func traversalValue(attribute *hcl.Attribute) string {
	if attribute == nil {
		return ""
	}
	traversal, diagnostics := hcl.AbsTraversalForExpr(attribute.Expr)
	if diagnostics.HasErrors() {
		return ""
	}
	return traversalString(traversal)
}

func traversalList(attribute *hcl.Attribute) []string {
	if attribute == nil {
		return nil
	}
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return nil
	}
	values := make([]string, 0, len(expressions))
	for _, expression := range expressions {
		traversal, traversalDiagnostics := hcl.AbsTraversalForExpr(expression)
		if traversalDiagnostics.HasErrors() {
			continue
		}
		values = append(values, traversalString(traversal))
	}
	return values
}

func traversalString(traversal hcl.Traversal) string {
	if len(traversal) == 0 {
		return ""
	}
	parts := []string{traversal.RootName()}
	for _, step := range traversal[1:] {
		attribute, ok := step.(hcl.TraverseAttr)
		if !ok {
			return ""
		}
		parts = append(parts, attribute.Name)
	}
	return strings.Join(parts, ".")
}

func stringList(attribute *hcl.Attribute) []string {
	if attribute == nil {
		return nil
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !(value.Type().IsTupleType() || value.Type().IsListType() || value.Type().IsSetType()) {
		return nil
	}
	values := make([]string, 0, value.LengthInt())
	iterator := value.ElementIterator()
	for iterator.Next() {
		_, item := iterator.Element()
		if !item.IsNull() && item.Type().Equals(cty.String) {
			values = append(values, item.AsString())
		}
	}
	return values
}

func mustWorkflowKind(value string) WorkflowKind {
	switch value {
	case "state_change":
		return StateChange
	case "state_view":
		return StateView
	case "automation":
		return Automation
	case "translation":
		return Translation
	}
	return ""
}

func mustElementKind(value string) ElementKind {
	switch value {
	case "command":
		return Command
	case "readmodel":
		return ReadModel
	case "screen":
		return Screen
	case "processor":
		return Processor
	case "screen_image":
		return ScreenImage
	case "table":
		return Table
	}
	return ""
}

func mustStepKind(value string) StepKind {
	switch value {
	case "given":
		return Given
	case "when":
		return When
	case "then":
		return Then
	}
	return ""
}

func rootSchema() hcl.BodySchema {
	return hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "bounded_context", LabelNames: []string{"id"}}, {Type: "actor", LabelNames: []string{"id"}}, {Type: "team", LabelNames: []string{"id"}}, {Type: "system", LabelNames: []string{"id"}}, {Type: "chapter", LabelNames: []string{"id"}}, {Type: "hotspot", LabelNames: []string{"id"}}, {Type: "state_change", LabelNames: []string{"id"}}, {Type: "state_view", LabelNames: []string{"id"}}, {Type: "automation", LabelNames: []string{"id"}}, {Type: "translation", LabelNames: []string{"id"}}}}
}

func contextSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "description", "external", "owner"), Blocks: []hcl.BlockHeaderSchema{{Type: "aggregate", LabelNames: []string{"name"}}, {Type: "field_type", LabelNames: []string{"name"}}, {Type: "event", LabelNames: []string{"id"}}}}
}

func aggregateSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "description")}
}

func eventSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("group_id", "tags", "title", "description", "aggregate", "aggregate_dependencies", "service", "sketched", "prototype", "list_element"), Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}}}
}

func workflowSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "status", "description", "owner"), Blocks: []hcl.BlockHeaderSchema{{Type: "command", LabelNames: []string{"id"}}, {Type: "readmodel", LabelNames: []string{"id"}}, {Type: "screen", LabelNames: []string{"id"}}, {Type: "screen_image", LabelNames: []string{"id"}}, {Type: "processor", LabelNames: []string{"id"}}, {Type: "table", LabelNames: []string{"id"}}, {Type: "scenario", LabelNames: []string{"id"}}}}
}

func elementSchema(kind string) hcl.BodySchema {
	names := []string{"group_id", "tags", "title", "description", "aggregate", "aggregate_dependencies", "api_endpoint", "service", "creates_aggregate", "external_trigger", "triggers", "sketched", "prototype", "list_element", "from", "to"}
	if kind == "readmodel" {
		names = append(names, "question")
	}
	if kind == "screen" {
		names = append(names, "actor")
	}
	if kind == "screen_image" {
		names = []string{"title", "url"}
	}
	if kind == "table" {
		names = []string{"title"}
	}
	return hcl.BodySchema{Attributes: attributes(names...), Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}}}
}

func fieldSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("type", "example", "mapping", "optional", "technical_attribute", "generated", "id_attribute", "pii", "schema", "cardinality"), Blocks: []hcl.BlockHeaderSchema{{Type: "subfield", LabelNames: []string{"name"}}}}
}

func scenarioSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "description"), Blocks: []hcl.BlockHeaderSchema{{Type: "given"}, {Type: "when"}, {Type: "then"}, {Type: "comment"}}}
}

func stepSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "tags", "examples", "event", "command", "readmodel", "processor", "error", "expect_empty_list"), Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}}}
}

func actorSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "auth_required", "description")}
}

func ownerSchema(kind string) hcl.BodySchema {
	names := []string{"title", "description"}
	if kind == "system" {
		names = append(names, "external")
	}
	return hcl.BodySchema{Attributes: attributes(names...)}
}

func chapterSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("title", "description", "workflows")}
}
func hotspotSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: attributes("question", "description", "on", "status")}
}
func commentSchema() hcl.BodySchema { return hcl.BodySchema{Attributes: attributes("description")} }

func attributes(names ...string) []hcl.AttributeSchema {
	result := make([]hcl.AttributeSchema, 0, len(names))
	for _, name := range names {
		result = append(result, hcl.AttributeSchema{Name: name})
	}
	return result
}

func contentOf(body hcl.Body, schema hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	return body.Content(&schema)
}
