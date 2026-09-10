package renderer

import (
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/model"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/validator"
	"github.com/hashicorp/hcl/v2"
)

// loadTestModel parses, validates, and builds source the same way the
// deleted model.Load used to. Production code (internal/app) composes these
// steps once; this package's tests only need a valid *model.Model to render,
// so they compose the same three steps locally instead of reaching for a
// helper that no longer exists in model's public API.
func loadTestModel(t *testing.T, filename string, source []byte) (*model.Model, hcl.Diagnostics) {
	t.Helper()
	doc, diagnostics := syntax.Parse(filename, source)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	validationDiagnostics := validator.ValidateDocument(doc, validator.Valid)
	if validationDiagnostics.HasErrors() {
		return nil, validationDiagnostics
	}
	return model.Build(doc), validationDiagnostics
}
