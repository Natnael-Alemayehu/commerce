.PHONY: all run build test deps migrate-up migrate-down sqlc-generate swagger

all: build

deps:
	go mod tidy
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/swaggo/swag/cmd/swag@latest

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:
	go test -v -race ./internal/auth/...

test-integration:
	go test -v ./internal/integration/...

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

sqlc-generate:
	sqlc generate

swagger:
	swag init -g cmd/api/main.go -o docs
