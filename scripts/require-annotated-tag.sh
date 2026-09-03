#!/usr/bin/env sh

set -eu

tag=${1:-}

if ! git check-ref-format "refs/tags/$tag"; then
  printf 'invalid tag name: %s\n' "$tag" >&2
  exit 1
fi

git fetch --force --no-tags origin "refs/tags/$tag:refs/tags/$tag"

if [ "$(git cat-file -t "refs/tags/$tag")" != tag ]; then
  printf 'release tag %s must be annotated\n' "$tag" >&2
  exit 1
fi
