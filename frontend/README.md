# Zenith Poker frontend

React + TypeScript single-page app for the planning poker rooms served by [`../backend`](../backend).

## Run locally

Requires Node 22+ and the backend running (`make up` in `../backend`, API on `:8090`).

```sh
npm install
npm run dev      # http://localhost:5173, proxies /api to http://localhost:8090
```

Set `API_TARGET` to proxy somewhere else, for example `API_TARGET=http://localhost:8080 npm run dev`.

## Development

| Command | Purpose |
|---|---|
| `npm run dev` | Dev server with hot reload |
| `npm run build` | Type-check and build into `dist/` |
| `npm run lint` | ESLint |
| `npm run typecheck` | TypeScript only |
| `npm test` | Unit tests (Vitest) |

## Docker

`make up` in `../backend` also builds this app into an nginx image and serves it on
<http://localhost:8091>. nginx proxies `/api` (including the WebSocket stream) to the
backend, so the browser only talks to one origin.
