-include .env
export

NETWORK := $(notdir $(CURDIR))_default
SSLMODE := $(or $(POSTGRES_SSLMODE),disable)
TEST_DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=$(SSLMODE)

.PHONY: up down logs fmt lint test test-integration

up:
	docker compose up -d --build --wait

down:
	docker compose down

logs:
	docker compose logs -f app

fmt:
	golangci-lint fmt

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-integration:
	docker compose up -d --wait postgres
	docker compose run --rm migrate
	@docker run --rm --network $(NETWORK) -v "$(CURDIR)":/src -w /src \
		-e TEST_DATABASE_URL="$(TEST_DB_URL)" \
		golang:1.27 go test -race -count=1 ./...
