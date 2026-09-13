# Cross-Cloud DICOM Routing

Sends approved DICOM studies between AEGIS cloud environments (GCP, AWS, Azure) using DICOMweb STOW-RS over HTTPS or DIMSE C-STORE over TCP. No code changes required — configuration only.

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

## DIMSE Forwarding (C-STORE)

All three DIMSE C-STORE receivers are operational. Cross-cloud forwarding can use DIMSE as an alternative to STOW-RS.

### DIMSE Receiver Endpoints

| Cloud | IP Address | Port | VM Type | Terraform File |
|-------|-----------|------|---------|----------------|
| GCP   | `<GCP_DIMSE_PUBLIC_IP>` | 11112 | Compute Engine (Debian 12) | `terraform/infra/dimse.tf` |
| AWS   | *(Elastic IP — run `terraform output dimse_receiver_ip` in `terraform/aws/`)* | 11112 | EC2 (Amazon Linux 2023) | `terraform/aws/dimse.tf` |
| Azure | `<AZURE_DIMSE_PUBLIC_IP>` | 11112 | Azure Linux VM (Debian 12) | `terraform/azure/dimse.tf` |

### GCP → AWS (DIMSE C-STORE)

1. Note the AWS DIMSE IP: `cd terraform/aws && terraform output dimse_receiver_ip`
2. Create a DIMSE destination on GCP:
   ```json
   {
     "name": "AWS DIMSE",
     "type": "dimse",
     "ae_title": "AEGIS",
     "host": "<aws_dimse_ip>",
     "port": 11112,
     "enabled": true
   }
   ```
3. Create a `route_to` routing rule pointing to this destination
4. Test with `POST /api/destinations/{id}/test` (sends C-ECHO)

### GCP → Azure (DIMSE C-STORE)

1. Create a DIMSE destination on GCP:
   ```json
   {
     "name": "Azure DIMSE",
     "type": "dimse",
     "ae_title": "AEGIS",
     "host": "<AZURE_DIMSE_PUBLIC_IP>",
     "port": 11112,
     "enabled": true
   }
   ```
2. Create a `route_to` routing rule on GCP pointing to this destination
3. Test with `POST /api/destinations/{id}/test`

### Azure → GCP (DIMSE C-STORE)

1. Create a DIMSE destination on Azure:
   ```json
   {
     "name": "GCP DIMSE",
     "type": "dimse",
     "ae_title": "AEGIS",
     "host": "<GCP_DIMSE_PUBLIC_IP>",
     "port": 11112,
     "enabled": true
   }
   ```
2. Create a `route_to` routing rule on Azure pointing to this destination
3. Test with `POST /api/destinations/{id}/test`

### Other Pairs (AWS → Azure, AWS → GCP, Azure → AWS)

Follow the same pattern — create a DIMSE destination on the **sending** cloud with the **receiving** cloud's DIMSE IP and port 11112, then create a `route_to` routing rule.

### Firewall Requirements

Each cloud restricts inbound DIMSE traffic on TCP 11112 via the `dimse_source_ranges` Terraform variable. To enable cross-cloud DIMSE forwarding, add the source cloud's DIMSE VM egress IP (or NAT IP) to the destination cloud's `dimse_source_ranges`:

| Cloud | Variable location | Default |
|-------|------------------|---------|
| GCP | `terraform/infra/terraform.tfvars` | `["0.0.0.0/0"]` |
| AWS | `terraform/aws/terraform.tfvars` | `["203.0.113.0/24"]` (placeholder) |
| Azure | `terraform/azure/terraform.tfvars` | `["0.0.0.0/0"]` |

**Note:** DIMSE forwarding proxies through the sending cloud's DIMSE receiver's `/forward` endpoint, so `DIMSE_RECEIVER_URL` must be set on the sending cloud's API.

## Key Files

| File | Description |
|------|-------------|
| `api/handler/export_forward.go` | Export forwarding engine (STOW-RS + DIMSE) |
| `api/handler/stow_receiver.go` | Inbound STOW-RS endpoint with API key auth |
| `api/routing/engine.go` | Routing rules evaluation |
| `terraform/aws/main.tf` | ALB public path rules (line ~507) |
| `terraform/aws/dimse.tf` | AWS DIMSE receiver EC2 infrastructure |

---

## Azure Cross-Cloud Routing

Azure Container Apps run the same Go API with `AUTH_PROVIDER=azure`. The STOW-RS endpoint (`/api/stow/studies`) accepts API key authentication at the handler level — no Azure AD required for machine-to-machine routing.

### Azure → GCP

1. Create an API key on the **GCP** admin dashboard (Settings → API Keys) — label it `azure-stow-sender`
2. Note the raw key: `aegis_<...>`
3. On the **Azure** admin dashboard, create a DICOMweb destination:
   ```json
   {
     "name": "GCP STOW-RS",
     "type": "dicomweb",
     "dicomweb_url": "https://api.aegisimaging.ai/api/stow",
     "dicomweb_auth_header": "Bearer aegis_<key-from-step-2>",
     "enabled": true
   }
   ```
4. Create a `route_to` routing rule on Azure pointing to this destination
5. Test: `POST /api/destinations/{id}/test` on Azure → should get non-4xx

### GCP → Azure

1. Create an API key on the **Azure** admin dashboard — label it `gcp-stow-sender`
2. On the **GCP** admin dashboard, create a DICOMweb destination:
   ```json
   {
     "name": "Azure STOW-RS",
     "type": "dicomweb",
     "dicomweb_url": "https://<azure-api-fqdn>/api/stow",
     "dicomweb_auth_header": "Bearer aegis_<azure-api-key>",
     "enabled": true
   }
   ```
3. Create a `route_to` routing rule on GCP pointing to this destination

### AWS → Azure (and vice versa)

Follow the same pattern — create API keys on the receiving cloud, create DICOMweb destinations on the sending cloud pointing to `/api/stow` with `Bearer <api-key>` auth header.

The public path bypass rule must include `/api/stow` on the receiving cloud's load balancer:
- **GCP**: `/api/stow` already in Cloud Armor bypass list
- **AWS**: `stow` entry in `public_path_rules` in `terraform/aws/main.tf` (priority 85)
- **Azure**: Application Gateway WAF — add a WAF exclusion or custom rule to allow `/api/stow` without AAD auth

### Key Files (Azure)

| File | Description |
|------|-------------|
| `terraform/azure/main.tf` | Core Azure infrastructure |
| `terraform/azure/container_apps.tf` | Container App definitions |
| `.github/workflows/deploy-azure.yml` | Azure auto-deploy CI/CD |
| `api/storage/azure.go` | Azure Blob Storage backend |
