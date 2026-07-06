// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { defineComponent } from "vue";

function resetDocumentTheme() {
  document.documentElement.className = "";
  delete document.documentElement.dataset.themePreset;
  document.documentElement.style.colorScheme = "";
  localStorage.clear();
}

function stubMatchMedia(matches: boolean) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
    })),
  });
}

async function mountUseColorMode() {
  vi.resetModules();
  const mod = await import("./useColorMode");
  let api: any = null;

  mount(
    defineComponent({
      setup() {
        api = mod.useColorMode();
        return () => null;
      },
    }),
  );

  if (!api) {
    throw new Error("useColorMode did not initialize");
  }

  return api;
}

describe("useColorMode", () => {
  beforeEach(() => {
    resetDocumentTheme();
    stubMatchMedia(false);
  });

  it("boots from legacy localStorage and migrates caffeine to soft-pop", async () => {
    localStorage.setItem("color-mode", "dark");
    localStorage.setItem("theme-preset", "caffeine");

    const colorMode = await mountUseColorMode();

    expect(colorMode.colorMode.value).toBe("dark");
    expect(colorMode.themePreset.value).toBe("soft-pop");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("soft-pop");
  });

  it("previews an appearance change and restores the persisted values", async () => {
    const colorMode = await mountUseColorMode();

    colorMode.hydrateFromUserSettings({
      theme_mode: "light",
      theme_preset: "twitter",
    });
    colorMode.previewAppearance("dark", "ocean-breeze");

    expect(colorMode.hasUnsavedAppearance.value).toBe(true);
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("ocean-breeze");

    colorMode.restorePersistedAppearance();

    expect(colorMode.colorMode.value).toBe("light");
    expect(colorMode.themePreset.value).toBe("twitter");
    expect(colorMode.hasUnsavedAppearance.value).toBe(false);
    expect(document.documentElement.classList.contains("light")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("twitter");
  });

  it("resolves system mode from the active media query", async () => {
    stubMatchMedia(true);
    localStorage.setItem("color-mode", "system");
    localStorage.setItem("theme-preset", "twitter");

    const colorMode = await mountUseColorMode();

    expect(colorMode.colorMode.value).toBe("system");
    expect(colorMode.resolvedColorMode.value).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("falls back to default values for invalid saved settings", async () => {
    const colorMode = await mountUseColorMode();

    colorMode.hydrateFromUserSettings({
      theme_mode: "night" as any,
      theme_preset: "forest" as any,
    });

    expect(colorMode.colorMode.value).toBe("system");
    expect(colorMode.themePreset.value).toBe("twitter");
    expect(localStorage.getItem("color-mode")).toBe("system");
    expect(localStorage.getItem("theme-preset")).toBe("twitter");
  });
});
