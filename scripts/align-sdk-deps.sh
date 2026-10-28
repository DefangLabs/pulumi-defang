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
# Usage: scripts/align-sdk-deps.sh <sdk-module-dir>
set -euo pipefail

sdk_dir=$1
root_dir=$(cd "$(dirname "$0")/.." && pwd)

list_modules() {
  (cd "$1" && go list -m -f '{{if not .Main}}{{.Path}} {{.Version}}{{end}}' all)
}

declare -A root_version
while read -r mod ver; do
  [ -n "$mod" ] && root_version[$mod]=$ver
done < <(list_modules "$root_dir")

requires=()
while read -r mod ver; do
  want=${root_version[$mod]:-}
  if [ -n "$want" ] && [ "$want" != "$ver" ]; then
    requires+=("-require=$mod@$want")
  fi
done < <(list_modules "$sdk_dir")

if [ ${#requires[@]} -gt 0 ]; then
  (cd "$sdk_dir" && go mod edit "${requires[@]}" && go mod tidy)
fi
