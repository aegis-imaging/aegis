.PHONY: up down clean build api lint lint-go lint-python lint-frontend check logs test test-unit test-race

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

lint: lint-go lint-python lint-frontend

lint-go:
	cd api && go vet ./...

lint-python:
	@for svc in defacing phi-detection qc-service bids-service classification-service; do \
		echo "Checking $$svc..."; \
		find $$svc -name '*.py' -exec python3 -m py_compile {} +; \
	done

lint-frontend:
	cd client && npx tsc --noEmit
	cd frontend/upload-portal && npx tsc --noEmit
	cd frontend/admin-dashboard && npx tsc --noEmit

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
