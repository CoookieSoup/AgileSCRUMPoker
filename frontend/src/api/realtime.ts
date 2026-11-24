import type { RoomEvent } from "./types";

export const CLOSE_UNAUTHORIZED = 4401;
export const CLOSE_ROOM_GONE = 4404;

export interface RealtimeHandlers {
  /** Subscribed: fetch the room state now, then apply events. */
  onReady: () => void;
  onEvent: (event: RoomEvent) => void;
  /** Connection lost; a reconnect is scheduled. */
  onDisconnect: () => void;
  onUnauthorized: () => void;
  onRoomGone: () => void;
}

const MAX_BACKOFF_MS = 10_000;

export function eventsUrl(code: string, location: Pick<Location, "protocol" | "host"> = window.location): string {
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  return `${scheme}//${location.host}/api/v1/rooms/${encodeURIComponent(code)}/events`;
}

/** Keeps one WebSocket open to the room, reconnecting until stopped. Returns a stop function. */
export function connectRoom(code: string, token: string, handlers: RealtimeHandlers): () => void {
  let socket: WebSocket | undefined;
  let stopped = false;
  let attempt = 0;
  let retryTimer: ReturnType<typeof setTimeout> | undefined;

  const open = () => {
    if (stopped) return;
    const current = new WebSocket(eventsUrl(code));
    socket = current;
    current.onopen = () => current.send(JSON.stringify({ type: "auth", token }));
    current.onmessage = (message) => {
      if (stopped) return;
      let data: { type?: string };
      try {
        data = JSON.parse(String(message.data));
      } catch {
        return;
      }
      if (data.type === "ready") {
        attempt = 0;
        handlers.onReady();
      } else if (data.type) {
        handlers.onEvent(data as RoomEvent);
      }
    };
    current.onclose = (close) => {
      if (socket === current) socket = undefined;
      if (stopped) return;
      if (close.code === CLOSE_UNAUTHORIZED) return handlers.onUnauthorized();
      if (close.code === CLOSE_ROOM_GONE) return handlers.onRoomGone();
      handlers.onDisconnect();
      const delay = Math.min(MAX_BACKOFF_MS, 500 * 2 ** attempt);
      attempt += 1;
      retryTimer = setTimeout(open, delay);
    };
  };

  open();

  return () => {
    stopped = true;
    clearTimeout(retryTimer);
    socket?.close(1000);
  };
}
