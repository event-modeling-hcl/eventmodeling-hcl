# Event Modeling HCL v0.8.0 reference

This is a compact authoring checklist derived from the normative v0.8.0
specification. A model is one `.em.hcl` file or a folder of `.em.hcl` files.
v0.8.0 adds the method facts of RFC 0007: scenario `case`, `empty_history`,
dispatch assertions, aggregate `identity`, `lifecycle`, `evidence_source` and
`evidence`, `decision`, hotspot kinds, `field_source` with field references,
and the key flags (see "Method facts"). Every v0.7.0 model stays valid.
v0.7.0 added a screen `kind` (`"ui"` or `"api"`) and stopped counting
`api_endpoint` as a command reason. v0.6.0 added view form `automation`
scenarios. v0.5.0 was a breaking revision: a v0.4.0 model does not validate
until `emhcl migrate` converts it (see "Tooling").

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
  `subfield`. The method facts have their own required attributes (see
  "Method facts").

## Top-level blocks

```text
board,
bounded_context, actor, team, system,
chapter, hotspot,
lifecycle, evidence_source, evidence, decision,
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
`processor`, `screen_image`, `table`, `field_source`, and `scenario` blocks,
subject to its pattern's flow rules.

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
not v0.8.0 contracts and not methodology lineage. `emhcl import` can write them
to keep JSON data. Keep them in an imported file, but do not add new ones. Do not write
`session:`, `latest:`, `derived:`, or `aggregate:` expressions. Record a
source or a derivation in a `field_source` block (see "Field sources").
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
`actor` and `kind`. Use only attributes with actual known meaning; do not fill
metadata for completeness.

A Screen is the interface of the system at one step: what one actor sees or
sends. A page with several actions is several screens. `kind` is `"ui"`
(default when omitted) or `"api"`: a request that a client program sends, or a
response that it reads. Another string is `EM008`, a non-string is `EM007`, and
`kind` on any other block is `EM000`. An API screen uses the same edges and
rules (`EM201`, `EM401`) as a UI screen. Its `actor` is the person or partner
system that decides; its `api_endpoint` is the route, for example
`"POST /pets"`. `fmt` never adds `kind`.

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
| Qualified element for a record `on` | `command.place_order.place_order` |
| State inside its lifecycle | `state.open` |
| Lifecycle part for a record `on` | `lifecycle.order`, `state.order.open`, `transition.order.place`, `invalid_transition.order.ship_open` |
| Field of an event | `event.orders.order_placed.field.order_id` |
| Field of a local element | `command.place_order.field.order_id` |
| Field of a qualified element | `command.place_order.place_order.field.order_id` |
| Nested field | `command.place_order.field.address.field.city` |
| Evidence source | `evidence_source.order_policy` |
| Decision | `decision.one_order_list` |
| Hotspot, for an evidence `on` | `hotspot.payment_authority` |

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

A Command should have an incoming edge or `external_trigger = true`; otherwise
validation reports `EM404`. `api_endpoint` on a Command records where code
receives it; it is not a reason. A command sent through an API gets an API
screen as its issuer:

```hcl
state_change "add_pet" {
  screen "add_pet_request" {
    kind         = "api"
    api_endpoint = "POST /owners/{ownerId}/pets"
    actor        = actor.clinic_staff
    to           = [command.add_pet]
  }

  command "add_pet" {
    to = [event.pet_management.pet_added]
  }
}
```

The validator does not count issuers. Methodology still requires exactly one
screen or processor issuer. `external_trigger` does not replace that issuer
when the trigger can be modeled. A fact that another system reports, such as a
webhook, is not an API screen: model it as an Event of an `external = true`
context and translate it with an `automation`.

`triggers` is an unchecked string list. It does not create an issuer and does
not replace an event. Do not use it as an invisible signal.

## Attributes and blocks that do not exist

Do not invent `linked_copy`, `element_copy`, `storyline`, `query`, `note`, or
`roles`. Canvas linked
copies are one catalog event referenced from many workflows. Storylines are
multiple scenarios or ordered `given` steps. There is no backward-edge form;
a todo list includes its completion event in `readmodel.from`.

The CLI accepts `mapping` and `schema` as unchecked strings. They are not a
field-source language or a specification contract. Use `field_source`. Do not add them to new or
repaired models. Leave existing values unless metadata repair is requested.

## Scenarios

Each step has exactly one target. `error` is a literal string. These
productions are a copy of the "Scenarios" section of the specification, in its
EBNF notation: `{ x }` is zero or more, `|` is a choice, quoted text is
literal, and `(* ... *)` is a comment. Steps may appear in any order. The
specification wins if the two differ. The validator reports a wrong count as
`EM301`, a step with zero or two targets as `EM302`, and a target that the
workflow kind or the form does not allow as `EM303`. A `case`,
`empty_history`, or `expect_no_dispatch` that does not fit the scenario is
`EM304`. The count of `when` steps selects the form of an `automation`
scenario: one `when` step gives the action form, no `when` step gives the view
form. `empty_history` does not select a form. A view form `then` read model of
another workflow (`readmodel.<other_workflow>.<id>`) is `EM101`.

```text
state_change_scenario = 'scenario' id '{' { sc_given | sc_when | sc_then | comment | attribute } '}'
  (* zero or more sc_given, exactly one sc_when, one or more sc_then *)
  (* optional case from the command cases, no empty_history *)
sc_given              = 'given' '{' event_target step_body '}'
sc_when               = 'when'  '{' command_target step_body '}'
sc_then               = 'then'  '{' ( event_target | error_target ) step_body '}'

state_view_scenario   = 'scenario' id '{' { sv_given | sv_then | comment | attribute } '}'
  (* one or more sv_given, no when step, one or more sv_then *)
  (* with empty_history = true: no sv_given *)
  (* optional case from the read model cases *)
sv_given              = 'given' '{' event_target step_body '}'
sv_then               = 'then'  '{' ( readmodel_target | error_target ) step_body '}'

automation_scenario   = action_scenario | view_scenario
action_scenario       = 'scenario' id '{' { au_given | au_when | au_then | comment | attribute } '}'
  (* zero or more au_given, exactly one au_when, one or more au_then *)
  (* optional case from the command cases, no empty_history *)
view_scenario         = 'scenario' id '{' { av_given | av_then | comment | attribute } '}'
  (* one or more av_given, no when step, one or more av_then *)
  (* with empty_history = true: no av_given *)
  (* optional case from the read model cases *)
au_given              = 'given' '{' ( event_target | readmodel_target ) step_body '}'
au_when               = 'when'  '{' ( processor_target | command_target ) step_body '}'
au_then               = 'then'  '{' ( event_target | error_target | dispatch_target ) step_body '}'
  (* dispatch_target only when the au_when step targets a processor *)
av_given              = 'given' '{' event_target step_body '}'
av_then               = 'then'  '{' ( readmodel_target | error_target ) step_body '}'
  (* the read model of av_then belongs to the same automation *)

event_target          = 'event' '=' event_ref
command_target        = 'command' '=' command_ref
readmodel_target      = 'readmodel' '=' readmodel_ref
processor_target      = 'processor' '=' processor_ref
error_target          = 'error' '=' string
dispatch_target       = 'command' '=' command_ref
  (* a command of the same automation, local form *)
step_body             = { attribute | field | field_list }
  (* a step sets exactly one target attribute *)
  (* expect_no_dispatch = true only on a dispatch_target step, with no field
     and no fields list *)
```

A canvas automation spec that puts a command in Then is a dispatch assertion:
a `then` command step in an action form scenario whose `when` step targets the
processor. A spec that asserts "nothing dispatched" adds
`expect_no_dispatch = true` to that step. A `then` command step in another
scenario is `EM303`. A command of another workflow is `EM101`. Two `then`
steps that name one command, one with `expect_no_dispatch` and one without, are
`EM305`. The validator does not require a `processor.to` edge to the command.

```hcl
then {
  expect_no_dispatch = true
  command            = command.record_deposit
}
```

A scenario with no `when` step is legal only in the view form: given events,
then a read model of the same automation. An empty `when {}` block is still
illegal, because each step sets exactly one target (`EM302`).

`case` names the pattern of a scenario. A `state_change` scenario and an
`automation` action form scenario use the command cases: `happy_path`,
`variant`, `refused_input`, `state_violation`, `business_duplicate`, `limit`,
`authorization`, `alternative_path`, `external_failure`, `compensation`. A
`state_view` scenario and an `automation` view form scenario use the read model
cases: `appear`, `change`, `disappear`. An unknown string is `EM008`. A case of
the other group is `EM304`. The validator does not compare the case with the
steps. Write a case only when the business model states it.

`empty_history = true` states a read model before any event. It is legal on a
`state_view` scenario and an `automation` view form scenario. Such a scenario
has no `given` step (`EM301`). On another scenario it is `EM304`.

```hcl
scenario "no_booking_yet" {
  empty_history = true

  then {
    expect_empty_list = true
    readmodel         = readmodel.open_bookings
  }
}
```

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

Use `expect_empty_list = true` only for an intentionally empty read-model
result, never as an error. Use `expect_no_dispatch` for a no-dispatch
assertion. A scenario cannot assert
that one selected row is absent. Show concrete remaining rows or an empty
list. Include the completion event in a worker todo list's `from` events
when it closes the work. A translation relays an external fact and does not
use its internal result as a completion closer.

Check scenario targets against the actual flow. The CLI does not establish
that the command or processor produces the asserted event. Do not combine an
event and an error as if both occurred. Encode compensation as its own
state change or automation, not an event in When and a command in Then.

If a supplied duplicate-trigger rule requires harmless repetition, state it in
an automation action form scenario with `case = "business_duplicate"` and a
`then` command step with `expect_no_dispatch = true`. A target command's
supplied already-done rejection can have its own `state_change` scenario with
an `error` step. HCL cannot mark a command rejection as a successful no-op.
Report any remaining representation gap instead of changing the rule.

Business scenario coverage and field-source review belong to
[event-modeling](../../event-modeling/SKILL.md). For HCL output, encode at
least one supplied view example for each read model, including todo lists.
Do not invent examples or failure wording to fill a checklist.

## Diagnostic signals and business judgment

`EM401` signals a screen with more than one command. `EM402` signals a command
with more than one event. `EM403` signals a read model with more than one
input event. `EM405` signals uneven scenario coverage. These signals do not
replace the business thresholds in the sibling skill. Do not automatically
split a workflow merely to remove a fan-in or fan-out warning.

`EM404` checks for a command reason: an incoming canonical flow or a justified
`external_trigger`. `api_endpoint` on a command is not a reason. The business
method separately requires a real issuer. Do not add `external_trigger` or
`triggers` to hide missing causality.

## Core diagnostics of v0.8.0

| Code | Condition |
| --- | --- |
| `EM017` | A lifecycle is not valid: a second lifecycle for one aggregate, no `state`, an empty `events` list or an empty invalid-transition `before` list, one state and command pair in a transition and in an invalid transition, or a transition event of another aggregate. |
| `EM018` | An event of an aggregate with an `identity` has no top-level field of the identity type. |
| `EM202` | A field source is not valid: an empty `inputs` list, an input that does not reach the target along one canonical edge of the workflow, an event or command field with more than one input, or a second field source with the same target. |
| `EM304` | `case`, `empty_history`, or `expect_no_dispatch` does not fit the scenario or the step. |
| `EM305` | A scenario states that the processor issues a command and also that it does not issue it. |

Existing codes cover more cases. `EM007` is an empty required text, for
example a decision `reason`. `EM101` is `state.<id>` outside a lifecycle, a
`then` command of another workflow, an `identity` that is not a `field_type`
traversal, an index step such as `x[0]` in a field reference, or a record `on`
whose kind the record does not accept. `EM102` is a field name that the lookup
does not find. `EM000` is a missing `source` on `"stated"` or `"implied"`
evidence, `signal` on a `"choice"` decision, a missing or misplaced hotspot
`concerns`, or a key flag on a `field_type`. Core codes are `EM000` to
`EM305`.

## Method facts

Each method fact is optional. Write one only when the business model states
it. None of them adds a flow edge, a board position, or a chapter membership.

### Aggregate identity and key flags

```hcl
aggregate "booking" {
  identity = field_type.booking_id
}
```

- `identity` is a `field_type` traversal (`EM101` otherwise).
- Each event with `aggregate` set to that aggregate has a top-level field of
  the identity type: a `field` block with that `type`, or a `fields` entry.
  A built-in type with the same name does not count (`EM018`).
- `correlation_key = true` on a `field` or `subfield` states that the value
  links an outside fact to an entity of the model. `deduplication_key = true`
  states that a second delivery with the same value records no new fact.
  Neither flag adds a rule. On a `field_type` they are `EM000`.

### Lifecycles

```hcl
lifecycle "booking" {
  aggregate = aggregate.booking.booking

  state "open" {
    path = "main"
  }

  state "confirmed" {
    end  = true
    path = "main"
  }

  state "canceled" {
    end  = true
    path = "cancel"
  }

  transition "confirm_booking" {
    command = command.confirm_booking.confirm_booking
    events  = [event.booking.booking_confirmed]
    before  = [state.open]
    after   = state.confirmed
  }

  invalid_transition "confirm_canceled_booking" {
    command = command.confirm_booking.confirm_booking
    reason  = "A canceled booking cannot be confirmed."
    before  = [state.canceled]
  }
}
```

- `aggregate` is `aggregate.<context>.<id>`. `command` is
  `command.<workflow>.<id>`. Each `events` entry is `event.<context>.<id>`.
  Another form is `EM101`.
- `state` accepts `end` (Boolean) and `path` (`"main"`, `"cancel"`,
  `"failure"`, `"compensation"`).
- `transition` requires `command`, a non-empty `events` list, `before` (a list
  of `state.<id>`, empty when the transition starts the stream), and `after`.
- `invalid_transition` requires `command`, a non-empty `before` list, and a
  non-empty `reason` in business words.
- One lifecycle for each aggregate. The rules of `EM017` apply.
- The validator does not compare a transition with flow edges or scenarios.

### Field sources

```hcl
field_source {
  inputs = [screen.booking_form.field.name]
  target = command.open_booking.field.guest_name
}
```

- A `field_source` is an unlabeled block inside a workflow. It requires
  `target` (one field reference) and `inputs` (a non-empty list of field
  references). `description` states the rule of a derived value.
- Elements of the same workflow use the local form. Events use
  `event.<context>.<id>`. A qualified element of another workflow is `EM101`.
- Each input element connects to the target element by one canonical edge of
  the workflow, from input to target. An event or command target has exactly
  one input. A second source for the same target is `EM202`.
- The lookup reads the `field` blocks and the `fields` entries of the element,
  then the `subfield` blocks of the field itself. It does not read the
  subfields of a `field_type`. A name that the lookup does not find is `EM102`.
- The validator does not compare types or units, and the language has no
  formula language.

### Records and attach targets

`hotspot`, `evidence`, and `decision` are records. Each `on` names one attach
target: a catalog item, a workflow, an owner, an actor, a workflow-qualified
element, a lifecycle part, or a field reference. Evidence also accepts
`decision.<id>` and `hotspot.<id>`. A hotspot also accepts `decision.<id>`.
Another kind is `EM101`. An undeclared target is `EM102`.

```hcl
evidence_source "booking_policy" {
  kind     = "document"
  location = "docs/booking-policy.pdf"
}

evidence "canceled_booking_rule" {
  certainty = "stated"
  location  = "section 2, page 3"
  source    = evidence_source.booking_policy
  on        = invalid_transition.booking.confirm_canceled_booking
}

decision "one_open_bookings_list" {
  kind      = "accepted_debt"
  signal    = "EM403"
  statement = "The open bookings list reads two events."
  reason    = "The guest sees one list. A split gives two lists with one purpose."
  on        = readmodel.open_bookings.open_bookings
}
```

- `evidence_source.kind` is `"document"`, `"code"`, `"application"`, or
  `"person"`. `location` is optional.
- `evidence` requires `on` and `certainty` (`"stated"`, `"implied"`,
  `"assumed"`). `"stated"` and `"implied"` require `source` (`EM000`). Evidence
  never changes validity or severity.
- `decision` requires `statement` and `reason`. `kind` is `"choice"` (default)
  or `"accepted_debt"`. `signal` names one code from `"EM401"` to `"EM406"`,
  only with `"accepted_debt"`. A decision never removes or changes a
  diagnostic. Without `on`, it is about the whole model.

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

`hotspot.on` names one attach target (see "Records and attach targets"), for
example a workflow-qualified element such as
`processor.capture_payment.gateway`.

A hotspot accepts four optional attributes for its kind:

```hcl
hotspot "refund_rule" {
  kind     = "assumption"
  category = "rule"
  question = "Does the guest lose the deposit after a cancellation?"
  options  = ["The guest loses the deposit.", "The guest gets the deposit back."]
  on       = command.cancel_booking.cancel_booking
}
```

- `kind` is `"question"` (default), `"assumption"`, or `"request"`.
- `category` is `"action"`, `"data"`, `"next"`, `"rule"`, `"example"`,
  `"view"`, `"who"`, or `"time"`.
- `options` is a list of non-empty strings: the answers that the author
  offers.
- `concerns` names the kind of model part that a request concerns, for
  example `"scenario"` or `"field_source"`. A request requires it, and another
  kind must not have it (`EM000`).
- Only `status` controls `EM406`.

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
  90-hotspots.em.hcl     # hotspot, lifecycle, evidence, and decision blocks
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
- Every Command has a validator reason (incoming flow or `external_trigger`).
  `api_endpoint` alone is not a reason. Exactly one issuer is methodology, not
  a counted check.
- No invented `linked_copy`, `storyline`, `query`, or lineage `mapping` DSL.
- Scenario cardinality and targets match the workflow pattern.
- Every `field` and `subfield` has an explicit `type`. Field examples match the
  type and cardinality; no duplicate shorthand fields.
- One `board`; it lists each story workflow once, in business-time order.
- No workflow is in two chapters, and no chapter lists a workflow twice.
- Genuine unknowns are hotspots, not weakened or invented contracts.
- Each method fact comes from the business model. No `case`, lifecycle,
  evidence, decision, or field source is invented to fill a checklist.
- Each `field_source` follows one canonical edge, and each event or command
  target has one input.
- Titles are written only where the text differs from what the ID conveys.

## Tooling

Use emhcl v0.17.0 or newer. It implements specification v0.8.0. v0.16.0
implements v0.7.0 and rejects the method facts. v0.14.0 and v0.15.0 implement
v0.6.0 and reject a screen `kind` with `EM000`. v0.12.0 and
v0.13.0 implement v0.5.0 and reject view form automation scenarios. Older
releases implement v0.4.0 and reject a `board` block.

```text
emhcl migrate -w old-model.em.hcl
emhcl fmt -w model.em.hcl
emhcl validate model.em.hcl
emhcl validate --profile strict model.em.hcl
emhcl diagram model.em.hcl -o model.html
emhcl serve model.em.hcl
```

`validate`, `diagram`, `serve`, `export`, and `migrate` accept a file or a
folder in place of `model.em.hcl`. From v0.15.0, `fmt` also accepts a folder
and formats each member file. v0.14.0 formats one file and refuses a folder,
so with v0.14.0 run `fmt -w` on each member file. `serve` reloads the page
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
  hotspots, teams, systems, read-model questions, and screen `kind = "api"`
  (the screen keeps `apiEndpoint`; import makes it a `"ui"` screen). It also
  drops `identity`, lifecycles, evidence sources, evidence, decisions,
  `field_source` blocks, key flags, and scenario `case`. It omits an
  `empty_history` scenario and a no-dispatch `then` step, and a scenario that
  then has no `then` step. `import`
  drops the original canvas IDs and field-name spellings.
- Both commands fail when one workflow or slice has two or more actors.
- Do not use `export` and `import` to repair a model. Edit the HCL.
