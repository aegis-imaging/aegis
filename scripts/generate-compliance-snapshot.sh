#!/usr/bin/env bash
set -euo pipefail

out_file="docs/monitoring/compliance-snapshot.md"
mkdir -p "$(dirname "$out_file")"

run_check() {
  local script="$1"
  if "./$script" >/tmp/check.out 2>/tmp/check.err; then
    echo "- ✅ $script"
  else
    echo "- ❌ $script"
    if [[ -s /tmp/check.err ]]; then
      echo "  - stderr: $(tr '\n' ' ' </tmp/check.err)"
    fi
  fi
}

{
  echo "# Compliance Snapshot"
  echo
  echo "Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
  echo
  echo "## Baseline Checks"
  run_check "scripts/check-aws-deploy-parity.sh"
  run_check "scripts/check-auth-guardrails.sh"
  run_check "scripts/check-edge-security-parity.sh"
  run_check "scripts/check-monitoring-alert-parity.sh"
  run_check "scripts/check-hipaa-day1-controls.sh"
  run_check "scripts/check-cross-cloud-conformance.sh"
  run_check "scripts/check-runbook-readiness.sh"
  run_check "scripts/check-terraform-security-baseline.sh"
} >"$out_file"

echo "Wrote $out_file"
