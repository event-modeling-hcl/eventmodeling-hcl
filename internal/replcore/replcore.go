// Package replcore is the shared, testable core behind the WebAssembly
// browser editor (cmd/wasm) and the local live-reload server
// (internal/serve): parse, validate, and render .em.hcl source held in
// memory, returning plain, JSON/JS-friendly types instead of *hcl.Diagnostic
// and its pointer-heavy neighbors. Neither caller needs to know how a
// diagnostic is represented internally, and both need the exact same
// behavior, so it delegates to internal/app — the same application service
// the CLI uses — rather than assembling the parse/validate/render pipeline
// itself.
package replcore

import (
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/app"
)

// Profile re-exports app.Profile so callers need only import replcore.
type Profile = app.Profile

// Profile values, re-exported from app for the same reason.
const (
	Workshop = app.Workshop
	Valid    = app.Valid
	Strict   = app.Strict
)

// ParseProfile converts a profile name ("workshop", "valid", "strict") into
// its typed Profile, so callers such as cmd/wasm and internal/serve need not
// import internal/validator themselves just to accept a profile as a plain
// string. Reports false, with Valid, for any other input.
func ParseProfile(value string) (Profile, bool) {
	return app.ParseProfile(value)
}

// Diagnostic is a plain, serialization-friendly rendering of an HCL
// diagnostic: no interfaces or pointers into HCL internals, so it marshals
// cleanly to JSON and to a JavaScript value from within WebAssembly.
type Diagnostic = app.Diagnostic

// RenderResult is the output of Render.
type RenderResult = app.RenderResult

// Render parses and validates source under profile and, if it contains no
// errors, renders it to a self-contained interactive HTML canvas. This
// mirrors the diagram command: a model with errors is refused and no HTML
// is produced; modeling-smell warnings are reported but do not block
// rendering.
func Render(filename string, source []byte, profile Profile) RenderResult {
	return app.Render(filename, source, profile)
}

// FormatResult is the output of Format.
type FormatResult = app.FormatResult

// Format canonicalizes source's whitespace and attribute order, the same
// transformation `eventmodeling-hcl fmt` applies. Formatting only requires
// source to parse as HCL; it does not run Event Modeling validation.
func Format(filename string, source []byte) FormatResult {
	return app.Format(filename, source)
}
