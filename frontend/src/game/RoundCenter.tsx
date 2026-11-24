import { useState, type FormEvent } from "react";
import type { Participant, Round, StartRoundRequest } from "../api/types";
import { summarizeVotes } from "../room/roomReducer";

interface RoundCenterProps {
  round?: Round;
  participants: Participant[];
  isHost: boolean;
  onStart: (round: StartRoundRequest) => Promise<void>;
  onReveal: () => Promise<void>;
}

export function RoundCenter({ round, participants, isHost, onStart, onReveal }: RoundCenterProps) {
  const [busy, setBusy] = useState(false);
  const run = (action: () => Promise<void>) => async () => {
    setBusy(true);
    try {
      await action();
    } catch {
      // The room shows the error.
    } finally {
      setBusy(false);
    }
  };

  if (!round) {
    return isHost ? (
      <StartRoundForm onStart={onStart} cta="Start round" />
    ) : (
      <p className="center-hint">Waiting for the host to start a round…</p>
    );
  }

  const voted = participants.filter((p) => p.hasVoted).length;

  if (!round.revealed) {
    return (
      <div className="center-stack">
        <div className="center-card">?</div>
        <p className="center-hint">
          {voted} of {participants.length} voted
        </p>
        {isHost && (
          <button className="button button-accent" disabled={busy || voted === 0} onClick={run(onReveal)}>
            Reveal cards
          </button>
        )}
      </div>
    );
  }

  const summary = summarizeVotes(round.votes);
  return (
    <div className="center-stack">
      {summary.distribution.length === 0 ? (
        <p className="center-hint">Nobody voted this round.</p>
      ) : (
        <>
          <div className="result">
            <span className="result-label">{summary.consensus ? "Consensus!" : "Average"}</span>
            <span className="result-value">
              {summary.consensus ? summary.distribution[0]![0] : formatAverage(summary.average)}
            </span>
          </div>
          <div className="distribution">
            {summary.distribution.map(([card, count]) => (
              <span key={card} className="chip">
                <strong>{card}</strong> × {count}
              </span>
            ))}
          </div>
        </>
      )}
      {isHost && <StartRoundForm onStart={onStart} cta="New round" compact />}
    </div>
  );
}

function formatAverage(value?: number): string {
  return value === undefined ? "–" : String(Math.round(value * 10) / 10);
}

function StartRoundForm({
  onStart,
  cta,
  compact,
}: {
  onStart: (round: StartRoundRequest) => Promise<void>;
  cta: string;
  compact?: boolean;
}) {
  const [label, setLabel] = useState("");
  const [jiraKey, setJiraKey] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    try {
      await onStart({ label: label.trim(), jiraKey: jiraKey.trim() || undefined });
      setLabel("");
      setJiraKey("");
    } catch {
      // Keep the inputs so the host can fix them; the room shows the error.
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className={`round-form ${compact ? "compact" : ""}`} onSubmit={submit}>
      <input value={label} maxLength={200} onChange={(e) => setLabel(e.target.value)} placeholder="What are we estimating?" />
      <input
        className="jira-input"
        value={jiraKey}
        maxLength={32}
        onChange={(e) => setJiraKey(e.target.value.toUpperCase())}
        placeholder="ABC-123"
        aria-label="Jira key (optional)"
      />
      <button className="button button-accent" disabled={busy}>
        {cta}
      </button>
    </form>
  );
}
