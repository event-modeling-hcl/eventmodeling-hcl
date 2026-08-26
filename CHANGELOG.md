# Changelog

All notable changes to this project are documented in this file.

## v0.1.0

- Initial native HCL v1 Event Modeling Specification release.
- Strict `eventmodeling-hcl validate <model.em.hcl>` validator with literal-only
  HCL evaluation and source-located diagnostics.
- Native `.em.hcl` examples, including the complete reference model and the
  pet-management golden model.
- Go 1.25+ is required to include the fixed `golang.org/x/text` dependency for
  `GO-2026-5970`.
- Optional field `pii` metadata is represented as `pii = <bool>` in HCL.

### Scope

V1 keeps events slice-local and does not provide bounded contexts, event
groups, external-event blocks, cross-reference resolution, multi-file models,
formatting, conversion, generators, or editor integration.
