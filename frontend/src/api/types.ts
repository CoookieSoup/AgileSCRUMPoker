// Mirrors backend/api/openapi.yaml and backend/docs/realtime.md.

export const DECK = ["1", "2", "3", "5", "8", "13", "21"] as const;
export type Card = (typeof DECK)[number];

export interface Participant {
  id: string;
  displayName: string;
  isHost: boolean;
  hasVoted: boolean;
}

export interface Vote {
  participantId: string;
  card: string;
}

export interface Round {
  label: string;
  jiraKey?: string;
  revealed: boolean;
  votes: Vote[];
}

export interface RoomState {
  code: string;
  expiresAt: string;
  participants: Participant[];
  round?: Round;
  ownVote?: string;
}

export interface Session {
  participantId: string;
  sessionToken: string;
  room: RoomState;
}

export interface StartRoundRequest {
  label?: string;
  jiraKey?: string;
}

export type CloseReason = "closed" | "expired";

export type RoomEvent =
  | { type: "participant_joined"; payload: Participant }
  | { type: "participant_left"; payload: { participantId: string } }
  | { type: "host_changed"; payload: { participantId: string } }
  | { type: "round_started"; payload: { label: string; jiraKey?: string } }
  | { type: "participant_voted"; payload: { participantId: string } }
  | { type: "votes_revealed"; payload: { votes: Vote[] } }
  | { type: "room_closed"; payload: { reason: CloseReason } };
