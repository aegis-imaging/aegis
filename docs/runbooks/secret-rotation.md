# AEGIS Secret Rotation Runbook

Covers DB credential rotation for GCP (Cloud SQL + Secret Manager) and AWS (RDS managed password).
Run this procedure when rotating credentials on a schedule or in response to a suspected exposure.

---

## GCP — Cloud SQL Password Rotation

### How credentials flow

```
terraform/infra apply
  └─ random_password.db_password  →  google_secret_manager_secret_version.db_password
       └─ google_sql_user.api.password (Cloud SQL user set on apply)
            └─ Cloud Run env: DB_PASSWORD (injected via secret_key_ref, not inline)
```

Cloud Run reads `DB_PASSWORD` from Secret Manager at **container startup** using the `secret_key_ref`
block — not as an inline env var. No restart is needed on secret version update unless `latest` is
pinned (which it is by default in AEGIS).

### Rotate with Terraform (recommended)

This is the safest method — Terraform regenerates the password, updates Cloud SQL, and
updates the Secret Manager version atomically.

```bash
cd terraform/infra

# Force regeneration of the random password
terraform apply -replace=random_password.db_password
```

Terraform will:
1. Generate a new 32-character random password
2. Update the `aegis-<env>-db-password` Secret Manager secret with a new version
3. Update the Cloud SQL user password

After apply, new Cloud Run instances will pick up the new secret on next startup.
Force a redeploy to ensure no instances are using the old credential:

```bash
gcloud run services update aegis-<env>-api \
  --region us-central1 \
  --project <project_id>
```

### Rotate manually (emergency)

Use this if Terraform state is unavailable or rotation must happen immediately.

```bash
export PROJECT_ID="<GCP_PROJECT_ID>"
export ENV="prod"
export SECRET_ID="aegis-${ENV}-db-password"
export INSTANCE="aegis-${ENV}-postgres"

# 1. Generate a new strong password (32+ chars, mixed)
NEW_PASSWORD=$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9_%@' | head -c 32)

# 2. Update Cloud SQL user password
gcloud sql users set-password aegis-api \
  --instance="${INSTANCE}" \
  --password="${NEW_PASSWORD}" \
  --project="${PROJECT_ID}"

# 3. Add new Secret Manager version
printf '%s' "${NEW_PASSWORD}" | \
  gcloud secrets versions add "${SECRET_ID}" \
    --data-file=- \
    --project="${PROJECT_ID}"

# 4. Force Cloud Run redeploy to pick up new secret version
gcloud run services update aegis-${ENV}-api \
  --region us-central1 \
  --project "${PROJECT_ID}"
```

### Verify

```bash
# Confirm latest secret version is active
gcloud secrets versions list "${SECRET_ID}" --project="${PROJECT_ID}"

# Confirm API is healthy after redeploy
curl -sf https://api.aegisimaging.ai/healthz | python3 -m json.tool
# Expected: {"status":"ok","database":"healthy",...}
```

---

## AWS — RDS Managed Master Password Rotation

### How credentials flow

```
aws_db_instance.main (manage_master_user_password = true)
  └─ AWS Secrets Manager: auto-created secret (ARN in master_user_secret[0].secret_arn)
       └─ ECS task secrets: DB_PASSWORD (valueFrom = secret_arn:password::)
```

AWS RDS with `manage_master_user_password = true` automatically creates and rotates the master
credential in Secrets Manager. No Terraform-managed password variable is used.

### Trigger immediate rotation (manual)

```bash
# Rotate now (asynchronous — takes ~30s)
aws rds modify-db-instance \
  --db-instance-identifier aegis-postgres \
  --rotate-master-user-password \
  --apply-immediately \
  --region us-east-1

# Wait for rotation to complete
aws rds wait db-instance-available \
  --db-instance-identifier aegis-postgres \
  --region us-east-1
```

ECS tasks will automatically pick up the new credential on the next task launch because the
`valueFrom` reference in the task definition points at the Secrets Manager secret ARN, not a
pinned version.

### Force ECS service redeploy

```bash
aws ecs update-service \
  --cluster aegis \
  --service aegis-api \
  --force-new-deployment \
  --region us-east-1
```

### Verify

```bash
# Confirm secret exists and is accessible
aws secretsmanager describe-secret \
  --secret-id $(aws rds describe-db-instances \
    --db-instance-identifier aegis-postgres \
    --query 'DBInstances[0].MasterUserSecret.SecretArn' \
    --output text) \
  --region us-east-1

# Confirm API healthy
curl -sf https://<your-aws-api-domain>/healthz | python3 -m json.tool
```

---

## Post-Rotation Checklist

- [ ] `GET /healthz` returns `"database":"healthy"` after redeploy
- [ ] Admin dashboard login works end-to-end
- [ ] At least one study list API call succeeds
- [ ] Old Secret Manager version disabled (GCP) or previous rotation confirmed (AWS)
- [ ] Rotation event recorded in ops log with date and initiator

---

## CI Guard

The `scripts/check-infra-placeholders.sh` script (run in CI via `infra-guard` job) blocks:
- Known-weak plaintext passwords in `.tf` files
- Any non-example `.tfvars` files tracked in git

If a PR triggers this guard, fix the issue in the Terraform source and re-push.
