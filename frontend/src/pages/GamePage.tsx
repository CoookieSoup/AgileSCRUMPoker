import { useState } from "react";
import { useParams } from "react-router-dom";
import { loadSession, type StoredSession } from "../api/session";
import { Header } from "../components/Header";
import { JoinGameForm } from "./JoinGameForm";

export function GamePage() {
  const code = (useParams().code ?? "").toUpperCase();
  const [session, setSession] = useState<StoredSession | undefined>(() => loadSession(code));

  if (!session) return <JoinGameForm key={code} code={code} onJoined={setSession} />;

  return (
    <>
      <Header title={`Game ${code}`} />
      <main className="centered">
        <div className="panel message-panel">
          <h2>You're in game {code}</h2>
          <p>Playing as {session.displayName}.</p>
        </div>
      </main>
    </>
  );
}
