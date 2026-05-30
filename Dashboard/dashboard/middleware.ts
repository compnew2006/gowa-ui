import { NextResponse, type NextRequest } from "next/server";
import { SESSION_COOKIE, getAuthenticatedUser, hasLocalDashboardAuth } from "@/lib/server-auth";

const PUBLIC_PATHS = new Set(["/login", "/register"]);
const isProduction = process.env.NODE_ENV === "production";

interface RateLimitRecord {
  count: number;
  resetTime: number;
}

const rateLimitMap = new Map<string, RateLimitRecord>();
const MAX_MAP_SIZE = 10000;

function checkRateLimit(ip: string, limit: number, windowMs: number): boolean {
  const now = Date.now();
  
  // Basic cleanup to prevent memory leak when map gets too large
  if (rateLimitMap.size > MAX_MAP_SIZE) {
    for (const [key, record] of rateLimitMap.entries()) {
      if (now > record.resetTime) {
        rateLimitMap.delete(key);
      }
    }
  }

  const record = rateLimitMap.get(ip);
  if (!record) {
    rateLimitMap.set(ip, { count: 1, resetTime: now + windowMs });
    return true;
  }

  if (now > record.resetTime) {
    record.count = 1;
    record.resetTime = now + windowMs;
    return true;
  }

  record.count += 1;
  return record.count <= limit;
}

function getClientIp(request: NextRequest): string {
  const forwardedFor = request.headers.get("x-forwarded-for");
  if (forwardedFor) {
    return forwardedFor.split(",")[0].trim();
  }
  const realIp = request.headers.get("x-real-ip");
  if (realIp) {
    return realIp;
  }
  return (request as any).ip || "127.0.0.1";
}

function isPublicAsset(pathname: string) {
  return (
    pathname.startsWith("/_next/") ||
    pathname.startsWith("/favicon") ||
    pathname.startsWith("/robots.txt") ||
    pathname.startsWith("/sitemap") ||
    pathname.match(/\.(?:ico|png|jpg|jpeg|gif|webp|svg|css|js|txt|xml)$/)
  );
}

function createNonce() {
  return crypto.randomUUID().replaceAll("-", "");
}

function createContentSecurityPolicy(nonce: string) {
  return [
    "default-src 'self'",
    "base-uri 'self'",
    "object-src 'none'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    `script-src 'self' 'unsafe-inline' 'unsafe-eval' 'nonce-${nonce}' 'strict-dynamic'`,
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob: https:",
    "font-src 'self' data:",
    "media-src 'self' blob: https:",
    "worker-src 'self' blob:",
    "connect-src 'self' https://*.sentry.io",
    isProduction ? "upgrade-insecure-requests" : "",
  ]
    .filter(Boolean)
    .join("; ");
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // App-level rate limiting on sensitive authentication/api endpoints
  if (pathname.startsWith("/api/auth/")) {
    const ip = getClientIp(request);
    // Limit to 30 requests per minute
    const isAllowed = checkRateLimit(ip, 30, 60 * 1000);
    if (!isAllowed) {
      return NextResponse.json(
        { detail: "Too many authentication requests. Please try again later." },
        { status: 429 }
      );
    }
  }

  const nonce = createNonce();
  const contentSecurityPolicy = createContentSecurityPolicy(nonce);
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("Content-Security-Policy", contentSecurityPolicy);

  if (pathname.startsWith("/api/") || PUBLIC_PATHS.has(pathname) || isPublicAsset(pathname)) {
    const response = NextResponse.next({ request: { headers: requestHeaders } });
    response.headers.set("Content-Security-Policy", contentSecurityPolicy);
    return response;
  }

  const isAuthenticated = hasLocalDashboardAuth()
    ? Boolean(await getAuthenticatedUser(request))
    : request.cookies.has(SESSION_COOKIE);

  if (!isAuthenticated) {
    // Use the forwarded host if available, otherwise fall back to request host
    const forwardedHost = request.headers.get("x-forwarded-host");
    const forwardedProto = request.headers.get("x-forwarded-proto") || "https";
    const hostname = forwardedHost || request.nextUrl.hostname;
    const protocol = forwardedProto || request.nextUrl.protocol.replace(":", "");

    const loginUrl = new URL(`${protocol}://${hostname}/login`);
    loginUrl.searchParams.set("next", pathname);
    const response = NextResponse.redirect(loginUrl);
    response.headers.set("Content-Security-Policy", contentSecurityPolicy);
    return response;
  }

  const response = NextResponse.next({ request: { headers: requestHeaders } });
  response.headers.set("Content-Security-Policy", contentSecurityPolicy);
  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image).*)"],
};
