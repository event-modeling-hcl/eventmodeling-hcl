package model

import (
	"os"
	"path/filepath"
	"testing"

	sourcepkg "github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
)

// buildFixture parses source with syntax.Parse and lowers it with Build,
// failing the test on any parse diagnostics. It exists so Build tests do not
// need to route through Load (and therefore validation) to exercise pure
// lowering behavior.
func buildFixture(t *testing.T, filename string, source []byte) *Model {
	t.Helper()
	doc, diagnostics := syntax.Parse(filename, source)
	if diagnostics.HasErrors() {
		t.Fatalf("syntax.Parse diagnostics = %s", diagnostics.Error())
	}
	validated, validationDiagnostics := validator.ValidateDecodedDocument(sourcepkg.Decode(doc), validator.Valid)
	if validationDiagnostics.HasErrors() {
		t.Fatalf("validation diagnostics = %s", validationDiagnostics.Error())
	}
	built := Build(validated)
	if built == nil {
		t.Fatal("Build returned nil model")
	}
	return built
}

func TestBuild_LowersCompleteModelIntoCanonicalIR(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "complete.em.hcl")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	built := buildFixture(t, path, source)

	if got, want := len(built.Workflows), 4; got != want {
		t.Fatalf("workflow count = %d, want %d", got, want)
	}
	if got := workflowByID(t, built, "register_pet").Kind; got != StateChange {
		t.Fatalf("register_pet kind = %q, want %q", got, StateChange)
	}
	if got := workflowByID(t, built, "pet_directory").Kind; got != StateView {
		t.Fatalf("pet_directory kind = %q, want %q", got, StateView)
	}
	if got := workflowByID(t, built, "notify_owner").Kind; got != Automation {
		t.Fatalf("notify_owner kind = %q, want %q", got, Automation)
	}
	if got := workflowByID(t, built, "import_partner_pet").Kind; got != Translation {
		t.Fatalf("import_partner_pet kind = %q, want %q", got, Translation)
	}

	requireEdge(t, built, "register_pet", "screen.pet_screen", "command.register_pet_command")
	requireEdge(t, built, "pet_directory", "event.clinic.pet_registered", "readmodel.pet_summary")
	requireEdge(t, built, "pet_directory", "readmodel.pet_summary", "screen.pet_summary_screen")

	command := elementByID(t, workflowByID(t, built, "register_pet"), "register_pet_command")
	if got, want := command.Semantic.Aggregate, "aggregate.clinic.pet"; got != want {
		t.Fatalf("command aggregate = %q, want %q", got, want)
	}
	if !command.Semantic.CreatesAggregate {
		t.Fatal("command creates aggregate = false, want true")
	}
}

func TestBuild_LowersTableAndScreenImageBlocks(t *testing.T) {
	source := []byte(`state_change "register_pet" {
  screen "pet_screen" {}
  command "register_pet" {}

  screen_image "pet_screen" {
    title = "Pet screen"
    url   = "https://example.com/pet.png"
  }

  table "pet_table" {
    title = "Pets"
  }
}`)

	built := buildFixture(t, "model.em.hcl", source)

	workflow := workflowByID(t, built, "register_pet")
	table := elementByID(t, workflow, "pet_table")
	if table.Kind != Table {
		t.Fatalf("table kind = %q, want %q", table.Kind, Table)
	}

	var screenImage *Element
	for index := range workflow.Elements {
		if workflow.Elements[index].Kind == ScreenImage {
			screenImage = &workflow.Elements[index]
			break
		}
	}
	if screenImage == nil {
		t.Fatal("screen_image element not found")
	}
	if screenImage.Title != "Pet screen" {
		t.Fatalf("screen_image title = %q, want %q", screenImage.Title, "Pet screen")
	}
}
