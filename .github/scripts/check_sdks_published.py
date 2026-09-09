#!/usr/bin/env python3
"""Fail unless the version the examples now point at is really on the registries.

Usage: check_sdks_published.py FILES_JSONL VERSION

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

# "exists": the URL is the version itself, so 200 means published and 404 does not.
# "listed": NuGet has no per-version JSON endpoint, so read its index instead.
REGISTRIES = {
    "nodejs": ("npm", "exists",
               "https://registry.npmjs.org/@defang-io/pulumi-defang-{cloud}/{version}"),
    "python": ("PyPI", "exists",
               "https://pypi.org/pypi/pulumi-defang-{cloud}/{version}/json"),
    "dotnet": ("NuGet", "listed",
               "https://api.nuget.org/v3-flatcontainer/defanglabs.defang{cloud}/index.json"),
}


def fail(msg):
    print(f"::error::{msg}")
    sys.exit(1)


def published(url, kind, version):
    """Return (published, why-not). Every failure is retried: a package can appear."""
    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            if kind == "exists":
                return True, None
            listed = json.load(resp).get("versions", [])
            return version in listed, f"{version} not listed yet"
    except urllib.error.HTTPError as err:
        return False, "not published yet" if err.code == 404 else str(err)
    except (urllib.error.URLError, OSError, ValueError) as err:
        return False, str(err)


def check(name, url, kind, version):
    for attempt in range(1, ATTEMPTS + 1):
        ok, why = published(url, kind, version)
        if ok:
            print(f"  {name}: {version} published")
            return
        if attempt < ATTEMPTS:
            print(f"  {name}: {why}; retrying in {DELAY}s ({attempt}/{ATTEMPTS})")
            time.sleep(DELAY)
    fail(f"{name} does not have {version} at {url} — leaving this PR for a human")


def main():
    if len(sys.argv) != 3:
        fail("usage: check_sdks_published.py FILES_JSONL VERSION")
    version = sys.argv[2]
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"{version!r} is not a plain X.Y.Z release version")

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
        name, kind, url = REGISTRIES[language]
        check(f"{name} {cloud}", url.format(cloud=cloud, version=version), kind, version)
        checked += 1

    if not checked:
        fail("no registry could be checked for this diff")
    print(f"{checked} packages confirmed published at {version}.")


if __name__ == "__main__":
    main()
