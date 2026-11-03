# RFC 0001: Machine-Readable Tool Contracts

## Status

Accepted for emhcl v0.15.0. This RFC changes the `emhcl` tool and its agent skill. It changes no language rule, so it needs no RFC in the [specification repository](https://github.com/event-modeling-hcl/spec). The specification lists the command synopsis, and its `[Unreleased]` section records the new options (section 3). "Before" in this RFC means emhcl v0.14.0, which implements specification v0.6.0.

## 1. Summary and Motivation

This RFC adds five tool contracts that a program can read:

1. `emhcl validate --format json` writes each diagnostic as JSON, with its file, its judgment flag, and the address of the declaration that it points to.
2. `emhcl diff <base> <head>` lists the declarations that a change added, removed, or changed.
3. `emhcl fmt` accepts a folder. A new `--check` mode lists the member files that are not in canonical layout.
4. `emhcl diagram --title <text>` sets the title of the diagram.
5. `emhcl version --format json` writes the tool version and the specification version.

It also adds two fields to the metadata of the agent skill. The fields state the specification version and the oldest `emhcl` release that implements it.

Programs call `emhcl` in three places today: the agent skill, the Makefile model targets, and CI jobs. A fourth caller is planned. It is a Claude plugin that lets a team view, change, and review a model through merge requests. Each caller reads text that the tool writes for people. That text has no stable contract, so a change to a message can change the behavior of a caller.

The table shows each gap and the control that this RFC adds for it.

| Caller need | Behavior before | Effect | Control |
| --- | --- | --- | --- |
| Stop a repair loop after two failed edits for one code on one element (FR10.2). | A diagnostic line names a file, a line, and a column, but no element. | The agent guesses the element from the line. An edit that moves lines changes the count. | The `address` key of a JSON diagnostic |
| Tell a core error from a policy rejection. | `validate` exits 1 for both. The text line does not show the judgment flag. | The caller must know that the `EM4xx` codes are judgment signals. | The `judgment` key of a JSON diagnostic |
| See which workflows a change touched. | No command compares two models. | An agent compares source text. It counts a moved block or a new layout as a change. | `emhcl diff` |
| Require canonical layout for a folder model. | `fmt` refuses a folder with "fmt formats one .em.hcl file at a time". | The Makefile copied the member rule R2 of the loader in `MODEL_FILES`. Every other caller had to copy it too. | `fmt <folder>` and `fmt --check` |
| Show the branch and the commit of a rendered snapshot. | The diagram title is the name of the model file or folder. | A caller that renders a snapshot in a temporary folder gets the name of that folder. | `diagram --title` |
| Compare the installed tool with the specification that the skill teaches. | `version` prints only the tool version. The skill states its range in prose. | The caller parses prose. | `version --format json` and the skill metadata |

## 2. Changes

### 2.1 The output format option

`validate`, `diff`, and `version` accept `--format text|json`. The default is `text`. Text output stays byte for byte the same as before. With `json`, the command writes one JSON object to stdout. It writes nothing to stderr, except a usage error.

Exit codes keep their meaning: 0 for success, 1 when a model has problems, and 2 for a command line that the tool cannot read. The format never changes the exit code. One function in `internal/cli`, `parseFormat`, reads the option for all three commands.

### 2.2 Diagnostics as JSON

```text
emhcl validate [--profile workshop|valid|strict] [--format text|json] <model.em.hcl | folder>
```

The options can come in any order, and each option can appear one time. A repeated option is a usage error, in `validate` and in `diff`. The JSON object has this form. The file, the line, and the column are an illustration.

```json
{
  "path": "model",
  "profile": "strict",
  "has_errors": true,
  "has_core_errors": false,
  "diagnostics": [
    {
      "code": "EM404",
      "severity": "Error",
      "summary": "Every command has a reason",
      "detail": "command \"place_order\" has no incoming flow, api_endpoint, or external_trigger.",
      "line": 14,
      "column": 3,
      "judgment": true,
      "file": "model/20-orders.em.hcl",
      "address": "command.place_order.place_order"
    }
  ]
}
```

These rules apply to the object:

1. A diagnostic has the keys `code`, `severity`, `summary`, `detail`, `line`, `column`, `judgment`, and `file`, plus `address` when the diagnostic has one. The WASM module gives the browser the same keys.
2. `severity` is `Error`, `Warning`, or `Info`. Under `workshop`, each judgment signal has `Info`.
3. The diagnostics have the same order as the text output.
4. `has_errors` is true when the command exits 1. `has_core_errors` is true when the tool withholds the model (`Diagnostics.HasCoreErrors`).
5. `diagnostics` is `[]` when there is no diagnostic.
6. `address` names the innermost declaration that contains the start of the diagnostic location and that has a reference form in the specification.

The reference forms come from the section "References and Identity" of the specification:

| Declaration | Address |
| --- | --- |
| Bounded context, actor, team, system, workflow | `bounded_context.<id>`, `actor.<id>`, `team.<id>`, `system.<id>`, `workflow.<id>` |
| Event, aggregate, field type | `<kind>.<context>.<id>` |
| Command, read model, screen, processor | `<kind>.<workflow>.<id>`, the workflow-qualified form that `hotspot.on` uses |

A declaration without a reference form gives the address of the declaration that contains it. Examples are a `screen_image`, a `table`, a `scenario`, and a `field`. A diagnostic outside every declaration with a reference form has no `address` key. Examples are a syntax error, `EM001`, and a diagnostic in a chapter, a hotspot, or the `board` block. The empty range at the start of the model, which `EM015` uses for a missing board, also has no address.

`internal/source` owns the address. It already owned the canonical reference strings (`CanonicalFieldTypeReference`, `ReferenceString`). This RFC adds the `Address` type, `ElementHasAddress`, and `(*Document).AddressAt`, which maps an HCL range to an `Address`. `syntax.BlockRange` gives the extent of a block. The lookup matches the file name and the byte offset, so an offset in one member file never matches a block in another file.

The validator rules do not change. Before, `validator.referenceExists` had its own list of the four element kinds with a workflow-qualified form. It now reads `source.ElementHasAddress`, so that list has one owner.

`app.Diagnostic` gets an `Address` field. Its `Filename` field gets the JSON key `file`, in place of `json:"-"`. `Diagnostics.MarshalJSON` encodes no diagnostics as `[]`. The CLI encodes `app.Diagnostic` with these tags. `cmd/wasm` builds its JavaScript values from the same JSON encoding, so the list of keys has one owner. `serve` hashes the same encoding to decide when to reload.

### 2.3 Compare two models

```text
emhcl diff [--format text|json] <base> <head>
```

Each argument is a file model or a folder model. The command runs these steps:

1. Load each model and run the validation with the `valid` profile.
2. If either model has a core error, write the diagnostics of each refused model, write no comparison, and exit 1.
3. Otherwise, compare the two canonical models (`internal/model`) and exit 0. The exit code is 0 also when the models differ.

The comparison uses these units:

| Unit | Identity | Address |
| --- | --- | --- |
| Actor, team, system, bounded context | ID | `actor.<id>`, `team.<id>`, `system.<id>`, `bounded_context.<id>` |
| Event, aggregate, field type | Context and ID | `<kind>.<context>.<id>` |
| Workflow | ID | `workflow.<id>` |
| Command, read model, screen, processor | Workflow, kind, and ID | `<kind>.<workflow>.<id>` |
| Screen image, table | Workflow, kind, and ID | None |
| Scenario | Workflow and ID | None |
| Chapter, hotspot | ID | None |
| Board | One for each model | None |

A unit is added when only the head has it. A unit is removed when only the base has it. A unit is changed when both models have it and its canonical values differ.

The value of a bounded context is its own attributes. Its events, aggregates, and field types are units of their own. The value of a workflow includes its elements, its scenarios, and its edges (`Model.Edges` with its `WorkflowID`). A changed workflow lists its added, removed, and changed elements and scenarios. A field in the canonical model keeps its `field_type` reference and does not copy the metadata of that type. So a change to a field type marks only `field_type.<context>.<id>` as changed.

The canonical model does not depend on file names or block positions (change rule 13). So a block that moves to another member file gives no entry. A change of layout gives no entry either.

Entries follow the order of the unit table: actor, team, system, bounded context, event, aggregate, field type, workflow, chapter, hotspot, board. Inside one kind, entries follow the byte order of the identity. Inside a workflow, elements come first, by kind and then ID, and then scenarios. The text output has this form:

```text
removed event.clinic.pet_archived
added workflow.cancel_visit
changed workflow.register_pet
  changed command.register_pet.register_pet_command
  added scenario register_pet_rejects_duplicate
changed board
```

The JSON output has this form:

```json
{
  "base": "base/model",
  "head": "head/model",
  "changes": [
    {"change": "removed", "kind": "event", "context": "clinic", "id": "pet_archived", "address": "event.clinic.pet_archived"},
    {"change": "added", "kind": "workflow", "id": "cancel_visit", "address": "workflow.cancel_visit"},
    {"change": "changed", "kind": "workflow", "id": "register_pet", "address": "workflow.register_pet",
     "children": [
       {"change": "changed", "kind": "command", "id": "register_pet_command", "address": "command.register_pet.register_pet_command"},
       {"change": "added", "kind": "scenario", "id": "register_pet_rejects_duplicate"}
     ]},
    {"change": "changed", "kind": "board"}
  ]
}
```

`changes` is `[]` when the models are equal. If a model has a core error, the object has the key `diagnostics` from section 2.2 and no key `changes`. Each diagnostic names its file, so the base and the head stay apart.

`internal/compare` holds the comparison. It reads only canonical models: no parsing, no validation, and no I/O. `app.DiffPaths` runs the use case, and the CLI only writes the result.

### 2.4 Format a folder

```text
emhcl fmt [-w | --check] <model.em.hcl | folder>
```

These rules apply:

1. A folder uses the member files of the loader (rule R2), in the order of rule R3. `app.FormatPath` reads them through `app.ReadModel`.
2. Without a flag, the command prints each file. For a folder, each file starts with the line `# ==> <file> <==`, as `migrate` without `-w` does.
3. With `-w`, the command rewrites only the files that change.
4. With `--check`, the command writes no file. It prints the name of each member file whose layout differs from the `fmt` output, one name on each line. It exits 1 when at least one file differs, and 0 when none differs.
5. `-w` and `--check` together are a usage error.
6. If a member file has a syntax error, the command writes the diagnostics, changes no file, and exits 1.

The formatter still works on one file at a time, as its module contract states. Formatting still needs no valid model. `app.FormatFile` is gone, because `app.FormatPath` covers a file and a folder.

The Makefile targets `model-fmt-check` and `examples-fmt-check` make one `emhcl fmt --check` call for each model. `MODEL_FILES`, `EXAMPLE_FILES`, and the `emhcl-fmt-check` macro are gone, so rule R2 has one owner again.

### 2.5 Diagram title

```text
emhcl diagram [--title <text>] [-o <file>] <model.em.hcl | folder>
```

`--title` sets the title of the view model. The masthead and the document title of the viewer show it. The viewer writes the title with `textContent` and `document.title`, so the text is not read as HTML. An empty `--title` is a usage error with exit code 2. `--title` with `export` or `import` is a usage error.

The renderer takes the title as an input: `renderer.Render(title, model)` and `renderer.BuildViewModel(title, model)`. `renderer.ModelName(path)` is the default title, the name of the file or folder, as before. `app.RenderPathTitled` passes a title. `app.RenderPath` and `app.RenderFiles` pass the default, so `serve` and the playground keep their titles.

The language has no model title. The title is a fact about one rendering, so it is a command option and not a language construct.

### 2.6 Versions as data

```text
emhcl version [--format text|json]
```

The JSON object has two keys:

```json
{"tool": "v0.15.0", "spec": "v0.6.0"}
```

`tool` has the same value as the text output. For a release, that value is the release tag, with its `v`. For a local build, it is `dev`. `spec` is `version.Spec`.

### 2.7 Skill metadata

The front matter of `.agents/skills/event-modeling-hcl/SKILL.md` has two new keys under `metadata`:

```yaml
metadata:
  version: "2.4.0"
  spec_version: "v0.6.0"
  emhcl_min: "v0.14.0"
```

`spec_version` is the specification that the skill teaches. `emhcl_min` is the oldest release that implements that specification. The `compatibility` text stays for people.

The skill still formats a folder model one member file at a time, so its commands work with v0.14.0. Its gate graph notes that v0.15.0 also accepts the folder.

The Makefile target `skill-version-check` is part of `make verify`. It fails when `spec_version` differs from the `spec` value of `emhcl version --format json`.

## 3. Compatibility and Migration

Each change adds an option, a command, or a key. Each existing call gives the same output and the same exit code as before, with these exceptions:

- `emhcl fmt <folder>` exited 2 before. Now it formats the folder.
- `emhcl fmt` with a path that is neither a `.em.hcl` file nor a folder now gives the same message as the other commands: "model path must be a .em.hcl file or a folder". Before, it gave "model file must use the .em.hcl extension".
- The usage message lists the new options and the `diff` command.
- `validate` accepts its options in any order. A repeated option is still a usage error.

The WASM module adds the key `address` where it applies, and gives `[]` for no diagnostics as before. The playground (`web/playground/editor.js`) reads the keys `severity`, `code`, `judgment`, and `file` by name, so the new key changes nothing there.

The specification lists the command synopsis. Change rules 1 and 20 of the architecture guide require the specification edit first. The new synopsis in the specification is:

```text
emhcl validate [--profile workshop|valid|strict] [--format text|json] <model.em.hcl | folder>
emhcl fmt [-w | --check] <model.em.hcl | folder>
emhcl migrate [-w] <model.em.hcl | folder>
```

The agent skill copies the exit codes and the gate commands (deviation D6). The skill text and its language reference now name `fmt --check` and the folder form of `fmt`.

The release that ships this RFC is emhcl v0.15.0, a minor release. The specification version stays v0.6.0.

## 4. Module Impact

| Module | Change |
| --- | --- |
| `internal/syntax` | Adds `BlockRange`, the extent of a block from its type keyword to its closing brace |
| `internal/source` | Adds the `Address` type, `WorkflowAddress`, `ElementHasAddress`, and `AddressAt` |
| `internal/validator` | `referenceExists` reads `source.ElementHasAddress`. No rule changes. |
| `internal/model` | None |
| `internal/compare` | New. Lists the declaration changes between two canonical models. |
| `internal/formatter` | None |
| `internal/renderer` | `Render` and `BuildViewModel` take a title. `ModelName` is exported as the default title. |
| `internal/app` | `Diagnostic` gets `Address` and the JSON key `file`. Adds `Diagnostics.MarshalJSON`, `DiffPaths`, `FormatPath`, and `RenderPathTitled`. Removes `FormatFile`. |
| `internal/cli` | Parses `--format`, `--check`, and `--title`. Adds the `diff` command. Encodes JSON. `fmt` and `migrate` share the helpers that print and rewrite member files. |
| `internal/serve` | Hashes the JSON encoding of the render result, which now includes the file of each diagnostic |
| `cmd/wasm` | Builds diagnostics from the JSON encoding of `app.Diagnostics` |
| `Makefile` | Replaces the shell loops of the format targets. Adds `skill-version-check`. |
| `.agents/skills/event-modeling-hcl` | Adds the metadata keys. Names `fmt --check` and the folder form of `fmt`. |
| `architecture-guide.md` | Adds the requirements of section 5, the new module contracts, and change rules 22 and 23 |

Change rule 5 applies. Each use case goes into `internal/app` one time, and the CLI only translates arguments and output.

## 5. Design Analysis

This section uses the terms and the requirement IDs of the [architecture guide](../architecture-guide.md).

### 5.1 Need and requirements

The RFC adds one need. N8: a program that calls `emhcl` gets results with a stable contract. The constraints are these:

1. Text output and exit codes of existing calls do not change, except as section 3 lists.
2. No language rule changes.
3. Each fact keeps one owner.

The new requirements are children of FR2, FR6, and FR10:

| FR | Acceptance criterion | DP |
| --- | --- | --- |
| FR2.1: Turn a source range into the address of a declaration | Each reference form gets its innermost declaration. A chapter, a hotspot, the board, and the model start get no address. An offset in one member file never matches a block of another file. | DP2.1: `Address`, `ElementHasAddress`, and `AddressAt` in `internal/source`, with `syntax.BlockRange` |
| FR6.1: Report diagnostics as data | For each model in `examples/` and `testdata/` and each profile, the JSON diagnostics equal the text diagnostics in code, severity, file, line, column, and order | DP6.1: The JSON tags of `app.Diagnostic`, `Diagnostics.MarshalJSON`, and `parseFormat` in `internal/cli` |
| FR6.2: Compare two models by declaration | A moved block and a new layout give no entry. Each edit in the fixture set gives exactly its expected entry. | DP6.2: `internal/compare`, `app.DiffPaths`, and the `diff` command |
| FR6.3: Report the layout of each member file | `fmt --check` names exactly the member files that differ from `fmt` output | DP6.3: `app.FormatPath`, which reads the member files through `app.ReadModel`, and the `--check` option |
| FR6.4: Show a title that the caller supplies | The masthead shows the `--title` text. Without the option, the title is the name of the file or folder. | DP6.4: The `title` input of the renderer, `app.RenderPathTitled`, and `--title` |
| FR6.5: Report the versions as data | `spec` equals `version.Spec`, and `tool` equals the text output | DP6.5: `version --format json` |
| FR10.8: State the supported tool range as data | `skill-version-check` fails when `spec_version` differs from `version.Spec` | DP10.8: The metadata keys and `skill-version-check` |

Each requirement had alternatives. The table gives the reason against each one.

| FR | Alternative | Reason against it |
| --- | --- | --- |
| FR2.1 | The validator attaches an address to each diagnostic | Each diagnostic site in the validator then depends on the address policy, so FR3 gets an `X` under a new DP at every site |
| FR2.1 | `internal/app` computes the address | A second owner reads blocks and references (change rule 2) |
| FR6.1 | Each adapter builds its own map of keys | Two owners of the key list. `cmd/wasm` had this copy before. |
| FR6.2 | Compare source text, or compare the interchange JSON | Source text couples the result to the layout (DP5). Interchange JSON couples it to DP7 and drops native data. |
| FR6.3 | Loop over member files in the CLI or the Makefile | A copy of loader rule R2 (change rule 5) |
| FR6.4 | `internal/app` overwrites the title of the view model | `internal/app` then owns a presentation fact (change rule 4) |
| FR6.4 | A model title in the language | Presentation in the canonical model (change rule 4) |

### 5.2 Independence matrix for the new children

| FR \ DP | DP2.1 Address | DP6.1 Diagnostics | DP6.2 Diff | DP6.3 Format | DP6.4 Title | DP6.5 Versions |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| FR2.1 Address | X | 0 | 0 | 0 | 0 | 0 |
| FR6.1 Diagnostics as data | X | X | 0 | 0 | 0 | 0 |
| FR6.2 Compare models | X | X | X | 0 | 0 | 0 |
| FR6.3 Layout of member files | 0 | 0 | 0 | X | 0 | 0 |
| FR6.4 Diagram title | 0 | 0 | 0 | 0 | X | 0 |
| FR6.5 Versions as data | 0 | X | 0 | 0 | 0 | X |

The matrix is lower triangular, so the new branch is decoupled. Set DP2.1 first, then DP6.1. Then set DP6.2, DP6.3, DP6.4, and DP6.5 in any order.

The off-diagonal entries are these influences:

- FR6.1 depends on DP2.1 for the `address` key.
- FR6.2 depends on DP2.1 for the address strings. It depends on DP6.1, because `diff` writes diagnostics with the same encoding when a model has a core error, and it reads the same `--format` option.
- FR6.5 depends on DP6.1, because `version` reads the same `--format` option.

The important zeros are these:

- FR6.3 has `0` under DP6.1. `fmt` has no `--format` option, and its diagnostics stay text.
- FR6.4 has `0` under DP6.2. The title is a command option and is not part of the model, so a title never appears as a change.
- FR2.1 has `0` under every DP6 column. The address exists before an adapter reads it.

### 5.3 Influences across branches

These rows depend on DPs of other branches:

- FR6.1 depends on DP3 for the diagnostics, and on DP8 for `file`.
- FR6.2 depends on DP4, because it compares canonical models.
- FR6.3 depends on DP5 for the layout and on DP8 for the member files.
- FR6.4 depends on DP5, because the renderer writes the title.
- FR10.8 depends on DP6.5 and on `internal/version`.

One existing row depends on DP2.1. FR3 has an `X` under DP2.1, because `validator.referenceExists` reads `source.ElementHasAddress`. FR3 already depends on DP2, and DP2.1 is part of DP2, which comes before DP3. So the list of element kinds has one owner, and the order of the tool matrix stays the same.

The DP6.1 to DP6.5 columns have no `X` in the rows FR1 to FR5 or FR7 to FR9. Each of these DPs only reads a result that an earlier DP made. So the tool matrix stays lower triangular. The order is DP2.1, then DP6.1, then the other FR6 children, then DP10.8.

The RFC also removes two copies of a fact:

- The Makefile copy of rule R2 in `MODEL_FILES` and `EXAMPLE_FILES`.
- The list of diagnostic keys in `diagnosticsToJS` of `cmd/wasm`, which repeated the fields of `app.Diagnostic`.

Two risks remain:

- `internal/compare` lists the collections of `model.Model`. A test fails when the model gets a collection that `compare` does not read. Change rule 23 states the same duty.
- `AddressAt` treats the empty range at the model start as outside every declaration. A future diagnostic with the same empty range at byte 0 gets no address. A block header at byte 0 is not empty, so it keeps its address.

### 5.4 Information content

This RFC gives no number for information content. No caller measured how often it reads a text diagnostic wrong, or how often an agent compares two models wrong. Two measurements can decide:

1. Make pairs of models from `examples/` with known edits: a moved block, a new layout, a changed field type, and a new workflow. Count the pairs where an agent that compares text gives the expected entries. `diff` must give them for every pair.
2. Run the skill evals in `.agents/skills/event-modeling-hcl/evals/evals.json` several times with text diagnostics and with JSON diagnostics. Count the runs where the circuit breaker stops at the correct code and element.

## 6. Acceptance Tests

The tests are in the repository:

| Test | Package | What it shows |
| --- | --- | --- |
| `TestRun_ValidateJSONMatchesTextOutput` | `internal/cli` | For each model in `examples/` and `testdata/` and each profile, the JSON diagnostics give the text output and the same exit code |
| `TestRun_ValidateJSONReportsJudgmentAndAddress` | `internal/cli` | A strict `EM404` is a judgment error with `address` `command.example.submit`, and `has_core_errors` is false |
| `TestAddressAt_NamesTheInnermostAddressableDeclaration` | `internal/source` | Each reference form, and the fallback to the workflow for a table and a scenario |
| `TestAddressAt_GivesNoAddressOutsideAddressableDeclarations` | `internal/source` | A chapter, a hotspot, the board, and the model start have no address |
| `TestAddressAt_BlockAtFileStartKeepsItsAddress` | `internal/source` | A block header at byte 0 keeps its address |
| `TestAddressAt_MatchesTheFileOfTheRange` | `internal/source` | Two member files with overlapping offsets |
| `TestRun_DiffIgnoresFileLayoutAndListsDeclarationChanges` | `internal/cli` | A folder model and the same model in one file with a new layout give no entry. A changed field type gives only its own entry. Text and JSON output. |
| `TestRun_DiffRefusesAModelWithCoreErrors` | `internal/cli` | Exit 1, diagnostics in the head file, and no `changes` key |
| `TestModels_CoversEveryTopLevelCollectionOfTheModel` | `internal/compare` | A new collection in `model.Model` fails the test |
| `TestModels_EdgeChangeMarksItsWorkflowOnly` | `internal/compare` | An edge belongs to the value of its workflow |
| `TestRun_FormatFolderWritesAndChecksOnlyMemberFiles` | `internal/cli` | `fmt -w`, `fmt --check`, and `fmt <folder>`. A file in a subfolder is not changed or listed. |
| `TestRun_FormatFolderWithSyntaxErrorChangesNoFile` | `internal/cli` | A syntax error in one member changes no file |
| `TestRun_DiagramTitle` | `internal/cli` | `--title`, the default title, and the empty title |
| `TestRun_VersionJSONNamesToolAndSpecification` | `internal/cli` | `tool` and `spec` |
| `TestRun_RejectsInvalidArguments` | `internal/cli` | `-w` with `--check`, `diff` with one model, `--title` for `export`, and `version` with an argument |

`make verify` runs `skill-version-check`. A copy of `SKILL.md` with `spec_version: "v0.4.0"` makes it fail.

## 7. Alternatives and Unresolved Questions

The table lists the alternatives that this RFC does not take, and the reason against each one.

| Alternative | Reason against it |
| --- | --- |
| Add the judgment flag and the address to the text line | The text is for people. The agent skill and the scripts copy its form (deviation D6), so a change breaks them. |
| A new exit code for a policy rejection without core errors | The specification states that `validate` exits unsuccessfully on either. The key `has_core_errors` gives the same fact with no change to exit codes. |
| SARIF or GitLab Code Quality output | A CI job can convert the JSON object. A later RFC can add `--format sarif` with the same option. |
| Each caller compares source text | A moved block or a new layout counts as a change. Each caller becomes a second owner of identity rules. |
| A model title in the language | The title of a rendering is not a fact about the domain. The titles of the language belong to declarations. |

These questions stay open:

1. Chapters, hotspots, scenarios, and the board have no reference form. Does the specification need `chapter.<id>` and `hotspot.<id>`?
2. Does `diff` need a second list of the workflows that use a changed catalog declaration? This RFC reports only the declaration itself.
3. Does `serve` need `--title`?
4. Does `diagram` need `--profile`, so that a rendering shows the judgment signals as `serve --profile strict` does?
5. Does the viewer need a route for one workflow, so that a link can open one slice?
