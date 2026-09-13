#!/usr/bin/env bash
set -euo pipefail

WORKFLOW_NAME="Terraform Azure"
REF="main"
CONFIRM_VALUE="APPLY"
WATCH_RUN=1

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/trigger_terraform_azure_apply.sh [--ref=main] [--confirm=APPLY] [--no-watch]

Examples:
  ./scripts/trigger_terraform_azure_apply.sh
  ./scripts/trigger_terraform_azure_apply.sh --ref=main --confirm=APPLY
  ./scripts/trigger_terraform_azure_apply.sh --no-watch

Notes:
  - Triggers the GitHub Actions workflow "Terraform Azure" via workflow_dispatch.
  - By default, watches the newest workflow run on the selected branch.
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --ref=*) REF="${arg#*=}" ;;
    --confirm=*) CONFIRM_VALUE="${arg#*=}" ;;
    --no-watch) WATCH_RUN=0 ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg'" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh CLI is not installed or not in PATH" >&2
  exit 1
fi

echo "==> Triggering workflow '$WORKFLOW_NAME' on ref '$REF' with confirm_apply='$CONFIRM_VALUE'"
gh workflow run "$WORKFLOW_NAME" --ref "$REF" -f confirm_apply="$CONFIRM_VALUE"

sleep 3
RUN_ID="$(gh run list --workflow "$WORKFLOW_NAME" --branch "$REF" --limit 1 --json databaseId --jq '.[0].databaseId')"

if [ -z "$RUN_ID" ] || [ "$RUN_ID" = "null" ]; then
  echo "warning: workflow dispatched but run id could not be resolved yet"
  echo "check manually: gh run list --workflow '$WORKFLOW_NAME' --branch '$REF' --limit 5"
  exit 0
fi

echo "==> Run queued: $RUN_ID"
echo "==> URL: $(gh run view "$RUN_ID" --json url --jq '.url')"

if [ "$WATCH_RUN" -eq 1 ]; then
  echo "==> Watching run until completion"
  gh run watch "$RUN_ID"
fi
