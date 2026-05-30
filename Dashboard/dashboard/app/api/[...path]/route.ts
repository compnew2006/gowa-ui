import type { NextRequest } from "next/server";
import { proxyApiRequest } from "@/lib/server-api-proxy";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

function pathFromParams(params: { path?: string[] }) {
  return params.path?.join("/") ?? "";
}

export async function GET(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

export async function HEAD(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

export async function POST(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

export async function PUT(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

export async function PATCH(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

export async function DELETE(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

export async function OPTIONS(request: NextRequest, { params }: { params: Promise<{ path?: string[] }> }) {
  return proxyApiRequest(request, pathFromParams(await params));
}

