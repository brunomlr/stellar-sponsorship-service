.PHONY: build run test test-integration migrate-up migrate-down docker-up docker-down lint

build:
	go build -o bin/sponsorship-service ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./internal/... -v

test-integration:
	go test ./internal/... -tags=integration -v

migrate-up:
	migrate -path migrations -database $(DATABASE_URL) up

migrate-down:
	migrate -path migrations -database $(DATABASE_URL) down

docker-up:
	docker compose -f docker/docker-compose.yml up --build -d

docker-down:
	docker compose -f docker/docker-compose.yml down

lint:
	golangci-lint run ./...
