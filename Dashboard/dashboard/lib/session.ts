export interface StoredUser {
  name?: string;
  email?: string;
  role?: string;
  permissions?: Record<string, boolean>;
}

const USER_KEY = "user";
const USER_COOKIE = "dashboard_user";

function getStorage(kind: "session" | "local"): Storage | null {
  if (typeof window === "undefined") return null;
  try {
    return kind === "session" ? window.sessionStorage : window.localStorage;
  } catch {
    return null;
  }
}

function getCookieValue(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(^| )${name}=([^;]*)`));
  return match ? match[2] : null;
}

export function setSession(user: StoredUser | null | undefined) {
  const session = getStorage("session");
  const local = getStorage("local");

  local?.removeItem(USER_KEY);

  if (!session) return;
  session.removeItem(USER_KEY);
  if (user) session.setItem(USER_KEY, JSON.stringify(user));
}

export function clearSession() {
  for (const storage of [getStorage("session"), getStorage("local")]) {
    storage?.removeItem(USER_KEY);
  }
  if (typeof document !== "undefined") {
    document.cookie = `${USER_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax; Secure`;
  }
}

export function getStoredUser(): StoredUser | null {
  const rawCookieUser = getCookieValue(USER_COOKIE);
  const rawUser = rawCookieUser ? decodeURIComponent(rawCookieUser) : getStorage("session")?.getItem(USER_KEY);
  if (!rawUser) return null;
  try {
    return JSON.parse(rawUser) as StoredUser;
  } catch {
    return null;
  }
}
