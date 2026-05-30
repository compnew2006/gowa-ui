import { NextResponse, type NextRequest } from "next/server";
import {
  SESSION_COOKIE,
  USER_COOKIE,
  backendApiUrl,
  createLocalSessionToken,
  getLocalDashboardUser,
  hasLocalDashboardAuth,
  validateLocalCredentials,
  readableUserCookieOptions,
  sessionCookieOptions,
} from "@/lib/server-auth";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function POST(request: NextRequest) {
  let credentials: { email?: string; password?: string } = {};

  try {
    const rawBody = await request.text();
    if (!rawBody.trim()) {
      return NextResponse.json({ detail: "Request body cannot be empty." }, { status: 400 });
    }
    credentials = JSON.parse(rawBody);
    if (!credentials || typeof credentials !== "object") {
      return NextResponse.json({ detail: "Request body must be a valid JSON object." }, { status: 400 });
    }
  } catch {
    return NextResponse.json({ detail: "Invalid JSON format." }, { status: 400 });
  }

  if (hasLocalDashboardAuth()) {
    if (!validateLocalCredentials(credentials.email || "", credentials.password || "")) {
      return NextResponse.json({ detail: "Invalid credentials." }, { status: 401 });
    }

    const user = getLocalDashboardUser();
    const response = NextResponse.json({ user });
    response.cookies.set(SESSION_COOKIE, await createLocalSessionToken(), sessionCookieOptions());
    response.cookies.set(USER_COOKIE, JSON.stringify(user), readableUserCookieOptions());
    return response;
  }

  const upstream = await fetch(backendApiUrl("auth/login"), {
    method: "POST",
    headers: {
      "Content-Type": request.headers.get("content-type") ?? "application/json",
      Accept: request.headers.get("accept") ?? "application/json",
    },
    body: JSON.stringify(credentials),
    cache: "no-store",
  });

  const payload = await upstream.json().catch(() => null);
  if (!upstream.ok) {
    return NextResponse.json(payload ?? { detail: `HTTP ${upstream.status}` }, { status: upstream.status });
  }

  const token = payload?.access_token;
  if (typeof token !== "string" || token.length === 0) {
    return NextResponse.json({ detail: "Login response did not include an access token." }, { status: 502 });
  }

  const user = payload?.user ?? null;
  const response = NextResponse.json({ user });
  response.cookies.set(SESSION_COOKIE, token, sessionCookieOptions());
  if (user) {
    response.cookies.set(USER_COOKIE, JSON.stringify(user), readableUserCookieOptions());
  }
  return response;
}
