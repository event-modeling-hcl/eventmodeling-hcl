# Event Modeling methodology

Use this reference to turn incomplete domain input into a coherent business
story. It works for workshops, plain-language models, and business reviews.
It does not require HCL or a CLI.

The authoritative source is
[EVENT_MODELING_METHODOLOGY.md](https://github.com/event-modeling-hcl/spec/blob/main/guides/EVENT_MODELING_METHODOLOGY.md).
Where its sources disagree, Appendix C records the tension and the operative
rule. Follow Appendix C rather than a conflicting worked example.

Business meaning and its representation are separate concerns. A canvas has
columns, lanes, stickies, linked copies, and storyline drawings. These do not
create additional business elements. If encoding is requested, use the sibling
[Event Modeling HCL skill](../../event-modeling-hcl/SKILL.md) for legal syntax,
types, scenarios, validation, and technical reports. Preserve an intent that
the encoding cannot express directly, and state the gap. Do not invent syntax.

Event Modeling does not require Event Sourcing. Model the process as it works
on paper. Do not add brokers, databases, retries, handlers, or implementation
architecture unless the user asks. Spinners, cache refreshes, and session checks
are not business steps.

## Modes

Keep discovery and review separate within a pass:

- Discovery captures or extends the process. Flag a naming slip or incomplete
  precondition and continue. Do not stop each flow for a full audit.
- Review examines completeness and readiness. Report each distinct violation
  with evidence and its business consequence. Do not quietly restructure.
- Review with fixes applies the requested corrections within the permissions
  of the [shared edit policy](edit-policy.md).

A hotspot is an open question, pain point, or dispute. Attach it to the affected
rule or element. Record an answer before resolving it. Keep genuine unknowns
on workshop drafts. A visible assumption does not resolve a material question
unless the responsible stakeholder accepts it. Never invent a rule to make
a model appear ready. Do not call it implementation-ready while material
business questions remain open. Technical validity is a separate concern.

## Elements

Use domain language for each element:

- Event: An immutable past-tense business fact. Use a specific name such as
  "order placed", not "record updated", "sidebar opened", or "API called".
  A correction is a new event. A recalculated value that changes with source
  data is not an event. Each fact belongs to one context and one aggregate.
  A name alone can carry a payload-free fact. Otherwise include its identity
  and the facts that make it meaningful. Do not pad the payload.
- Command: Rejectable imperative intent, such as "place order", not "load
  orders". It has exactly one issuer. Success records one or more events.
  A refused precondition records no event. Show its error in a scenario.
  A failure that the business tracks, such as "payment failed", is an event.
  Validate commands against prior facts, never against a read model.
- Read model: A rebuildable projection that answers one question. Name the
  data, such as "order summary", not a projector or repository. It never
  validates a command.
- Screen: One decision surface for a person or client interaction. One screen
  state issues at most one command. Several actions mean several screen states
  and usually separate state changes. No person or client decision means no
  screen. Use a processor. Repeated titles do not make two elements identical.
- Processor: The machine decision point, often drawn as a gear. Give it a
  business name, such as "billing scheduler", not "order handler". It reads
  pending work, decides, and issues a command. It does not write target events.
- Hotspot: A question or dispute, not an event. State the question and keep
  the answer visible when resolved. Use the shared policy for its lifecycle.

"User" is not a useful actor name. Name each human role and each external
system. State what each role can and cannot do. A role with no command and
no view is decorative. Give it real work or remove it.

Decision-time totals are a source tension. Do not store values that recalculate
as data changes in events. A total fixed when a command succeeds is a domain
choice. Keep a hotspot unless the user already decided.

## Causality and workflows

The information cycle is:

```text
Screen or processor -> command -> event -> read model -> screen or processor
```

A story starts with a state view, or an automation reacting to an existing
fact. It does not start with an unmotivated command. Each command has one
issuer, even if several possible issuers seem convenient.

A workflow is one pattern-sized capability. Use exactly three patterns:

| Pattern | Business flow |
| --- | --- |
| State change | A screen expresses an actor's intent through a command, which records events on success. An API request still names the deciding actor. |
| State view | Events supply a read model for one screen or API response. There is no command for the query. |
| Automation | Events open a pending-work read model. A processor issues a command, which records events. Translation is this pattern across an ownership boundary. |

Keep the issuing screen and command together. A view of the result is a
separate workflow. Never combine an independent state change and state view.
An automation includes its own pending-work read model and issued command.
Do not invent a fourth pattern.

A deliberately omitted real issuer needs an explicit explanation. It is not
permission to skip a screen, processor, or translation. A route describes an
interaction, not its cause. A fact another system reports, such as a webhook,
is an external event to translate. It is not an API screen.

An invisible signal or timer is not a sufficient cause. Identify the real
business fact that opens the work. Deadlines and "submissions closed" are
processors fed by events, not human screens.

Name a workflow after its defining command, read model, or automation command.
Avoid names such as "order management". Workflows communicate through facts:
"depends on order placed", not "depends on the place-order workflow".

Story transitions include view to change, change to view, change to automation,
automation to change, and automation to view. A change followed by another
change without a new screen or processor is a gap. The second command needs
its own issuer.

## Discovery order

Enter at the first incomplete step. Do not repeat finished discovery unless
the user asks. Plot the story before fields. One finished slice, a small
business capability, beats several speculative slices.

1. State the scope in a few sentences when it is unclear. Name the perspective,
   such as bidder versus issuer or customer versus warehouse.
2. Identify roles and brainstorm past-tense facts. Ask what happens first,
   last, and next. Assign each fact to one owning context.
3. Order the facts in business time. Include failures and cancellations.
4. Group the story into small capabilities. Keep boundaries tentative.
5. Add screens for human decisions and processors for machine decisions.
6. Add commands with a specific actor, prior facts, success events, and each
   distinct failure. Use one command per screen state.
7. Add read models. Every view screen needs one. An input screen also needs
   prior state unless it is a genuinely blank creation form.
8. Establish teams, systems, ownership, and aggregate boundaries. Translate
   facts that cross ownership boundaries.
9. Detail fields and scenarios one flow at a time. Then perform a review pass.

Write down the rare blank-form exemption. "Session context" does not supply
missing prior state. Do not invent extra slices after every command, read
model, and automation has its workflow and the requested scope is covered.

Chapters group journeys, not verbs or pages. About 6–15 workflows is a smell
threshold, not a quota. When in doubt about chapter boundaries, merge them.
A branch that starts another story belongs in a new chapter. A branch that
changes only the ending stays in the current chapter.

The board, an ordered view of the story, follows business time. Chapter
membership groups workflows but does not order them. Do not move every view
to the end if people use it earlier. File or source order is not business time.

Document work can proceed one chapter at a time. Workshops can plot all
stories before details. Both approaches must preserve the same business-time
story. A cold reader must find scope, identities, assumptions, and rejected
alternatives in the journey notes. Do not pad a simple chapter.

When extending a model, preserve its valid identity vocabulary and story.
Find existing definitions across the whole model before creating a new one.
Insert new work where it occurs in business time and in its journey. Migrate
affected references without duplicate contracts or obsolete aliases, subject
to the shared edit policy.

## Fields and sources

Every supplied field on a command, event, read model, or screen needs a source.
A missing source is a gap, not a detail to fill later. References alone do
not establish where information comes from.

Apply these information-completeness contracts to every flow and scenario:

| Contract | Rule |
| --- | --- |
| Command to event invariant | Every event field comes from an accepted command field, a known system-generated value, or a documented derivation. Record any rename. Command inputs must not silently disappear. |
| Events to read model invariant | Every read-model field comes from a connected source event or a documented derivation of those event fields. |
| Issuer to command precondition | Before a command can work, its issuer supplies every input that the system does not generate. Prior facts used for carried values must appear in the rule's scenarios. |
| Model completion condition | Every field has a traceable source or an explicit domain rule. Otherwise an attached hotspot names the gap. |

Trace sources according to the element:

- A command field is user input on its issuing screen, a value from a prior
  fact used in its scenarios, or a known system-filled value such as a new
  identifier or timestamp. User input is never system-generated.
- An event field is copied from the accepted command, with a documented rename
  if needed, or is a documented generated value or derivation.
- A read-model field comes from connected facts or a documented derivation.
  Recalculating totals belong here, not in a recalculation event.
- A processor-issued command field comes from its consumed event or pending-work
  read model, or from a documented external result in a translation.
- A screen field comes from its connected read model or is input to the one
  command that the screen state issues.

Document a derivation on the owning element or scenario. A shared field name
or reusable domain concept does not prove lineage without a real connection.
If a required source is not connected, establish the evidenced connection or
record a hotspot. Never invent a join or a private lineage language.

Use the domain's vocabulary for recurring concepts. Record data meaning and
precision when known. Prefer decimal precision for money. Distinguish dates
from instants and whole numbers from fractional values without guessing.
Identify identity fields and compound identities explicitly. Record privacy,
optionality, list size, technical data, and examples only when the evidence
supplies them. Do not add infrastructure guesses as domain fields.

Reuse realistic examples along the chain. An example email remains the same
email. Preserve non-empty examples and avoid filler such as "foo" or "test".
Carry renames and added fields through only the connected chain. Apply the
[shared edit policy](edit-policy.md) before changing that chain or its contracts.

## Scenarios

A scenario states an observable rule using prior facts and expected outcomes.
Name it after the rule, such as "late bid is rejected", not "test 1".
A command scenario describes one isolated transition, not an entire storyline.
A storyline spans several scenarios or an ordered set of prior facts.

Specify intent for each kind of behavior:

- State change: Identify prior facts, the issued command, and resulting events
  or a refusal with a precise business reason.
- State view: Identify source facts and the resulting read-model content.
  The observation does not introduce a command or query step.
- Automation action: Identify setup and trigger facts, with the trigger last.
  State what the processor must issue or decline to issue, and what resulting
  facts or errors the business expects.
- Pending-work view: Identify source facts and the pending rows or an empty
  list. Each automation's pending-work list needs this coverage too.

These are business scenario intentions, not instructions for a file grammar.
A target format can express an intention differently or need a semantic note.
When HCL is requested, let the HCL skill choose legal forms and record any gap.
Do not insert a canvas form into HCL merely because it reads naturally.

A refused command records no event. A failure the business tracks is an event.
An empty list is a view result, not a command error. Do not claim both an
accepted event and a refusal in the same outcome. Use separate scenarios.

Use actual prior facts rather than unsupported prose such as "customer exists".
If another chapter produced a required fact, reference that same fact rather
than copying it or dropping the precondition. Check that the scenario's facts
and outcomes belong to the real flow. Encoding tools do not prove this.

Use these seven questions as a review list, not a quota of seven scenarios:

1. What happens on the happy path?
2. What happens with invalid input?
3. What happens in an illegal state?
4. What happens on a duplicate request or delivery?
5. Is there another successful outcome?
6. What happens when an external system fails?
7. Is compensation needed?

Write each scenario the input supports. Explain why a category cannot occur.
For a possible but unspecified rule, ask or attach a hotspot. Do not fabricate
error wording or business decisions.

Every read model needs at least one view scenario. Command tests elsewhere do
not cover its projection. Show concrete rows or an explicitly empty result.
For pending work, show what opens a row and what remains after completion.

For repeated automation triggers, preserve the supplied harmless-repeat behavior.
If the command is issued again, specify its already-done rejection when that
rule is known. Do not equate rejection with a successful no-op. If the processor
dispatches nothing, keep that intent without inventing a rejection or event.
Record missing duplicate rules as questions. Compensation is a separate
reversing command in its own state change or automation, not a hidden action.

## Automations and pending work

Every fully modeled automation has a pending-work read model. There is no
pure-relay exemption. If the user explicitly requests a minimal legal encoding
without one, record that exception and its lost business detail. Only an
encoding request needs this exception. It is not a default workshop shortcut.

The flow stays in one automation:

```text
Events -> pending-work read model -> processor -> command -> events
```

Ask a pending-work question, such as "Which placed orders still need a
confirmation?" Do not use a general entity-status view instead. List membership
means open work. Do not add a status field merely to mark a row done.
A completion fact removes the row from the worker's list. Specify the content
before and after completion with view scenarios, including any remaining rows.

Distinguish workers and translations:

- A worker opens from internal events. Its completion event, often its own
  result, also feeds the list and removes completed work. This is the canvas
  method's single backward-arrow exception, not reversed causality.
- A translation opens from a foreign fact. Do not feed the internal result
  back as its closer. It relays the fact rather than tracking completion.
- A following worker exists only for a new decision, invariant, choice, or
  missing data. If it only renames an already-decided fact, stop after
  translation and project a state view from the internal event.

Describe supplied business failures and technical retry outcomes in notes or
scenarios. A technical retry that leaves work pending is not a business event
unless the business records the failure. Do not design retry infrastructure.

A long wait, human approval, compensation, or joined decisions spans several
workflows connected by events. Do not hide a workflow engine in one processor.
A list-shaped presentation or a list-valued field alone does not define
pending work. State the list's business question.

## Translation

Translate another system's or another team's fact before local worker decisions:

```text
Foreign event -> pending-work read model -> processor -> command -> internal event
```

The internal event is a new fact in this domain's language, not a linked copy.
Names can differ, such as foreign "payment captured" and local "payment
authorized". Reject transport names such as "payment signal received" or
"order synced" when they hide the business fact.

Recover this domain's identity before recording the internal event. If the
model initiates the external action, record the outward correlation or a
pairing fact on this side. Missing correlation needs a failure scenario and
often a manual-review hotspot. It is not permission to guess a join.

Keep the foreign identifier for deduplication and reconciliation, not as the
primary identity of this domain's entity. A repeated delivery on that reference
must not record a second domain fact. Specify the command's duplicate rejection.

Classify every external field as copied into a domain field, enriched from
local facts, ignored, or inferred. State the source and rule for each inference.
Do not copy a foreign payload shape wholesale. Document conversions such as
cents to decimal amounts as derivations.

Translation is not a new business decision, but it still validates and can
fail. Preserve both facts. Do not add another decision event that only repeats
the translation.

## Read models

Use one question per read model. If a person points to two areas separately,
such as a statistics tile and live availability, model separate state views.
Do not use one wide read model or a copied screen to hide two questions.
Illustrations do not establish information flow.

More than three source events is a candidate for discussion, not an automatic
split. Trace fields rather than counting edges alone. Remove sources that no
field uses. If a field needs the full lifecycle, name the field and explain
why. That field does not justify unrelated fields riding along.

Place the view after its producers when they share one journey. Keep it where
the narrative uses it, rather than moving all views to the end. A read model
without a consuming screen or processor is an orphan.

When a later fact updates the same question, extend the same projection's
sources. Create another state view when the question changes. Do not copy a
projection merely to show a later point in time.

## Aggregates

An aggregate is the history and consistency boundary of one business identity.
"All orders" and mixed logs are read models, not aggregates. Each event belongs
to one aggregate within its owning context.

Length alone is not a reason to split. Split when events belong to different
entity lifecycles. A child with its own lifecycle has its own aggregate. A
small child created and destroyed with its parent can stay inside the parent.

A command that must atomically change two aggregates is a boundary smell.
Revisit the boundary or record a fact in one aggregate and use a later workflow
in the other. A rule determines which prior facts it needs. Scenarios show
that scope. Do not widen dependencies to hide an unclear boundary.

## Ownership

System shape follows ownership of facts. Each bounded context owns its events.
Name its responsible team, system, or context. Keep one canonical definition
of each fact and reference it wherever the story needs it. A linked canvas
copy does not create another fact.

Translate across another team's or system's ownership boundary before local
worker decisions. Mark a context as external only when it sits outside the
modeled system. Another team's context can remain internal to that system.
It still needs an explicit translation contract at the ownership boundary.
If an encoding treats that edge as an internal worker, keep the business
translation visible and attach a hotspot for the boundary the tool cannot see.

No circular "team A waits on team B waits on team A" is acceptable without
explicit event contracts. Revisit the boundary rather than adding a hidden
call. Do not invent one workflow lane per team. Ownership and canvas layout
are separate concerns.

## Review shapes

Shape names are internal shorthand. Describe the actual concern in plain
words with stakeholders. Methodology thresholds differ from tool warnings.
The HCL skill owns those diagnostic thresholds and their reporting.

| Shape | Business concern and operative threshold |
| --- | --- |
| Bed | A screen state issues any second command. Always split the actions, related or not. |
| Left chair | A command produces more than two events. Ask whether they always happen together. It is a candidate, not an automatic split. |
| Right chair | A read model consumes more than three events. Judge each field and source, not the count alone. |
| Shelf | One workflow has noticeably more scenarios than its neighbors. There is no fixed count. Investigate missing coverage or oversized scope. |

Do not split a justified multi-event command merely to silence a tool warning.
Do not ignore a second command because it feels related. Drop a candidate only
after events, fields, or scenarios demonstrate one job.

## Governance

Apply the [shared edit policy](edit-policy.md) before modifying any existing
workflow, connected chain, or shared catalog contract. It is the sole source
for statuses, named approvals, movement, renaming, and locked-consumer impact.
Carry its permissions into the handoff. Review findings do not grant edits.

## Evidence

Make the source of a non-obvious modeling choice visible:

- Documents: Treat tender text, specifications, and tickets as requirements.
  Preserve deadlines, exclusions, penalties, numbers, and dates verbatim.
  Record conflicting passages in one hotspot that cites both sources.
  Label conclusions as stated, implied, or assumed.
- Code and schemas: Treat them as hypotheses. Persisted state changes suggest
  events. A person-triggered write suggests a screen and command. A job or
  inbound call suggests a processor. A query suggests a read model. Do not
  reuse technical type names as business names. Name the construct and the
  conclusion, such as "Assumed from OrderService.confirm; not confirmed by a
  domain expert". Add a hotspot for an assumed material rule.
- Tests and UI: Use them to check vocabulary and observed behavior. They do
  not outrank explicit stakeholder rules. Capture intent, such as "confirm
  the order", rather than button labels. A new screen state changes navigation
  or content, not decoration. Do not model every possible page combination.
- Drift: Compare implementation and model only when both are available.
  Report each difference once at its origin. Do not fabricate a comparison.
  A created or planned workflow without code is expected, not drift.
  Diagnosis does not authorize changes to locked workflows.

When no one can answer and the user says not to block, choose a reasonable,
visible, correctable assumption. Keep material uncertainty attached to its
rule. One focused question beats several speculative slices.

## Review checklist

A valid encoding is not proof of a good business model. Ask these questions:

- Are events past-tense facts and commands imperative intent, rather than CRUD,
  UI, or framework words?
- Does each command have one issuer, and each screen state at most one command?
- Does a second state change have a new issuer?
- Are foreign facts translated before local workers decide?
- Does each fully modeled automation include pending work, or a recorded
  user-requested encoding exception?
- Does every supplied field have a visible source and any derivation rule?
- Do scenarios distinguish refusal, recorded failure, and empty view results?
- Does every read model, including each pending-work list, have view coverage?
- Does each aggregate answer "the history of which entity?"?
- Are unknowns hotspots or visible assumptions rather than invented rules?
- Did all edits respect the shared edit policy?
- Can a new modeler follow the story quickly?
- Can a projection change without rewriting event history?

## Operative tensions

Keep these distinctions visible rather than removing one side:

- Decision-time calculated event fields remain unresolved until the user
  decides. Values that recalculate from changing data are never events.
- A refused command produces an error and no event. A tracked business failure
  produces an event. The domain expert draws the line.
- Workers consume internal facts. Translation is the automation that receives
  foreign facts, including across a team's ownership boundary.
- Foreign identifiers support reconciliation, not primary domain identity.
- Methodology and tool fan-in or fan-out thresholds differ. Split a bed always.
  Treat chair signals as questions and use the business thresholds for judgment.
- A state view observes facts without a new command or query step.
- Scenarios need real prior facts, not unsupported prose. A target format's
  legal scenario grammar belongs to its encoding skill.
- A slice is a workflow, not a separate implementation artifact. Finish its
  specification and record its planning status under the shared edit policy.
- One command belongs to one screen state. Several actions need several states.
- Chapter-at-a-time and whole-story-first are process choices. Business-time
  order stays explicit, and chapter membership only groups workflows.
- Planning numbers in source material are illustrations. Estimate in workflows
  when asked. Do not invent velocity.
