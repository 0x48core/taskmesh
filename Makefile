.PHONY: build test migrate wire bench up down

build:   ## TODO: go build ./cmd/taskmesh
test:    ## TODO: go test ./...
migrate: ## TODO: apply services/taskmesh/migrations
wire:    ## TODO: generate services/taskmesh/internal/deps/wire_gen.go
bench:   ## TODO: run k6 scripts in benchmarks/
up:      ## TODO: docker compose -f docker/docker-compose.dev.yaml up
down:    ## TODO: docker compose -f docker/docker-compose.dev.yaml down
