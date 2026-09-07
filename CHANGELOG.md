# Changelog

This repository records implementation releases of the validator, formatter,
and typed IR. The authoritative Event Modeling HCL language history is in the
[specification changelog](https://github.com/event-modeling-hcl/spec/blob/main/CHANGELOG.md).

## [Unreleased]

## [v0.3.1] - 2026-09-07

This release polishes the diagram canvas introduced in v0.3.0.

### Changed

- Actor cards render directly beside their associated screen card instead of
  in a dedicated actor swimlane.
- Backward-pointing flow arrows (whose target sits left of their source in
  the left-to-right layout) render with a dashed stroke, reverting to solid
  only while hovered.

## [v0.3.0] - 2026-09-06

This release adds rendering tooling while retaining full compatibility with the
v0.2.0 Event Modeling HCL language specification.

### Added

- `diagram <model.em.hcl> [-o <file>]` command for generating a self-contained,
  interactive HTML Event Model canvas from the validated typed IR.
- Workflow-level `screen_image` previews, canonical flow arrows, slice filters,
  scenario details, field metadata, chapters, and hotspots in rendered diagrams.
- Content-adaptive left-to-right slice layouts with dedicated actor and processor
  swimlanes, horizontally grouped events, and actor-to-screen highlighting.
- Event Storming pink styling for events owned by external bounded contexts,
  while domain events retain their orange styling.

### Changed

- The typed IR retains field-type ID/PII metadata and scenario examples needed
  by downstream renderers.

## [v0.2.0] - 2026-09-05

Implements the v0.2.0 native HCL language surface. See the
[specification changelog](https://github.com/event-modeling-hcl/spec/blob/main/CHANGELOG.md)
for the normative language history.

### Added

- `fmt` command that canonicalizes whitespace and attribute order idempotently
  while preserving semantic block and scenario order.
- Validation profiles `--profile workshop|valid|strict`, with `valid` as the
  default and `strict` escalating unreasoned commands and open hotspots to errors.
- Stable `EMxxx` diagnostic codes, reported as
  `file:line:column: Severity EMxxx: message`.
- Typed IR via `internal/model.Load`: a validation-gated model with effective
  titles, normalized source-to-target edges, and semantic/presentation separation.
- Invalid fixtures for reverse-flow and State-View `when` scenarios.

### Changed

- Moves the public Go module to
  `github.com/event-modeling-hcl/eventmodeling-hcl`.
- Enforces one canonical flow spelling per edge; reverse forms now fail validation.
- State View scenarios take one or more event `given` steps and `then` read-model
  steps, with no `when`.
- Titles are optional and derived from labels when absent.

### Removed

- The `query` scenario target and reverse flow spellings.

## [v0.1.0] - 2026-09-02

- Initial native HCL validator release.
