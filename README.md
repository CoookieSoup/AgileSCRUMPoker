# AgileSCRUMPoker

Zenith Poker: short-lived planning poker rooms for Scrum teams.

| Part | Stack | Docs |
|---|---|---|
| [`backend`](backend) | Go, MySQL, WebSocket | [backend/README.md](backend/README.md) |
| [`frontend`](frontend) | React, TypeScript, Vite | [frontend/README.md](frontend/README.md) |

## Quick start

```sh
cd backend && make up    # MySQL, API on :8090, web app on http://localhost:8091
```

For frontend development with hot reload, keep the stack running and run `npm install && npm run dev`
in `frontend` (http://localhost:5173).
