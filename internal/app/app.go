// Package app composes the eventmodeling-hcl pipeline — parse, validate,
// build, render/format — exactly once per operation, and defines the single
// plain Diagnostic type every entrypoint (CLI, WASM, serve) reports through.
// Each entrypoint used to assemble internal/syntax, internal/validator,
// internal/model, and internal/renderer/internal/formatter itself, which
// meant parsing a document twice per render (once to validate, once to
// build) and maintaining two separate hcl.Diagnostic-to-plain conversions
// that had to be kept in lockstep by hand. app exists so there is exactly
// one of each.
package app

import (
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/formatter"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/renderer"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

// Profile re-exports validator.Profile so callers need only import app.
type Profile = validator.Profile

// Profile values, re-exported from validator for the same reason.
const (
	Workshop = validator.Workshop
	Valid    = validator.Valid
	Strict   = validator.Strict
)

// ParseProfile converts a profile name ("workshop", "valid", "strict") into
// its typed Profile, so callers such as cmd/wasm and internal/serve need not
// import internal/validator themselves just to accept a profile as a plain
// string. Reports false, with Valid, for any other input.
func ParseProfile(value string) (Profile, bool) {
	return validator.ParseProfile(value)
}

// Diagnostic is a plain, serialization-friendly rendering of an HCL
// diagnostic: no interfaces or pointers into HCL internals, so it marshals
// cleanly to JSON and to a JavaScript value from within WebAssembly.
type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`

	// Filename is the source file the diagnostic points at, used by the CLI
	// to print "file:line:column: ..." text lines. It is not part of the
	// JSON/JS contract (json:"-"): callers that marshal Diagnostic to JSON
	// (or, in cmd/wasm, build the JS object field-by-field) never see it,
	// so adding it here changes no observable output.
	Filename string `json:"-"`
}

// RenderResult is the output of Render.
type RenderResult struct {
	// HTML is the self-contained interactive canvas document — the same
	// bytes `eventmodeling-hcl diagram` would write to a file — or empty
	// when Diagnostics contains an error.
	HTML        string       `json:"html"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Render parses and validates source under profile and, if it contains no
// errors, renders it to a self-contained interactive HTML canvas. This
// mirrors the diagram command: a model with errors is refused and no HTML
// is produced; modeling-smell warnings are reported but do not block
// rendering. source is parsed exactly once, whether or not it validates.
func Render(filename string, source []byte, profile Profile) RenderResult {
	doc, parseDiagnostics := syntax.Parse(filename, source)
	if parseDiagnostics.HasErrors() {
		return RenderResult{Diagnostics: toDiagnostics(parseDiagnostics)}
	}

	diagnostics := validator.ValidateDocument(doc, profile)
	result := RenderResult{Diagnostics: toDiagnostics(diagnostics)}
	if diagnostics.HasErrors() {
		return result
	}

	built := model.Build(doc)
	html, err := renderer.Render(filename, built)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "Error",
			Summary:  "Failed to render diagram",
			Detail:   err.Error(),
			Filename: filename,
		})
		return result
	}
	result.HTML = html
	return result
}

// FormatResult is the output of Format.
type FormatResult struct {
	// Source is the canonicalized document, or empty when Diagnostics
	// contains an error.
	Source      string       `json:"source"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Format canonicalizes source's whitespace and attribute order, the same
// transformation `eventmodeling-hcl fmt` applies. Formatting only requires
// source to parse as HCL; it does not run Event Modeling validation.
func Format(filename string, source []byte) FormatResult {
	formatted, diagnostics := formatter.Format(filename, source)
	return FormatResult{
		Source:      string(formatted),
		Diagnostics: toDiagnostics(diagnostics),
	}
}

// Validate parses and validates source under profile, returning its
// diagnostics as the plain Diagnostic type. source is parsed exactly once.
func Validate(filename string, source []byte, profile Profile) []Diagnostic {
	doc, parseDiagnostics := syntax.Parse(filename, source)
	if parseDiagnostics.HasErrors() {
		return toDiagnostics(parseDiagnostics)
	}
	return toDiagnostics(validator.ValidateDocument(doc, profile))
}

// toDiagnostics converts hcl.Diagnostics to the plain Diagnostic slice both
// public functions return. A nil or empty input converts to nil, so JSON
// marshaling produces `"diagnostics":[]` only when there is content to
// report — never a spurious null the JS/WASM side would need to guard
// against separately from the empty-array case.
func toDiagnostics(diagnostics hcl.Diagnostics) []Diagnostic {
	if len(diagnostics) == 0 {
		return nil
	}
	converted := make([]Diagnostic, len(diagnostics))
	for index, diagnostic := range diagnostics {
		converted[index] = toDiagnostic(diagnostic)
	}
	return converted
}

// toDiagnostic converts a single *hcl.Diagnostic, extracting the stable
// EMxxx code and the source position the same way the CLI's own
// formatDiagnostic used to (cmd/eventmodeling-hcl/main.go), so a diagnostic
// looks identical whether it reached you via the terminal, the browser
// editor, or `serve`.
func toDiagnostic(diagnostic *hcl.Diagnostic) Diagnostic {
	severity := ""
	switch diagnostic.Severity {
	case hcl.DiagInvalid:
		severity = "Info"
	case hcl.DiagError:
		severity = "Error"
	case hcl.DiagWarning:
		severity = "Warning"
	}

	converted := Diagnostic{
		Code:     validator.DiagnosticCode(diagnostic),
		Severity: severity,
		Summary:  diagnostic.Summary,
		Detail:   diagnostic.Detail,
	}
	if diagnostic.Subject != nil {
		converted.Line = diagnostic.Subject.Start.Line
		converted.Column = diagnostic.Subject.Start.Column
		converted.Filename = diagnostic.Subject.Filename
	}
	return converted
}
