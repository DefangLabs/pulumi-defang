#!/usr/bin/env bash
# Raise every module a generated Go SDK shares with the root module to at
# least the root's version.
#
# `make go_sdk` recreates each SDK's go.mod from scratch, so its indirect
# dependencies resolve to whatever the Pulumi SDK itself requires, however
# old. Dependabot cannot bump them there (they are indirect, and the next
# regeneration would undo it anyway), so security fixes it lands in the root
# go.mod never reached the published SDKs. Requiring the root's version lets
# MVS keep the higher of the two, so this never downgrades anything.
#
# Raising versions can pull in further shared modules, so it repeats until a
# pass finds no new ones. Each module is required at most once: one that still
# differs afterwards is one the SDK needs at a higher version than the root.
# Bash 3.2-compatible (macOS's system bash).
#
# Usage: scripts/align-sdk-deps.sh <sdk-module-dir>
set -euo pipefail

sdk_dir=$1
root_dir=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

list_modules() {
  (cd "$1" && go list -m -f '{{if not .Main}}{{.Path}} {{.Version}}{{end}}' all) > "$2"
}

list_modules "$root_dir" "$tmp/root"
: > "$tmp/done"

for _ in 1 2 3 4 5 6 7 8 9 10; do
  list_modules "$sdk_dir" "$tmp/sdk"
  awk 'FILENAME == ARGV[1] { done[$1] = 1; next }
       FILENAME == ARGV[2] { if ($1 != "") want[$1] = $2; next }
       ($1 in want) && !($1 in done) && want[$1] != $2 {
         print "-require=" $1 "@" want[$1]
       }' "$tmp/done" "$tmp/root" "$tmp/sdk" > "$tmp/requires"
  if [ ! -s "$tmp/requires" ]; then
    exit 0
  fi
  sed -e 's/^-require=//' -e 's/@.*//' "$tmp/requires" >> "$tmp/done"
  # shellcheck disable=SC2046 # one flag per line, no spaces in module@version
  (cd "$sdk_dir" && go mod edit $(cat "$tmp/requires") && go mod tidy)
done

echo "align-sdk-deps: $sdk_dir still differs from the root after 10 passes:" >&2
cat "$tmp/requires" >&2
exit 1
