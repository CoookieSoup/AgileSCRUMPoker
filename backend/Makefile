TEST_DATABASE_DSN ?= poker:poker@tcp(127.0.0.1:3316)/poker?parseTime=true&loc=UTC
export TEST_DATABASE_DSN

.PHONY: generate build test test-integration lint vet up down

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

build:
	go build ./...

test:
	go test -race ./...

test-integration:
	go test -race -count=1 -tags integration ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...
	go vet -tags integration ./...

up:
	docker compose up -d --wait

down:
	docker compose down -v
