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

missing=0
for dir in provider/defangaws provider/defanggcp provider/defangazure; do
  # Pulumi-owned plugin SDKs imported under this cloud's provider package,
  # reduced to the module identifier Dockerfile's grep patterns key off of
  # (e.g. "pulumi-aws/sdk/v7", "pulumi-azure-native-sdk", "pulumi-random/sdk/v4").
  # pulumi-go-provider is the provider framework, not a deployable plugin.
  imports=$(grep -rhoE 'github\.com/pulumi/pulumi-[a-z0-9-]+(/sdk/v[0-9]+)?' "$dir" --include='*.go' \
    | sed -E 's#^github\.com/pulumi/##' \
    | grep -v '^pulumi-go-provider$' \
    | grep -v '^pulumi-defang' \
    | sort -u)
  for imp in $imports; do
    if ! grep -qF "$imp" Dockerfile; then
      echo "error: $dir imports $imp, but no Dockerfile line installs a plugin for it" >&2
      missing=1
    fi
  done
done

if [ "$missing" -ne 0 ]; then
  echo >&2
  echo "A provider package imports a Pulumi plugin SDK that isn't precached." >&2
  echo "Add a 'pulumi plugin install resource <name> \$(grep ...)' line to the" >&2
  echo "relevant image stage(s) in Dockerfile (see plugins-*-upstream stages)." >&2
  exit 1
fi
echo "ok: every Pulumi plugin SDK imported by a provider package is precached in the Dockerfile"
