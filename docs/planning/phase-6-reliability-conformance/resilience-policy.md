# Resilience Parity Policy (Phase 6)

## Critical service minimums

These minima apply to staging/prod unless explicitly risk-accepted:

- API: minimum 1 warm instance/replica
- Admin dashboard: minimum 1 instance/replica
- Core processing sidecars: minimum 1 replica where required for operational SLA
- DIMSE receiver: HA/availability strategy documented per cloud

## Policy goals

1. Avoid total cold-start dependency for critical user/admin paths.
2. Keep equivalent resilience intent across GCP, AWS, Azure.
3. Validate one controlled disruption scenario per cloud.

## Validation checklist

- [ ] IaC values reflect minimum policy
- [ ] Failover/disruption drill completed
- [ ] Evidence captured with timings and owner sign-off
