// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { defineComponent } from "vue";

const mocks = vi.hoisted(() => ({
  modulesService: {
    listGlobal: vi.fn(),
    listOrganization: vi.fn(),
    updateGlobal: vi.fn(),
    updateOrganization: vi.fn(),
  },
  configStore: {
    fetchModules: vi.fn(),
  },
  authStore: {
    organizationId: "org-1",
    user: { is_super_admin: false },
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("@/services/modules", () => ({
  modulesService: mocks.modulesService,
}));

vi.mock("@/stores/config", () => ({
  useConfigStore: () => mocks.configStore,
}));

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => mocks.authStore,
}));

vi.mock("vue-sonner", () => ({
  toast: mocks.toast,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

import ModulesView from "./ModulesView.vue";

const ButtonStub = defineComponent({
  emits: ["click"],
  template:
    '<button v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>',
});
const passthroughStub = defineComponent({
  template: '<div v-bind="$attrs"><slot /></div>',
});

describe("ModulesView", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authStore.organizationId = "org-1";
    mocks.authStore.user = { is_super_admin: false };
    mocks.modulesService.listOrganization.mockResolvedValue({
      data: {
        data: [
          {
            key: "facebook-comments",
            display_name: "Facebook Comments",
            version: "1.0.0",
            dependencies: ["facebook-accounts"],
            global_enabled: true,
            organization_enabled: true,
            effective_enabled: true,
            technical: false,
          },
        ],
      },
    });
    mocks.modulesService.updateOrganization.mockResolvedValue({ data: {} });
  });

  it("updates organization module state and refreshes effective modules", async () => {
    const wrapper = mount(ModulesView, {
      global: {
        stubs: {
          Button: ButtonStub,
          Card: passthroughStub,
          CardContent: passthroughStub,
          CardDescription: passthroughStub,
          CardHeader: passthroughStub,
          CardTitle: passthroughStub,
          PageHeader: passthroughStub,
          Badge: passthroughStub,
          Loader2: passthroughStub,
          Boxes: passthroughStub,
        },
      },
    });
    await flushPromises();

    await wrapper
      .get('[data-testid="organization-toggle-facebook-comments"]')
      .trigger("click");
    await flushPromises();

    expect(mocks.modulesService.updateOrganization).toHaveBeenCalledWith(
      "org-1",
      "facebook-comments",
      false,
    );
    expect(mocks.configStore.fetchModules).toHaveBeenCalledWith(true);
    expect(mocks.modulesService.listOrganization).toHaveBeenCalledTimes(2);
  });
});
