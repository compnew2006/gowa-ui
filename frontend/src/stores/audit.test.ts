// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mocks = vi.hoisted(() => ({
  auditService: {
    list: vi.fn(),
  },
}));

vi.mock("@/services/audit", () => ({
  auditService: mocks.auditService,
}));

import { useAuditStore } from "./audit";

describe("audit store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("fetch populates events and total", async () => {
    mocks.auditService.list.mockResolvedValue({
      events: [
        {
          id: "1",
          created_at: "2026-01-01T00:00:00Z",
          category: "auth",
          action: "login_success",
          source: "user",
          success: true,
        },
      ],
      total: 1,
      page: 1,
      per_page: 50,
    });

    const store = useAuditStore();
    await store.fetch();

    expect(store.events).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
    expect(store.error).toBeNull();
  });

  it("fetch captures an error on failure", async () => {
    mocks.auditService.list.mockRejectedValue({
      response: { data: { message: "boom" } },
    });

    const store = useAuditStore();
    await store.fetch();

    expect(store.events).toHaveLength(0);
    expect(store.error).toBe("boom");
    expect(store.loading).toBe(false);
  });

  it("setFilter resets page to 1", () => {
    const store = useAuditStore();
    store.filters.page = 3;
    store.setFilter("category", "auth");
    expect(store.filters.page).toBe(1);
    expect(store.filters.category).toBe("auth");
  });

  it("resetFilters clears back to defaults", () => {
    const store = useAuditStore();
    store.setFilter("category", "auth");
    store.setFilter("action", "login_success");
    store.resetFilters();
    expect(store.filters.category).toBeUndefined();
    expect(store.filters.action).toBeUndefined();
    expect(store.filters.page).toBe(1);
    expect(store.filters.per_page).toBe(50);
  });

  it("goToPage updates the page and refetches", async () => {
    mocks.auditService.list.mockResolvedValue({
      events: [],
      total: 0,
      page: 2,
      per_page: 50,
    });

    const store = useAuditStore();
    await store.goToPage(2);

    expect(store.filters.page).toBe(2);
    expect(mocks.auditService.list).toHaveBeenCalledTimes(1);
  });
});
