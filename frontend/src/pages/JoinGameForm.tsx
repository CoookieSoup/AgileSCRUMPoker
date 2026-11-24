import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { ApiError, api } from "../api/client";
import { lastDisplayName, saveSession, type StoredSession } from "../api/session";
import { Header } from "../components/Header";

const MESSAGES: Record<string, string> = {
  room_not_found: "This game doesn't exist or has already ended.",
  room_full: "This game is full (20 players).",
};

export function JoinGameForm({ code, onJoined }: { code: string; onJoined: (session: StoredSession) => void }) {
  const [displayName, setDisplayName] = useState(lastDisplayName);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError>();

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      const name = displayName.trim();
      const session = await api.joinRoom(code, name);
      const stored = { participantId: session.participantId, sessionToken: session.sessionToken, displayName: name };
      saveSession(session.room.code, stored);
      onJoined(stored);
    } catch (caught) {
      setError(caught instanceof ApiError ? caught : new ApiError(0, "unknown", "Something went wrong. Try again."));
      setBusy(false);
    }
  };

  const gone = error?.code === "room_not_found";

  return (
    <>
      <Header title="Join game" />
      <main className="centered">
        <form className="panel form-panel" onSubmit={submit}>
          <h2 className="form-title">
            Join game <span className="room-code">{code}</span>
          </h2>
          <label className="field">
            <span>Your name</span>
            <input
              autoFocus
              required
              maxLength={32}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="How teammates will see you"
            />
          </label>
          {error && (
            <p className="form-error" role="alert">
              {MESSAGES[error.code] ?? error.message}
            </p>
          )}
          {gone ? (
            <Link to="/new" className="button button-accent button-large form-submit">
              Start a new game
            </Link>
          ) : (
            <button type="submit" className="button button-accent button-large form-submit" disabled={busy || !displayName.trim()}>
              {busy ? "Joining..." : "Join game"}
            </button>
          )}
        </form>
      </main>
    </>
  );
}
