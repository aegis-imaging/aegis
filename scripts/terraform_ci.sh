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
      then "[\(.["@level"])] \(.["@message"])\(.diagnostic.range | if . then " (\(.filename):\(.start.line))" else "" end)\n\(.diagnostic.detail // "")"
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

# The full human-readable plan (attribute values included) is what an
# approver needs, and it must never reach the public logs. When PLAN_ARCHIVE
# is set — a gs://, s3:// or https://<account>.blob.core.windows.net/<container>[/prefix]
# location, normally the private Terraform state bucket — it is written there
# as <run id>-plan.txt and the location is printed.
archive_plan() {
  [ -n "${PLAN_ARCHIVE:-}" ] || return 0
  local name="${GITHUB_RUN_ID:-local}-plan.txt" txt
  txt="$(mktemp)"
  terraform -chdir="$DIR" show -no-color tfplan.bin > "$txt"
  case "$PLAN_ARCHIVE" in
    gs://*)
      gcloud storage cp --quiet "$txt" "${PLAN_ARCHIVE%/}/$name" ;;
    s3://*)
      aws s3 cp --quiet "$txt" "${PLAN_ARCHIVE%/}/$name" ;;
    https://*.blob.core.windows.net/*)
      local rest account container prefix
      rest="${PLAN_ARCHIVE#https://}"
      account="${rest%%.blob.core.windows.net*}"
      rest="${rest#*.blob.core.windows.net/}"
      container="${rest%%/*}"
      prefix="${rest#"$container"}"; prefix="${prefix#/}"
      az storage blob upload --only-show-errors --auth-mode key --overwrite \
        --account-name "$account" --container-name "$container" \
        --name "${prefix:+$prefix/}$name" --file "$txt" >/dev/null ;;
    *)
      echo "[warn] PLAN_ARCHIVE has an unsupported scheme: $PLAN_ARCHIVE"; rm -f "$txt"; return 0 ;;
  esac
  rm -f "$txt"
  echo "[info] Full plan with attribute values archived to ${PLAN_ARCHIVE%/}/$name"
  [ -z "${GITHUB_STEP_SUMMARY:-}" ] || echo "Full plan: \`${PLAN_ARCHIVE%/}/$name\`" >> "$GITHUB_STEP_SUMMARY"
}

LOG="$(mktemp)"
trap 'rm -f "$LOG" "$DIR/tfplan.bin"' EXIT

case "$MODE" in
  plan)
    terraform -chdir="$DIR" plan -input=false -lock-timeout=5m -out=tfplan.bin -json | filter | tee "$LOG"
    archive_plan
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
