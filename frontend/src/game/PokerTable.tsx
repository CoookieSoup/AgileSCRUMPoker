import type { CSSProperties, ReactNode } from "react";
import type { Participant, Round } from "../api/types";
import { Avatar } from "../components/Avatar";

interface PokerTableProps {
  participants: Participant[];
  selfId: string;
  round?: Round;
  ownVote?: string;
  children: ReactNode;
}

/** Spreads the other players over the rim, leaving the bottom-centre seat for you. */
export function seatAngles(count: number): number[] {
  if (count === 0) return [];
  // Degrees, 0 = right, 90 = top. Few players sit across the top; more wrap down the sides.
  const spread = Math.min(300, 140 + count * 20);
  const start = 90 + spread / 2;
  if (count === 1) return [90];
  return Array.from({ length: count }, (_, index) => start - (spread * index) / (count - 1));
}

export function PokerTable({ participants, selfId, round, ownVote, children }: PokerTableProps) {
  const self = participants.find((p) => p.id === selfId);
  const others = participants.filter((p) => p.id !== selfId);
  const angles = seatAngles(others.length);
  const cardFor = (id: string) => round?.votes.find((v) => v.participantId === id)?.card;

  return (
    <div className="table-area">
      <div className="table-felt">{children}</div>
      {others.map((participant, index) => (
        <Seat
          key={participant.id}
          participant={participant}
          angle={angles[index]!}
          revealed={round?.revealed ?? false}
          card={cardFor(participant.id)}
          inRound={!!round}
        />
      ))}
      {self && (
        <Seat
          participant={self}
          angle={270}
          self
          revealed={round?.revealed ?? false}
          card={round?.revealed ? cardFor(self.id) : ownVote}
          inRound={!!round}
        />
      )}
    </div>
  );
}

interface SeatProps {
  participant: Participant;
  angle: number;
  self?: boolean;
  revealed: boolean;
  card?: string;
  inRound: boolean;
}

function Seat({ participant, angle, self, revealed, card, inRound }: SeatProps) {
  const radians = (angle * Math.PI) / 180;
  const style = { "--cos": Math.cos(radians).toFixed(4), "--sin": Math.sin(radians).toFixed(4) } as CSSProperties;

  let cardState = "empty";
  let cardText = "";
  if (revealed) {
    cardState = card ? "revealed" : "empty";
    cardText = card ?? "–";
  } else if (participant.hasVoted) {
    cardState = "voted";
    cardText = self && card ? card : "";
  }

  return (
    <div className={`seat ${self ? "seat-self" : ""}`} style={style}>
      {inRound && <div className={`seat-card seat-card-${cardState}`}>{cardText}</div>}
      <div className="seat-avatar">
        <Avatar name={participant.displayName} seed={participant.id} size={44} />
        {participant.isHost && (
          <span className="host-badge" title="Host">
            ★
          </span>
        )}
      </div>
      <span className="seat-name">{self ? "You" : participant.displayName}</span>
    </div>
  );
}
