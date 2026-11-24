import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, api } from "../api/client";
import { connectRoom } from "../api/realtime";
import type { StoredSession } from "../api/session";
import type { CloseReason, RoomEvent, RoomState, StartRoundRequest } from "../api/types";
import { applyEvent } from "./roomReducer";

export type RoomStatus = "connecting" | "live" | "reconnecting" | "ended" | "unauthorized";

export interface RoomView {
  status: RoomStatus;
  room?: RoomState;
  endedReason?: CloseReason;
}

/**
 * Live room state for one session: opens the event stream, loads the snapshot
 * once subscribed, and replays any events that arrived while it loaded.
 */
export function useRoom(code: string, session: StoredSession) {
  const [view, setView] = useState<RoomView>({ status: "connecting" });
  const roomRef = useRef<RoomState | undefined>(undefined);
  const token = session.sessionToken;

  const setRoom = useCallback((room: RoomState) => {
    roomRef.current = room;
    setView((current) => (current.status === "ended" ? current : { ...current, status: "live", room }));
  }, []);

  const end = useCallback((reason: CloseReason) => {
    setView((current) => ({ ...current, status: "ended", endedReason: reason }));
  }, []);

  const handleError = useCallback(
    (error: unknown) => {
      if (error instanceof ApiError && error.status === 401) setView({ status: "unauthorized" });
      else if (error instanceof ApiError && error.code === "room_not_found") end("closed");
      else setView((current) => ({ ...current, status: "reconnecting" }));
    },
    [end],
  );

  useEffect(() => {
    let loading = 0;
    let buffer: RoomEvent[] | undefined;

    const stop = connectRoom(code, token, {
      onReady: () => {
        const generation = ++loading;
        buffer = [];
        api
          .getRoom(code, token)
          .then((snapshot) => {
            if (generation !== loading) return;
            const room = (buffer ?? []).reduce(applyEvent, snapshot);
            buffer = undefined;
            setRoom(room);
          })
          .catch((error: unknown) => {
            if (generation === loading) handleError(error);
          });
      },
      onEvent: (event) => {
        if (event.type === "room_closed") return end(event.payload.reason);
        if (buffer) return void buffer.push(event);
        if (roomRef.current) setRoom(applyEvent(roomRef.current, event));
      },
      onDisconnect: () => setView((current) => ({ ...current, status: "reconnecting" })),
      onUnauthorized: () => setView({ status: "unauthorized" }),
      onRoomGone: () => end("closed"),
    });

    return stop;
  }, [code, token, setRoom, end, handleError]);

  const vote = useCallback(
    async (card: string) => {
      const before = roomRef.current;
      if (before) {
        setRoom({
          ...before,
          ownVote: card,
          participants: before.participants.map((p) => (p.id === session.participantId ? { ...p, hasVoted: true } : p)),
        });
      }
      try {
        await api.castVote(code, token, card);
      } catch (error) {
        if (before) setRoom(before);
        throw error;
      }
    },
    [code, token, session.participantId, setRoom],
  );

  const startRound = useCallback(
    async (round: StartRoundRequest) => setRoom(await api.startRound(code, token, round)),
    [code, token, setRoom],
  );

  const reveal = useCallback(async () => setRoom(await api.revealVotes(code, token)), [code, token, setRoom]);
  const leave = useCallback(() => api.leaveRoom(code, token), [code, token]);
  const close = useCallback(() => api.closeRoom(code, token), [code, token]);

  return { ...view, vote, startRound, reveal, leave, close };
}
