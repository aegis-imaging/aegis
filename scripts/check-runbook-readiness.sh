#!/usr/bin/env bash
set -euo pipefail

missing=0

check_file() {
  local file="$1"
  local label="$2"
  if [[ ! -f "$file" ]]; then
    echo "❌ Missing file: $label ($file)"
    missing=1
  fi
}

check_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"
  if ! grep -q "$pattern" "$file"; then
    echo "❌ Missing pattern: $label ($file :: $pattern)"
    missing=1
  fi
}

check_file "docs/runbooks/unified-parity-operations.md" "Unified parity runbook"
check_file "docs/runbooks/incident-response.md" "Incident response runbook"
check_file "docs/runbooks/alert-response.md" "Alert response runbook"
check_file "docs/runbooks/secret-rotation.md" "Secret rotation runbook"
check_file "docs/planning/phase-7-runbook-readiness/go-live-signoff-template.md" "Go-live sign-off template"
check_file "docs/planning/phase-7-runbook-readiness/risk-acceptance-register-template.md" "Risk acceptance template"

check_pattern "docs/runbooks/unified-parity-operations.md" 'incident-response.md' "Unified runbook references incident response"
check_pattern "docs/runbooks/unified-parity-operations.md" 'alert-response.md' "Unified runbook references alert response"
check_pattern "docs/runbooks/unified-parity-operations.md" 'secret-rotation.md' "Unified runbook references secret rotation"

if [[ "$missing" -ne 0 ]]; then
  echo "❌ Runbook readiness check failed"
  exit 1
fi

echo "✅ Runbook readiness check passed"
