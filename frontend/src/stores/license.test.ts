// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mocks = vi.hoisted(() => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
}));

import { useLicenseStore } from "./license";

describe("useLicenseStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("loads bootstrap state and normalizes usage payloads", async () => {
    mocks.api.get.mockResolvedValueOnce({
      data: {
        data: {
          enabled: true,
          status: "active",
          locked: false,
          hwid_full: "abc123",
          hwid_short: "abc123",
          hwid_hash: "abc123",
          tier: "starter",
          license_kind: "paid",
          duration_label: "56d",
          max_organizations: 1,
          max_users_per_org: 5,
          max_whatsapp_endpoints_per_org: 5,
          max_workers: 2,
          max_workers_per_org: 2,
          max_storage_bytes_per_org: 5368709120,
          expiring_soon: true,
          days_until_expiry: 10,
          quota_overages: {},
          usage: {
            organizations: {
              current: 1,
              limit: 1,
              over_quota: false,
              overage: 0,
            },
            users_per_org: {
              current: 3,
              limit: 5,
              over_quota: false,
              overage: 0,
            },
            whatsapp_endpoints_per_org: {
              current: 2,
              limit: 5,
              over_quota: false,
              overage: 0,
            },
            storage_bytes_per_org: {
              current: 1024,
              limit: 5368709120,
              over_quota: false,
              overage: 0,
            },
            organization_details: [],
          },
        },
      },
    });

    const store = useLicenseStore();
    const result = await store.fetchBootstrap();

    expect(result.status).toBe("active");
    expect(store.isLocked).toBe(false);
    expect(store.showExpiryWarning).toBe(true);
    expect(store.state.usage.users_per_org.current).toBe(3);
    expect(store.state.usage.storage_bytes_per_org.current).toBe(1024);
    expect(store.state.duration_label).toBe("56d");
  });

  it("marks the current state as locked without dropping known HWID values", () => {
    const store = useLicenseStore();
    store.setState({
      enabled: true,
      status: "active",
      locked: false,
      hwid_full: "server-hwid",
      hwid_short: "server-short",
      hwid_hash: "server-hash",
      max_organizations: 1,
      max_users_per_org: 5,
      max_whatsapp_endpoints_per_org: 5,
      max_workers: 2,
      max_workers_per_org: 2,
      max_storage_bytes_per_org: 5368709120,
      expiring_soon: false,
      quota_overages: {},
    });

    store.markLocked("time_rollback");

    expect(store.isLocked).toBe(true);
    expect(store.state.reason).toBe("time_rollback");
    expect(store.state.hwid_full).toBe("server-hwid");
  });

  it("exposes disabled licensing state without marking the deployment as locked", () => {
    const store = useLicenseStore();

    store.setState({
      enabled: false,
      status: "disabled",
      locked: false,
      hwid_full: "server-hwid",
      hwid_short: "server-short",
      hwid_hash: "server-hash",
      max_organizations: 0,
      max_users_per_org: 0,
      max_whatsapp_endpoints_per_org: 0,
      max_workers: 0,
      max_workers_per_org: 0,
      max_storage_bytes_per_org: 0,
      expiring_soon: false,
      quota_overages: {},
    });

    expect(store.isDisabled).toBe(true);
    expect(store.isLocked).toBe(false);
  });
});
