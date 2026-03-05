# AEGIS Implementation Plan (Ordered Checklist)

Date: 2026-02-26  
Source of truth: `hardening-plan.MD`

## How to Use This Plan

- Execute phases in order (Phase 0 → Phase 8).
- Do not start a later phase until its gate criteria are complete.
- For every task, record owner, date, evidence link, and pass/fail status.

---

## Phase 0 — Program Setup & Governance (Start Here)

### 0.1 Assign ownership
- [ ] Assign executive sponsor for beta hardening
- [ ] Assign technical lead per domain: Platform, Security, Compliance, Ops
- [ ] Assign cloud owners: GCP owner, AWS owner, Azure owner
- [ ] Assign incident commander and security officer (HIPAA admin safeguard)

### 0.2 Create tracking artifacts
- [ ] Create a single tracker (board/spreadsheet) with all tasks from this plan
- [ ] Add required fields: owner, due date, status, blocker, evidence URL
- [ ] Create an evidence folder structure for audit artifacts (per cloud + global)

### 0.3 Define environment policy
- [ ] Define which environments are in beta scope (e.g., `prod`, `staging`)
- [ ] Define immutable policy for non-dev auth: `AUTH_ENABLED=true`
- [ ] Define accepted `AUTH_PROVIDER` values per environment

**Gate to Phase 1:** owners assigned, tracker live, environment policy approved.

---

## Phase 1 — Service Deployment Parity (P0)

### 1.1 Build canonical service matrix
- [ ] Create canonical list of required services:
  - API, admin-dashboard, landing, dwv,
  - defacing, phi-detection, qc-service, bids-service,
  - classification-service, protocol-service, synth-service,
  - mcp-server, dimse-receiver
- [ ] Map each service to cloud runtime target (Cloud Run/ECS/ACA/VM)

### 1.2 Fix AWS deploy drift
- [ ] Update AWS deploy workflow to redeploy all required ECS services
- [ ] Ensure `mcp-server` is included in ECS deployment loop
- [ ] Ensure landing deployment logic matches provisioned infra state
- [ ] Add a check that fails CI if built images lack deployment targets

### 1.3 Validate parity deployment
- [ ] Run deployment in staging for each cloud
- [ ] Confirm image digest update for each service
- [ ] Confirm service health endpoints pass post-deploy
- [ ] Archive deployment logs as evidence

**Gate to Phase 2:** all required services deploy successfully across in-scope clouds with no drift.

---

## Phase 2 — Edge Security/WAF Parity (P0)

### 2.1 Baseline edge policy definition
- [ ] Define standard edge policy intent (allowlist/rate limits/managed rules/TLS)
- [ ] Define exceptions (e.g., DICOM upload paths) with justification

### 2.2 Azure edge hardening implementation
- [ ] Select Azure edge architecture (Application Gateway WAF v2 or Front Door WAF)
- [ ] Implement production-ready WAF ingress path for API/admin
- [ ] Validate TLS, routing, and backend health probes
- [ ] Document cutover plan and rollback plan

### 2.3 Cross-cloud policy alignment
- [ ] Compare GCP Cloud Armor, AWS WAF v2, Azure WAF rules against baseline intent
- [ ] Normalize major differences (rate limits, geo/IP policy, managed rule sets)
- [ ] Record accepted deviations and risk sign-off

**Gate to Phase 3:** equivalent edge protection intent enforced for all in-scope clouds.

---

## Phase 3 — Auth Guardrails in CI/CD (P0)

### 3.1 Implement pipeline enforcement
- [ ] Add deploy-time validation that blocks non-dev deploys when `AUTH_ENABLED=false`
- [ ] Add validation that `AUTH_PROVIDER` is set to allowed value for environment
- [ ] Add clear error output with remediation message

### 3.2 Add policy tests
- [ ] Add automated tests/lint checks for auth policy scripts
- [ ] Add negative test cases to verify deployment blocking behavior

### 3.3 Verify runtime auth posture
- [ ] Validate active runtime env vars for each cloud deployment
- [ ] Confirm admin access path works only through approved identity flow
- [ ] Capture screenshots/logs proving enforcement

**Gate to Phase 4:** unsafe auth configs are technically impossible to deploy in non-dev.

---

## Phase 4 — Monitoring & Alert Parity (P0)

### 4.1 Define critical alert catalog
- [ ] Required alerts: API availability/latency/error rate
- [ ] Required alerts: pipeline step failures (all sidecars)
- [ ] Required alerts: destination probe failures
- [ ] Required alerts: DIMSE retry queue/dead-letter thresholds

### 4.2 Implement parity alerts in each cloud
- [ ] Create/update alert rules in GCP
- [ ] Create/update alert rules in AWS
- [ ] Create/update alert rules in Azure
- [ ] Align severity labels and escalation routes

### 4.3 Validate on-call routing
- [ ] Execute synthetic failure tests for each alert family
- [ ] Confirm notifications reach on-call channel
- [ ] Confirm runbook links exist in alert payloads/descriptions

**Gate to Phase 5:** all critical alerts fire, route, and close correctly across clouds.

---

## Phase 5 — HIPAA Day-1 Controls (P0/P1 blend)

### 5.1 Administrative safeguards
- [ ] Document security officer + incident commander responsibilities
- [ ] Establish privileged access review cadence and ownership
- [ ] Archive security awareness/training records

### 5.2 Technical safeguards
- [ ] Verify TLS enforcement at edge and internal trust boundaries
- [ ] Verify encryption at rest for DB/object/log stores in each cloud
- [ ] Verify least-privilege IAM for runtime identities
- [ ] Verify MFA + conditional access for admin/operator accounts
- [ ] Verify centralized audit logging + retention controls

### 5.3 Infrastructure safeguards
- [ ] Validate network segmentation and ingress CIDR controls
- [ ] Validate DIMSE VM hardening baseline and patch level
- [ ] Validate cloud account boundary controls (org/subscription/project access)

### 5.4 Operations safeguards
- [ ] Run incident response tabletop + one live drill
- [ ] Run secret rotation procedure once pre-beta and log evidence
- [ ] Validate runbook completeness for primary incident classes

**Gate to Phase 6:** HIPAA day-1 checklist materially complete with owners and evidence.

---

## Phase 6 — Reliability, Recovery, and Conformance (P1)

### 6.1 Backup/restore validation
- [ ] Test DB restore for each cloud environment
- [ ] Test object storage restore for each cloud environment
- [ ] Record achieved RTO/RPO and compare with target
- [ ] Document failures and remediation actions

### 6.2 Cross-cloud conformance gate automation
- [ ] Build automated checker for:
  - service presence,
  - required env vars,
  - secret references,
  - health endpoints
- [ ] Run checker in CI as required pre-deploy gate
- [ ] Fail pipeline on parity violations

### 6.3 Resilience parity
- [ ] Define minimum replica/instance policy for critical services
- [ ] Apply policy in GCP, AWS, Azure IaC
- [ ] Validate failover behavior with one controlled disruption test

**Gate to Phase 7:** restore tests pass, conformance gate enforced, resilience policy applied.

---

## Phase 7 — Runbook & Operational Drift Reduction (P1)

### 7.1 Consolidate runbooks
- [ ] Produce unified parity runbook for common incidents
- [ ] Keep cloud-specific command appendices per provider
- [ ] Ensure all alert descriptions reference the unified runbook

### 7.2 Operational readiness review
- [ ] Conduct cross-functional review (Platform/Security/Ops)
- [ ] Resolve all P0/P1 open blockers or document risk acceptance
- [ ] Sign off beta go-live decision record

**Gate to Phase 8:** runbooks unified, go-live decision signed.

---

## Phase 8 — Post-Beta Hardening Program (P2)

### 8.1 Policy-as-code expansion
- [ ] Add Terraform security baseline policy checks in PR workflows
- [ ] Add mandatory review for policy violations

### 8.2 Continuous compliance dashboard
- [ ] Build dashboard for auth posture, secret age, alert health, backup freshness
- [ ] Add weekly automated export to evidence archive

### 8.3 Security testing cadence
- [ ] Schedule dependency/container scans
- [ ] Schedule periodic threat-model refresh and remediation planning
- [ ] Track findings to closure SLA

---

## Final Beta Exit Checklist (Must All Be True)

- [ ] No service deployment drift across selected clouds
- [ ] WAF/edge controls aligned by policy intent
- [ ] Non-dev auth misconfiguration blocked by CI/CD guardrails
- [ ] Critical monitoring/alerting validated end-to-end
- [ ] HIPAA day-1 safeguards implemented and evidenced
- [ ] Backup/restore and incident drills completed with acceptable outcomes
- [ ] Owners sign off on residual risk register

---

## Suggested Execution Order Summary

1. Phase 0: governance + policy setup  
2. Phase 1: service deployment parity  
3. Phase 2: edge/WAF parity  
4. Phase 3: auth guardrails  
5. Phase 4: monitoring/alerts parity  
6. Phase 5: HIPAA day-1 controls  
7. Phase 6: recovery + conformance automation  
8. Phase 7: runbook consolidation + readiness sign-off  
9. Phase 8: post-beta maturity improvements
