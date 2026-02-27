# AEGIS Hardening Plan (Multi-Cloud Parity + Security + HIPAA Day-1)

Date: 2026-02-26  
Scope: GCP, AWS, Azure deployments for Anonymization & Exchange Gateway for Imaging Studies (AEGIS)

## Executive Summary

AEGIS is close to functional parity across GCP, AWS, and Azure for core workloads (API, sidecars, synthetic generation, DIMSE ingress), but there are launch-critical drift and hardening gaps that should be addressed before beta.

Primary risks are:

1. **Deployment parity drift** (especially AWS workflow coverage)
2. **Edge protection inconsistency** (Azure WAF path not fully active by default)
3. **Production auth guardrail gaps** (risk of unsafe runtime auth config)
4. **Alert/operations parity gaps** (failure detection and response consistency)

---

## Findings (Confirmed)

### 1) Service Deployment Parity

- Core service set exists across clouds: API, admin dashboard, landing, weasis, defacing, phi-detection, qc-service, bids-service, classification-service, protocol-service, synth-service, mcp-server, dimse-receiver.
- **Confirmed AWS drift:** deployment workflow updates only 10 ECS services and omits `mcp-server` (and does not actively deploy landing ECS service in the current loop), while images are built and infra references exist.
- GCP Cloud Build and Azure deploy flow appear broader in service rollout coverage than AWS deploy loop.

### 2) DIMSE Parity

- DIMSE is intentionally VM/instance-based (not serverless) across clouds due to TCP 11112 requirements.
- Mechanisms differ by cloud (GCE metadata+reset, EC2 update/reboot path, Azure VM flow), which is acceptable functionally but increases operational drift risk.

### 3) MCP / Agent Service Parity

- `mcp-server` is represented in infra and image pipelines.
- **Risk:** AWS deployment path likely leaves MCP stale if ECS service is not included in forced redeploy loop.

### 4) Edge Security / WAF Parity

- GCP: Cloud Armor policy attached to API backend.
- AWS: WAF v2 present with rate limiting and optional allowlists.
- Azure: comments indicate Application Gateway/WAF layering can be added later; current setup may rely on direct Container Apps ingress.
- **Risk:** uneven edge defense posture across clouds.

### 5) Auth / Access Controls

- API middleware supports provider-based auth and role checks (`AUTH_ENABLED`, `AUTH_PROVIDER`, provider headers + API key fallback).
- Project-scoped/institution-scoped access work has been implemented and merged for backend read paths and visibility controls.
- **Risk:** missing CI/CD guardrails could permit unsafe auth runtime configuration in non-dev environments.

### 6) Ops / Monitoring Parity

- Monitoring resources and alerts exist in each cloud stack, but exact policy parity and severity mapping may differ.
- DIMSE retry/dead-letter telemetry and health signals are present in app services, but alerting and escalation consistency should be validated across all cloud environments.

### 7) HIPAA Day-1 Readiness

Foundational capabilities are present (de-identification pipeline, audit trail mechanisms, cloud-native secrets/KMS integrations), but day-1 readiness depends on proving controls are uniformly enforced and operationalized:

- least-privilege IAM
- encryption in transit and at rest
- secret rotation execution
- immutable audit evidence
- incident response playbooks with ownership and SLAs
- backup/restore drills with evidence

---

## Priority Remediation Plan

## P0 — Must Complete Before Beta

1. **Fix AWS deploy parity drift**
   - Include all production services in deployment loop (notably `mcp-server`, and landing where provisioned).
   - Add validation that every built image has a matching deployment target.

2. **Establish Azure WAF-equivalent edge path for public ingress**
   - Activate Application Gateway/Front Door + WAF architecture for production.
   - Ensure API/admin exposure mirrors GCP/AWS security intent.

3. **Enforce auth safety in CI/CD**
   - Block non-dev deploys when `AUTH_ENABLED=false`.
   - Require explicit valid provider configuration (`iap|aws|azure|auto` policy as defined by environment standard).

4. **Normalize critical alert coverage**
   - Ensure equivalent alerts for: pipeline step failures, destination probe failures, DIMSE queue/dead-letter growth, API availability and latency.

## P1 — Strongly Recommended Before GA

1. **Cross-cloud conformance gate**
   - Automated check comparing intended service matrix vs deployed services, env vars, secrets, and health endpoints.

2. **Resilience parity**
   - Standardize minimum replicas/instances for critical services and dependency chains.

3. **Backup/restore evidence**
   - Perform and document restore tests for DB + object storage in all clouds with explicit RTO/RPO acceptance.

4. **Operational drift reduction**
   - Consolidate cloud-specific runbooks for DIMSE and sidecar incident handling into one parity matrix with cloud-specific command appendices.

## P2 — Post-Beta Hardening

1. Policy-as-code checks for security baselines in Terraform PRs.
2. Continuous compliance dashboard (auth posture, secret age, alert health, backup freshness).
3. Extended security testing cadence (dependency scanning, container image runtime controls, periodic threat-model refresh).

---

## HIPAA Day-1 Control Checklist

## Administrative Safeguards

- [ ] Assigned security officer and incident commander roles documented
- [ ] Access review cadence defined (at least monthly for privileged roles)
- [ ] Workforce security training evidence archived

## Technical Safeguards

- [ ] TLS enforced for all external and service-to-service traffic where applicable
- [ ] Encryption at rest enabled for DB/object/log stores
- [ ] Least-privilege IAM verified for runtime service identities
- [ ] MFA + conditional access for admin/operator accounts
- [ ] Centralized audit logging with tamper-evident retention controls
- [ ] Auth runtime guardrails enforced in CI/CD

## Physical/Infrastructure Safeguards (Cloud-Scoped)

- [ ] Cloud account/subscription/project access boundaries reviewed
- [ ] Network segmentation + ingress allowlists validated
- [ ] VM hardening baseline applied for DIMSE hosts

## Operations & Recovery

- [ ] Incident response runbook tested (tabletop + live drill)
- [ ] Backup/restore drill completed per cloud
- [ ] Monitoring + alert on-call routing tested end-to-end
- [ ] Secret rotation runbook executed once pre-beta and logged

## Evidence Pack (Required Before Go-Live)

- [ ] Architecture + data-flow diagram set (current)
- [ ] IAM role/policy exports and review sign-off
- [ ] Key/cert/secrets inventory with rotation dates
- [ ] Alert policy export and test evidence
- [ ] Backup restore logs with timings
- [ ] Audit trail sample proving actor, action, timestamp, resource traceability

---

## Recommended Immediate Next Actions

1. Patch AWS deploy workflow service loop for parity (MCP + landing as applicable).
2. Add CI check that fails production deploys on unsafe auth settings.
3. Finalize Azure WAF ingress architecture decision and implementation path.
4. Generate a cloud-by-cloud control verification worksheet and run it in staging.

---

## Acceptance Criteria for Beta Security Readiness

AEGIS is considered beta-ready when:

1. All required services deploy consistently across selected clouds with no workflow drift.
2. Edge protections (WAF/rate limiting/allowlisting strategy) are equivalent by policy intent.
3. Production auth cannot be deployed in insecure mode.
4. Critical operational alerts are active, routed, and tested.
5. Backup/restore and incident response drills are completed with evidence.
6. HIPAA safeguard checklist items are materially complete with documented owners.
