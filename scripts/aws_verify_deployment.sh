#!/usr/bin/env bash
set -euo pipefail

TFVARS_PATH="terraform/aws/terraform.tfvars"
TERRAFORM_DIR="terraform/aws"
AWS_REGION=""
PROJECT_NAME=""
ALB_DNS=""
EXPECT_ADMIN_AUTH="true"

usage() {
  cat <<'USAGE'
Usage: ./scripts/aws_verify_deployment.sh [options]

Options:
  --tfvars=<path>              Default: terraform/aws/terraform.tfvars
  --terraform-dir=<path>       Default: terraform/aws
  --region=<region>            Override region (else read from tfvars or us-east-1)
  --project-name=<name>        Override project name (else read from tfvars or aegis)
  --alb-dns=<dns-name>         Override ALB DNS (else terraform output alb_dns)
  --expect-admin-auth=<bool>   Whether admin root should challenge auth (default: true)
  -h, --help                   Show this help

Checks:
1) required local tools + tfvars validation
2) AWS credential access (STS caller identity)
3) terraform outputs for ALB/ECS/Cognito
4) API /healthz over ALB returns HTTP 200
5) admin route challenges unauthenticated requests
6) ECS api/admin services are ACTIVE and running desired task count
7) API/admin target groups have healthy backend targets
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
}

pass() {
  echo "PASS: $*"
}

tfvars_read_string() {
  local key="$1"
  local file="$2"
  awk -F= -v key="$key" '
    $1 ~ "^[[:space:]]*" key "[[:space:]]*$" {
      value = $2
      sub(/^[[:space:]]*/, "", value)
      sub(/[[:space:]]*#.*/, "", value)
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      print value
      exit
    }
  ' "$file"
}

normalize_bool() {
  local raw="$1"
  raw="$(echo "$raw" | tr '[:upper:]' '[:lower:]')"
  case "$raw" in
    true|1|yes|y) echo "true" ;;
    false|0|no|n) echo "false" ;;
    *) echo "" ;;
  esac
}

terraform_output_raw() {
  local key="$1"
  terraform -chdir="$TERRAFORM_DIR" output -raw "$key" 2>/dev/null
}

check_ecs_service() {
  local cluster="$1"
  local service="$2"

  local status
  local desired
  local running
  local pending

  status="$(aws ecs describe-services --region "$AWS_REGION" --cluster "$cluster" --services "$service" --query 'services[0].status' --output text 2>/dev/null || true)"
  desired="$(aws ecs describe-services --region "$AWS_REGION" --cluster "$cluster" --services "$service" --query 'services[0].desiredCount' --output text 2>/dev/null || true)"
  running="$(aws ecs describe-services --region "$AWS_REGION" --cluster "$cluster" --services "$service" --query 'services[0].runningCount' --output text 2>/dev/null || true)"
  pending="$(aws ecs describe-services --region "$AWS_REGION" --cluster "$cluster" --services "$service" --query 'services[0].pendingCount' --output text 2>/dev/null || true)"

  [ "$status" = "ACTIVE" ] || fail "ECS service '$service' not ACTIVE (status=$status)"
  [ "$desired" != "None" ] || fail "ECS service '$service' desired count unavailable"
  [ "$running" != "None" ] || fail "ECS service '$service' running count unavailable"
  [ "$pending" != "None" ] || fail "ECS service '$service' pending count unavailable"
  [ "$running" = "$desired" ] || fail "ECS service '$service' running/desired mismatch ($running/$desired)"
  [ "$pending" = "0" ] || fail "ECS service '$service' has pending tasks ($pending)"

  pass "ECS service '$service' running desired task count ($running/$desired)"
}

check_target_group_health() {
  local tg_name="$1"

  local tg_arn
  local healthy_count
  tg_arn="$(aws elbv2 describe-target-groups --region "$AWS_REGION" --names "$tg_name" --query 'TargetGroups[0].TargetGroupArn' --output text 2>/dev/null || true)"
  [ -n "$tg_arn" ] && [ "$tg_arn" != "None" ] || fail "target group '$tg_name' not found"

  healthy_count="$(
    aws elbv2 describe-target-health \
      --region "$AWS_REGION" \
      --target-group-arn "$tg_arn" \
      --query "length(TargetHealthDescriptions[?TargetHealth.State=='healthy'])" \
      --output text \
      2>/dev/null || true
  )"
  [ "$healthy_count" != "None" ] || fail "unable to read health for target group '$tg_name'"
  case "$healthy_count" in
    ''|*[!0-9]*) fail "unexpected healthy target count '$healthy_count' for target group '$tg_name'" ;;
  esac
  [ "$healthy_count" -ge 1 ] || fail "target group '$tg_name' has no healthy targets"

  pass "target group '$tg_name' has $healthy_count healthy target(s)"
}

for arg in "$@"; do
  case "$arg" in
    --tfvars=*) TFVARS_PATH="${arg#*=}" ;;
    --terraform-dir=*) TERRAFORM_DIR="${arg#*=}" ;;
    --region=*) AWS_REGION="${arg#*=}" ;;
    --project-name=*) PROJECT_NAME="${arg#*=}" ;;
    --alb-dns=*) ALB_DNS="${arg#*=}" ;;
    --expect-admin-auth=*) EXPECT_ADMIN_AUTH="${arg#*=}" ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument '$arg'"
      ;;
  esac
done

for cmd in terraform aws curl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "required command not found: $cmd"
  fi
done

if [ ! -f "$TFVARS_PATH" ]; then
  fail "tfvars file not found: $TFVARS_PATH"
fi

if [ ! -d "$TERRAFORM_DIR" ]; then
  fail "terraform directory not found: $TERRAFORM_DIR"
fi

./scripts/validate_tfvars_required.sh \
  "$TFVARS_PATH" \
  acm_certificate_arn \
  cognito_domain_prefix

if [ -z "$AWS_REGION" ]; then
  AWS_REGION="$(tfvars_read_string "aws_region" "$TFVARS_PATH")"
fi
AWS_REGION="${AWS_REGION:-us-east-1}"

if [ -z "$PROJECT_NAME" ]; then
  PROJECT_NAME="$(tfvars_read_string "project_name" "$TFVARS_PATH")"
fi
PROJECT_NAME="${PROJECT_NAME:-aegis}"

if [ -z "$ALB_DNS" ]; then
  ALB_DNS="$(terraform_output_raw "alb_dns" || true)"
fi
[ -n "$ALB_DNS" ] || fail "alb dns could not be resolved (terraform output alb_dns unavailable)"

EXPECT_ADMIN_AUTH="$(normalize_bool "$EXPECT_ADMIN_AUTH")"
EXPECT_ADMIN_AUTH="${EXPECT_ADMIN_AUTH:-true}"

if ! aws sts get-caller-identity --region "$AWS_REGION" >/dev/null 2>&1; then
  fail "AWS credentials unavailable for region $AWS_REGION (run aws configure or export env credentials)"
fi
pass "AWS caller identity available in region $AWS_REGION"

ecs_cluster="$(terraform_output_raw "ecs_cluster" || true)"
ecs_api_service="$(terraform_output_raw "ecs_api_service" || true)"
ecs_admin_service="$(terraform_output_raw "ecs_admin_service" || true)"
cognito_user_pool_id="$(terraform_output_raw "cognito_user_pool_id" || true)"

[ -n "$ecs_cluster" ] || fail "terraform output ecs_cluster unavailable"
[ -n "$ecs_api_service" ] || fail "terraform output ecs_api_service unavailable"
[ -n "$ecs_admin_service" ] || fail "terraform output ecs_admin_service unavailable"
[ -n "$cognito_user_pool_id" ] || fail "terraform output cognito_user_pool_id unavailable"
pass "terraform outputs for ECS + Cognito are present"

health_code="$(curl -sS -o /dev/null -w '%{http_code}' "https://${ALB_DNS}/healthz")"
[ "$health_code" = "200" ] || fail "API health endpoint failed (https://${ALB_DNS}/healthz -> HTTP $health_code)"
pass "API /healthz returned HTTP 200 via ALB"

admin_code="$(curl -sS -o /dev/null -w '%{http_code}' "https://${ALB_DNS}/")"
if [ "$EXPECT_ADMIN_AUTH" = "true" ]; then
  case "$admin_code" in
    302|401|403) pass "admin route enforces auth challenge (HTTP $admin_code)" ;;
    *) fail "expected admin auth challenge (302/401/403), got HTTP $admin_code from https://${ALB_DNS}/" ;;
  esac
else
  case "$admin_code" in
    2*|3*|4*) pass "admin route reachable without strict auth requirement (HTTP $admin_code)" ;;
    *) fail "admin route returned unexpected HTTP $admin_code from https://${ALB_DNS}/" ;;
  esac
fi

aws cognito-idp describe-user-pool --region "$AWS_REGION" --user-pool-id "$cognito_user_pool_id" >/dev/null 2>&1 || \
  fail "unable to describe Cognito user pool '$cognito_user_pool_id'"
pass "Cognito user pool '$cognito_user_pool_id' exists"

check_ecs_service "$ecs_cluster" "$ecs_api_service"
check_ecs_service "$ecs_cluster" "$ecs_admin_service"

check_target_group_health "${PROJECT_NAME}-api"
check_target_group_health "${PROJECT_NAME}-admin"

echo "ok: aws deployment verification checks passed"
