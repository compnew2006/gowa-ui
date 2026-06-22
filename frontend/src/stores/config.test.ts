// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mocks = vi.hoisted(() => ({
  api: {
    get: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
}));

import { useConfigStore } from "./config";

const appConfig = {
  whatsapp_provider: "meta",
  features: {
    templates: true,
    flows: true,
    catalog: true,
    business_profile: true,
    campaigns: true,
    meta_insights: true,
  },
};

describe("useConfigStore managed modules", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("keeps compiled modules enabled until module state is loaded", () => {
    const store = useConfigStore();

    expect(store.isModuleEnabled("facebook-comments")).toBe(true);
  });

  it("loads effective module state with the application config", async () => {
    mocks.api.get.mockImplementation(async (path: string) => {
      if (path === "/config") {
        return { data: { data: appConfig } };
      }
      if (path === "/modules/effective") {
        return {
          data: {
            data: [
              {
                key: "facebook-comments",
                effective_enabled: false,
              },
              {
                key: "facebook-accounts",
                effective_enabled: true,
              },
            ],
          },
        };
      }
      throw new Error(`unexpected path: ${path}`);
    });

    const store = useConfigStore();
    await store.fetchConfig();

    expect(store.modulesLoaded).toBe(true);
    expect(store.isModuleEnabled("facebook-comments")).toBe(false);
    expect(store.isModuleEnabled("facebook-accounts")).toBe(true);
    expect(store.isModuleEnabled("facebook-unknown")).toBe(false);
  });
});
