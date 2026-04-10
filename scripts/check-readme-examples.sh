#!/usr/bin/env bash
# Regenerate code blocks in README.md from example source files.
#
# The README uses <!-- source: path/to/file --> markers before fenced code
# blocks. This script reads each marker, slurps the referenced file, and
# replaces the code block that follows.
#
# Usage:
#   scripts/check-readme-examples.sh           # check only (exit 1 if stale)
#   scripts/check-readme-examples.sh --update  # rewrite README.md in place
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
README="$REPO_ROOT/README.md"
UPDATE=false
if [ "${1:-}" = "--update" ]; then
  UPDATE=true
fi

# Map file extensions to Markdown fence languages
fence_lang() {
  case "$1" in
    *.ts)    echo "typescript" ;;
    *.py)    echo "python" ;;
    *.go)    echo "go" ;;
    *.cs)    echo "csharp" ;;
    *.yaml)  echo "yaml" ;;
    *)       echo "" ;;
  esac
}

# Build the updated README into a temp file
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

source_file=""
in_code=false
skip_until_fence_close=false

while IFS= read -r line; do
  # Detect <!-- source: path --> markers
  if [[ "$line" =~ ^\<!--\ source:\ (.+)\ --\>$ ]]; then
    source_file="${BASH_REMATCH[1]}"
    printf '%s\n' "$line" >> "$tmp"
    continue
  fi

  # If we just saw a source marker, the next ``` opens the block to replace
  if [ -n "$source_file" ] && [[ "$line" =~ ^\`\`\` ]]; then
    local_path="$REPO_ROOT/$source_file"
    lang=$(fence_lang "$source_file")
    printf '```%s\n' "$lang" >> "$tmp"
    cat "$local_path" >> "$tmp"
    printf '```\n' >> "$tmp"
    source_file=""
    skip_until_fence_close=true
    continue
  fi

  # Skip old content inside the replaced block
  if $skip_until_fence_close; then
    if [[ "$line" =~ ^\`\`\`$ ]]; then
      skip_until_fence_close=false
    fi
    continue
  fi

  printf '%s\n' "$line" >> "$tmp"
done < "$README"

if diff -q "$README" "$tmp" > /dev/null 2>&1; then
  echo "README.md is up to date."
  exit 0
fi

if $UPDATE; then
  cp "$tmp" "$README"
  echo "README.md updated."
else
  echo "README.md is out of date. Diff:"
  diff -u --label "README.md (current)" --label "README.md (expected)" "$README" "$tmp" || true
  echo ""
  echo "Run 'scripts/check-readme-examples.sh --update' or 'make README.md' to fix."
  exit 1
fi
