package validator

import (
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

func TestValidateFiles_ReportsMissingBoardAtStartOfFirstFile(t *testing.T) {
	diagnostics := validateBoardFiles(t, Valid,
		modelFile("b.em.hcl", "state_change \"second\" { title = \"Second\" }\n"),
		modelFile("a.em.hcl", "state_change \"first\" { title = \"First\" }\n"),
	)

	missing := requireOneDiagnostic(t, diagnostics, "EM015", hcl.DiagError, "a.em.hcl")
	if missing.Summary != "Missing board" || missing.Subject.Start.Line != 1 || missing.Subject.Start.Column != 1 {
		t.Fatalf("EM015 = %q at %v, want Missing board at a.em.hcl:1:1", missing.Summary, missing.Subject)
	}
}

func TestValidateSource_ReportsMissingBoardInModelWithoutWorkflows(t *testing.T) {
	diagnostics := ValidateSource("model.em.hcl", []byte(`actor "staff" { auth_required = true }`))

	requireOneDiagnostic(t, diagnostics, "EM015", hcl.DiagError, "model.em.hcl")
}

func TestValidateFiles_ReportsRepeatedBoardAtLaterBoard(t *testing.T) {
	diagnostics := validateBoardFiles(t, Valid,
		modelFile("b.em.hcl", "board {\n  workflows = [workflow.first]\n}\n"),
		modelFile("a.em.hcl", "state_change \"first\" { title = \"First\" }\nboard {\n  workflows = [workflow.first]\n}\n"),
	)

	repeated := requireOneDiagnostic(t, diagnostics, "EM015", hcl.DiagError, "b.em.hcl")
	if want := "a model must declare exactly one board block; first declared at a.em.hcl:2:1."; repeated.Detail != want {
		t.Fatalf("EM015 detail = %q, want %q", repeated.Detail, want)
	}
	if repeated.Summary != "Repeated board" {
		t.Fatalf("EM015 summary = %q, want Repeated board", repeated.Summary)
	}
}

func TestValidateSource_ReportsWorkflowRepeatedOnBoard(t *testing.T) {
	model := `state_change "first" { title = "First" }
state_change "second" { title = "Second" }
board {
  workflows = [workflow.first, workflow.second, workflow.first]
}
`

	repeated := requireOneDiagnostic(t, ValidateSource("model.em.hcl", []byte(model)), "EM016", hcl.DiagError, "model.em.hcl")
	if want := "workflow.first is already on the board."; repeated.Detail != want {
		t.Fatalf("EM016 detail = %q, want %q", repeated.Detail, want)
	}
	if repeated.Subject.Start.Line != 4 || repeated.Subject.Start.Column != 49 {
		t.Fatalf("EM016 subject = %v, want the third entry at 4:49", repeated.Subject)
	}
}

func TestValidateSource_ReportsUnknownBoardWorkflow(t *testing.T) {
	model := `state_change "first" { title = "First" }
board {
  workflows = [workflow.first, workflow.missing]
}
`

	unresolved := requireOneDiagnostic(t, ValidateSource("model.em.hcl", []byte(model)), "EM102", hcl.DiagError, "model.em.hcl")
	if want := "workflow.missing is not declared in the model."; unresolved.Detail != want {
		t.Fatalf("EM102 detail = %q, want %q", unresolved.Detail, want)
	}
}

func TestValidateSource_ReportsWrongKindBoardEntry(t *testing.T) {
	model := `state_change "first" { title = "First" }
chapter "one" {
  workflows = [workflow.first]
}
board {
  workflows = [chapter.one]
}
`

	requireOneDiagnostic(t, ValidateSource("model.em.hcl", []byte(model)), "EM101", hcl.DiagError, "model.em.hcl")
}

func TestValidateSource_ReportsBoardSchemaErrorsAsHCLDiagnostics(t *testing.T) {
	tests := []struct {
		name    string
		board   string
		summary string
	}{
		{"missing workflows", "board {}\n", "Missing required argument"},
		{"unknown attribute", "board {\n  workflows = []\n  title = \"Story\"\n}\n", "Unsupported argument"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := ValidateSource("model.em.hcl", []byte(test.board))
			schema := requireOneDiagnostic(t, diagnostics, "EM000", hcl.DiagError, "model.em.hcl")
			if schema.Summary != test.summary {
				t.Fatalf("EM000 summary = %q, want %q", schema.Summary, test.summary)
			}
			requireNoDiagnosticWithCode(t, diagnostics, "EM015")
		})
	}
}

func TestValidateSource_AcceptsEmptyAndPartialBoardsInEveryProfile(t *testing.T) {
	workflows := `state_change "first" { title = "First" }
state_change "second" { title = "Second" }
`
	for _, board := range []string{"board {\n  workflows = []\n}\n", "board {\n  workflows = [workflow.second]\n}\n"} {
		for _, profile := range []Profile{Workshop, Valid, Strict} {
			if diagnostics := ValidateSourceWithProfile("model.em.hcl", []byte(workflows+board), profile); len(diagnostics) != 0 {
				t.Fatalf("%s board %q diagnostics = %s, want none", profile, board, diagnostics.Error())
			}
		}
	}
}

func TestValidateDecodedDocument_PolicyRejectionKeepsValidatedDocument(t *testing.T) {
	model := withBoard(`state_change "first" { title = "First" }
hotspot "open_question" {
  question = "Which rule applies?"
  on       = workflow.first
}`)

	validated, diagnostics := validateDecoded(t, model, Strict)

	if validated == nil {
		t.Fatal("ValidatedDocument = nil under a policy rejection, want the core-clean document")
	}
	if !diagnostics.HasErrors() {
		t.Fatal("strict diagnostics have no error, want EM406 as a policy rejection")
	}
	if open := requireOneDiagnostic(t, diagnostics, "EM406", hcl.DiagError, "model.em.hcl"); !IsJudgment(open) {
		t.Fatal("EM406 is not a judgment signal")
	}
	if hasCoreErrors(diagnostics) {
		t.Fatalf("diagnostics = %s, want no core error", diagnostics.Error())
	}
}

func TestValidateDecodedDocument_CoreErrorWithholdsValidatedDocument(t *testing.T) {
	validated, diagnostics := validateDecoded(t, `state_change "first" { title = "First" }`, Workshop)

	if validated != nil {
		t.Fatal("ValidatedDocument != nil for a model without a board")
	}
	if !hasCoreErrors(diagnostics) || IsJudgment(diagnostics[0]) {
		t.Fatalf("diagnostics = %s, want a core error", diagnostics.Error())
	}
}

func validateDecoded(t *testing.T, model string, profile Profile) (*ValidatedDocument, hcl.Diagnostics) {
	t.Helper()
	document, parseDiagnostics := syntax.Parse("model.em.hcl", []byte(model))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("Parse diagnostics = %s, want none", parseDiagnostics.Error())
	}
	return ValidateDecodedDocument(source.Decode(document), profile)
}
