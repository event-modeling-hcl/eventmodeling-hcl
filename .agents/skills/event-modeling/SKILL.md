---
name: event-modeling
description: >-
  Discover, explain, extend, or review the business meaning of an Event Model
  from product ideas, requirements, workshop notes, user stories, domain prose,
  or existing behavior. Use for Event Modeling workshops, past-tense Events,
  Commands, Read Models, State Change, State View, Automation, Translation,
  business-time stories, slice boundaries, information completeness, ownership,
  and Given/When/Then scenario intent. This skill does not require HCL or tools.
  For .em.hcl syntax, formatting, validation, migration, import, or export, use
  the sibling event-modeling-hcl skill. When creating HCL from prose, use this
  skill first, then hand the business model to event-modeling-hcl.
metadata:
  version: "1.0.0"
  author: workspace
  methodology: "https://github.com/event-modeling-hcl/spec/blob/main/guides/EVENT_MODELING_METHODOLOGY.md"
---

# Event Modeling

Help domain experts and modelers agree on business facts and information flow.
Discover the story before choosing implementation structure or a file format.
A business workshop or review can finish in plain language without writing HCL
or running a CLI.

## References and boundaries

Read [the methodology](references/methodology.md) before discovery, extension,
or a business review. Read [the shared edit policy](references/edit-policy.md)
before changing an existing model. That policy is the sole authority for status
locks, chain permissions, and changes to shared contracts.

This skill owns meaning: scope, vocabulary, facts, rules, ownership, fields,
scenarios, uncertainty, and business review. The sibling
[Event Modeling HCL skill](../event-modeling-hcl/SKILL.md) owns encoding,
legal syntax, references, file layout, CLI gates, and technical reports.
Load it only when the request needs HCL work. A syntax-only repair does not
require a new workshop. A business review of an HCL model still uses this skill.

Do not create implementation code, database schemas, brokers, APIs, retry
mechanisms, or framework architecture unless the user asks. Event Modeling
does not require Event Sourcing. Do not invent a serialization format.

## Deliverable

Produce or update the requested business model in the user's existing medium.
Without an existing medium, use plain-language notes, tables, and ordered flows.
Include the supplied behavior and every named acceptance criterion. Do not
replace the requested model with a plan for a later workshop.

The deliverable must let a reader identify:

- The scope, perspective, actors, vocabulary, and story in business time.
- The facts, rules, ownership, and consistency boundaries.
- The flows, fields, sources, and observable scenario outcomes.
- The unresolved questions, evidence, assumptions, and edit permissions.

A workshop draft can contain real unknowns. Attach each unknown to the affected
rule or element as a hotspot, an open question or dispute. Do not invent a rule
to make the model appear complete. Do not call a model implementation-ready
while a material business question remains open. Technical acceptance is a
separate result owned by the HCL skill when HCL is requested.

## Choose a mode

Discovery captures or extends the process. Flag local naming slips or missing
preconditions and continue useful work. Enter at the first incomplete step.
Do not repeat finished discovery or turn each extension into a full audit.

Review examines the existing business model. Report each distinct finding at
its origin, with its evidence, consequence, and unresolved decision. Do not
quietly restructure the model. A review request does not grant edit permission.
When the user also requests fixes, apply only permitted changes and explain
what each fix resolves. Review is a mode of this skill, not another skill.

## Work from evidence

Read the user's input and relevant requirements, tests, domain types, existing
models, and examples. Reuse the domain's vocabulary. Do not ask for information
that files already provide.

Separate stated facts, implied conclusions, and assumptions. For each
non-obvious choice, name its source and label it as stated, implied, or assumed.
Documents supply authoritative requirements. Keep dates, thresholds, exclusions,
and penalties verbatim. An explicit stakeholder rule outranks code or screens.
Code, schemas, tests, and UI provide evidence, not permission to invent rules.

If sources conflict, record both in one hotspot. Ask one focused question when
a missing decision changes the model materially. If useful work can continue,
record the gap and continue. If the user says not to block, make a visible,
correctable assumption and retain a hotspot for an assumed material rule.

## Discover one flow at a time

1. State scope and perspective if either is unclear.
2. Identify the roles and brainstorm concrete past-tense business facts.
3. Order facts in business time, including failures and cancellations.
4. Group the story into small capabilities without forcing final boundaries.
5. Add human decision surfaces and processors where no person acts.
6. Add commands and read models to explain inputs and outputs.
7. Establish teams, systems, owning contexts, and aggregate boundaries.
8. Detail fields, sources, and scenarios for one flow before moving on.
9. Review the completed story and stop when the supplied scope is covered.

One well-specified slice, a small business capability, beats several speculative
slices. Use one workflow per pattern-sized capability. Keep an issuing screen
with its command. Keep a separate view of the result in its own workflow.
An automation includes its pending-work read model, processor, and command.

Use exactly these three patterns:

- State change: An actor expresses intent through a screen or API interaction,
  a command accepts or rejects it, and success records events.
- State view: Existing events feed a read model that answers one question for
  a screen or API response. A query is not a new command.
- Automation: Events open pending work, a processor issues a command, and
  success records events. Translation is an automation that converts another
  context's facts into the receiving context's language, not a fourth pattern.

One screen state issues at most one command. Every command has exactly one
issuer. A reported fact, such as a webhook, is an external event to translate,
not a screen. Do not hide a missing issuer behind an unexplained signal.

## Make ownership and information complete

Give each event one owning bounded context, a domain with its own language.
Identify who owns each context and which systems sit outside the modeled system.
An aggregate is the history and consistency boundary of one business identity.
It is not every noun, a mixed log, or all instances of an entity. Revisit a
command that must atomically change several aggregates.

Trace every supplied field through real flows. A field must come from user
input, prior facts, a known system-generated value, or a documented derivation.
The issuer must supply each command input that the system does not generate.
Event fields come from accepted command inputs or documented generated values
and derivations. Read-model fields come from connected facts or their documented
derivations. Do not silently discard command inputs or add implementation data.
Keep meaningful example values consistent along the chain.

For translation, recover this domain's identity through an evidenced correlation.
Classify each foreign field as copied, enriched, ignored, or inferred. Use the
foreign identity for reconciliation, not as this domain entity's identity.
Describe duplicate delivery and missing correlation without inventing a join.

## Specify and review behavior

For each supplied rule, describe prior facts, the decision or observation, and
the expected result. Distinguish a refused command with no recorded event from
a failure the business records as an event. Model compensation as its own
reversing command and flow. Do not invent error wording.

Cover supported success and failure paths. Use the seven review questions in
[the methodology](references/methodology.md#scenarios), not a quota of seven
scenarios. Every read model, including a pending-work list, needs a view
scenario. Show concrete rows or an explicitly empty result. Keep scenario
intent separate from any target format's legal scenario grammar.

Review these shapes with the stakeholder in plain words:

- Bed: A second command from one screen state is always a problem.
- Left chair: More than two events from one command needs discussion. Ask
  whether they always happen together. Do not split by count alone.
- Right chair: More than three events feed one read model. Trace each field,
  remove unused sources, and justify any field that needs the full history.
- Shelf: One workflow has noticeably more scenarios than its neighbors.
  Investigate the imbalance. There is no fixed scenario threshold.

Also ask whether a new modeler can follow the story quickly and whether a
projection can change without rewriting event history. Format validity does
not answer either question.

## Hand off only when encoding is requested

Complete the business work first. Then load the
[Event Modeling HCL skill](../event-modeling-hcl/SKILL.md) to encode the result.
Use the user's existing artifacts and plain-language notes for the handoff.
Do not prescribe a new exchange format.

Supply the scope and perspective, vocabulary and facts, ownership and aggregate
boundaries, ordered flows, fields and sources, scenario intent, uncertainty,
and evidence. Include applicable workflow statuses, named approvals, locked
consumers, and blocked chain changes from the shared edit policy.

HCL has a form for each of these method facts: `field_source` for a field
source, `lifecycle` for the states of an aggregate, `evidence_source` and
`evidence` for a source and its certainty, `decision` for a choice and its
reason, scenario `case`, and `expect_no_dispatch` for a processor that issues
no command. The HCL skill chooses the syntax.

The HCL skill must preserve this meaning while choosing legal encodings. If the
format cannot express an intent directly, retain the intent and identify the
gap. Do not weaken a rule or invent syntax to hide that gap.

For a business-only result, report the covered scope, open questions, assumptions,
review findings, and any blocked changes. Do not claim CLI validation or require
a technical report to complete a workshop.
