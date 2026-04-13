// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";

const authStore = {
  isAuthenticated: true,
  restoreSession: vi.fn(async () => true),
  hasPermission: vi.fn<(resource: string, action?: string) => boolean>(
    () => true,
  ),
  user: null as any,
  userRole: "admin",
};

const licenseStore = {
  isLocked: false,
  showQuotaOverage: false,
  fetchBootstrap: vi.fn(async () => ({})),
};

const configStore = {
  isWhatsmeow: false,
};

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => authStore,
}));

vi.mock("@/stores/license", () => ({
  useLicenseStore: () => licenseStore,
}));

vi.mock("@/stores/config", () => ({
  useConfigStore: () => configStore,
}));

describe("router license route", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authStore.isAuthenticated = true;
    authStore.user = null;
    authStore.userRole = "admin";
    authStore.restoreSession.mockResolvedValue(true);
    authStore.hasPermission.mockReturnValue(true);
    licenseStore.isLocked = false;
    licenseStore.showQuotaOverage = false;
    licenseStore.fetchBootstrap.mockResolvedValue({});
    configStore.isWhatsmeow = false;
    window.history.replaceState({}, "", "/");
  });

  it("redirects authenticated admins from /activate to the in-app license page", async () => {
    const { default: router } = await import("./index");

    await router.push("/activate");
    await router.isReady();

    expect(router.currentRoute.value.name).toBe("license-settings");
  });

  it("allows authenticated admins to open /settings/license", async () => {
    const { default: router } = await import("./index");

    await router.push("/settings/license");
    await router.isReady();

    expect(router.currentRoute.value.name).toBe("license-settings");
  });

  it("redirects authenticated users to cleanup mode when quota is over", async () => {
    const { default: router } = await import("./index");
    licenseStore.showQuotaOverage = true;

    await router.push("/chat");
    await router.isReady();

    expect(router.currentRoute.value.name).toBe("license-cleanup");
  });

  it("allows users with uploads cleanup permission to access /settings", async () => {
    const { default: router } = await import("./index");
    authStore.hasPermission.mockImplementation(
      (permission: string) => permission === "settings.uploads_cleanup",
    );

    await router.push("/settings");
    await router.isReady();

    expect(router.currentRoute.value.path).toBe("/settings");
  });
});
