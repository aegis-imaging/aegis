.PHONY: up down clean build api lint lint-go lint-python lint-python-scripts lint-frontend lint-mcp lint-shell lint-terraform lint-infra-guard check logs test test-unit test-race smoke cloud-smoke-from-terraform dimse-e2e gcp-preflight gcp-bootstrap-project gcp-build-images gcp-apply-infra gcp-install-poc gcp-verify-deployment aws-build-images aws-apply-infra aws-install-poc aws-verify-deployment

# ── Docker Compose ──────────────────────────────────────────────────

up:
	docker compose up -d

down:
	docker compose down

clean:
	docker compose down -v

build:
	docker compose build

logs:
	docker compose logs -f

# ── Local dev (outside Docker) ──────────────────────────────────────

api:
	cd api && go run .

# ── Lint ────────────────────────────────────────────────────────────

lint: lint-go lint-python lint-python-scripts lint-frontend lint-mcp lint-shell lint-terraform lint-infra-guard

lint-go:
	cd api && go vet ./...

lint-python:
	@for svc in defacing phi-detection qc-service bids-service classification-service protocol-service dimse-receiver; do \
		echo "Checking $$svc..."; \
		find $$svc -name '*.py' -exec python3 -m py_compile {} +; \
	done

lint-python-scripts:
	python3 -m py_compile scripts/cloud_smoke_test.py scripts/dimse_pacs_e2e_harness.py

lint-frontend:
	cd client && npx tsc --noEmit
	cd frontend/upload-portal && npx tsc --noEmit
	cd frontend/admin-dashboard && npx tsc --noEmit
	cd frontend/landing && npx tsc --noEmit
	cd frontend/export-portal && npx tsc --noEmit

lint-mcp:
	cd mcp-server && npm run typecheck

lint-shell:
	@for s in scripts/*.sh; do bash -n "$$s" && echo "OK: $$s"; done

lint-terraform:
	terraform -chdir=terraform/project fmt -check
	terraform -chdir=terraform/infra fmt -check
	terraform -chdir=terraform/aws fmt -check

lint-infra-guard:
	./scripts/check-infra-placeholders.sh

# ── Tests ──────────────────────────────────────────────────────────

test:
	cd api && go test -v -count=1 ./...

test-unit:
	cd api && go test -short -v ./...

test-race:
	cd api && go test -race -count=1 ./...

# ── Health check ────────────────────────────────────────────────────

check:
	@curl -sf http://localhost:8080/healthz | python3 -m json.tool 2>/dev/null || echo "API: DOWN"

# Seed the first admin user when the admin_users table is empty (dev mode).
# In production, set FIRST_ADMIN_EMAIL env var on the API Cloud Run service instead.
# Usage: make first-admin EMAIL=ops@example.com [NAME="AEGIS Ops"] [API_URL=http://localhost:8080]
first-admin:
	@if [ -z "$(EMAIL)" ]; then \
		echo "usage: make first-admin EMAIL=ops@example.com [NAME='AEGIS Ops'] [API_URL=http://localhost:8080]"; \
		exit 1; \
	fi
	@URL="$${API_URL:-http://localhost:8080}"; \
	NAME_VAL="$${NAME:-$(EMAIL)}"; \
	echo "Creating admin user $(EMAIL) at $$URL ..."; \
	curl -sf -X POST "$$URL/api/admin-users" \
		-H "Content-Type: application/json" \
		-d "{\"email\":\"$(EMAIL)\",\"name\":\"$$NAME_VAL\",\"role\":\"admin\",\"enabled\":true}" \
	| python3 -m json.tool 2>/dev/null || echo "(API returned non-JSON or error — check that AUTH_ENABLED=false)"

smoke:
	@if [ -z "$(BASE_URL)" ]; then \
		echo "usage: make smoke BASE_URL=https://api-dev.aegisimaging.ai [ADMIN_HEADER='Header: value'] [IAP_EMAIL=user@example.com] [PROJECT_SLUG=default]"; \
		exit 1; \
	fi
	@CMD="python3 scripts/cloud_smoke_test.py --base-url $(BASE_URL) --project-slug $${PROJECT_SLUG:-default}"; \
	if [ -n "$$ADMIN_HEADER" ]; then CMD="$$CMD --admin-header '$$ADMIN_HEADER'"; fi; \
	if [ -n "$$IAP_EMAIL" ]; then CMD="$$CMD --iap-email '$$IAP_EMAIL'"; fi; \
	eval "$$CMD"

cloud-smoke-from-terraform:
	@CMD="./scripts/cloud_smoke_from_terraform.sh"; \
	if [ -n "$$PROVIDER" ]; then CMD="$$CMD --provider='$$PROVIDER'"; fi; \
	if [ -n "$$TERRAFORM_DIR" ]; then CMD="$$CMD --terraform-dir='$$TERRAFORM_DIR'"; fi; \
	if [ -n "$$BASE_URL" ]; then CMD="$$CMD --base-url='$$BASE_URL'"; fi; \
	if [ -n "$$PROJECT_SLUG" ]; then CMD="$$CMD --project-slug='$$PROJECT_SLUG'"; fi; \
	if [ -n "$$TIMEOUT_SECONDS" ]; then CMD="$$CMD --timeout='$$TIMEOUT_SECONDS'"; fi; \
	if [ -n "$$PIPELINE_TIMEOUT_SECONDS" ]; then CMD="$$CMD --pipeline-timeout='$$PIPELINE_TIMEOUT_SECONDS'"; fi; \
	if [ -n "$$ADMIN_HEADER" ]; then CMD="$$CMD --admin-header='$$ADMIN_HEADER'"; fi; \
	if [ -n "$$IAP_EMAIL" ]; then CMD="$$CMD --iap-email='$$IAP_EMAIL'"; fi; \
	if [ -n "$$REQUIRE_AUTH_CONTEXT" ]; then CMD="$$CMD --require-auth-context='$$REQUIRE_AUTH_CONTEXT'"; fi; \
	eval "$$CMD"

dimse-e2e:
	@if [ -x ".venv/bin/python" ]; then \
		.venv/bin/python scripts/dimse_pacs_e2e_harness.py; \
	else \
		python3 scripts/dimse_pacs_e2e_harness.py; \
	fi

gcp-preflight:
	./scripts/gcp_preflight.sh

gcp-bootstrap-project:
	./scripts/gcp_bootstrap_project.sh

gcp-build-images:
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "usage: make gcp-build-images PROJECT_ID=<gcp-project> [REGION=us-central1] [TAG=latest] [REPOSITORY=aegis-services]"; \
		exit 1; \
	fi
	@REGION="$${REGION:-us-central1}" TAG="$${TAG:-latest}" REPOSITORY="$${REPOSITORY:-aegis-services}" \
		./scripts/gcp_build_push_images.sh --project-id="$(PROJECT_ID)" --region="$$REGION" --tag="$$TAG" --repository="$$REPOSITORY"

gcp-apply-infra:
	./scripts/gcp_apply_infra.sh

gcp-install-poc:
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "usage: make gcp-install-poc PROJECT_ID=<gcp-project> [REGION=us-central1] [TAG=latest]"; \
		exit 1; \
	fi
	@REGION="$${REGION:-us-central1}" TAG="$${TAG:-latest}" \
		./scripts/gcp_install_poc.sh --project-id="$(PROJECT_ID)" --region="$$REGION" --tag="$$TAG"

gcp-verify-deployment:
	@CMD="./scripts/gcp_verify_deployment.sh"; \
	if [ -n "$$TFVARS" ]; then CMD="$$CMD --tfvars='$$TFVARS'"; fi; \
	if [ -n "$$TERRAFORM_DIR" ]; then CMD="$$CMD --terraform-dir='$$TERRAFORM_DIR'"; fi; \
	if [ -n "$$PROJECT_ID" ]; then CMD="$$CMD --project-id='$$PROJECT_ID'"; fi; \
	if [ -n "$$REGION" ]; then CMD="$$CMD --region='$$REGION'"; fi; \
	if [ -n "$$API_URL" ]; then CMD="$$CMD --api-url='$$API_URL'"; fi; \
	if [ -n "$$ADMIN_URL" ]; then CMD="$$CMD --admin-url='$$ADMIN_URL'"; fi; \
	if [ -n "$$EXPECT_ADMIN_AUTH" ]; then CMD="$$CMD --expect-admin-auth='$$EXPECT_ADMIN_AUTH'"; fi; \
	eval "$$CMD"

aws-apply-infra:
	./scripts/aws_apply_infra.sh

aws-build-images:
	@REGION="$${REGION:-us-east-1}" PROJECT_NAME="$${PROJECT_NAME:-aegis}" TAG="$${TAG:-latest}" \
		./scripts/aws_build_push_images.sh --region="$$REGION" --project-name="$$PROJECT_NAME" --tag="$$TAG"

aws-install-poc:
	@REGION="$${REGION:-us-east-1}" PROJECT_NAME="$${PROJECT_NAME:-aegis}" TAG="$${TAG:-latest}" \
		./scripts/aws_install_poc.sh --region="$$REGION" --project-name="$$PROJECT_NAME" --tag="$$TAG"

aws-verify-deployment:
	@CMD="./scripts/aws_verify_deployment.sh"; \
	if [ -n "$$TFVARS" ]; then CMD="$$CMD --tfvars='$$TFVARS'"; fi; \
	if [ -n "$$TERRAFORM_DIR" ]; then CMD="$$CMD --terraform-dir='$$TERRAFORM_DIR'"; fi; \
	if [ -n "$$REGION" ]; then CMD="$$CMD --region='$$REGION'"; fi; \
	if [ -n "$$PROJECT_NAME" ]; then CMD="$$CMD --project-name='$$PROJECT_NAME'"; fi; \
	if [ -n "$$ALB_DNS" ]; then CMD="$$CMD --alb-dns='$$ALB_DNS'"; fi; \
	if [ -n "$$EXPECT_ADMIN_AUTH" ]; then CMD="$$CMD --expect-admin-auth='$$EXPECT_ADMIN_AUTH'"; fi; \
	eval "$$CMD"
