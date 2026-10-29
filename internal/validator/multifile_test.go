package validator

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// validateFiles validates files with a board listing every declared workflow
// appended to the last file, for tests about rules other than the board.
func validateFiles(t *testing.T, profile Profile, files ...syntax.File) hcl.Diagnostics {
	t.Helper()
	sources := make([]string, 0, len(files))
	for _, file := range files {
		sources = append(sources, string(file.Source))
	}
	withBoard := append([]syntax.File(nil), files...)
	last := &withBoard[len(withBoard)-1]
	last.Source = append(append([]byte(nil), last.Source...), boardFor(sources...)...)
	return validateBoardFiles(t, profile, withBoard...)
}

// validateBoardFiles validates files exactly as given.
func validateBoardFiles(t *testing.T, profile Profile, files ...syntax.File) hcl.Diagnostics {
	t.Helper()
	document, parseDiagnostics := syntax.ParseFiles(files)
	if parseDiagnostics.HasErrors() {
		t.Fatalf("ParseFiles diagnostics = %s, want none", parseDiagnostics.Error())
	}
	_, diagnostics := ValidateDecodedDocument(source.Decode(document), profile)
	return diagnostics
}

func diagnosticsWithCode(diagnostics hcl.Diagnostics, code string) []*hcl.Diagnostic {
	var matches []*hcl.Diagnostic
	for _, diagnostic := range diagnostics {
		if DiagnosticCode(diagnostic) == code {
			matches = append(matches, diagnostic)
		}
	}
	return matches
}

func requireOneDiagnostic(t *testing.T, diagnostics hcl.Diagnostics, code string, severity hcl.DiagnosticSeverity, filename string) *hcl.Diagnostic {
	t.Helper()
	matches := diagnosticsWithCode(diagnostics, code)
	if len(matches) != 1 {
		t.Fatalf("%s diagnostics = %d in %#v, want exactly 1", code, len(matches), diagnostics)
	}
	match := matches[0]
	if match.Severity != severity {
		t.Fatalf("%s severity = %v, want %v", code, match.Severity, severity)
	}
	if match.Subject == nil || match.Subject.Filename != filename {
		t.Fatalf("%s subject = %v, want file %s", code, match.Subject, filename)
	}
	return match
}

func requireNoDiagnosticWithCode(t *testing.T, diagnostics hcl.Diagnostics, code string) {
	t.Helper()
	if matches := diagnosticsWithCode(diagnostics, code); len(matches) != 0 {
		t.Fatalf("%s diagnostics = %#v, want none", code, matches)
	}
}

func modelFile(name, content string) syntax.File {
	return syntax.File{Name: name, Source: []byte(content)}
}

func TestValidateFiles_ResolvesReferencesAcrossFiles(t *testing.T) {
	diagnostics := validateFiles(t, Valid,
		modelFile("a.em.hcl", `chapter "pets" {
  workflows = [workflow.add_pet]
}
`),
		modelFile("b.em.hcl", `state_change "add_pet" { title = "Add Pet" }
`),
	)

	requireNoDiagnosticWithCode(t, diagnostics, "EM102")
	requireNoErrors(t, diagnostics)
}

func TestValidateFiles_ReportsDuplicateWorkflowAcrossFilesWithFirstLocation(t *testing.T) {
	diagnostics := validateFiles(t, Valid,
		modelFile("a.em.hcl", "state_change \"other\" { title = \"Other\" }\nstate_change \"add_pet\" { title = \"Add Pet\" }\n"),
		modelFile("b.em.hcl", "state_change \"add_pet\" { title = \"Add Pet\" }\n"),
	)

	duplicate := requireOneDiagnostic(t, diagnostics, "EM002", hcl.DiagError, "b.em.hcl")
	want := `workflow "add_pet" is declared more than once in its namespace. First declared at a.em.hcl:2:1.`
	if duplicate.Detail != want {
		t.Fatalf("EM002 detail = %q, want %q", duplicate.Detail, want)
	}
}

func TestValidateFiles_ReportsDuplicateCatalogIDAcrossFiles(t *testing.T) {
	diagnostics := validateFiles(t, Valid,
		modelFile("a.em.hcl", "actor \"customer\" { auth_required = true }\n"),
		modelFile("b.em.hcl", "bounded_context \"clinic\" { title = \"Clinic\" }\nactor \"customer\" { auth_required = false }\n"),
	)

	duplicate := requireOneDiagnostic(t, diagnostics, "EM002", hcl.DiagError, "b.em.hcl")
	if !strings.HasSuffix(duplicate.Detail, "First declared at a.em.hcl:1:1.") {
		t.Fatalf("EM002 detail = %q, want first declaration a.em.hcl:1:1", duplicate.Detail)
	}
}

func TestValidateSource_DuplicateIDNamesFirstDeclarationInSameFile(t *testing.T) {
	model := "state_change \"add_pet\" { title = \"Add Pet\" }\nstate_change \"add_pet\" { title = \"Again\" }\n"

	duplicate := requireOneDiagnostic(t, validateModel(t, model), "EM002", hcl.DiagError, "model.em.hcl")
	want := `workflow "add_pet" is declared more than once in its namespace. First declared at model.em.hcl:1:1.`
	if duplicate.Detail != want {
		t.Fatalf("EM002 detail = %q, want %q", duplicate.Detail, want)
	}
}

func TestValidateFiles_AcceptsChaptersInSeveralFiles(t *testing.T) {
	diagnostics := validateFiles(t, Strict,
		modelFile("a.em.hcl", `state_change "first" { title = "First" }
chapter "one" {
  workflows = [workflow.first]
}
`),
		modelFile("b.em.hcl", `state_change "second" { title = "Second" }
chapter "two" {
  workflows = [workflow.second]
}
`),
	)

	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %s, want none", diagnostics.Error())
	}
}

func TestValidateFiles_ReportsWorkflowInSeveralChapters(t *testing.T) {
	diagnostics := validateFiles(t, Valid,
		modelFile("a.em.hcl", `chapter "one" {
  workflows = [workflow.add_pet]
}
chapter "two" {
  workflows = [workflow.add_pet]
}
`),
		modelFile("b.em.hcl", `state_change "add_pet" { title = "Add Pet" }
`),
	)

	several := requireOneDiagnostic(t, diagnostics, "EM014", hcl.DiagError, "a.em.hcl")
	if want := "workflow.add_pet is already in chapter.one."; several.Detail != want {
		t.Fatalf("EM014 detail = %q, want %q", several.Detail, want)
	}
	if several.Subject.Start.Line != 5 {
		t.Fatalf("EM014 subject line = %d, want 5 (the later chapter)", several.Subject.Start.Line)
	}
	if several.Summary != "Workflow in several chapters" {
		t.Fatalf("EM014 summary = %q", several.Summary)
	}
}

func TestValidateSource_ReportsWorkflowInSeveralChaptersInOneFileModel(t *testing.T) {
	model := `state_change "first" { title = "First" }
chapter "one" {
  workflows = [workflow.first]
}
chapter "two" {
  workflows = [workflow.first]
}
`

	several := requireOneDiagnostic(t, validateModel(t, model), "EM014", hcl.DiagError, "model.em.hcl")
	if several.Subject.Start.Line != 6 {
		t.Fatalf("EM014 subject line = %d, want 6 (the later chapter)", several.Subject.Start.Line)
	}
}

func TestValidateSource_ReportsRepeatedChapterEntry(t *testing.T) {
	model := `state_change "first" { title = "First" }
chapter "one" {
  workflows = [workflow.first, workflow.first]
}
`

	diagnostics := validateModel(t, model)

	repeated := requireOneDiagnostic(t, diagnostics, "EM002", hcl.DiagError, "model.em.hcl")
	if want := "workflow.first is listed more than once in chapter.one."; repeated.Detail != want {
		t.Fatalf("EM002 detail = %q, want %q", repeated.Detail, want)
	}
	if repeated.Subject.Start.Column != 32 {
		t.Fatalf("EM002 subject column = %d, want 32 (the repeated entry)", repeated.Subject.Start.Column)
	}
	requireNoDiagnosticWithCode(t, diagnostics, "EM014")
}

func TestValidateFiles_UnchapteredWorkflowsHaveNoDiagnosticInAnyProfile(t *testing.T) {
	files := []syntax.File{
		modelFile("a.em.hcl", `state_change "first" { title = "First" }
chapter "one" {
  workflows = [workflow.first]
}
`),
		modelFile("b.em.hcl", "state_view \"second\" { title = \"Second\" }\n"),
	}
	for _, profile := range []Profile{Workshop, Valid, Strict} {
		t.Run(profile.String(), func(t *testing.T) {
			if diagnostics := validateFiles(t, profile, files...); len(diagnostics) != 0 {
				t.Fatalf("diagnostics = %s, want none", diagnostics.Error())
			}
		})
	}
}

func TestValidateFiles_UnresolvedChapterReferenceDoesNotChapterAWorkflow(t *testing.T) {
	diagnostics := validateFiles(t, Valid,
		modelFile("a.em.hcl", `state_change "first" { title = "First" }
chapter "one" {
  workflows = [workflow.missing, workflow.first]
}
chapter "two" {
  workflows = [workflow.missing]
}
`),
		modelFile("b.em.hcl", "state_change \"second\" { title = \"Second\" }\n"),
	)

	if matches := diagnosticsWithCode(diagnostics, "EM102"); len(matches) != 2 {
		t.Fatalf("EM102 diagnostics = %#v, want one per unresolved entry", matches)
	}
	requireNoDiagnosticWithCode(t, diagnostics, "EM014")
}
