#!/usr/bin/env python3
"""Fail unless the version the examples now point at is really on the registries.

Usage: check_sdks_published.py FILES_JSONL BRANCH

Which registries get checked follows the example directories the PR touched:
examples/<cloud>-<language>/... . Go is skipped: the Go examples resolve through
the module proxy, which has no versions for this repo's SDK path.
"""

import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

# A release publishes minutes earlier and NuGet indexing lags; the env vars are
# there so a maintainer debugging this job does not wait out the retries.
ATTEMPTS = int(os.environ.get("SDK_CHECK_ATTEMPTS", "5"))
DELAY = int(os.environ.get("SDK_CHECK_DELAY", "30"))

REGISTRIES = {
    "nodejs": ("npm", "https://registry.npmjs.org/@defang-io/pulumi-defang-{cloud}",
               lambda d: list(d.get("versions", {}))),
    "python": ("PyPI", "https://pypi.org/pypi/pulumi-defang-{cloud}/json",
               lambda d: list(d.get("releases", {}))),
    "dotnet": ("NuGet", "https://api.nuget.org/v3-flatcontainer/defanglabs.defang{cloud}/index.json",
               lambda d: d.get("versions", [])),
}


def fail(msg):
    print(f"::error::{msg}")
    sys.exit(1)


def versions(url, extract):
    with urllib.request.urlopen(url, timeout=30) as resp:
        return extract(json.load(resp))


def check(name, url, extract, version):
    for attempt in range(1, ATTEMPTS + 1):
        try:
            if version in versions(url, extract):
                print(f"  {name}: {version} published")
                return
            why = f"{version} not listed yet"
        except (urllib.error.URLError, OSError, ValueError) as err:
            why = str(err)
        if attempt < ATTEMPTS:
            print(f"  {name}: {why}; retrying in {DELAY}s ({attempt}/{ATTEMPTS})")
            time.sleep(DELAY)
    fail(f"{name} does not have {version} at {url} — leaving this PR for a human")


def main():
    if len(sys.argv) != 3:
        fail("usage: check_sdks_published.py FILES_JSONL BRANCH")
    version = re.fullmatch(r"chore/regenerate-examples-v(\d+\.\d+\.\d+)", sys.argv[2])
    if not version:
        fail(f"branch {sys.argv[2]!r} is not chore/regenerate-examples-vX.Y.Z")
    version = version.group(1)

    with open(sys.argv[1]) as fh:
        paths = [json.loads(line)["filename"] for line in fh if line.strip()]

    targets = set()
    for path in paths:
        m = re.match(r"examples/([a-z0-9]+)-([a-z0-9]+)/", path)
        if m:
            targets.add(m.groups())

    checked = 0
    for cloud, language in sorted(targets):
        if language not in REGISTRIES:
            print(f"  {cloud}-{language}: no registry check")
            continue
        name, url, extract = REGISTRIES[language]
        check(f"{name} {cloud}", url.format(cloud=cloud), extract, version)
        checked += 1

    if not checked:
        fail("no registry could be checked for this diff")
    print(f"{checked} packages confirmed published at {version}.")


if __name__ == "__main__":
    main()
