# Minimal Footprint — keeping AWS and Azure answering cheaply

Goal: every cloud keeps answering `https://<cloud>.api.aegisimaging.ai/healthz`
with HTTP 200 and serves its admin sign-in page, at the lowest steady monthly
cost. Only the API, admin dashboard, PostgreSQL and object storage run; the
Python sidecars, DWV viewer, MCP server, upload portal and DIMSE receivers are
switched off with terraform flags. Flip the same flags back for a full deploy.

GCP needs no change: every Cloud Run service already scales to zero and Cloud
SQL is a `db-f1-micro`.

## Cost (rough list prices, per month)

| | AWS (us-east-1) | Azure (eastus2) |
|---|---|---|
| NAT gateway | ~$33 | ~$33 |
| Load balancer / ingress | ALB ~$17 | included in Container Apps |
| PostgreSQL | db.t3.micro + 20 GB ~$15 | B_Standard_B1ms + 32 GB ~$17 |
| API | Fargate 0.5 vCPU / 1 GB ~$18 | 0.5 vCPU / 1 Gi, 1 replica ~$34 |
| Admin dashboard | Fargate 0.25 vCPU / 0.5 GB ~$9 | 0.25 vCPU / 0.5 Gi, 1 replica ~$20 |
| WAF, KMS, logs, registry, Key Vault, alerts | ~$15 | ~$12 |
| **Total** | **~$105–120** | **~$115–125** |

Biggest levers if that is still too much: the two NAT gateways (~$66 of it),
and `api_min_replicas = 0` on Azure (saves ~$34, adds a few seconds of cold
start to the first request after idle).

## Step 0 — find out what is there right now

`scripts/aws_bootstrap_ci.sh` (AWS CloudShell) and
`scripts/azure_bootstrap_ci.sh` (Azure Cloud Shell) run this check first and
stop if resources exist without state. To do it by hand, run the commands
below from a machine with `aws`, `az` and `terraform`. Each cloud ends up in
one of two branches: **A. fresh** (nothing exists) or **B. in place**
(resources exist, apply the trimmed profile over them).

### AWS

```bash
aws sts get-caller-identity
curl -s --max-time 10 https://aws.api.aegisimaging.ai/healthz; echo
terraform -chdir=terraform/aws init
terraform -chdir=terraform/aws state list | wc -l
aws ecs list-services --cluster aegis-cluster --region us-east-1 2>/dev/null || echo "no cluster"
```

- `state list` prints 0 **and** `list-services` says no cluster → **A. fresh**.
- `state list` prints resources → **B. in place** (whether or not `curl` answered).
- Resources exist but state is empty → stop; the state file was lost. Recover it
  from the S3 backend history or `terraform import` before applying anything.

### Azure

```bash
az account show
curl -s --max-time 10 https://azure.api.aegisimaging.ai/healthz; echo
terraform -chdir=terraform/azure init
terraform -chdir=terraform/azure state list | wc -l
az containerapp list -g aegis-prod -o table 2>/dev/null || echo "no resource group"
```

Same interpretation. The resource group is `<project_name>-<environment>`
(`aegis-prod` with the example tfvars).

## AWS

### 1. tfvars

In `terraform/aws/terraform.tfvars` (copy `terraform.tfvars.example` if it does
not exist), alongside the usual required values:

```hcl
enable_sidecars      = false
enable_dwv           = false
enable_mcp_server    = false
api_cpu              = 512
api_memory           = 1024
admin_cpu            = 256
admin_memory         = 512
dimse_receiver_image = ""                      # no DIMSE EC2 instance
api_domain           = "aws.api.aegisimaging.ai"
admin_domain         = "aws.admin.aegisimaging.ai"
```

`acm_certificate_arn` must be an **issued** certificate in us-east-1 covering
both hostnames. If the old certificate is still in ACM, reuse its ARN;
otherwise request one (aws-deployment.md, Phase 2) and put its validation
CNAME records in Cloudflare as DNS-only records.

### 2A. Fresh

Everything runs from GitHub Actions (`ci-cd.md`): `scripts/aws_bootstrap_ci.sh`
in AWS CloudShell bootstraps the identity, the certificate and the tfvars and
prints the secrets and variables (including `AWS_BUILD_SERVICES` =
`api admin-dashboard`). Set them, then dispatch **Terraform AWS** with `APPLY`
and approve `aws-prod`. When it finishes, dispatch **Deploy to AWS** once so
the two images exist and the tasks start; every later merge to `main` rolls
them automatically. Then `./scripts/aws_bootstrap_ci.sh --post-apply` creates
the first Cognito user for `first_admin_email` (the API seeds that address as
an admin) and prints the DNS records for step 3.

### 2B. In place

Update the `AWS_TERRAFORM_TFVARS` secret, dispatch **Terraform AWS** with
`confirm_apply` empty and read the plan. Expect: the nine sidecar services and
task definitions, `aegis-dwv` and `aegis-mcp-server` destroyed; the API and
admin task definitions updated (new sizes, blank `*_SERVICE_URL`). Nothing
touches RDS, S3, the ALB or Cognito. Then dispatch again with `APPLY` and
approve `aws-prod`.

Images already in ECR are reused; each merge to `main` rebuilds and rolls
the services that exist.

### 3. DNS (Cloudflare, DNS-only / grey cloud)

```
CNAME  aws.api    ->  <terraform -chdir=terraform/aws output -raw alb_dns>
CNAME  aws.admin  ->  <same ALB DNS name>
```

The Cognito callback URL defaults to `https://aws.admin.aegisimaging.ai/oauth2/idpresponse`
whenever `admin_domain` is set, so no Cognito change is needed.

### 4. Verify

```bash
curl -s https://aws.api.aegisimaging.ai/healthz | jq .
```

`status` should be `ok`, `database` and `storage` `healthy`, and `services`
empty (no sidecars). `https://aws.admin.aegisimaging.ai` must show the Cognito
sign-in page.

## Azure

### 1. tfvars

In `terraform/azure/terraform.tfvars`:

```hcl
enable_sidecars      = false
enable_dwv           = false
enable_mcp_server    = false
enable_upload_portal = false
api_cpu              = 0.5
api_memory           = "1Gi"
db_sku_name          = "B_Standard_B1ms"
dimse_receiver_image = ""
api_image_tag        = "placeholder"           # fresh installs only, see 2A
api_domain           = ""                      # set in step 3, after DNS exists
admin_domain         = ""
admin_dashboard_url  = "https://azure.admin.aegisimaging.ai"
```

`enable_landing` is already `false` by default — the public landing page is
served by Cloudflare Pages from `frontend/landing`.

### 2A. Fresh

`scripts/azure_bootstrap_ci.sh` in Azure Cloud Shell bootstraps the state
storage, the identity and the tfvars and prints every secret and variable
(including `AZURE_BUILD_SERVICES` = `api admin-dashboard`,
`AZURE_RESOURCE_GROUP` and `AZURE_ACR_LOGIN_SERVER`, which are deterministic).
Set them, then dispatch **Terraform Azure** with `APPLY` and approve
`azure-prod`. That creates everything with the hello-world placeholder image.
Then dispatch **Deploy to Azure** once: it builds the two images and points
the apps at them. Terraform ignores image changes after creation, so the
deploy workflow is the deploy path from here on. Step 3 is
`./scripts/azure_bootstrap_ci.sh --post-apply` (prints the records) followed
by `--bind-domains` (requests the certificates) once the tfvars carry the
domains.

### 2B. In place

Update the `AZURE_TERRAFORM_TFVARS` secret, dispatch **Terraform Azure** with
`confirm_apply` empty and read the plan. Expect: the nine sidecar apps, `dwv`,
`mcp-server`, `landing` and `upload-portal` destroyed; the API app updated
(cpu/memory, sidecar env vars removed); the admin app updated
(`MCP_SERVER_URL`); PostgreSQL resized. Resizing the flexible server restarts
it — a few minutes of API 503s. Then dispatch again with `APPLY` and approve
`azure-prod`.

### 3. Custom domains

```bash
terraform -chdir=terraform/azure output api_fqdn admin_dashboard_fqdn custom_domain_verification_id
```

Create in Cloudflare (DNS-only):

```
CNAME  azure.api          ->  <api_fqdn>
TXT    asuid.azure.api    ->  <custom_domain_verification_id>
CNAME  azure.admin        ->  <admin_dashboard_fqdn>
TXT    asuid.azure.admin  ->  <custom_domain_verification_id>
```

Once they resolve, set `api_domain = "azure.api.aegisimaging.ai"` and
`admin_domain = "azure.admin.aegisimaging.ai"` in tfvars, apply, then request
the free managed certificates:

```bash
ENV=$(terraform -chdir=terraform/azure output -raw container_app_environment_name)
az containerapp hostname bind -g "$RG" -n aegis-prod-api \
  --hostname azure.api.aegisimaging.ai   --environment "$ENV" --validation-method CNAME
az containerapp hostname bind -g "$RG" -n aegis-prod-admin-dashboard \
  --hostname azure.admin.aegisimaging.ai --environment "$ENV" --validation-method CNAME
```

Certificates take 5–15 minutes. Easy Auth's redirect URI is derived from
`admin_dashboard_url`, so admin sign-in only works through the custom hostname,
and the signing-in account must belong to the `azure_ad_tenant_id` tenant and
match `first_admin_email`.

### 4. Verify

```bash
curl -s https://azure.api.aegisimaging.ai/healthz | jq .
```

Same expectations as AWS. `https://azure.admin.aegisimaging.ai` must redirect
to the Microsoft sign-in page.

## Landing page

The "Live on three clouds" cards on aegisimaging.ai probe each `/healthz` from
the visitor's browser and link to it. They turn green on their own once the
endpoints answer over HTTPS — nothing to configure.

## Going back to full size

Remove the `enable_*` and sizing lines from the tfvars secrets (or set the
flags to `true`), clear the `AWS_BUILD_SERVICES` / `AZURE_BUILD_SERVICES`
variables, and dispatch the terraform workflow with `APPLY`, then the deploy
workflow once.
