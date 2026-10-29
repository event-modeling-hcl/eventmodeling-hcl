// Package source owns facts decoded once from a parsed Event Modeling HCL
// document that are shared by validation and canonical model construction.
package source

import (
	"strings"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
	"github.com/hashicorp/hcl/v2"
)

// Document is the source-order-preserving representation shared by the
// semantic stages of the language pipeline.
type Document struct {
	parsed *syntax.Document
}

// Decode wraps a parsed document for the semantic stages without applying
// semantic validation. The parsed HCL document remains the single owner of
// source ranges.
func Decode(parsed *syntax.Document) *Document {
	return &Document{parsed: parsed}
}

// Parsed returns the immutable parsed HCL document underlying this source
// document.
func (d *Document) Parsed() *syntax.Document {
	return d.parsed
}

// CanonicalFieldTypeReference expands a context-local field_type reference.
func CanonicalFieldTypeReference(contextID, reference string) string {
	parts := strings.Split(reference, ".")
	if contextID != "" && len(parts) == 2 && parts[0] == "field_type" {
		return "field_type." + contextID + "." + parts[1]
	}
	return reference
}

// ReferenceParts decodes an absolute HCL traversal into its address segments.
func ReferenceParts(expression hcl.Expression) ([]string, hcl.Diagnostics) {
	traversal, diagnostics := AbsoluteTraversal(expression)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	parts, ok := TraversalParts(traversal)
	if !ok {
		return nil, diagnostics
	}
	return parts, diagnostics
}

// AbsoluteTraversal decodes expression as an absolute HCL traversal.
func AbsoluteTraversal(expression hcl.Expression) (hcl.Traversal, hcl.Diagnostics) {
	return hcl.AbsTraversalForExpr(expression)
}

// TraversalParts converts an absolute traversal to plain address segments.
func TraversalParts(traversal hcl.Traversal) ([]string, bool) {
	if len(traversal) == 0 {
		return nil, false
	}
	parts := []string{traversal.RootName()}
	for _, step := range traversal[1:] {
		attribute, ok := step.(hcl.TraverseAttr)
		if !ok {
			return nil, false
		}
		parts = append(parts, attribute.Name)
	}
	return parts, true
}

// ReferenceString returns the canonical dotted spelling of an absolute
// traversal, or an empty string for an invalid expression.
func ReferenceString(expression hcl.Expression) string {
	parts, diagnostics := ReferenceParts(expression)
	if diagnostics.HasErrors() || len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ".")
}

// ReferenceList returns canonical dotted spellings for a traversal list.
func ReferenceList(expression hcl.Expression) []string {
	expressions, diagnostics := hcl.ExprList(expression)
	if diagnostics.HasErrors() {
		return nil
	}
	values := make([]string, 0, len(expressions))
	for _, item := range expressions {
		if value := ReferenceString(item); value != "" {
			values = append(values, value)
		}
	}
	return values
}
