DATABASE_URL ?= postgres://minimarket:minimarket@localhost:5433/minimarket?sslmode=disable
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@latest

.PHONY: db-up db-down migrate-up migrate-down run run-buysell test vet docker-up docker-down docker-logs docker-build

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate-up:
	$(GOOSE) -dir auth-app/migrations postgres "$(DATABASE_URL)" up
	$(GOOSE) -dir buySell-app/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) -dir buySell-app/migrations postgres "$(DATABASE_URL)" down
	$(GOOSE) -dir auth-app/migrations postgres "$(DATABASE_URL)" down

run:
	go run ./auth-app/cmd/app

run-buysell:
	HTTP_PORT=8081 go run ./buySell-app/cmd/app

test:
	go test ./...

vet:
	go vet ./...

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-build:
	docker compose build
