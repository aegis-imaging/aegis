# Secret Rotation Execution Record

Rotation date:
Executor:
Scope: `db-password|api-keys|smtp|dimse-operator|other`

## Pre-checks

- [ ] Current secret inventory captured
- [ ] Rollback plan prepared
- [ ] Maintenance window approved (if required)

## Execution log

| Step | Result | Evidence |
|---|---|---|
| Generate new secret | pass/fail | _link_ |
| Update secret manager | pass/fail | _link_ |
| Restart/redeploy dependent services | pass/fail | _link_ |
| Validate health checks | pass/fail | _link_ |
| Revoke old secret | pass/fail | _link_ |

## Post-checks

- [ ] No authentication failures observed
- [ ] Audit entries verified
- [ ] Stakeholders notified

## Approvals

- Security Lead:
- Platform Lead:
