# Event Modeling Source Model to HCL Mapping

This table maps every property from the upstream [Event Modeling
Specification](https://github.com/dilgerma/event-modeling-spec) to native HCL
v1. The HCL contract is defined in [the language specification](../eventmodeling.hclspec.md).

## Top-Level Model and Slice

| JSON path/property | HCL representation | Notes |
| --- | --- | --- |
| `slices` | repeated top-level `slice "<id>"` blocks | Only allowed top-level block. |
| `Slice.id` | `slice` label | No `id` attribute. |
| `Slice.status` | `status` | Optional enum. |
| `Slice.index` | `index` | Optional integer. |
| `Slice.title` | `title` | Required string. |
| `Slice.context` | `context` | Optional string. |
| `Slice.sliceType` | `slice_type` | Required enum. |
| `Slice.commands` | repeated `command "<id>"` | Omitted means `[]`. |
| `Slice.events` | repeated `event "<id>"` | Omitted means `[]`; remains slice-local. |
| `Slice.readmodels` | repeated `readmodel "<id>"` | Omitted means `[]`. |
| `Slice.screens` | repeated `screen "<id>"` | Omitted means `[]`. |
| `Slice.screenImages` | repeated `screen_image "<id>"` | Omitted means `[]`. |
| `Slice.processors` | repeated `processor "<id>"` | Omitted means `[]`. |
| `Slice.tables` | repeated `table "<id>"` | Omitted means `[]`. |
| `Slice.specifications` | repeated `specification "<id>"` | Omitted means `[]`. |
| `Slice.actors` | repeated `actor "<name>"` | Omitted means `[]`. |
| `Slice.aggregates` | `aggregates` | Optional list(string). |

## Elements

`Element` is used by `command`, `event`, `readmodel`, `screen`, and
`processor`. Each HCL block requires a matching explicit `type` attribute.

| JSON property | HCL representation |
| --- | --- |
| `groupId` | `group_id` |
| `id` | enclosing element block label |
| `tags` | `tags` |
| `domain` | `domain` |
| `modelContext` | `model_context` |
| `context` | `context` |
| `slice` | `slice` |
| `title` | `title` (required) |
| `fields` | repeated `field "<name>"` blocks; omitted means `[]` |
| `type` | `type` (required and block-compatible enum) |
| `description` | `description` |
| `aggregate` | `aggregate` |
| `aggregateDependencies` | `aggregate_dependencies` |
| `dependencies` | repeated `dependency "<id>"` blocks; omitted means `[]` |
| `apiEndpoint` | `api_endpoint` |
| `service` | `service` |
| `createsAggregate` | `creates_aggregate` |
| `triggers` | `triggers` |
| `sketched` | `sketched` |
| `prototype` | `prototype` object literal |
| `listElement` | `list_element` |

## Supporting Blocks

| JSON definition/property | HCL representation | Notes |
| --- | --- | --- |
| `ScreenImage.id` | `screen_image` label | No `id` attribute. |
| `ScreenImage.title` | `title` | Required. |
| `ScreenImage.url` | `url` | Optional. |
| `Table.id` | `table` label | No `id` attribute. |
| `Table.title` | `title` | Required. |
| `Table.fields` | repeated `field "<name>"` | Omitted means `[]`. |
| `Specification.id` | `specification` label | No `id` attribute. |
| `Specification.vertical` | `vertical` | Optional bool. |
| `Specification.sliceName` | `slice_name` | Optional string. |
| `Specification.title` | `title` | Required. |
| `Specification.given` | repeated `given "<id>"` | Omitted means `[]`. |
| `Specification.when` | repeated `when "<id>"` | Omitted means `[]`. |
| `Specification.then` | repeated `then "<id>"` | Omitted means `[]`. |
| `Specification.comments` | repeated `comment` | Omitted means `[]`. |
| `Specification.linkedId` | `linked_id` | Required opaque string. |
| `SpecificationStep.title` | `title` | Required. |
| `SpecificationStep.tags` | `tags` | Optional list(string). |
| `SpecificationStep.examples` | `examples` | Optional list(object). |
| `SpecificationStep.id` | `given`/`when`/`then` label | No `id` attribute. |
| `SpecificationStep.index` | `index` | Optional integer. |
| `SpecificationStep.specRow` | `spec_row` | Optional integer. |
| `SpecificationStep.type` | `type` | Required enum. |
| `SpecificationStep.fields` | repeated `field "<name>"` | Omitted means `[]`. |
| `SpecificationStep.linkedId` | `linked_id` | Optional opaque string. |
| `SpecificationStep.expectEmptyList` | `expect_empty_list` | Optional bool. |
| `Comment.description` | `comment { description = "..." }` | Required semantic comment. |
| `Actor.name` | `actor` label | No `name` attribute. |
| `Actor.authRequired` | `auth_required` | Required bool. |
| `Dependency.id` | `dependency` label | No `id` attribute. |
| `Dependency.type` | `type` | Required enum. |
| `Dependency.title` | `title` | Required string. |
| `Dependency.elementType` | `element_type` | Required enum. |

## Fields

| JSON property | HCL representation | Notes |
| --- | --- | --- |
| `Field.name` | `field` or `subfield` label | No `name` attribute. |
| `Field.type` | `type` | Required enum. |
| `Field.example` | `example` | Native literal; scalar, object, list, or `null`. |
| `Field.subfields` | repeated `subfield "<name>"` | Omitted means `[]`. |
| `Field.mapping` | `mapping` | Optional string. |
| `Field.optional` | `optional` | Optional bool. |
| `Field.technicalAttribute` | `technical_attribute` | Optional bool. |
| `Field.generated` | `generated` | Optional bool. |
| `Field.idAttribute` | `id_attribute` | Optional bool. |
| `Field.pii` | `pii` | Optional bool. |
| `Field.schema` | `schema` | Optional string. |
| `Field.cardinality` | `cardinality` | Optional enum. |

`subfield` bodies use the same attributes and may contain further `subfield`
blocks. In element, table, and specification-step bodies, use `field` for the
corresponding source `fields` collection.

## Complete Reference Coverage

`../examples/complete.em.hcl` has been checked against every property mapped
above. Each group below is represented by a native block, label, or attribute
in that executable example.

- [x] `Slice`: `id`, `status`, `index`, `title`, `context`, `sliceType`,
  `commands`, `events`, `readmodels`, `screens`, `screenImages`, `processors`,
  `tables`, `specifications`, `actors`, and `aggregates`.
- [x] `Element`: `groupId`, `id`, `tags`, `domain`, `modelContext`, `context`,
  `slice`, `title`, `fields`, `type`, `description`, `aggregate`,
  `aggregateDependencies`, `dependencies`, `apiEndpoint`, `service`,
  `createsAggregate`, `triggers`, `sketched`, `prototype`, and `listElement`.
- [x] `ScreenImage`: `id`, `title`, and `url`; `Table`: `id`, `title`, and
  `fields`.
- [x] `Specification`: `vertical`, `id`, `sliceName`, `title`, `given`,
  `when`, `then`, `comments`, and `linkedId`; `SpecificationStep`: `title`,
  `tags`, `examples`, `id`, `index`, `specRow`, `type`, `fields`, `linkedId`,
  and `expectEmptyList`.
- [x] `Comment`: `description`; `Actor`: `name`, `authRequired`; `Dependency`:
  `id`, `type`, `title`, `elementType`.
- [x] `Field`: `name`, `type`, `example`, `subfields`, `mapping`, `optional`,
  `technicalAttribute`, `generated`, `idAttribute`, `pii`, `schema`, and
  `cardinality`.
- [x] All supported enum values, the element block/type combinations, a
  `prototype` object, and specification-step object examples. HCL additionally
  permits scalar, `null`, and list `Field.example` literals as documented
  language extensions.
