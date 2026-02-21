#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  cat <<'USAGE' >&2
Usage: ./scripts/validate_tfvars_required.sh <tfvars-path> <required_key_1> [required_key_2] ...

Validates that required keys exist and have non-empty values in a tfvars file.
USAGE
  exit 1
fi

TFVARS_PATH="$1"
shift

if [ ! -f "$TFVARS_PATH" ]; then
  echo "error: tfvars file not found: $TFVARS_PATH" >&2
  exit 1
fi

missing=()

for key in "$@"; do
  # Allow optional whitespace and quoted/unquoted values.
  if ! rg -q --pcre2 "^\s*${key}\s*=\s*(\"[^\"]+\"|[^#\s][^#]*)\s*(#.*)?$" "$TFVARS_PATH"; then
    missing+=("$key")
  fi
done

if [ "${#missing[@]}" -gt 0 ]; then
  echo "error: missing required tfvars values in $TFVARS_PATH:" >&2
  for key in "${missing[@]}"; do
    echo "  - $key" >&2
  done
  exit 1
fi

echo "ok: required tfvars keys present in $TFVARS_PATH"
