# Architecture Guide

This guide describes how `emhcl` is organized. It also shows how the design uses Axiomatic Design to keep changes predictable. The guide describes the current repository. The language contract is in the [Event Modeling HCL specification](https://github.com/event-modeling-hcl/spec). The tool implements Specification v0.8.0: a model is one file or one folder, a `board` block sets the story order, file layout has no meaning, an `automation` scenario has an action form and a view form, a screen has a kind, `"ui"` or `"api"`, and a model can state the method facts of RFC 0007.

The [v0.5.0 review record](v0.5.0-review.md) records the status before the fixes, the conformance defects that the review reproduced, and the difference between the local diagonal authoring controls and the decoupled semantic pipeline. The section [Known deviations](#known-deviations) lists the places where the current code does not follow this guide.

## Axiomatic Design terms

Axiomatic Design separates what a system must do from the mechanisms that do it. This guide uses these terms:

- A need is an outcome that a user or a maintainer expects.
- A functional requirement (FR) is a testable outcome that does not name a mechanism.
- A design parameter (DP) is the mechanism that satisfies an FR.
- The Independence Axiom states that each FR must stay controllable by its own DP.
- The Information Axiom prefers the design with the highest probability of meeting all FRs.

An influence matrix shows which DP changes which FR. Each row is an FR and each column is a DP. `X` means that a change to the DP changes the FR. `0` means that the DP has no material influence on the FR in the supported range. The supported range is one model per run. A model is one file or one folder of files.

A design is uncoupled when the matrix has `X` only on the diagonal. A design is decoupled when the matrix is triangular. A decoupled design is valid when you set the DPs in the order of the matrix. A design is coupled when no order makes the matrix triangular.

Information content is `I = -log2(p)`, where `p` is the probability of meeting the requirement. Source lines and package counts do not measure it.

## Needs, requirements, and parameters

| Need | Functional requirement | Acceptance criterion | Design parameter |
| --- | --- | --- | --- |
| N1: Authors receive feedback that they can trust | FR1: Parse native `.em.hcl` syntax | Syntax errors keep their HCL source locations | DP1: `internal/syntax` grammar and parser |
| N2: Every frontend sees the same model | FR8: Load a model from a file or a folder | A path to a file or a folder gives the same ordered files every time, and each diagnostic names its own file | DP8: Model loader (`app.ReadModel`, `syntax.ParseFiles`, `syntax.Document.SourceOf`) |
| N1 | FR2: Interpret references in the same way in every stage | One owner turns a reference expression into an address | DP2: `internal/source` document wrapper and reference helpers |
| N1 | FR3: Enforce language and modeling rules | Core rules give the same diagnostics in every profile. The profile sets only the severity of judgment signals. | DP3: `internal/validator` core passes and policy evaluator |
| N2: Every frontend sees the same model | FR4: Construct one canonical model | Only validated input reaches model construction | DP4: `internal/model` lowering |
| N3: Users can consume models in useful forms | FR5: Format or render the same output for the same input | Formatting and HTML generation do no OS actions | DP5: `internal/formatter` and `internal/renderer` |
| N5: Users exchange models with Event Modeling JSON tools | FR7: Convert between the canonical model and slice-based JSON | Each conversion gives a schema-valid document or a native model, or it fails with a stated reason | DP7: `internal/interchange` |
| N4: The CLI, the browser, and the server agree | FR6: Compose and expose the use cases | Each entry point delegates to one application service | DP6: `internal/app` and thin runtime adapters |
| N6: Authors move v0.4.0 models to the current language | FR9: Rewrite a v0.4.0 model without changing its meaning | The rewritten model passes the current core rules, or nothing is written | DP9: `internal/migrate` with `app.MigratePath` |

Slice-based JSON is the document format that the [published Event Modeling JSON schema](https://github.com/dilgerma/event-modeling-spec/blob/main/eventmodeling.schema.json) defines. A slice is one JSON entry that matches one native workflow.

## Independence matrix for the tool

| FR \ DP | DP1 Syntax | DP8 Loader | DP2 Source | DP3 Validation | DP4 Model | DP5 Output | DP7 Interchange | DP9 Migration | DP6 Adapters |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| FR1 Parse syntax | X | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| FR8 Load a model from a file or a folder | X | X | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| FR2 Decode source facts | X | X | X | 0 | 0 | 0 | 0 | 0 | 0 |
| FR3 Enforce validity | X | X | X | X | 0 | 0 | 0 | 0 | 0 |
| FR4 Construct canonical model | X | X | X | X | X | 0 | 0 | 0 | 0 |
| FR5 Produce the same output | X | X | X | X | X | X | 0 | 0 | 0 |
| FR7 Convert to and from JSON | X | X | X | X | X | 0 | X | 0 | 0 |
| FR9 Preserve a migrated model | X | X | X | X | 0 | 0 | 0 | X | 0 |
| FR6 Expose consistent operations | X | X | X | X | X | X | X | X | X |

The matrix is lower triangular, so the design is decoupled. The control sequence for the native pipeline is:

```text
app.ReadModel
  -> syntax.ParseFiles
  -> source.Decode
  -> validator.ValidateDecodedDocument
  -> model.Build
  -> renderer.Render
```

`validator.ValidatedDocument` enforces this sequence. `model.Build` does not accept input that was only parsed. Formatting is a separate branch. It needs HCL that parses, but it does not need a valid model.

Validation has two parts. The core passes (`validateBody`) run the language rules and report the same diagnostics in every profile. The policy evaluator (`applyProfile`) then assigns a severity to each judgment signal (EM401–EM406) for the selected profile and changes nothing else. There are three profiles: `workshop`, `valid`, and `strict`. `ValidateDecodedDocument` returns a `ValidatedDocument` only when the core passes report no error. Thus the profile never decides whether a model exists. `app.Diagnostic.Judgment` marks the judgment signals, and `Diagnostics.HasCoreErrors` tells adapters whether a model, HTML, or JSON was produced. `Diagnostics.HasErrors` decides the exit code. The CLI commands `diagram` and `export` always use the `valid` profile. Only `serve --profile` and the WASM render function accept another profile. With `strict`, they report the rejection and still return the HTML.

The migration branch is separate:

```text
app.MigratePath
  -> app.ReadModel
  -> migrate.Migrate (v0.4.0 reading on raw HCL)
  -> app.ValidateFiles (current rules, profile valid)
```

`internal/migrate` is the only package that reads v0.4.0 rules: the effective chapter and file order of workflows, typeless field resolution, derived titles, and the `translation` keyword. It works on `hclsyntax` and `hclwrite` and imports only `internal/syntax`, so no v0.4.0 rule can reach the validator, the model, or a renderer. It refuses a v0.4.0 chapter range and a chapter that is spread over several files. Edited files keep their comments. Files that need no edit are returned unchanged. `MigratePath` returns the files only when they have no core error.

The important zeros go in one direction. A change to the renderer or the CLI cannot change what syntax the parser accepts. A change to the model or the renderer cannot make invalid input valid. Adapters do not define a second diagnostic policy or a second rendering policy.

The DP8 column has an `X` in every row from FR8 down, and a `0` in the FR1 row. The FR8 entry is the diagonal. FR1 has a `0` because the syntax rules do not depend on which files the loader finds. The other entries are real influences:

- FR2: The loader decides which files `syntax.Document.Body()` merges with `hcl.MergeFiles`. `source.Decode` only wraps that document, but every reference that DP2 interprets comes from the merged body.
- FR3: No rule depends on the number of files. One-file and folder models use the same rules. The byte order of member files fixes only three things: which declaration a duplicate diagnostic reports as the first one, where a repeated `board` is reported, and where a missing `board` (EM015) is reported. A missing board is reported at the start of the first file.
- FR4: The model does not depend on file order. `Model.Workflows` is the board sequence, then the off-board workflows (`Model.OffBoard`) in ID byte order. Catalog items, chapters, chapter members, hotspots, lifecycles, evidence sources, evidence, decisions, workflow elements, tables, screen images, and scenarios are sorted by ID. The states, transitions, and invalid transitions of a lifecycle are sorted by ID. The field sources of a workflow are sorted by the text of the target reference. Elements sort by kind and then ID. Scenario step order and explicit reference lists (`to`, `from`, the board) keep their source order. The fields of an element are the entries of the `fields` list in source order, then the `field` blocks in source order. Edges follow the workflow order, then element ID order. Inside one element, all `to` edges come before all `from` edges, each in list order. An edge whose source is an event of an external bounded context has `External` set. `TitlePresent` distinguishes omission from a literal empty `Title`. No title is derived.
- FR5, FR7, and FR6: These rows depend on DP8 through DP2, DP3, and DP4, in the same way that the guide treats DP1.

FR9 is included even though migration is separate from the native pipeline. DP9 reads raw syntax through DP1 and DP8 and checks its output through DP2 and DP3. The application adapters expose migration, so FR6 also depends on DP9. The reverse influence is absent: migration does not alter the native validator's acceptance rules.

DP8 comes after DP1 in the build order because `syntax.File` and `syntax.ParseFiles` belong to the syntax package.

The FR7 row has `X` under DP1 to DP4 for two reasons. Export reads only the canonical model. Import must write HCL that the grammar and the validator accept. The `0` under DP5 is correct because `interchange.Import` returns unformatted source. `internal/app` formats it afterward. The `0` under DP6 is correct because the conversion result is the same for every entry point.

## Interchange sequence

`internal/app` runs each conversion in one fixed sequence.

Export:

```text
app.ValidatedModel (profile valid)
  -> interchange.Export
  -> interchange.MarshalDocument (schema check)
```

Import:

```text
interchange.ParseDocument (schema check, then decode)
  -> interchange.Import
  -> app.Format
  -> app.Validate (profile valid)
```

Import sends its result through the native formatter and the native validator. Thus `internal/validator` stays the only owner of model validity. `internal/interchange` depends only on `internal/model` and the HCL libraries.

## Requirements for the interchange module

FR7 has six child requirements. The table gives each one its mechanism.

| Child FR | Acceptance criterion | DP | Location |
| --- | --- | --- | --- |
| FR7.1: Accept and write only schema-valid documents | Import accepts a document only when `ajv` accepts it. Export applies the same rules to its output. | DP7.1: Schema check | `conform.go`, `decode.go` |
| FR7.2: Map JSON identities to stable native labels | One JSON context gives one native context. An event without an `id` keeps its own flows. | DP7.2: Identity mapping | `import.go` context and event-occurrence maps |
| FR7.3: Keep JSON values unchanged | Object keys keep their text and their order. Keys such as `for` and `a-b` produce valid HCL. | DP7.3: Value writer and reader | `jsonTokens` in `import.go`, `jsonValue` in `internal/model` |
| FR7.4: Write only flows that the grammar allows | Import writes no illegal flow. Each omitted flow has a warning. | DP7.4: Flow and scenario policy | `allowedFlowRoots` and `placeFlows` in `import.go`, `workflowAllows` and `scenarioAllows` in `import_policy.go` |
| FR7.5: Keep at most one actor per slice | Import refuses a slice with two or more actors. Export refuses a workflow whose screens name two or more actors. | DP7.5: Actor gate | `rejectMultipleActors` in `import.go`, `slice` in `export.go` |
| FR7.6: Disclose each loss | Each value that a conversion drops produces a warning. | DP7.6: Warnings at the point of loss | `warn` calls in `import.go`, `import_policy.go`, `export.go` |

The actor gate follows from the language. Specification v0.8.0 gives a screen one optional actor and gives a workflow no actor attribute. A JSON slice lists actors without links to screens. Thus a slice with two actors has no correct native form. The gate stops the conversion instead of guessing. If only some screens of a workflow name the actor, export drops the actor and writes a warning.

The JSON schema has no field for the screen kind. Export writes an API screen as a `SCREEN` with `apiEndpoint` and a warning. Import writes no `kind`, so the screen is a `"ui"` screen. The JSON round trip stays stable, because the JSON never holds the kind.

The JSON schema also has no field for the method facts of Specification v0.8.0. Export writes a warning and drops aggregate `identity`, lifecycles, evidence sources, evidence, decisions, `field_source` blocks, key flags, and scenario `case`. It omits an `empty_history` scenario, because import rejects a view specification without given steps. It omits a `then` step with `expect_no_dispatch`. When that leaves no `then` step, it omits the scenario. Import accepts a `then` command (`SPEC_COMMAND`) in an automation specification whose one `when` step resolves to a processor. This is a copy of the validator rule (change rule 11).

The native language has no translation keyword. Export writes slice type `TRANSLATION` for an `automation` that has at least one edge with `External` set, and `AUTOMATION` otherwise. Import writes `TRANSLATION` as `automation`. Import writes a `board` that lists the slices in the order of their `index` value. Slices without an `index` come last, in document order. Export writes `index` as the position of the slice. Export writes off-board workflows after the board workflows, in ID order, with a warning.

Round-trip stability is an acceptance criterion, not a separate FR. A round trip is export, then import, then a second export. The round trip is stable when the second export decodes to the same document as the first. The tests compare the decoded documents, not the bytes. Stability is the result of FR7.1 to FR7.6 together, so it has no DP of its own.

## Independence matrix for the interchange module

| FR \ DP | DP7.1 Schema | DP7.2 Identity | DP7.3 Values | DP7.4 Flows | DP7.5 Actors | DP7.6 Warnings |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| FR7.1 Schema-valid documents | X | 0 | 0 | 0 | 0 | 0 |
| FR7.2 Stable labels | X | X | 0 | 0 | 0 | 0 |
| FR7.3 Unchanged values | X | 0 | X | 0 | 0 | 0 |
| FR7.4 Legal flows | X | X | 0 | X | 0 | 0 |
| FR7.5 One actor per slice | X | 0 | 0 | 0 | X | 0 |
| FR7.6 Disclosed losses | X | X | X | X | X | X |

This matrix is also lower triangular, so the module is decoupled. Set the DPs in this order: schema check, identity mapping, value writer, flow policy, actor gate, warnings.

The first column has `X` in every row because each later step reads only the decoded document. The repository gave evidence for the FR7.3 entry. `ParseDocument` once encoded the checked document a second time. That step sorted the keys in every prototype. It now decodes the original bytes.

FR7.4 depends on DP7.2 because each flow refers to identities that DP7.2 resolves. An empty event `id` once sent flows to the wrong event. Event occurrences now have their own map.

FR7.6 depends on every DP, because each DP decides what it drops. For this reason, each DP writes its own warning where the loss occurs. No separate step tries to find losses later.

The zeros in the FR7.3 row are correct. Labels and flows do not change the content of a value.

## Influences across branches

Two influences cross from other branches into FR7. Each one has a cost when it changes.

DP3 influences FR7.4. The interchange module repeats three flow rules of `internal/validator` so that import writes no illegal flow:

- `allowedFlowRoots` in `import.go` repeats the flow grammar in `validator/references.go`.
- `workflowAllows` in `import_policy.go` repeats the workflow element rules in `validator/structure.go`.
- `scenarioRules` and `scenarioAllows` in `import_policy.go` repeat `scenarioRules` and `scenarioTargetAllowed` in `validator/scenarios.go`. Both copies select the form of an `automation` scenario by the count of `when` steps. A scenario without a `when` step uses the view form, which has the step rules of a `state_view` scenario. Both copies also refuse a view form read model of another workflow: the validator with `EM101`, the importer with a warning. Both copies accept a `then` command step only in an action form scenario whose one `when` step targets a processor.

No shared code ties these copies together. Two tests tie the scenario copy partly. The round-trip test exports and imports `examples/pet-clinic`, which has a view form scenario, so a refused scenario would change the second export. `TestImport_AutomationViewFormSpecifications` checks that the importer keeps a view form scenario, omits a step that targets a read model of another slice, and omits a view form scenario without a `given` step. If you change one of these rules in the validator, you must change its copy in the same change.

DP4 influences FR7.3. Export reads JSON values from the canonical model. `jsonValue` in `internal/model` builds the keys and the structure of each value from the HCL source text, which it reads with `syntax.Document.SourceOf`. Keys do not go through `cty`, because `cty` normalizes Unicode keys. Normalization would merge the keys `é` and `e\u0301`. Scalar values still go through `cty` (see [Information Axiom and evidence](#information-axiom-and-evidence)). Export writes an example that is not a string or an object as JSON text (`exportExample` in `export.go`).

`internal/serve` imports `internal/syntax` only for the `syntax.File` type that `app.ReadModel` returns. It hashes those files to detect a change. It does not parse or select member files, so the loader rules R1 to R4 keep one owner.

## Requirements for the diagram viewer

FR5 includes the HTML diagram. `internal/renderer` changes the canonical model into a view model (`BuildViewModel` in `adapter.go`). It writes one file that holds the view model as JSON and the viewer scripts in `internal/renderer/assets`. The browser runs the viewer. The viewer has nine child requirements.

| Child FR | Acceptance criterion | DP | Location |
| --- | --- | --- | --- |
| FR5.1: Scope each page to one chapter or to the whole model | A fragment gives the same chapter every time. Slices follow the board. Board workflows outside every chapter appear in a "Not in a chapter" UI group, not as a chapter. Off-board workflows never enter the story and are listed separately. | DP5.1: Route grammar and load-time scoping | `parseHash`, `normalizeChapters`, and `MODEL` at the top of `viewer.js` |
| FR5.2: Give every view the same read-only facts | Each view reads elements, edges, labels, and the slice timeline from one place | DP5.2: Shared index and helpers | `ELEMENTS`, `EDGES`, `NEIGHBORS`, and `EMC.sliceTimeline` in `viewer.js` |
| FR5.3: Show the details of one slice on request | The drawer shows the scenarios and elements of the slice that the reader selects | DP5.3: Detail drawer | `openSlice` in `viewer.js` |
| FR5.4: Filter the boards by status and bounded context | One filter state applies to every board. The controls offer only values on the current board. | DP5.4: Filter owner | `filterState`, `applyFilters`, and `EMC.onFilterChange` in `viewer.js` |
| FR5.5: Draw the Model, Compact, Storming, and Context Map boards | Each board renders on its own. The Model, Compact, and Storming boards apply the filter state. The Context Map shows no filter bar and no drawer. | DP5.5: Board renderers | `renderEventModel` in `viewer.js`, `viewer.compact.js`, `viewer.eventstorming.js`, `viewer.contextmap.js` |
| FR5.6: Preview every chapter | Each card shows the first and last slice of its chapter | DP5.6: Chapter overview | `viewer.chapters.js` |
| FR5.7: Present one slice to readers who do not know Event Modeling | The slide shows the screen, then the examples, then the open questions, then the event model of the slice | DP5.7: Slides | `viewer.slides.js` |
| FR5.9: List the off-board workflows | The `#offboard` page lists each off-board workflow with its timeline and opens its details on request | DP5.9: Off-board page | `renderOffBoard` and `openSliceDetail` in `viewer.js` |
| FR5.8: Navigate between pages and views | Each fragment shows one page. Browser Back works. Links work inside the sandboxed `srcdoc` frame of the playground. | DP5.8: Navigation | `parseRoute`, `renderNavigation`, `showPage`, and the fragment click handler in `viewer.js` |

## Independence matrix for the diagram viewer

| FR \ DP | DP5.1 Scope | DP5.2 Index | DP5.3 Drawer | DP5.4 Filters | DP5.5 Boards | DP5.6 Overview | DP5.7 Slides | DP5.9 Off-board | DP5.8 Navigation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| FR5.1 Scope | X | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| FR5.2 Shared facts | X | X | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| FR5.3 Detail drawer | X | X | X | 0 | 0 | 0 | 0 | 0 | 0 |
| FR5.4 Filters | X | X | 0 | X | 0 | 0 | 0 | 0 | 0 |
| FR5.5 Boards | X | X | X | X | X | 0 | 0 | 0 | 0 |
| FR5.6 Overview | X | X | 0 | 0 | 0 | X | 0 | 0 | 0 |
| FR5.7 Slides | X | X | 0 | 0 | 0 | 0 | X | 0 | 0 |
| FR5.9 Off-board | X | X | X | 0 | 0 | 0 | 0 | X | 0 |
| FR5.8 Navigation | X | 0 | 0 | 0 | X | X | X | X | X |

The matrix is lower triangular, so the viewer is decoupled. FR5.9 keeps its number, but its row comes before FR5.8 because navigation depends on it. Set the DPs in this order: scope, shared index, drawer, filters, boards, overview, slides, off-board page, navigation.

The off-diagonal entries are these influences:

- DP5.1 influences FR5.2 to FR5.9 through three values. The boards, the slides, and the shared index read the scoped `MODEL`. The overview reads `FULL_MODEL` and the chapter list from `normalizeChapters`. The off-board page reads `OFF_BOARD`. Navigation reads the route grammar of DP5.1.
- The boards open the drawer when the reader clicks a slice, and the Model, Compact, and Storming boards apply the filter state of DP5.4.
- The overview, the slides, and the off-board page draw a slice with `EMC.sliceTimeline` from DP5.2. The off-board page opens the drawer of DP5.3 through `openSliceDetail`.
- Navigation calls the render function of the page that it shows, so DP5.5, DP5.6, DP5.7, and DP5.9 influence FR5.8.

The important zeros are these:

- No board influences another board. Each board registers its own filter function with `EMC.onFilterChange` when it renders. A Storming page that you open directly does not render the Model board.
- The slides do not use the filter state. They show every fact of one slice.
- The overview does not influence the slides. The slice timeline belongs to DP5.2, not to the overview.
- Navigation does not change what a view draws. It only selects the page and passes the route values, such as the slide number.

The viewer had three couplings that this decomposition removed:

- The filter state and its controls were inside `renderEventModel`. That function also called the filter functions of the Storming and Compact boards. Thus FR5.4 depended on the Model board, and the Storming and Compact pages rendered the Model board first. Now DP5.4 owns the state and the controls, and each board registers itself.
- The slides used a function that the overview script exported. A change to the overview changed the slides. Now both use `EMC.sliceTimeline` and the shared `.timeline` styles.
- Two regular expressions defined the route grammar: one for load-time scoping and one for navigation. A new route needed two changes. Now `parseHash` is the only grammar.

Chapter scoping is set when the page loads, before every other DP. When the reader opens another chapter, navigation reloads the page at the new fragment. This reload is the control junction that keeps the decoupled order. A change of view inside the same chapter does not reload the page.

Some residual shared items are intentional:

- Each board places hotspots by its own layout rules, but DP5.1 decides which hotspots are in scope. The adapter gives a workflow hotspot the target ID `slice__<id>`, and scoping and each board depend on this text.
- Each view writes its own entry in the shared `LEGENDS` object.
- The drawer and the popovers both close on Escape.
- The board scripts set `window.render*` and `window.relayout*` functions that `viewer.js` calls by name. These names work only because `viewer.js` loads first.
- The resolved and open hotspot styles have one owner: the `--hot-resolved-*` colors and the `.is-resolved` class in `viewer.css`. The status check that selects them is copied into three boards (see [Known deviations](#known-deviations)).

The slides list every hotspot in scope under "Open questions", resolved hotspots too, with the status as plain text. The Compact board does not show a hotspot that it cannot pin. The Model and Storming boards list such hotspots in their legend.

## Requirements for the tool contracts

[RFC 0001](rfcs/0001-machine-readable-tool-contracts.md) adds one need: N8, a program that calls `emhcl` gets results with a stable contract. The agent skill, the Makefile targets, CI jobs, and the planned Claude plugin are such programs. The text output and the exit codes of existing calls do not change.

The need adds one child of FR2 and five children of FR6:

| FR | Acceptance criterion | DP |
| --- | --- | --- |
| FR2.1: Turn a source range into the address of a declaration | Each reference form gets its innermost declaration. A chapter, an evidence record, the board, and the model start (`Document.Start`) get no address. A hotspot gets `hotspot.<id>`, and a state gets `state.<lifecycle>.<id>`. An offset in one member file never matches a block of another file. | DP2.1: `source.Address`, `source.ElementHasAddress`, and `(*source.Document).AddressAt`, with `syntax.BlockRange` for the extent of a block |
| FR6.1: Report diagnostics as data | For every model in `examples/` and `testdata/` and each profile, `validate --format json` gives the text diagnostics in the same order | DP6.1: The JSON tags of `app.Diagnostic`, `Diagnostics.MarshalJSON`, and the one `--format` parser in `internal/cli` |
| FR6.2: Compare two models by declaration | A block in another member file or a new layout gives no entry. Each edit gives exactly its entry. | DP6.2: `internal/compare`, `app.DiffPaths`, and the `diff` command |
| FR6.3: Report the layout of each member file | `fmt --check` names exactly the member files that differ from `fmt` output | DP6.3: `app.FormatPath`, which reads the members through `app.ReadModel`, and the `--check` option |
| FR6.4: Show a title that the caller supplies | The masthead shows the `--title` text. Without the option, the title is `renderer.ModelName` of the path. | DP6.4: The `title` input of `renderer.Render`, `app.RenderPathTitled`, and `--title` |
| FR6.5: Report the versions as data | `spec` equals `version.Spec`, and `tool` equals the text output | DP6.5: `version --format json` |

| FR \ DP | DP2.1 Address | DP6.1 Diagnostics | DP6.2 Diff | DP6.3 Format | DP6.4 Title | DP6.5 Versions |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| FR2.1 Address | X | 0 | 0 | 0 | 0 | 0 |
| FR6.1 Diagnostics as data | X | X | 0 | 0 | 0 | 0 |
| FR6.2 Compare models | X | X | X | 0 | 0 | 0 |
| FR6.3 Layout of member files | 0 | 0 | 0 | X | 0 | 0 |
| FR6.4 Diagram title | 0 | 0 | 0 | 0 | X | 0 |
| FR6.5 Versions as data | 0 | X | 0 | 0 | 0 | X |

The matrix is lower triangular. Set DP2.1 first, then DP6.1, then the other DPs in any order. The off-diagonal entries are these influences:

- FR6.1 depends on DP2.1 for the `address` key.
- FR6.2 depends on DP2.1 for the address strings. It depends on DP6.1, because `diff` writes diagnostics with the same encoding and reads the same `--format` option.
- FR6.5 depends on DP6.1, because `version` reads the same `--format` option.

The important zeros are these:

- FR6.3 has `0` under DP6.1. `fmt` has no `--format` option.
- FR6.4 has `0` under DP6.2. The title is an input of one rendering and is not part of the model, so a title never appears as a change.
- FR2.1 has `0` under every DP6 column. The address exists before an adapter reads it.

The new FR6 rows also depend on DPs of other branches. FR6.2 depends on DP4, because it compares canonical models. FR4 already makes the model independent of file names and block positions, so `diff` reports no layout change. FR6.3 depends on DP5 and DP8. FR6.4 depends on DP5.

One existing row depends on DP2.1. FR3 has an `X` under DP2.1, because `validator.referenceExists` reads `source.ElementHasAddress`. FR3 already depends on DP2, and DP2.1 is part of DP2, which comes before DP3. Thus the list of element kinds has one owner, and the order stays the same. The DP6.1 to DP6.5 columns have no `X` in the rows FR1 to FR5 or FR7 to FR9. The tool matrix stays lower triangular.

The change removed two copies of a fact. The Makefile no longer copies loader rule R2 to find member files. `cmd/wasm` no longer lists the diagnostic keys in its own map.

Two risks remain:

- `internal/compare` lists the collections of `model.Model`. `TestModels_CoversEveryTopLevelCollectionOfTheModel` fails when the model gets a collection that `compare` does not read.
- `AddressAt` treats the empty range at the model start as outside every declaration, because `EM015` uses it. A future diagnostic with the same empty range at byte 0 gets no address. A block header at byte 0 is not empty, so it keeps its address.

## Method facts in the pipeline

Specification v0.8.0 ([spec RFC 0007](https://github.com/event-modeling-hcl/spec/blob/main/rfcs/0007-method-facts.md)) adds scenario facts, aggregate identity, lifecycles, evidence, decisions, hotspot kinds, field sources, and key flags. The change adds no top-level FR and no top-level DP. Each fact is a child of an existing DP, so the tool matrix keeps its rows, its columns, and its lower triangular form.

| Parent DP | Children that RFC 0007 adds |
| --- | --- |
| DP1 Syntax | The schemas `LifecycleSchema`, `StateSchema`, `TransitionSchema`, `InvalidTransitionSchema`, `EvidenceSourceSchema`, `EvidenceSchema`, `DecisionSchema`, `FieldSourceSchema`, and `FieldTypeSchema`. `FieldTypeSchema` has no key flags. |
| DP2 Source | `source.SplitFieldReference` and `source.QualifiedElement`, the one interpretation of field references (change rule 2). `AddressAt` gives an address inside a hotspot, a decision, an evidence source, and a lifecycle. |
| DP3 Validation | `validator/fields.go` checks field references and `field_source` blocks (`EM202`). `validator/records.go` checks attach targets, aggregate identity (`EM018`), lifecycles (`EM017`), evidence, and decisions. `validator/scenarios.go` checks `case`, `empty_history` (`EM304`), and dispatch assertions (`EM303`, `EM304`, `EM305`). Each rule is one more term of the core-validity summation. A new core error prevents the IR, as each earlier core error does. |
| DP4 Model | `Model.Lifecycles`, `Model.EvidenceSources`, `Model.Evidence`, and `Model.Decisions`, the field sources of a workflow, and the new scenario, step, hotspot, aggregate, and field values. Each new collection extends the canonical order of FR4. |
| DP5 Output | The formatter writes the attribute order of RFC 0007 section 4.2. `on` sorts last only in `evidence` and `decision`. Every v0.7.0 file formats to the same text. The renderer shows a no-dispatch step with the flag `expect_no_dispatch` in the Model view and with "Expects that the command is not issued" in the slides. |
| DP6.2 Diff | `internal/compare` compares lifecycles, evidence sources, evidence, and decisions, after hotspots (change rule 23). A hotspot change carries the address `hotspot.<id>`. |
| DP7 Interchange | Export warnings for each fact without a JSON field, and the `then` command rule of import (see [Requirements for the interchange module](#requirements-for-the-interchange-module)). |

Inside DP3, the new rules form four children. FR3.A is the scenario facts, FR3.B is identity and lifecycles, FR3.D is field references and field sources, and FR3.C is the record rules for attach targets, evidence, and decisions.

| FR \ DP | DP3.B Identity and lifecycle | DP3.D Field references | DP3.A Scenario facts | DP3.C Records |
| --- | ---: | ---: | ---: | ---: |
| FR3.B Identity and lifecycle | X | 0 | 0 | 0 |
| FR3.D Field sources | 0 | X | 0 | 0 |
| FR3.A Scenario facts | 0 | 0 | X | 0 |
| FR3.C Records | X | X | 0 | X |

The matrix is lower triangular. RFC 0007 section 6.3 gives the one influence across the children: record attach targets read the lifecycle declarations and the field declarations. A rename or a removal of a state, a transition, or a field breaks each record that names it (`EM102`). The repair is a rename of the reference, as for every other identity. No scenario rule reads a lifecycle, and no lifecycle rule reads `identity` or the flow edges.

## Requirements for the agent skills

The repository ships two agent skills. `event-modeling` owns business discovery and business-quality review. `event-modeling-hcl` owns HCL representation and CLI operations. A domain-to-HCL request uses them in that order. Both read the shared edit policy in `event-modeling/references/edit-policy.md`. The HCL skill is a client of the tool. It adds one need, one FR, and one DP:

| Need | Functional requirement | Acceptance criterion | Design parameter |
| --- | --- | --- | --- |
| N7: Authors who work with an AI agent get a model that the tool accepts | FR10: Guide an agent to a model that passes the gates, and to a true report of the gate that it reached | The agent claims a gate only when `emhcl` exited 0 for that gate. Its report names each command and its exit code. | DP10: The skill (`SKILL.md` and `references/`) and the Makefile targets `model-fmt-check`, `model-valid`, and `model-ready` |

In the tool matrix, the FR10 row has `X` under DP1, DP8, DP2, DP3, DP5, DP7, DP9, and DP6. The skill copies grammar facts from DP1 and DP2, the folder rules from DP8, and the scenario productions and diagnostic codes from DP3. It runs `emhcl fmt`, `import`, `migrate`, and `validate`, so it reads the results of DP5, DP7, DP9, and DP6. FR10 has `0` under DP4, because the skill does not read the typed model.

The DP10 column has `0` in every other row. The skill and the Makefile targets only call the CLI and read its exit code. They cannot change what the parser, the validator, the formatter, or an adapter accepts. Thus the FR10 row and the DP10 column go at the end of the matrix, and the matrix stays lower triangular.

FR10 has eight child requirements. Each DP is a formalism that the skill states in a few lines, in place of a prose rule.

| Child FR | Acceptance criterion | DP |
| --- | --- | --- |
| FR10.3: Write only legal scenarios | Each scenario matches the production for its workflow kind | DP10.3: The EBNF scenario productions, copied from the specification into `references/language-v0.8.md` |
| FR10.5: Give each field a source | Each field has a source, a written derivation, or an open hotspot | DP10.5: The field-source contracts in `event-modeling/references/methodology.md`, carried into HCL `field_source` blocks and scenarios |
| FR10.4: Leave locked workflows unchanged | No direct edit or shared-contract change affecting a locked workflow without confirmation that names it and the change | DP10.4: The shared policy in `event-modeling/references/edit-policy.md` |
| FR10.1: Claim a gate only after the tool passed it | A gate is claimed only when its command exited 0 | DP10.1: The Makefile gate graph (`formatted`, `valid`, `implementation-ready`). In this repository, `make model-ready MODEL=<path>`. |
| FR10.2: Stop a repair loop that does not converge | No third edit for the same code on the same element without a new user decision | DP10.2: The repair counter with threshold k = 2 in the HCL skill's “Format, validate, and repair” section |
| FR10.7: Report review findings and show which tool made each one | Each finding is one line. `Error` and `Warning` lines are copied unchanged from the CLI output. The agent's own findings use `Review:` and carry no EM code. | DP10.7: The line formats in the HCL skill's “Report observed results” section |
| FR10.6: Report the result in a form that a program can read | A file-operation report starts with a `Key: Value` header block | DP10.6: The header block in the HCL skill's “Report observed results” section |
| FR10.8: State the supported tool range as data | `make skill-version-check` fails when `metadata.spec_version` differs from the `spec` value of `emhcl version --format json` | DP10.8: The keys `spec_version` and `emhcl_min` under `metadata` in `SKILL.md`, and the Makefile target `skill-version-check` |

| FR \ DP | DP10.3 Productions | DP10.5 Contracts | DP10.4 Status table | DP10.1 Gate graph | DP10.2 Breaker | DP10.7 Diagnostics | DP10.6 Header | DP10.8 Versions |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| FR10.3 Legal scenarios | X | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| FR10.5 Field sources | 0 | X | 0 | 0 | 0 | 0 | 0 | 0 |
| FR10.4 Locked workflows | 0 | 0 | X | 0 | 0 | 0 | 0 | 0 |
| FR10.1 True gates | X | X | X | X | 0 | 0 | 0 | 0 |
| FR10.2 Bounded repair | 0 | 0 | 0 | X | X | 0 | 0 | 0 |
| FR10.7 Review findings | 0 | X | 0 | X | 0 | X | 0 | 0 |
| FR10.6 Report | 0 | 0 | X | X | X | 0 | X | 0 |
| FR10.8 Supported range | 0 | 0 | 0 | 0 | 0 | 0 | 0 | X |

The matrix is lower triangular, so the skill is decoupled. Set the DPs in this order: productions, contracts, status table, gate graph, breaker, diagnostics, header, versions. Outside the skill, FR10.8 depends on DP6.5, because `skill-version-check` reads `emhcl version --format json`.

The off-diagonal entries are these influences:

- FR10.1 depends on DP10.3, because a scenario that breaks a production fails `validate`. It depends on DP10.5, because a `field_source` block that breaks a rule fails `validate` with `EM202`. It depends on DP10.4, because a lock can stop the repair that a gate needs.
- FR10.2 depends on DP10.1, because the breaker counts the failures that the gate commands report.
- FR10.7 depends on DP10.1 for the `EM4xx` lines, and on DP10.5 for the findings that have no code.
- FR10.6 reports the gate (DP10.1), an open breaker (DP10.2), and the locked workflows (DP10.4).

The important zeros are these:

- FR10.1 does not depend on a missing field source. The validator checks only the `field_source` blocks that a model has, so a field without a source never changes a gate.
- FR10.3 has `0` under DP10.1. The gate graph finds an illegal scenario, but only the productions and the validator decide what is legal. A gate never makes an illegal scenario legal.
- No DP of the skill has an `X` in a row of the tool matrix.

The Makefile targets keep one owner for each fact. `model-fmt-check` runs `emhcl fmt --check`, so the formatter owns the layout and the loader owns the member files. `model-valid` and `model-ready` run `emhcl validate`, so the validator owns validity and the profiles. The targets only fix the order of the three calls. `examples-fmt-check` runs `emhcl fmt --check` on each model in `examples/`. `skill-version-check` compares the skill metadata with `emhcl version --format json`, so `internal/version` owns the specification version.

## Module contracts

| Module | Owns | Input | Output and failure contract |
| --- | --- | --- | --- |
| `internal/syntax` | HCL body schemas, including the lifecycle, record, and field-source schemas of v0.8.0, and parsing | One or more files, each with a filename and source bytes | Parsed document or HCL syntax diagnostics. Each diagnostic names its own file. Two files with the same name give a diagnostic and no document. |
| `internal/source` | Reference interpretation shared by the validator and the model, and declaration addresses | Parsed document | A wrapper around the parsed document, helpers that turn reference expressions into addresses, the split of a field reference into its element and its field path, and `AddressAt`, which turns a source range into the address of a declaration. It owns the element kinds that have a reference form. No I/O and no policy. |
| `internal/validator` | Core language rules, the catalog index, and the judgment policy evaluator | Decoded source and a profile | Diagnostics. Without core errors, also a `ValidatedDocument`, in every profile. |
| `internal/model` | Canonical model for renderers and interchange, with normalized edges | `ValidatedDocument` | The same `Model` for the same input. No parsing, no validation, no I/O. |
| `internal/compare` | The declaration changes between two models | Two canonical `Model` values | Added, removed, and changed declarations in a fixed order, with addresses from `internal/source`. No parsing, no validation, no I/O. |
| `internal/formatter` | Canonical HCL layout | Source bytes of one file | Formatted bytes or parse diagnostics |
| `internal/renderer` | View model, standalone HTML, the viewer scripts, and the default diagram title (`ModelName`) | Canonical `Model` and a title | HTML, or a template or serialization error. The viewer obeys the matrix for the diagram viewer. |
| `internal/interchange` | Schema check and conversion for slice-based JSON | JSON bytes, or a canonical `Model` | A document or unformatted HCL with warnings. A returned error means that the input has no correct conversion. No I/O. |
| `internal/app` | Use-case sequence and plain diagnostics, with their one JSON encoding | Source in memory, or a model path (one file or one folder) | Results for validate, format, render, diff, import, export, and migrate that all adapters share. A model, HTML, JSON, or a diff exists only when there is no core error. Import is the exception: it returns the HCL source together with its validation diagnostics. |
| `internal/migrate` | v0.4.0 reading and the rewrite to v0.5.0 | Member files of one v0.4.0 model | Every member file, edited or unchanged, and a report, or error diagnostics and no files. No I/O. |
| `internal/version` | The specification version and the tool version | The release version that the linker injects | `version.Spec` and `ToolVersion`. Used by `cmd/emhcl` and the renderer. |
| `internal/playground`, `cmd/playground-seed` | The example source for the browser editor | One `.em.hcl` file | `seed.js` for `web/playground`. A build step, not part of the language pipeline. |
| CLI (`internal/cli`, `cmd/emhcl`), WASM (`cmd/wasm`), `internal/serve` | Actions for each runtime, and the `--format` option | Arguments, files, signals, HTTP, JavaScript values | Exit codes, text or JSON output, files, browser values, and server lifecycle. The CLI starts `internal/serve`. |

The loader (DP8) is `app.ReadModel` with `syntax.ParseFiles` and `syntax.Document.SourceOf`. Its contract has four rules:

- R1: A path to a file is a one-file model. A path to a folder is a folder model.
- R2: A folder model uses every regular file directly in the folder whose name ends in `.em.hcl` and does not start with a dot. Symlinks to files count. Subfolders and other files are ignored.
- R3: `syntax.ParseFiles` sorts the member files by name, byte by byte, and not by locale. `app.ReadModel` does not sort. This order fixes only the diagnostic order and which duplicate is reported first. It never changes the model.
- R4: A folder with no member file gives error EM001 with the detail `folder <path> has no .em.hcl files`.

`syntax.ParseFiles` takes the files in that order and parses each one on its own. If any file has a syntax error, it returns no document and all the syntax diagnostics. `SourceOf` returns the bytes of the file whose name equals the filename of an HCL range, so each diagnostic and each source excerpt names its own file. A folder with one member file behaves like that file.

OS actions stay at the boundary. The server injects file, listener, browser, stream, and signal actions. Thus tests can examine lifecycle behavior without a change to language calculations.

The CLI owns output-file safety. `emhcl export` refuses an `-o` target that has the `.em.hcl` extension, that is the source file, or that is a member file of the source folder. It exits with code 2. `emhcl import` refuses an `-o` target that does not end in `.em.hcl`. When a decode or a conversion fails, `emhcl import` does not change an existing output file. When only validation fails, it writes the file and reports the diagnostics.

## Change rules

1. Change the language grammar in `internal/syntax` and in the normative specification first. Then update source decoding and validator rules, in that order.
2. Add a shared interpretation to `internal/source`. Do not rebuild catalogs, field-type addresses, or traversals in validators or renderers.
3. Keep validation policy in `internal/validator`. A model or a renderer must not correct invalid input without a diagnostic.
4. Keep the canonical model independent of presentation. Layout for a renderer belongs in `internal/renderer`.
5. Add a use case one time, in `internal/app`. Runtime adapters translate inputs and outputs. They do not build the pipeline again.
6. Send new OS effects through an adapter or the injected server environment. Define cancellation and failure behavior for each worker.
7. Keep the JSON schema rules in `conform.go`. Import and export must both use them.
8. Decode the original JSON bytes. Do not encode an accepted document a second time, because that changes key order.
9. On import, write JSON values with `jsonTokens`. On export, read them with `jsonValue`. Do not send object keys through `cty` on import or on export.
10. If a conversion has no correct native form, return an error. If it drops data, write a warning at the place where it drops the data.
11. If you change a flow rule in `internal/validator`, change its copy in `import.go` or `import_policy.go` in the same change.
12. After each change to the interchange mapping, make sure that the round trip stays stable. The tests do this for each model in `examples/` and for the five JSON files in `testdata/valid/`. The two `.em.hcl` files in `testdata/valid/` have no round-trip test.
13. Renaming a member file or moving a block between files does not change the model. It can change which duplicate declaration EM002 reports as the first one. Do not add a rule that depends on the number of files or on block position.
14. Add a route only in `parseHash`. Load-time scoping and navigation read the same grammar.
15. A board must not render or call another board. A new board registers its filter function with `EMC.onFilterChange`. It also needs its render function in `showPage`, its layout function in `relayoutAll`, and its script in `render.go`. No registration hook exists for these three places.
16. Put a presentation that two views share in `viewer.js` under `EMC`. Examples are a slice timeline and a hotspot note. Do not export it from one view to another, and do not copy it.
17. Change the fragment with `location.hash` or `replaceHash`. Do not let the browser follow a fragment link, because inside the `srcdoc` frame of the playground the link loads the playground page.
18. Keep v0.4.0 rules inside `internal/migrate`. No other package can infer field types, derive titles, or read the `translation` keyword.
19. A profile assigns only the severity of a judgment signal. Do not let a profile, a judgment signal, or a renderer decide whether a model, HTML, or JSON is produced. Only core errors decide it.
20. If you change a scenario rule in `validator/scenarios.go`, a diagnostic code, a profile severity, or an exit code of `internal/cli`, change the specification first (rule 1). Then change the copy in `.agents/skills/event-modeling-hcl` in the same change.
21. Keep the Makefile model targets as calls to `emhcl`. Do not add a rule to them that the CLI does not check.
22. Add a diagnostic key only as a JSON tag of `app.Diagnostic`. The CLI, the WASM module, and `serve` all encode that type.
23. If you add a collection to `model.Model`, add its unit to `internal/compare` in the same change.

A change breaks the Independence Axiom when, for example:

- A renderer starts to decide validity.
- A new frontend builds its own diagnostics.
- Model construction accepts HCL that the validator did not accept.
- A grammar fact gets two owners.
- The importer writes a model that the validator did not accept.
- A viewer page renders another board to get filters, data, or a helper.
- A viewer route gets a second parser.
- The agent skill states a language rule that the specification and the validator do not have.

## Known deviations

The code at release v0.17.0 does not follow this guide in six places. Each one keeps the matrices triangular, but each one gives a fact two owners. Two owners make a change more likely to break one of the FRs.

| ID | Deviation | Rule | Effect | Correction |
| --- | --- | --- | --- | --- |
| D1 | `internal/source` has no catalogs. The validator builds its own catalog index (`buildIndex` in `validator/index.go`). `model.Build` decodes every block again from `syntax.Content`. The validator also calls `hcl.AbsTraversalForExpr` directly in several passes. | 2 | FR3 and FR4 each read catalogs by their own code. The model can read a catalog in a different way from the validator. Model tests are the only protection. | Move the catalog index into `internal/source` and let the validator and the model read it. |
| D2 | `filesArg` in `cmd/wasm/main.go` sorts the files and refuses duplicate names. `syntax.ParseFiles` already does both. | 5, R3 | The result is the same, but a change to R3 needs two edits. | Delete both checks from `filesArg`. |
| D3 | The resolved-hotspot check (the class, the mark, and the label) is copied into `viewer.js`, `viewer.compact.js`, and `viewer.eventstorming.js`. The slides do not use it. | 16 | A change to the meaning of a hotspot status needs three edits, and the slides can disagree. | Add one hotspot-status helper to `EMC` and use it in the boards and the slides. |
| D4 | The interchange module repeats three validator flow rules (see [Influences across branches](#influences-across-branches)). | 11 | The importer can write a flow that the validator refuses. `app.Import` then reports the error, so the user sees it. | Move the flow tables to a package that both modules import. |
| D5 | A new board needs edits in `showPage`, `relayoutAll`, and `render.go`. | 15 | Each board name has three owners. | Let each board register its render and layout functions with `EMC`. |
| D6 | The agent skill copies the scenario productions, the diagnostic codes, the profile severities, and the exit codes (see [Requirements for the agent skill](#requirements-for-the-agent-skill)). | 20 | The skill can teach a rule that the validator no longer has. The gate graph still reports the real result, so the agent cannot claim a gate that the tool refused. | Generate the skill copy from the specification, or check it in `make verify`. |

## Information Axiom and evidence

The repository does not give a number for information content. It does not measure the probability that each FR succeeds in production. The design reduces known failure modes with these mechanisms:

- Dependencies that go in one direction.
- A sequence that the types enforce.
- Stable diagnostics.
- Injected effects.
- Automated tests.

`make verify` runs a repeatable set of steps. They cover formatting, module tidiness, `go vet`, unit tests, the race detector, Staticcheck, switch exhaustiveness, vulnerabilities, a compile of the WASM module, the validation and the `emhcl fmt` layout of the shipped examples, and three checks of the release tag and version.

The agent skill has less evidence. `evals/evals.json` holds six prompts with expectations. The formalisms in DP10 come from a published A/B benchmark on Gemini models (Randal L. Schwartz, "What LLMs 'Know' That You Don't Know They Know", DEV Community, 2026). That benchmark does not measure this skill.

One manual run on 2026-10-31 compared the versions of the skill, with one run for each eval and blind graders. E1 is a true report: each claim matches what `emhcl` prints, and no line looks like `emhcl` output unless `emhcl` printed it.

| Skill version | Expectations passed | E1 passed |
| --- | --- | --- |
| v2.0 (prose rules) | 32 of 32 | 6 of 6 |
| v2.1 (formalisms, own findings in the `emhcl` line format) | 31 of 32 | 3 of 6 |
| v2.2 (own findings as `Review:` lines), evals 2, 3, 4, and 6 only | 19 of 20 | 4 of 4 |

The E1 failures in v2.1 had one cause. The skill told the agent to write its own findings in the `emhcl` line format, so a reader could not tell them from tool output. v2.2 gives those findings their own `Review:` form, which is why FR10.7 now has that criterion. The one expectation that failed in v2.1 and v2.2 is the same one: eval 2 did not use a reusable `field_type` for each payment field. One run per eval cannot show whether that is a regression or chance. The data is in `skills/event-modeling-hcl-workspace/iteration-2` of the workspace, outside this repository. These samples are too small to give a probability for the Information Axiom. Run each eval several times for each version before you claim that one version is better.

The interchange module has this evidence:

- Regression tests for each defect from the review of the `json-cmd` branch. They are in `review_regressions_test.go` and `export_value_regressions_test.go` in `internal/interchange`, and in `internal/cli/interchange_test.go`.
- `TestExportImport_RoundTripsEveryShippedExample`, which compares the first and the second export of each file and folder model in `examples/`.
- `real_fixtures_test.go`, which runs import, export, and the round trip on the five JSON files in `testdata/valid/`. A manual run with `ajv` accepted the source files and the exported files. The repository has no automated `ajv` run. `conform.go` repeats the `ajv` rules in Go.

Some limits are known. String values (not keys) pass through `cty` on both import and export. Thus Unicode normalization can change the text of a string value. Two read models in the Cart fixtures have different `aggregate` metadata. No evidence shows which value is correct.

The diagram viewer has less evidence. The Go tests check that the HTML contains the viewer code and the view model JSON. No automated test runs the viewer in a browser, and no test covers the resolved-hotspot styles. The matrix for the viewer comes from a manual browser check: a Storming page that opened directly rendered no Model board and applied the context filter. The route checks covered valid, unknown, and malformed fragments. A browser test of the routes, the filters, the hotspot styles, and the slides would turn these checks into repeatable evidence.

To compare this architecture with an alternative under the Information Axiom, measure release history, escaped defect counts, flaky-test rates, and change lead time.
