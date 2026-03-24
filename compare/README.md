# compare

Tools for comparing `pulumi preview --json` output across two stacks.

## compare.py

Compares resource counts and properties between two preview output files.

Input files may contain mixed plaintext and JSON (e.g. progress messages alongside resource steps). Both the old (`new`/`old`) and new (`newState`/`oldState`) Pulumi preview schemas are supported.

The fields `stackTrace`, `sourcePosition`, and `propertyDependencies` are automatically stripped from resource state before comparison.

### Usage

**Resource type summary** — shows counts per AWS resource type, only where they differ:

```
python3 compare.py <file_a> <file_b>
```

Pass `--all` to include non-`aws:` types (providers, dynamic resources, custom components):

```
python3 compare.py <file_a> <file_b> --all
```

**Property-level diff** — drill into a specific resource type:

```
python3 compare.py <file_a> <file_b> --type aws:ecs/service:Service
```

### Example output

```
Resource type                                           oss.txt cd.txt
-----------------------------------------------------------------
  aws:ecs/service:Service                                    5    4
  aws:iam/role:Role                                          7    9
  ...
-----------------------------------------------------------------
  TOTAL (aws: only)                                         74  131
```

```
Resource type: aws:ecs/service:Service
  oss.txt: 5  cd.txt: 4

Only in oss.txt:
  release
    + capacityProviderStrategies: [...]
    + desiredCount: 1
    ...

Shared (0):
```
