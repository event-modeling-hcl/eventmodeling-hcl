package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	sourcepkg "github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
)

// buildFiles follows the production parse, decode, validate, and build
// sequence for a model made of several files. Only validation errors block the
// build; warnings are allowed.
func buildFiles(t *testing.T, files []syntax.File) *Model {
	t.Helper()
	doc, diagnostics := syntax.ParseFiles(files)
	if diagnostics.HasErrors() {
		t.Fatalf("syntax.ParseFiles diagnostics = %s", diagnostics.Error())
	}
	validated, validationDiagnostics := validator.ValidateDecodedDocument(sourcepkg.Decode(doc), validator.Valid)
	if validationDiagnostics.HasErrors() {
		t.Fatalf("validation diagnostics = %s", validationDiagnostics.Error())
	}
	return Build(validated)
}

func workflowIDs(built *Model) []string {
	ids := make([]string, 0, len(built.Workflows))
	for _, workflow := range built.Workflows {
		ids = append(ids, workflow.ID)
	}
	return ids
}

const orderCatalog = `bounded_context "shop" {
  event "happened" {}
}
`

func TestBuild_PartialBoardOrdersBoardThenOffBoardByID(t *testing.T) {
	built := buildFiles(t, []syntax.File{
		{Name: "model/a.em.hcl", Source: []byte(orderCatalog + `
state_change "w_z" {}
state_change "w_a" {
  command "a_cmd" {
    external_trigger = true
    to               = [event.shop.happened]
  }
}
board {
  workflows = [workflow.w_c, workflow.w_a]
}
`)},
		{Name: "model/b.em.hcl", Source: []byte(`state_change "w_d" {}
state_change "w_c" {
  command "c_cmd" {
    external_trigger = true
    to               = [event.shop.happened]
  }
}
state_change "w_b" {}
`)},
	})

	if got, want := workflowIDs(built), []string{"w_c", "w_a", "w_b", "w_d", "w_z"}; !sameStrings(got, want) {
		t.Fatalf("workflow order = %v, want %v", got, want)
	}
	if got, want := built.Board, []string{"w_c", "w_a"}; !sameStrings(got, want) {
		t.Fatalf("board = %v, want %v", got, want)
	}
	if got, want := built.OffBoard, []string{"w_b", "w_d", "w_z"}; !sameStrings(got, want) {
		t.Fatalf("off-board = %v, want %v", got, want)
	}
	if got, want := len(built.Edges), 2; got != want {
		t.Fatalf("edge count = %d, want %d", got, want)
	}
	if built.Edges[0].WorkflowID != "w_c" || built.Edges[1].WorkflowID != "w_a" {
		t.Fatalf("edges = %+v, want w_c then w_a", built.Edges)
	}
}

func TestBuild_EmptyBoardPutsEveryWorkflowOffBoard(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(`state_view "second" {}
state_change "first" {}
board {
  workflows = []
}
`)}})

	if len(built.Board) != 0 {
		t.Fatalf("board = %v, want empty", built.Board)
	}
	if got, want := built.OffBoard, []string{"first", "second"}; !sameStrings(got, want) {
		t.Fatalf("off-board = %v, want %v", got, want)
	}
	if got, want := workflowIDs(built), []string{"first", "second"}; !sameStrings(got, want) {
		t.Fatalf("workflow order = %v, want %v", got, want)
	}
}

// layoutModel is one model whose declarations are deliberately out of ID
// order, with chapter lists that disagree with the board.
var layoutModel = []string{
	`team "zeta_team" {}
system "alpha_system" {}
actor "zed" { auth_required = true }
actor "amy" { auth_required = false }
`,
	`bounded_context "zoo" {
  field_type "z_id" { type = "UUID" }
  field_type "a_id" { type = "UUID" }
  aggregate "zebra" {}
  aggregate "ant" {}
  event "z_happened" {
    field "z_id" { type = field_type.z_id }
    field "a_id" { type = field_type.a_id }
  }
  event "a_happened" {}
}
bounded_context "alpha" {}
`,
	`state_change "register" {
  screen "form" {
    to = [command.register]
  }
  command "register" {
    to = [event.zoo.z_happened, event.zoo.a_happened]
  }
  scenario "z_case" {
    when { command = command.register }
    then { event = event.zoo.z_happened }
    then { event = event.zoo.a_happened }
  }
  scenario "a_case" {
    when { command = command.register }
    then { event = event.zoo.a_happened }
  }
}
state_view "directory" {
  readmodel "list" {
    question = "Which?"
    from     = [event.zoo.z_happened]
  }
}
state_view "audit" {
  readmodel "log" {
    question = "What happened?"
    from     = [event.zoo.a_happened]
  }
}
`,
	`chapter "zz_chapter" {
  workflows = [workflow.register, workflow.directory]
}
chapter "aa_chapter" {
  workflows = [workflow.audit]
}
hotspot "z_question" {
  question = "Z?"
  status   = "resolved"
}
hotspot "a_question" {
  question = "A?"
  status   = "resolved"
}
board {
  workflows = [workflow.register]
}
`,
}

func TestBuild_ModelIsIndependentOfFileLayout(t *testing.T) {
	oneFile := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(strings.Join(layoutModel, "\n"))}})
	split := buildFiles(t, []syntax.File{
		{Name: "folder/d.em.hcl", Source: []byte(layoutModel[0])},
		{Name: "folder/c.em.hcl", Source: []byte(layoutModel[1])},
		{Name: "folder/b.em.hcl", Source: []byte(layoutModel[2])},
		{Name: "folder/a.em.hcl", Source: []byte(layoutModel[3])},
	})
	renamed := buildFiles(t, []syntax.File{
		{Name: "other/1-board.em.hcl", Source: []byte(layoutModel[3])},
		{Name: "other/2-workflows.em.hcl", Source: []byte(layoutModel[2])},
		{Name: "other/3-people-and-catalog.em.hcl", Source: []byte(layoutModel[0] + layoutModel[1])},
	})

	if !reflect.DeepEqual(oneFile, split) {
		t.Fatalf("split model = %+v, want the one-file model %+v", split, oneFile)
	}
	if !reflect.DeepEqual(oneFile, renamed) {
		t.Fatalf("renamed model = %+v, want the one-file model %+v", renamed, oneFile)
	}
}

func TestBuild_SortsCollectionsByIDAndKeepsSourceSequences(t *testing.T) {
	built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: []byte(strings.Join(layoutModel, "\n"))}})

	if got, want := workflowIDs(built), []string{"register", "audit", "directory"}; !sameStrings(got, want) {
		t.Fatalf("workflow order = %v, want board then off-board %v", got, want)
	}
	requireOrder(t, "actors", []string{built.Actors[0].ID, built.Actors[1].ID}, "amy", "zed")
	requireOrder(t, "owners", []string{built.Owners[0].Kind + "." + built.Owners[0].ID, built.Owners[1].Kind + "." + built.Owners[1].ID}, "system.alpha_system", "team.zeta_team")
	requireOrder(t, "contexts", []string{built.Contexts[0].ID, built.Contexts[1].ID}, "alpha", "zoo")
	zoo := contextByID(t, built, "zoo")
	requireOrder(t, "aggregates", []string{zoo.Aggregates[0].ID, zoo.Aggregates[1].ID}, "ant", "zebra")
	requireOrder(t, "field types", []string{zoo.FieldTypes[0].ID, zoo.FieldTypes[1].ID}, "a_id", "z_id")
	requireOrder(t, "events", []string{zoo.Events[0].ID, zoo.Events[1].ID}, "a_happened", "z_happened")
	requireOrder(t, "event fields", fieldNames(eventByID(t, zoo, "z_happened").Fields), "z_id", "a_id")
	requireOrder(t, "chapters", []string{built.Chapters[0].ID, built.Chapters[1].ID}, "aa_chapter", "zz_chapter")
	requireOrder(t, "chapter members", built.Chapters[1].Workflows, "workflow.directory", "workflow.register")
	requireOrder(t, "hotspots", []string{built.Hotspots[0].ID, built.Hotspots[1].ID}, "a_question", "z_question")

	register := workflowByID(t, built, "register")
	requireOrder(t, "elements", []string{string(register.Elements[0].Kind) + "." + register.Elements[0].ID, string(register.Elements[1].Kind) + "." + register.Elements[1].ID}, "command.register", "screen.form")
	requireOrder(t, "scenarios", []string{register.Scenarios[0].ID, register.Scenarios[1].ID}, "a_case", "z_case")
	steps := register.Scenarios[1].Steps
	requireOrder(t, "scenario steps", []string{steps[1].Ref, steps[2].Ref}, "event.zoo.z_happened", "event.zoo.a_happened")

	var edges []string
	for _, edge := range built.Edges {
		edges = append(edges, edge.WorkflowID+":"+edge.From+">"+edge.To)
	}
	requireOrder(t, "edges", edges,
		"register:command.register>event.zoo.z_happened",
		"register:command.register>event.zoo.a_happened",
		"register:screen.form>command.register",
		"audit:event.zoo.a_happened>readmodel.log",
		"directory:event.zoo.z_happened>readmodel.list",
	)
}

func TestBuild_MarksEdgesFromExternalContextEvents(t *testing.T) {
	source := func(external bool) []byte {
		flag := "false"
		if external {
			flag = "true"
		}
		return withBoard(`bounded_context "partner" {
  external = ` + flag + `
  event "pet_received" {}
}
bounded_context "clinic" {
  event "pet_imported" {}
}
automation "import_pet" {
  processor "translate" {
    from = [event.partner.pet_received, event.clinic.pet_imported]
    to   = [command.import]
  }
  command "import" {
    to = [event.clinic.pet_imported]
  }
}`)
	}

	for _, external := range []bool{true, false} {
		built := buildFiles(t, []syntax.File{{Name: "model.em.hcl", Source: source(external)}})
		for _, edge := range built.Edges {
			want := external && edge.From == "event.partner.pet_received"
			if edge.External != want {
				t.Fatalf("external=%t: edge %+v External = %t, want %t", external, edge, edge.External, want)
			}
		}
	}
}

func TestBuild_MultiFileDecodesQuotedUnicodeKeysFromTheirOwnFile(t *testing.T) {
	// The key is written with a combining accent. HCL evaluation would
	// normalize it, so the exact spelling can only come from the source bytes
	// of the file that holds the expression.
	const key = "e\u0301clair"
	built := buildFiles(t, []syntax.File{
		{Name: "model/a.em.hcl", Source: []byte(`bounded_context "shop" {
  event "happened" {}
}

state_change "w_a" {}
state_change "w_b" {}
board {
  workflows = [workflow.w_a, workflow.w_b, workflow.w_c]
}
`)},
		{Name: "model/b.em.hcl", Source: []byte(`state_change "w_c" {
  command "c_cmd" {
    external_trigger = true
    prototype        = { "` + key + `" = 1 }
    to               = [event.shop.happened]
  }
}
`)},
	})

	data := workflowByID(t, built, "w_c").Elements[0].Presentation.PrototypeData
	var decoded map[string]int
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal prototype data %q: %v", data, err)
	}
	if value, ok := decoded[key]; !ok || value != 1 {
		t.Fatalf("prototype data = %q, want exact key %q with value 1", data, key)
	}
}

func requireOrder(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	if !sameStrings(got, want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}
