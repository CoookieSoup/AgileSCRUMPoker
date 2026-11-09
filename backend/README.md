# Scrum Poker backend

Go service for short-lived planning poker rooms: no accounts, at most 20 participants, rooms expire after one hour.

- How it works, endpoint by endpoint: [`docs/overview.md`](docs/overview.md)
- REST contract: [`api/openapi.yaml`](api/openapi.yaml)
- Realtime protocol: [`docs/realtime.md`](docs/realtime.md)

## Run locally

Requires Docker and Go 1.26+.

```sh
make up      # MySQL on :3316 and the API on http://localhost:8090
make down    # stop and delete the data
```

To run the server outside Docker against the compose database:

```sh
cp .env.example .env
set -a && . ./.env && set +a && go run ./cmd/server
```

## Development

| Command | Purpose |
|---|---|
| `make generate` | Regenerate `internal/api/generated` from `api/openapi.yaml` |
| `make build` | Build every package |
| `make test` | Unit tests with the race detector |
| `make test-integration` | Unit and integration tests against the compose MySQL (`make up` first) |
| `make lint` | golangci-lint |
| `make vet` | `go vet`, with and without the `integration` tag |

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_DSN` | required | MySQL DSN; must include `parseTime=true&loc=UTC` |
| `HTTP_ADDRESS` | `:8080` | Listen address |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Comma-separated origins for CORS and WebSocket |
| `CLEANUP_INTERVAL` | `1m` | How often expired rooms are deleted |
