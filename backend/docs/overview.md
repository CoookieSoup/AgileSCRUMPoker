# Backend overview

This document describes the backend's behaviour: what each endpoint does and the rules it enforces. For exact request and response schemas see [`api/openapi.yaml`](../api/openapi.yaml); for the WebSocket protocol see [`realtime.md`](realtime.md).

## Core concepts

| Concept | What it is |
|---|---|
| **Room** | A temporary poker session, identified by a 6-character invite code. It expires one hour after creation and holds at most 20 participants. |
| **Participant** | A person in a room: a display name, a host flag, and a session token. There are no accounts; a participant exists only inside one room. |
| **Host** | The participant who controls the room: starts rounds, reveals votes, closes the room. There is exactly one per room. |
| **Round** | The task currently being estimated. A room has at most one round; starting a new one replaces it. |
| **Vote** | One card per participant per round, from the deck `1, 2, 3, 5, 8, 13, 21`. Hidden from others until the host reveals. |
| **Session token** | A random 256-bit secret returned once when you create or join a room. It is sent as `Authorization: Bearer <token>`. The server stores only its SHA-256 hash. |

## How a request flows

```
HTTP request
  → chi router: panic recovery, CORS
  → generated wrapper (internal/api/generated): parses path parameters and marks secured operations
  → auth middleware (internal/api/middleware): on secured operations only, resolves the token to a participant
  → generated strict handler: decodes the JSON body (malformed → 400 invalid_request)
  → controller (internal/api/<area>): turns generated types into service calls
  → service (internal/service/<area>): validates input and applies the business rules
  → store (internal/infra/mysql): reads and writes MySQL, using a locked transaction where rules must hold under concurrency
  → service publishes events to the room's WebSocket subscribers, only after the write has committed
  → controller maps the result to the response; any error goes through one table (internal/api/respond)
```

Each layer depends only on interfaces it declares itself. Controllers declare the service methods they need, and services declare the store, publisher, generator and clock methods they need. Concrete types are assembled only in `internal/app`.

## Authentication

Secured endpoints need `Authorization: Bearer <sessionToken>`. They resolve the token as follows (`internal/service/session`):

1. Normalise the room code from the path: trim, uppercase, check the length and alphabet. A malformed code returns **404 `room_not_found`**.
2. Missing token → **401 `unauthorized`**.
3. Look the participant up by the SHA-256 hash of the token. Not found → **401** (unknown token, or the participant left or the room closed).
4. The token belongs to a different room than the one in the path → **401**.
5. The room has expired → **404 `room_not_found`**.

The resolved participant, including the current host flag, is then available to the endpoint.

## Validation rules

All validation happens in `internal/model`, before anything is written.

| Field | Rule | Error |
|---|---|---|
| `displayName` | Trimmed; 1–32 characters; printable only (Lithuanian letters are fine). Duplicates allowed. | 422 `invalid_display_name` |
| room `code` | Trimmed and uppercased; 6 characters from `23456789ABCDEFGHJKLMNPQRSTUVWXYZ` (no `0 O 1 I`) | 404 `room_not_found` |
| `card` | One of `1 2 3 5 8 13 21`, sent as a string | 422 `invalid_card` |
| `label` | Optional; trimmed; at most 200 printable characters | 422 `invalid_label` |
| `jiraKey` | Optional; trimmed and uppercased; must look like `ABC-123` (`^[A-Z][A-Z0-9_]+-[1-9][0-9]*$`); at most 32 characters | 422 `invalid_jira_key` |

A malformed code returns not found rather than a validation error, so the response doesn't distinguish a badly formed code from one that doesn't exist.

---

## Endpoints

All paths are under `/api/v1`.

### `GET /healthz` — health check

- **Auth:** none.
- **Does:** pings the database.
- **Returns:** `200 {"status":"ok"}`, or **503 `unavailable`** if the database can't be reached.

### `POST /rooms` — create a room

- **Auth:** none. Body: `{"displayName": "..."}`.
- **Business logic:**
  1. Validate the display name.
  2. Create the caller as a participant with **`isHost = true`** and generate their session token.
  3. Generate a random room code and save the room with `expiresAt = now + 1 hour`, together with the host.
  4. If the code is already in use, generate a new one and retry, up to 5 times. In practice this never happens; if all 5 attempts collide, the response is 500.
- **Events:** none, because nobody else is in the room yet.
- **Returns:** **201** `Session`: `participantId`, `sessionToken`, and the room state. This is the only time the token is returned; the client must keep it.
- **Errors:** 400 malformed body, 422 `invalid_display_name`.

### `POST /rooms/{code}/participants` — join a room

- **Auth:** none. Body: `{"displayName": "..."}`.
- **Business logic:**
  1. Normalise the code; validate the display name.
  2. In one transaction, **lock the room row**, then:
     - room missing or expired → 404
     - room already has 20 participants → 409 `room_full`
     - otherwise insert the new participant (`isHost = false`)

     The lock makes the limit exact even when many people join at the same moment.
- **Events:** `participant_joined`, with the new participant's id and name, to everyone in the room.
- **Returns:** **201** `Session` with the new participant's token and the current room state.
- **Errors:** 400, 404 `room_not_found`, 409 `room_full`, 422 `invalid_display_name`.

### `GET /rooms/{code}` — room state

- **Auth:** participant of this room.
- **Business logic:** loads the room, its participants ordered by join time, the current round and its votes. It then builds the **state as seen by the caller** (`model.NewRoomView`):
  - each participant's `hasVoted` flag is visible to everyone
  - `round.votes` is **empty until the round is revealed**, so nobody can see other people's cards early
  - `ownVote` is the caller's own card, so a reloaded page can restore their selection
- **Returns:** **200** `RoomState`.
- **Errors:** 401, 404.

Clients should call this after the WebSocket sends `ready`, and again whenever they reconnect.

### `DELETE /rooms/{code}/participants/me` — leave a room

- **Auth:** participant of this room.
- **Business logic** (one transaction with the room row locked):
  1. Delete the participant and their vote in the current round.
  2. If **nobody is left**, delete the room.
  3. Otherwise, if the leaving participant was the **host**, the **earliest-joined remaining participant becomes host**.
- **Events:**
  - room deleted → `room_closed` (`reason: closed`), and every stream for the room ends
  - otherwise `participant_left`, followed by `host_changed` if the host role moved
- **Returns:** **204**. The caller's token stops working.
- **Errors:** 401, 404.

### `DELETE /rooms/{code}` — close a room

- **Auth:** participant of this room; **host only** (otherwise 403 `not_host`).
- **Business logic:** deletes the room. Database foreign keys cascade, so all participants, the round and the votes are removed in the same statement. Nothing about the room remains.
- **Events:** `room_closed` (`reason: closed`); every stream for the room ends.
- **Returns:** **204**. Afterwards every token for the room returns 401, and joining the code returns 404.
- **Errors:** 401, 403 `not_host`, 404.

### `POST /rooms/{code}/round` — start a round

- **Auth:** participant; **host only**.
- **Body:** `{"label": "...", "jiraKey": "ABC-123"}`. Both fields are optional.
- **Business logic:**
  1. Check the caller is host, then validate the label and Jira key.
  2. In one transaction with the room row locked: **delete the current round and all its votes**, then insert the new round with `revealed = false`.

  Only the current round is kept; there is no history. The Jira key is only validated and stored: Jira itself is not contacted yet.
- **Events:** `round_started` (label and Jira key).
- **Returns:** **201** room state, with the new round and no votes.
- **Errors:** 401, 403 `not_host`, 404, 422 `invalid_label` or `invalid_jira_key`.

### `PUT /rooms/{code}/round/vote` — cast or change a vote

- **Auth:** any participant of the room, **including the host**. Body: `{"card": "8"}`.
- **Business logic:**
  1. Validate the card against the deck.
  2. In one transaction, **lock the current round row**, then:
     - no round → 404 `round_not_found`
     - round already revealed → 409 `round_revealed`
     - otherwise **insert or replace** the participant's vote

     Participants can change their card any number of times before the reveal. The lock means a vote can't slip in while a reveal is in progress.
- **Events:** `participant_voted` with **only the participant's id, never the card**.
- **Returns:** **204**.
- **Errors:** 401, 404 `round_not_found`, 409 `round_revealed`, 422 `invalid_card`.

### `POST /rooms/{code}/round/reveal` — reveal votes

- **Auth:** participant; **host only**.
- **Business logic:** locks the current round and marks it revealed. Calling it again has no further effect and is not an error. Once revealed, the round accepts no more votes until the host starts a new one.
- **Events:** `votes_revealed` with every vote (`participantId`, `card`), in participant join order. It is sent on every call, including repeats.
- **Returns:** **200** room state, with `round.votes` now filled in.
- **Errors:** 401, 403 `not_host`, 404 `room_not_found` or `round_not_found`.

### `GET /rooms/{code}/events` — live updates (WebSocket)

- **Auth:** the first WebSocket message is `{"type":"auth","token":"..."}`, sent within 5 seconds. The token goes in a message rather than the URL so it never appears in access logs.
- **Does:** the server replies `{"type":"ready"}` once subscribed, then pushes every event for the room listed above.
- **Ends:** when the room closes or expires (`room_closed`, then a normal close), or if the client falls more than 32 events behind (it should reconnect and fetch the state again).
- Close codes and payloads are in [`realtime.md`](realtime.md).

---

## Background cleanup

`internal/service/cleanup` runs every `CLEANUP_INTERVAL` (default 1 minute):

1. In one transaction, find every room whose `expiresAt` has passed and delete it. The cascade removes all of its data.
2. For each deleted room, send `room_closed` (`reason: expired`) and end its streams.

A room disappears from the API as soon as it expires, even before the sweep runs: every read filters out expired rooms, and authentication returns 404 for them. A room's lifetime is never extended by activity.

## Errors

Every error has the body `{"code": "...", "message": "..."}`. Clients should use `code`, which is stable, to choose their (Lithuanian) message.

| Status | `code` | Meaning |
|---|---|---|
| 400 | `invalid_request` | Malformed JSON or path |
| 401 | `unauthorized` | Missing or invalid token, a token for another room, or a participant who has already left |
| 403 | `not_host` | The action is host only |
| 404 | `room_not_found` | Code is malformed, doesn't exist, or the room is closed or expired |
| 404 | `round_not_found` | The room has no round yet |
| 409 | `room_full` | The room already has 20 participants |
| 409 | `round_revealed` | Votes are already revealed |
| 422 | `invalid_display_name`, `invalid_card`, `invalid_label`, `invalid_jira_key` | Validation failed |
| 503 | `unavailable` | Database unreachable (health check) |
| 500 | `internal_error` | Anything unexpected. The details are logged on the server, never returned to the client. |

## Data and privacy

- **Stored:** rooms, participants (display name, token hash, host flag, join time), the current round, and votes. Nothing else.
- **Deleted:** all of it is hard-deleted when the room closes, empties or expires. No user data outlives the room (requirement 2.1).
- **Not stored:** raw session tokens; only their SHA-256 hashes are kept.
- **Not logged:** display names, tokens and IP addresses. Request logging is deliberately left out.

## Concurrency and scaling

- Rules that read and then write (the participant limit, vote vs reveal, host transfer, deleting an empty room) run inside a single database transaction that holds a row lock. They therefore hold under concurrent requests.
- Events are published only after the database write succeeds, so clients never hear about a change that was rolled back.
- The WebSocket broker lives **in the server's memory**. The backend therefore runs as a **single instance**: with several instances, clients connected to one would miss events from the others.

## Known limitations

- No rate limiting on joins or room creation.
- No online/offline presence: a dropped connection does not remove the participant. They stay listed until they leave or the room ends.
- Jira keys are validated and displayed but Jira is never contacted.
- Single-instance only (see above).

## Where things live

| Area | Controller | Service | Store |
|---|---|---|---|
| Health | `internal/api/health` | — | `internal/infra/mysql/database.go` |
| Create, join, get, leave, close | `internal/api/room` | `internal/service/room` | `internal/infra/mysql/room_store.go` |
| Start, vote, reveal | `internal/api/round` | `internal/service/round` | `internal/infra/mysql/round_store.go` |
| Authentication | `internal/api/middleware` | `internal/service/session` | `internal/infra/mysql/session_store.go` |
| WebSocket | `internal/api/realtime` | — | `internal/infra/realtime` (broker) |
| Expiry sweep | — | `internal/service/cleanup` | `internal/infra/mysql/cleanup_store.go` |

Validation, the per-viewer room view and event types live in `internal/model`. Error-to-status mapping lives in `internal/api/respond/errors.go`. Everything is wired together in `internal/app/app.go`.
