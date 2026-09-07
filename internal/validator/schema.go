package validator

import "github.com/hashicorp/hcl/v2"

func modelSchema() hcl.BodySchema {
	return hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{
		{Type: "bounded_context", LabelNames: []string{"id"}},
		{Type: "actor", LabelNames: []string{"id"}},
		{Type: "team", LabelNames: []string{"id"}},
		{Type: "system", LabelNames: []string{"id"}},
		{Type: "chapter", LabelNames: []string{"id"}},
		{Type: "hotspot", LabelNames: []string{"id"}},
		{Type: "state_change", LabelNames: []string{"id"}},
		{Type: "state_view", LabelNames: []string{"id"}},
		{Type: "automation", LabelNames: []string{"id"}},
		{Type: "translation", LabelNames: []string{"id"}},
	}}
}

func boundedContextSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "description"}, {Name: "external"}, {Name: "owner"}},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "aggregate", LabelNames: []string{"name"}},
			{Type: "field_type", LabelNames: []string{"name"}},
			{Type: "event", LabelNames: []string{"id"}},
		},
	}
}

func aggregateSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "description"}}}
}

func eventSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "group_id"}, {Name: "tags"}, {Name: "title"}, {Name: "description"},
			{Name: "aggregate"}, {Name: "aggregate_dependencies"}, {Name: "service"}, {Name: "sketched"},
			{Name: "prototype"}, {Name: "list_element"}, {Name: "fields"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}},
	}
}

func workflowSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "status"}, {Name: "description"}, {Name: "owner"}},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "command", LabelNames: []string{"id"}}, {Type: "readmodel", LabelNames: []string{"id"}},
			{Type: "screen", LabelNames: []string{"id"}}, {Type: "screen_image", LabelNames: []string{"id"}},
			{Type: "processor", LabelNames: []string{"id"}}, {Type: "table", LabelNames: []string{"id"}},
			{Type: "scenario", LabelNames: []string{"id"}},
		},
	}
}

func elementSchema(kind string) hcl.BodySchema {
	attributes := []hcl.AttributeSchema{
		{Name: "group_id"}, {Name: "tags"}, {Name: "title"}, {Name: "description"},
		{Name: "aggregate"}, {Name: "aggregate_dependencies"}, {Name: "api_endpoint"}, {Name: "service"},
		{Name: "creates_aggregate"}, {Name: "external_trigger"}, {Name: "triggers"}, {Name: "sketched"}, {Name: "prototype"},
		{Name: "list_element"}, {Name: "from"}, {Name: "to"}, {Name: "fields"},
	}
	if kind == "readmodel" {
		attributes = append(attributes, hcl.AttributeSchema{Name: "question", Required: true})
	}
	if kind == "screen" {
		attributes = append(attributes, hcl.AttributeSchema{Name: "actor"})
	}
	return hcl.BodySchema{Attributes: attributes, Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}}}
}

func fieldSchema(_ string) hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "type"}, {Name: "example"}, {Name: "mapping"}, {Name: "optional"},
			{Name: "technical_attribute"}, {Name: "generated"}, {Name: "id_attribute"}, {Name: "pii"},
			{Name: "schema"}, {Name: "cardinality"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "subfield", LabelNames: []string{"name"}}},
	}
}

func tableSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "fields"}}, Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}}}
}

func scenarioSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "description"}},
		Blocks:     []hcl.BlockHeaderSchema{{Type: "given"}, {Type: "when"}, {Type: "then"}, {Type: "comment"}},
	}
}

func scenarioStepSchema() hcl.BodySchema {
	return hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "title"}, {Name: "tags"}, {Name: "examples"}, {Name: "event"}, {Name: "command"},
			{Name: "readmodel"}, {Name: "processor"}, {Name: "error"}, {Name: "expect_empty_list"}, {Name: "fields"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "field", LabelNames: []string{"name"}}},
	}
}

func actorSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "auth_required", Required: true}, {Name: "description"}}}
}

func ownerSchema(kind string) hcl.BodySchema {
	attributes := []hcl.AttributeSchema{{Name: "title"}, {Name: "description"}}
	if kind == "system" {
		attributes = append(attributes, hcl.AttributeSchema{Name: "external"})
	}
	return hcl.BodySchema{Attributes: attributes}
}

func chapterSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "description"}, {Name: "workflows", Required: true}}}
}

func hotspotSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "question", Required: true}, {Name: "description"}, {Name: "on"}, {Name: "status"}}}
}

func screenImageSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "title"}, {Name: "url"}}}
}

func commentSchema() hcl.BodySchema {
	return hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "description", Required: true}}}
}
