import { describe, expect, it } from "vitest";
import type { RoomState } from "../api/types";
import { applyEvent, summarizeVotes } from "./roomReducer";

const base: RoomState = {
  code: "ABC234",
  expiresAt: "2026-01-01T00:00:00Z",
  participants: [
    { id: "h", displayName: "Host", isHost: true, hasVoted: true },
    { id: "g", displayName: "Guest", isHost: false, hasVoted: false },
  ],
  round: { label: "Login", revealed: false, votes: [] },
  ownVote: "5",
};

describe("applyEvent", () => {
  it("adds a joined participant once", () => {
    const joined = { id: "n", displayName: "New", isHost: false, hasVoted: false };
    const once = applyEvent(base, { type: "participant_joined", payload: joined });
    const twice = applyEvent(once, { type: "participant_joined", payload: joined });
    expect(twice.participants.map((p) => p.id)).toEqual(["h", "g", "n"]);
  });

  it("removes a participant who left", () => {
    const next = applyEvent(base, { type: "participant_left", payload: { participantId: "g" } });
    expect(next.participants.map((p) => p.id)).toEqual(["h"]);
  });

  it("moves the host flag", () => {
    const next = applyEvent(base, { type: "host_changed", payload: { participantId: "g" } });
    expect(next.participants.filter((p) => p.isHost).map((p) => p.id)).toEqual(["g"]);
  });

  it("clears votes when a round starts", () => {
    const next = applyEvent(base, { type: "round_started", payload: { label: "Signup", jiraKey: "ABC-1" } });
    expect(next.round).toEqual({ label: "Signup", jiraKey: "ABC-1", revealed: false, votes: [] });
    expect(next.ownVote).toBeUndefined();
    expect(next.participants.every((p) => !p.hasVoted)).toBe(true);
  });

  it("marks a voter without learning the card", () => {
    const next = applyEvent(base, { type: "participant_voted", payload: { participantId: "g" } });
    expect(next.participants.find((p) => p.id === "g")?.hasVoted).toBe(true);
  });

  it("reveals votes", () => {
    const votes = [{ participantId: "h", card: "5" }];
    const next = applyEvent(base, { type: "votes_revealed", payload: { votes } });
    expect(next.round).toMatchObject({ revealed: true, votes });
  });

  it("ignores a reveal without a round", () => {
    const noRound = { ...base, round: undefined };
    expect(applyEvent(noRound, { type: "votes_revealed", payload: { votes: [] } })).toBe(noRound);
  });
});

describe("summarizeVotes", () => {
  it("averages and orders the distribution", () => {
    const summary = summarizeVotes([{ card: "3" }, { card: "5" }, { card: "5" }, { card: "13" }]);
    expect(summary.average).toBe(6.5);
    expect(summary.consensus).toBe(false);
    expect(summary.distribution).toEqual([
      ["5", 2],
      ["3", 1],
      ["13", 1],
    ]);
  });

  it("detects consensus", () => {
    expect(summarizeVotes([{ card: "8" }, { card: "8" }]).consensus).toBe(true);
  });

  it("handles no votes", () => {
    expect(summarizeVotes([])).toEqual({ average: undefined, consensus: false, distribution: [] });
  });
});
