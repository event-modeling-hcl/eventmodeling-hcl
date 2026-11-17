package validator

import (
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// storylineChapters declares three workflows and one chapter for each, for
// the storyline tests.
const storylineChapters = `state_change "first" { title = "First" }
state_change "second" { title = "Second" }
state_view "third" { title = "Third" }
chapter "one" {
  workflows = [workflow.first]
}
chapter "two" {
  workflows = [workflow.second]
}
chapter "three" {
  workflows = [workflow.third]
}
`

func TestValidateFiles_AcceptsStorylinesInSeveralFiles(t *testing.T) {
	files := []syntax.File{
		modelFile("a.em.hcl", storylineChapters),
		modelFile("b.em.hcl", `storyline "intake" {
  title       = "Intake"
  description = "The pet arrives."
  chapters    = [chapter.two, chapter.one]
}
`),
		modelFile("c.em.hcl", `storyline "care" {
  chapters = [chapter.three]
}
`),
	}
	for _, profile := range []Profile{Workshop, Valid, Strict} {
		t.Run(profile.String(), func(t *testing.T) {
			if diagnostics := validateFiles(t, profile, files...); len(diagnostics) != 0 {
				t.Fatalf("diagnostics = %s, want none", diagnostics.Error())
			}
		})
	}
}

func TestValidateFiles_ChaptersInNoStorylineHaveNoDiagnosticInAnyProfile(t *testing.T) {
	files := []syntax.File{
		modelFile("a.em.hcl", storylineChapters),
		modelFile("b.em.hcl", `storyline "intake" {
  chapters = [chapter.one]
}
`),
	}
	for _, profile := range []Profile{Workshop, Valid, Strict} {
		t.Run(profile.String(), func(t *testing.T) {
			if diagnostics := validateFiles(t, profile, files...); len(diagnostics) != 0 {
				t.Fatalf("diagnostics = %s, want none", diagnostics.Error())
			}
		})
	}
}

func TestValidateSource_ReportsInvalidStorylines(t *testing.T) {
	// storylineChapters has 12 lines, so each storyline starts on line 13.
	tests := []struct {
		name      string
		storyline string
		code      string
		summary   string
		detail    string
		line      int
		column    int
	}{
		{
			name:      "empty chapters",
			storyline: "storyline \"intake\" {\n  chapters = []\n}\n",
			code:      "EM005", summary: "Invalid storyline", detail: "storyline must contain at least one chapter.",
			line: 14, column: 14,
		},
		{
			name:      "repeated chapter",
			storyline: "storyline \"intake\" {\n  chapters = [chapter.one, chapter.one]\n}\n",
			code:      "EM002", summary: "Duplicate storyline chapter", detail: "chapter.one is listed more than once in storyline.intake.",
			line: 14, column: 28,
		},
		{
			name:      "chapter in two storylines",
			storyline: "storyline \"intake\" {\n  chapters = [chapter.one]\n}\nstoryline \"care\" {\n  chapters = [chapter.two, chapter.one]\n}\n",
			code:      "EM019", summary: "Chapter in several storylines", detail: "chapter.one is already in storyline.intake.",
			line: 17, column: 28,
		},
		{
			name:      "wrong kind",
			storyline: "storyline \"intake\" {\n  chapters = [workflow.first]\n}\n",
			code:      "EM101", summary: "Invalid reference", detail: "reference must target chapter.<id>.",
			line: 14, column: 15,
		},
		{
			name:      "unknown chapter",
			storyline: "storyline \"intake\" {\n  chapters = [chapter.missing]\n}\n",
			code:      "EM102", summary: "Unresolved reference", detail: "chapter.missing is not declared in the model.",
			line: 14, column: 15,
		},
		{
			name:      "duplicate storyline",
			storyline: "storyline \"intake\" {\n  chapters = [chapter.one]\n}\nstoryline \"intake\" {\n  chapters = [chapter.two]\n}\n",
			code:      "EM002", summary: "Duplicate storyline id", detail: `storyline "intake" is declared more than once in its namespace. First declared at model.em.hcl:13:1.`,
			line: 16, column: 1,
		},
		{
			name:      "label",
			storyline: "storyline \"Intake\" {\n  chapters = [chapter.one]\n}\n",
			code:      "EM003",
			line:      13, column: 1,
		},
		{
			name:      "title kind",
			storyline: "storyline \"intake\" {\n  title    = 3\n  chapters = [chapter.one]\n}\n",
			code:      "EM007", summary: "Invalid attribute value", detail: "title must be a string.",
			line: 14, column: 14,
		},
		{
			name:      "missing chapters",
			storyline: "storyline \"intake\" {\n  title = \"Intake\"\n}\n",
			code:      "EM000", summary: "Missing required argument",
			line: 13, column: 20,
		},
		{
			name:      "unknown attribute",
			storyline: "storyline \"intake\" {\n  chapters  = [chapter.one]\n  workflows = [workflow.first]\n}\n",
			code:      "EM000", summary: "Unsupported argument",
			line: 15, column: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := validateModel(t, storylineChapters+test.storyline)

			found := requireOneDiagnostic(t, diagnostics, test.code, hcl.DiagError, "model.em.hcl")
			if test.summary != "" && found.Summary != test.summary {
				t.Fatalf("%s summary = %q, want %q", test.code, found.Summary, test.summary)
			}
			if test.detail != "" && found.Detail != test.detail {
				t.Fatalf("%s detail = %q, want %q", test.code, found.Detail, test.detail)
			}
			if start := found.Subject.Start; start.Line != test.line || start.Column != test.column {
				t.Fatalf("%s subject = %d:%d, want %d:%d", test.code, start.Line, start.Column, test.line, test.column)
			}
		})
	}
}

func TestValidateFiles_ReportsChapterInSeveralStorylinesAcrossFiles(t *testing.T) {
	diagnostics := validateFiles(t, Valid,
		modelFile("a.em.hcl", storylineChapters+`storyline "intake" {
  chapters = [chapter.one]
}
`),
		modelFile("b.em.hcl", `storyline "care" {
  chapters = [chapter.one]
}
`),
	)

	several := requireOneDiagnostic(t, diagnostics, "EM019", hcl.DiagError, "b.em.hcl")
	if want := "chapter.one is already in storyline.intake."; several.Detail != want {
		t.Fatalf("EM019 detail = %q, want %q", several.Detail, want)
	}
}

func TestValidateSource_UnresolvedStorylineEntryDoesNotPlaceAChapter(t *testing.T) {
	diagnostics := validateModel(t, storylineChapters+`storyline "intake" {
  chapters = [chapter.missing, chapter.one]
}
storyline "care" {
  chapters = [chapter.missing]
}
`)

	if matches := diagnosticsWithCode(diagnostics, "EM102"); len(matches) != 2 {
		t.Fatalf("EM102 diagnostics = %#v, want one per unresolved entry", matches)
	}
	requireNoDiagnosticWithCode(t, diagnostics, "EM019")
}
