import { describe, expect, it, vi, afterEach } from "vitest";
import { apiFetch } from "@/lib/api";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("apiFetch", () => {
  it("returns parsed JSON for successful responses", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ ok: true }), { status: 200 })));

    await expect(apiFetch<{ ok: boolean }>("/health")).resolves.toEqual({ ok: true });
  });

  it("throws typed ApiError for failed responses", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ detail: "Nope" }), { status: 403 })));

    await expect(apiFetch("/forbidden")).rejects.toMatchObject({
      name: "ApiError",
      status: 403,
      message: "Nope",
    });
  });

  it("does not force JSON content type for FormData", async () => {
    let capturedInit: RequestInit | undefined;
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      capturedInit = init;
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const form = new FormData();
    form.append("file", new Blob(["x"]), "x.txt");
    await apiFetch("/upload", { method: "POST", body: form });

    expect(new Headers(capturedInit?.headers).has("Content-Type")).toBe(false);
  });
});
