import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, api } from "../api/client";
import { lastDisplayName, saveSession } from "../api/session";
import { DECK } from "../api/types";
import { Header } from "../components/Header";

const NAME_LIMIT = 32;
const LABEL_LIMIT = 200;

export function CreateGamePage() {
  const navigate = useNavigate();
  const [displayName, setDisplayName] = useState(lastDisplayName);
  const [gameName, setGameName] = useState("");
  const [jiraKey, setJiraKey] = useState("");
  const [advanced, setAdvanced] = useState(false);
  const [joinCode, setJoinCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(undefined);
    try {
      const session = await api.createRoom(displayName.trim());
      const code = session.room.code;
      saveSession(code, {
        participantId: session.participantId,
        sessionToken: session.sessionToken,
        displayName: displayName.trim(),
      });
      // The backend has no room name: the game's name becomes the first round.
      if (gameName.trim() || jiraKey.trim()) {
        await api.startRound(code, session.sessionToken, { label: gameName.trim(), jiraKey: jiraKey.trim() || undefined });
      }
      navigate(`/game/${code}`);
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : "Something went wrong. Try again.");
      setBusy(false);
    }
  };

  const join = (event: FormEvent) => {
    event.preventDefault();
    const code = joinCode.trim().toUpperCase();
    if (code) navigate(`/game/${code}`);
  };

  return (
    <>
      <Header title="Create game" />
      <main className="centered">
        <div className="form-stack">
          <form className="panel form-panel" onSubmit={submit}>
            <label className="field">
              <span>Your name</span>
              <input
                autoFocus={!displayName}
                required
                maxLength={NAME_LIMIT}
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="How teammates will see you"
              />
            </label>

            <label className="field">
              <span>Game's name</span>
              <input
                autoFocus={!!displayName}
                maxLength={LABEL_LIMIT}
                value={gameName}
                onChange={(e) => setGameName(e.target.value)}
                placeholder="e.g. Team Velocity - Sprint 2"
              />
            </label>

            <label className="field">
              <span>Deck</span>
              <select defaultValue="fibonacci">
                <option value="fibonacci">Fibonacci sequence ({DECK.join(", ")})</option>
              </select>
            </label>

            <button type="button" className="link-button" onClick={() => setAdvanced((v) => !v)}>
              {advanced ? "Hide advanced settings" : "Show advanced settings..."}
            </button>

            {advanced && (
              <label className="field">
                <span>Jira issue for the first round</span>
                <input
                  value={jiraKey}
                  maxLength={32}
                  onChange={(e) => setJiraKey(e.target.value.toUpperCase())}
                  placeholder="ABC-123"
                  pattern="[A-Z][A-Z0-9_]+-[1-9][0-9]*"
                  title="Looks like ABC-123"
                />
              </label>
            )}

            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}

            <button type="submit" className="button button-accent button-large form-submit" disabled={busy || !displayName.trim()}>
              {busy ? "Creating..." : "Create game"}
            </button>
          </form>

          <form className="join-inline" onSubmit={join}>
            <span>Have a code?</span>
            <input
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
              placeholder="ABC234"
              maxLength={6}
              aria-label="Room code"
            />
            <button type="submit" className="button button-ghost button-small" disabled={joinCode.trim().length !== 6}>
              Join
            </button>
          </form>
        </div>
      </main>
    </>
  );
}
