import type { NextConfig } from "next";
import path from "path";

const isProduction = process.env.NODE_ENV === "production";

function isLoopback(hostname: string) {
  return ["localhost", "127.0.0.1", "::1"].includes(hostname);
}

if (isProduction && !process.env.API_URL) {
  throw new Error("API_URL must be set in production.");
}

if (isProduction && process.env.API_URL) {
  const apiUrl = new URL(process.env.API_URL);
  if (apiUrl.protocol !== "https:" && !isLoopback(apiUrl.hostname)) {
    throw new Error("API_URL must use HTTPS in production unless it targets loopback.");
  }
}

const securityHeaders = [
  { key: "Referrer-Policy", value: "no-referrer" },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), payment=(), usb=()" },
  ...(isProduction ? [{ key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains; preload" }] : []),
];

const nextConfig: NextConfig = {
  outputFileTracingRoot: path.resolve(__dirname),
  reactStrictMode: true,
  poweredByHeader: false,
  images: {
    unoptimized: true,
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: securityHeaders,
      },
    ];
  },
};

module.exports = nextConfig;
