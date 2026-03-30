.PHONY: build run test test-v lint docker-up docker-down migrate seed clean

# Build
build:
	go build -o bin/wpp-gateway ./cmd/server
	go build -o bin/seed ./cmd/seed

# Run
run:
	go run ./cmd/server

# Tests
test:
	go test ./... -count=1

test-v:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Docker
docker-up:
	cd docker && docker compose up -d

docker-down:
	cd docker && docker compose down

docker-build:
	cd docker && docker compose build

docker-logs:
	cd docker && docker compose logs -f api

# Database
migrate:
	go run ./cmd/server

seed:
	go run ./cmd/seed

# Lint
lint:
	golangci-lint run ./...

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html tmp/

# Dev: start deps only (Postgres + Redis)
dev-deps:
	cd docker && docker compose up -d postgres redis

# Dev: run with hot reload (requires air)
dev: dev-deps
	air

# Tidy
tidy:
	go mod tidy
	go mod verify
