---
name: event-modeling-hcl
description: >-
  Author, extend, repair, review, validate, format, split, migrate, import,
  export, or explain Event Modeling HCL v0.8.0 files and folder models.
  Use for `.em.hcl` syntax, references, scenarios, diagnostics, and emhcl
  commands. For business discovery or business-quality review, use the sibling
  event-modeling skill. When a request turns domain prose into HCL, use
  event-modeling first, then this skill. Do not use this skill for a
  workshop-only discussion with no HCL deliverable.
compatibility: >-
  Targets Event Modeling HCL Specification v0.8.0. For machine verification,
  use emhcl v0.17.0 or newer. v0.16.0 implements v0.7.0 and rejects the
  method facts of v0.8.0. v0.14.0 and v0.15.0 implement v0.6.0 and reject
  a screen `kind`. v0.12.0 and v0.13.0 implement v0.5.0. Older releases
  implement v0.4.0.
metadata:
  version: "3.1.0"
  spec_version: "v0.8.0"
  emhcl_min: "v0.17.0"
  author: workspace
  specification: "https://github.com/event-modeling-hcl/spec"
---

# Event Modeling HCL

Produce or update the requested `.em.hcl` file or model folder. This skill
owns representation and tool operations, not business decisions. A diagram or
prose explanation does not replace a requested model file.

## Load the owning rules

Read these references for the requested operation:

- Read [the language reference](references/language-v0.8.md) before writing,
  repairing, reviewing, splitting, migrating, importing, or exporting HCL.
- Read [the shared edit policy](../event-modeling/references/edit-policy.md)
  before any mutation, including formatting, conversion, and repair. It is the
  single source for workflow locks and shared-contract permissions.
- Read [canonical HCL patterns](references/patterns.md) when the encoding of a
  pattern or scenario is uncertain. Adapt vocabulary to the domain.

Use [event-modeling](../event-modeling/SKILL.md) when business meaning is
incomplete or the user requests business-quality review. Do not rerun discovery
for a syntax-only repair whose meaning is already clear.

The normative v0.8.0 specification wins over examples, existing files, and
general HCL knowledge. Legal syntax does not prove business quality. Do not
change business meaning to satisfy the grammar. Report a representation gap
and return the unresolved decision to business modeling.

## Establish the handoff and edit scope

For domain prose, establish the business model with the sibling skill first.
Carry its scope, vocabulary, owned facts, flows, fields and their sources,
scenario intent, evidence, unknowns, and permitted changes into HCL. The
handoff can be the existing conversation or model. Do not require a new file
format or duplicate artifact.

Preserve supplied behavior and every named acceptance criterion. If a business
fact is missing, ask a focused question or retain an attached open `hotspot`.
Never invent a rule, field source, error message, or implementation detail.
An unresolved material question prevents a claim of implementation readiness.

For existing models, inspect every member file before adding global IDs or
changing shared contracts. Apply the shared edit policy before planning the
change. A formatter or migration command is a mutation too. Do not run a
whole-model rewrite when its changes exceed the permitted scope.

On review-only requests, report findings without editing files. Business
review uses the sibling skill. Syntax review uses this skill and the CLI.
For review-and-fix requests, make only authorized changes.

## Preserve model structure

Keep the existing file or folder shape unless the user requests a change.
Write a new model as one file unless the user requests a folder or the model
has many workflows or several owners.

Every model has exactly one `board`. Its `workflows` list records the business
story order. Include every workflow that belongs in that story. A workflow
omitted from the board is valid but is not part of the story.

In a folder model, put the board and catalog blocks in one file, chapter
blocks in any member file, and each workflow in its own file. Move blocks
whole. IDs are global across member files. File names do not define story
order. Follow the folder rules in the language reference.

When adding a workflow, update the board at its business-time position and its
chapter membership. Carry authorized renames through every affected reference.
Do not leave obsolete aliases or duplicate contracts.

For a v0.4.0 model, use `emhcl migrate -w <model>` within the permitted scope
before editing the migrated source. Read its report and review the result.
Remove migrated titles that only repeat the ID. Preserve meaningful titles.
For JSON input, use `emhcl import` to obtain a draft, then review it. Read every
conversion warning. Import does not preserve canvas IDs or invent missing
targets. Export is lossy. Do not use export and import as a repair loop.

## Encode the business model

Use stable lower-snake-case identities. Put canonical `event`, `aggregate`,
and reusable `field_type` blocks inside their owning `bounded_context`.
Represent external ownership with `external = true` on the supplied systems
and contexts. Use `title` only when the literal text differs from the ID.

Encode each selected capability as its matching workflow kind. Translation
uses `automation` over external facts, not a `translation` keyword. Preserve
the business model's pending-work read model. If the user requests only a
minimal legal automation, record any omission of that methodological element.
A client request or response uses a screen with `kind = "api"`. A reported
external fact uses an external event, not a pseudo-screen. Put a supplied API
route on the API screen as `api_endpoint`, not on the command as its reason.

Every `field` and `subfield` needs an explicit `type` (`EM009`). Use reusable
`field_type` declarations for recurring domain concepts, referenced as
`field_type.<context>.<id>` outside the owning context or `field_type.<id>`
inside it. Use built-in types for one-off fields. Encode identifiers, PII,
optionality, technical data, cardinality, and examples only when known.
Use `generated = true` only for system-filled values, never for user input.

Preserve field sources from the business model. Encode each one as a
`field_source` block in the workflow, with `description` for a derivation. A
`field` has no `description` attribute (`EM000`). Do not add a `mapping`
lineage language. The CLI accepts unchecked strings that do not establish a
source contract.

Encode the other method facts that the business model states with their HCL
forms: scenario `case`, `empty_history`, dispatch assertions and
`expect_no_dispatch`, aggregate `identity`, `lifecycle`, `evidence_source` and
`evidence`, `decision`, hotspot `kind`, `category`, `options`, and `concerns`,
and the key flags. Never invent one to fill a checklist.

Relationships are unquoted HCL traversals, not strings. Write an edge once:
use `to` on a workflow source, or `from` on the receiving workflow element
when a catalog event is the source. Use fully qualified context-owned
references outside the context and local references inside a workflow.
Do not write reverse edges such as `command.from = [screen...]`,
`screen.from = [readmodel...]`, or `processor.from = [readmodel...]`.

Keep scenarios inside their workflow. Follow the productions in the language
reference, not canvas notation. An automation action scenario has one `when`.
An automation view scenario has no `when` and states its read-model content.
Each step has exactly one typed target. State views have no query step.
Encode rejection as `then { error = "..." }` with supplied error wording.
A recorded business failure is an event. Use semantic `comment` blocks for
notes, not ordinary HCL comments.

Do not invent `linked_copy`, `element_copy`, `storyline`, `query`, `note`,
or a lineage DSL. Do not add implementation code,
database schemas, brokers, retries, or framework architecture unless asked.

## Format, validate, and repair

Prefer a compatible repository-local binary, then a PATH-installed binary.
Check `emhcl version`. v0.8.0 needs v0.17.0 or newer. A checkout's `dev` build
counts as current only if `emhcl version --format json` reports
`"spec": "v0.8.0"`. Do not install tooling or run untrusted downloads without
the user's request.

Run these gates in order, within the permitted edit scope. `MODEL` is one
file or folder. Each target completes only when its command exits 0:

```make
implementation-ready: valid
	emhcl validate --profile strict $(MODEL)
valid: formatted
	emhcl validate $(MODEL)
formatted:
	emhcl fmt -w $(MODEL)
```

Compatible releases accept folder formatting. When a whole-model format is
not authorized, format only permitted member files and use `fmt --check` on
the whole model. Do not claim the formatted gate if that check fails.
For review-only requests, use `emhcl fmt --check MODEL` instead of `fmt -w`.
Run read-only validation without attempting repairs. If a requested strict
diagnostic review follows a failed earlier gate, report its output without
claiming that the later gate completed.

A workshop HCL draft completes at `valid`. Strict validation treats an open
hotspot as an error. Keep genuine unknowns instead of inventing rules to pass.
Call a model implementation-ready only after the strict gate completes and
material business findings are resolved. After a source change, start again
at formatting. Stop later gates after a failure. Exit 1 means model problems.
Exit 2 means command-line misuse. `make model-ready MODEL=<path>` runs the
repository's equivalent graph but checks formatting instead of rewriting it.

Repair source errors, not diagnostic severity. Diagnostic families are
`EM0xx` structure, `EM1xx` references, `EM2xx` flow, `EM3xx` scenarios, and
`EM4xx` modeling judgment. Core errors apply in every profile. Profiles change
only `EM401`–`EM406`. Folder diagnostics name their member file. Board and
chapter codes include `EM015` missing or repeated board, `EM016` repeated
board workflow, `EM014` workflow in two chapters, and `EM005` empty chapter.
Method fact codes are `EM017` lifecycle, `EM018` aggregate identity, `EM202`
field source, `EM304` misplaced scenario attribute, and `EM305` contradictory
dispatch assertion.

Use a separate repair counter for each diagnostic code on each element.
After two unsuccessful repairs, stop editing for that code. Report its
location and both attempts. Only a new user-supplied rule or decision permits
one further repair. If that attempt fails, stop again. Never suppress the
code or invent a business rule. Check permission before each repair.

If no compatible CLI exists, complete the static checklist in the language
reference. State that machine validation did not run. Inspection is not
machine validation.

## Report observed results

Keep CLI diagnostics separate from business-review findings. Copy tool lines
unchanged. Only tool output uses `Error`, `Warning`, or an EM code. Write your
own file findings as `<file>:<line>:<column>: Review: <detail>`. For prose
input, use `input: Review: <detail>`. Do not count tool signals again by hand.
When strict diagnostics are requested, use `emhcl validate --profile strict`.

For file operations, start the final report with a `Key: Value` block. Repeat
`Ran:` for every `fmt`, `validate`, `import`, `export`, or `migrate` command in
execution order. Include the exit code and counts of every reported EM code,
including warnings:

```text
Model: <model.em.hcl>
Member-Files: none
Bounded-Contexts: <context>
Workflows: <workflow>
Open-Hotspots: none
Locked-Workflows: none
Gate: valid
Ran: emhcl fmt -w <model.em.hcl> => exit 0
Ran: emhcl validate <model.em.hcl> => exit 0
```

`Member-Files` lists folder members or `none`. `Gate` is the last completed
target: `none`, `formatted`, `valid`, or `implementation-ready`. It reports
only tool acceptance. List any remaining business findings after the block.
Without a compatible CLI, write `Gate: none` and
`Ran: none (no compatible emhcl; static checklist only)`.

For review-and-fix work, list resolved findings after the block, then add at
most a short paragraph. For explanation-only requests, answer directly without
pretending that commands ran or requiring a model-operation report.
