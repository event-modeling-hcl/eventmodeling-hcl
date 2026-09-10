// Package model lowers validated Event Modeling source into a canonical model.
package model

import (
	"encoding/json"
	"strings"

	sourcepkg "github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
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
	IDAttribute bool    `json:"id_attribute,omitempty"`
	PII         bool    `json:"pii,omitempty"`
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
	Kind            StepKind        `json:"kind"`
	Title           string          `json:"title,omitempty"`
	Target          string          `json:"target"`
	Ref             string          `json:"ref,omitempty"`
	Error           string          `json:"error,omitempty"`
	Examples        json.RawMessage `json:"examples,omitempty"`
	ExpectEmptyList bool            `json:"expect_empty_list,omitempty"`
	Fields          []Field         `json:"fields,omitempty"`
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

// Build lowers a validated source document into the canonical Model. It is a
// pure transformation: parsing and validation have already succeeded.
func Build(doc *validator.ValidatedDocument) *Model {
	sourceDocument := doc.Source()
	content, _ := syntax.Content(sourceDocument.Parsed().Body(), syntax.ModelSchema())
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
			model.Contexts = append(model.Contexts, decodeContext(block, sourceDocument))
		case "chapter":
			model.Chapters = append(model.Chapters, decodeChapter(block))
		case "hotspot":
			model.Hotspots = append(model.Hotspots, decodeHotspot(block))
		case "state_change", "state_view", "automation", "translation":
			workflow, edges := decodeWorkflow(block, sourceDocument)
			model.Workflows = append(model.Workflows, workflow)
			model.Edges = append(model.Edges, edges...)
		}
	}
	return model
}

func decodeActor(block *hcl.Block) Actor {
	content, _ := syntax.Content(block.Body, syntax.ActorSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	authRequired, _ := boolValue(content.Attributes["auth_required"])
	return Actor{ID: block.Labels[0], Title: title, TitleExplicit: explicit, AuthRequired: authRequired, Description: stringValue(content.Attributes["description"])}
}

func decodeOwner(block *hcl.Block) Owner {
	content, _ := syntax.Content(block.Body, syntax.OwnerSchema(block.Type))
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	external, _ := boolValue(content.Attributes["external"])
	return Owner{Kind: block.Type, ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"]), External: external}
}

func decodeContext(block *hcl.Block, document *sourcepkg.Document) Context {
	content, _ := syntax.Content(block.Body, syntax.BoundedContextSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	external, _ := boolValue(content.Attributes["external"])
	context := Context{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"]), External: external, Owner: traversalValue(content.Attributes["owner"])}
	fc := fieldContext{contextID: block.Labels[0], document: document}
	for _, child := range content.Blocks {
		switch child.Type {
		case "aggregate":
			context.Aggregates = append(context.Aggregates, decodeAggregate(child))
		case "field_type":
			context.FieldTypes = append(context.FieldTypes, decodeFieldType(child, fc))
		case "event":
			context.Events = append(context.Events, decodeEvent(child, fc))
		}
	}
	return context
}

func decodeAggregate(block *hcl.Block) Aggregate {
	content, _ := syntax.Content(block.Body, syntax.AggregateSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	return Aggregate{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"])}
}

func decodeFieldType(block *hcl.Block, fc fieldContext) FieldType {
	content, _ := syntax.Content(block.Body, syntax.FieldSchema())
	idAttribute, _ := boolValue(content.Attributes["id_attribute"])
	pii, _ := boolValue(content.Attributes["pii"])
	return FieldType{ID: block.Labels[0], Type: stringValue(content.Attributes["type"]), Cardinality: stringValue(content.Attributes["cardinality"]), IDAttribute: idAttribute, PII: pii, Fields: decodeFields(content.Blocks, fc)}
}

func decodeEvent(block *hcl.Block, fc fieldContext) Event {
	content, _ := syntax.Content(block.Body, syntax.EventSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	return Event{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Semantic: semanticFrom(content.Attributes), Presentation: presentationFrom(content.Attributes), Fields: decodeElementFields(content, fc)}
}

func decodeWorkflow(block *hcl.Block, document *sourcepkg.Document) (Workflow, []Edge) {
	content, _ := syntax.Content(block.Body, syntax.WorkflowSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	workflow := Workflow{Kind: mustWorkflowKind(block.Type), ID: block.Labels[0], Title: title, TitleExplicit: explicit, Status: stringValue(content.Attributes["status"]), Owner: traversalValue(content.Attributes["owner"]), Description: stringValue(content.Attributes["description"])}
	var edges []Edge
	for _, child := range content.Blocks {
		switch child.Type {
		case "command", "readmodel", "screen", "processor", "screen_image", "table":
			element, elementEdges := decodeElement(workflow.ID, child, document)
			workflow.Elements = append(workflow.Elements, element)
			edges = append(edges, elementEdges...)
		case "scenario":
			workflow.Scenarios = append(workflow.Scenarios, decodeScenario(child, document))
		}
	}
	return workflow, edges
}

func decodeElement(workflowID string, block *hcl.Block, document *sourcepkg.Document) (Element, []Edge) {
	content, _ := syntax.Content(block.Body, elementSchemaFor(block.Type))
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	from := traversalList(content.Attributes["from"])
	to := traversalList(content.Attributes["to"])
	element := Element{Kind: mustElementKind(block.Type), ID: block.Labels[0], Title: title, TitleExplicit: explicit, Semantic: semanticFrom(content.Attributes), Presentation: presentationFrom(content.Attributes), Fields: decodeElementFields(content, fieldContext{document: document}), From: from, To: to}
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

func decodeScenario(block *hcl.Block, document *sourcepkg.Document) Scenario {
	content, _ := syntax.Content(block.Body, syntax.ScenarioSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	scenario := Scenario{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"])}
	for _, child := range content.Blocks {
		if child.Type == "comment" {
			comment, _ := syntax.Content(child.Body, syntax.CommentSchema())
			scenario.Comments = append(scenario.Comments, stringValue(comment.Attributes["description"]))
			continue
		}
		step := decodeStep(child, document)
		scenario.Steps = append(scenario.Steps, step)
	}
	return scenario
}

func decodeStep(block *hcl.Block, document *sourcepkg.Document) Step {
	content, _ := syntax.Content(block.Body, syntax.ScenarioStepSchema())
	step := Step{Kind: mustStepKind(block.Type), Title: stringValue(content.Attributes["title"]), Examples: jsonValue(content.Attributes["examples"]), ExpectEmptyList: boolValueOrFalse(content.Attributes["expect_empty_list"]), Fields: decodeElementFields(content, fieldContext{document: document})}
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

func jsonValue(attribute *hcl.Attribute) json.RawMessage {
	if attribute == nil {
		return nil
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() {
		return nil
	}
	encoded, err := ctyjson.Marshal(value, value.Type())
	if err != nil {
		return nil
	}
	return encoded
}

func decodeChapter(block *hcl.Block) Chapter {
	content, _ := syntax.Content(block.Body, syntax.ChapterSchema())
	title, explicit := effectiveTitle(block.Labels[0], content.Attributes["title"])
	return Chapter{ID: block.Labels[0], Title: title, TitleExplicit: explicit, Description: stringValue(content.Attributes["description"]), Workflows: traversalList(content.Attributes["workflows"])}
}

func decodeHotspot(block *hcl.Block) Hotspot {
	content, _ := syntax.Content(block.Body, syntax.HotspotSchema())
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

// fieldContext carries the information a typeless field block needs to infer its
// field_type. contextID is the owning bounded_context for event and subfield
// fields and empty for workflow-element fields, which resolve by unique name.
type fieldContext struct {
	contextID string
	document  *sourcepkg.Document
}

// inferredType synthesizes the canonical field_type reference a typeless field
// block infers from its name, matching the string an explicit type decodes to.
// Load runs only on validated source, so resolution always succeeds here.
func (fc fieldContext) inferredType(name string) string {
	return fc.document.InferFieldType(fc.contextID, name)
}

// canonicalRef expands a two-part context-local field_type reference used in a
// fields = [...] entry into its fully qualified form.
func (fc fieldContext) canonicalRef(reference string) string {
	return sourcepkg.CanonicalFieldTypeReference(fc.contextID, reference)
}

// decodeElementFields merges the shorthand fields = [...] list attribute with
// the field blocks that share its namespace, list entries first.
func decodeElementFields(content *hcl.BodyContent, fc fieldContext) []Field {
	fields := decodeFieldList(content.Attributes["fields"], fc)
	return append(fields, decodeFields(content.Blocks, fc)...)
}

func decodeFieldList(attribute *hcl.Attribute, fc fieldContext) []Field {
	references := traversalList(attribute)
	fields := make([]Field, 0, len(references))
	for _, reference := range references {
		canonical := fc.canonicalRef(reference)
		fields = append(fields, Field{Name: canonical[strings.LastIndex(canonical, ".")+1:], Type: canonical})
	}
	return fields
}

func decodeFields(blocks hcl.Blocks, fc fieldContext) []Field {
	fields := make([]Field, 0, len(blocks))
	for _, block := range blocks {
		if block.Type != "field" && block.Type != "subfield" {
			continue
		}
		content, _ := syntax.Content(block.Body, syntax.FieldSchema())
		optional, _ := boolValue(content.Attributes["optional"])
		technical, _ := boolValue(content.Attributes["technical_attribute"])
		generated, _ := boolValue(content.Attributes["generated"])
		idAttribute, _ := boolValue(content.Attributes["id_attribute"])
		pii, _ := boolValue(content.Attributes["pii"])
		fieldType := typeValue(content.Attributes["type"])
		if content.Attributes["type"] == nil {
			fieldType = fc.inferredType(block.Labels[0])
		}
		fields = append(fields, Field{Name: block.Labels[0], Type: fieldType, Cardinality: stringValue(content.Attributes["cardinality"]), Mapping: stringValue(content.Attributes["mapping"]), Optional: optional, TechnicalAttribute: technical, Generated: generated, IDAttribute: idAttribute, PII: pii, Schema: stringValue(content.Attributes["schema"]), Fields: decodeFields(content.Blocks, fc)})
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
	return sourcepkg.ReferenceString(attribute.Expr)
}

func traversalList(attribute *hcl.Attribute) []string {
	if attribute == nil {
		return nil
	}
	return sourcepkg.ReferenceList(attribute.Expr)
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

// elementSchemaFor selects the syntax package schema for a workflow child
// block kind. screen_image and table each have their own dedicated grammar
// (no shared element attributes), matching how the validator dispatches the
// same block types onto syntax.ScreenImageSchema/syntax.TableSchema.
func elementSchemaFor(kind string) hcl.BodySchema {
	switch kind {
	case "screen_image":
		return syntax.ScreenImageSchema()
	case "table":
		return syntax.TableSchema()
	default:
		return syntax.ElementSchema(kind)
	}
}
