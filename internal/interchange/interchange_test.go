package interchange_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/app"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/interchange"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
)

// parsed decodes an exported document. Export order is canonical, so two
// exports of the same model compare equal without any re-sorting.
func parsed(t *testing.T, encoded string) *interchange.Document {
	t.Helper()
	document, err := interchange.ParseDocument([]byte(encoded))
	if err != nil {
		t.Fatalf("exported JSON does not parse: %v", err)
	}
	return document
}

func TestExportImport_RoundTripsEveryShippedExample(t *testing.T) {
	// Test files
	filePaths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.em.hcl"))
	if err != nil || len(filePaths) == 0 {
		t.Fatalf("find examples: %v", err)
	}
	// Test folders
	entries, err := os.ReadDir(filepath.Join("..", "..", "examples"))
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}
	var folderPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			folderPaths = append(folderPaths, filepath.Join("..", "..", "examples", entry.Name()))
		}
	}

	// Combine paths
	paths := append(filePaths, folderPaths...)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			exported := app.ExportPath(path)
			if exported.Diagnostics.HasErrors() {
				t.Fatalf("export diagnostics = %#v", exported.Diagnostics)
			}
			imported := app.Import("model.json", []byte(exported.JSON), "imported.em.hcl")
			if imported.Diagnostics.HasErrors() {
				t.Fatalf("imported model is invalid: %#v\n%s", imported.Diagnostics, imported.Source)
			}
			reexported := app.Export("imported.em.hcl", []byte(imported.Source))
			if reexported.Diagnostics.HasErrors() {
				t.Fatalf("re-export diagnostics = %#v", reexported.Diagnostics)
			}
			if first, second := parsed(t, exported.JSON), parsed(t, reexported.JSON); !reflect.DeepEqual(first, second) {
				t.Fatalf("re-export differs from export\nfirst:  %s\nsecond: %s", exported.JSON, reexported.JSON)
			}
		})
	}
}

func TestImport_ForeignDocumentProducesValidModel(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "nebulit.json"))
	if err != nil {
		t.Fatal(err)
	}
	imported := app.Import("nebulit.json", data, "cart.em.hcl")
	if imported.Diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %#v\n%s", imported.Diagnostics, imported.Source)
	}
	built, diagnostics := app.ValidatedModel("cart.em.hcl", []byte(imported.Source), app.Valid)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	contexts := map[string]model.Context{}
	for _, context := range built.Contexts {
		contexts[context.ID] = context
	}
	if events := contexts["cart"].Events; len(events) != 1 || events[0].ID != "item_added" {
		t.Fatalf("cart events = %#v, want the copied Item Added event declared once", events)
	}
	if external := contexts["external"]; !external.External || len(external.Events) != 1 {
		t.Fatalf("external context = %#v, want an external context holding Inventory Changed", external)
	}

	readModel := elementOfKind(t, built.Workflows[1], model.ReadModel)
	if want := []string{"event.cart.item_added", "event.external.inventory_changed"}; !reflect.DeepEqual(readModel.From, want) {
		t.Fatalf("readmodel from = %v, want %v (copy resolved through linkedId)", readModel.From, want)
	}

	command := elementOfKind(t, built.Workflows[0], model.Command)
	examples := map[string]string{}
	types := map[string]string{}
	for _, field := range command.Fields {
		examples[field.Name] = string(field.Example)
		types[field.Name] = field.Type
	}
	if examples["quantity"] != "2" || examples["price"] != "9.99" || types["price"] != "Double" {
		t.Fatalf("command fields = %#v, want numeric examples decoded and Number mapped to Double", command.Fields)
	}
	if command.Semantic.Aggregate != "aggregate.cart.cart" {
		t.Fatalf("command aggregate = %q", command.Semantic.Aggregate)
	}

	steps := built.Workflows[0].Scenarios[0].Steps
	if last := steps[len(steps)-1]; last.Target != "error" || last.Error != "Cart is full" {
		t.Fatalf("last step = %#v, want SPEC_ERROR imported as an error step", last)
	}
}

// elementOfKind returns the workflow's single element of kind; elements are
// ID-sorted, so tests find them by kind rather than position.
func elementOfKind(t *testing.T, workflow model.Workflow, kind model.ElementKind) model.Element {
	t.Helper()
	for _, element := range workflow.Elements {
		if element.Kind == kind {
			return element
		}
	}
	t.Fatalf("workflow.%s has no %s", workflow.ID, kind)
	return model.Element{}
}

// conformingSlice is the smallest schema-valid slice; schema-violation cases
// replace one part of it.
const conformingSlice = `{"id":"s","title":"S","sliceType":"STATE_CHANGE","index":%s,"commands":[%s],"events":[],"readmodels":[],"screens":[],"processors":[],"tables":[],"specifications":[%s]%s}`

const conformingCommand = `{"id":"c","title":"C","type":"COMMAND","fields":[{"name":"v","type":"String","example":"x"}],"dependencies":[]}`

func sliceDocument(index, command, specification, extra string) string {
	return `{"slices":[` + fmt.Sprintf(conformingSlice, index, command, specification, extra) + `]}`
}

// TestParseDocument_EnforcesTheInterchangeSchema mirrors ajv verdicts for
// eventmodeling.schema.json: unknown properties, missing required properties,
// values outside an enum and mistyped values are all rejected with their path.
func TestParseDocument_EnforcesTheInterchangeSchema(t *testing.T) {
	for name, test := range map[string]struct{ input, violation string }{
		"unknown root property":  {`{"slices":[],"flows":[]}`, `/: must NOT have additional property "flows"`},
		"unknown field property": {sliceDocument("0", `{"id":"c","title":"C","type":"COMMAND","fields":[{"name":"v","type":"String","excludeFromApi":false}],"dependencies":[]}`, "", ""), `/slices/0/commands/0/fields/0: must NOT have additional property "excludeFromApi"`},
		"missing sliceType":      {`{"slices":[{"id":"s","title":"S","commands":[],"events":[],"readmodels":[],"screens":[],"processors":[],"tables":[],"specifications":[]}]}`, `/slices/0: must have required property "sliceType"`},
		"missing element fields": {sliceDocument("0", `{"id":"c","title":"C","type":"COMMAND","dependencies":[]}`, "", ""), `/slices/0/commands/0: must have required property "fields"`},
		"missing spec linkedId":  {sliceDocument("0", conformingCommand, `{"id":"p","title":"P","given":[],"when":[],"then":[]}`, ""), `/slices/0/specifications/0: must have required property "linkedId"`},
		"misspelled actor auth":  {sliceDocument("0", conformingCommand, "", `,"actors":[{"name":"Staff","authzRequired":false}]`), `/slices/0/actors/0: must have required property "authRequired"`},
		"unknown element type":   {sliceDocument("0", `{"id":"c","title":"C","type":"HTML_SCREEN","fields":[],"dependencies":[]}`, "", ""), `/slices/0/commands/0/type: must be equal to one of the allowed values`},
		"unknown direction":      {sliceDocument("0", `{"id":"c","title":"C","type":"COMMAND","fields":[],"dependencies":[{"id":"e","type":"SIDEWAYS","title":"E","elementType":"EVENT"}]}`, "", ""), `/slices/0/commands/0/dependencies/0/type: must be equal to one of the allowed values`},
		"list example":           {sliceDocument("0", `{"id":"c","title":"C","type":"COMMAND","fields":[{"name":"v","type":"Custom","example":[{"a":1}]}],"dependencies":[]}`, "", ""), `/slices/0/commands/0/fields/0/example: must be string or object`},
		"object trigger":         {sliceDocument("0", `{"id":"c","title":"C","type":"COMMAND","fields":[],"dependencies":[],"triggers":[{"id":"e"}]}`, "", ""), `/slices/0/commands/0/triggers/0: must be string`},
		"fractional index":       {sliceDocument("1.5", conformingCommand, "", ""), `/slices/0/index: must be integer`},
		"null required string":   {`{"slices":[{"id":"s","title":null,"sliceType":"STATE_VIEW","commands":[],"events":[],"readmodels":[],"screens":[],"processors":[],"tables":[],"specifications":[]}]}`, `/slices/0/title: must be string`},
		"missing slices":         {`{}`, `/: must have required property "slices"`},
		"null slices":            {`{"slices": null}`, `/slices: must be array`},
		"trailing document":      {`{"slices": []} {"slices": []}`, `trailing content`},
		"trailing garbage":       {`{"slices": []} garbage`, `trailing content`},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := interchange.ParseDocument([]byte(test.input))
			if err == nil || !strings.Contains(err.Error(), test.violation) {
				t.Fatalf("ParseDocument error = %v, want violation %q", err, test.violation)
			}
		})
	}
}

// JSON Schema treats 2.0 as an integer and service as nullable; ajv accepts
// both, so the importer must too.
func TestParseDocument_AcceptsSchemaValidEdgeValues(t *testing.T) {
	input := sliceDocument("2.0", `{"id":"c","title":"C","type":"COMMAND","fields":[],"dependencies":[],"service":null}`, "", "")
	document, err := interchange.ParseDocument([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if index := document.Slices[0].Index; index == nil || *index != 2 {
		t.Fatalf("index = %v, want 2", index)
	}
}

// Every shipped example must export JSON that the interchange schema accepts,
// so the output can be validated with ajv and imported by other tools.
func TestExport_EveryShippedExampleConformsToTheSchema(t *testing.T) {
	// Test files
	filePaths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.em.hcl"))
	if err != nil || len(filePaths) == 0 {
		t.Fatalf("examples: %v %v", filePaths, err)
	}
	// Test folders
	entries, err := os.ReadDir(filepath.Join("..", "..", "examples"))
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}
	var folderPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			folderPaths = append(folderPaths, filepath.Join("..", "..", "examples", entry.Name()))
		}
	}

	// Combine paths
	paths := append(filePaths, folderPaths...)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			exported := app.ExportPath(path)
			if exported.Diagnostics.HasErrors() {
				t.Fatalf("export diagnostics = %#v", exported.Diagnostics)
			}
			if violations := interchange.CheckConformance([]byte(exported.JSON)); len(violations) > 0 {
				t.Fatalf("export violates the schema: %v", violations)
			}
		})
	}
}

func TestExport_FieldExamplesUseOnlyStringsOrObjects(t *testing.T) {
	examples := map[string]struct {
		input string
		want  string
	}{
		"string": {`"sample"`, `"sample"`},
		"object": {`{"value":2}`, `{"value":2}`},
		"number": {`2`, `"2"`},
		"bool":   {`true`, `"true"`},
		"list":   {`[1,null]`, `"[1,null]"`},
		"null":   {`null`, `"null"`},
	}
	for name, example := range examples {
		t.Run(name, func(t *testing.T) {
			document, _, err := interchange.Export(&model.Model{Workflows: []model.Workflow{{
				ID: "save", Kind: model.StateChange, Title: "Save",
				Elements: []model.Element{{ID: "save", Kind: model.Command, Title: "Save",
					Fields: []model.Field{{Name: "value", Type: "Custom", Example: json.RawMessage(example.input)}}}},
			}}})
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(document.Slices[0].Commands[0].Fields[0].Example, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(example.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("exported example = %#v, want %#v", got, want)
			}
		})
	}
}

func TestExport_DisclosesCatalogDeclarationsWithoutASchemaHome(t *testing.T) {
	document, warnings, err := interchange.Export(&model.Model{
		Actors: []model.Actor{{ID: "staff", Title: "Staff"}},
		Contexts: []model.Context{{ID: "orders",
			Aggregates: []model.Aggregate{{ID: "order", Title: "Order"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Slices) != 0 {
		t.Fatalf("catalog-only model exported slices: %+v", document.Slices)
	}
	for _, want := range []string{"actor.staff", "aggregate.orders.order"} {
		if !strings.Contains(strings.Join(warnings, "\n"), want) {
			t.Errorf("catalog loss warning missing %q: %v", want, warnings)
		}
	}
}

// A JSON round trip must keep the author order of to/from lists and the
// identity of bounded contexts, including an external context that shares the
// display name of an internal one.
func TestExportImport_PreservesFlowOrderAndContextIdentity(t *testing.T) {
	source := []byte(`board {
  workflows = [workflow.place, workflow.quotes, workflow.accept_quote]
}
bounded_context "orders" {
  title = "Orders"
  event "placed" {}
  event "rejected" {}
}
bounded_context "orders_external" {
  title    = "Orders"
  external = true
  event "quoted" {}
}
state_change "place" {
  owner = bounded_context.orders
  command "place" {
    api_endpoint = "POST /orders"
    to           = [event.orders.rejected, event.orders.placed]
  }
}
state_view "quotes" {
  owner = bounded_context.orders
  readmodel "quotes" {
    question = "Which quotes exist?"
    from     = [event.orders_external.quoted, event.orders.rejected]
  }
}
automation "accept_quote" {
  owner = bounded_context.orders_external
  processor "accept" {
    from = [event.orders_external.quoted]
    to   = [command.place]
  }
  command "place" {
    to = [event.orders.placed]
  }
}`)
	original, diagnostics := app.ValidatedModel("orders.em.hcl", source, app.Valid)
	if diagnostics.HasErrors() {
		t.Fatalf("source diagnostics: %+v", diagnostics)
	}
	exported := app.Export("orders.em.hcl", source)
	imported := app.Import("orders.json", []byte(exported.JSON), "again.em.hcl")
	again, diagnostics := app.ValidatedModel("again.em.hcl", []byte(imported.Source), app.Valid)
	if diagnostics.HasErrors() {
		t.Fatalf("reimport diagnostics: %+v\n%s", diagnostics, imported.Source)
	}
	for workflow := range original.Workflows {
		if before, after := original.Workflows[workflow].Owner, again.Workflows[workflow].Owner; before != after {
			t.Fatalf("workflow %s owner %s became %s", original.Workflows[workflow].ID, before, after)
		}
		elements := map[string]model.Element{}
		for _, element := range again.Workflows[workflow].Elements {
			elements[element.ID] = element
		}
		for _, before := range original.Workflows[workflow].Elements {
			after := elements[before.ID]
			if !reflect.DeepEqual(before.To, after.To) || !reflect.DeepEqual(before.From, after.From) {
				t.Fatalf("%s flows changed: to %v -> %v, from %v -> %v", before.ID, before.To, after.To, before.From, after.From)
			}
		}
	}
	type identity struct {
		id, title string
		external  bool
	}
	contexts := func(m *model.Model) []identity {
		var result []identity
		for _, context := range m.Contexts {
			result = append(result, identity{context.ID, context.Title, context.External})
		}
		return result
	}
	if got, want := contexts(again), contexts(original); !reflect.DeepEqual(got, want) {
		t.Fatalf("contexts = %+v, want %+v", got, want)
	}
}

// TRANSLATION is a derived view: import writes an automation, the model marks
// its consumed external event edge, and export derives TRANSLATION again.
func TestImportExport_TranslationRoundTripsThroughAutomation(t *testing.T) {
	input := []byte(`{"slices":[{"id":"workflow.accept","title":"Accept","context":"orders","sliceType":"TRANSLATION",
		"events":[
			{"id":"event.partner.quoted","title":"Quoted","type":"EVENT","modelContext":"partner","context":"EXTERNAL","fields":[],"dependencies":[{"id":"processor.accept.accept","type":"OUTBOUND","title":"Accept","elementType":"AUTOMATION"}]},
			{"id":"event.orders.accepted","title":"Accepted","type":"EVENT","modelContext":"orders","context":"INTERNAL","fields":[],"dependencies":[{"id":"command.accept.accept","type":"INBOUND","title":"Accept","elementType":"COMMAND"}]}],
		"processors":[{"id":"processor.accept.accept","title":"Accept","type":"AUTOMATION","fields":[],"dependencies":[{"id":"event.partner.quoted","type":"INBOUND","title":"Quoted","elementType":"EVENT"},{"id":"command.accept.accept","type":"OUTBOUND","title":"Accept","elementType":"COMMAND"}]}],
		"commands":[{"id":"command.accept.accept","title":"Accept","type":"COMMAND","fields":[],"dependencies":[{"id":"processor.accept.accept","type":"INBOUND","title":"Accept","elementType":"AUTOMATION"},{"id":"event.orders.accepted","type":"OUTBOUND","title":"Accepted","elementType":"EVENT"}]}],
		"readmodels":[],"screens":[],"tables":[],"specifications":[]}]}`)
	imported := app.Import("translation.json", input, "translation.em.hcl")
	if imported.Diagnostics.HasErrors() {
		t.Fatalf("import diagnostics: %+v\n%s", imported.Diagnostics, imported.Source)
	}
	if !strings.Contains(imported.Source, `automation "accept"`) || strings.Contains(imported.Source, "translation") {
		t.Fatalf("TRANSLATION slice not imported as an automation:\n%s", imported.Source)
	}
	built, diagnostics := app.ValidatedModel("translation.em.hcl", []byte(imported.Source), app.Valid)
	if diagnostics.HasErrors() {
		t.Fatalf("model diagnostics: %+v", diagnostics)
	}
	external := false
	for _, edge := range built.Edges {
		external = external || edge.WorkflowID == "accept" && edge.External && edge.From == "event.partner.quoted"
	}
	if built.Workflows[0].Kind != model.Automation || !external {
		t.Fatalf("automation does not consume the external event: %+v %+v", built.Workflows[0], built.Edges)
	}
	exported := app.Export("translation.em.hcl", []byte(imported.Source))
	if exported.Diagnostics.HasErrors() {
		t.Fatalf("export diagnostics: %+v", exported.Diagnostics)
	}
	if got := parsed(t, exported.JSON).Slices[0].SliceType; got != "TRANSLATION" {
		t.Fatalf("sliceType = %q, want TRANSLATION", got)
	}
}

func TestExport_AutomationWithoutExternalEventIsAutomation(t *testing.T) {
	source := []byte(`board { workflows = [workflow.react] }
bounded_context "orders" {
  event "placed" {}
  event "confirmed" {}
}
automation "react" {
  processor "confirm" {
    from = [event.orders.placed]
    to   = [command.confirm]
  }
  command "confirm" {
    to = [event.orders.confirmed]
  }
}`)
	exported := app.Export("react.em.hcl", source)
	if exported.Diagnostics.HasErrors() {
		t.Fatalf("export diagnostics: %+v", exported.Diagnostics)
	}
	if got := parsed(t, exported.JSON).Slices[0].SliceType; got != "AUTOMATION" {
		t.Fatalf("sliceType = %q, want AUTOMATION", got)
	}
}

func TestExport_OffBoardWorkflowsFollowTheBoardInIDOrder(t *testing.T) {
	source := []byte(`board { workflows = [workflow.middle] }
state_change "zeta" {
  command "zeta" { api_endpoint = "POST /zeta" }
}
state_change "middle" {
  command "middle" { api_endpoint = "POST /middle" }
}
state_change "alpha" {
  command "alpha" { api_endpoint = "POST /alpha" }
}`)
	exported := app.Export("offboard.em.hcl", source)
	if exported.Diagnostics.HasErrors() {
		t.Fatalf("export diagnostics: %+v", exported.Diagnostics)
	}
	var order []string
	for index, slice := range parsed(t, exported.JSON).Slices {
		if slice.Index == nil || *slice.Index != float64(index) {
			t.Fatalf("slice %s index = %v, want %d", slice.ID, slice.Index, index)
		}
		order = append(order, slice.ID)
	}
	if want := []string{"workflow.middle", "workflow.alpha", "workflow.zeta"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("slice order = %v, want %v", order, want)
	}
	var offBoard []string
	for _, warning := range exported.Warnings {
		if strings.Contains(warning, "off-board") {
			offBoard = append(offBoard, warning)
		}
	}
	if len(offBoard) != 1 || !strings.Contains(offBoard[0], "workflow.alpha, workflow.zeta") || strings.Contains(offBoard[0], "workflow.middle") {
		t.Fatalf("off-board warnings = %v, want one naming workflow.alpha and workflow.zeta", offBoard)
	}
}

func TestImport_WritesABoardInSliceIndexOrder(t *testing.T) {
	slice := func(id string, index int) string {
		return fmt.Sprintf(`{"id":"workflow.%s","title":"%s","index":%d,"sliceType":"STATE_CHANGE","commands":[{"id":"command.%s.%s","title":"Do","type":"COMMAND","fields":[],"dependencies":[]}],"events":[],"readmodels":[],"screens":[],"processors":[],"tables":[],"specifications":[]}`, id, id, index, id, id)
	}
	input := []byte(`{"slices":[` + slice("second", 1) + `,` + slice("first", 0) + `,` + slice("third", 2) + `]}`)
	imported := app.Import("board.json", input, "board.em.hcl")
	if imported.Diagnostics.HasErrors() {
		t.Fatalf("import diagnostics: %+v\n%s", imported.Diagnostics, imported.Source)
	}
	if !strings.HasPrefix(imported.Source, "board {\n  workflows = [workflow.first, workflow.second, workflow.third]\n}\n") {
		t.Fatalf("board not written first in slice index order:\n%s", imported.Source)
	}
	built, diagnostics := app.ValidatedModel("board.em.hcl", []byte(imported.Source), app.Valid)
	if diagnostics.HasErrors() {
		t.Fatalf("model diagnostics: %+v", diagnostics)
	}
	if want := []string{"first", "second", "third"}; !reflect.DeepEqual(built.Board, want) || len(built.OffBoard) != 0 {
		t.Fatalf("board = %v, off-board = %v, want %v", built.Board, built.OffBoard, want)
	}
}

func TestExport_UntitledConstructsUseTheirID(t *testing.T) {
	source := []byte(`board { workflows = [workflow.register] }
actor "clerk" { auth_required = false }
bounded_context "clinic" {
  aggregate "pet" {}
  event "pet_registered" {
    aggregate = aggregate.pet
  }
}
state_change "register" {
  screen "form" {
    actor = actor.clerk
    to    = [command.register_pet]
  }
  command "register_pet" {
    to = [event.clinic.pet_registered]
  }
  scenario "registers" {
    when { command = command.register_pet }
    then { event = event.clinic.pet_registered }
  }
}`)
	exported := app.Export("untitled.em.hcl", source)
	if exported.Diagnostics.HasErrors() {
		t.Fatalf("export diagnostics: %+v", exported.Diagnostics)
	}
	slice := parsed(t, exported.JSON).Slices[0]
	got := []string{slice.Title, slice.Screens[0].Title, slice.Commands[0].Title, slice.Events[0].Title, slice.Events[0].Aggregate, slice.Actors[0].Name, slice.Specifications[0].Title, slice.Specifications[0].When[0].Title}
	if want := []string{"register", "form", "register_pet", "pet_registered", "pet", "clerk", "registers", "register_pet"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("untitled names = %v, want raw IDs %v", got, want)
	}
	for _, warning := range exported.Warnings {
		if strings.Contains(warning, "actor.clerk label") {
			t.Fatalf("untitled actor whose ID is its name warned: %s", warning)
		}
	}
}
