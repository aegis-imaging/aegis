# Phase 1 Service Deployment Matrix (Canonical)

Source plan: `implementation-plan.MD` (Phase 1)

This matrix defines the expected deployment target for each production service.

## Canonical Services

| Service | GCP Target | AWS Target | Azure Target | Notes |
|---|---|---|---|---|
| api | Cloud Run | ECS Fargate | Container Apps | Core API |
| admin-dashboard | Cloud Run | ECS Fargate | Container Apps | Internal UI |
| landing | Cloud Run | ECS Fargate (optional by tf var) | Container Apps | Public site |
| dwv | Cloud Run | ECS Fargate | Container Apps | Viewer |
| defacing | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| phi-detection | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| qc-service | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| bids-service | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| classification-service | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| protocol-service | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| synth-service | Cloud Run | ECS Fargate | Container Apps | Sidecar |
| mcp-server | Cloud Run | ECS Fargate | Container Apps | Agent integration |
| dimse-receiver | GCE VM | EC2 VM | Azure VM | TCP 11112 constraint |

## AWS Deployment Mapping Rules

- ECS deploy targets (must be covered by deploy loop when provisioned):
  - `aegis-api`
  - `aegis-admin-dashboard`
  - `aegis-dwv`
  - `aegis-defacing`
  - `aegis-phi-detection`
  - `aegis-qc-service`
  - `aegis-bids-service`
  - `aegis-classification-service`
  - `aegis-protocol-service`
  - `aegis-synth-service`
  - `aegis-mcp-server`
  - `aegis-landing` (optional; deploy only if ECS service exists)
- VM deploy target:
  - `dimse-receiver` via SSM parameter update + EC2 reboot path.

## Acceptance (Phase 1)

1. Every built image has a deployment target path.
2. AWS workflow redeploys `mcp-server`.
3. AWS workflow handles optional `landing` deployment safely when not provisioned.
4. CI contains a parity validation check to prevent regression.
