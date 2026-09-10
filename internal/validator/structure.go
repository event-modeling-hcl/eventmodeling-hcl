package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/event-modeling-hcl/eventmodeling-hcl/internal/syntax"
)

func (v *modelValidator) validateBoundedContext(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.BoundedContextSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, boundedContextRule)...)
	context := contextValidator{model: v, contextID: block.Labels[0]}
	if owner := content.Attributes["owner"]; owner != nil {
		diagnostics = append(diagnostics, v.validateOwnerReference(owner)...)
	}
	for _, child := range content.Blocks {
		switch child.Type {
		case "aggregate":
			diagnostics = append(diagnostics, validateSimpleBlock(child.Body, syntax.AggregateSchema(), aggregateRule)...)
		case "field_type":
			diagnostics = append(diagnostics, context.validateField(child, true)...)
		case "event":
			diagnostics = append(diagnostics, context.validateEvent(child)...)
		}
	}
	return diagnostics
}

func (v *modelValidator) validateActor(block *hcl.Block) hcl.Diagnostics {
	return validateSimpleBlock(block.Body, syntax.ActorSchema(), actorRule)
}

func (v *modelValidator) validateOwner(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.OwnerSchema(block.Type)
	return validateSimpleBlock(block.Body, schema, ownerRule)
}

func (v *modelValidator) validateChapter(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.ChapterSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, chapterRule)...)
	if workflows := content.Attributes["workflows"]; workflows != nil {
		diagnostics = append(diagnostics, v.validateReferenceList(workflows, "workflow", "")...)
		diagnostics = append(diagnostics, v.validateChapterRange(workflows)...)
		if references, referenceDiagnostics := hcl.ExprList(workflows.Expr); !referenceDiagnostics.HasErrors() && len(references) == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidChapter, workflows.Expr.Range(), "Invalid chapter", "chapter must contain at least one workflow."))
		}
	}
	return diagnostics
}

func (v *modelValidator) validateHotspot(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.HotspotSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, hotspotRule)...)
	if on := content.Attributes["on"]; on != nil {
		diagnostics = append(diagnostics, v.validateGlobalReference(on)...)
	}
	if status, ok := literalString(content.Attributes["status"]); !ok || status != "resolved" {
		diagnostics = append(diagnostics, warningDiagnostic(codeOpenHotspot, block.DefRange, "Open hotspot", fmt.Sprintf("hotspot %q is unresolved.", block.Labels[0])))
	}
	return diagnostics
}

func (v *modelValidator) validateWorkflow(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.WorkflowSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, workflowRule)...)
	if owner := content.Attributes["owner"]; owner != nil {
		diagnostics = append(diagnostics, v.validateOwnerReference(owner)...)
	}
	for _, child := range content.Blocks {
		if !workflowAllows(block.Type, child.Type) {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidWorkflowChild, child.DefRange, "Invalid workflow child", fmt.Sprintf("%s blocks are not allowed in %s workflows.", child.Type, block.Type)))
			continue
		}
		switch child.Type {
		case "command", "readmodel", "screen", "processor":
			diagnostics = append(diagnostics, v.validateElement(block.Type, block.Labels[0], child)...)
		case "screen_image":
			diagnostics = append(diagnostics, validateSimpleBlock(child.Body, syntax.ScreenImageSchema(), screenImageRule)...)
		case "table":
			diagnostics = append(diagnostics, v.validateTable(child)...)
		case "scenario":
			diagnostics = append(diagnostics, v.validateScenario(block.Type, block.Labels[0], child)...)
		}
	}
	diagnostics = append(diagnostics, v.workflowSmells(block.Type, block.DefRange, content.Blocks)...)
	return diagnostics
}

func workflowAllows(workflowType, childType string) bool {
	if childType == "table" || childType == "scenario" {
		return true
	}
	if childType == "screen_image" {
		return workflowType == "state_change" || workflowType == "state_view"
	}
	switch workflowType {
	case "state_change":
		return childType == "screen" || childType == "command"
	case "state_view":
		return childType == "readmodel" || childType == "screen"
	case "automation", "translation":
		return childType == "readmodel" || childType == "processor" || childType == "command"
	default:
		return false
	}
}

func (v *modelValidator) validateElement(workflowType, workflowID string, block *hcl.Block) hcl.Diagnostics {
	schema := syntax.ElementSchema(block.Type)
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, elementRule)...)
	context := contextValidator{model: v}
	if aggregate := content.Attributes["aggregate"]; aggregate != nil {
		diagnostics = append(diagnostics, context.validateAggregate(aggregate, false)...)
	}
	if dependencies := content.Attributes["aggregate_dependencies"]; dependencies != nil {
		diagnostics = append(diagnostics, context.validateAggregateList(dependencies, false)...)
	}
	if actor := content.Attributes["actor"]; actor != nil {
		diagnostics = append(diagnostics, v.validateReference(actor, "actor", workflowID)...)
	}
	for _, direction := range []string{"from", "to"} {
		if attribute := content.Attributes[direction]; attribute != nil {
			diagnostics = append(diagnostics, v.validateFlowReferences(attribute, workflowType, workflowID, block.Type, direction)...)
		}
	}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

func (v *modelValidator) validateTable(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.TableSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, tableRule)...)
	context := contextValidator{model: v}
	diagnostics = append(diagnostics, context.validateElementFields(content)...)
	return diagnostics
}

func (v *contextValidator) validateEvent(block *hcl.Block) hcl.Diagnostics {
	schema := syntax.EventSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, elementRule)...)
	if aggregate := content.Attributes["aggregate"]; aggregate != nil {
		diagnostics = append(diagnostics, v.validateAggregate(aggregate, true)...)
	}
	if dependencies := content.Attributes["aggregate_dependencies"]; dependencies != nil {
		diagnostics = append(diagnostics, v.validateAggregateList(dependencies, true)...)
	}
	diagnostics = append(diagnostics, v.validateElementFields(content)...)
	return diagnostics
}

// validateElementFields validates the optional shorthand fields = [...] list
// attribute together with the field blocks that share its namespace.
func (v *contextValidator) validateElementFields(content *hcl.BodyContent) hcl.Diagnostics {
	seen := map[string]bool{}
	var diagnostics hcl.Diagnostics
	if attribute := content.Attributes["fields"]; attribute != nil {
		diagnostics = append(diagnostics, v.validateFieldList(attribute, seen)...)
	}
	return append(diagnostics, v.validateFields(content.Blocks, seen)...)
}

func (v *contextValidator) validateFields(fields hcl.Blocks, seen map[string]bool) hcl.Diagnostics {
	var diagnostics hcl.Diagnostics
	for _, field := range fields {
		if len(field.Labels) == 0 {
			continue
		}
		name := field.Labels[0]
		diagnostics = append(diagnostics, validateLabel(field, name)...)
		if seen[name] {
			diagnostics = append(diagnostics, duplicateDiagnostic(field, field.Type, name))
			continue
		}
		seen[name] = true
		diagnostics = append(diagnostics, v.validateField(field, false)...)
	}
	return diagnostics
}

// validateFieldList resolves each entry of a fields = [...] attribute as a
// field_type reference and records its synthesized field name in seen.
func (v *contextValidator) validateFieldList(attribute *hcl.Attribute, seen map[string]bool) hcl.Diagnostics {
	expressions, diagnostics := hcl.ExprList(attribute.Expr)
	if diagnostics.HasErrors() {
		return diagnostics
	}
	for _, expression := range expressions {
		entry := &hcl.Attribute{Name: "fields", Expr: expression, Range: attribute.Range, NameRange: attribute.NameRange}
		address, referenceDiagnostics := v.fieldTypeAddress(entry)
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if referenceDiagnostics.HasErrors() {
			continue
		}
		name := address[strings.LastIndex(address, ".")+1:]
		if seen[name] {
			diagnostics = append(diagnostics, errorDiagnostic(codeDuplicateID, expression.Range(), "Duplicate field name", fmt.Sprintf("field %q is declared more than once in its namespace.", name)))
			continue
		}
		seen[name] = true
	}
	return diagnostics
}

func (v *contextValidator) validateField(block *hcl.Block, fieldTypeDeclaration bool) hcl.Diagnostics {
	schema := syntax.FieldSchema()
	content, diagnostics := syntax.Content(block.Body, schema)
	diagnostics = append(diagnostics, validateLiteralAttributes(content.Attributes, schema.Attributes, fieldRule)...)
	typeAttribute := content.Attributes["type"]
	if typeAttribute == nil {
		if fieldTypeDeclaration {
			return append(diagnostics, errorDiagnostic(codeInvalidFieldType, block.DefRange, "Invalid field type", "a field_type declaration must set an explicit built-in type."))
		}
		address, inferenceDiagnostics := v.inferredFieldTypeAddress(block.Labels[0], block.DefRange)
		diagnostics = append(diagnostics, inferenceDiagnostics...)
		if !inferenceDiagnostics.HasErrors() {
			if reference, exists := v.model.index.fieldTypes[address]; exists {
				diagnostics = append(diagnostics, validateReferencedExample(content.Attributes, reference)...)
			}
		}
		return append(diagnostics, v.validateFields(content.Blocks, map[string]bool{})...)
	}
	if fieldTypeDeclaration {
		fieldType, ok := literalString(typeAttribute)
		if !ok {
			diagnostics = append(diagnostics, errorDiagnostic(codeInvalidFieldType, typeAttribute.Expr.Range(), "Invalid field type", "field_type type must be a built-in type string."))
		} else {
			diagnostics = append(diagnostics, validateFieldTypeAndExample(content.Attributes, typeAttribute, fieldType)...)
		}
	} else if fieldType, ok := literalString(typeAttribute); ok {
		diagnostics = append(diagnostics, validateFieldTypeAndExample(content.Attributes, typeAttribute, fieldType)...)
	} else {
		address, referenceDiagnostics := v.fieldTypeAddress(typeAttribute)
		diagnostics = append(diagnostics, referenceDiagnostics...)
		if reference, exists := v.model.index.fieldTypes[address]; exists {
			diagnostics = append(diagnostics, validateReferencedExample(content.Attributes, reference)...)
		}
	}
	diagnostics = append(diagnostics, v.validateFields(content.Blocks, map[string]bool{})...)
	return diagnostics
}

func validateFieldTypeAndExample(attributes hcl.Attributes, attribute *hcl.Attribute, fieldType string) hcl.Diagnostics {
	if !slices.Contains(fieldTypes, fieldType) {
		return hcl.Diagnostics{invalidEnum(attribute, "type", fieldTypes)}
	}
	return validateExample(attributes, fieldType, "field")
}
