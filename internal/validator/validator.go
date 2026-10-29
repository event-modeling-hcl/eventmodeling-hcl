// Package validator implements the strict validator for the native HCL Event
// Modeling Specification defined in eventmodeling.hclspec.md.
package validator

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

// ValidatedDocument is the capability required to construct the canonical
// model. Its source cannot be supplied without first passing validation.
type ValidatedDocument struct {
	source *source.Document
}

// Source returns the decoded source document that passed validation.
func (d *ValidatedDocument) Source() *source.Document {
	return d.source
}

// ValidateSource parses and validates one in-memory Event Modeling document.
func ValidateSource(filename string, source []byte) hcl.Diagnostics {
	return ValidateSourceWithProfile(filename, source, Valid)
}

// ValidateSourceWithProfile parses and validates one in-memory Event Modeling
// document using the requested validation profile.
func ValidateSourceWithProfile(filename string, source []byte, profile Profile) hcl.Diagnostics {
	document, diagnostics := syntax.Parse(filename, source)
	if diagnostics.HasErrors() {
		return applyProfile(diagnostics, profile)
	}
	return ValidateDocument(document, profile)
}

// ValidateDocument validates an already-parsed Document using the requested
// validation profile. Callers that need to both validate and further process
// a document (for example, building it into internal/model's canonical
// Model) can parse once with syntax.Parse and pass the result here, instead
// of parsing again through ValidateSourceWithProfile.
func ValidateDocument(doc *syntax.Document, profile Profile) hcl.Diagnostics {
	_, diagnostics := ValidateDecodedDocument(source.Decode(doc), profile)
	return diagnostics
}

// ValidateDecodedDocument validates decoded source. It returns a construction
// capability iff the core rules report no error. The profile only assigns the
// severity of judgment signals; it never decides whether the capability
// exists, so a policy rejection still yields a ValidatedDocument.
func ValidateDecodedDocument(doc *source.Document, profile Profile) (*ValidatedDocument, hcl.Diagnostics) {
	diagnostics := validateBody(doc)
	var validated *ValidatedDocument
	if !hasCoreErrors(diagnostics) {
		validated = &ValidatedDocument{source: doc}
	}
	return validated, applyProfile(diagnostics, profile)
}

type fieldTypeRef struct {
	fieldType   string
	cardinality string
}

type workflowIndex struct {
	elements map[string]map[string]bool
}

type modelIndex struct {
	actors           map[string]bool
	aggregates       map[string]bool
	boundedContexts  map[string]bool
	catalogEvents    map[string]bool
	fieldTypes       map[string]fieldTypeRef
	chapters         map[string]bool
	hotspots         map[string]bool
	systems          map[string]bool
	teams            map[string]bool
	workflows        map[string]workflowIndex
	scenarioCounts   map[string]int
	definitionRanges map[string]hcl.Range
	declarations     map[string]hcl.Range
}

type modelValidator struct {
	index modelIndex
	// chaptered maps each workflow listed by a chapter to that chapter's id.
	chaptered map[string]string
}

type contextValidator struct {
	model     *modelValidator
	contextID string
}

// validateBody runs every validation pass over a parsed document body: it
// builds the symbol index (index.go), then the structural/block-eligibility
// checks per top-level block (structure.go, which in turn covers scenarios
// in scenarios.go and per-workflow smells in smells.go), the one-board rule,
// and finally the whole-model shelf-anti-pattern smell check.
func validateBody(document *source.Document) hcl.Diagnostics {
	index, diagnostics := buildIndex(document)
	validator := &modelValidator{index: index, chaptered: map[string]string{}}
	content, contentDiagnostics := syntax.Content(document.Parsed().Body(), syntax.ModelSchema())
	diagnostics = append(diagnostics, contentDiagnostics...)
	var boards hcl.Blocks
	for _, block := range content.Blocks {
		switch block.Type {
		case "board":
			boards = append(boards, block)
			diagnostics = append(diagnostics, validator.validateBoard(block)...)
		case "bounded_context":
			diagnostics = append(diagnostics, validator.validateBoundedContext(block)...)
		case "actor":
			diagnostics = append(diagnostics, validator.validateActor(block)...)
		case "team", "system":
			diagnostics = append(diagnostics, validator.validateOwner(block)...)
		case "chapter":
			diagnostics = append(diagnostics, validator.validateChapter(block)...)
		case "hotspot":
			diagnostics = append(diagnostics, validator.validateHotspot(block)...)
		case "state_change", "state_view", "automation":
			diagnostics = append(diagnostics, validator.validateWorkflow(block)...)
		}
	}
	diagnostics = append(diagnostics, boardCountDiagnostics(document.Parsed().Start(), boards)...)
	diagnostics = append(diagnostics, validator.validateShelfSmell()...)
	return diagnostics
}

// boardCountDiagnostics enforces exactly one board block. A missing board is
// reported at start, the start of the first member file; each repeated board
// is reported at its own definition.
func boardCountDiagnostics(start hcl.Range, boards hcl.Blocks) hcl.Diagnostics {
	if len(boards) == 0 {
		return hcl.Diagnostics{errorDiagnostic(codeInvalidBoard, start, "Missing board", "a model must declare exactly one board block.")}
	}
	var diagnostics hcl.Diagnostics
	first := boards[0].DefRange
	for _, board := range boards[1:] {
		diagnostics = append(diagnostics, errorDiagnostic(codeInvalidBoard, board.DefRange, "Repeated board", fmt.Sprintf("a model must declare exactly one board block; first declared at %s:%d:%d.", first.Filename, first.Start.Line, first.Start.Column)))
	}
	return diagnostics
}

func literalString(attribute *hcl.Attribute) (string, bool) {
	if attribute == nil {
		return "", false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !value.Type().Equals(cty.String) {
		return "", false
	}
	return value.AsString(), true
}

func literalBool(attribute *hcl.Attribute) (bool, bool) {
	if attribute == nil {
		return false, false
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if diagnostics.HasErrors() || value.IsNull() || !value.Type().Equals(cty.Bool) {
		return false, false
	}
	return value.True(), true
}
