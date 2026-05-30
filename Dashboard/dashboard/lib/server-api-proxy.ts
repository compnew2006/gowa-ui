import { NextResponse, type NextRequest } from "next/server";
import { backendApiUrl, getAuthenticatedUser, getSessionToken, hasLocalDashboardAuth } from "@/lib/server-auth";

const HOP_BY_HOP_HEADERS = new Set([
  "connection",
  "content-length",
  "host",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
  "cookie",
]);

const RESPONSE_HEADERS_TO_SKIP = new Set([
  "content-encoding",
  "content-length",
  "set-cookie",
  "transfer-encoding",
]);

function getForwardHeaders(request: NextRequest) {
  const headers = new Headers();
  request.headers.forEach((value, key) => {
    if (!HOP_BY_HOP_HEADERS.has(key.toLowerCase())) headers.set(key, value);
  });

  const token = getSessionToken(request);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  return headers;
}

function getResponseHeaders(upstream: Response) {
  const headers = new Headers();
  upstream.headers.forEach((value, key) => {
    if (!RESPONSE_HEADERS_TO_SKIP.has(key.toLowerCase())) headers.set(key, value);
  });
  return headers;
}

export async function proxyApiRequest(request: NextRequest, path: string) {
  if (hasLocalDashboardAuth()) {
    const user = await getAuthenticatedUser(request);
    if (!user) return NextResponse.json({ detail: "Unauthorized" }, { status: 401 });
  }

  const target = backendApiUrl(path, request.nextUrl.search);
  const method = request.method.toUpperCase();
  const hasBody = method !== "GET" && method !== "HEAD";
  const body = hasBody ? await request.arrayBuffer() : undefined;

  const upstream = await fetch(target, {
    method,
    headers: getForwardHeaders(request),
    body,
    redirect: "manual",
    cache: "no-store",
  });

  return new NextResponse(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: getResponseHeaders(upstream),
  });
}
