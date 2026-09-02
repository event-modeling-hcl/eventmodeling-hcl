# Releasing

This runbook publishes a stable release of `eventmodeling-hcl`. Stable release
tags use exactly `vMAJOR.MINOR.PATCH`; prerelease tags such as `v0.1.0b0` do
not publish a stable release.

## Repository Settings

Configure these controls in the GitHub repository before the first stable
release:

- Set `main` as the default branch.
- Require pull requests before merging to `main`.
- Require the CI `verify` and `build` checks to pass before merging.
- Disallow force pushes and branch deletion for `main`.
- Allow GitHub Actions to create releases and publish artifact attestations.

## Preflight

Run the full gate from a clean checkout with a writable Go build cache:

```bash
git switch main
git pull --ff-only origin main
git status --short
make verify
```

`git status --short` must produce no output. Review the `v0.1.0` changelog
entry and make any release-note correction before continuing.

## Publish

Push the verified commit, then create an annotated tag that points to it:

```bash
git push origin main
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

Do not move, replace, or reuse `v0.1.0b0`; it remains the historical
pre-release tag. Do not delete or retag a published stable release.

## Verify the GitHub Release

The Release workflow reruns `make verify`, creates these assets with
GoReleaser, and publishes GitHub provenance attestations:

- Linux: `amd64` and `arm64` `.tar.gz` archives.
- macOS: `amd64` and `arm64` `.tar.gz` archives.
- Windows: `amd64` and `arm64` `.zip` archives.
- `checksums.txt`.

Download an archive and `checksums.txt` from the GitHub release, then verify
their contents and provenance:

```bash
sha256sum -c checksums.txt
gh attestation verify eventmodeling-hcl_0.1.0_linux_amd64.tar.gz \
  --repo dclimber/event-modeling-hcl
```

Install one archive on each supported operating-system family and confirm the
embedded release version:

```bash
eventmodeling-hcl version
# eventmodeling-hcl v0.1.0
```
