#!/usr/bin/env python3
"""Fail unless a regenerate-examples PR is only this release's version bump.

Usage: check_examples_bump.py FILES_JSONL BRANCH

FILES_JSONL is one JSON object per line from the pulls/N/files API; BRANCH is the
PR's head ref, `chore/regenerate-examples-vX.Y.Z`.
"""

import json
import re
import sys

SEMVER = re.compile(r"\d+\.\d+\.\d+")
MANIFESTS = ("go.mod", "package.json", "requirements.txt", ".csproj")
MAX_FILES = 40  # 3 clouds x 4 languages today, with room to grow


def fail(msg):
    print(f"::error::{msg}")
    sys.exit(1)


def version_from_branch(branch):
    m = re.fullmatch(r"chore/regenerate-examples-v(\d+\.\d+\.\d+)", branch)
    if not m:
        fail(f"branch {branch!r} is not chore/regenerate-examples-vX.Y.Z")
    return m.group(1)


def check_patch(path, patch, version):
    added, removed = [], []
    for line in patch.splitlines():
        if line.startswith(("+++", "---")):
            continue
        if line.startswith("+"):
            added.append(line[1:])
        elif line.startswith("-"):
            removed.append(line[1:])

    if not added or len(added) != len(removed):
        fail(f"{path}: {len(removed)} lines removed but {len(added)} added; "
             "a pure version bump replaces each line one for one")

    for old, new in zip(removed, added):
        if SEMVER.sub("#", old) != SEMVER.sub("#", new):
            fail(f"{path}: line changes more than a version number:\n  -{old}\n  +{new}")
        if version not in new:
            fail(f"{path}: line does not bump to {version}:\n  +{new}")


def main():
    if len(sys.argv) != 3:
        fail("usage: check_examples_bump.py FILES_JSONL BRANCH")
    version = version_from_branch(sys.argv[2])

    with open(sys.argv[1]) as fh:
        files = [json.loads(line) for line in fh if line.strip()]

    if not files:
        fail("the PR changes no files")
    if len(files) > MAX_FILES:
        fail(f"{len(files)} files changed; a regeneration touches one manifest per example")

    for f in files:
        path = f["filename"]
        if not path.startswith("examples/"):
            fail(f"{path} is outside examples/")
        if not path.endswith(MANIFESTS):
            fail(f"{path} is not an example manifest ({', '.join(MANIFESTS)})")
        if f["status"] != "modified":
            fail(f"{path} is {f['status']}, not modified")
        if "patch" not in f:
            fail(f"{path} has no patch to check")
        check_patch(path, f["patch"], version)

    print(f"{len(files)} example manifests, all bumped to {version} and nothing else.")


if __name__ == "__main__":
    main()
