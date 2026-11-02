#!/bin/bash
# Prints the release notes for a version: its CHANGELOG.md section, then the install block.
#   scripts/release-notes.sh 1.2.0
set -euo pipefail
cd "$(dirname "$0")/.."
v="${1#v}"

section="$(awk -v v="$v" '
  /^## / { inside = ($2 == v) ; next }
  inside { print }
' CHANGELOG.md)"
if [ -z "$(echo "$section" | tr -d '[:space:]')" ]; then
  echo "CHANGELOG.md has no section for $v" >&2
  exit 1
fi

printf '%s\n\n---\n\n' "$section"
cat docs/RELEASE_NOTES.md
