import type { RoomEvent, RoomState } from "../api/types";

/**
 * Applies a realtime event to the room state. Every case is idempotent, so
 * events buffered while the initial state was loading can be replayed safely.
 */
export function applyEvent(room: RoomState, event: RoomEvent): RoomState {
  switch (event.type) {
    case "participant_joined":
      if (room.participants.some((p) => p.id === event.payload.id)) return room;
      return { ...room, participants: [...room.participants, event.payload] };

    case "participant_left":
      return {
        ...room,
        participants: room.participants.filter((p) => p.id !== event.payload.participantId),
      };

    case "host_changed":
      return {
        ...room,
        participants: room.participants.map((p) => ({ ...p, isHost: p.id === event.payload.participantId })),
      };

    case "round_started":
      return {
        ...room,
        ownVote: undefined,
        round: { label: event.payload.label, jiraKey: event.payload.jiraKey, revealed: false, votes: [] },
        participants: room.participants.map((p) => ({ ...p, hasVoted: false })),
      };

    case "participant_voted":
      return {
        ...room,
        participants: room.participants.map((p) =>
          p.id === event.payload.participantId ? { ...p, hasVoted: true } : p,
        ),
      };

    case "votes_revealed":
      if (!room.round) return room;
      return { ...room, round: { ...room.round, revealed: true, votes: event.payload.votes } };

    case "room_closed":
      return room;
  }
}

export interface VoteSummary {
  average?: number;
  consensus: boolean;
  /** Card and vote count, most common first. */
  distribution: [string, number][];
}

export function summarizeVotes(votes: { card: string }[]): VoteSummary {
  const counts = new Map<string, number>();
  for (const vote of votes) counts.set(vote.card, (counts.get(vote.card) ?? 0) + 1);

  const numbers = votes.map((v) => Number(v.card)).filter((n) => Number.isFinite(n));
  const average = numbers.length ? numbers.reduce((a, b) => a + b, 0) / numbers.length : undefined;

  return {
    average,
    consensus: votes.length > 0 && counts.size === 1,
    distribution: [...counts].sort((a, b) => b[1] - a[1] || Number(a[0]) - Number(b[0])),
  };
}
