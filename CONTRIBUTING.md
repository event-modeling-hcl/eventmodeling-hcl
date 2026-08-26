# Contributing

Thanks for considering a contribution to `event-modeling-hcl`.

## Development setup

Install Go 1.25 or newer, then build and test from a checkout:

```bash
go build ./...
make verify
```

- `make help` lists individual commands.
- `make verify` runs the same checks as CI (Run it before opening a pull request):
  - `gofmt` formatting,
  - `go mod tidy` drift,
  - `go vet`,
  - unit tests,
  - race tests,
  - `staticcheck`,
  - `govulncheck`,
  - and validation of every example in `examples/`.
  
## Code conventions

- Keep the package layout as-is: `cmd/eventmodeling-hcl` is the thin CLI, the
  parsing/validation engine lives in `internal/validator` and is not a public
  API.
- Prefer table-driven tests with `t.Run` subtests, matching the existing style
  in `internal/validator/validator_test.go` and
  `cmd/eventmodeling-hcl/main_test.go`.
- Diagnostics use `hcl.Diagnostics`: a short, no-period `Summary` and a full
  sentence `Detail`, matching the existing HCL/Terraform ecosystem
  convention.
- Exported identifiers need a doc comment starting with the identifier name.

## Changing the language grammar

`eventmodeling.hclspec.md` is the normative v1 grammar. Any change to what
the validator accepts must update that document, `schema/eventmodeling.hcl.schema.md`
if the source-model mapping changes, and `CHANGELOG.md`.

## Pull requests

- Keep changes focused; unrelated cleanup belongs in a separate PR.
- Update `CHANGELOG.md` for user-visible changes.
- Make sure `make verify` passes before requesting review.
