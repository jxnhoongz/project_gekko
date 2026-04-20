.PHONY: help db-up db-down db-logs migrate migrate-status migrate-down sqlc dev-backend dev-admin dev-storefront seed test test-backend test-admin

help:  ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

db-up:  ## Start local Postgres
	docker compose -f infra/docker-compose.dev.yml up -d

db-down:  ## Stop local Postgres
	docker compose -f infra/docker-compose.dev.yml down

db-logs:  ## Tail Postgres logs
	docker compose -f infra/docker-compose.dev.yml logs -f postgres

DB_URL ?= postgres://gekko:gekko@localhost:5433/gekko?sslmode=disable

migrate:  ## Run goose migrations up
	cd backend && goose -dir ./migrations postgres "$(DB_URL)" up

migrate-status:  ## Show migration status
	cd backend && goose -dir ./migrations postgres "$(DB_URL)" status

migrate-down:  ## Roll back one migration
	cd backend && goose -dir ./migrations postgres "$(DB_URL)" down

sqlc:  ## Regenerate sqlc Go code
	cd backend && sqlc generate

dev-backend:  ## Start Go backend with hot reload
	cd backend && air

dev-admin:  ## Start admin dev server
	bun run dev:admin

dev-storefront:  ## Start storefront dev server
	bun run dev:storefront

seed:  ## Seed first admin (ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=Zen make seed)
	cd backend && go run ./cmd/gekko-seed \
	  -email "$(ADMIN_EMAIL)" -password "$(ADMIN_PASSWORD)" -name "$(ADMIN_NAME)"

test: test-backend test-admin  ## Run all tests

test-backend:  ## Run Go tests
	cd backend && go test ./...

test-admin:  ## Run admin Vitest tests
	bun --cwd apps/admin run test
