#!/usr/bin/env bash
# terraform_ci.sh plan|apply <module-dir>
#
# Runs terraform in machine-readable mode and prints resource-level messages
# only. Workflow logs on a public repository are world-readable, so attribute
# values (which can include credentials and account identifiers) must never
# reach them, and the binary plan must never be uploaded as an artifact.
set -euo pipefail

MODE="${1:-}"
DIR="${2:-}"
if [ -z "$MODE" ] || [ -z "$DIR" ]; then
  echo "usage: $0 plan|apply <module-dir>" >&2
  exit 2
fi
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

# Keep: what will change / changed, drift, and errors. Drop: everything that
# carries attribute values (resource diffs, outputs).
filter() {
  jq -r '
    select(.type == "planned_change" or .type == "change_summary" or .type == "resource_drift"
        or .type == "apply_start" or .type == "apply_progress" or .type == "apply_complete"
        or .type == "apply_errored" or .type == "diagnostic")
    | if .type == "diagnostic"
      then "[\(.["@level"])] \(.["@message"])\n\(.diagnostic.detail // "")"
      else "[\(.["@level"])] \(.["@message"])"
      end'
}

summary() {
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
      echo "### Terraform $MODE — \`$DIR\`"
      echo
      echo '```'
      grep -E '^\[(info|error)\] (Plan:|Apply complete|Error)' "$1" || echo "(no change summary)"
      echo '```'
    } >> "$GITHUB_STEP_SUMMARY"
  fi
}

LOG="$(mktemp)"
trap 'rm -f "$LOG"' EXIT

case "$MODE" in
  plan)
    terraform -chdir="$DIR" plan -input=false -lock-timeout=5m -json | filter | tee "$LOG"
    ;;
  apply)
    terraform -chdir="$DIR" apply -input=false -auto-approve -lock-timeout=5m -json | filter | tee "$LOG"
    ;;
  *)
    echo "unknown mode: $MODE" >&2
    exit 2
    ;;
esac
summary "$LOG"
