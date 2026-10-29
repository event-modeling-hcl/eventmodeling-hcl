package renderer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/version"
)

func TestBuildViewModel_AdaptsCurrentCompleteModel(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	loaded, diagnostics := loadTestModel(t, path, source)
	if diagnostics.HasErrors() {
		t.Fatalf("load example: %s", diagnostics.Error())
	}

	view := BuildViewModel(path, loaded)

	if got, want := view.Title, "complete"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	if got, want := view.Version, version.Spec; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
	if got, want := len(view.Slices), 4; got != want {
		t.Fatalf("slices = %d, want %d", got, want)
	}

	registration := findSlice(t, view, "register_pet")
	if got, want := registration.Title, "Register Pet"; got != want {
		t.Fatalf("slice title = %q, want %q", got, want)
	}
	image := findElement(t, registration, "register_pet__screen_image__pet_form")
	if got, want := image.ImageURL, "https://example.test/pet-form.png"; got != want {
		t.Fatalf("image URL = %q, want %q", got, want)
	}

	command := findElement(t, registration, "register_pet__command__register_pet_command")
	petID := findField(t, command, "pet_id")
	if got, want := petID.Type, "UUID"; got != want {
		t.Fatalf("field type = %q, want %q", got, want)
	}
	if !petID.ID {
		t.Fatal("field ID badge = false, want true")
	}

	if !hasEdge(view, "register_pet__screen__pet_screen", "register_pet__command__register_pet_command") {
		t.Fatal("missing screen -> command edge")
	}
	if !hasEdge(view, "notify_owner__processor__pet_notification", "notify_owner__command__send_owner_notification") {
		t.Fatal("missing automation processor -> command edge")
	}

	scenario := registration.Scenarios[0]
	if got := string(scenario.Given[0].Examples); got != `[{"owner_id":9}]` {
		t.Fatalf("scenario examples = %q", got)
	}

	directory := findSlice(t, view, "pet_directory")
	if len(directory.Scenarios) != 0 {
		t.Fatalf("state view scenarios = %d, want 0 in complete fixture", len(directory.Scenarios))
	}

	hotspot := view.Hotspots[0]
	if got, want := hotspot.OnID, "notify_owner__processor__pet_notification"; got != want {
		t.Fatalf("hotspot target = %q, want %q", got, want)
	}
}

func TestBuildViewModel_PlacesScenarioOnlyEventsByStepKind(t *testing.T) {
	loaded := &model.Model{
		Contexts: []model.Context{{ID: "clinic", Events: []model.Event{
			{ID: "upstream", Title: "Upstream"},
			{ID: "outcome", Title: "Outcome"},
		}}},
		Workflows: []model.Workflow{
			{Kind: model.StateView, ID: "view", Title: "View", Scenarios: []model.Scenario{{Steps: []model.Step{
				{Kind: model.Given, Target: "event", Ref: "event.clinic.upstream"},
				{Kind: model.Then, Target: "readmodel", Ref: "readmodel.view"},
			}}}},
			{Kind: model.StateChange, ID: "change", Title: "Change", Scenarios: []model.Scenario{{Steps: []model.Step{
				{Kind: model.Then, Target: "event", Ref: "event.clinic.outcome"},
			}}}},
		},
	}

	view := BuildViewModel("scenario-only.em.hcl", loaded)

	upstream := findElement(t, findSlice(t, view, "view"), "event__clinic__upstream")
	if !upstream.Given {
		t.Fatal("given event is not marked upstream")
	}
	outcome := findElement(t, findSlice(t, view, "change"), "event__clinic__outcome")
	if outcome.Given {
		t.Fatal("then event is marked upstream")
	}
}

func TestBuildViewModel_UsesBrowserSafeEmptyCollections(t *testing.T) {
	view := BuildViewModel("empty.em.hcl", &model.Model{
		Workflows: []model.Workflow{{Kind: model.StateChange, ID: "empty", Title: "Empty"}},
	})

	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal view: %v", err)
	}
	for _, expected := range []string{`"chapters":[]`, `"hotspots":[]`, `"edges":[]`, `"elements":[]`, `"scenarios":[]`} {
		if !strings.Contains(string(encoded), expected) {
			t.Errorf("JSON missing %s: %s", expected, encoded)
		}
	}
}

func TestBuildViewModel_AssignsIncreasingStagesToLocalFlow(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	loaded, diagnostics := loadTestModel(t, path, source)
	if diagnostics.HasErrors() {
		t.Fatalf("load example: %s", diagnostics.Error())
	}

	view := BuildViewModel(path, loaded)
	for _, slice := range view.Slices {
		stages := map[string]int{}
		for _, element := range slice.Elements {
			stages[element.ID] = element.Stage
		}
		for _, edge := range view.Edges {
			fromStage, fromLocal := stages[edge.From]
			toStage, toLocal := stages[edge.To]
			if fromLocal && toLocal && fromStage >= toStage {
				t.Errorf("slice %q edge %q -> %q has stages %d -> %d", slice.ID, edge.From, edge.To, fromStage, toStage)
			}
		}
		if slice.StageCount < 1 {
			t.Errorf("slice %q stage count = %d, want at least 1", slice.ID, slice.StageCount)
		}
	}

	translation := findSlice(t, view, "import_partner_pet")
	if got, want := translation.Type, "translation"; got != want {
		t.Fatalf("import_partner_pet type = %q, want derived %q", got, want)
	}
	if got, want := findSlice(t, view, "notify_owner").Type, "automation"; got != want {
		t.Fatalf("notify_owner type = %q, want %q", got, want)
	}
	externalEvent := findElement(t, translation, "event__partner__pet_received")
	processor := findElement(t, translation, "import_partner_pet__processor__translate_pet")
	if !externalEvent.External {
		t.Fatal("partner event is not marked external")
	}
	if externalEvent.Stage >= processor.Stage {
		t.Fatalf("external event stage = %d, processor stage = %d", externalEvent.Stage, processor.Stage)
	}
}

func TestBuildViewModel_AssignsActorsToScreens(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	loaded, diagnostics := loadTestModel(t, path, source)
	if diagnostics.HasErrors() {
		t.Fatalf("load example: %s", diagnostics.Error())
	}

	view := BuildViewModel(path, loaded)
	registration := findSlice(t, view, "register_pet")
	screen := findElement(t, registration, "register_pet__screen__pet_screen")
	if got, want := screen.Actor, "clinic_staff"; got != want {
		t.Fatalf("screen actor = %q, want %q", got, want)
	}
	if got, want := view.Actors[screen.Actor].Title, "Clinic staff"; got != want {
		t.Fatalf("actor title = %q, want %q", got, want)
	}
	if !view.Actors[screen.Actor].AuthRequired {
		t.Fatal("actor authRequired = false, want true")
	}
}

func findSlice(t *testing.T, view *ViewModel, id string) *Slice {
	t.Helper()
	for index := range view.Slices {
		if view.Slices[index].ID == id {
			return &view.Slices[index]
		}
	}
	t.Fatalf("slice %q not found", id)
	return nil
}

func findElement(t *testing.T, slice *Slice, id string) *Element {
	t.Helper()
	for index := range slice.Elements {
		if slice.Elements[index].ID == id {
			return &slice.Elements[index]
		}
	}
	t.Fatalf("element %q not found", id)
	return nil
}

func findField(t *testing.T, element *Element, name string) Field {
	t.Helper()
	for _, field := range element.Fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found", name)
	return Field{}
}

func hasEdge(view *ViewModel, from, to string) bool {
	for _, edge := range view.Edges {
		if edge.From == from && edge.To == to {
			return true
		}
	}
	return false
}

// offBoardSource is a v0.5.0 model whose board lists two workflows in an order
// that is not ID order, leaves a third off the board, and gives a chapter one
// on-board and one off-board member.
const offBoardSource = `
board {
  workflows = [workflow.zeta_view, workflow.alpha_change]
}

bounded_context "clinic" {
  title = "Clinic"

  event "pet_added" {
    title = "Pet added"

    field "name" {
      type = "String"
    }
  }

  event "pet_archived" {
    field "name" {
      type = "String"
    }
  }
}

state_change "alpha_change" {
  title = "Add pet"

  command "add_pet" {
    title        = "Add pet"
    api_endpoint = "POST /pets"
    to           = [event.clinic.pet_added]

    field "name" {
      type = "String"
    }
  }
}

state_view "zeta_view" {
  title = "Pet list"

  readmodel "pets" {
    title    = "Pets"
    question = "Which pets are registered?"
    from     = [event.clinic.pet_added]

    field "name" {
      type = "String"
    }
  }
}

state_change "archive_pet" {
  command "archive" {
    api_endpoint = "POST /pets/archive"
    to           = [event.clinic.pet_archived]

    field "name" {
      type = "String"
    }
  }
}

chapter "care" {
  title     = "Care"
  workflows = [workflow.alpha_change, workflow.archive_pet, workflow.zeta_view]
}
`

func TestBuildViewModel_KeepsOffBoardWorkflowsOutOfTheStory(t *testing.T) {
	loaded, diagnostics := loadTestModel(t, "off-board.em.hcl", []byte(offBoardSource))
	if diagnostics.HasErrors() {
		t.Fatalf("load model: %s", diagnostics.Error())
	}

	view := BuildViewModel("off-board.em.hcl", loaded)

	if got, want := sliceIDs(view.Slices), []string{"zeta_view", "alpha_change"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("story slices = %v, want board order %v", got, want)
	}
	if got, want := sliceIDs(view.OffBoard), []string{"archive_pet"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("off-board slices = %v, want %v", got, want)
	}
	if got, want := view.Chapters, []Chapter{{ID: "care", Title: "Care", Slices: []string{"zeta_view", "alpha_change"}}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("chapters = %#v, want board members in board order %#v", got, want)
	}
	// An event that only an off-board workflow touches stays with it.
	archived := findElement(t, &view.OffBoard[0], "event__clinic__pet_archived")
	if archived.Given {
		t.Fatal("off-board produced event is marked upstream")
	}
	for _, slice := range view.Slices {
		for _, element := range slice.Elements {
			if strings.HasPrefix(element.ID, "archive_pet__") || element.ID == "event__clinic__pet_archived" {
				t.Fatalf("story slice %q carries off-board element %q", slice.ID, element.ID)
			}
		}
	}
}

func TestBuildViewModel_LabelsUntitledConstructsWithRawIDs(t *testing.T) {
	loaded, diagnostics := loadTestModel(t, "off-board.em.hcl", []byte(offBoardSource))
	if diagnostics.HasErrors() {
		t.Fatalf("load model: %s", diagnostics.Error())
	}

	view := BuildViewModel("off-board.em.hcl", loaded)

	archive := view.OffBoard[0]
	if got, want := archive.Title, "archive_pet"; got != want {
		t.Fatalf("untitled workflow label = %q, want raw ID %q", got, want)
	}
	if got, want := findElement(t, &archive, "archive_pet__command__archive").Title, "archive"; got != want {
		t.Fatalf("untitled command label = %q, want raw ID %q", got, want)
	}
	if got, want := findElement(t, &archive, "event__clinic__pet_archived").Title, "pet_archived"; got != want {
		t.Fatalf("untitled event label = %q, want raw ID %q", got, want)
	}
}

func TestBuildViewModel_LabelsEveryUntitledKindWithItsRawID(t *testing.T) {
	view := BuildViewModel("untitled.em.hcl", &model.Model{
		Actors:   []model.Actor{{ID: "front_desk"}},
		Contexts: []model.Context{{ID: "clinic_ops", Aggregates: []model.Aggregate{{ID: "pet_record"}}}},
		Board:    []string{"check_in"},
		Workflows: []model.Workflow{{Kind: model.StateChange, ID: "check_in",
			Elements: []model.Element{{Kind: model.Command, ID: "check_in_pet",
				Semantic: model.Semantic{Aggregate: "aggregate.clinic_ops.pet_record"}}},
			Scenarios: []model.Scenario{{ID: "happy_path", Steps: []model.Step{
				{Kind: model.When, Target: "command", Ref: "command.check_in_pet"},
			}}},
		}},
		Chapters: []model.Chapter{{ID: "arrival", Workflows: []string{"workflow.check_in"}}},
	})

	slice := view.Slices[0]
	command := findElement(t, &slice, "check_in__command__check_in_pet")
	for name, pair := range map[string][2]string{
		"actor":     {view.Actors["front_desk"].Title, "front_desk"},
		"context":   {view.Contexts["clinic_ops"].Title, "clinic_ops"},
		"map node":  {findContextNode(t, view.ContextMap, "clinic_ops").Title, "clinic_ops"},
		"workflow":  {slice.Title, "check_in"},
		"element":   {command.Title, "check_in_pet"},
		"aggregate": {command.AggTitle, "pet_record"},
		"scenario":  {slice.Scenarios[0].Title, "happy_path"},
		"step ref":  {slice.Scenarios[0].When.Ref, "check_in_pet"},
		"chapter":   {view.Chapters[0].Title, "arrival"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s label = %q, want raw ID %q", name, pair[0], pair[1])
		}
	}
}

func sliceIDs(slices []Slice) []string {
	ids := make([]string, 0, len(slices))
	for _, slice := range slices {
		ids = append(ids, slice.ID)
	}
	return ids
}
