// Package validator implements the strict validator for the native HCL v1
// Event Modeling Specification defined in eventmodeling.hclspec.md.
package validator

import (
	"fmt"
	"math/big"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// ValidateFile parses and validates one native HCL Event Modeling document.
func ValidateFile(path string) hcl.Diagnostics {
	source, err := os.ReadFile(path)
	if err != nil {
		return hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to read file",
			Detail:   fmt.Sprintf("The configuration file %q could not be read.", path),
		}}
	}
	return ValidateSource(path, source)
}

// ValidateSource parses and validates HCL source. It is primarily useful for
// callers that already have a model in memory.
func ValidateSource(filename string, source []byte) hcl.Diagnostics {
	parser := hclparse.NewParser()
	file, diagnostics := parser.ParseHCL(source, filename)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	return validateBody(file.Body)
}

// The functions below build the HCL schema for every block kind in the
// grammar. Each schema lists exactly the attributes and child block types
// that block is allowed to have; anything else in the source document is
// rejected once it is checked with hcl.Body.Content (see validateBody and
// its callees further down this file). The schemas mirror
// eventmodeling.hclspec.md one for one, so that document is the place to
// look when a block's allowed shape needs to change.

// modelSchema describes the top-level document: a sequence of "slice"
// blocks and nothing else.
func modelSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "slice", LabelNames: []string{"id"}}},
	}
}

// sliceSchema describes a "slice" block: its own attributes, plus every
// kind of block that is allowed to appear directly inside it.
func sliceSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "title", Required: true},
			{Name: "status"},
			{Name: "index"},
			{Name: "context"},
			{Name: "slice_type", Required: true},
			{Name: "aggregates"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "command", LabelNames: []string{"id"}},
			{Type: "event", LabelNames: []string{"id"}},
			{Type: "readmodel", LabelNames: []string{"id"}},
			{Type: "screen", LabelNames: []string{"id"}},
			{Type: "screen_image", LabelNames: []string{"id"}},
			{Type: "processor", LabelNames: []string{"id"}},
			{Type: "table", LabelNames: []string{"id"}},
			{Type: "specification", LabelNames: []string{"id"}},
			{Type: "actor", LabelNames: []string{"name"}},
		},
	}
}

// elementSchema describes the shared shape of every "element" block kind
// (command, event, readmodel, screen, processor): the same attributes and
// the same two child block types (field, dependency) apply to all five.
func elementSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "group_id"},
			{Name: "tags"},
			{Name: "domain"},
			{Name: "model_context"},
			{Name: "context"},
			{Name: "slice"},
			{Name: "title", Required: true},
			{Name: "type", Required: true},
			{Name: "description"},
			{Name: "aggregate"},
			{Name: "aggregate_dependencies"},
			{Name: "api_endpoint"},
			{Name: "service"},
			{Name: "creates_aggregate"},
			{Name: "triggers"},
			{Name: "sketched"},
			{Name: "prototype"},
			{Name: "list_element"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "field", LabelNames: []string{"name"}},
			{Type: "dependency", LabelNames: []string{"id"}},
		},
	}
}

// screenImageSchema describes a "screen_image" block: title and an
// optional URL, with no child blocks.
func screenImageSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "title", Required: true},
			{Name: "url"},
		},
	}
}

// tableSchema describes a "table" block: a title plus any number of
// "field" children.
func tableSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "title", Required: true}},
		Blocks:     []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}},
	}
}

// specificationSchema describes a "specification" block: its own
// attributes, plus the "given"/"when"/"then" steps and optional "comment"
// block that make up the specification body.
func specificationSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "vertical"},
			{Name: "title", Required: true},
			{Name: "slice_name"},
			{Name: "linked_id", Required: true},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "given", LabelNames: []string{"id"}},
			{Type: "when", LabelNames: []string{"id"}},
			{Type: "then", LabelNames: []string{"id"}},
			{Type: "comment"},
		},
	}
}

// actorSchema describes an "actor" block: just the auth_required flag,
// with no child blocks.
func actorSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "auth_required", Required: true}},
	}
}

// fieldSchema describes a "field" block (and, recursively, a "subfield"
// block, which shares the same shape): its attributes, plus any number of
// nested "subfield" children.
func fieldSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "type", Required: true},
			{Name: "example"},
			{Name: "mapping"},
			{Name: "optional"},
			{Name: "technical_attribute"},
			{Name: "generated"},
			{Name: "id_attribute"},
			{Name: "pii"},
			{Name: "schema"},
			{Name: "cardinality"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "subfield", LabelNames: []string{"name"}}},
	}
}

// dependencySchema describes a "dependency" block: its three required
// attributes, with no child blocks.
func dependencySchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "type", Required: true},
			{Name: "title", Required: true},
			{Name: "element_type", Required: true},
		},
	}
}

// specificationStepSchema describes a "given"/"when"/"then" step block:
// its attributes, plus any number of "field" children.
func specificationStepSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "title", Required: true},
			{Name: "tags"},
			{Name: "examples"},
			{Name: "index"},
			{Name: "spec_row"},
			{Name: "type", Required: true},
			{Name: "linked_id"},
			{Name: "expect_empty_list"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}},
	}
}

// commentSchema describes a "comment" block: a single required
// description, with no child blocks.
func commentSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "description", Required: true}},
	}
}

// attributeRule is the validation rule for one HCL attribute: the kind of
// literal value it must hold, plus the closed set of allowed string values
// when the attribute is an enum. An empty enum means any value of the
// given kind is accepted.
type attributeRule struct {
	kind valueKind
	enum []string
}

// attributeRuleLookup finds the attributeRule for one attribute name,
// reporting false when the name is not a recognized attribute for the
// block being validated. Each *AttributeRule function below implements
// this signature.
type attributeRuleLookup func(string) (attributeRule, bool)

// valueKind names the shape an HCL attribute's literal value must have,
// independent of any specific enum values it might also be restricted to.
type valueKind uint8

const (
	anyValue valueKind = iota
	stringValue
	stringOrNullValue
	boolValue
	integerValue
	stringListValue
	objectValue
	objectListValue
)

// The functions below are the attributeRuleLookup for each block kind:
// given an attribute name that has already been confirmed to exist on the
// block (see validateAttributes), they say what kind of value it must hold
// and, for enum attributes, which string values are allowed.

// sliceAttributeRule is the attributeRuleLookup for "slice" block attributes.
func sliceAttributeRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "context":
		return attributeRule{kind: stringValue}, true
	case "status":
		return attributeRule{kind: stringValue, enum: []string{"Created", "Done", "InProgress"}}, true
	case "index":
		return attributeRule{kind: integerValue}, true
	case "slice_type":
		return attributeRule{kind: stringValue, enum: []string{"STATE_CHANGE", "STATE_VIEW", "AUTOMATION"}}, true
	case "aggregates":
		return attributeRule{kind: stringListValue}, true
	default:
		return attributeRule{}, false
	}
}

// elementAttributeRule is the attributeRuleLookup shared by every element
// block kind (command, event, readmodel, screen, processor).
func elementAttributeRule(name string) (attributeRule, bool) {
	switch name {
	case "group_id", "domain", "model_context", "slice", "title", "description", "aggregate", "api_endpoint":
		return attributeRule{kind: stringValue}, true
	case "tags", "aggregate_dependencies", "triggers":
		return attributeRule{kind: stringListValue}, true
	case "context":
		return attributeRule{kind: stringValue, enum: []string{"INTERNAL", "EXTERNAL"}}, true
	case "type":
		return attributeRule{kind: stringValue, enum: []string{"COMMAND", "EVENT", "READMODEL", "SCREEN", "AUTOMATION"}}, true
	case "service":
		return attributeRule{kind: stringOrNullValue}, true
	case "creates_aggregate", "sketched", "list_element":
		return attributeRule{kind: boolValue}, true
	case "prototype":
		return attributeRule{kind: objectValue}, true
	default:
		return attributeRule{}, false
	}
}

// screenImageAttributeRule is the attributeRuleLookup for "screen_image"
// block attributes.
func screenImageAttributeRule(name string) (attributeRule, bool) {
	if name == "title" || name == "url" {
		return attributeRule{kind: stringValue}, true
	}
	return attributeRule{}, false
}

// tableAttributeRule is the attributeRuleLookup for "table" block attributes.
func tableAttributeRule(name string) (attributeRule, bool) {
	if name == "title" {
		return attributeRule{kind: stringValue}, true
	}
	return attributeRule{}, false
}

// specificationAttributeRule is the attributeRuleLookup for
// "specification" block attributes.
func specificationAttributeRule(name string) (attributeRule, bool) {
	switch name {
	case "vertical":
		return attributeRule{kind: boolValue}, true
	case "title", "slice_name", "linked_id":
		return attributeRule{kind: stringValue}, true
	default:
		return attributeRule{}, false
	}
}

// actorAttributeRule is the attributeRuleLookup for "actor" block attributes.
func actorAttributeRule(name string) (attributeRule, bool) {
	if name == "auth_required" {
		return attributeRule{kind: boolValue}, true
	}
	return attributeRule{}, false
}

// fieldAttributeRule is the attributeRuleLookup shared by "field" and
// "subfield" blocks.
func fieldAttributeRule(name string) (attributeRule, bool) {
	switch name {
	case "type":
		return attributeRule{kind: stringValue, enum: []string{"String", "Boolean", "Double", "Decimal", "Long", "Custom", "Date", "DateTime", "UUID", "Int"}}, true
	case "example":
		return attributeRule{kind: anyValue}, true
	case "mapping", "schema":
		return attributeRule{kind: stringValue}, true
	case "optional", "technical_attribute", "generated", "id_attribute", "pii":
		return attributeRule{kind: boolValue}, true
	case "cardinality":
		return attributeRule{kind: stringValue, enum: []string{"List", "Single"}}, true
	default:
		return attributeRule{}, false
	}
}

// dependencyAttributeRule is the attributeRuleLookup for "dependency"
// block attributes.
func dependencyAttributeRule(name string) (attributeRule, bool) {
	switch name {
	case "type":
		return attributeRule{kind: stringValue, enum: []string{"INBOUND", "OUTBOUND"}}, true
	case "title":
		return attributeRule{kind: stringValue}, true
	case "element_type":
		return attributeRule{kind: stringValue, enum: []string{"EVENT", "COMMAND", "READMODEL", "SCREEN", "AUTOMATION"}}, true
	default:
		return attributeRule{}, false
	}
}

// specificationStepAttributeRule is the attributeRuleLookup shared by
// "given", "when", and "then" step blocks.
func specificationStepAttributeRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "linked_id":
		return attributeRule{kind: stringValue}, true
	case "tags":
		return attributeRule{kind: stringListValue}, true
	case "examples":
		return attributeRule{kind: objectListValue}, true
	case "index", "spec_row":
		return attributeRule{kind: integerValue}, true
	case "type":
		return attributeRule{kind: stringValue, enum: []string{"SPEC_EVENT", "SPEC_COMMAND", "SPEC_READMODEL", "SPEC_ERROR"}}, true
	case "expect_empty_list":
		return attributeRule{kind: boolValue}, true
	default:
		return attributeRule{}, false
	}
}

// commentAttributeRule is the attributeRuleLookup for "comment" block
// attributes.
func commentAttributeRule(name string) (attributeRule, bool) {
	if name == "description" {
		return attributeRule{kind: stringValue}, true
	}
	return attributeRule{}, false
}

// flatMapDiagnostics validates each item with validate and concatenates the
// results, replacing the repeated "for range { append(diagnostics,
// validate(x)...) }" accumulation pattern used throughout this file.
func flatMapDiagnostics[T any](items []T, validate func(T) hcl.Diagnostics) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, item := range items {
		diagnostics = append(diagnostics, validate(item)...)
	}
	return diagnostics
}

// The functions below walk a parsed HCL document top to bottom, one block
// kind at a time, and collect every diagnostic found along the way. Each
// function calls hcl.Body.Content (not the more lenient PartialContent)
// with that block's schema from above; Content is what turns any attribute
// or child block outside the schema into an "Unsupported argument" or
// "Unsupported block type" diagnostic, so the structural rules from
// eventmodeling.hclspec.md are enforced by this call, not by code written
// here. Because of that, every dispatcher below (validateSliceChild,
// validateElementChild, validateSpecificationChild) can safely fall
// through to "default: return nil" for a child type it does not recognize:
// by the time the dispatcher runs, Content has already rejected any block
// whose type is not in the parent's schema, so the default case is a
// defensive fallback that should never actually be reached, not a place
// where a real violation could slip through unreported.

// validateBody validates the top-level document: every "slice" block in
// turn.
func validateBody(body hcl.Body) hcl.Diagnostics {
	schema := modelSchema()
	content, diagnostics := body.Content(&schema)
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateSlice)...)
}

// validateSlice validates one "slice" block: its own attributes, then
// every direct child block.
func validateSlice(block *hcl.Block) hcl.Diagnostics {
	schema := sliceSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, sliceAttributeRule)...)
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateSliceChild)...)
}

// validateSliceChild routes one direct child of a "slice" block to the
// validation logic for its specific block type.
func validateSliceChild(child *hcl.Block) hcl.Diagnostics {
	switch child.Type {
	case "command":
		return validateElement(child, "COMMAND")
	case "event":
		return validateElement(child, "EVENT")
	case "readmodel":
		return validateElement(child, "READMODEL")
	case "screen":
		return validateElement(child, "SCREEN")
	case "processor":
		return validateElement(child, "AUTOMATION")
	case "screen_image":
		return validateSimpleBlock(child.Body, screenImageSchema(), screenImageAttributeRule)
	case "table":
		return validateTable(child)
	case "specification":
		return validateSpecification(child)
	case "actor":
		return validateSimpleBlock(child.Body, actorSchema(), actorAttributeRule)
	default:
		return nil
	}
}

// validateElement validates one element block (command, event, readmodel,
// screen, or processor): its attributes, that its "type" attribute matches
// expectedType for the kind of block it is, and every field/dependency
// child.
func validateElement(block *hcl.Block, expectedType string) hcl.Diagnostics {
	schema := elementSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, elementAttributeRule)...)
	if attribute, ok := content.Attributes["type"]; ok {
		typeValue, typeDiagnostics := attribute.Expr.Value(nil)
		diagnostics = append(diagnostics, typeDiagnostics...)
		if !typeDiagnostics.HasErrors() && matchesKind(typeValue, stringValue) && typeValue.AsString() != expectedType {
			diagnostics = append(diagnostics, errorDiagnostic(
				attribute.Expr.Range(),
				"Invalid element type",
				fmt.Sprintf("type must be %q for an %s block.", expectedType, block.Type),
			))
		}
	}
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateElementChild)...)
}

// validateElementChild routes one direct child of an element block to the
// validation logic for its specific block type.
func validateElementChild(child *hcl.Block) hcl.Diagnostics {
	switch child.Type {
	case "field":
		return validateField(child)
	case "dependency":
		return validateSimpleBlock(child.Body, dependencySchema(), dependencyAttributeRule)
	default:
		return nil
	}
}

// validateSimpleBlock validates a block that has attributes but no child
// blocks of its own (screen_image, actor, comment): it checks the body
// against schema and validates the resulting attributes with ruleFor.
func validateSimpleBlock(body hcl.Body, schema hcl.BodySchema, ruleFor attributeRuleLookup) hcl.Diagnostics {
	content, diagnostics := body.Content(&schema)
	return append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, ruleFor)...)
}

// validateTable validates one "table" block: its attributes, then every
// "field" child.
func validateTable(block *hcl.Block) hcl.Diagnostics {
	schema := tableSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, tableAttributeRule)...)
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateField)...)
}

// validateSpecification validates one "specification" block: its
// attributes, then every given/when/then step and comment child.
func validateSpecification(block *hcl.Block) hcl.Diagnostics {
	schema := specificationSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, specificationAttributeRule)...)
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateSpecificationChild)...)
}

// validateSpecificationChild routes one direct child of a "specification"
// block to the validation logic for its specific block type.
func validateSpecificationChild(child *hcl.Block) hcl.Diagnostics {
	switch child.Type {
	case "given", "when", "then":
		return validateSpecificationStep(child)
	case "comment":
		return validateSimpleBlock(child.Body, commentSchema(), commentAttributeRule)
	default:
		return nil
	}
}

// validateSpecificationStep validates one "given"/"when"/"then" block: its
// attributes, then every "field" child.
func validateSpecificationStep(block *hcl.Block) hcl.Diagnostics {
	schema := specificationStepSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, specificationStepAttributeRule)...)
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateField)...)
}

// validateField validates one "field" block: its attributes, then every
// nested "subfield" child, recursively, since a subfield has the same
// shape as a field.
func validateField(block *hcl.Block) hcl.Diagnostics {
	schema := fieldSchema()
	content, diagnostics := block.Body.Content(&schema)
	diagnostics = append(diagnostics, validateAttributes(content.Attributes, schema.Attributes, fieldAttributeRule)...)
	return append(diagnostics, flatMapDiagnostics(content.Blocks, validateField)...)
}

// validateAttributes checks every attribute in schema that is actually
// present in attributes against the rule ruleFor returns for its name, and
// reports a diagnostic for each one that fails.
func validateAttributes(attributes hcl.Attributes, schema []hcl.AttributeSchema, ruleFor attributeRuleLookup) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, schemaAttribute := range schema {
		name := schemaAttribute.Name
		attribute, ok := attributes[name]
		if !ok {
			// The document simply omits this optional attribute; there is
			// nothing to check.
			continue
		}
		value, attributeDiagnostics := attribute.Expr.Value(nil)
		diagnostics = append(diagnostics, attributeDiagnostics...)
		if attributeDiagnostics.HasErrors() {
			// The expression itself is malformed or non-literal (for
			// example a variable reference); Value already reported that,
			// so there is no literal value left here to type- or
			// enum-check.
			continue
		}

		rule, ok := ruleFor(name)
		if !ok {
			// This attribute has no rule defined for it. That only
			// happens for attributes this validator intentionally leaves
			// unchecked beyond the schema's presence/required rules, so
			// accept the value as is.
			continue
		}
		if !matchesKind(value, rule.kind) {
			diagnostics = append(diagnostics, errorDiagnostic(
				attribute.Expr.Range(),
				"Invalid attribute value",
				fmt.Sprintf("%s must be %s.", name, rule.kind),
			))
			continue
		}
		if len(rule.enum) > 0 && !slices.Contains(rule.enum, value.AsString()) {
			diagnostics = append(diagnostics, errorDiagnostic(
				attribute.Expr.Range(),
				"Invalid enum value",
				fmt.Sprintf("%s must be one of: %s.", name, strings.Join(rule.enum, ", ")),
			))
		}
	}
	return diagnostics
}

// String describes kind in the article-plus-noun phrasing used inside
// "<attribute> must be <kind>." diagnostic messages, for example "a
// string" or "a list of objects".
func (kind valueKind) String() string {
	switch kind {
	case anyValue:
		return "a literal value"
	case stringValue:
		return "a string"
	case stringOrNullValue:
		return "a string or null"
	case boolValue:
		return "a boolean"
	case integerValue:
		return "an integer"
	case stringListValue:
		return "a list of strings"
	case objectValue:
		return "an object"
	case objectListValue:
		return "a list of objects"
	default:
		return "a valid value"
	}
}

// matchesKind reports whether value's runtime type matches kind. It is the
// one place in this file that reaches into the cty type system directly,
// because HCL's literal-only evaluation means a valid attribute value is
// always a concrete cty.Value with a concrete type, never something that
// needs further evaluation.
func matchesKind(value cty.Value, kind valueKind) bool {
	switch kind {
	case anyValue:
		return true
	case stringValue:
		return !value.IsNull() && value.Type().Equals(cty.String)
	case stringOrNullValue:
		return value.IsNull() || value.Type().Equals(cty.String)
	case boolValue:
		return !value.IsNull() && value.Type().Equals(cty.Bool)
	case integerValue:
		if value.IsNull() || !value.Type().Equals(cty.Number) {
			return false
		}
		// cty numbers are arbitrary-precision decimals; big.Exact confirms
		// this particular number has no fractional part, i.e. it is
		// actually an integer and not just a whole-looking float.
		_, accuracy := value.AsBigFloat().Int(nil)
		return accuracy == big.Exact
	case stringListValue:
		if value.IsNull() || !(value.Type().IsTupleType() || value.Type().IsListType() || value.Type().IsSetType()) {
			return false
		}
		// HCL list literals decode as tuples, so a "list of strings"
		// attribute must accept a tuple, list, or set type and then check
		// every element is itself a non-null string.
		iterator := value.ElementIterator()
		for iterator.Next() {
			_, item := iterator.Element()
			if item.IsNull() || !item.Type().Equals(cty.String) {
				return false
			}
		}
		return true
	case objectValue:
		return !value.IsNull() && value.Type().IsObjectType()
	case objectListValue:
		if value.IsNull() || !(value.Type().IsTupleType() || value.Type().IsListType() || value.Type().IsSetType()) {
			return false
		}
		// Same reasoning as stringListValue above, but each element must
		// be an object rather than a string.
		iterator := value.ElementIterator()
		for iterator.Next() {
			_, item := iterator.Element()
			if item.IsNull() || !item.Type().IsObjectType() {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// errorDiagnostic builds one hcl.Diagnostic at error severity, located at
// subject, with the given summary and detail text.
func errorDiagnostic(subject hcl.Range, summary, detail string) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   detail,
		Subject:  &subject,
	}
}
