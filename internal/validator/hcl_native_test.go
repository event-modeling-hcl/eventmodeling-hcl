package validator

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestValidateSource_AcceptsTypedHCLReferences(t *testing.T) {
	model := nativeCatalog() + `
actor "clinic_staff" {
  title         = "Clinic staff"
  auth_required = true
}

state_change "add_pet" {
  title  = "Add Pet"
  owner  = bounded_context.pet_management
  status = "planned"

  screen "add_pet_form" {
    title = "Add pet form"
    actor = actor.clinic_staff
    to    = [command.add_pet]
  }

  command "add_pet" {
    title     = "Add Pet"
    aggregate = aggregate.pet_management.pet
    to        = [event.pet_management.pet_added]

    field "pet_id" {
      type = field_type.pet_management.pet_id
    }
  }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
}

func TestValidateSource_AcceptsExplicitTranslationWorkflow(t *testing.T) {
	model := nativeCatalog() + `
translation "import_pet" {
  title = "Import Pet"

  processor "translate_pet" {
    title = "Translate Pet"
    from  = [event.partner.pet_received]
    to    = [command.add_pet]
  }

  command "add_pet" {
    title = "Add Pet"
    to    = [event.pet_management.pet_added]
  }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
}

func TestValidateSource_RejectsUnresolvedTypedReference(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"

  screen "form" {
    title = "Form"
    to    = [command.add_pet]
  }

  command "add_pet" {
    title = "Add Pet"
    to    = [event.pet_management.missing]
  }
}
`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, `event.pet_management.missing`)
}

func TestValidateSource_AllowsSameElementIDInDifferentWorkflows(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  command "submit" { title = "Add Pet" }
}

state_change "edit_pet" {
  title = "Edit Pet"
  command "submit" { title = "Edit Pet" }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
}

func TestValidateSource_RejectsLegacyReferenceConstructs(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"event_ref", nativeCatalog() + `state_change "add_pet" {
  title = "Add Pet"
  event_ref "pet_management.pet_added" {}
}`},
		{"field_ref", `bounded_context "pet_management" {
  title = "Pet Management"
  event "pet_added" {
    title = "Pet Added"
    field_ref "pet_id" {}
  }
}`},
		{"inbound", nativeCatalog() + `state_view "pets" {
  title = "Pets"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets exist?"
    inbound  = ["pet_management.pet_added"]
  }
}`},
		{"specification", nativeCatalog() + `state_change "add_pet" {
  title = "Add Pet"
  specification "success" {
    title     = "Success"
    linked_id = "add_pet"
  }
}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := validateModel(t, test.model)
			requireDiagnostic(t, diagnostics, "Unsupported")
		})
	}
}

func TestValidateSource_RequiresReadModelQuestion(t *testing.T) {
	model := nativeCatalog() + `
state_view "pets" {
  title = "Pets"
  readmodel "pets" { title = "Pets" }
}
`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "Missing required argument")
}

func TestValidateSource_AcceptsPatternSpecificScenarios(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"

  command "add_pet" {
    title = "Add Pet"
    to    = [event.pet_management.pet_added]
  }

  scenario "success" {
    title = "Pet is added"
    given { event = event.partner.pet_received }
    when  { command = command.add_pet }
    then  { event = event.pet_management.pet_added }
  }

  scenario "invalid" {
    title = "Pet is rejected"
    when  { command = command.add_pet }
    then  { error = "Pet name is required" }
  }
}

state_view "list_pets" {
  title = "List Pets"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets exist?"
    from     = [event.pet_management.pet_added]
  }
  scenario "loaded" {
    title = "Pets are listed"
    given { event = event.pet_management.pet_added }
    then  { readmodel = readmodel.pets }
  }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
}

func TestValidateSource_RejectsScenarioTargetForWrongPattern(t *testing.T) {
	model := nativeCatalog() + `
state_view "list_pets" {
  title = "List Pets"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets exist?"
  }
  scenario "wrong" {
    title = "Wrong"
    when { readmodel = readmodel.pets }
    then { readmodel = readmodel.pets }
  }
}
`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "state_view scenarios may not contain when steps")
}

func TestValidateSource_RejectsReverseFlowForms(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"command from screen", nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  screen "form" { title = "Form" }
  command "add_pet" {
    title = "Add Pet"
    from  = [screen.form]
  }
}`},
		{"screen from readmodel", nativeCatalog() + `
state_view "list_pets" {
  title = "List Pets"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets exist?"
  }
  screen "list" {
    title = "List"
    from  = [readmodel.pets]
  }
}`},
		{"processor from readmodel", nativeCatalog() + `
automation "notify_pet" {
  title = "Notify Pet"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets need notification?"
  }
  processor "notifier" {
    title = "Notifier"
    from  = [readmodel.pets]
  }
}`},
		{"command from processor", nativeCatalog() + `
automation "notify_pet" {
  title = "Notify Pet"
  processor "notifier" { title = "Notifier" }
  command "notify" {
    title = "Notify"
    from  = [processor.notifier]
  }
}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireDiagnostic(t, validateModel(t, test.model), "Invalid flow reference")
		})
	}
}

func TestValidateSource_RejectsQueryScenarioTarget(t *testing.T) {
	model := nativeCatalog() + `
state_view "list_pets" {
  title = "List Pets"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets exist?"
  }
  scenario "loaded" {
    title = "Pets are listed"
    given { event = event.pet_management.pet_added }
    when  { query = readmodel.pets }
    then  { readmodel = readmodel.pets }
  }
}`

	requireDiagnostic(t, validateModel(t, model), "Unsupported argument")
}

func TestValidateSource_RequiresGivenForStateViewScenario(t *testing.T) {
	model := nativeCatalog() + `
state_view "list_pets" {
  title = "List Pets"
  readmodel "pets" {
    title    = "Pets"
    question = "Which pets exist?"
  }
  scenario "loaded" {
    title = "Pets are listed"
    then { readmodel = readmodel.pets }
  }
}`

	requireDiagnostic(t, validateModel(t, model), "state_view scenarios must contain at least one given step")
}

func TestValidateSource_AcceptsTitlelessBlocks(t *testing.T) {
	model := `bounded_context "catalog" {
  aggregate "pet" {}
  field_type "pet_id" { type = "UUID" }
  event "pet_added" {}
}
actor "viewer" { auth_required = true }
team "clinic" {}
system "partner" {}
state_change "add_pet" {
  command "add_pet" {}
  screen_image "form" {}
  table "pets" {}
  scenario "success" {
    when { command = command.add_pet }
    then { error = "Invalid pet" }
  }
}
state_view "view_pets" {
  readmodel "pets" { question = "Which pets exist?" }
  screen "list" {}
}
automation "notify_pet" {
  processor "notifier" {}
}
chapter "registration" {
  workflows = [workflow.add_pet, workflow.view_pets, workflow.notify_pet]
}`

	requireNoErrors(t, validateModel(t, model))
}

func TestValidateSource_ReverseCommandFlowDoesNotProvideAReason(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  screen "form" { title = "Form" }
  command "add_pet" {
    title = "Add Pet"
    from  = [screen.form]
    to    = [event.pet_management.pet_added]
  }
}`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "Invalid flow reference")
	requireWarning(t, diagnostics, "Every command has a reason")
}

func TestValidateSource_AcceptsWorkshopNotation(t *testing.T) {
	model := nativeCatalog() + `
team "clinic_team" { title = "Clinic team" }
system "partner" {
  title    = "Partner"
  external = true
}
actor "clinic_staff" {
  title         = "Clinic staff"
  auth_required = true
}

state_change "add_pet" {
  title = "Add Pet"
  owner = team.clinic_team
}

chapter "registration" {
  title     = "Registration"
  workflows = [workflow.add_pet]
}

hotspot "owner_source" {
  question = "Where does owner information come from?"
  on       = workflow.add_pet
  status   = "open"
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
}

func TestValidateSource_AcceptsHotspotOnWorkflowElement(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  command "submit" {
    title            = "Submit"
    external_trigger = true
  }
}
hotspot "validation_rules" {
  question = "Which validation rules apply?"
  on       = command.add_pet.submit
}`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
}

func TestValidateSource_AcceptsCheatSheetStatuses(t *testing.T) {
	statuses := []string{"created", "planned", "assigned", "in_progress", "review", "blocked", "done", "informational"}
	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			model := `state_change "workflow" {
  title  = "Workflow"
  status = "` + status + `"
}`
			requireNoErrors(t, validateModel(t, model))
		})
	}
}

func TestValidateSource_ReportsEventModelingSmellsAsWarnings(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  screen "form" {
    title = "Form"
    to    = [command.one, command.two]
  }
  command "one" {
    title = "One"
    to    = [event.pet_management.pet_added, event.partner.pet_received]
  }
  command "two" { title = "Two" }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
	requireWarning(t, diagnostics, "bed")
	requireWarning(t, diagnostics, "left chair")
}

func TestValidateSource_ReportsRightChairAsWarning(t *testing.T) {
	model := nativeCatalog() + `
state_view "pet_history" {
  title = "Pet history"
  readmodel "history" {
    title    = "History"
    question = "What happened to this pet?"
    from     = [event.pet_management.pet_added, event.partner.pet_received]
  }
}`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
	requireWarning(t, diagnostics, "right chair")
}

func TestValidateSource_ReportsShelfAsWarning(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  command "add" {
    title            = "Add"
    external_trigger = true
  }
  scenario "success" {
    title = "Success"
    when { command = command.add }
    then { event = event.pet_management.pet_added }
  }
  scenario "failure" {
    title = "Failure"
    when { command = command.add }
    then { error = "Invalid pet" }
  }
}
state_change "edit_pet" { title = "Edit Pet" }
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
	requireWarning(t, diagnostics, "shelf")
}

func TestValidateSource_WarnsWhenCommandHasNoReason(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  command "add_pet" {
    title = "Add Pet"
    to    = [event.pet_management.pet_added]
  }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
	requireWarning(t, diagnostics, "Every command has a reason")
}

func TestValidateSource_AcceptsCommandReasonFromIncomingFlow(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  screen "form" {
    title = "Form"
    to    = [command.add_pet]
  }
  command "add_pet" {
    title = "Add Pet"
    to    = [event.pet_management.pet_added]
  }
}
`

	diagnostics := validateModel(t, model)

	requireNoErrors(t, diagnostics)
	requireNoWarning(t, diagnostics, "Every command has a reason")
}

func TestValidateSource_RejectsTranslationWithoutExternalInput(t *testing.T) {
	model := nativeCatalog() + `
translation "translate_pet" {
  title = "Translate Pet"
  processor "translator" {
    title = "Translator"
    from  = [event.pet_management.pet_added]
  }
}
`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "translation must consume at least one event from an external bounded_context")
}

func TestValidateSource_RejectsAutomationWithExternalInput(t *testing.T) {
	model := nativeCatalog() + `
automation "translate_pet" {
  title = "Translate Pet"
  processor "translator" {
    title = "Translator"
    from  = [event.partner.pet_received]
  }
}
`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "use a translation workflow")
}

func TestValidateSource_RejectsChapterWhoseWorkflowsAreNotContiguous(t *testing.T) {
	model := `state_change "first" { title = "First" }
state_change "middle" { title = "Middle" }
state_change "last" { title = "Last" }
chapter "disconnected" {
  title     = "Disconnected"
  workflows = [workflow.first, workflow.last]
}`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "chapter workflows must be contiguous")
}

func TestValidateSource_RejectsEmptyChapter(t *testing.T) {
	model := `chapter "empty" {
  title     = "Empty"
  workflows = []
}`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, "chapter must contain at least one workflow")
}

func TestValidateSource_RejectsDuplicateWorkshopItemID(t *testing.T) {
	model := `hotspot "open_question" { question = "First?" }
hotspot "open_question" { question = "Second?" }`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, `hotspot "open_question" is declared more than once`)
}

func nativeCatalog() string {
	return `bounded_context "pet_management" {
  title = "Pet Management"
  aggregate "pet" {}
  field_type "pet_id" { type = "UUID" }
  event "pet_added" {
    title     = "Pet Added"
    aggregate = aggregate.pet
    field "pet_id" { type = field_type.pet_id }
  }
}

bounded_context "partner" {
  title    = "Partner"
  external = true
  event "pet_received" { title = "Pet Received" }
}
`
}

func requireNoErrors(t *testing.T, diagnostics hcl.Diagnostics) {
	t.Helper()
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s", diagnostics.Error())
	}
}

func requireWarning(t *testing.T, diagnostics hcl.Diagnostics, want string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == hcl.DiagWarning &&
			(diagnostic.Summary == want || containsText(diagnostic.Summary, diagnostic.Detail, want)) {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want warning containing %q", diagnostics, want)
}

func requireNoWarning(t *testing.T, diagnostics hcl.Diagnostics, unwanted string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == hcl.DiagWarning && containsText(diagnostic.Summary, diagnostic.Detail, unwanted) {
			t.Fatalf("diagnostics = %#v, did not want warning containing %q", diagnostics, unwanted)
		}
	}
}

func containsText(summary, detail, want string) bool {
	return containsFold(summary, want) || containsFold(detail, want)
}

func containsFold(value, fragment string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(fragment))
}
