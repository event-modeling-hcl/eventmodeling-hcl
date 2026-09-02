#!/usr/bin/env sh

set -eu

tag=${1:-}

if ! printf '%s\n' "$tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  printf 'stable release tag must use vMAJOR.MINOR.PATCH, got %s\n' "$tag" >&2
  exit 1
fi
