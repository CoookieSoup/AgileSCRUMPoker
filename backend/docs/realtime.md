# Realtime protocol

REST commands change room state. Each change is pushed to everyone in the room over one WebSocket per client.

## Connecting

1. Open `GET /api/v1/rooms/{code}/events` as a WebSocket.
2. Within 5 seconds, send:

   ```json
   {"type": "auth", "token": "<sessionToken>"}
   ```

3. The server replies `{"type": "ready"}` once the connection is subscribed.
4. Fetch `GET /api/v1/rooms/{code}` and apply events from that point on. Events published after `ready` are never missed.

The token is sent in the first message, not the URL, so it never appears in proxy or access logs.

The server ignores any other client messages. It sends a ping every 30 seconds.

## Events

Every event is an envelope:

```json
{"type": "<event type>", "payload": { ... }}
```

| Type | Payload | Sent when |
|---|---|---|
| `participant_joined` | `{"id", "displayName", "isHost", "hasVoted"}` | someone joins |
| `participant_left` | `{"participantId"}` | someone leaves |
| `host_changed` | `{"participantId"}` | the host left and the earliest remaining participant took over |
| `round_started` | `{"label", "jiraKey"?}` | the host starts a round; all votes are cleared |
| `participant_voted` | `{"participantId"}` | someone votes or changes their vote; **the card is never included** |
| `votes_revealed` | `{"votes": [{"participantId", "card"}]}` | the host reveals |
| `room_closed` | `{"reason": "closed" \| "expired"}` | the room is gone; the stream ends right after |

## Close codes

| Code | Meaning | Client action |
|---|---|---|
| `1000` | Stream ended: the room closed, or this client fell behind | Reconnect; if the room is gone you get `4404` |
| `4401` | Missing, invalid or wrong-room token, or a bad first message | Return to the join screen |
| `4404` | Room closed or expired | Show that the room has ended |

A client that falls more than 32 events behind is disconnected rather than slowing the room down. It should reconnect and fetch the state again.
