// @vitest-environment happy-dom

import { describe, expect, it } from "vitest";

import { unwrapWhatsAppFilterResultsPage } from "./api";

const result = {
  id: "result-1",
  batch_id: "batch-1",
  phone_number: "+14155552671",
  contact_name: "Alice",
  is_valid: true,
  checked_at: "2026-05-30T10:00:00Z",
  created_at: "2026-05-30T09:59:00Z",
};

describe("unwrapWhatsAppFilterResultsPage", () => {
  it("unwraps fastglue-enveloped paginated results", () => {
    const page = unwrapWhatsAppFilterResultsPage({
      status: "success",
      data: {
        data: [result],
        total: 1,
        page: 1,
        limit: 25,
      },
    });

    expect(page.data).toEqual([result]);
    expect(page.total).toBe(1);
    expect(page.page).toBe(1);
    expect(page.limit).toBe(25);
  });

  it("keeps already-normalized paginated results", () => {
    const page = unwrapWhatsAppFilterResultsPage({
      data: [result],
      total: 1,
      page: 1,
      limit: 25,
    });

    expect(page.data).toEqual([result]);
    expect(page.total).toBe(1);
  });

  it("accepts raw result arrays from mocks or legacy responses", () => {
    const page = unwrapWhatsAppFilterResultsPage([result]);

    expect(page.data).toEqual([result]);
    expect(page.total).toBe(1);
  });
});
