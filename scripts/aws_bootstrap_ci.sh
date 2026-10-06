#!/usr/bin/env bash
# aws_bootstrap_ci.sh — one-time AWS bootstrap for the GitHub Actions pipeline.
#
# Run in AWS CloudShell (region us-east-1) from a clone of this repository,
# signed in as an administrator of the target account:
#
#   git clone -b develop https://github.com/aegis-imaging/aegis.git && cd aegis
#   ./scripts/aws_bootstrap_ci.sh                 # bootstrap (safe to re-run)
#   ./scripts/aws_bootstrap_ci.sh --post-apply    # after Terraform AWS + Deploy to AWS ran
#
# Bootstrap:
#   1. Reports whether the account is fresh or already carries an AEGIS deployment.
#   2. Installs terraform into ~/.local/bin when CloudShell lacks it.
#   3. Creates the S3 state bucket and DynamoDB lock table if missing and
#      imports them into state so the first CI apply does not try to create them.
#   4. Finds or requests the ACM certificate for the API and admin hostnames,
#      prints its validation CNAMEs for Cloudflare, and waits until it is issued.
#   5. Applies only the GitHub OIDC provider and roles (terraform -target).
#   6. Writes terraform/aws/terraform.tfvars with the minimal-footprint profile
#      and prints the GitHub secrets and variables to set.
# Post-apply:
#   Prints the Cloudflare CNAME records for the load balancer, creates the first
#   Cognito user, and prints the health check to run.
#
# Overrides: REGION, API_HOST, ADMIN_HOST, ADMIN_EMAIL, ENVIRONMENT, PROJECT_NAME, TF_VERSION.
set -euo pipefail

REGION="${REGION:-us-east-1}"
API_HOST="${API_HOST:-aws.api.aegisimaging.ai}"
ADMIN_HOST="${ADMIN_HOST:-aws.admin.aegisimaging.ai}"
ADMIN_EMAIL="${ADMIN_EMAIL:-ops@aegisimaging.ai}"
PROJECT_NAME="${PROJECT_NAME:-aegis}"
TF_VERSION="${TF_VERSION:-1.9.8}"
LOCK_TABLE="aegis-terraform-locks"
TF_DIR="terraform/aws"
MODE="${1:-bootstrap}"

cd "$(dirname "$0")/.."
# The state bucket name lives in backend.tf (S3 names are global, so it carries a suffix).
STATE_BUCKET="$(awk -F'"' '/^[[:space:]]+bucket[[:space:]]+=/{print $2; exit}' "$TF_DIR/backend.tf")"
export AWS_DEFAULT_REGION="$REGION" AWS_PAGER=""
export PATH="$HOME/.local/bin:$PATH"

command -v aws >/dev/null 2>&1 || { echo "error: aws CLI required" >&2; exit 1; }
ACCOUNT="$(aws sts get-caller-identity --query Account --output text)"
echo "==> account $ACCOUNT, region $REGION"

ensure_terraform() {
  if ! command -v terraform >/dev/null 2>&1; then
    mkdir -p "$HOME/.local/bin"
    curl -fsSL "https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_amd64.zip" -o /tmp/terraform.zip
    unzip -oq /tmp/terraform.zip -d "$HOME/.local/bin" && rm -f /tmp/terraform.zip
    echo "==> installed terraform $TF_VERSION into ~/.local/bin"
  fi
  echo "==> $(terraform version | head -1)"
}

tf() { terraform -chdir="$TF_DIR" "$@"; }
in_state() { tf state list 2>/dev/null | grep -qx "$1"; }

# ── post-apply ───────────────────────────────────────────────────────────────
if [ "$MODE" = "--post-apply" ]; then
  ensure_terraform
  tf init -input=false >/dev/null
  ALB="$(tf output -raw alb_dns)"
  POOL="$(tf output -raw cognito_user_pool_id)"
  cat <<EOF

Cloudflare DNS records (DNS only, grey cloud):
  CNAME  ${API_HOST%%.aegisimaging.ai}    ->  $ALB
  CNAME  ${ADMIN_HOST%%.aegisimaging.ai}  ->  $ALB
EOF
  if aws cognito-idp admin-get-user --user-pool-id "$POOL" --username "$ADMIN_EMAIL" >/dev/null 2>&1; then
    echo "==> Cognito user $ADMIN_EMAIL already exists"
  else
    TEMP_PW="Aegis!$(head -c 6 /dev/urandom | od -An -tx1 | tr -d ' \n')9"
    aws cognito-idp admin-create-user --user-pool-id "$POOL" --username "$ADMIN_EMAIL" \
      --user-attributes Name=email,Value="$ADMIN_EMAIL" Name=email_verified,Value=true \
      --temporary-password "$TEMP_PW" --message-action SUPPRESS >/dev/null
    echo "==> created Cognito user $ADMIN_EMAIL with temporary password: $TEMP_PW"
    echo "    (sign in at https://$ADMIN_HOST once DNS resolves; it asks for a new password)"
  fi
  cat <<EOF

Verify when the records resolve:
  curl -s https://$API_HOST/healthz
  open https://$ADMIN_HOST   (Cognito sign-in page)
EOF
  exit 0
fi

# ── 1. discovery ─────────────────────────────────────────────────────────────
have_state=0; have_cluster=0
aws s3api head-object --bucket "$STATE_BUCKET" --key aws/terraform.tfstate >/dev/null 2>&1 && have_state=1
[ "$(aws ecs describe-clusters --clusters "$PROJECT_NAME-cluster" --query 'clusters[0].status' --output text 2>/dev/null)" = "ACTIVE" ] && have_cluster=1
if [ "$have_state" = 1 ]; then
  echo "==> existing terraform state found: the CI apply will trim the deployment in place"
elif [ "$have_cluster" = 1 ]; then
  echo "error: ECS cluster $PROJECT_NAME-cluster exists but there is no state file." >&2
  echo "       Recover the state (S3 bucket $STATE_BUCKET version history) before continuing." >&2
  exit 1
else
  echo "==> fresh account: no AEGIS state or cluster found"
fi

# ── 2. terraform ─────────────────────────────────────────────────────────────
ensure_terraform

# ── 3. state backend ─────────────────────────────────────────────────────────
if ! aws s3api head-bucket --bucket "$STATE_BUCKET" 2>/dev/null; then
  if [ "$REGION" = "us-east-1" ]; then
    aws s3api create-bucket --bucket "$STATE_BUCKET" >/dev/null
  else
    aws s3api create-bucket --bucket "$STATE_BUCKET" --create-bucket-configuration "LocationConstraint=$REGION" >/dev/null
  fi
  echo "==> created state bucket $STATE_BUCKET"
fi
aws s3api put-bucket-versioning --bucket "$STATE_BUCKET" --versioning-configuration Status=Enabled
aws s3api put-public-access-block --bucket "$STATE_BUCKET" \
  --public-access-block-configuration BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
if ! aws dynamodb describe-table --table-name "$LOCK_TABLE" >/dev/null 2>&1; then
  aws dynamodb create-table --table-name "$LOCK_TABLE" \
    --attribute-definitions AttributeName=LockID,AttributeType=S \
    --key-schema AttributeName=LockID,KeyType=HASH --billing-mode PAY_PER_REQUEST >/dev/null
  aws dynamodb wait table-exists --table-name "$LOCK_TABLE"
  echo "==> created lock table $LOCK_TABLE"
fi
tf init -input=false >/dev/null
echo "==> terraform backend ready"

# ── 4. certificate ───────────────────────────────────────────────────────────
find_cert() {
  local arn sans
  for arn in $(aws acm list-certificates --certificate-statuses ISSUED PENDING_VALIDATION \
                 --query 'CertificateSummaryList[].CertificateArn' --output text); do
    sans=" $(aws acm describe-certificate --certificate-arn "$arn" \
              --query 'Certificate.SubjectAlternativeNames' --output text | tr '\t' ' ') "
    if [[ "$sans" == *" $API_HOST "* && "$sans" == *" $ADMIN_HOST "* ]]; then echo "$arn"; return 0; fi
  done
  return 1
}
CERT_ARN="$(find_cert || true)"
if [ -z "$CERT_ARN" ]; then
  CERT_ARN="$(aws acm request-certificate --domain-name "$API_HOST" \
               --subject-alternative-names "$ADMIN_HOST" --validation-method DNS \
               --query CertificateArn --output text)"
  echo "==> requested certificate $CERT_ARN"
  sleep 8
fi
if [ "$(aws acm describe-certificate --certificate-arn "$CERT_ARN" --query Certificate.Status --output text)" != "ISSUED" ]; then
  echo
  echo "==> ADD THESE RECORDS IN CLOUDFLARE (type CNAME, DNS only), then leave this running:"
  aws acm describe-certificate --certificate-arn "$CERT_ARN" \
    --query 'Certificate.DomainValidationOptions[].ResourceRecord.[Name,Value]' --output text \
    | awk '{printf "    name: %s\n    target: %s\n\n", $1, $2}'
  echo "    (Cloudflare accepts the full name; it trims the zone itself. Re-run if the list is empty.)"
  echo "==> waiting for ACM to issue the certificate (checks every minute, up to 40 minutes)..."
  aws acm wait certificate-validated --certificate-arn "$CERT_ARN"
fi
echo "==> certificate issued: $CERT_ARN"

# ── 5. tfvars ────────────────────────────────────────────────────────────────
ENVIRONMENT="${ENVIRONMENT:-}"
COGNITO_PREFIX=""
if in_state aws_cognito_user_pool.admin; then
  pool_name="$(tf state show aws_cognito_user_pool.admin | awk -F'"' '/^[[:space:]]+name[[:space:]]+=/{print $2; exit}')"
  [ -n "$ENVIRONMENT" ] || ENVIRONMENT="${pool_name#"$PROJECT_NAME"-}"; ENVIRONMENT="${ENVIRONMENT%-admin-users}"
fi
if in_state aws_cognito_user_pool_domain.admin; then
  COGNITO_PREFIX="$(tf state show aws_cognito_user_pool_domain.admin | awk -F'"' '/^[[:space:]]+domain[[:space:]]+=/{print $2; exit}')"
fi
[ -n "$ENVIRONMENT" ] || ENVIRONMENT="prod"
[ -n "$COGNITO_PREFIX" ] || COGNITO_PREFIX="$PROJECT_NAME-$(head -c 4 /dev/urandom | od -An -tx1 | tr -d ' \n')"

TFVARS="$TF_DIR/terraform.tfvars"
cat > "$TFVARS" <<EOF
aws_region   = "$REGION"
environment  = "$ENVIRONMENT"
project_name = "$PROJECT_NAME"

acm_certificate_arn   = "$CERT_ARN"
cognito_domain_prefix = "$COGNITO_PREFIX"

api_domain   = "$API_HOST"
admin_domain = "$ADMIN_HOST"

first_admin_email = "$ADMIN_EMAIL"
alert_email       = "$ADMIN_EMAIL"

# Minimal footprint (docs/runbooks/minimal-footprint.md)
enable_sidecars      = false
enable_dwv           = false
enable_mcp_server    = false
api_cpu              = 512
api_memory           = 1024
admin_cpu            = 256
admin_memory         = 512
dimse_receiver_image = ""
EOF
echo "==> wrote $TFVARS (environment=$ENVIRONMENT, cognito prefix=$COGNITO_PREFIX)"

# ── 6. import pre-existing pieces, then apply only the GitHub identity ───────
import_if() { in_state "$1" || { tf import -input=false "$1" "$2" >/dev/null && echo "    imported $1"; }; }
import_if aws_s3_bucket.tf_state "$STATE_BUCKET"
import_if aws_s3_bucket_versioning.tf_state "$STATE_BUCKET"
import_if aws_s3_bucket_server_side_encryption_configuration.tf_state "$STATE_BUCKET"
import_if aws_s3_bucket_public_access_block.tf_state "$STATE_BUCKET"
import_if aws_dynamodb_table.tf_locks "$LOCK_TABLE"
OIDC_ARN="$(aws iam list-open-id-connect-providers --query 'OpenIDConnectProviderList[].Arn' --output text | tr '\t' '\n' | grep 'token.actions.githubusercontent.com' || true)"
[ -z "$OIDC_ARN" ] || import_if 'aws_iam_openid_connect_provider.github_actions[0]' "$OIDC_ARN"
for role in aegis-github-actions-terraform:github_actions_terraform aegis-github-actions-deploy:github_actions_deploy; do
  name="${role%%:*}"; addr="${role##*:}"
  aws iam get-role --role-name "$name" >/dev/null 2>&1 && import_if "aws_iam_role.${addr}[0]" "$name"
done

echo "==> applying the GitHub OIDC provider and roles"
tf apply -input=false -auto-approve \
  -target=aws_iam_openid_connect_provider.github_actions \
  -target=aws_iam_role.github_actions_terraform \
  -target=aws_iam_role_policy_attachment.github_actions_terraform_admin \
  -target=aws_iam_role.github_actions_deploy \
  -target=aws_iam_policy.github_actions_deploy \
  -target=aws_iam_role_policy_attachment.github_actions_deploy \
  | grep -E '^(Apply complete|Error|.*(created|imported|updated))' || true

TF_ROLE="$(tf output -raw github_actions_terraform_role_arn)"
DEPLOY_ROLE="$(tf output -raw github_actions_deploy_role_arn)"

cat <<EOF

================ GitHub → Settings → Secrets and variables → Actions ================

Secrets:
  AWS_TERRAFORM_ROLE_ARN = $TF_ROLE
  AWS_DEPLOY_ROLE_ARN    = $DEPLOY_ROLE
  AWS_TERRAFORM_TFVARS   = the whole file printed below

Variables:
  AWS_CI_ENABLED     = true
  AWS_REGION         = $REGION
  AWS_API_URL        = https://$API_HOST
  AWS_ADMIN_URL      = https://$ADMIN_HOST
  AWS_BUILD_SERVICES = api admin-dashboard
  AWS_PLAN_ARCHIVE   = s3://$STATE_BUCKET/plans

Environment: aws-prod with yourself as required reviewer (Settings → Environments).

Then: Actions → Terraform AWS → Run workflow (main, confirm_apply = APPLY), approve aws-prod;
      Actions → Deploy to AWS → Run workflow (main);
      back here: ./scripts/aws_bootstrap_ci.sh --post-apply

---------------- AWS_TERRAFORM_TFVARS (copy everything between the lines) ----------------
$(cat "$TFVARS")
------------------------------------------------------------------------------------------
EOF
