import { useCallback, useState } from "react";
import { useParams } from "react-router-dom";
import { loadSession, type StoredSession } from "../api/session";
import { GameRoom } from "../game/GameRoom";
import { JoinGameForm } from "./JoinGameForm";

export function GamePage() {
  const code = (useParams().code ?? "").toUpperCase();
  const [session, setSession] = useState<StoredSession | undefined>(() => loadSession(code));
  const dropSession = useCallback(() => setSession(undefined), []);

  if (!session) return <JoinGameForm key={code} code={code} onJoined={setSession} />;

  return <GameRoom key={code} code={code} session={session} onSessionLost={dropSession} />;
}
