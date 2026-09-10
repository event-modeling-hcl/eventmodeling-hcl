package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	sourcepkg "github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

// loadTestModel follows the production parse, decode, validate, and build
// sequence while keeping model assertions inside this package.
func loadTestModel(t *testing.T, filename string, source []byte) (*Model, hcl.Diagnostics) {
	t.Helper()
	doc, diagnostics := syntax.Parse(filename, source)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	validated, validationDiagnostics := validator.ValidateDecodedDocument(sourcepkg.Decode(doc), validator.Valid)
	if validationDiagnostics.HasErrors() {
		return nil, validationDiagnostics
	}
	return Build(validated), validationDiagnostics
}

func TestBuild_DecodesCompleteModelIntoCanonicalIR(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	loaded, diagnostics := loadTestModel(t, path, source)

	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
	if loaded == nil {
		t.Fatal("model = nil, want decoded model")
	}
	if got, want := len(loaded.Workflows), 4; got != want {
		t.Fatalf("workflow count = %d, want %d", got, want)
	}
	if got := workflowByID(t, loaded, "register_pet").Kind; got != StateChange {
		t.Fatalf("register_pet kind = %q, want %q", got, StateChange)
	}
	if got := workflowByID(t, loaded, "pet_directory").Kind; got != StateView {
		t.Fatalf("pet_directory kind = %q, want %q", got, StateView)
	}
	if got := workflowByID(t, loaded, "notify_owner").Kind; got != Automation {
		t.Fatalf("notify_owner kind = %q, want %q", got, Automation)
	}
	if got := workflowByID(t, loaded, "import_partner_pet").Kind; got != Translation {
		t.Fatalf("import_partner_pet kind = %q, want %q", got, Translation)
	}

	requireEdge(t, loaded, "register_pet", "screen.pet_screen", "command.register_pet_command")
	requireEdge(t, loaded, "pet_directory", "event.clinic.pet_registered", "readmodel.pet_summary")
	requireEdge(t, loaded, "pet_directory", "readmodel.pet_summary", "screen.pet_summary_screen")

	command := elementByID(t, workflowByID(t, loaded, "register_pet"), "register_pet_command")
	if got, want := command.Semantic.Aggregate, "aggregate.clinic.pet"; got != want {
		t.Fatalf("command aggregate = %q, want %q", got, want)
	}
	if got, want := command.Semantic.APIEndpoint, "POST /pets"; got != want {
		t.Fatalf("command API endpoint = %q, want %q", got, want)
	}
	if !command.Semantic.CreatesAggregate {
		t.Fatal("command creates aggregate = false, want true")
	}
	if got, want := command.Presentation.GroupID, "registration"; got != want {
		t.Fatalf("command group id = %q, want %q", got, want)
	}
	wantTags := []string{"write", "pet"}
	if !sameStrings(command.Presentation.Tags, wantTags) {
		t.Fatalf("command tags = %#v, want %#v", command.Presentation.Tags, wantTags)
	}

	scenario := workflowByID(t, loaded, "register_pet").Scenarios[0]
	gotSteps := []StepKind{scenario.Steps[0].Kind, scenario.Steps[1].Kind, scenario.Steps[2].Kind, scenario.Steps[3].Kind}
	wantSteps := []StepKind{Given, When, Then, Then}
	if !sameSteps(gotSteps, wantSteps) {
		t.Fatalf("scenario steps = %#v, want %#v", gotSteps, wantSteps)
	}
}

func TestBuild_DerivesTitlesWithoutChangingExplicitTitles(t *testing.T) {
	source := []byte(`state_change "register_pet" {
  command "register_pet" {}
}
state_change "explicit_title" {
  title = "A custom title"
}`)

	loaded, diagnostics := loadTestModel(t, "model.em.hcl", source)

	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
	derived := workflowByID(t, loaded, "register_pet")
	if derived.Title != "Register Pet" || derived.TitleExplicit {
		t.Fatalf("derived workflow = %#v, want derived Register Pet", derived)
	}
	command := elementByID(t, derived, "register_pet")
	if command.Title != "Register Pet" || command.TitleExplicit {
		t.Fatalf("derived command = %#v, want derived Register Pet", command)
	}
	explicit := workflowByID(t, loaded, "explicit_title")
	if explicit.Title != "A custom title" || !explicit.TitleExplicit {
		t.Fatalf("explicit workflow = %#v, want explicit title", explicit)
	}
}

func TestBuild_PreservesFieldTypeBadgesAndScenarioExamples(t *testing.T) {
	source := []byte(`bounded_context "clinic" {
  field_type "pet_id" {
    type         = "UUID"
    id_attribute = true
    pii          = true
  }

  event "owner_registered" {}
  event "pet_registered" {}
}

state_change "register_pet" {
  command "register_pet" {
    external_trigger = true
    to               = [event.clinic.pet_registered]
  }

  scenario "register_pet" {
    given {
      event    = event.clinic.owner_registered
      examples = [{ owner_id = 9 }]
    }
    when { command = command.register_pet }
    then { event = event.clinic.pet_registered }
  }
}`)

	loaded, diagnostics := loadTestModel(t, "model.em.hcl", source)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}

	fieldType := loaded.Contexts[0].FieldTypes[0]
	if !fieldType.IDAttribute || !fieldType.PII {
		t.Fatalf("field type badges = id:%t pii:%t, want both true", fieldType.IDAttribute, fieldType.PII)
	}

	examples := loaded.Workflows[0].Scenarios[0].Steps[0].Examples
	var decoded []map[string]int
	if err := json.Unmarshal(examples, &decoded); err != nil {
		t.Fatalf("unmarshal examples: %v", err)
	}
	if len(decoded) != 1 || decoded[0]["owner_id"] != 9 {
		t.Fatalf("examples = %#v, want owner_id 9", decoded)
	}
}

func TestPetManagementDetailedModel_UsesCausalProjectionInputs(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "pet-management-detailed.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	loaded, diagnostics := loadTestModel(t, path, source)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}

	ownerRegistered := eventByID(t, contextByID(t, loaded, "owner_management"), "owner_registered")
	requireFieldNames(t, ownerRegistered.Fields, "owner_id", "owner_name", "owner_email")
	petUpdated := eventByID(t, contextByID(t, loaded, "pet_management"), "pet_details_updated")
	requireFieldNames(t, petUpdated.Fields, "pet_id", "owner_id", "pet_name", "birth_date", "pet_type")
	petTypeAdded := eventByID(t, contextByID(t, loaded, "pet_type_catalog"), "pet_type_added")
	requireFieldNames(t, petTypeAdded.Fields, "pet_type_id", "pet_type_name")

	requireEdge(t, loaded, "list_pet_types", "event.pet_type_catalog.pet_type_added", "readmodel.pet_type_list")
	requireEdge(t, loaded, "load_pet_details", "event.pet_management.pet_added", "readmodel.pet_details")
	requireEdge(t, loaded, "show_owner_details", "event.owner_management.owner_registered", "readmodel.owner_details")
	requireEdge(t, loaded, "show_owner_details", "event.pet_management.pet_added", "readmodel.owner_details")
	requireEdge(t, loaded, "show_owner_details", "event.pet_management.pet_details_updated", "readmodel.owner_details")

	if workflowIndex(t, loaded, "list_pet_types") >= workflowIndex(t, loaded, "add_pet") {
		t.Fatal("pet types must be available before adding a pet")
	}
	if workflowIndex(t, loaded, "load_pet_details") >= workflowIndex(t, loaded, "edit_pet") {
		t.Fatal("pet details must be loaded before editing a pet")
	}
	if workflowIndex(t, loaded, "show_owner_details") <= workflowIndex(t, loaded, "edit_pet") {
		t.Fatal("owner details projection must follow the events it consumes")
	}
}

func TestAppointmentWeatherPatternsModel_SpecifiesEveryWorkflowPattern(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "appointment-weather-patterns.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	loaded, diagnostics := loadTestModel(t, path, source)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}

	assertScenarioSteps(t, workflowByID(t, loaded, "schedule_appointment"), []StepKind{When, Then})
	assertScenarioSteps(t, workflowByID(t, loaded, "view_calendar"), []StepKind{Given, Then})
	assertScenarioSteps(t, workflowByID(t, loaded, "add_weather_forecast"), []StepKind{Given, Given, When, Then})
	assertScenarioSteps(t, workflowByID(t, loaded, "translate_weather_change"), []StepKind{Given, Given, When, Then})
}

func TestBuild_ResolvesFieldShorthands(t *testing.T) {
	source := []byte(`bounded_context "clinic" {
  field_type "pet_id" { type = "UUID" }
  field_type "pet_name" { type = "String" }

  event "pet_added" {
    fields = [field_type.pet_name]

    field "pet_id" {}
  }
}

state_change "add_pet" {
  screen "add_pet_form" {
    fields = [field_type.clinic.pet_name]

    field "pet_id" {}
  }
  command "add_pet" {
    external_trigger = true
    to               = [event.clinic.pet_added]
  }
}`)

	loaded, diagnostics := loadTestModel(t, "model.em.hcl", source)
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}

	event := loaded.Contexts[0].Events[0]
	if got := fieldNames(event.Fields); !sameStrings(got, []string{"pet_name", "pet_id"}) {
		t.Fatalf("event field order = %#v, want [pet_name pet_id]", got)
	}
	if got, want := event.Fields[0].Type, "field_type.clinic.pet_name"; got != want {
		t.Fatalf("list field type = %q, want %q", got, want)
	}
	if got, want := event.Fields[1].Type, "field_type.clinic.pet_id"; got != want {
		t.Fatalf("inferred event field type = %q, want %q", got, want)
	}

	screen := elementByID(t, workflowByID(t, loaded, "add_pet"), "add_pet_form")
	if got := fieldNames(screen.Fields); !sameStrings(got, []string{"pet_name", "pet_id"}) {
		t.Fatalf("screen field order = %#v, want [pet_name pet_id]", got)
	}
	if got, want := screen.Fields[1].Type, "field_type.clinic.pet_id"; got != want {
		t.Fatalf("inferred screen field type = %q, want %q", got, want)
	}
}

func fieldNames(fields []Field) []string {
	names := make([]string, len(fields))
	for index, field := range fields {
		names[index] = field.Name
	}
	return names
}

func workflowByID(t *testing.T, loaded *Model, id string) Workflow {
	t.Helper()
	for _, workflow := range loaded.Workflows {
		if workflow.ID == id {
			return workflow
		}
	}
	t.Fatalf("workflow %q not found", id)
	return Workflow{}
}

func assertScenarioSteps(t *testing.T, workflow Workflow, want []StepKind) {
	t.Helper()
	if len(workflow.Scenarios) != 1 {
		t.Fatalf("workflow %q scenarios = %d, want 1", workflow.ID, len(workflow.Scenarios))
	}
	got := make([]StepKind, len(workflow.Scenarios[0].Steps))
	for i, step := range workflow.Scenarios[0].Steps {
		got[i] = step.Kind
	}
	if !sameSteps(got, want) {
		t.Fatalf("workflow %q scenario steps = %#v, want %#v", workflow.ID, got, want)
	}
}

func elementByID(t *testing.T, workflow Workflow, id string) Element {
	t.Helper()
	for _, element := range workflow.Elements {
		if element.ID == id {
			return element
		}
	}
	t.Fatalf("element %q not found", id)
	return Element{}
}

func contextByID(t *testing.T, loaded *Model, id string) Context {
	t.Helper()
	for _, context := range loaded.Contexts {
		if context.ID == id {
			return context
		}
	}
	t.Fatalf("context %q not found", id)
	return Context{}
}

func eventByID(t *testing.T, context Context, id string) Event {
	t.Helper()
	for _, event := range context.Events {
		if event.ID == id {
			return event
		}
	}
	t.Fatalf("event %q not found in context %q", id, context.ID)
	return Event{}
}

func requireFieldNames(t *testing.T, fields []Field, names ...string) {
	t.Helper()
	for _, name := range names {
		found := false
		for _, field := range fields {
			if field.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("field %q not found", name)
		}
	}
}

func workflowIndex(t *testing.T, loaded *Model, id string) int {
	t.Helper()
	for index, workflow := range loaded.Workflows {
		if workflow.ID == id {
			return index
		}
	}
	t.Fatalf("workflow %q not found", id)
	return -1
}

func requireEdge(t *testing.T, loaded *Model, workflow, from, to string) {
	t.Helper()
	for _, edge := range loaded.Edges {
		if edge.WorkflowID == workflow && edge.From == from && edge.To == to {
			return
		}
	}
	t.Fatalf("edge %s: %s -> %s not found", workflow, from, to)
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameSteps(left, right []StepKind) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
