import { NextResponse, type NextRequest } from "next/server";

export const SESSION_COOKIE = "dashboard_session";
export const USER_COOKIE = "dashboard_user";

const DEFAULT_SESSION_MAX_AGE_SECONDS = 8 * 60 * 60;
const ALL_ADMIN_PERMISSIONS = {
  can_approve: true,
  can_reject: true,
  can_delete: true,
  can_manage_team: true,
  can_export: true,
  can_manage_settings: true,
  can_manage_campaigns: true,
};

const HAS_LOCAL_AUTH = Boolean(process.env.DASHBOARD_ADMIN_EMAIL && process.env.DASHBOARD_ADMIN_PASSWORD);
const LOCAL_DASHBOARD_USER = {
  email: process.env.DASHBOARD_ADMIN_EMAIL ?? "",
  name: process.env.DASHBOARD_ADMIN_NAME ?? "Dashboard Admin",
  role: "admin" as const,
  permissions: ALL_ADMIN_PERMISSIONS,
};

interface SessionPayload {
  email: string;
  name?: string;
  role: string;
  permissions: Record<string, boolean>;
  exp: number;
}

function isLoopback(hostname: string) {
  return ["localhost", "127.0.0.1", "::1"].includes(hostname);
}

export function getBackendApiUrl() {
  const value = process.env.API_URL;
  if (!value) {
    if (process.env.NODE_ENV === "production") {
      throw new Error("API_URL must be set in production.");
    }
    return "http://localhost:8000";
  }

  const url = new URL(value);
  if (process.env.NODE_ENV === "production" && url.protocol !== "https:" && !isLoopback(url.hostname)) {
    throw new Error("API_URL must use HTTPS in production unless it targets loopback.");
  }
  return url.origin;
}

export function backendApiUrl(path: string, search = "") {
  return new URL(`/api/${path.replace(/^\/+/, "")}${search}`, getBackendApiUrl()).toString();
}

export function sessionCookieOptions(maxAge = DEFAULT_SESSION_MAX_AGE_SECONDS) {
  return {
    httpOnly: true,
    secure: true,
    sameSite: "lax" as const,
    path: "/",
    maxAge,
  };
}

export function readableUserCookieOptions(maxAge = DEFAULT_SESSION_MAX_AGE_SECONDS) {
  return {
    httpOnly: true, // Security: Prevent XSS attacks from reading user data
    secure: true,
    sameSite: "lax" as const,
    path: "/",
    maxAge,
  };
}

export function clearAuthCookies(response: NextResponse) {
  response.cookies.set(SESSION_COOKIE, "", { ...sessionCookieOptions(0), maxAge: 0 });
  response.cookies.set(USER_COOKIE, "", { ...readableUserCookieOptions(0), maxAge: 0 });
}

export function getSessionToken(request: NextRequest) {
  return request.cookies.get(SESSION_COOKIE)?.value ?? null;
}

export function isRegistrationEnabled() {
  return process.env.ENABLE_REGISTRATION === "true" || process.env.NEXT_PUBLIC_ENABLE_REGISTRATION === "true";
}

export function hasLocalDashboardAuth() {
  return HAS_LOCAL_AUTH;
}

export function getLocalDashboardUser() {
  return LOCAL_DASHBOARD_USER;
}

export function validateLocalCredentials(email: string, password: string): boolean {
  return email === LOCAL_DASHBOARD_USER.email &&
         password === process.env.DASHBOARD_ADMIN_PASSWORD;
}

const SESSION_SECRET = (() => {
  const secret = process.env.DASHBOARD_SESSION_SECRET || process.env.JWT_SECRET;
  if (!secret) {
    if (process.env.NODE_ENV === "production") {
      throw new Error("DASHBOARD_SESSION_SECRET must be set when local dashboard auth is enabled.");
    }
    return "development-dashboard-session-secret";
  }
  return secret;
})();

function getSessionSecret() {
  return SESSION_SECRET;
}

function base64UrlEncodeBytes(bytes: Uint8Array) {
  let binary = "";
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
}

function base64UrlEncode(value: string) {
  return base64UrlEncodeBytes(new TextEncoder().encode(value));
}

function base64UrlDecode(value: string) {
  const padded = value.replaceAll("-", "+").replaceAll("_", "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const binary = atob(padded);
  return new TextDecoder().decode(Uint8Array.from(binary, (char) => char.charCodeAt(0)));
}

let cachedSigningKey: CryptoKey | null = null;

async function getSigningKey(): Promise<CryptoKey> {
  if (cachedSigningKey) return cachedSigningKey;
  cachedSigningKey = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(getSessionSecret()),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  return cachedSigningKey;
}

async function sign(value: string) {
  const key = await getSigningKey();
  const signature = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(value));
  return base64UrlEncodeBytes(new Uint8Array(signature));
}

export async function createLocalSessionToken() {
  const payload: SessionPayload = {
    ...getLocalDashboardUser(),
    exp: Math.floor(Date.now() / 1000) + DEFAULT_SESSION_MAX_AGE_SECONDS,
  };
  const encodedPayload = base64UrlEncode(JSON.stringify(payload));
  return `${encodedPayload}.${await sign(encodedPayload)}`;
}

export async function verifyLocalSessionToken(token: string | null) {
  if (!token) return null;
  const [encodedPayload, providedSignature] = token.split(".");
  if (!encodedPayload || !providedSignature) return null;
  const expectedSignature = await sign(encodedPayload);
  if (providedSignature !== expectedSignature) return null;

  try {
    const payload = JSON.parse(base64UrlDecode(encodedPayload)) as SessionPayload;
    if (!payload.email || payload.exp < Math.floor(Date.now() / 1000)) return null;
    return payload;
  } catch {
    return null;
  }
}

export async function getAuthenticatedUser(request: NextRequest) {
  if (!hasLocalDashboardAuth()) return null;
  return verifyLocalSessionToken(getSessionToken(request));
}