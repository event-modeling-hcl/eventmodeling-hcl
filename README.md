# Event Modeling HCL Specification

[![CI](https://github.com/dclimber/event-modeling-hcl/actions/workflows/ci.yml/badge.svg)](https://github.com/dclimber/event-modeling-hcl/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/dclimber/event-modeling-hcl.svg)](https://pkg.go.dev/github.com/dclimber/event-modeling-hcl)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Native HCL v1 for the Event Modeling Specification. This repository provides a
strict validator, normative language documentation, and executable examples.
The domain reference is the upstream [Event Modeling
Specification](https://github.com/dilgerma/event-modeling-spec); the HCL
specification and validator in this repository define its native v1 port. The
format is not a JSON embedding or conversion format.

## Requirements and Installation

Install Go 1.25 or newer, download a release
archive, or build the validator from source:

```bash
go install github.com/dclimber/event-modeling-hcl/cmd/eventmodeling-hcl@v0.1.0
# or, from a checkout:
go build -o bin/eventmodeling-hcl ./cmd/eventmodeling-hcl
```

Release archives for Linux, macOS, and Windows are available from the
[GitHub Releases page](https://github.com/dclimber/event-modeling-hcl/releases).
Verify the downloaded archive with the release's `checksums.txt` before use.

Validate one complete model per invocation:

```bash
./bin/eventmodeling-hcl validate examples/minimal.em.hcl
./bin/eventmodeling-hcl validate examples/complete.em.hcl
./bin/eventmodeling-hcl validate examples/pet-management-detailed.em.hcl
```

A valid document prints `<path> valid`. Validation errors use the form
`file:line:column: Error: message` and return a nonzero exit status.

Model files must use the `.em.hcl` extension. HCL is the language; the suffix
identifies a complete Event Modeling document to this validator. Check the
installed binary version with `eventmodeling-hcl version`.

## Authoring

Only `slice` blocks are allowed at top level. A block label carries the JSON
identity property, while remaining properties use snake_case attributes.
Events remain direct children of their owning slice.

```hcl
slice "add-pet" {
  title      = "Add Pet"
  slice_type = "STATE_CHANGE"

  command "add-pet" {
    title = "Add Pet"
    type  = "COMMAND"

    dependency "evt-pet-001" {
      type         = "OUTBOUND"
      title        = "Pet Added"
      element_type = "EVENT"
    }
  }

  event "evt-pet-001" {
    title = "Pet Added"
    type  = "EVENT"
  }
}
```

Repeated blocks represent source arrays; omitting a collection means an empty
array. Dependency IDs and `linked_id` values are opaque strings, so v1 does
not resolve references or add graph-validation rules. HCL values must be
literals: variables, functions, and interpolation are rejected.

Use native values for field examples:

```hcl
field "pets" {
  type        = "Custom"
  cardinality = "List"
  example = [{
    id   = 5
    name = "Mochi"
  }]
}
```

## Documentation and Examples

- [Language specification](eventmodeling.hclspec.md): grammar, nesting,
  required attributes, enums, literals, and validation boundary.
- [Source-model mapping](schema/eventmodeling.hcl.schema.md): one-to-one
  mapping from every source property to HCL.
- [Minimal example](examples/minimal.em.hcl): smallest useful model.
- [Complete reference](examples/complete.em.hcl): every supported HCL construct.
- [Pet-management port](examples/pet-management-detailed.em.hcl): the supplied
  five-slice golden model in native HCL.
- [Mapping ADR](docs/adr/0001-slice-local-events.md): direct slice-local event
  design decision.

## Verification

Run the complete local verification suite before submitting a change:

```bash
make verify
```

`make help` lists individual commands. The most commonly used are `make build`,
`make test`, and `make validate-examples`.

The test suite validates every `.em.hcl` file in `examples/` and asserts every
fixture in `testdata/invalid/` fails validation.

The original pet-management JSON fixture is retained under `testdata/golden/`
as reference data for the native HCL example.

## v1 Boundaries

v1 does not support event groups, external-event blocks, qualified event IDs,
`emits`, multi-file loading, formatting, JSON conversion, editor integration,
or cross-reference resolution.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
