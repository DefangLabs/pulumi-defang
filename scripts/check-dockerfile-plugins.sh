#!/usr/bin/env bash
# Guards against a provider package importing a new Pulumi-owned plugin SDK
# (pulumi-aws, pulumi-gcp, pulumi-azure-native-sdk, pulumi-random, ...)
# without a matching `pulumi plugin install resource ...` line in the
# Dockerfile. Without that line the plugin isn't precached, so `defang-cd`
# downloads it on first use at deploy time instead -- harmless when the
# plugin is tiny, but a surprise nobody will notice until a deploy's logs
# are read line by line. See DefangLabs/pulumi-defang#670.
set -euo pipefail
cd "$(dirname "$0")/.."

# go.mod version for a given module path, e.g. module_version
# "pulumi-random/sdk/v4" go.mod -> "v4.21.2". Empty if not a direct/indirect
# dependency of that go.mod.
module_version() {
  awk -v module="github.com/pulumi/$1" '$1 == module { print $2; exit }' "$2"
}

# Install commands only, so a plugin identifier merely mentioned in a comment
# (the plugins-base rationale above names several by hand) doesn't satisfy
# the check for a line that was actually removed.
install_lines=$(grep -F 'plugin install resource' Dockerfile)

missing=0
for dir in provider/defangaws provider/defanggcp provider/defangazure; do
  # Pulumi-owned plugin SDKs imported under this cloud's provider package,
  # reduced to the module identifier Dockerfile's grep patterns key off of
  # (e.g. "pulumi-aws/sdk/v7", "pulumi-azure-native-sdk", "pulumi-random/sdk/v4").
  # pulumi-go-provider is the provider framework, not a deployable plugin.
  # - restricted to non-test production code: a test-only or commented-out
  #   import shouldn't need to be precached for a real deploy.
  # - the leading `"` requires a real quoted import path, not prose that
  #   happens to name an SDK (e.g. this file's own comments).
  imports=$( (grep -rhoE '"github\.com/pulumi/pulumi-[a-z0-9-]+(/sdk/v[0-9]+)?' "$dir" \
    --include='*.go' --exclude='*_test.go' || true) \
    | sed -E 's#^"github\.com/pulumi/##' \
    | grep -v '^pulumi-go-provider$' \
    | grep -v '^pulumi-defang' \
    | sort -u)
  for imp in $imports; do
    # Which go.mod a plugin's precached version is actually read from differs
    # per package (see the Dockerfile comment above the plugins-*-upstream
    # stages): mirror that here so a version that drifts between the two
    # go.mod files is caught, not just a missing identifier.
    module="$imp"
    install_file=go.mod
    case "$imp" in
      pulumi-azure-native-sdk) module="$imp/v3"; install_file=cd/go.mod ;;
      pulumi-aws/sdk/v7 | pulumi-gcp/sdk/v9) install_file=cd/go.mod ;;
    esac
    provider_version=$(module_version "$module" go.mod)
    installed_version=$(module_version "$module" "$install_file")

    if ! grep -qF "$imp" <<<"$install_lines"; then
      echo "error: $dir imports $imp, but no Dockerfile line installs a plugin for it" >&2
      missing=1
    elif [ -z "$provider_version" ] || [ "$provider_version" != "$installed_version" ]; then
      echo "error: $dir requires $module@$provider_version, but Dockerfile installs $module@$installed_version (from $install_file)" >&2
      missing=1
    fi
  done
done

if [ "$missing" -ne 0 ]; then
  echo >&2
  echo "A provider package imports a Pulumi plugin SDK that isn't precached," >&2
  echo "or precaches a version that no longer matches what go.mod requires." >&2
  echo "Add or update the 'pulumi plugin install resource <name> \$(grep ...)'" >&2
  echo "line for it in the relevant image stage(s) in Dockerfile." >&2
  exit 1
fi
echo "ok: every Pulumi plugin SDK imported by a provider package is precached in the Dockerfile, at the right version"
