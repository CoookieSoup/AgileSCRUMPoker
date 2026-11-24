import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ApiError } from "../api/client";
import { clearSession, type StoredSession } from "../api/session";
import type { StartRoundRequest } from "../api/types";
import { Avatar } from "../components/Avatar";
import { LogoMark } from "../components/Logo";
import { useRoom } from "../room/useRoom";
import { CardDeck } from "./CardDeck";
import { Countdown } from "./Countdown";
import { PokerTable } from "./PokerTable";
import { RoundCenter } from "./RoundCenter";

interface GameRoomProps {
  code: string;
  session: StoredSession;
  onSessionLost: () => void;
}

export function GameRoom({ code, session, onSessionLost }: GameRoomProps) {
  const navigate = useNavigate();
  const { status, room, endedReason, vote, startRound, reveal, leave, close } = useRoom(code, session);
  const [error, setError] = useState<string>();
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (status === "unauthorized") {
      clearSession(code);
      onSessionLost();
    }
  }, [status, code, onSessionLost]);

  useEffect(() => {
    if (status === "ended") clearSession(code);
  }, [status, code]);

  const guard =
    <A extends unknown[]>(action: (...args: A) => Promise<unknown>) =>
    async (...args: A) => {
      setError(undefined);
      try {
        await action(...args);
      } catch (caught) {
        setError(caught instanceof ApiError ? caught.message : "Something went wrong.");
        throw caught;
      }
    };

  if (status === "ended") {
    return (
      <main className="centered">
        <div className="panel message-panel">
          <h2>This game has ended</h2>
          <p>{endedReason === "expired" ? "Rooms expire one hour after they are created." : "The host closed the room."}</p>
          <Link to="/new" className="button button-accent">
            Start a new game
          </Link>
        </div>
      </main>
    );
  }

  if (!room) {
    return (
      <main className="centered">
        <p className="center-hint">{status === "reconnecting" ? "Reconnecting…" : "Joining the table…"}</p>
      </main>
    );
  }

  const self = room.participants.find((p) => p.id === session.participantId);
  const isHost = !!self?.isHost;
  const canVote = !!room.round && !room.round.revealed;

  const invite = async () => {
    try {
      await navigator.clipboard.writeText(`${window.location.origin}/game/${code}`);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError(`Share this code: ${code}`);
    }
  };

  const exit = async () => {
    if (isHost && !window.confirm("Leave the game? The next player becomes the host.")) return;
    await leave().catch(() => undefined);
    clearSession(code);
    navigate("/");
  };

  const endGame = async () => {
    if (!window.confirm("Close the game for everyone?")) return;
    await guard(close)().catch(() => undefined);
  };

  const title = room.round?.label || `Game ${code}`;

  return (
    <div className="game">
      <header className="game-bar">
        <Link to="/" className="game-title" title="Home">
          <LogoMark size={32} />
        </Link>
        <h1 className="game-title-text" title={title}>
          {title}
          {room.round?.jiraKey && <span className="jira-tag">{room.round.jiraKey}</span>}
        </h1>
        <div className="game-bar-actions">
          <div className="avatar-stack" title={room.participants.map((p) => p.displayName).join(", ")}>
            {room.participants.slice(0, 5).map((p) => (
              <span key={p.id} className="avatar-stack-item">
                <Avatar name={p.displayName} seed={p.id} size={32} />
                <span className="online-dot" />
              </span>
            ))}
            {room.participants.length > 5 && <span className="avatar-more">+{room.participants.length - 5}</span>}
          </div>
          <Countdown until={room.expiresAt} />
          <button className="button button-outline" onClick={invite}>
            {copied ? "Link copied!" : "Invite players"}
          </button>
          <button className="button button-ghost button-small" onClick={exit}>
            Leave
          </button>
          {isHost && (
            <button className="button button-danger button-small" onClick={endGame}>
              End game
            </button>
          )}
        </div>
      </header>

      {status === "reconnecting" && <div className="banner">Connection lost. Reconnecting…</div>}
      {error && (
        <div className="banner banner-error" role="alert">
          {error}
          <button className="link-button" onClick={() => setError(undefined)}>
            Dismiss
          </button>
        </div>
      )}

      <main className="game-main">
        <PokerTable participants={room.participants} selfId={session.participantId} round={room.round} ownVote={room.ownVote}>
          <RoundCenter
            round={room.round}
            participants={room.participants}
            isHost={isHost}
            onStart={guard((request: StartRoundRequest) => startRound(request))}
            onReveal={guard(reveal)}
          />
        </PokerTable>

        <div className="deck-area">
          <p className="deck-hint">
            {!room.round
              ? "Cards unlock when the host starts a round."
              : room.round.revealed
                ? "Cards are revealed."
                : room.ownVote
                  ? "You can change your card until the host reveals."
                  : "Pick your card"}
          </p>
          <CardDeck
            selected={room.ownVote}
            disabled={!canVote}
            onPick={(card) => void guard(vote)(card).catch(() => undefined)}
          />
        </div>
      </main>
    </div>
  );
}
