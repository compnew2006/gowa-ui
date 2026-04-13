// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { defineComponent } from "vue";

const mocks = vi.hoisted(() => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
  usersService: {
    me: vi.fn(),
    updateSettings: vi.fn(),
    uploadChatBackground: vi.fn(),
  },
  organizationService: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    runUploadsCleanupNow: vi.fn(),
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
  chatSidebarUnifier: {
    readViewMode: vi.fn(),
    normalizeViewMode: vi.fn(),
    saveViewMode: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
  usersService: mocks.usersService,
  organizationService: mocks.organizationService,
}));

vi.mock("vue-sonner", () => ({
  toast: mocks.toast,
}));

vi.mock("vue-i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("vue-i18n")>();
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

vi.mock("@/lib/chat-sidebar-unifier", () => ({
  ChatSidebarUnifier: mocks.chatSidebarUnifier,
}));

import SettingsView from "./SettingsView.vue";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button v-bind="$attrs" @click="$emit('click', $event)"><slot /></button>`,
});

const InputStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number],
      default: "",
    },
    type: {
      type: String,
      default: "text",
    },
  },
  emits: ["update:modelValue", "change"],
  setup(props, { emit }) {
    function emitInputValue(event: Event) {
      const target = event.target as HTMLInputElement;
      const rawValue = target.value;
      if (props.type === "number") {
        emit("update:modelValue", rawValue === "" ? "" : Number(rawValue));
        return;
      }

      emit("update:modelValue", rawValue);
    }

    return { emitInputValue };
  },
  template: `<input v-bind="$attrs" :value="modelValue" :type="type" @input="emitInputValue" @change="$emit('change', $event)" />`,
});

const passthroughStub = defineComponent({
  template: `<div v-bind="$attrs"><slot /></div>`,
});

const RouterLinkStub = defineComponent({
  template: `<a v-bind="$attrs"><slot /></a>`,
});

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

function mountSettingsView() {
  return mount(SettingsView, {
    global: {
      mocks: {
        $t: (key: string) => key,
      },
      stubs: {
        RouterLink: RouterLinkStub,
        Button: ButtonStub,
        Input: InputStub,
        Label: passthroughStub,
        ScrollArea: passthroughStub,
        Separator: passthroughStub,
        Switch: passthroughStub,
        Tabs: passthroughStub,
        TabsContent: passthroughStub,
        TabsList: passthroughStub,
        TabsTrigger: passthroughStub,
        ToggleGroup: passthroughStub,
        ToggleGroupItem: passthroughStub,
        Select: passthroughStub,
        SelectContent: passthroughStub,
        SelectItem: passthroughStub,
        SelectTrigger: passthroughStub,
        SelectValue: passthroughStub,
        PageHeader: passthroughStub,
        LanguageSwitcher: passthroughStub,
        Settings: passthroughStub,
        Bell: passthroughStub,
        Loader2: passthroughStub,
        Globe: passthroughStub,
        Palette: passthroughStub,
        MessageSquare: passthroughStub,
        Play: passthroughStub,
        Archive: passthroughStub,
        ImageIcon: passthroughStub,
        LayoutGrid: passthroughStub,
        Upload: passthroughStub,
        CheckCircle2: passthroughStub,
        MoonStar: passthroughStub,
        SunMedium: passthroughStub,
        MonitorSmartphone: passthroughStub,
      },
    },
  });
}

describe("SettingsView appearance settings", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    resetDocumentTheme();
    stubMatchMedia(false);

    mocks.chatSidebarUnifier.readViewMode.mockReturnValue("unified");
    mocks.chatSidebarUnifier.normalizeViewMode.mockImplementation(
      (value: string) => value || "unified",
    );

    mocks.organizationService.getSettings.mockResolvedValue({
      data: {
        data: {
          name: "Whatomate",
          settings: {
            timezone: "UTC",
            date_format: "YYYY-MM-DD",
            mask_phone_numbers: false,
            uploads_cleanup_retention_days: 5,
            uploads_cleanup_schedule_hour: 3,
          },
        },
      },
    });
    mocks.organizationService.updateSettings.mockResolvedValue({ data: {} });
    mocks.organizationService.runUploadsCleanupNow.mockResolvedValue({
      data: {
        data: {
          message: "Uploads cleanup completed. Deleted 2 file(s).",
          deleted_files: 2,
          retention_days: 5,
        },
      },
    });
    mocks.usersService.me.mockResolvedValue({
      data: {
        data: {
          settings: {
            email_notifications: true,
            new_message_alerts: true,
            campaign_updates: true,
            notification_sound: "notification1",
            theme_mode: "dark",
            theme_preset: "ocean-breeze",
          },
        },
      },
    });
    mocks.usersService.updateSettings.mockResolvedValue({
      data: {
        data: {
          message: "Settings updated successfully",
          settings: {
            theme_mode: "light",
            theme_preset: "soft-pop",
          },
        },
      },
    });

    const authStore = useAuthStore();
    authStore.user = {
      id: "user-1",
      email: "user@example.com",
      full_name: "Test User",
      organization_id: "org-1",
      role: {
        id: "role-1",
        name: "admin",
        permissions: [
          "settings.general:read",
          "settings.general:write",
          "settings.uploads_cleanup:read",
          "settings.uploads_cleanup:write",
          "settings.uploads_cleanup:execute",
        ],
      },
      settings: {},
    };

    const configStore = useConfigStore();
    configStore.setShowPrintButtons(true);
    configStore.setShowDownloadButtons(true);
  });

  it("previews the selected appearance and saves the theme payload", async () => {
    const wrapper = mountSettingsView();
    await flushPromises();

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("ocean-breeze");

    await wrapper.get('[data-testid="appearance-mode-light"]').trigger("click");
    await wrapper
      .get('[data-testid="appearance-preset-soft-pop"]')
      .trigger("click");
    await flushPromises();

    expect(document.documentElement.classList.contains("light")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("soft-pop");

    await wrapper
      .get('[data-testid="settings-appearance-save"]')
      .trigger("click");
    await flushPromises();

    expect(mocks.usersService.updateSettings).toHaveBeenCalledWith({
      theme_mode: "light",
      theme_preset: "soft-pop",
    });
    expect(useAuthStore().user?.settings?.theme_mode).toBe("light");
    expect(useAuthStore().user?.settings?.theme_preset).toBe("soft-pop");
  });

  it("restores the persisted appearance when the view unmounts with unsaved changes", async () => {
    const wrapper = mountSettingsView();
    await flushPromises();

    await wrapper.get('[data-testid="appearance-mode-light"]').trigger("click");
    await wrapper
      .get('[data-testid="appearance-preset-twitter"]')
      .trigger("click");
    await flushPromises();

    expect(document.documentElement.classList.contains("light")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("twitter");

    wrapper.unmount();

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.dataset.themePreset).toBe("ocean-breeze");
  });

  it("saves uploads cleanup settings with retention and fixed schedule hour", async () => {
    const wrapper = mountSettingsView();
    await flushPromises();

    const retentionInput = wrapper.get(
      '[data-testid="uploads-cleanup-retention-days-input"]',
    );
    expect((retentionInput.element as HTMLInputElement).value).toBe("5");

    await retentionInput.setValue("7");
    await wrapper
      .get('[data-testid="uploads-cleanup-schedule-hour-input"]')
      .setValue("4");
    await wrapper.get('[data-testid="uploads-cleanup-save"]').trigger("click");
    await flushPromises();

    expect(mocks.organizationService.updateSettings).toHaveBeenCalledWith({
      uploads_cleanup_retention_days: 7,
      uploads_cleanup_schedule_hour: 4,
    });
  });

  it("runs uploads cleanup immediately after persisting the current form values", async () => {
    const wrapper = mountSettingsView();
    await flushPromises();

    await wrapper
      .get('[data-testid="uploads-cleanup-retention-days-input"]')
      .setValue("6");
    await wrapper
      .get('[data-testid="uploads-cleanup-schedule-hour-input"]')
      .setValue("5");
    await wrapper
      .get('[data-testid="uploads-cleanup-run-now"]')
      .trigger("click");
    await flushPromises();

    expect(mocks.organizationService.updateSettings).toHaveBeenCalledWith({
      uploads_cleanup_retention_days: 6,
      uploads_cleanup_schedule_hour: 5,
    });
    expect(
      mocks.organizationService.runUploadsCleanupNow,
    ).toHaveBeenCalledTimes(1);
    expect(mocks.toast.success).toHaveBeenCalledWith(
      "Uploads cleanup completed. Deleted 2 file(s).",
    );
  });
});
