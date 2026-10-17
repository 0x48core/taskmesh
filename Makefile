.PHONY: build test migrate migrate-down wire bench up down

DB_URL      ?= postgres://taskmesh:taskmesh@localhost:5432/taskmesh?sslmode=disable
MIGRATE     ?= migrate
COMPOSE_DEV := docker compose -f docker/docker-compose.dev.yaml

build: ## Build the service binary into bin/
	go build -o bin/taskmesh ./cmd/taskmesh

test: ## Run all tests with the race detector
	go test -race ./...

up: ## Start the local stack and wait until Postgres is healthy
	$(COMPOSE_DEV) up -d --wait

down: ## Stop the local stack (data volume is kept)
	$(COMPOSE_DEV) down

migrate: ## Apply all pending migrations
	$(MIGRATE) -path migrations -database "$(DB_URL)" up

migrate-down: ## Roll back the latest migration
	$(MIGRATE) -path migrations -database "$(DB_URL)" down 1

wire:    ## TODO: generate internal/app/wire_gen.go
bench:   ## TODO: run k6 scripts in benchmarks/