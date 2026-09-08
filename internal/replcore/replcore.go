// Package replcore is the shared, testable core behind the WebAssembly
// browser editor (cmd/wasm) and the local live-reload server
// (internal/serve): parse, validate, and render .em.hcl source held in
// memory, returning plain, JSON/JS-friendly types instead of *hcl.Diagnostic
// and its pointer-heavy neighbors. Neither caller needs to know how a
// diagnostic is represented internally, and both need the exact same
// behavior, so it lives here once rather than being duplicated or
// reimplemented per entrypoint.
package replcore

import (
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/formatter"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/renderer"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

// Profile re-exports model.Profile so callers need only import replcore.
type Profile = model.Profile

// Profile values, re-exported from model for the same reason.
const (
	Workshop = model.Workshop
	Valid    = model.Valid
	Strict   = model.Strict
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
// rendering.
func Render(filename string, source []byte, profile Profile) RenderResult {
	loaded, diagnostics := model.Load(filename, source, profile)
	result := RenderResult{Diagnostics: toDiagnostics(diagnostics)}
	if diagnostics.HasErrors() {
		return result
	}

	html, err := renderer.Render(filename, loaded)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "Error",
			Summary:  "Failed to render diagram",
			Detail:   err.Error(),
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
// formatDiagnostic does (cmd/eventmodeling-hcl/main.go), so a diagnostic
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
	}
	return converted
}
