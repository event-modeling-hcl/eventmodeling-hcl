package validator

import (
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
		{"translation screen", "translation", `screen "form" { title = "Form" }`},
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

	diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(model), Valid)

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

	diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(model), Valid)

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

	diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(model), Valid)

	if got, want := diagnosticDetailForCode(t, diagnostics, "EM301"), "state_view scenarios may not contain when steps. Found: when."; got != want {
		t.Fatalf("scenario diagnostic detail = %q, want %q", got, want)
	}
}

func TestValidateSourceWithProfile_AppliesProfileSeverity(t *testing.T) {
	model := `state_change "add_pet" {
  command "add_pet" {}
}
hotspot "missing_rule" {
  question = "Which rule is missing?"
}`

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

func validateModel(t *testing.T, model string) hcl.Diagnostics {
	t.Helper()
	return ValidateSource("model.em.hcl", []byte(model))
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
