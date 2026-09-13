#!/usr/bin/env bash
set -euo pipefail

workflow_file=".github/workflows/deploy-aws.yml"

if [[ ! -f "$workflow_file" ]]; then
  echo "❌ Missing $workflow_file"
  exit 1
fi

required_build_services=(
  "service: api"
  "service: admin-dashboard"
  "service: dwv"
  "service: defacing"
  "service: phi-detection"
  "service: qc-service"
  "service: bids-service"
  "service: classification-service"
  "service: protocol-service"
  "service: synth-service"
  "service: mcp-server"
  "service: dimse-receiver"
  "service: landing"
)

required_ecs_targets=(
  "aegis-api"
  "aegis-admin-dashboard"
  "aegis-dwv"
  "aegis-defacing"
  "aegis-phi-detection"
  "aegis-qc-service"
  "aegis-bids-service"
  "aegis-classification-service"
  "aegis-protocol-service"
  "aegis-synth-service"
  "aegis-mcp-server"
)

missing=0

for pattern in "${required_build_services[@]}"; do
  if ! grep -q "$pattern" "$workflow_file"; then
    echo "❌ Missing build service entry: $pattern"
    missing=1
  fi
done

for svc in "${required_ecs_targets[@]}"; do
  if ! grep -q "$svc" "$workflow_file"; then
    echo "❌ Missing ECS deploy target: $svc"
    missing=1
  fi
done

if ! grep -q 'deploy_if_exists "aegis-landing"' "$workflow_file"; then
  echo "❌ Missing conditional landing deploy target handling"
  missing=1
fi

if ! grep -q '/aegis/dimse-image' "$workflow_file"; then
  echo "❌ Missing DIMSE deployment target mapping"
  missing=1
fi

if [[ "$missing" -ne 0 ]]; then
  echo "❌ AWS deploy parity check failed"
  exit 1
fi

echo "✅ AWS deploy parity check passed"
