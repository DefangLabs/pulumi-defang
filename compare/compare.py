#!/usr/bin/env python3
"""
Compare two pulumi preview --json output files.

Handles:
- Mixed plaintext + JSON (extracts JSON objects by brace-depth tracking)
- Two schemas: old (new/old fields) and new (newState/oldState fields)
- Stack/project/org-specific values in URNs, names, tags
"""

import json
import re
import sys
from collections import Counter
from typing import Any


# ---------------------------------------------------------------------------
# Step 1: Extract JSON objects from a file with mixed content
# ---------------------------------------------------------------------------

def extract_json_objects(text: str) -> list[dict]:
    """
    Scan character-by-character, tracking brace depth.
    Emit a parsed dict each time depth returns to 0.
    Skips any non-JSON plaintext.
    """
    objects = []
    depth = 0
    buf = []
    in_string = False
    escape_next = False

    for ch in text:
        if escape_next:
            buf.append(ch)
            escape_next = False
            continue

        if ch == '\\' and in_string:
            buf.append(ch)
            escape_next = True
            continue

        if ch == '"' and not escape_next:
            in_string = not in_string

        if not in_string:
            if ch == '{':
                depth += 1
            elif ch == '}':
                depth -= 1

        if depth > 0 or (depth == 0 and buf):
            buf.append(ch)

        if depth == 0 and buf:
            raw = ''.join(buf).strip()
            if raw:
                try:
                    objects.append(json.loads(raw))
                except json.JSONDecodeError as e:
                    print(f"  [warn] failed to parse JSON object: {e}", file=sys.stderr)
            buf = []

    return objects


# ---------------------------------------------------------------------------
# Step 2: Normalize to a common schema
# ---------------------------------------------------------------------------

def normalize_resource(obj: dict) -> dict | None:
    """
    Map both schema variants to a common shape:
      { op, urn, type, inputs }
    Returns None if the object is not a resource step.
    """
    if 'op' not in obj:
        return None

    # Determine which schema variant we have
    state = dict(obj.get('newState') or obj.get('new') or {})
    state.pop('stackTrace', None)
    state.pop('sourcePosition', None)
    state.pop('propertyDependencies', None)

    resource_type = (
        obj.get('type')
        or state.get('type')
    )
    if not resource_type:
        return None

    urn = obj.get('urn', '')
    inputs = state.get('inputs', {}) or {}
    service_tag = (inputs.get('tags') or {}).get('defang:service')

    return {
        'op': obj.get('op'),
        'urn': urn,
        'type': resource_type,
        'inputs': inputs,
        'service_tag': service_tag,
    }


# ---------------------------------------------------------------------------
# Step 3: Derive a stable comparison key from the URN
# ---------------------------------------------------------------------------

# Matches trailing Pulumi-generated hash suffixes like "-ba695e9" (7 hex chars)
_HASH_SUFFIX = re.compile(r'-[0-9a-f]{7}$')

# Matches a provider UUID in a URN like "::04da6b54-80e4-46f7-96ec-b56ff0331ba9"
_PROVIDER_UUID = re.compile(r'::[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')

def urn_to_key(urn: str) -> tuple[str, str]:
    """
    Extract (resource_type, resource_name) from a Pulumi URN.

    URN format: urn:pulumi:STACK::PROJECT::TYPE_CHAIN::NAME
    TYPE_CHAIN may be: ParentType$ParentType$ResourceType
    We want just the innermost ResourceType and the NAME.
    """
    # Strip "urn:pulumi:STACK::PROJECT::" prefix
    # URN has the form: urn:pulumi:X::Y::REST where REST = TYPE_CHAIN::NAME
    parts = urn.split('::')
    if len(parts) < 4:
        return (urn, '')
    # parts[0] = "urn:pulumi:STACK", parts[1] = "PROJECT", parts[2] = TYPE_CHAIN, parts[3] = NAME
    type_chain = parts[2]
    name = parts[3]

    # Take the innermost type (last $-separated segment)
    resource_type = type_chain.split('$')[-1]

    # Strip hash suffix from name
    name = _HASH_SUFFIX.sub('', name)

    return (resource_type, name)


# ---------------------------------------------------------------------------
# Step 4: Normalize inputs for comparison
# ---------------------------------------------------------------------------

# Fields that are noise for cross-stack comparison
_SKIP_INPUT_KEYS = {'__defaults', 'tagsAll', 'tags', 'region', 'name'}

# Pulumi uses a fixed UUID as a placeholder for unresolved output references at preview time
_PULUMI_UNKNOWN = re.compile(
    r'^04da6b54-80e4-46f7-96ec-b56ff0331ba9$'
    r'|^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
)


def normalize_value(v: Any) -> Any:
    """Recursively normalize a value for comparison."""
    if isinstance(v, dict):
        result = {}
        for k, val in sorted(v.items()):
            if k in _SKIP_INPUT_KEYS:
                continue
            # Drop self: false from SG rule objects (always the default)
            if k == 'self' and val is False:
                continue
            result[k] = normalize_value(val)
        return result
    if isinstance(v, list):
        return [normalize_value(i) for i in v]
    if isinstance(v, str):
        # Replace unresolved output references with a readable token
        if _PULUMI_UNKNOWN.match(v):
            return '<ref>'
        # Normalize provider URNs (strip UUID suffix)
        v = _PROVIDER_UUID.sub('', v)
        # Normalize hash-suffixed resource names within strings
        v = _HASH_SUFFIX.sub('', v)
        # Replace redacted stack references
        v = v.replace('s***', 'STACK')
        return v
    return v


def normalize_inputs(inputs: dict) -> dict:
    return normalize_value(inputs)


# ---------------------------------------------------------------------------
# Step 5: Load and index a file
# ---------------------------------------------------------------------------

def load_file(path: str) -> dict[tuple[str, str], dict]:
    """
    Returns a dict mapping (resource_type, resource_name) -> normalized resource.
    """
    with open(path) as f:
        text = f.read()

    raw_objects = extract_json_objects(text)
    index = {}

    for obj in raw_objects:
        r = normalize_resource(obj)
        if r is None:
            continue
        key = urn_to_key(r['urn'])
        r['inputs'] = normalize_inputs(r['inputs'])
        index[key] = r

    return index


# ---------------------------------------------------------------------------
# Step 6: Compare and report
# ---------------------------------------------------------------------------

def compare(path_a: str, path_b: str, all_types_flag: bool = False):
    index_a = load_file(path_a)
    index_b = load_file(path_b)

    keys_a = set(index_a)
    keys_b = set(index_b)

    name_a = path_a.split('/')[-1]
    name_b = path_b.split('/')[-1]
    print(f"{'Resource type':<55} {name_a:>4} {name_b:>4}")
    print("-" * 65)

    types_a = Counter(rtype for rtype, _ in keys_a)
    types_b = Counter(rtype for rtype, _ in keys_b)
    all_types = sorted(set(types_a) | set(types_b))

    _IGNORE_PREFIXES = (
        'defang-gcp:', 'defang-aws:', 'defang-azure:',
        'pulumi:providers:gcp', 'pulumi:providers:aws', 'pulumi:providers:azure',
        'pulumi:pulumi:Stack',
    )

    for rtype in all_types:
        if not all_types_flag and rtype.startswith(_IGNORE_PREFIXES):
            continue
        a_count = types_a.get(rtype, 0)
        b_count = types_b.get(rtype, 0)
        if not all_types_flag and a_count == b_count:
            continue
        print(f"  {rtype:<55} {a_count:>4} {b_count:>4}")

    total_a, total_b = sum(types_a.values()), sum(types_b.values())
    label = 'TOTAL'
    print("-" * 65)
    print(f"  {label:<53} {total_a:>4} {total_b:>4}")


def diff_dicts(a: dict, b: dict, indent: int = 4) -> list[str]:
    lines = []
    pad = ' ' * indent
    for k in sorted(set(a) | set(b)):
        if k not in b:
            lines.append(f"{pad}- {k}: {json.dumps(a[k])}")
        elif k not in a:
            lines.append(f"{pad}+ {k}: {json.dumps(b[k])}")
        elif a[k] != b[k]:
            lines.append(f"{pad}- {k}: {json.dumps(a[k])}")
            lines.append(f"{pad}+ {k}: {json.dumps(b[k])}")
    return lines


def correlate(resources_a: dict, resources_b: dict) -> list[tuple]:
    """
    Match resources across two sets using:
      1. defang:service tag (exact match)
      2. name-suffix match (one name is a suffix of the other)

    Returns a list of (name_a, name_b, warning) tuples where either side may be
    None for unmatched resources. Warning is set for ambiguous suffix matches.
    """
    matched_a, matched_b = set(), set()
    pairs = []

    # Pass 1: exact name matches
    for name in sorted(set(resources_a) & set(resources_b)):
        pairs.append((name, name, None))
        matched_a.add(name)
        matched_b.add(name)

    # Pass 2: defang:service tag match
    unmatched_a = {n: r for n, r in resources_a.items() if n not in matched_a}
    unmatched_b = {n: r for n, r in resources_b.items() if n not in matched_b}

    by_tag_b = {}
    for name, r in unmatched_b.items():
        tag = r.get('service_tag')
        if tag:
            by_tag_b.setdefault(tag, []).append(name)

    for name_a, r in sorted(unmatched_a.items()):
        tag = r.get('service_tag')
        if tag and tag in by_tag_b and len(by_tag_b[tag]) == 1:
            name_b = by_tag_b[tag][0]
            pairs.append((name_a, name_b, None))
            matched_a.add(name_a)
            matched_b.add(name_b)

    # Pass 3: name-suffix match
    unmatched_a = {n: r for n, r in resources_a.items() if n not in matched_a}
    unmatched_b = {n: r for n, r in resources_b.items() if n not in matched_b}

    for name_a in sorted(unmatched_a):
        candidates = [nb for nb in unmatched_b if nb.endswith(name_a) or name_a.endswith(nb)]
        if len(candidates) == 1:
            name_b = candidates[0]
            pairs.append((name_a, name_b, None))
            matched_a.add(name_a)
            matched_b.add(name_b)
        elif len(candidates) > 1:
            # Ambiguous — report but don't match
            pairs.append((name_a, None, f"ambiguous suffix match: {candidates}"))
            matched_a.add(name_a)

    # Remaining unmatched
    for name in sorted(set(resources_a) - matched_a):
        pairs.append((name, None, None))
    for name in sorted(set(resources_b) - matched_b):
        pairs.append((None, name, None))

    return pairs


def compare_type(path_a: str, path_b: str, resource_type: str):
    index_a = load_file(path_a)
    index_b = load_file(path_b)

    name_a = path_a.split('/')[-1]
    name_b = path_b.split('/')[-1]

    resources_a = {name: r for (rtype, name), r in index_a.items() if rtype == resource_type}
    resources_b = {name: r for (rtype, name), r in index_b.items() if rtype == resource_type}

    if not resources_a and not resources_b:
        print(f"No resources of type '{resource_type}' found in either file.")
        sys.exit(1)

    print(f"Resource type: {resource_type}")
    print(f"  {name_a}: {len(resources_a)}  {name_b}: {len(resources_b)}")

    pairs = correlate(resources_a, resources_b)

    for na, nb, warning in pairs:
        if warning:
            print(f"\n  [warn] {na}: {warning}")
        elif na and nb:
            label = na if na == nb else f"{na}  <->  {nb}"
            diff = diff_dicts(resources_a[na]['inputs'], resources_b[nb]['inputs'])
            if diff:
                print(f"\n  {label}  (- {name_a}  + {name_b})")
                for line in diff:
                    print(line)
            else:
                print(f"\n  {label}  (identical)")
        elif na:
            print(f"\n  {na}  (only in {name_a})")
            for line in diff_dicts({}, resources_a[na]['inputs']):
                print(line)
        else:
            print(f"\n  {nb}  (only in {name_b})")
            for line in diff_dicts({}, resources_b[nb]['inputs']):
                print(line)


if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('file_a')
    parser.add_argument('file_b')
    parser.add_argument('--all', action='store_true', dest='all_types',
                        help='include non-aws: resource types')
    parser.add_argument('--type', dest='resource_type',
                        help='show property-level diff for a specific resource type')
    args = parser.parse_args()
    if args.resource_type:
        compare_type(args.file_a, args.file_b, args.resource_type)
    else:
        compare(args.file_a, args.file_b, all_types_flag=args.all_types)
