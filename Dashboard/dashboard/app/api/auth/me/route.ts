import { NextResponse, type NextRequest } from "next/server";
import { getAuthenticatedUser, hasLocalDashboardAuth } from "@/lib/server-auth";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function GET(request: NextRequest) {
  if (hasLocalDashboardAuth()) {
    const user = await getAuthenticatedUser(request);
    if (!user) return NextResponse.json({ detail: "Unauthorized" }, { status: 401 });
    return NextResponse.json({
      email: user.email,
      name: user.name,
      role: user.role,
      permissions: user.permissions,
    });
  }

  return NextResponse.json({ detail: "Authentication profile is not configured." }, { status: 501 });
}

