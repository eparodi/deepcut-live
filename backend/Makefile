.PHONY: lint build test-unit test-integration test migrate-up sqlc-check

DATABASE_URL ?= postgres://live:live@localhost:5432/live?sslmode=disable
MIGRATIONS_DIR ?= db/migrations

# ── Lint ──────────────────────────────────────────────────────────────
lint:
	go vet ./...

# ── Build ─────────────────────────────────────────────────────────────
build:
	go build ./...

# ── Unit Tests (no DB) ────────────────────────────────────────────────
test-unit:
	go test -short -count=1 ./...

# ── Integration Tests (requires DATABASE_URL) ─────────────────────────
test-integration:
	@echo "Running migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up
	@echo "Running integration tests..."
	go test -run Integration -count=1 ./...

# ── All Tests ─────────────────────────────────────────────────────────
test: test-unit test-integration

# ── Run Migrations Up ─────────────────────────────────────────────────
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

# ── sqlc Generate + Diff Check ────────────────────────────────────────
sqlc-check:
	sqlc generate -f db/sqlc.yaml
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: sqlc generated code is out of date. Please commit the changes."; \
		git diff; \
		exit 1; \
	else \
		echo "sqlc generated code is up to date."; \
	fi
