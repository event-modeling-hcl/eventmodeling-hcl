#!/usr/bin/env sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
validator="$script_dir/require-annotated-tag.sh"
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

origin="$temporary/origin.git"
source="$temporary/source"
checkout="$temporary/checkout"

git init --bare --initial-branch main "$origin" >/dev/null
git init --initial-branch main "$source" >/dev/null
git -C "$source" config user.email release-test@example.invalid
git -C "$source" config user.name "Release Test"
touch "$source/model.em.hcl"
git -C "$source" add model.em.hcl
git -C "$source" commit -m "Initial model" >/dev/null
git -C "$source" remote add origin "$origin"
git -C "$source" push origin main >/dev/null 2>&1
git -C "$source" tag -a v0.1.0 -m "Release v0.1.0"
git -C "$source" push origin v0.1.0 >/dev/null 2>&1
git clone --no-tags "$origin" "$checkout" >/dev/null 2>&1

if git -C "$checkout" show-ref --verify --quiet refs/tags/v0.1.0; then
  printf 'test checkout unexpectedly contains v0.1.0\n' >&2
  exit 1
fi
if ! (cd "$checkout" && sh "$validator" v0.1.0) >/dev/null 2>&1; then
  printf 'expected an annotated tag to be accepted after fetching it\n' >&2
  exit 1
fi

git -C "$source" tag v0.2.0
git -C "$source" push origin v0.2.0 >/dev/null 2>&1
if (cd "$checkout" && sh "$validator" v0.2.0) >/dev/null 2>&1; then
  printf 'expected a lightweight tag to be rejected\n' >&2
  exit 1
fi
