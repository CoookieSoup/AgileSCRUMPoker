import type { RoomState, Session, StartRoundRequest } from "./types";

const BASE = "/api/v1";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(method: string, path: string, token?: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers["Content-Type"] = "application/json";

  let response: Response;
  try {
    response = await fetch(`${BASE}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "network_error", "Cannot reach the server. Check your connection.");
  }

  if (response.status === 204) return undefined as T;

  const data: unknown = await response.json().catch(() => undefined);
  if (!response.ok) {
    const error = (data ?? {}) as { code?: string; message?: string };
    throw new ApiError(
      response.status,
      error.code ?? "unknown_error",
      error.message ?? `Request failed with status ${response.status}`,
    );
  }

  return data as T;
}

const room = (code: string) => `/rooms/${encodeURIComponent(code)}`;

export const api = {
  createRoom: (displayName: string) => request<Session>("POST", "/rooms", undefined, { displayName }),
  joinRoom: (code: string, displayName: string) =>
    request<Session>("POST", `${room(code)}/participants`, undefined, { displayName }),
  getRoom: (code: string, token: string) => request<RoomState>("GET", room(code), token),
  closeRoom: (code: string, token: string) => request<void>("DELETE", room(code), token),
  leaveRoom: (code: string, token: string) => request<void>("DELETE", `${room(code)}/participants/me`, token),
  startRound: (code: string, token: string, round: StartRoundRequest) =>
    request<RoomState>("POST", `${room(code)}/round`, token, round),
  castVote: (code: string, token: string, card: string) =>
    request<void>("PUT", `${room(code)}/round/vote`, token, { card }),
  revealVotes: (code: string, token: string) => request<RoomState>("POST", `${room(code)}/round/reveal`, token),
};
