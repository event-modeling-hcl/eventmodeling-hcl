package validator

import (
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

type valueKind uint8

const (
	anyValue valueKind = iota
	stringValue
	stringOrNullValue
	boolValue
	stringListValue
	objectValue
	objectListValue
)

type attributeRule struct {
	kind      valueKind
	enum      []string
	reference bool
}

type attributeRuleLookup func(string) (attributeRule, bool)

var fieldTypes = []string{"String", "Boolean", "Double", "Decimal", "Long", "Custom", "Date", "DateTime", "UUID", "Int"}

func boundedContextRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "description":
		return attributeRule{kind: stringValue}, true
	case "external":
		return attributeRule{kind: boolValue}, true
	case "owner":
		return attributeRule{reference: true}, true
	}
	return attributeRule{}, false
}

func workflowRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "description":
		return attributeRule{kind: stringValue}, true
	case "status":
		return attributeRule{kind: stringValue, enum: []string{"created", "planned", "assigned", "in_progress", "review", "blocked", "done", "informational"}}, true
	case "owner":
		return attributeRule{reference: true}, true
	}
	return attributeRule{}, false
}

func aggregateRule(name string) (attributeRule, bool) {
	return stringRule(name, "title", "description")
}

func elementRule(name string) (attributeRule, bool) {
	switch name {
	case "group_id", "title", "description", "api_endpoint", "question":
		return attributeRule{kind: stringValue}, true
	case "tags", "triggers":
		return attributeRule{kind: stringListValue}, true
	case "service":
		return attributeRule{kind: stringOrNullValue}, true
	case "creates_aggregate", "external_trigger", "sketched", "list_element":
		return attributeRule{kind: boolValue}, true
	case "prototype":
		return attributeRule{kind: objectValue}, true
	case "aggregate", "aggregate_dependencies", "from", "to", "actor":
		return attributeRule{reference: true}, true
	}
	return attributeRule{}, false
}

func fieldRule(name string) (attributeRule, bool) {
	switch name {
	case "type":
		return attributeRule{reference: true}, true
	case "example":
		return attributeRule{kind: anyValue}, true
	case "mapping", "schema":
		return attributeRule{kind: stringValue}, true
	case "optional", "technical_attribute", "generated", "id_attribute", "pii":
		return attributeRule{kind: boolValue}, true
	case "cardinality":
		return attributeRule{kind: stringValue, enum: []string{"List", "Single"}}, true
	}
	return attributeRule{}, false
}

func tableRule(name string) (attributeRule, bool)       { return stringRule(name, "title") }
func screenImageRule(name string) (attributeRule, bool) { return stringRule(name, "title", "url") }
func commentRule(name string) (attributeRule, bool)     { return stringRule(name, "description") }

func scenarioRule(name string) (attributeRule, bool) {
	return stringRule(name, "title", "description")
}

func scenarioStepRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "error":
		return attributeRule{kind: stringValue}, true
	case "tags":
		return attributeRule{kind: stringListValue}, true
	case "examples":
		return attributeRule{kind: objectListValue}, true
	case "expect_empty_list":
		return attributeRule{kind: boolValue}, true
	case "event", "command", "readmodel", "processor":
		return attributeRule{reference: true}, true
	}
	return attributeRule{}, false
}

func actorRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "description":
		return attributeRule{kind: stringValue}, true
	case "auth_required":
		return attributeRule{kind: boolValue}, true
	}
	return attributeRule{}, false
}

func ownerRule(name string) (attributeRule, bool) {
	switch name {
	case "title", "description":
		return attributeRule{kind: stringValue}, true
	case "external":
		return attributeRule{kind: boolValue}, true
	}
	return attributeRule{}, false
}

func chapterRule(name string) (attributeRule, bool) {
	if name == "workflows" {
		return attributeRule{reference: true}, true
	}
	return stringRule(name, "title", "description")
}

func hotspotRule(name string) (attributeRule, bool) {
	switch name {
	case "question", "description":
		return attributeRule{kind: stringValue}, true
	case "on":
		return attributeRule{reference: true}, true
	case "status":
		return attributeRule{kind: stringValue, enum: []string{"open", "resolved"}}, true
	}
	return attributeRule{}, false
}

func stringRule(name string, allowed ...string) (attributeRule, bool) {
	if slices.Contains(allowed, name) {
		return attributeRule{kind: stringValue}, true
	}
	return attributeRule{}, false
}

func validateSimpleBlock(body hcl.Body, schema hcl.BodySchema, ruleFor attributeRuleLookup) hcl.Diagnostics {
	content, diagnostics := body.Content(&schema)
	return append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, ruleFor)...)
}

func validateLiteralAttributes(attributes hcl.Attributes, schema []hcl.AttributeSchema, ruleFor attributeRuleLookup) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, expected := range schema {
		attribute := attributes[expected.Name]
		if attribute == nil {
			continue
		}
		rule, exists := ruleFor(expected.Name)
		if !exists || rule.reference {
			continue
		}
		value, valueDiagnostics := attribute.Expr.Value(nil)
		diagnostics = append(diagnostics, valueDiagnostics...)
		if valueDiagnostics.HasErrors() {
			continue
		}
		if !matchesKind(value, rule.kind) {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidAttribute, attribute.Expr.Range(), "Invalid attribute value", fmt.Sprintf("%s must be %s.", expected.Name, rule.kind)))
			continue
		}
		if len(rule.enum) > 0 && !slices.Contains(rule.enum, value.AsString()) {
			diagnostics = append(diagnostics, invalidEnum(attribute, expected.Name, rule.enum))
		}
	}
	return diagnostics
}

func invalidEnum(attribute *hcl.Attribute, name string, values []string) *hcl.Diagnostic {
	return errorDiagnostic(codeInvalidEnum, attribute.Expr.Range(), "Invalid enum value", fmt.Sprintf("%s must be one of: %s.", name, strings.Join(values, ", ")))
}

func validateExample(attributes hcl.Attributes, fieldType, subject string) hcl.Diagnostics {
	exampleAttribute := attributes["example"]
	if exampleAttribute == nil || fieldType == "" {
		return nil
	}
	example, diagnostics := exampleAttribute.Expr.Value(nil)
	if diagnostics.HasErrors() || example.IsNull() {
		return diagnostics
	}
	cardinality, _ := literalString(attributes["cardinality"])
	return validateExampleValue(exampleAttribute, example, fieldType, cardinality, subject)
}

func validateReferencedExample(attributes hcl.Attributes, reference fieldTypeRef) hcl.Diagnostics {
	exampleAttribute := attributes["example"]
	if exampleAttribute == nil {
		return nil
	}
	example, diagnostics := exampleAttribute.Expr.Value(nil)
	if diagnostics.HasErrors() || example.IsNull() {
		return diagnostics
	}
	cardinality, ok := literalString(attributes["cardinality"])
	if !ok {
		cardinality = reference.cardinality
	}
	return validateExampleValue(exampleAttribute, example, reference.fieldType, cardinality, "referenced field")
}

func validateExampleValue(attribute *hcl.Attribute, value cty.Value, fieldType, cardinality, subject string) hcl.Diagnostics {
	if cardinality == "List" {
		if !isSequence(value) {
			return hcl.Diagnostics{errorDiagnostic(codeInvalidExample, attribute.Expr.Range(), "Invalid example value", fmt.Sprintf("example for a List %s must be a list of %s values.", subject, fieldType))}
		}
		iterator := value.ElementIterator()
		for iterator.Next() {
			_, item := iterator.Element()
			if !matchesFieldType(fieldType, item) {
				return hcl.Diagnostics{errorDiagnostic(codeInvalidExample, attribute.Expr.Range(), "Invalid example value", fmt.Sprintf("every example element must match the field type %s.", fieldType))}
			}
		}
		return nil
	}
	if !matchesFieldType(fieldType, value) {
		return hcl.Diagnostics{errorDiagnostic(codeInvalidExample, attribute.Expr.Range(), "Invalid example value", fmt.Sprintf("example must match the field type %s.", fieldType))}
	}
	return nil
}

func matchesFieldType(fieldType string, value cty.Value) bool {
	if value.IsNull() {
		return true
	}
	switch fieldType {
	case "String", "Date", "DateTime", "UUID":
		return value.Type().Equals(cty.String)
	case "Boolean":
		return value.Type().Equals(cty.Bool)
	case "Double", "Decimal":
		return value.Type().Equals(cty.Number)
	case "Int", "Long":
		if !value.Type().Equals(cty.Number) {
			return false
		}
		_, accuracy := value.AsBigFloat().Int(nil)
		return accuracy == big.Exact
	case "Custom":
		return value.Type().IsObjectType()
	default:
		return false
	}
}

func isSequence(value cty.Value) bool {
	return !value.IsNull() && (value.Type().IsTupleType() || value.Type().IsListType() || value.Type().IsSetType())
}

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
	case stringListValue:
		return sequenceElementsMatch(value, func(item cty.Value) bool { return !item.IsNull() && item.Type().Equals(cty.String) })
	case objectValue:
		return !value.IsNull() && value.Type().IsObjectType()
	case objectListValue:
		return sequenceElementsMatch(value, func(item cty.Value) bool { return !item.IsNull() && item.Type().IsObjectType() })
	default:
		return false
	}
}

func sequenceElementsMatch(value cty.Value, match func(cty.Value) bool) bool {
	if !isSequence(value) {
		return false
	}
	iterator := value.ElementIterator()
	for iterator.Next() {
		_, item := iterator.Element()
		if !match(item) {
			return false
		}
	}
	return true
}

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
