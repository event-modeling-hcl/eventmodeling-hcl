# Event Modeling HCL Specification v1

## Status and Scope

This document defines the native HCL v1 representation of the Event Modeling
Specification. Its domain reference is the upstream [Event Modeling
Specification](https://github.com/dilgerma/event-modeling-spec). It is the
normative v1 HCL contract for this repository. This is a native language, not
a JSON embedding or a JSON/HCL conversion format.

A model is one HCL document. The validator command is:

```text
eventmodeling-hcl validate <model.em.hcl>
```

The `.em.hcl` suffix is mandatory for model files accepted by the validator.
It identifies the complete Event Modeling document; it does not change the HCL
syntax or semantics.

The document contains no event groups, external-event blocks, qualified event
IDs, or `emits` declarations. Events are direct children of their owning
`slice` blocks. Dependencies and `linked_id` values are opaque strings: v1
does not resolve references, require uniqueness, or validate a target kind.

## Conventions

- The only top-level block is `slice`.
- A labeled block encodes the JSON identity property shown in the mapping
  document. Identity is not also written as an HCL `id` or `name` attribute.
- JSON camelCase property names become snake_case HCL attribute names.
- Repeated JSON arrays become repeated blocks. Omitting a repeated block means
  the corresponding JSON array is empty.
- Attribute and block names are lower snake_case. Unknown names are invalid.
- All expressions must be HCL literals. Variables, traversals, functions, and
  interpolation are invalid.
- HCL comments are non-semantic. Specification comments use a `comment` block.

## Grammar

The grammar is structural. Attribute names and allowed values are defined in
the following sections.

```text
document        = { slice }
slice           = 'slice' id '{' slice_attribute | slice_child '}'
slice_child     = element | screen_image | table | specification | actor
element         = command | event | readmodel | screen | processor
command         = 'command' id '{' element_attribute | field | dependency '}'
event           = 'event' id '{' element_attribute | field | dependency '}'
readmodel       = 'readmodel' id '{' element_attribute | field | dependency '}'
screen          = 'screen' id '{' element_attribute | field | dependency '}'
processor       = 'processor' id '{' element_attribute | field | dependency '}'
screen_image    = 'screen_image' id '{' screen_image_attribute '}'
table           = 'table' id '{' table_attribute | field '}'
specification   = 'specification' id '{'
                    specification_attribute | given | when | then | comment
                  '}'
given           = 'given' id '{' step_attribute | field '}'
when            = 'when' id '{' step_attribute | field '}'
then            = 'then' id '{' step_attribute | field '}'
comment         = 'comment' '{' description '}'
actor           = 'actor' name '{' auth_required '}'
field           = 'field' name '{' field_attribute | subfield '}'
subfield        = 'subfield' name '{' field_attribute | subfield '}'
dependency      = 'dependency' id '{' dependency_attribute '}'
```

`id` and `name` above are ordinary quoted HCL block labels. For example,
`event "evt-pet-001" { ... }` represents `{ "id": "evt-pet-001" }`.

## Required Values and Nesting

| Block | Label | Required attributes | Allowed nested blocks |
| --- | --- | --- | --- |
| `slice` | `id` | `title`, `slice_type` | direct slice children |
| `command` | `id` | `title`, `type = "COMMAND"` | `field`, `dependency` |
| `event` | `id` | `title`, `type = "EVENT"` | `field`, `dependency` |
| `readmodel` | `id` | `title`, `type = "READMODEL"` | `field`, `dependency` |
| `screen` | `id` | `title`, `type = "SCREEN"` | `field`, `dependency` |
| `processor` | `id` | `title`, `type = "AUTOMATION"` | `field`, `dependency` |
| `screen_image` | `id` | `title` | none |
| `table` | `id` | `title` | `field` |
| `specification` | `id` | `title`, `linked_id` | `given`, `when`, `then`, `comment` |
| `given`, `when`, `then` | `id` | `title`, `type` | `field` |
| `comment` | none | `description` | none |
| `actor` | `name` | `auth_required` | none |
| `field`, `subfield` | `name` | `type` | `subfield` |
| `dependency` | `id` | `type`, `title`, `element_type` | none |

No child collection is required to have at least one member. This intentionally
adapts the JSON Schema's required collection properties to HCL's block-based
syntax while preserving their empty-array meaning.

## Attributes

All attributes are optional unless the preceding table marks them required.

| Parent block | Attributes |
| --- | --- |
| `slice` | `title` string; `status` enum; `index` integer; `context` string; `slice_type` enum; `aggregates` list(string) |
| element blocks | `group_id` string; `tags` list(string); `domain` string; `model_context` string; `context` enum; `slice` string; `title` string; `type` enum; `description` string; `aggregate` string; `aggregate_dependencies` list(string); `api_endpoint` string; `service` string or `null`; `creates_aggregate` bool; `triggers` list(string); `sketched` bool; `prototype` object; `list_element` bool |
| `screen_image` | `title` string; `url` string |
| `table` | `title` string |
| `specification` | `vertical` bool; `title` string; `slice_name` string; `linked_id` string |
| specification steps | `title` string; `tags` list(string); `examples` list(object); `index` integer; `spec_row` integer; `type` enum; `linked_id` string; `expect_empty_list` bool |
| `comment` | `description` string |
| `actor` | `auth_required` bool |
| `field`, `subfield` | `type` enum; `example` literal; `mapping` string; `optional` bool; `technical_attribute` bool; `generated` bool; `id_attribute` bool; `pii` bool; `schema` string; `cardinality` enum |
| `dependency` | `type` enum; `title` string; `element_type` enum |

## Enumerations

| Attribute | Allowed values |
| --- | --- |
| `slice.status` | `Created`, `Done`, `InProgress` |
| `slice.slice_type` | `STATE_CHANGE`, `STATE_VIEW`, `AUTOMATION` |
| element `context` | `INTERNAL`, `EXTERNAL` |
| element `type` | `COMMAND`, `EVENT`, `READMODEL`, `SCREEN`, `AUTOMATION`; it must match the enclosing block |
| dependency `type` | `INBOUND`, `OUTBOUND` |
| dependency `element_type` | `EVENT`, `COMMAND`, `READMODEL`, `SCREEN`, `AUTOMATION` |
| step `type` | `SPEC_EVENT`, `SPEC_COMMAND`, `SPEC_READMODEL`, `SPEC_ERROR` |
| field `type` | `String`, `Boolean`, `Double`, `Decimal`, `Long`, `Custom`, `Date`, `DateTime`, `UUID`, `Int` |
| field `cardinality` | `List`, `Single` |

## Literal Values

`field.example` accepts any literal HCL value, including strings, numbers,
booleans, `null`, objects, and lists. This is an intentional HCL extension:
the upstream JSON Schema permits strings and objects only. Objects and lists
remain native HCL values; they must not be supplied as JSON encoded strings.
`prototype` is an object literal, while specification-step `examples` is a
list of object literals. Lists used by list-valued attributes contain strings
unless stated otherwise.

```hcl
field "pets" {
  type        = "Custom"
  cardinality = "List"
  example = [{
    id        = 5
    name      = "Mochi"
    birthDate = "2020-01-12"
  }]

  subfield "id" {
    type         = "Int"
    example      = 5
    id_attribute = true
  }
}
```

## Validation Boundary

The validator rejects syntactically invalid HCL, unknown attributes or blocks,
invalid nesting, missing required attributes, non-literal expressions, invalid
attribute types, and invalid enum values. It reports source locations for
errors.

v1 deliberately does not load multiple documents, format HCL, convert JSON,
resolve IDs, check relationship consistency, or enforce uniqueness beyond the
upstream JSON Schema.
