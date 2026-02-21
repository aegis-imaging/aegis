.PHONY: up down clean build api lint lint-go lint-python lint-frontend check logs test test-unit test-race smoke dimse-e2e gcp-preflight gcp-bootstrap-project gcp-build-images gcp-apply-infra gcp-install-poc aws-build-images aws-apply-infra

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

lint: lint-go lint-python lint-python-scripts lint-frontend lint-mcp

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

smoke:
	@if [ -z "$(BASE_URL)" ]; then \
		echo "usage: make smoke BASE_URL=https://api-dev.aegisimaging.ai [ADMIN_HEADER='Header: value'] [PROJECT_SLUG=default]"; \
		exit 1; \
	fi
	@CMD="python3 scripts/cloud_smoke_test.py --base-url $(BASE_URL) --project-slug $${PROJECT_SLUG:-default}"; \
	if [ -n "$$ADMIN_HEADER" ]; then CMD="$$CMD --admin-header '$$ADMIN_HEADER'"; fi; \
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

aws-apply-infra:
	./scripts/aws_apply_infra.sh

aws-build-images:
	@REGION="$${REGION:-us-east-1}" PROJECT_NAME="$${PROJECT_NAME:-aegis}" TAG="$${TAG:-latest}" \
		./scripts/aws_build_push_images.sh --region="$$REGION" --project-name="$$PROJECT_NAME" --tag="$$TAG"
