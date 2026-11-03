package validator

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestValidateSource_AcceptsArbitraryPrecisionIntegerExample(t *testing.T) {
	model := `bounded_context "example" {
  title = "Example"
  field_type "large_number" {
    type    = "Int"
    example = 999999999999999999999999999999999999999999999999999999999999
  }
}`

	requireNoErrors(t, validateModel(t, model))
}

func TestValidateSource_AcceptsPIIMetadata(t *testing.T) {
	model := `bounded_context "example" {
  title = "Example"
  field_type "name" {
    type = "String"
    pii  = true
  }
}`

	requireNoErrors(t, validateModel(t, model))
}

func TestValidateSource_RejectsNonBooleanPIIMetadata(t *testing.T) {
	model := `bounded_context "example" {
  title = "Example"
  field_type "name" {
    type = "String"
    pii  = "personal"
  }
}`

	requireDiagnostic(t, validateModel(t, model), "pii must be a boolean")
}

func TestValidateSource_AcceptsNativeExampleLiterals(t *testing.T) {
	model := `bounded_context "example" {
  title = "Example"
  field_type "flag" {
    type = "Boolean"
    example = true
  }
  field_type "weight" {
    type = "Double"
    example = 4.2
  }
  field_type "unset" {
    type = "Int"
    example = null
  }
  field_type "metadata" {
    type = "Custom"
    example = { nested = "value" }
  }
}`

	requireNoErrors(t, validateModel(t, model))
}

func TestValidateSource_RejectsExampleThatDoesNotMatchFieldType(t *testing.T) {
	tests := []struct {
		name  string
		field string
	}{
		{"integer", "field_type \"value\" {\n    type = \"Int\"\n    example = \"five\"\n  }"},
		{"boolean", "field_type \"value\" {\n    type = \"Boolean\"\n    example = 1\n  }"},
		{"list shape", "field_type \"value\" {\n    type = \"Custom\"\n    cardinality = \"List\"\n    example = { id = 1 }\n  }"},
		{"list member", "field_type \"value\" {\n    type = \"Int\"\n    cardinality = \"List\"\n    example = [1, \"two\"]\n  }"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := "bounded_context \"example\" {\n  title = \"Example\"\n  " + test.field + "\n}"
			requireDiagnostic(t, validateModel(t, model), "Invalid example value")
		})
	}
}

func TestValidateSource_RejectsUnknownSyntax(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"top-level block", `event "event" {}`},
		{"workflow attribute", "state_change \"example\" {\n  title = \"Example\"\n  unknown = true\n}"},
		{"workflow block", "state_change \"example\" {\n  title = \"Example\"\n  unknown \"value\" {}\n}"},
		{"element attribute", "state_change \"example\" {\n  title = \"Example\"\n  command \"submit\" {\n    title = \"Submit\"\n    unknown = true\n  }\n}"},
		{"field attribute", "bounded_context \"example\" {\n  title = \"Example\"\n  field_type \"name\" {\n    type = \"String\"\n    unknown = true\n  }\n}"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireDiagnostic(t, validateModel(t, test.model), "Unsupported")
		})
	}
}

func TestValidateSource_RejectsLiteralVariableOutsideReferencePositions(t *testing.T) {
	model := `state_change "example" { title = var.title }`

	requireDiagnostic(t, validateModel(t, model), "Variables not allowed")
}

func TestValidateSource_RejectsNonSnakeCaseLabels(t *testing.T) {
	model := `state_change "add-pet" { title = "Add Pet" }`

	requireDiagnostic(t, validateModel(t, model), "lower_snake_case")
}

func TestValidateSource_RejectsDuplicateLocalElementID(t *testing.T) {
	model := `state_change "add_pet" {
  title = "Add Pet"
  command "submit" { title = "Submit" }
  command "submit" { title = "Submit again" }
}`

	requireDiagnostic(t, validateModel(t, model), "Duplicate command id")
}

func TestValidateSource_RejectsDuplicateCatalogAddress(t *testing.T) {
	model := `bounded_context "pets" {
  title = "Pets"
  event "pet_added" { title = "Pet Added" }
  event "pet_added" { title = "Pet Added Again" }
}`

	requireDiagnostic(t, validateModel(t, model), "Duplicate event id")
}

func TestValidateSource_RejectsDuplicateFieldNameInOneContract(t *testing.T) {
	model := `bounded_context "catalog" {
  title = "Catalog"
  event "book_added" {
    title = "Book Added"
    field "book_id" { type = "UUID" }
    field "book_id" { type = "UUID" }
  }
}`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, `field "book_id" is declared more than once`)
}

func TestValidateSource_RejectsDuplicateScenarioIDInOneWorkflow(t *testing.T) {
	model := nativeCatalog() + `
state_change "add_pet" {
  title = "Add Pet"
  command "add_pet" {
    title            = "Add Pet"
    external_trigger = true
  }
  scenario "success" {
    title = "Success"
    when { command = command.add_pet }
    then { event = event.pet_management.pet_added }
  }
  scenario "success" {
    title = "Also success"
    when { command = command.add_pet }
    then { event = event.pet_management.pet_added }
  }
}`

	diagnostics := validateModel(t, model)

	requireDiagnostic(t, diagnostics, `scenario "success" is declared more than once`)
}

func TestValidateSource_RejectsInvalidWorkflowChild(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		child    string
	}{
		{"state change readmodel", "state_change", "readmodel \"summary\" {\n    title = \"Summary\"\n    question = \"What happened?\"\n  }"},
		{"state view command", "state_view", `command "refresh" { title = "Refresh" }`},
		{"automation screen", "automation", `screen "form" { title = "Form" }`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := test.workflow + ` "workflow" {
  title = "Workflow"
  ` + test.child + `
}`
			requireDiagnostic(t, validateModel(t, model), "Invalid workflow child")
		})
	}
}

func TestValidateSource_RejectsWrongKindFlowReference(t *testing.T) {
	model := nativeCatalog() + `state_change "add_pet" {
  title = "Add Pet"
  command "add_pet" {
    title = "Add Pet"
    to    = [readmodel.missing]
  }
}`

	requireDiagnostic(t, validateModel(t, model), "Invalid flow reference")
}

func TestValidateSource_ReturnsIndependentDiagnosticsInSchemaOrder(t *testing.T) {
	model := `state_change "example" {
  status = "pending"
  title  = 1
}`

	diagnostics := validateModel(t, model)
	want := []string{
		"title must be a string.",
		"status must be one of: created, planned, assigned, in_progress, review, blocked, done, informational.",
	}
	if got := diagnosticMessages(diagnostics); !slices.Equal(got, want) {
		t.Fatalf("diagnostic messages = %#v, want %#v", got, want)
	}
}

func TestValidateSourceWithProfile_AssignsStableCodes(t *testing.T) {
	model := nativeCatalog() + `state_change "add_pet" {
  command "add_pet" {
    to = [readmodel.missing]
  }
}`

	diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(withBoard(model)), Valid)

	if got, want := diagnosticCodeOf(t, diagnostics), "EM201"; got != want {
		t.Fatalf("diagnostic code = %q, want %q", got, want)
	}
}

func TestValidateSourceWithProfile_ExplainsFlowReferenceKindMismatch(t *testing.T) {
	model := nativeCatalog() + `state_change "add_pet" {
  command "add_pet" {
    to = [readmodel.missing]
  }
}`

	diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(withBoard(model)), Valid)

	if got, want := diagnosticDetailForCode(t, diagnostics, "EM201"), "command.to may reference: event.<context>.<id>. You referenced: readmodel.missing."; got != want {
		t.Fatalf("flow diagnostic detail = %q, want %q", got, want)
	}
}

func TestValidateSourceWithProfile_ExplainsStateViewWhen(t *testing.T) {
	model := nativeCatalog() + `state_view "list_pets" {
  readmodel "pets" { question = "Which pets exist?" }
  scenario "loaded" {
    given { event = event.pet_management.pet_added }
    when { readmodel = readmodel.pets }
    then { readmodel = readmodel.pets }
  }
}`

	diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(withBoard(model)), Valid)

	if got, want := diagnosticDetailForCode(t, diagnostics, "EM301"), "state_view scenarios may not contain when steps. Found: when."; got != want {
		t.Fatalf("scenario diagnostic detail = %q, want %q", got, want)
	}
}

func TestValidateSourceWithProfile_AutomationScenarioForms(t *testing.T) {
	automation := func(scenario string) string {
		return nativeCatalog() + `automation "remind_owners" {
  readmodel "pets_to_remind" {
    question = "Which added pets still need a reminder?"
    from     = [event.pet_management.pet_added]
    to       = [processor.reminder]
  }
  processor "reminder" {
    to = [command.remind]
  }
  command "remind" {
    to = [event.pet_management.pet_added]
  }
` + scenario + `
}
state_view "list_pets" {
  readmodel "pets" {
    question = "Which pets exist?"
    from     = [event.pet_management.pet_added]
  }
}`
	}
	tests := []struct {
		name       string
		scenario   string
		wantCode   string
		wantDetail string
		absentCode string
	}{
		{
			name: "view form states the to-do list",
			scenario: `  scenario "pet_waits" {
    given { event = event.pet_management.pet_added }
    then  { readmodel = readmodel.pets_to_remind }
  }`,
		},
		{
			name: "view form accepts the self-qualified read model",
			scenario: `  scenario "pet_waits" {
    given { event = event.pet_management.pet_added }
    then  { readmodel = readmodel.remind_owners.pets_to_remind }
  }`,
		},
		{
			name: "action form keeps a given read model",
			scenario: `  scenario "reminded" {
    given { readmodel = readmodel.pets_to_remind }
    when  { processor = processor.reminder }
    then  { event = event.pet_management.pet_added }
  }`,
		},
		{
			name: "view form needs a given event",
			scenario: `  scenario "empty" {
    then { readmodel = readmodel.pets_to_remind }
  }`,
			wantCode:   "EM301",
			wantDetail: "an automation scenario without a when step is a view form scenario and must contain at least one given event step. An action form scenario needs exactly one when step.",
		},
		{
			name: "missing when makes a then event illegal",
			scenario: `  scenario "forgot_when" {
    given { event = event.pet_management.pet_added }
    then  { event = event.pet_management.pet_added }
  }`,
			wantCode:   "EM303",
			wantDetail: "automation view form scenario then steps may use: readmodel or error. You referenced: event. An automation scenario without a when step is a view form scenario. An action form scenario needs exactly one when step.",
		},
		{
			name: "view form given is an event",
			scenario: `  scenario "given_list" {
    given { readmodel = readmodel.pets_to_remind }
    then  { readmodel = readmodel.pets_to_remind }
  }`,
			wantCode:   "EM303",
			wantDetail: "automation view form scenario given steps may use: event. You referenced: readmodel. An automation scenario without a when step is a view form scenario. An action form scenario needs exactly one when step.",
		},
		{
			name: "view form read model of another workflow",
			scenario: `  scenario "other_list" {
    given { event = event.pet_management.pet_added }
    then  { readmodel = readmodel.list_pets.pets }
  }`,
			wantCode:   "EM101",
			wantDetail: `a view form scenario may reference only a read model of its own automation "remind_owners". You referenced: readmodel.list_pets.pets.`,
		},
		{
			name: "foreign given read model is a target error only",
			scenario: `  scenario "foreign_given" {
    given { readmodel = readmodel.list_pets.pets }
    then  { readmodel = readmodel.pets_to_remind }
  }`,
			wantCode:   "EM303",
			wantDetail: "automation view form scenario given steps may use: event. You referenced: readmodel. An automation scenario without a when step is a view form scenario. An action form scenario needs exactly one when step.",
			absentCode: "EM101",
		},
		{
			name: "action form with two when steps",
			scenario: `  scenario "twice" {
    when { processor = processor.reminder }
    when { command = command.remind }
    then { event = event.pet_management.pet_added }
  }`,
			wantCode:   "EM301",
			wantDetail: "a scenario must contain exactly one when step.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(withBoard(automation(test.scenario))), Valid)
			if test.wantCode == "" {
				if diagnostics.HasErrors() {
					t.Fatalf("diagnostics = %s, want no errors", diagnostics.Error())
				}
				return
			}
			if got := diagnosticDetailForCode(t, diagnostics, test.wantCode); got != test.wantDetail {
				t.Fatalf("%s detail = %q, want %q", test.wantCode, got, test.wantDetail)
			}
			for _, diagnostic := range diagnostics {
				if test.absentCode != "" && DiagnosticCode(diagnostic) == test.absentCode {
					t.Fatalf("diagnostics include %s: %s", test.absentCode, diagnostics.Error())
				}
			}
		})
	}
}

func TestValidateSourceWithProfile_AppliesProfileSeverity(t *testing.T) {
	model := withBoard(`state_change "add_pet" {
  command "add_pet" {}
}
hotspot "missing_rule" {
  question = "Which rule is missing?"
}`)

	valid := ValidateSourceWithProfile("model.em.hcl", []byte(model), Valid)
	requireSeverityForCode(t, valid, "EM404", hcl.DiagWarning)
	requireSeverityForCode(t, valid, "EM406", hcl.DiagWarning)
	if valid.HasErrors() {
		t.Fatalf("valid profile diagnostics = %s, want no errors", valid.Error())
	}

	strict := ValidateSourceWithProfile("model.em.hcl", []byte(model), Strict)
	requireSeverityForCode(t, strict, "EM404", hcl.DiagError)
	requireSeverityForCode(t, strict, "EM406", hcl.DiagError)
	if !strict.HasErrors() {
		t.Fatal("strict profile must escalate command and hotspot diagnostics")
	}

	workshop := ValidateSourceWithProfile("model.em.hcl", []byte(model), Workshop)
	requireSeverityForCode(t, workshop, "EM404", hcl.DiagInvalid)
	requireSeverityForCode(t, workshop, "EM406", hcl.DiagInvalid)
}

func TestValidateSourceWithProfile_ScreenKind(t *testing.T) {
	const catalog = `bounded_context "clinic" {
  event "pet_registered" {}
  event "pet_archived" {}
}
`
	tests := []struct {
		name     string
		workflow string
		wantCode string
	}{
		{name: "omitted", workflow: `state_change "register" {
  screen "request" { to = [command.register] }
  command "register" { to = [event.clinic.pet_registered] }
}`},
		{name: "api", workflow: `state_change "register" {
  screen "request" {
    kind         = "api"
    api_endpoint = "POST /pets"
    to           = [command.register]
  }
  command "register" { to = [event.clinic.pet_registered] }
}`},
		{name: "ui", workflow: `state_change "register" {
  screen "request" {
    kind = "ui"
    to   = [command.register]
  }
  command "register" { to = [event.clinic.pet_registered] }
}`},
		{name: "api response", workflow: `state_view "pets" {
  readmodel "pets" {
    question = "Which pets exist?"
    from     = [event.clinic.pet_registered]
    to       = [screen.response]
  }
  screen "response" { kind = "api" }
}`},
		{name: "unknown string", wantCode: "EM008", workflow: `state_change "register" {
  screen "request" {
    kind = "rest"
    to   = [command.register]
  }
  command "register" { to = [event.clinic.pet_registered] }
}`},
		{name: "not a string", wantCode: "EM007", workflow: `state_change "register" {
  screen "request" {
    kind = true
    to   = [command.register]
  }
  command "register" { to = [event.clinic.pet_registered] }
}`},
		{name: "on command", wantCode: "EM000", workflow: `state_change "register" {
  command "register" {
    kind             = "api"
    external_trigger = true
    to               = [event.clinic.pet_registered]
  }
}`},
		{name: "on readmodel", wantCode: "EM000", workflow: `state_view "pets" {
  readmodel "pets" {
    kind     = "api"
    question = "Which pets exist?"
    from     = [event.clinic.pet_registered]
  }
}`},
		{name: "on processor", wantCode: "EM000", workflow: `automation "archive" {
  processor "archiver" {
    kind = "api"
    from = [event.clinic.pet_registered]
    to   = [command.archive]
  }
  command "archive" { to = [event.clinic.pet_archived] }
}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(withBoard(catalog+test.workflow)), Strict)
			if test.wantCode == "" {
				if diagnostics.HasErrors() {
					t.Fatalf("diagnostics = %s, want no errors", diagnostics.Error())
				}
				return
			}
			requireSeverityForCode(t, diagnostics, test.wantCode, hcl.DiagError)
		})
	}
}

func TestValidateSourceWithProfile_CommandReason(t *testing.T) {
	const catalog = `bounded_context "clinic" {
  event "pet_registered" {}
  event "pet_archived" {}
}
`
	tests := []struct {
		name     string
		workflow string
		want     bool
	}{
		{name: "only api_endpoint", want: true, workflow: `state_change "register" {
  command "register" {
    api_endpoint = "POST /pets"
    to           = [event.clinic.pet_registered]
  }
}`},
		{name: "api screen and api_endpoint", workflow: `state_change "register" {
  screen "request" {
    kind = "api"
    to   = [command.register]
  }
  command "register" {
    api_endpoint = "POST /pets"
    to           = [event.clinic.pet_registered]
  }
}`},
		{name: "external_trigger", workflow: `state_change "register" {
  command "register" {
    external_trigger = true
    to               = [event.clinic.pet_registered]
  }
}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(withBoard(catalog+test.workflow)), Valid)
			if got := len(diagnosticsWithCode(diagnostics, "EM404")) == 1; got != test.want {
				t.Fatalf("EM404 reported = %v, want %v: %s", got, test.want, diagnostics.Error())
			}
		})
	}
}

func TestValidateSourceWithProfile_APIScreenBed(t *testing.T) {
	model := withBoard(`bounded_context "clinic" {
  event "pet_registered" {}
}
state_change "register" {
  screen "request" {
    kind = "api"
    to   = [command.register, command.import]
  }
  command "register" { to = [event.clinic.pet_registered] }
  command "import" { to = [event.clinic.pet_registered] }
}`)
	requireSeverityForCode(t, ValidateSourceWithProfile("model.em.hcl", []byte(model), Valid), "EM401", hcl.DiagWarning)
}

func TestValidateSource_RejectsTypelessEventField(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  field_type "pet_id" { type = "UUID" }
  event "pet_added" {
    field "pet_id" {}
  }
}`

	missing := requireOneDiagnostic(t, validateModel(t, model), "EM009", hcl.DiagError, "model.em.hcl")
	if missing.Summary != "Missing field type" || missing.Subject.Start.Line != 5 {
		t.Fatalf("EM009 = %q at line %d, want Missing field type at line 5", missing.Summary, missing.Subject.Start.Line)
	}
}

func TestValidateSource_RejectsTypelessWorkflowElementField(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  aggregate "pet" {}
  field_type "pet_id" { type = "UUID" }
  event "pet_added" { aggregate = aggregate.pet }
}

state_change "add_pet" {
  title = "Add Pet"
  command "add_pet" {
    title            = "Add Pet"
    external_trigger = true
    to               = [event.clinic.pet_added]
    field "pet_id" {}
  }
}`

	missing := requireOneDiagnostic(t, validateModel(t, model), "EM009", hcl.DiagError, "model.em.hcl")
	if missing.Summary != "Missing field type" || missing.Subject.Start.Line != 14 {
		t.Fatalf("EM009 = %q at line %d, want Missing field type at line 14", missing.Summary, missing.Subject.Start.Line)
	}
}

func TestValidateSource_RejectsTypelessNestedSubfield(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  field_type "street" { type = "String" }
  event "pet_added" {
    field "address" {
      type = "Custom"
      subfield "street" {}
    }
  }
}`

	missing := requireOneDiagnostic(t, validateModel(t, model), "EM009", hcl.DiagError, "model.em.hcl")
	if missing.Summary != "Missing field type" || missing.Subject.Start.Line != 7 {
		t.Fatalf("EM009 = %q at line %d, want Missing field type at line 7", missing.Summary, missing.Subject.Start.Line)
	}
}

func TestValidateSource_AcceptsScreenFieldsWithShorthand(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  field_type "pet_id" { type = "UUID" }
  field_type "pet_name" { type = "String" }
}

state_change "add_pet" {
  title = "Add Pet"
  screen "add_pet_form" {
    title  = "Add pet form"
    fields = [field_type.clinic.pet_name]

    field "pet_id" { type = field_type.clinic.pet_id }
  }
  command "add_pet" {
    title            = "Add Pet"
    external_trigger = true
  }
}`

	requireNoErrors(t, validateModel(t, model))
}

func TestValidateSource_RejectsFieldTypeDeclarationWithoutType(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  field_type "pet_id" {}
}`

	requireDiagnostic(t, validateModel(t, model), "must set an explicit built-in type")
}

func TestValidateSource_SynthesizesFieldsFromListShorthand(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  field_type "pet_id" { type = "UUID" }
  field_type "pet_name" { type = "String" }
  event "pet_added" {
    fields = [field_type.pet_id, field_type.pet_name]
  }
}`

	requireNoErrors(t, validateModel(t, model))
}

func TestValidateSource_RejectsDuplicateNameBetweenListAndBlockFields(t *testing.T) {
	model := `bounded_context "clinic" {
  title = "Clinic"
  field_type "pet_id" { type = "UUID" }
  event "pet_added" {
    fields = [field_type.pet_id]

    field "pet_id" { type = field_type.pet_id }
  }
}`

	requireDiagnostic(t, validateModel(t, model), `field "pet_id" is declared more than once`)
}

func validateModel(t *testing.T, model string) hcl.Diagnostics {
	t.Helper()
	return ValidateSource("model.em.hcl", []byte(withBoard(model)))
}

var workflowDeclaration = regexp.MustCompile(`(?m)^\s*(?:state_change|state_view|automation)\s+"([a-z][a-z0-9_]*)"`)

// withBoard appends to model a board that lists every workflow it declares,
// for tests about rules other than the board.
func withBoard(model string) string {
	return model + boardFor(model)
}

// boardFor returns a board block listing each workflow the sources declare
// once, in declaration order.
func boardFor(sources ...string) string {
	var entries []string
	for _, source := range sources {
		for _, match := range workflowDeclaration.FindAllStringSubmatch(source, -1) {
			if entry := "workflow." + match[1]; !slices.Contains(entries, entry) {
				entries = append(entries, entry)
			}
		}
	}
	return "\nboard {\n  workflows = [" + strings.Join(entries, ", ") + "]\n}\n"
}

func requireDiagnostic(t *testing.T, diagnostics hcl.Diagnostics, want string) {
	t.Helper()
	if !diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %#v, want error containing %q", diagnostics, want)
	}
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Summary, want) || strings.Contains(diagnostic.Detail, want) {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want error containing %q", diagnostics, want)
}

func diagnosticMessages(diagnostics hcl.Diagnostics) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Detail)
	}
	return messages
}

func diagnosticCodeOf(t *testing.T, diagnostics hcl.Diagnostics) string {
	t.Helper()
	if len(diagnostics) == 0 {
		t.Fatal("diagnostics = empty, want at least one diagnostic")
	}
	return DiagnosticCode(diagnostics[0])
}

func requireSeverityForCode(t *testing.T, diagnostics hcl.Diagnostics, code string, want hcl.DiagnosticSeverity) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if DiagnosticCode(diagnostic) == code {
			if diagnostic.Severity != want {
				t.Fatalf("diagnostic %s severity = %v, want %v", code, diagnostic.Severity, want)
			}
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want code %s", diagnostics, code)
}

func diagnosticDetailForCode(t *testing.T, diagnostics hcl.Diagnostics, code string) string {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if DiagnosticCode(diagnostic) == code {
			return diagnostic.Detail
		}
	}
	t.Fatalf("diagnostics = %#v, want code %s", diagnostics, code)
	return ""
}
