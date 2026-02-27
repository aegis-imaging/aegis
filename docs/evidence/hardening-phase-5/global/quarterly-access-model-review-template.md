# Quarterly Access Model Review Record

Review date (UTC):
Quarter covered:
Primary owner:
Security reviewer:
Compliance reviewer:

## Scope

- Access matrix version/file:
- Test catalog version/file:
- Environments checked:

## 1) Role Matrix Validation

- [ ] Route coverage guard executed (`scripts/check-access-matrix-coverage.sh`)
- [ ] Authenticated route inventory reviewed against matrix
- [ ] Role capability assumptions validated

Notes:

## 2) Cloud-Auth Parity Validation

- [ ] GCP IAP parity validated
- [ ] Azure Easy Auth parity validated
- [ ] AWS ALB/Cognito parity validated

Notes:

## 3) Access Log Sampling

- [ ] Sample size >= 10 scoped requests
- [ ] Expected denials observed for out-of-scope attempts
- [ ] No cross-project leakage observed

Evidence references (queries, screenshots, exports):

## Findings and Actions

| Finding | Severity | Owner | Due date | Tracking link |
|---|---|---|---|---|
| _none_ |  |  |  |  |

## Sign-off

- Platform Lead:
- Security Lead:
- Compliance/Privacy Lead:
