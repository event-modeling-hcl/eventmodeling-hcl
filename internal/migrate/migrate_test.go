package migrate

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const oneFile = `# Clinic model.

bounded_context "clinic" {
  title = "The Clinic"

  aggregate "pet" {}

  field_type "pet_id" {
    type = "UUID"
  }

  field_type "name" {
    type = "String"
  }

  event "pet_registered" {
    field "pet_id" {}
    field "name" {
      type = "String"
    }
  }
}

bounded_context "partner" {
  external = true

  field_type "partner_ref" {
    type = "String"
  }

  event "partner_pet_shared" {
    field "partner_ref" {}
  }
}

state_view "pet_directory" {
  readmodel "pets" {
    question = "Which pets exist?"
    from     = [event.clinic.pet_registered]

    field "pet_id" {}
  }
}

state_change "register_pet" {
  title = "Register a pet"

  command "register" {
    to = [event.clinic.pet_registered]

    field "partner_ref" {}
  }

  scenario "registers" {
    when {
      command = command.register
    }
    then {
      event = event.clinic.pet_registered
    }
  }
}

translation "import_partner_pet" {
  processor "importer" {
    from = [event.partner.partner_pet_shared]
    to   = [command.register_imported]
  }

  command "register_imported" {
    to = [event.clinic.pet_registered]
  }
}
`

func migrateOne(t *testing.T, source string) Result {
	t.Helper()
	result, diagnostics := Migrate([]syntax.File{{Name: "model.em.hcl", Source: []byte(source)}})
	if diagnostics.HasErrors() {
		t.Fatalf("Migrate: %v", diagnostics)
	}
	return result
}

func migrateError(t *testing.T, files []syntax.File, summary string) {
	t.Helper()
	result, diagnostics := Migrate(files)
	if !diagnostics.HasErrors() {
		t.Fatalf("Migrate succeeded, want %q error", summary)
	}
	if result.Files != nil || result.Report != nil {
		t.Fatalf("Migrate returned output with errors: %+v", result)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Summary == summary {
			return
		}
	}
	t.Fatalf("diagnostics %v lack %q", diagnostics, summary)
}

func requireContains(t *testing.T, text string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("output lacks %q:\n%s", fragment, text)
		}
	}
}

func TestOneFileMigration(t *testing.T) {
	result := migrateOne(t, oneFile)
	if len(result.Files) != 1 || result.Files[0].Name != "model.em.hcl" {
		t.Fatalf("files = %+v", result.Files)
	}
	output := string(result.Files[0].Source)
	if !strings.HasPrefix(output, "# Clinic model.\n\nboard {\n  workflows = [\n    workflow.pet_directory,\n    workflow.register_pet,\n    workflow.import_partner_pet,\n  ]\n}\n\nbounded_context \"clinic\" {") {
		t.Fatalf("board not at the start in source order:\n%s", output)
	}
	requireContains(t, output,
		// Context-owned typeless fields bind to their own context.
		"field \"pet_id\" {\n      type = field_type.clinic.pet_id\n",
		"field \"partner_ref\" {\n      type = field_type.partner.partner_ref\n",
		// Workflow typeless fields bind to the unique field_type.
		"field \"pet_id\" {\n      type = field_type.clinic.pet_id\n    }\n  }\n}\n\nstate_change",
		"field \"partner_ref\" {\n      type = field_type.partner.partner_ref\n    }\n  }\n\n  scenario",
		// Explicit types and titles stay.
		"type = \"String\"",
		"title = \"The Clinic\"",
		"title = \"Register a pet\"",
		// Derived titles become explicit.
		"aggregate \"pet\" {\n    title = \"Pet\"\n  }",
		"event \"pet_registered\" {\n    title = \"Pet Registered\"",
		"bounded_context \"partner\" {\n  title    = \"Partner\"",
		"state_view \"pet_directory\" {\n  title = \"Pet Directory\"",
		"readmodel \"pets\" {\n    title    = \"Pets\"",
		"scenario \"registers\" {\n    title = \"Registers\"",
		// translation becomes automation with an unchanged body.
		"automation \"import_partner_pet\" {\n  title = \"Import Partner Pet\"\n\n  processor \"importer\" {\n    title = \"Importer\"\n    from  = [event.partner.partner_pet_shared]",
	)
	if strings.Contains(output, "translation") {
		t.Fatalf("translation keyword remains:\n%s", output)
	}
	if strings.Contains(output, "title = \"Clinic\"") || strings.Contains(output, "Register Pet\"") {
		t.Fatalf("explicit title overwritten:\n%s", output)
	}
	for _, step := range []string{"when {\n", "then {\n"} {
		if strings.Contains(output, step+"      title") {
			t.Fatalf("scenario step got a title:\n%s", output)
		}
	}
	want := []string{
		"board: inserted at the start of model.em.hcl with 3 workflows",
		"types: wrote 4 explicit field types",
		"titles: wrote 11 explicit titles",
		`translation: model.em.hcl:64: workflow "import_partner_pet" rewritten as automation`,
	}
	if !slices.Equal(result.Report, want) {
		t.Fatalf("report = %q, want %q", result.Report, want)
	}
}

func TestOutputIsFormattedAndStable(t *testing.T) {
	first := migrateOne(t, oneFile)
	second := migrateOne(t, oneFile)
	if !bytes.Equal(first.Files[0].Source, second.Files[0].Source) || !slices.Equal(first.Report, second.Report) {
		t.Fatal("two migrations of the same input differ")
	}
	if formatted := hclwrite.Format(first.Files[0].Source); !bytes.Equal(formatted, first.Files[0].Source) {
		t.Fatalf("output is not formatted:\n%s\n---\n%s", first.Files[0].Source, formatted)
	}
}

func TestCommentsAndAttributeOrderSurvive(t *testing.T) {
	source := `# Header comment.

# Actors.
actor "vet" { # after brace
  # Own-line comment above an attribute.
  description   = "Treats pets." # trailing comment
  auth_required = true
  // Slash comment.
}

bounded_context "clinic" {
  field_type "pet_id" {
    type = "UUID"
  }

  event "pet_registered" {
    /* block comment */
    field "pet_id" {
      example = "07917485-4a6b-4e4e-8920-92ce6062a129" # example comment
    }
  }
}
`
	output := string(migrateOne(t, source).Files[0].Source)
	want := `# Header comment.

board {
  workflows = []
}

# Actors.
actor "vet" { # after brace
  title = "Vet"
  # Own-line comment above an attribute.
  description   = "Treats pets." # trailing comment
  auth_required = true
  // Slash comment.
}

bounded_context "clinic" {
  title = "Clinic"

  field_type "pet_id" {
    type = "UUID"
  }

  event "pet_registered" {
    title = "Pet Registered"

    /* block comment */
    field "pet_id" {
      type    = field_type.clinic.pet_id
      example = "07917485-4a6b-4e4e-8920-92ce6062a129" # example comment
    }
  }
}
`
	if output != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output, want)
	}
}

func TestUneditedMemberIsReturnedVerbatim(t *testing.T) {
	files := []syntax.File{
		{Name: "a.em.hcl", Source: []byte("actor \"a\" {\n  title = \"A\"\n  auth_required = true\n}\n")},
		{Name: "b.em.hcl", Source: []byte("hotspot   \"h\" {\n  question = \"?\"   # odd spacing\n}\n")},
	}
	result, diagnostics := Migrate(files)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	if !bytes.Equal(result.Files[1].Source, files[1].Source) {
		t.Fatalf("unedited member changed:\n%s", result.Files[1].Source)
	}
}

func TestSingleLineBlocksAndEmptyBoard(t *testing.T) {
	result := migrateOne(t, "actor \"front_desk\" { auth_required = true }\nteam \"vets\" {}\n")
	output := string(result.Files[0].Source)
	want := "board {\n  workflows = []\n}\n\nactor \"front_desk\" {\n  title         = \"Front Desk\"\n  auth_required = true\n}\nteam \"vets\" {\n  title = \"Vets\"\n}\n"
	if output != want {
		t.Fatalf("output:\n%s\nwant:\n%s", output, want)
	}
}

func TestAmbiguousTypelessFieldStops(t *testing.T) {
	source := `bounded_context "a" {
  field_type "id" {
    type = "UUID"
  }
}

bounded_context "b" {
  field_type "id" {
    type = "String"
  }
}

state_view "v" {
  readmodel "r" {
    question = "?"
    field "id" {}
  }
}
`
	migrateError(t, []syntax.File{{Name: "m.em.hcl", Source: []byte(source)}}, "Ambiguous field type")
}

func TestUnresolvedTypelessFieldStops(t *testing.T) {
	source := `bounded_context "a" {
  event "e" {
    field "missing" {}
  }
}
`
	migrateError(t, []syntax.File{{Name: "m.em.hcl", Source: []byte(source)}}, "Unresolved field type")
}

func TestAutomationConsumingExternalStops(t *testing.T) {
	source := strings.Replace(oneFile, `translation "import_partner_pet"`, `automation "import_partner_pet"`, 1)
	migrateError(t, []syntax.File{{Name: "m.em.hcl", Source: []byte(source)}}, "Invalid automation")
}

func TestTranslationWithoutExternalStops(t *testing.T) {
	source := strings.Replace(oneFile, "external = true", "external = false", 1)
	migrateError(t, []syntax.File{{Name: "m.em.hcl", Source: []byte(source)}}, "Invalid translation")
}

func TestExistingBoardStops(t *testing.T) {
	files := []syntax.File{
		{Name: "a.em.hcl", Source: []byte("actor \"a\" {\n  auth_required = true\n}\n")},
		{Name: "b.em.hcl", Source: []byte("board {\n  workflows = []\n}\n")},
	}
	migrateError(t, files, "Existing board block")
}

func TestParseFailureStops(t *testing.T) {
	migrateError(t, []syntax.File{{Name: "m.em.hcl", Source: []byte("actor \"a\" {\n")}}, "Unclosed configuration block")
}

func TestFolderMigration(t *testing.T) {
	files := []syntax.File{
		{Name: "dir/10-flows.em.hcl", Source: []byte(`state_view "zeta" {
  readmodel "z" {
    question = "?"
  }
}

state_view "alpha" {
  readmodel "a" {
    question = "?"
  }
}

state_view "middle" {
  readmodel "m" {
    question = "?"
  }
}

state_view "loose" {
  readmodel "l" {
    question = "?"
  }
}
`)},
		{Name: "dir/01-chapters.em.hcl", Source: []byte(`# Story groups.
chapter "second" {
  workflows = [workflow.middle]
}

chapter "first" {
  title     = "Opening"
  workflows = [workflow.alpha, workflow.zeta]
}
`)},
		{Name: "dir/00-catalog.em.hcl", Source: []byte(`// Catalog.

bounded_context "clinic" {
  title = "Clinic"
}
`)},
	}
	first, diagnostics := Migrate(files)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics)
	}
	second, _ := Migrate(files)
	for index, file := range first.Files {
		if file.Name != files[index].Name {
			t.Fatalf("file %d = %s, want %s", index, file.Name, files[index].Name)
		}
		if !bytes.Equal(file.Source, second.Files[index].Source) {
			t.Fatalf("%s differs across runs", file.Name)
		}
		if index != 2 && strings.Contains(string(file.Source), "board") {
			t.Fatalf("board written outside the byte-first file %s:\n%s", file.Name, file.Source)
		}
	}
	catalog := string(first.Files[2].Source)
	want := "// Catalog.\n\nboard {\n  workflows = [\n    workflow.middle,\n    workflow.alpha,\n    workflow.zeta,\n    workflow.loose,\n  ]\n}\n\nbounded_context \"clinic\" {\n  title = \"Clinic\"\n}\n"
	if catalog != want {
		t.Fatalf("catalog:\n%s\nwant:\n%s", catalog, want)
	}
	requireContains(t, string(first.Files[1].Source),
		"# Story groups.\nchapter \"second\" {\n  title     = \"Second\"\n  workflows = [workflow.middle]\n}",
		"title     = \"Opening\"\n  workflows = [workflow.alpha, workflow.zeta]",
	)
	if first.Report[0] != "board: inserted at the start of dir/00-catalog.em.hcl with 4 workflows" {
		t.Fatalf("report = %q", first.Report)
	}
}

func TestFolderWithOneMemberUsesSourceOrder(t *testing.T) {
	source := `chapter "c" {
  workflows = [workflow.b]
}

state_view "a" {
  readmodel "r" {
    question = "?"
  }
}

state_view "b" {
  readmodel "r" {
    question = "?"
  }
}
`
	result := migrateOne(t, source)
	requireContains(t, string(result.Files[0].Source), "workflows = [\n    workflow.a,\n    workflow.b,\n  ]")
}
