APP_BINARY=bin/api
MIGRATE_BINARY=bin/migrate
GO_ENV=GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/gomod

.PHONY: setup up down run build fmt lint test test-unit test-integration test-functional migrate-up migrate-down ensure-go-cache

ensure-go-cache:
	@mkdir -p .cache/go-build .cache/gomod

setup:
	@if [ ! -f .env ]; then cp .env.example .env; fi

up:
	docker compose up --build -d

down:
	docker compose down --remove-orphans

run: ensure-go-cache
	$(GO_ENV) go run ./cmd/api

build: ensure-go-cache
	@mkdir -p bin
	$(GO_ENV) go build -o $(APP_BINARY) ./cmd/api
	$(GO_ENV) go build -o $(MIGRATE_BINARY) ./cmd/migrate

fmt: ensure-go-cache
	$(GO_ENV) go fmt ./...

lint: ensure-go-cache
	$(GO_ENV) go vet ./...

test: ensure-go-cache
	$(GO_ENV) go test ./...

test-unit: ensure-go-cache
	$(GO_ENV) go test ./internal/...

test-integration: ensure-go-cache
	$(GO_ENV) go test ./test/integration/...

test-functional: ensure-go-cache
	$(GO_ENV) go test ./test/functional/...

migrate-up: ensure-go-cache
	$(GO_ENV) go run ./cmd/migrate --direction up

migrate-down: ensure-go-cache
	$(GO_ENV) go run ./cmd/migrate --direction down
