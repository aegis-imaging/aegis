# HIPAA Day-1 Control Checklist

## Tracking fields

- Owner:
- Due date:
- Status: `not_started|in_progress|blocked|completed`
- Evidence path:
- Notes:

## 5.1 Administrative safeguards

- [ ] Security Officer and Incident Commander responsibilities documented
- [ ] Privileged access review cadence defined and owner assigned
- [ ] Security awareness/training evidence archived

## 5.2 Technical safeguards

- [ ] TLS enforcement verified at ingress and internal trust boundaries
- [ ] Encryption-at-rest verified for DB/object/log stores (all in-scope clouds)
- [ ] Least-privilege IAM verified for runtime identities
- [ ] MFA + conditional access verified for admin/operator accounts
- [ ] Centralized audit logging + retention controls verified

## 5.3 Infrastructure safeguards

- [ ] Network segmentation and ingress CIDR controls validated
- [ ] DIMSE VM baseline hardening + patch status validated
- [ ] Cloud account boundary and access model validated

## 5.4 Operations safeguards

- [ ] Incident response tabletop completed and recorded
- [ ] One live drill completed and recorded
- [ ] Secret rotation runbook executed and evidence captured
- [ ] Runbook completeness validated for primary incident classes

## Required approvals

- [ ] Security Lead sign-off
- [ ] Compliance Lead sign-off
- [ ] Platform Lead sign-off
