# Cross-Cloud DICOM Routing: GCP → AWS

Sends approved DICOM studies from the GCP AEGIS environment to the AWS AEGIS environment using DICOMweb STOW-RS over HTTPS. No code changes required — configuration only.

## Architecture

```
GCP Upload → GCP Pipeline → Approve → route_to rule fires
  → STOW-RS POST (multipart/related) → AWS ALB /api/stow
    → AWS API receives & stores → AWS routing rules → AWS pipeline
```

- **Protocol:** DICOMweb STOW-RS (`multipart/related; type="application/dicom"`)
- **Auth:** API key (`Authorization: Bearer aegis_...`)
- **Endpoint:** `POST /api/stow/studies` (public on AWS ALB, priority 85 — no Cognito)

## Prerequisites

- AWS AEGIS environment deployed and running (ECS services, RDS, ALB)
- GCP AEGIS environment deployed and running
- Both environments accessible over HTTPS
- Admin access to both environments (for API key creation and routing config)

## Setup Steps

### Step 1: Create an API key on AWS

This gives GCP a bearer token to authenticate STOW-RS calls.

```bash
curl -X POST https://<aws-api-domain>/api/api-keys \
  -H "Authorization: <aws-admin-auth-header>" \
  -H "Content-Type: application/json" \
  -d '{"name":"GCP cross-cloud forwarder"}'
```

Response:
```json
{"id":"<uuid>", "key":"aegis_<random>", "name":"GCP cross-cloud forwarder", ...}
```

**Save the `key` value** — it is shown only once and cannot be retrieved later.

### Step 2: Create a DICOMweb destination on GCP

Points GCP's routing engine at the AWS STOW-RS endpoint.

```bash
curl -X POST https://<gcp-api-domain>/api/destinations \
  -H "Authorization: <gcp-admin-auth-header>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AWS STOW-RS",
    "type": "dicomweb",
    "dicomweb_url": "https://<aws-api-domain>/api/stow/studies",
    "dicomweb_auth_header": "Bearer aegis_<key-from-step-1>",
    "enabled": true
  }'
```

Save the returned `id` for the next step.

**Alternative:** Use the admin dashboard Routing tab → Destinations → "Add Destination".

### Step 3: Create a routing rule on GCP

Tells GCP to forward studies to the AWS destination.

```bash
curl -X POST https://<gcp-api-domain>/api/routing-rules \
  -H "Authorization: <gcp-admin-auth-header>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Forward to AWS",
    "priority": 10,
    "action": "route_to",
    "destination_id": "<id-from-step-2>",
    "enabled": true
  }'
```

Optional filters (set to `null` for "any"):
- `project_id` — scope to a specific project
- `modality` — e.g. `"MRI"`, `"CT"`
- `body_part` — e.g. `"HEAD"`
- `source` — `"external"` or `"internal"`

**Alternative:** Use the admin dashboard Routing tab → Routing Rules → "Add Rule".

### Step 4: Test connectivity

Verify the destination is reachable before sending real data.

```bash
curl -X POST https://<gcp-api-domain>/api/destinations/<dest-id>/test \
  -H "Authorization: <gcp-admin-auth-header>"
```

Response:
```json
{"destination_id":"...", "type":"dicomweb", "success":true, "latency_ms":142}
```

**Alternative:** Click the "Test" button next to the destination in the admin dashboard.

## Running the Demo

1. Upload a DICOM study to GCP (via upload portal or DIMSE)
2. Let the GCP pipeline process the study (classification, defacing, QC, etc.)
3. Approve the study (manually or via `auto_approve` routing rule)
4. The `route_to` rule fires automatically on approval
5. Check the GCP study's routing log: `GET /api/studies/{id}/routing-log`
6. Verify the study arrived on AWS: check the AWS admin dashboard or `GET /api/studies`

## Monitoring

- **GCP routing log:** Per-study log shows each rule that fired and forwarding result
- **GCP audit trail:** `export.triggered`, `export.complete`, `export.failed` entries
- **AWS audit trail:** `stow.received` entry with source study UID
- **GCP export status:** Study's `export_status` field transitions `pending → exporting → exported`

## Troubleshooting

| Symptom | Check |
|---------|-------|
| `export_status=failed` | GCP API logs — look for STOW-RS HTTP error; verify AWS ALB is reachable |
| 401 Unauthorized | API key is invalid, disabled, or expired on AWS; re-create if needed |
| 403 Forbidden | AWS ALB Cognito is intercepting — verify `/api/stow` is in public path rules (priority 85) |
| Connection timeout | AWS ALB security group allows inbound 443; GCP egress is not blocked |
| Study received but no pipeline | AWS routing rules not configured; check `GET /api/routing-rules` on AWS |

## DIMSE Alternative (C-STORE)

For DIMSE-based forwarding (port 11112) instead of STOW-RS:

1. Deploy the AWS DIMSE receiver: set `dimse_receiver_image` in `terraform/aws/terraform.tfvars` and run `terraform apply`
2. Note the output `dimse_receiver_ip` (static Elastic IP)
3. Create a DIMSE destination on GCP:
   ```json
   {
     "name": "AWS DIMSE",
     "type": "dimse",
     "ae_title": "AEGIS",
     "host": "<dimse_receiver_ip>",
     "port": 11112,
     "enabled": true
   }
   ```
4. Create a `route_to` routing rule pointing to this destination
5. Test with `POST /api/destinations/{id}/test` (sends C-ECHO)

**Note:** DIMSE forwarding proxies through the GCP DIMSE receiver's `/forward` endpoint, so `DIMSE_RECEIVER_URL` must be set on the GCP API.

## Key Files

| File | Description |
|------|-------------|
| `api/handler/export_forward.go` | Export forwarding engine (STOW-RS + DIMSE) |
| `api/handler/stow_receiver.go` | Inbound STOW-RS endpoint with API key auth |
| `api/routing/engine.go` | Routing rules evaluation |
| `terraform/aws/main.tf` | ALB public path rules (line ~507) |
| `terraform/aws/dimse.tf` | AWS DIMSE receiver EC2 infrastructure |
