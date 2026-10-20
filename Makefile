-include .env
export

NETWORK := $(notdir $(CURDIR))_default
SSLMODE := $(or $(POSTGRES_SSLMODE),disable)
DEV_DB_PORT ?= 5433
LINT_IMAGE := golangci/golangci-lint:v2.14.0
LINT := docker run --rm -v "$(CURDIR)":/app -v golangci-lint-cache:/root/.cache -w /app $(LINT_IMAGE) golangci-lint
TEST_DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=$(SSLMODE)
COMPOSE_DEV := docker compose -f docker-compose.yml -f docker-compose.dev.yml

.PHONY: help up down logs build run fmt lint test test-integration

help:
	@echo "Команды:"
	@echo "  make up                Собрать и запустить сервис в Docker"
	@echo "  make down              Остановить сервис"
	@echo "  make logs              Смотреть логи приложения"
	@echo "  make build             Собрать бинарник в bin/app"
	@echo "  make run               Запустить приложение локально, база в Docker"
	@echo "  make fmt               Отформатировать код"
	@echo "  make lint              Запустить линтеры"
	@echo "  make test              Юнит-тесты"
	@echo "  make test-integration  Тесты с настоящим Postgres в Docker"

up:
	docker compose up -d --build --wait

down:
	docker compose down

logs:
	docker compose logs -f app

build:
	go build -o bin/app ./cmd/app

run:
	docker compose stop app
	$(COMPOSE_DEV) up -d --wait postgres
	$(COMPOSE_DEV) run --rm migrate
	POSTGRES_HOST=localhost POSTGRES_PORT=$(DEV_DB_PORT) go run ./cmd/app

fmt:
	$(LINT) fmt

lint:
	$(LINT) run ./...

test:
	go test -race ./...

test-integration:
	docker compose up -d --wait postgres
	docker compose run --rm migrate
	@docker run --rm --network $(NETWORK) -v "$(CURDIR)":/src -w /src \
		-e TEST_DATABASE_URL="$(TEST_DB_URL)" \
		golang:1.27 go test -race -count=1 ./...
