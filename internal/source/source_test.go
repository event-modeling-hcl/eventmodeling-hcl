package source_test

import (
	"testing"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/source"
	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestDecode_IndexesFieldTypesByNameAndContext(t *testing.T) {
	document, diagnostics := syntax.Parse("model.em.hcl", []byte(`
bounded_context "clinic" {
  field_type "pet_id" { type = "UUID" }
}
bounded_context "shelter" {
  field_type "pet_id" { type = "UUID" }
  field_type "shelter_id" { type = "UUID" }
}
`))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics = %s", diagnostics.Error())
	}

	decoded := source.Decode(document)

	assertStrings(t, decoded.FieldTypeContexts("pet_id"), []string{"clinic", "shelter"})
	assertStrings(t, decoded.FieldTypeContexts("shelter_id"), []string{"shelter"})
}

func TestDecode_PreservesParsedDocument(t *testing.T) {
	parsed, diagnostics := syntax.Parse("model.em.hcl", []byte(`actor "staff" { auth_required = true }`))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics = %s", diagnostics.Error())
	}

	decoded := source.Decode(parsed)

	if decoded.Parsed() != parsed {
		t.Fatal("Decoded document did not preserve the parsed document")
	}
}

func TestDocument_InferFieldTypeUsesContextOrUniqueDocumentDeclaration(t *testing.T) {
	parsed, diagnostics := syntax.Parse("model.em.hcl", []byte(`
bounded_context "clinic" {
  field_type "pet_id" { type = "UUID" }
}
bounded_context "shelter" {
  field_type "pet_id" { type = "UUID" }
  field_type "shelter_id" { type = "UUID" }
}
`))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics = %s", diagnostics.Error())
	}
	decoded := source.Decode(parsed)

	if got, want := decoded.InferFieldType("clinic", "pet_id"), "field_type.clinic.pet_id"; got != want {
		t.Fatalf("context-local inference = %q, want %q", got, want)
	}
	if got, want := decoded.InferFieldType("", "shelter_id"), "field_type.shelter.shelter_id"; got != want {
		t.Fatalf("unique inference = %q, want %q", got, want)
	}
	if got := decoded.InferFieldType("", "pet_id"); got != "" {
		t.Fatalf("ambiguous inference = %q, want empty", got)
	}
}

func TestCanonicalFieldTypeReference_QualifiesContextLocalReference(t *testing.T) {
	if got, want := source.CanonicalFieldTypeReference("clinic", "field_type.pet_id"), "field_type.clinic.pet_id"; got != want {
		t.Fatalf("canonical reference = %q, want %q", got, want)
	}
	if got, want := source.CanonicalFieldTypeReference("clinic", "field_type.shelter.pet_id"), "field_type.shelter.pet_id"; got != want {
		t.Fatalf("qualified reference = %q, want %q", got, want)
	}
}

func TestReferenceParts_DecodesAnAbsoluteTraversal(t *testing.T) {
	expression, diagnostics := hclsyntax.ParseExpression([]byte("event.clinic.pet_registered"), "model.em.hcl", hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		t.Fatalf("parse expression diagnostics = %s", diagnostics.Error())
	}

	parts, referenceDiagnostics := source.ReferenceParts(expression)

	if referenceDiagnostics.HasErrors() {
		t.Fatalf("reference diagnostics = %s", referenceDiagnostics.Error())
	}
	assertStrings(t, parts, []string{"event", "clinic", "pet_registered"})
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("values = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("values = %#v, want %#v", got, want)
		}
	}
}
