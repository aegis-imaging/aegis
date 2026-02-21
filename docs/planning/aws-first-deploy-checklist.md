# AEGIS AWS First Deploy Checklist

Created: 2026-02-20

This is the fastest path to stand up the first AWS-backed AEGIS environment with ALB + Cognito + ECS.

## One-Command Option

After `terraform/aws/terraform.tfvars` is configured, you can run the full flow with:

```bash
make aws-install-poc REGION=us-east-1 PROJECT_NAME=aegis TAG=latest
```

The detailed steps below are the same sequence, broken out for control and troubleshooting.

## 1) Verify AWS credentials and tooling

```bash
aws sts get-caller-identity
terraform version
```

## 2) Configure Terraform variables

```bash
cp terraform/aws/terraform.tfvars.example terraform/aws/terraform.tfvars
```

Set at minimum:
- `aws_region`
- `environment`
- `project_name`
- `acm_certificate_arn`
- `cognito_domain_prefix`
- `api_image_tag`
- `admin_image_tag`

Use unique image tags per deployment (for example Git SHA or date tag).

## 3) Plan/apply AWS infrastructure

```bash
./scripts/aws_apply_infra.sh --tfvars=terraform/aws/terraform.tfvars --apply
```

Equivalent Make target (plan/apply script wrapper):

```bash
make aws-apply-infra
```

## 4) Build and push images to ECR

```bash
./scripts/aws_build_push_images.sh --region=us-east-1 --project-name=aegis --tag=latest
```

Equivalent Make target:

```bash
make aws-build-images REGION=us-east-1 PROJECT_NAME=aegis TAG=latest
```

## 5) Roll new image tags through ECS

Update `api_image_tag` and `admin_image_tag` in `terraform/aws/terraform.tfvars`, then apply again:

```bash
./scripts/aws_apply_infra.sh --tfvars=terraform/aws/terraform.tfvars --apply
```

## 6) Create first Cognito admin user

```bash
USER_POOL_ID=$(terraform -chdir=terraform/aws output -raw cognito_user_pool_id)
aws cognito-idp admin-create-user \
  --user-pool-id "$USER_POOL_ID" \
  --username "<admin-email@example.com>" \
  --user-attributes Name=email,Value=<admin-email@example.com> Name=email_verified,Value=true
```

## 7) Verify deployment

```bash
ALB_DNS=$(terraform -chdir=terraform/aws output -raw alb_dns)
curl -I "https://$ALB_DNS/healthz"
```

Then:
- open `https://$ALB_DNS/` and confirm Cognito login challenge,
- log in and confirm admin dashboard loads,
- verify API-authenticated flows operate from the dashboard.

## 8) Post-deploy checks

- Confirm ECS services are stable:
```bash
aws ecs list-services --cluster "$(terraform -chdir=terraform/aws output -raw ecs_cluster)"
```
- Confirm target group health for API and admin target groups.
- Confirm RDS and CloudWatch logs show healthy startup for both ECS services.
