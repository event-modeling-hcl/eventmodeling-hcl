package renderer

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/version"
)

// translationPattern is the view pattern of an automation that consumes an
// external event. It is derived; the language has no translation kind.
const translationPattern = "translation"

type fieldTypeEntry struct {
	typeName    string
	cardinality string
	id          bool
	pii         bool
}

type eventEntry struct {
	contextID string
	event     model.Event
	external  bool
}

// eventPlacement names the slice that shows an event: an index into the board
// slices, or into the off-board slices when offBoard is set.
type eventPlacement struct {
	offBoard   bool
	sliceIndex int
	given      bool
}

type adapter struct {
	model        *model.Model
	fieldTypes   map[string]fieldTypeEntry
	events       map[string]eventEntry
	titles       map[string]string
	owners       map[string]string
	aggregates   map[string]string
	offBoard     map[string]bool
	translations map[string]bool
}

func BuildViewModel(filename string, source *model.Model) *ViewModel {
	a := newAdapter(source)
	view := &ViewModel{
		Title:    modelName(filename),
		Version:  version.Spec,
		Actors:   map[string]Actor{},
		Contexts: map[string]Context{},
		Slices:   []Slice{},
		OffBoard: []Slice{},
	}
	for _, actor := range source.Actors {
		view.Actors[actor.ID] = Actor{Title: label(actor.Title, actor.ID, actor.TitlePresent), AuthRequired: actor.AuthRequired}
	}
	for _, context := range source.Contexts {
		view.Contexts[context.ID] = Context{Title: label(context.Title, context.ID, context.TitlePresent), External: context.External}
	}
	for _, workflow := range source.Workflows {
		if a.offBoard[workflow.ID] {
			view.OffBoard = append(view.OffBoard, a.adaptWorkflow(workflow))
		} else {
			view.Slices = append(view.Slices, a.adaptWorkflow(workflow))
		}
	}
	view.Edges = a.adaptEdges()
	a.placeEvents(view)
	assignLayout(view.Slices, view.Edges)
	assignLayout(view.OffBoard, view.Edges)
	view.Chapters = a.adaptChapters(source.Chapters)
	view.ContextMap = a.buildContextMap()
	view.Hotspots = a.adaptHotspots(source.Hotspots)
	return view
}

// label is the display text of a construct: its title, or its raw ID when the
// source states no title.
func label(title, id string, present bool) string {
	if title == "" && !present {
		return id
	}
	return title
}

func newAdapter(source *model.Model) *adapter {
	a := &adapter{
		model:        source,
		fieldTypes:   map[string]fieldTypeEntry{},
		events:       map[string]eventEntry{},
		titles:       map[string]string{},
		owners:       map[string]string{},
		aggregates:   map[string]string{},
		offBoard:     make(map[string]bool, len(source.OffBoard)),
		translations: map[string]bool{},
	}
	for _, id := range source.OffBoard {
		a.offBoard[id] = true
	}
	for _, edge := range source.Edges {
		if edge.External {
			a.translations[edge.WorkflowID] = true
		}
	}
	for _, actor := range source.Actors {
		a.owners["actor."+actor.ID] = label(actor.Title, actor.ID, actor.TitlePresent)
	}
	for _, owner := range source.Owners {
		a.owners[owner.Kind+"."+owner.ID] = label(owner.Title, owner.ID, owner.TitlePresent)
	}
	for _, context := range source.Contexts {
		a.owners["bounded_context."+context.ID] = label(context.Title, context.ID, context.TitlePresent)
		for _, aggregate := range context.Aggregates {
			a.aggregates["aggregate."+context.ID+"."+aggregate.ID] = label(aggregate.Title, aggregate.ID, aggregate.TitlePresent)
		}
		for _, fieldType := range context.FieldTypes {
			a.fieldTypes[context.ID+"."+fieldType.ID] = fieldTypeEntry{
				typeName: fieldType.Type, cardinality: fieldType.Cardinality,
				id: fieldType.IDAttribute, pii: fieldType.PII,
			}
		}
		for _, event := range context.Events {
			address := context.ID + "." + event.ID
			a.events[address] = eventEntry{contextID: context.ID, event: event, external: context.External}
			a.titles["event."+address] = label(event.Title, event.ID, event.TitlePresent)
		}
	}
	for _, workflow := range source.Workflows {
		a.titles["workflow."+workflow.ID] = label(workflow.Title, workflow.ID, workflow.TitlePresent)
		for _, element := range workflow.Elements {
			elementLabel := label(element.Title, element.ID, element.TitlePresent)
			a.titles[string(element.Kind)+"."+workflow.ID+"."+element.ID] = elementLabel
			a.titles[workflow.ID+"::"+string(element.Kind)+"."+element.ID] = elementLabel
		}
	}
	return a
}

// pattern is the view pattern of a workflow: its kind, or the derived
// translation pattern for an automation with an external edge.
func (a *adapter) pattern(workflow model.Workflow) string {
	if workflow.Kind == model.Automation && a.translations[workflow.ID] {
		return translationPattern
	}
	return string(workflow.Kind)
}

func (a *adapter) adaptWorkflow(workflow model.Workflow) Slice {
	slice := Slice{
		ID: workflow.ID, Type: a.pattern(workflow), Title: label(workflow.Title, workflow.ID, workflow.TitlePresent),
		Status: workflow.Status, Owner: a.ownerTitle(workflow.Owner), Description: workflow.Description,
		Elements: []Element{}, Scenarios: []Scenario{},
	}
	for _, source := range workflow.Elements {
		slice.Elements = append(slice.Elements, a.adaptElement(workflow.ID, source))
	}
	for _, source := range workflow.Scenarios {
		slice.Scenarios = append(slice.Scenarios, a.adaptScenario(workflow.ID, source))
	}
	return slice
}

func assignLayout(slices []Slice, edges []Edge) {
	for sliceIndex := range slices {
		slice := &slices[sliceIndex]
		stages := make(map[string]int, len(slice.Elements))
		local := make(map[string]bool, len(slice.Elements))
		for _, element := range slice.Elements {
			local[element.ID] = true
			stages[element.ID] = naturalStage(slice.Type, element)
		}
		for range slice.Elements {
			changed := false
			for _, edge := range edges {
				if !local[edge.From] || !local[edge.To] || stages[edge.To] > stages[edge.From] {
					continue
				}
				stages[edge.To] = stages[edge.From] + 1
				changed = true
			}
			if !changed {
				break
			}
		}
		alignPresentationStages(slice, stages)
		stages = compactStages(stages)
		maxStage := 0
		for elementIndex := range slice.Elements {
			element := &slice.Elements[elementIndex]
			element.Stage = stages[element.ID]
			maxStage = max(maxStage, element.Stage)
		}
		slice.StageCount = maxStage + 1
	}
}

// naturalStage places an element by the slice's view pattern.
func naturalStage(pattern string, element Element) int {
	switch pattern {
	case string(model.StateChange):
		switch element.Kind {
		case "screen", "screen_image":
			return 0
		case "command", "table":
			return 1
		case "event":
			return 2
		}
	case string(model.StateView):
		switch element.Kind {
		case "event":
			return 0
		case "readmodel", "table":
			return 1
		case "screen", "screen_image":
			return 2
		}
	case string(model.Automation), translationPattern:
		switch element.Kind {
		case "event":
			if element.Given {
				return 0
			}
			return 4
		case "readmodel", "table":
			return 1
		case "processor":
			return 2
		case "command":
			return 3
		}
	}
	return 0
}

func alignPresentationStages(slice *Slice, stages map[string]int) {
	screenStage := -1
	semanticStage := -1
	for _, element := range slice.Elements {
		switch element.Kind {
		case "screen":
			if screenStage == -1 || stages[element.ID] < screenStage {
				screenStage = stages[element.ID]
			}
		case "command", "readmodel":
			if semanticStage == -1 || stages[element.ID] < semanticStage {
				semanticStage = stages[element.ID]
			}
		}
	}
	for _, element := range slice.Elements {
		switch {
		case element.Kind == "screen_image" && screenStage >= 0:
			stages[element.ID] = screenStage
		case element.Kind == "table" && semanticStage >= 0:
			stages[element.ID] = semanticStage
		}
	}
}

func compactStages(stages map[string]int) map[string]int {
	values := make([]int, 0, len(stages))
	seen := map[int]bool{}
	for _, stage := range stages {
		if !seen[stage] {
			seen[stage] = true
			values = append(values, stage)
		}
	}
	sort.Ints(values)
	compact := make(map[int]int, len(values))
	for index, stage := range values {
		compact[stage] = index
	}
	result := make(map[string]int, len(stages))
	for id, stage := range stages {
		result[id] = compact[stage]
	}
	return result
}

func (a *adapter) adaptElement(workflowID string, source model.Element) Element {
	contextID := contextFromAggregate(source.Semantic.Aggregate)
	return Element{
		ID:       elementNodeID(workflowID, string(source.Kind), source.ID),
		Kind:     string(source.Kind),
		Title:    label(source.Title, source.ID, source.TitlePresent),
		Actor:    lastPart(source.Semantic.Actor),
		ImageURL: source.Presentation.URL,
		Agg:      lastPart(source.Semantic.Aggregate),
		AggTitle: a.aggregateTitle(source.Semantic.Aggregate),
		API:      source.Semantic.APIEndpoint,
		Question: source.Semantic.Question,
		Tags:     source.Presentation.Tags,
		Fields:   a.adaptFields(source.Fields, contextID),
	}
}

func (a *adapter) adaptFields(source []model.Field, contextID string) []Field {
	fields := make([]Field, 0, len(source))
	for _, sourceField := range source {
		field := Field{Name: sourceField.Name, Type: sourceField.Type, ID: sourceField.IDAttribute, PII: sourceField.PII}
		cardinality := sourceField.Cardinality
		if address, ok := fieldTypeAddress(sourceField.Type, contextID); ok {
			if fieldType, exists := a.fieldTypes[address]; exists {
				field.Type = fieldType.typeName
				field.ID = field.ID || fieldType.id
				field.PII = field.PII || fieldType.pii
				if cardinality == "" {
					cardinality = fieldType.cardinality
				}
			}
		}
		if cardinality == "List" {
			field.Type += "[]"
		}
		fields = append(fields, field)
	}
	return fields
}

func (a *adapter) adaptScenario(workflowID string, source model.Scenario) Scenario {
	scenario := Scenario{Title: label(source.Title, source.ID, source.TitlePresent), Comments: append([]string(nil), source.Comments...)}
	for _, sourceStep := range source.Steps {
		step := a.adaptStep(workflowID, sourceStep)
		switch sourceStep.Kind {
		case model.Given:
			scenario.Given = append(scenario.Given, step)
		case model.When:
			scenario.When = &step
		case model.Then:
			scenario.Then = append(scenario.Then, step)
		}
	}
	return scenario
}

func (a *adapter) adaptStep(workflowID string, source model.Step) Step {
	title := source.Title
	if title == "" && !source.TitlePresent {
		if source.Error != "" {
			title = source.Error
		} else {
			title = a.referenceTitle(workflowID, source.Ref)
		}
	}
	fieldNames := make([]string, 0, len(source.Fields))
	for _, field := range source.Fields {
		fieldNames = append(fieldNames, field.Name)
	}
	return Step{
		Title: title, RefKind: source.Target, Ref: a.referenceTitle(workflowID, source.Ref),
		Examples: source.Examples, Fields: fieldNames, Error: source.Error,
		EmptyList: source.ExpectEmptyList,
	}
}

// referenceTitle is the label of a referenced construct, or the reference's
// raw ID when it names nothing in the model.
func (a *adapter) referenceTitle(workflowID, reference string) string {
	if reference == "" {
		return ""
	}
	if title, ok := a.titles[reference]; ok {
		return title
	}
	if title, ok := a.titles[workflowID+"::"+reference]; ok {
		return title
	}
	return lastPart(reference)
}

// aggregateTitle is the label of an aggregate reference, or "" when there is
// no reference.
func (a *adapter) aggregateTitle(reference string) string {
	if reference == "" {
		return ""
	}
	if title, ok := a.aggregates[reference]; ok {
		return title
	}
	return lastPart(reference)
}

func (a *adapter) buildContextMap() ContextMap {
	contextMap := ContextMap{
		Nodes: make([]ContextNode, 0, len(a.model.Contexts)),
		Edges: make([]ContextEdge, 0),
	}
	for _, context := range a.model.Contexts {
		contextMap.Nodes = append(contextMap.Nodes, ContextNode{
			ID: context.ID, Title: label(context.Title, context.ID, context.TitlePresent), External: context.External,
			Team: a.ownerTitle(context.Owner), Events: len(context.Events), Aggregates: len(context.Aggregates),
		})
	}

	edges := map[string]*ContextEdge{}
	workflows := map[string]map[string]bool{}
	for _, workflow := range a.model.Workflows {
		produced := map[string][]int{}
		consumed := map[string][]int{}
		for _, edge := range a.model.Edges {
			if edge.WorkflowID != workflow.ID {
				continue
			}
			recordEventReference(edge.To, 0, produced)
			recordEventReference(edge.From, 0, consumed)
		}
		for _, scenario := range workflow.Scenarios {
			for _, step := range scenario.Steps {
				if step.Target != "event" {
					continue
				}
				switch step.Kind {
				case model.Given:
					recordEventReference(step.Ref, 0, consumed)
				case model.Then:
					recordEventReference(step.Ref, 0, produced)
				case model.When:
				}
			}
		}

		homeContexts := map[string]bool{}
		for address := range produced {
			if entry, ok := a.events[address]; ok && !entry.external {
				homeContexts[entry.contextID] = true
			}
		}
		home := ""
		for _, context := range a.model.Contexts {
			if homeContexts[context.ID] {
				home = context.ID
				break
			}
		}
		if home == "" {
			for _, element := range workflow.Elements {
				if ctx := contextFromAggregate(element.Semantic.Aggregate); ctx != "" {
					home = ctx
					break
				}
			}
		}
		if home == "" {
			continue
		}

		for address := range consumed {
			entry, ok := a.events[address]
			if !ok || entry.contextID == home {
				continue
			}
			key := entry.contextID + ">" + home
			edge := edges[key]
			if edge == nil {
				edge = &ContextEdge{
					Upstream: entry.contextID, Downstream: home, Pattern: "customer_supplier",
				}
				edges[key] = edge
				workflows[key] = map[string]bool{}
			}
			if a.translations[workflow.ID] {
				edge.Pattern = "anticorruption"
			}
			workflows[key][workflow.ID] = true
		}
	}

	contextMap.Edges = make([]ContextEdge, 0, len(edges))
	for key, edge := range edges {
		edge.Via = make([]string, 0, len(workflows[key]))
		for workflowID := range workflows[key] {
			edge.Via = append(edge.Via, workflowID)
		}
		sort.Strings(edge.Via)
		contextMap.Edges = append(contextMap.Edges, *edge)
	}
	sort.Slice(contextMap.Edges, func(i, j int) bool {
		if contextMap.Edges[i].Upstream == contextMap.Edges[j].Upstream {
			return contextMap.Edges[i].Downstream < contextMap.Edges[j].Downstream
		}
		return contextMap.Edges[i].Upstream < contextMap.Edges[j].Upstream
	})
	return contextMap
}

func (a *adapter) adaptEdges() []Edge {
	edges := make([]Edge, 0, len(a.model.Edges))
	seen := map[string]bool{}
	for _, source := range a.model.Edges {
		from := nodeIDForReference(source.WorkflowID, source.From)
		to := nodeIDForReference(source.WorkflowID, source.To)
		key := from + ">" + to
		if from == "" || to == "" || seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, Edge{From: from, To: to})
	}
	return edges
}

// placeEvents shows each catalog event once. A board slice wins over an
// off-board one, so an event that the story touches stays in the story: the
// last board producer, else the first board consumer (as upstream input).
// Events that only off-board workflows touch go to an off-board slice by the
// same rule.
func (a *adapter) placeEvents(view *ViewModel) {
	positions := make(map[string]eventPlacement, len(view.Slices)+len(view.OffBoard))
	for index, slice := range view.Slices {
		positions[slice.ID] = eventPlacement{sliceIndex: index}
	}
	for index, slice := range view.OffBoard {
		positions[slice.ID] = eventPlacement{offBoard: true, sliceIndex: index}
	}
	boardProducers, boardConsumers := map[string][]int{}, map[string][]int{}
	offProducers, offConsumers := map[string][]int{}, map[string][]int{}
	record := func(workflowID, reference string, produced bool) {
		position, ok := positions[workflowID]
		if !ok {
			return
		}
		switch {
		case position.offBoard && produced:
			recordEventReference(reference, position.sliceIndex, offProducers)
		case position.offBoard:
			recordEventReference(reference, position.sliceIndex, offConsumers)
		case produced:
			recordEventReference(reference, position.sliceIndex, boardProducers)
		default:
			recordEventReference(reference, position.sliceIndex, boardConsumers)
		}
	}
	for _, edge := range a.model.Edges {
		record(edge.WorkflowID, edge.To, true)
		record(edge.WorkflowID, edge.From, false)
	}
	for _, workflow := range a.model.Workflows {
		for _, scenario := range workflow.Scenarios {
			for _, step := range scenario.Steps {
				if step.Target != "event" {
					continue
				}
				switch step.Kind {
				case model.Given:
					record(workflow.ID, step.Ref, false)
				case model.Then:
					record(workflow.ID, step.Ref, true)
				case model.When:
				}
			}
		}
	}

	placements := map[string]eventPlacement{}
	for address := range a.events {
		switch {
		case len(boardProducers[address]) > 0:
			placements[address] = eventPlacement{sliceIndex: maxIndex(boardProducers[address])}
		case len(boardConsumers[address]) > 0:
			placements[address] = eventPlacement{sliceIndex: minIndex(boardConsumers[address]), given: true}
		case len(offProducers[address]) > 0:
			placements[address] = eventPlacement{offBoard: true, sliceIndex: maxIndex(offProducers[address])}
		case len(offConsumers[address]) > 0:
			placements[address] = eventPlacement{offBoard: true, sliceIndex: minIndex(offConsumers[address]), given: true}
		}
	}
	addresses := make([]string, 0, len(placements))
	for address := range placements {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	for _, address := range addresses {
		placement := placements[address]
		entry := a.events[address]
		slices := view.Slices
		if placement.offBoard {
			slices = view.OffBoard
		}
		slices[placement.sliceIndex].Elements = append(slices[placement.sliceIndex].Elements, Element{
			ID:       eventNodeID(address),
			Kind:     "event",
			Title:    label(entry.event.Title, entry.event.ID, entry.event.TitlePresent),
			Agg:      lastPart(entry.event.Semantic.Aggregate),
			AggTitle: a.aggregateTitle(entry.event.Semantic.Aggregate),
			Ctx:      entry.contextID,
			External: entry.external,
			Given:    placement.given,
			Tags:     entry.event.Presentation.Tags,
			Fields:   a.adaptFields(entry.event.Fields, entry.contextID),
		})
	}
}

func (a *adapter) adaptHotspots(source []model.Hotspot) []Hotspot {
	hotspots := make([]Hotspot, 0, len(source))
	for _, hotspot := range source {
		hotspots = append(hotspots, Hotspot{
			ID: hotspot.ID, OnID: nodeIDForGlobalReference(hotspot.On), Target: hotspot.On,
			Status: hotspot.Status, Question: hotspot.Question,
		})
	}
	return hotspots
}

// adaptChapters keeps the model's chapter order (by ID) and lists each
// chapter's board workflows in board order.
func (a *adapter) adaptChapters(source []model.Chapter) []Chapter {
	chapters := make([]Chapter, 0, len(source))
	for _, chapter := range source {
		members := make(map[string]bool, len(chapter.Workflows))
		for _, workflow := range chapter.Workflows {
			members[lastPart(workflow)] = true
		}
		slices := make([]string, 0, len(chapter.Workflows))
		for _, workflow := range a.model.Workflows {
			if members[workflow.ID] && !a.offBoard[workflow.ID] {
				slices = append(slices, workflow.ID)
			}
		}
		chapters = append(chapters, Chapter{ID: chapter.ID, Title: label(chapter.Title, chapter.ID, chapter.TitlePresent), Slices: slices})
	}
	return chapters
}

func (a *adapter) ownerTitle(reference string) string {
	if title, ok := a.owners[reference]; ok {
		return title
	}
	return lastPart(reference)
}

func nodeIDForReference(workflowID, reference string) string {
	parts := strings.Split(reference, ".")
	switch len(parts) {
	case 2:
		return elementNodeID(workflowID, parts[0], parts[1])
	case 3:
		if parts[0] == "event" {
			return eventNodeID(parts[1] + "." + parts[2])
		}
		return elementNodeID(parts[1], parts[0], parts[2])
	default:
		return ""
	}
}

func nodeIDForGlobalReference(reference string) string {
	parts := strings.Split(reference, ".")
	if len(parts) == 2 && parts[0] == "workflow" {
		return "slice__" + parts[1]
	}
	return nodeIDForReference("", reference)
}

func elementNodeID(workflowID, kind, id string) string {
	return workflowID + "__" + kind + "__" + id
}

func eventNodeID(address string) string {
	return "event__" + strings.ReplaceAll(address, ".", "__")
}

func recordEventReference(reference string, index int, target map[string][]int) {
	parts := strings.Split(reference, ".")
	if len(parts) == 3 && parts[0] == "event" {
		address := parts[1] + "." + parts[2]
		target[address] = append(target[address], index)
	}
}

func fieldTypeAddress(reference, contextID string) (string, bool) {
	parts := strings.Split(reference, ".")
	if len(parts) == 3 && parts[0] == "field_type" {
		return parts[1] + "." + parts[2], true
	}
	if len(parts) == 2 && parts[0] == "field_type" && contextID != "" {
		return contextID + "." + parts[1], true
	}
	return "", false
}

func contextFromAggregate(reference string) string {
	parts := strings.Split(reference, ".")
	if len(parts) == 3 && parts[0] == "aggregate" {
		return parts[1]
	}
	return ""
}

func lastPart(reference string) string {
	parts := strings.Split(reference, ".")
	return parts[len(parts)-1]
}

func minIndex(indices []int) int {
	minimum := indices[0]
	for _, index := range indices[1:] {
		minimum = min(minimum, index)
	}
	return minimum
}

func maxIndex(indices []int) int {
	maximum := indices[0]
	for _, index := range indices[1:] {
		maximum = max(maximum, index)
	}
	return maximum
}

// modelName names the model by its path, unchanged: a file path gives its file
// name without the .em.hcl suffix. A path without that suffix is a folder
// model and gives the folder's own name, resolving "." and "/" through the
// absolute path.
func modelName(filename string) string {
	base := filepath.Base(filename)
	if strings.HasSuffix(base, ".em.hcl") {
		return strings.TrimSuffix(base, ".em.hcl")
	}
	base = filepath.Base(filepath.Clean(filename))
	if base == "." || base == string(filepath.Separator) {
		if absolute, err := filepath.Abs(filename); err == nil {
			base = filepath.Base(absolute)
		}
	}
	return base
}
