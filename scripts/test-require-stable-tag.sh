#!/usr/bin/env sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
validator="$script_dir/require-stable-tag.sh"

assert_accepts() {
  tag=$1
  if ! sh "$validator" "$tag" >/dev/null 2>&1; then
    printf 'expected stable tag %s to be accepted\n' "$tag" >&2
    exit 1
  fi
}

assert_rejects() {
  tag=$1
  if sh "$validator" "$tag" >/dev/null 2>&1; then
    printf 'expected non-stable tag %s to be rejected\n' "$tag" >&2
    exit 1
  fi
}

assert_accepts v0.1.0
assert_accepts v1.2.3
assert_accepts v12.34.56
assert_rejects v0.1.0b0
assert_rejects v0.1.0-rc.1
assert_rejects v01.2.3
assert_rejects 0.1.0
assert_rejects v1.2
assert_rejects v1.2.3.4
