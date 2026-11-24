// Sessions live in sessionStorage, so each browser tab is its own participant
// and a refresh keeps you in the room.

export interface StoredSession {
  participantId: string;
  sessionToken: string;
  displayName: string;
}

const key = (code: string) => `zenith.session.${code.toUpperCase()}`;
const NAME_KEY = "zenith.displayName";

export function loadSession(code: string): StoredSession | undefined {
  try {
    const raw = sessionStorage.getItem(key(code));
    return raw ? (JSON.parse(raw) as StoredSession) : undefined;
  } catch {
    return undefined;
  }
}

export function saveSession(code: string, session: StoredSession): void {
  try {
    sessionStorage.setItem(key(code), JSON.stringify(session));
    localStorage.setItem(NAME_KEY, session.displayName);
  } catch {
    // Storage unavailable: the session only lasts until the page reloads.
  }
}

export function clearSession(code: string): void {
  try {
    sessionStorage.removeItem(key(code));
  } catch {
    // Nothing to clear.
  }
}

export function lastDisplayName(): string {
  try {
    return localStorage.getItem(NAME_KEY) ?? "";
  } catch {
    return "";
  }
}
