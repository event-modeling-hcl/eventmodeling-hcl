package renderer

import (
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	sourcepkg "github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

// loadTestModel follows the production control sequence to give renderer tests
// a valid canonical model without introducing file I/O into the renderer.
func loadTestModel(t *testing.T, filename string, source []byte) (*model.Model, hcl.Diagnostics) {
	t.Helper()
	doc, diagnostics := syntax.Parse(filename, source)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	validated, validationDiagnostics := validator.ValidateDecodedDocument(sourcepkg.Decode(doc), validator.Valid)
	if validationDiagnostics.HasErrors() {
		return nil, validationDiagnostics
	}
	return model.Build(validated), validationDiagnostics
}
