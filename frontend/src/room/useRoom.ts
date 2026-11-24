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

  // Once the room has ended, later connection trouble must not replace that screen.
  const setStatus = useCallback((status: "reconnecting" | "unauthorized") => {
    setView((current) => (current.status === "ended" ? current : { ...current, status }));
  }, []);

  const handleError = useCallback(
    (error: unknown) => {
      if (error instanceof ApiError && error.status === 401) setStatus("unauthorized");
      else if (error instanceof ApiError && error.code === "room_not_found") end("closed");
      else setStatus("reconnecting");
    },
    [end, setStatus],
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
        if (event.type === "room_closed") {
          stop();
          return end(event.payload.reason);
        }
        if (buffer) return void buffer.push(event);
        if (roomRef.current) setRoom(applyEvent(roomRef.current, event));
      },
      onDisconnect: () => setStatus("reconnecting"),
      onUnauthorized: () => setStatus("unauthorized"),
      onRoomGone: () => end("closed"),
    });

    return stop;
  }, [code, token, setRoom, end, setStatus, handleError]);

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
