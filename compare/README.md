# compare

Tools for comparing `pulumi preview --json` output across two stacks.

## Setup

Input files can be generated with `make`:

```
make cd.txt          # generate from defang estimate
make oss.txt         # generate from pulumi preview
make compare         # generate both (if missing) and run summary comparison
make compare-type type=aws:lb/listener:Listener  # property-level diff for a specific type
```

## compare.py

Compares resource counts and properties between two preview output files.

Input files may contain mixed plaintext and JSON (e.g. progress messages alongside resource steps). Both the old (`new`/`old`) and new (`newState`/`oldState`) Pulumi preview schemas are supported.

The following fields are automatically stripped from inputs before comparison: `__defaults__`, `tagsAll`, `tags`, `region`, `name`, `stackTrace`, `sourcePosition`, `propertyDependencies`. Unresolved output references (shown as UUIDs at preview time) are replaced with `<ref>`.

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

Resources are correlated across files by `defang:service` tag first, then by name-suffix matching as a fallback. Ambiguous matches are flagged with a warning.

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
Resource type: aws:ec2/securityGroup:SecurityGroup
  oss.txt: 4  cd.txt: 8

  postgres  <->  E-cd-estimate-postgres  (- oss.txt  + cd.txt)
    - description: "RDS security group for postgres"
    + description: "Managed by Defang"
    - ingress: [{"fromPort": 5432, "protocol": "tcp", "securityGroups": ["<ref>"], "toPort": 5432}]
    + ingress: [{"description": "postgres", ...}, {"description": "Allow ICMP Path MTU Discovery", ...}]

  alb-sg  (only in oss.txt)
    ...
```
