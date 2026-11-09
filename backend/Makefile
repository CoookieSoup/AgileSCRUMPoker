.PHONY: generate build test lint vet

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

build:
	go build ./...

test:
	go test -race ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...
