# Event Modeling HCL v0.2 — canonicalization, typed IR, formatter

## Context

The language is at **v0.2.0 (Draft)**. It already represents Event Modeling concepts
directly as HCL block kinds and validates strictly, but three things weaken it as an
input for humans, LLMs, and (soon) software-generating agents:

1. **Non-canonical flow.** One edge can be written from either endpoint
   (`screen.to=[command]` *and* `command.from=[screen]` both validate).
   `examples/complete.em.hcl` even declares the same edge twice. Multiple equivalent
   spellings hurt canonicality and make LLM generation non-deterministic.
2. **An artificial `query` step.** State View scenarios are forced into the
   GIVEN/WHEN/THEN shape with a synthetic `when { query = readmodel.x }`, which is not
   an Event Modeling concept. The correct form is `GIVEN event → THEN read model`.
3. **Boilerplate + no canonical semantic model.** `title` is mandatory everywhere even
   when it just re-humanizes the label, and there is **no AST/IR** — the validator is
   diagnostics-only and downstream tooling would have to re-read raw HCL.

v0.2 fixes these while preserving every design principle (block-kind = concept, label =
identity, source order = model order, typed traversals, strict validation, uncertainty
as hotspots). It does **not** redesign the language.

### Decisions locked with the user

- **Flow: hard break.** Reverse/duplicate flow forms become validation errors in v0.2.
  Examples migrated by hand; documented in a migration guide. Exactly one representation.
- **State View scenarios: require ≥1 `given` event.** Canonical `GIVEN event → THEN read
  model`; no `when`.
- **Scope: everything**, including the typed IR (CHANGE 4) and the formatter (CHANGE 7).

### Current architecture (verified)

- `internal/validator` — diagnostics-only, four files: `validator.go` (walk + rules),
  `schema.go` (body schemas), `references.go` (traversal resolution + `allowedFlowRoots`),
  `attributes.go` (literal/enum/example rules). No decode, no typed model.
- `cmd/eventmodeling-hcl/main.go` — hand-rolled CLI: `validate` / `version`; compact
  one-line diagnostic format asserted by `main_test.go`.
- No formatter exists. `hclwrite` ships with `hcl/v2` (already required).
- `hcl.Diagnostic.Extra interface{}` is free to carry a stable code.
- Gate suite (`make verify`) includes `exhaustive -default-signifies-exhaustive=false` —
  new IR enum switches must enumerate every case (no relying on `default`).

## Target architecture

```
Human / LLM
   │
   ▼  .em.hcl  (human-readable serialization; one canonical spelling per concept)
hclparse
   │
   ▼  semantic validation  ← the validation boundary (internal/validator, precise ranges)
   │
   ▼  internal/model.Load → *model.Model   (canonical typed IR; only valid models decode)
   ├── validate CLI
   ├── fmt CLI (formatter reads HCL directly via hclwrite; IR not required to format)
   └── future renderer / JSON / software-gen agents  ← consume IR, never raw HCL
```

**Why validation stays on HCL, not on the IR:** rewriting the working, well-tested
rule engine to run over a freshly-built IR is a second large rewrite with no diagnostic
benefit (HCL bodies carry the precise source ranges). Instead the validator remains the
boundary, and `model.Load` = *validate, then decode only if clean*. The IR is the
**normalized** artifact: one canonical flow-edge list, effective titles, presentation vs
semantic fields separated. Downstream tools reason from the IR. This satisfies CHANGE 4
without a premature compiler framework; documented in a new ADR.

## [x] Phase A — Core language semantics (TDD)

### A1. Canonical flow — hard break (CHANGE 1)
`internal/validator/references.go` `allowedFlowRoots`, remove the reverse entries so
each edge has exactly one spelling (source-local element uses `to`; catalog Event as
source uses `from` on the receiver):

| Keep (canonical) | Remove (now an error) |
| --- | --- |
| `screen.to=[command]`, `command.to=[event]` | `command.from=[screen]` |
| `readmodel.from=[event]`, `readmodel.to=[screen]` | `screen.from=[readmodel]` |
| `readmodel.to=[processor]`, `processor.from=[event]`, `processor.to=[command]`, `command.to=[event]` | `processor.from=[readmodel]`, `command.from=[processor]` |

- `validator.go` `workflowWarnings`: the "every command has a reason" check currently
  treats `command.from` as a reason. Canonically a command's reason is an incoming
  `screen.to`/`processor.to` (already caught by `incomingReferences`), `api_endpoint`,
  or `external_trigger`. Drop the `hasFrom` branch.
- `validateWorkflowExternality` reads `from` for external events — still correct
  (`processor.from=[event.partner.x]` remains canonical).

### A2. Remove `query`; pattern-specific `when` (CHANGES 2)
- Delete `query` from `schema.go` `scenarioStepSchema`, `attributes.go`
  `scenarioStepRule`, the `targets` slice + query→readmodel remap in
  `validateScenarioStep`, and the state_view `when` case in `scenarioTargetAllowed`.
  A stray `query = …` then fails as an unknown argument ("Unsupported"). ✔ CHANGE 2 goal.
- `validator.go` `validateScenario`: replace the universal `whenCount != 1` rule with
  per-pattern cardinality:
  - **state_change / automation / translation:** exactly one `when`; ≥1 `then`; `given` ≥0.
  - **state_view:** **zero** `when` (a `when` here is an explicit error — EM code below),
    **≥1 `given`**, **≥1 `then`**.
- New `scenarioTargetAllowed`:
  - state_change: given→event, when→command, then→event|error
  - state_view: given→event, then→readmodel|error (no when)
  - automation/translation: given→event|readmodel, when→processor|command, then→event|error
- `allowedScenarioTargets` target list drops `query`.

### A3. Derived titles (CHANGE 3)
- `schema.go`: change `title` from `Required:true` → optional in `boundedContextSchema`,
  `eventSchema`, `workflowSchema`, `elementSchema`, `tableSchema`, `actorSchema`,
  `ownerSchema`, `chapterSchema`, `screenImageSchema`, `scenarioSchema`. Keep
  `readmodel.question` and `actor.auth_required` required. Type-checking of a present
  `title` is unchanged.
- Title derivation lives in the IR (A/Phase C): `humanize(label)` title-cases the
  snake_case label (`pet_registered`→"Pet Registered"). IR carries `Title` (effective)
  and `TitleExplicit bool`; the derived title is **never** serialized back to source.
- Tests: a titleless block validates; IR derives the title; explicit `title` overrides.

**Phase A tests** update the existing fixtures that used reverse flow / `query` (these
are intentional breaking changes): `hcl_native_test.go`
`AcceptsTypedHCLReferences` (drop `command.from`), `AcceptsExplicitTranslationWorkflow`
(`processor.to=[command]`, drop `command.from`), `AcceptsPatternSpecificScenarios`
(state_view: drop `when{query}`), and `RejectsScenarioTargetForWrongPattern` (now
asserts "when not allowed in state_view"). Add: state_view given+then valid; state_view
with `when` rejected; `query=` rejected as unknown; titleless-block valid;
explicit-over-derived title.

## [x] Phase B — First-class diagnostics with stable codes (CHANGES 6, 5)

- New `internal/validator/diagnostics.go`: `EMxxx` code constants grouped by category —
  `EM0xx` structural, `EM1xx` reference resolution, `EM2xx` reference-kind/flow, `EM3xx`
  scenarios, `EM4xx` judgment smells (warnings). A `diag(code, sev, subject, summary,
  detail)` constructor stores the code in `Diagnostic.Extra` (a small `diagCode` type).
  Migrate every `errorDiagnostic`/`warningDiagnostic` call site to a coded constructor.
- Upgrade the Detail text on the flagship diagnostics to the "expected / found" shape,
  e.g. flow-reference-kind mismatch: `readmodel.from may reference: event.<context>.<id>.
  You referenced: command.create_pet.`; scenario when-in-state-view; wrong scenario target.
- `cmd/.../main.go` `formatDiagnostic`: prefix the code —
  `file:line:col: error EM203: Summary: Detail`. Keep the compact one-line shape so
  `main_test.go` stays meaningful (update its expected strings to include codes).
- **Validation profiles (CHANGE 5):** `Profile` enum {Workshop, Valid, Strict}, default
  **Valid** (= today's behavior). A per-code severity map decides escalation for a *small*
  set: `Strict` escalates "every command has a reason" (EM4xx→error) and open `hotspot`s
  to build-blocking; the bed/left-chair/right-chair/shelf smells stay warnings in every
  profile (pure judgment). `Workshop` keeps smells informational. Wire `validate
  [--profile workshop|valid|strict]`; document the lifecycle intent even though the
  escalating set is deliberately tiny for now. Uncertainty is represented as hotspots,
  never by weakening a rule.

## [x] Phase C — Canonical typed IR (CHANGE 4)

New package `internal/model`:
- Types (exported, JSON-tagged): `Model{Title, Version, Actors, Owners, Contexts,
  Workflows, Chapters, Hotspots, Edges}`; `Context{Aggregates, FieldTypes, Events}`;
  `Workflow{Kind, ID, Title, TitleExplicit, Status, Owner, Elements, Scenarios}`;
  `Element{Kind, ID, Title, TitleExplicit, Semantic{…}, Presentation{sketched, prototype,
  list_element, group_id, tags, …}, Fields, From, To}`; `Field`, `Scenario{Steps}`,
  `Step{Kind, Target, Ref, …}`, `Hotspot`, etc.
- Enums as typed constants with exhaustive switches: `WorkflowKind{StateChange, StateView,
  Automation, Translation}`, `ElementKind{Command, ReadModel, Screen, Processor,
  ScreenImage, Table}`, `StepKind{Given, When, Then}`, `Profile`.
- `Load(filename string, src []byte, p Profile) (*Model, hcl.Diagnostics)` — runs the
  validator; if `!HasErrors()`, decodes with the public HCL API (`PartialContent`,
  `hcl.ExprList`, `hcl.AbsTraversalForExpr`) plus small local literal readers.
- **Normalization at the IR boundary:** flow is stored as a single canonical `Edges`
  list (source-node → target-node), so downstream never sees `from`-vs-`to`. Titles are
  resolved via `humanize`. Presentation vs semantic attributes (CHANGE 8) are split into
  sub-structs — *no surface syntax change*, only IR grouping + spec documentation.
- Tests: decode `examples/complete.em.hcl` into the IR; assert workflow kinds, canonical
  edges, derived vs explicit titles, scenario steps, presentation/semantic split.

## [x] Phase D — Canonical formatter (CHANGE 7)

New package `internal/formatter` + `fmt` subcommand:
- Base pass: `hclwrite`-format the token stream (indentation, whitespace, list layout,
  trailing newline).
- Canonical **attribute ordering within each block**, preserving block order (blocks —
  scenarios, fields, given/then, workflows — carry semantic order and are **never**
  reordered): metadata → ownership/status → semantic config → relationships (`from`,`to`)
  → then nested blocks in source order. Reorder via `hclwrite` by rebuilding each block
  body's attributes with `SetAttributeRaw(name, expr.BuildTokens(nil))` so unquoted
  traversals and complex expressions are preserved verbatim.
- CLI: `fmt [-w] <model.em.hcl>` — stdout by default, in-place with `-w`.
- **Idempotence tests:** `fmt(fmt(src)) == fmt(src)` over every `examples/*.em.hcl`; a
  messy-input fixture formats to the canonical shape.

## [x] Phase E — Migration of shipped models + tests

Hard break, so migrate every model to canonical v0.2:
- `examples/`: `complete.em.hcl` (drop redundant translation `command.import_pet.from`;
  move automation `processor.from=[readmodel]` → `readmodel.to=[processor]`),
  `minimal.em.hcl` (drop `command.from`), `pet-management-detailed.em.hcl` (4 state_view
  scenarios: drop `when{query}`, add a `given` event each from the read model's source;
  fix reverse flow), `course_subscriptions.em.hcl`, `appointment-weather-patterns.em.hcl`.
  Drop a few now-redundant titles to demonstrate derivation; keep some explicit to show
  override. Then run `fmt -w` on all to lock canonical formatting.
- `testdata/valid/*.em.hcl`: same migration. `testdata/invalid/*`: add fixtures for a
  reverse-flow edge and a state_view `when`.
- `docs/migration-v0.2.md`: before/after for each breaking change (flow direction, `query`
  removal, state_view `when`→`given`, optional titles).

## [x] Phase F — Specification + docs (normative)

- `eventmodeling.hclspec.md` → **v0.2.0 (Draft)**: canonical flow rules table (one row per
  edge), pattern-specific scenario grammar (state_view has no `when`, ≥1 given),
  title-default behavior, three reference scopes (domain/catalog · workflow-local ·
  workflow-qualified — CHANGE 9), the one canonical field syntax with explicit rejection
  of `fields={…}` shorthand (CHANGE 10), a semantic-vs-presentation attribute
  classification table (CHANGE 8), the conservative attribute-vocabulary note keeping
  `aggregate_dependencies`/`technical_attribute`/`id_attribute`/`external_trigger` with
  justification (CHANGE 11), the chapters trade-off left unchanged (CHANGE 12), diagnostic
  codes + profiles, and a migration/compatibility section. Add the canonical IR pipeline
  description.
- `schema/eventmodeling.hcl.schema.md`: remove `query`; update flow + scenario tables;
  bump version.
- `docs/adr/0005-canonical-flow-and-typed-ir.md`: record hard-break flow, query removal,
  required state_view given, derived titles, and the validate→decode IR boundary.
- `README.md`: v0.2.0; `fmt` + `--profile` usage. `CHANGELOG.md`: v0.2.0 section
  (Added: fmt, diagnostic codes, profiles, typed IR; Changed: canonical flow, optional
  titles, state_view scenarios; Removed: query, reverse flow forms).
- `cmd/.../main.go`: `usageMessage`, `parseCommand` (`fmt`, `--profile`), `main_test.go`.

## Breaking changes (v0.1 → v0.2)

1. **Reverse flow spellings are errors** — `command.from=[screen]`,
   `screen.from=[readmodel]`, `processor.from=[readmodel]`, `command.from=[processor]`.
   Use the source-endpoint `to` form.
2. **`query` is removed** — State View scenarios use `given { event = … }` +
   `then { readmodel = … }` with **no `when`**; a `when` in a State View is an error.
3. **State View scenarios require ≥1 `given` event.**
4. `title` becomes optional (derived from the label); not itself breaking, but shipped
   examples change.
5. Diagnostic text now carries `EMxxx` codes (affects anyone scraping messages).

## Verification

- `make verify` green — `fmt-check`, `tidy-check`, `vet`, `test`, `test-race`,
  `staticcheck`, `exhaustive` (new IR enum switches exhaustive), `vulncheck`,
  `validate-examples`.
- `go run ./cmd/eventmodeling-hcl validate examples/*.em.hcl` → all valid; a reverse-flow
  and a state_view-`when` fixture → exit 1 with the right `EMxxx` code.
- `go run ./cmd/eventmodeling-hcl fmt examples/complete.em.hcl` idempotent (piped through
  twice, byte-identical); `fmt -w` leaves examples unchanged after Phase E.
- `validate --profile strict` escalates a command-without-reason model to an error;
  `--profile valid` (default) keeps it a warning.
- Unit tests green for: canonical-flow accept/reject, `query` rejected, state_view
  given+then valid / `when` rejected, derived vs explicit titles, IR decode of
  `complete.em.hcl`, formatter idempotence, diagnostic codes on flagship diagnostics.
