.PHONY: up down clean build api lint lint-go lint-python lint-frontend check logs test test-unit test-race smoke dimse-e2e

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
