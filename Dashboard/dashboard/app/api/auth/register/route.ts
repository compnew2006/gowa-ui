import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { isRegistrationEnabled } from "@/lib/server-auth";
import { proxyApiRequest } from "@/lib/server-api-proxy";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function POST(request: NextRequest) {
  if (!isRegistrationEnabled()) {
    return NextResponse.json({ detail: "Registration is disabled." }, { status: 403 });
  }
  return proxyApiRequest(request, "auth/register");
}

