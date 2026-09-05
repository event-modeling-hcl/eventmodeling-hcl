package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DecodesCompleteModelIntoCanonicalIR(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	loaded, diagnostics := Load(path, source, Valid)

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

func TestLoad_DerivesTitlesWithoutChangingExplicitTitles(t *testing.T) {
	source := []byte(`state_change "register_pet" {
  command "register_pet" {}
}
state_change "explicit_title" {
  title = "A custom title"
}`)

	loaded, diagnostics := Load("model.em.hcl", source, Valid)

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
