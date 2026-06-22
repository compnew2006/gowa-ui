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
  fetchConfig: vi.fn(async () => undefined),
  isModuleEnabled: vi.fn<(key: string) => boolean>(() => true),
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
    configStore.fetchConfig.mockResolvedValue(undefined);
    configStore.isModuleEnabled.mockReturnValue(true);
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

  it("redirects away from a disabled compiled module", async () => {
    const { default: router } = await import("./index");
    configStore.isModuleEnabled.mockImplementation(
      (key: string) => key !== "facebook-comments",
    );

    await router.push("/facebook/comments");
    await router.isReady();

    expect(configStore.fetchConfig).toHaveBeenCalled();
    expect(configStore.isModuleEnabled).toHaveBeenCalledWith(
      "facebook-comments",
    );
    expect(router.currentRoute.value.path).not.toBe("/facebook/comments");
  });

  it("allows an enabled compiled module route", async () => {
    const { default: router } = await import("./index");

    await router.push("/facebook/accounts");
    await router.isReady();

    expect(configStore.isModuleEnabled).toHaveBeenCalledWith(
      "facebook-accounts",
    );
    expect(router.currentRoute.value.path).toBe("/facebook/accounts");
  });

  it("allows admins to open module administration", async () => {
    const { default: router } = await import("./index");

    await router.push("/settings/modules");
    await router.isReady();

    expect(router.currentRoute.value.name).toBe("modules-settings");
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
