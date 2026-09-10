// Package syntax owns the HCL grammar for the native Event Modeling
// language: the hcl.BodySchema definitions and the single parse of a
// document's source. These tests lock in grammar acceptance, diagnostics on
// violations, and range fidelity of the decoded content.
package syntax_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

func TestParse_AcceptsValidDocument(t *testing.T) {
	source := `actor "clinic_staff" {
  title         = "Clinic staff"
  auth_required = true
}
`
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(source))
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics = %s, want no errors", diagnostics.Error())
	}
	if document == nil {
		t.Fatal("document = nil, want non-nil Document on a valid parse")
	}
}

func TestParse_ReturnsDiagnosticsOnMalformedSyntax(t *testing.T) {
	source := `actor "clinic_staff" {
  title = "Clinic staff"
`
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(source))
	if !diagnostics.HasErrors() {
		t.Fatal("diagnostics has no errors, want a syntax error for the unclosed block")
	}
	if document != nil {
		t.Fatalf("document = %#v, want nil Document on a syntax error", document)
	}
}

func TestContent_ModelSchemaRejectsUnknownTopLevelBlock(t *testing.T) {
	source := `bogus_block "x" {}`
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(source))
	if diagnostics.HasErrors() {
		t.Fatalf("unexpected parse diagnostics: %s", diagnostics.Error())
	}
	_, contentDiagnostics := syntax.Content(document.Body(), syntax.ModelSchema())
	if !contentDiagnostics.HasErrors() {
		t.Fatal("content diagnostics has no errors, want an unsupported-block-type diagnostic")
	}
	found := false
	for _, diagnostic := range contentDiagnostics {
		if diagnostic.Summary == "Unsupported block type" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want an \"Unsupported block type\" diagnostic", contentDiagnostics)
	}
}

func TestContent_ActorSchemaRequiresAuthRequired(t *testing.T) {
	source := `actor "clinic_staff" {
  title = "Clinic staff"
}
`
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(source))
	if diagnostics.HasErrors() {
		t.Fatalf("unexpected parse diagnostics: %s", diagnostics.Error())
	}
	content, modelDiagnostics := syntax.Content(document.Body(), syntax.ModelSchema())
	if modelDiagnostics.HasErrors() {
		t.Fatalf("unexpected model diagnostics: %s", modelDiagnostics.Error())
	}
	if len(content.Blocks) != 1 {
		t.Fatalf("len(content.Blocks) = %d, want 1", len(content.Blocks))
	}
	_, actorDiagnostics := syntax.Content(content.Blocks[0].Body, syntax.ActorSchema())
	if !actorDiagnostics.HasErrors() {
		t.Fatal("actor diagnostics has no errors, want a missing-required-argument diagnostic for auth_required")
	}
	found := false
	for _, diagnostic := range actorDiagnostics {
		if diagnostic.Summary == "Missing required argument" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want a \"Missing required argument\" diagnostic", actorDiagnostics)
	}
}

func TestPartialContent_IgnoresAttributesOutsideSchema(t *testing.T) {
	source := `hotspot "open_question" {
  question    = "First?"
  unsupported = "ignored by PartialContent"
}
`
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(source))
	if diagnostics.HasErrors() {
		t.Fatalf("unexpected parse diagnostics: %s", diagnostics.Error())
	}
	content, _, modelDiagnostics := syntax.PartialContent(document.Body(), syntax.ModelSchema())
	if modelDiagnostics.HasErrors() {
		t.Fatalf("unexpected model diagnostics: %s", modelDiagnostics.Error())
	}
	if len(content.Blocks) != 1 {
		t.Fatalf("len(content.Blocks) = %d, want 1", len(content.Blocks))
	}
	partialContent, _, hotspotDiagnostics := syntax.PartialContent(content.Blocks[0].Body, syntax.HotspotSchema())
	if hotspotDiagnostics.HasErrors() {
		t.Fatalf("PartialContent reported diagnostics for an attribute outside its schema: %s", hotspotDiagnostics.Error())
	}
	if partialContent.Attributes["question"] == nil {
		t.Fatal("partialContent.Attributes[\"question\"] = nil, want the known attribute to still decode")
	}
	if _, unsupported := partialContent.Attributes["unsupported"]; unsupported {
		t.Fatal("partialContent.Attributes contains \"unsupported\", want only schema-known attributes")
	}
}

func TestContent_PreservesSourceRanges(t *testing.T) {
	source := `bounded_context "pet_management" {
  title = "Pet Management"
}
`
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(source))
	if diagnostics.HasErrors() {
		t.Fatalf("unexpected parse diagnostics: %s", diagnostics.Error())
	}
	content, modelDiagnostics := syntax.Content(document.Body(), syntax.ModelSchema())
	if modelDiagnostics.HasErrors() {
		t.Fatalf("unexpected model diagnostics: %s", modelDiagnostics.Error())
	}
	if len(content.Blocks) != 1 {
		t.Fatalf("len(content.Blocks) = %d, want 1", len(content.Blocks))
	}
	block := content.Blocks[0]
	wantBlockStart := hcl.Pos{Line: 1, Column: 1}
	if block.DefRange.Start != wantBlockStart {
		t.Fatalf("block.DefRange.Start = %#v, want %#v", block.DefRange.Start, wantBlockStart)
	}

	contextContent, contextDiagnostics := syntax.Content(block.Body, syntax.BoundedContextSchema())
	if contextDiagnostics.HasErrors() {
		t.Fatalf("unexpected bounded_context diagnostics: %s", contextDiagnostics.Error())
	}
	title := contextContent.Attributes["title"]
	if title == nil {
		t.Fatal("contextContent.Attributes[\"title\"] = nil, want the title attribute to decode")
	}
	wantTitleLine := 2
	if title.Expr.Range().Start.Line != wantTitleLine {
		t.Fatalf("title.Expr.Range().Start.Line = %d, want %d", title.Expr.Range().Start.Line, wantTitleLine)
	}
	wantNameRangeColumn := 3
	if title.NameRange.Start.Column != wantNameRangeColumn {
		t.Fatalf("title.NameRange.Start.Column = %d, want %d", title.NameRange.Start.Column, wantNameRangeColumn)
	}
}
