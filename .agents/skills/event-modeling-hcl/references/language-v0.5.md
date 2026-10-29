# Event Modeling HCL v0.5.0 reference

This is a compact authoring checklist derived from the normative v0.5.0
specification. A model is one `.em.hcl` file or a folder of `.em.hcl` files.
v0.5.0 is a breaking revision: a v0.4.0 model does not validate until
`emhcl migrate` converts it (see "Tooling").

## Language principles

- Block kind expresses the concept; quoted labels are stable identity.
- Labels match `^[a-z][a-z0-9_]*$`.
- Each concern has one control. The `board` block alone sets the story order.
  Chapters are membership sets. File names and block positions carry no
  meaning; one-file and folder models use the same rules.
- Unknown syntax, unresolved references, wrong-kind references, computed values,
  variables, functions, and interpolation are errors.
- Relationships are unquoted traversals, not strings.
- Titles are optional literal text. An omitted title is absent; tools never
  derive one from the label and show the ID instead. Required semantic data:
  `readmodel.question`, `actor.auth_required`, `hotspot.question`, board and
  chapter `workflows`, and the `type` of every `field_type`, `field`, and
  `subfield`.

## Top-level blocks

```text
board,
bounded_context, actor, team, system,
chapter, hotspot,
state_change, state_view, automation
```

## Board

Every model has exactly one unlabeled `board` block:

```hcl
board {
  workflows = [
    workflow.review_cart,
    workflow.place_order,
  ]
}
```

- The list is the story sequence, in business time.
- A missing or repeated board is `EM015`. A workflow listed twice is `EM016`.
  A non-workflow entry is `EM101`; an unknown workflow is `EM102`.
- The list may omit workflows, and may be empty. An omitted workflow is
  off-board: it is valid, gets no diagnostic in any profile, and is never part
  of the story. Add a workflow to the board when it belongs in the story.
- A board edit changes only the story order: not edges, chapters, or
  diagnostics.

## Block contents

A bounded context may contain only `aggregate`, `field_type`, and canonical
`event` blocks. A workflow may contain `command`, `readmodel`, `screen`,
`processor`, `screen_image`, `table`, and `scenario` blocks, subject to its
pattern's flow rules.

## Catalog attributes

```hcl
team "fulfillment_team" {
  title       = "Fulfillment"
  description = "Optional description."
}

system "payment_provider" {
  title       = "Payment provider"
  description = "Optional description."
  external    = true
}

actor "customer" {
  title         = "Customer"
  description   = "Optional description."
  auth_required = true # required
}

bounded_context "orders" {
  title       = "Orders"
  description = "Optional description."
  external    = false
  owner       = team.fulfillment_team
}
```

`bounded_context.owner` may reference a bounded context, team, or system.
Teams are internal owner records; only systems have the `external` attribute.

## Context-owned contracts

```hcl
bounded_context "orders" {
  aggregate "order" {
    title       = "Order"
    description = "Consistency boundary for an order."
  }

  field_type "order_id" {
    type         = "UUID"
    cardinality  = "Single"
    example      = "0e42f9d1-2f55-4cc6-b9e2-3d1926bd21bf"
    id_attribute = true
  }

  event "order_placed" {
    title                  = "Order Placed"
    description            = "Optional description."
    group_id               = "checkout"
    tags                   = ["orders"]
    aggregate              = aggregate.order
    aggregate_dependencies = []
    service                = null
    fields                 = [field_type.order_id]
    sketched               = false
    prototype              = { route = "/orders" }
    list_element           = false
  }
}
```

Outside the context use qualified references:

```hcl
event.orders.order_placed
aggregate.orders.order
field_type.orders.order_id
```

Inside the owning context, local `aggregate.order` and `field_type.order_id`
forms are allowed. Both forms are exact lookups.

## Fields

Built-in types:

```text
String Boolean Double Decimal Long Custom Date DateTime UUID Int
```

Cardinality is `Single` or `List`.

A reusable `field_type` must have an explicit built-in `type`. It may contain
nested `subfield` blocks. Every `field` and `subfield` requires `type`, wherever
it appears: events, workflow elements, tables, scenario steps, and nested
subfields. A missing type is `EM009` in every profile. The type is never
inferred from the field name.

```hcl
field "customer_id" {
  type                = field_type.customers.customer_id # traversal or built-in string
  example             = "customer-42"
  cardinality         = "Single"
  optional            = false
  technical_attribute = false
  generated           = false
  id_attribute        = true
  pii                 = false
}
```

The current CLI also accepts unchecked `mapping` and `schema` strings. They are
not v0.5.0 contracts and not methodology lineage. `emhcl import` can write them
to keep JSON data. Keep them in an imported file, but do not add new ones. Do not write
`session:`, `latest:`, `derived:`, or `aggregate:` expressions. Record a
derivation in `description`, a scenario `comment`, or a `hotspot`.
`generated = true` means the system fills the value; it is not a source link.

Use `fields` for several reusable types with no per-field overrides:

```hcl
fields = [field_type.orders.order_id, field_type.orders.customer_id]
```

List entries become fields named after the final traversal segment and typed
by the traversal. Do not duplicate a name between `fields` and nested `field`
blocks.

For structured data use `type = "Custom"`, nested `subfield` blocks (each with
its own `type`), and `cardinality = "List"` when applicable. Examples are
native HCL values and must match the field's type and cardinality.

## Workflow attributes

Every workflow accepts `title`, `description`, `owner`, and `status`. Status is
one of:

```text
created planned assigned in_progress review blocked done informational
```

Commands, Read Models, Screens, and Processors accept:

```text
group_id, tags, title, description,
aggregate, aggregate_dependencies, api_endpoint, service,
creates_aggregate, external_trigger, triggers,
sketched, prototype, list_element,
from, to, fields, nested field blocks
```

A Read Model additionally requires `question`. A Screen additionally accepts
`actor`. Use only attributes with actual known meaning; do not fill metadata for
completeness.

Presentation blocks:

```hcl
screen_image "checkout_wireframe" {
  title = "Checkout wireframe"
  url   = "https://example.test/checkout.png"
}

table "example_orders" {
  title  = "Example orders"
  fields = [field_type.orders.order_id]
}
```

These blocks have no flow edges.

## Reference forms

| Target | Form |
| --- | --- |
| Bounded context | `bounded_context.orders` |
| Actor | `actor.customer` |
| Team | `team.fulfillment_team` |
| System | `system.payment_provider` |
| Workflow | `workflow.place_order` |
| Context Event | `event.orders.order_placed` |
| Context aggregate | `aggregate.orders.order` |
| Context field type | `field_type.orders.order_id` |
| Local workflow element | `command.place_order` |
| Qualified element for hotspot | `command.place_order.place_order` |

Reference lists use native HCL lists. Never quote a reference:

```hcl
to = [event.orders.order_placed] # correct
to = ["event.orders.order_placed"] # invalid
```

## Canonical flows

| Workflow | Legal canonical edges |
| --- | --- |
| `state_change` | `screen.to -> command`; `command.to -> event` |
| `state_view` | `event -> readmodel.from`; `readmodel.to -> screen` |
| `automation` | `event -> readmodel.from` or `processor.from`; `readmodel.to -> processor`; `processor.to -> command`; `command.to -> event` |

The Event belongs to the catalog, so the receiver records Event input with
`from`. Workflow-local sources record outgoing edges with `to`. Express each
edge exactly once. Reverse forms are errors.

An `automation` may consume internal events, external events, or both. There
is no `translation` keyword (it is an unsupported block, `EM000`). An
automation that consumes at least one Event of an external bounded context is
a translation: tools derive it from the edges (the IR marks such edges
`External`) and may show it as one. Name the workflow for the translation it
performs.

A Command should have an incoming edge, an `api_endpoint`, or
`external_trigger = true`; otherwise validation reports a modeling diagnostic.
The validator does not count issuers. Methodology still requires exactly one
screen or processor issuer. `external_trigger` does not replace that issuer
when the trigger can be modeled.

`triggers` is an unchecked string list. It does not create an issuer and does
not replace an event. Do not use it as an invisible signal.

## Attributes and blocks that do not exist

Do not invent `linked_copy`, `element_copy`, `storyline`, `query`, `note`,
`roles`, `expect_no_dispatch`, or a field-source traversal. Canvas linked
copies are one catalog event referenced from many workflows. Storylines are
multiple scenarios or ordered `given` steps. There is no backward-edge form;
a todo list includes its completion event in `readmodel.from`.

## Scenarios

Each step has exactly one target. `error` is a literal string.

| Workflow | Given | When | Then |
| --- | --- | --- | --- |
| State Change | zero or more Events | exactly one Command | one or more Events/Errors |
| State View | one or more Events | none | one or more Read Models/Errors |
| Automation | zero or more Events/Read Models | exactly one Processor/Command | one or more Events/Errors |

Canvas automation specs that use an empty When, a Then command, or "nothing
dispatched" are not this grammar. Do not encode them by inventing attributes.

Scenario and step attributes:

```hcl
scenario "success" {
  title       = "Optional title"
  description = "Optional description"

  given {
    title             = "Optional step title"
    tags              = ["setup"]
    examples          = [{ order_id = "order-42" }]
    expect_empty_list = false
    event             = event.orders.order_placed
    fields            = [field_type.orders.order_id]
  }

  comment {
    description = "Semantic note retained in the model."
  }
}
```

A step may target `event`, `command`, `readmodel`, `processor`, or `error` only
as allowed by its workflow and step kind. There is no `query` target. Normal HCL
comments are non-semantic.

## Workshop notation

```hcl
chapter "checkout" {
  title       = "Checkout"
  description = "Optional description."
  workflows   = [workflow.review_cart, workflow.place_order]
}
```

A chapter is a non-empty membership set (`EM005` when empty). Its list order
has no meaning, and chapter blocks may live in any file. A workflow in two
chapters is `EM014`; a workflow listed twice in one chapter is `EM002`. A
chapter may contain off-board workflows. Chapters never change the board.

```hcl
hotspot "payment_authority" {
  question    = "Which payment status is authoritative?"
  description = "Optional context."
  status      = "open" # open or resolved
  on          = workflow.capture_payment
}
```

`hotspot.on` can reference catalog/workshop declarations, a workflow, or a
workflow-qualified element such as `processor.capture_payment.gateway`.

## Folder models

A model path is a file or a folder. Use a folder when one file is too large to
read, or when different people own different contexts or workflows.

The member files of a folder model obey these rules:

- A member file is a regular file directly in the folder. Its name ends in
  `.em.hcl` and does not start with `.`. A symlink to a file counts.
  Subfolders and other files are not members.
- The tool sorts member files by name, byte by byte. A number prefix such as
  `10-` is not language syntax. It only sets the file order.
- The model is the union of the top-level blocks of all files. A reference in
  one file can point to a block in any other file. Every check runs on the
  whole model. Each file must parse on its own.
- A block never spans files. Thus a `bounded_context` and all its events,
  aggregates, and field types are in one file.
- IDs are global across files. Each block kind has its own ID space, as in a
  one-file model. All workflow kinds share one space. The same ID twice is
  `EM002`. The detail names the first declaration as `file:line:column`.
- A folder with no member file is `EM001`. A folder with exactly one member
  file behaves exactly like that file. Every rule is the same for one file and
  for a folder.
- The `board` is in one file, like every block. `chapter` blocks may be in
  any files.
- File order never changes the model. It only decides which declaration a
  duplicate diagnostic names as the first one.
- Each diagnostic names the member file of the source that it points at.

The language has no `include` or `import` block, no subfolder members, no
modules or name spaces, and no cross-folder references.

Use this layout when you split a model:

```text
model/
  00-catalog.em.hcl      # board, bounded_context, actor, team, system
  01-chapters.em.hcl     # chapter blocks
  10-register-pet.em.hcl # one workflow per file
  11-pet-directory.em.hcl
  90-hotspots.em.hcl     # hotspot blocks
```

Move each block whole. A new workflow file changes nothing on the board until
you add the workflow to `board.workflows`.

## Static completion checklist

- Each file name ends in `.em.hcl`. The model is one complete file or one
  complete folder.
- All labels are lower snake case and unique in their scope.
- Events, aggregates, and field types are owned by bounded contexts.
- All references are unquoted, resolvable, correctly qualified, and right-kind.
- Every edge appears once in canonical direction.
- Every Read Model has one concrete `question`.
- Every Command has a validator reason (incoming flow, `api_endpoint`, or
  `external_trigger`). Exactly one issuer is methodology, not a counted check.
- No invented `linked_copy`, `storyline`, `query`, or lineage `mapping` DSL.
- Scenario cardinality and targets match the workflow pattern.
- Every `field` and `subfield` has an explicit `type`. Field examples match the
  type and cardinality; no duplicate shorthand fields.
- One `board`; it lists each story workflow once, in business-time order.
- No workflow is in two chapters, and no chapter lists a workflow twice.
- Genuine unknowns are hotspots, not weakened or invented contracts.
- Titles are written only where the text differs from what the ID conveys.

## Tooling

Use emhcl v0.12.0 or newer. It implements specification v0.5.0. Older releases
implement v0.4.0 and reject a `board` block.

```text
emhcl migrate -w old-model.em.hcl
emhcl fmt -w model.em.hcl
emhcl validate model.em.hcl
emhcl validate --profile strict model.em.hcl
emhcl diagram model.em.hcl -o model.html
emhcl serve model.em.hcl
```

`validate`, `diagram`, `serve`, `export`, and `migrate` accept a file or a
folder in place of `model.em.hcl`. `fmt` formats one file and refuses a folder.
For a folder model, run `fmt -w` on each member file. `serve` reloads the page
when a member file is added, removed, renamed, or edited.

Profiles assign severity to the judgment signals `EM401`–`EM406` only:
`workshop` reports them as information, the default `valid` as warnings, and
`strict` makes unreasoned Commands (`EM404`) and open hotspots (`EM406`)
errors. Core rules are errors in every profile. A profile never hides the
model: without core errors, `diagram`, `serve`, and `export` still produce
output, but a `strict` error makes `validate` exit 1. Formatting is idempotent
and preserves block and scenario order.

### Migrating a v0.4.0 model

`emhcl migrate <file|folder>` prints the v0.5.0 result; `-w` rewrites the
changed files. It adds a `board` with every workflow in the old effective
order, writes each formerly inferred field type as a qualified traversal,
writes each formerly derived title as an explicit `title`, and turns
`translation` into `automation`. It writes nothing when the input does not
parse, already has a board, has a typeless field with no unique type, or broke
the v0.4.0 translation rules. Afterward, delete written titles that only
repeat the ID, and remove workflows from the board that do not belong in the
story.

### JSON conversion

emhcl converts slice-based Event Modeling JSON. JSON conversion is a tool
feature, not part of the language.

```text
emhcl import model.json -o model.em.hcl
emhcl export model.em.hcl -o model.json
```

- `import` writes one formatted file and validates it. It exits with code 1 if
  the result has validation errors. Repair the written source.
- `export` accepts a file or a folder. It refuses an `-o` target that ends in
  `.em.hcl` or that is the source file.
- `import` writes a `board` in slice order and turns slice type `TRANSLATION`
  into `automation`. `export` writes `TRANSLATION` for an automation with an
  external edge, and writes off-board workflows after the board, with a
  warning.
- Conversion is not lossless. Read each warning. `export` drops chapters,
  hotspots, teams, systems, and read-model questions. `import` drops the
  original canvas IDs and field-name spellings.
- Both commands fail when one workflow or slice has two or more actors.
- Do not use `export` and `import` to repair a model. Edit the HCL.
