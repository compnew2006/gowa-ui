// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";

const mocks = vi.hoisted(() => ({
  instancesService: {
    list: vi.fn(),
    get: vi.fn(),
    health: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
    reconnect: vi.fn(),
    pairPhone: vi.fn(),
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  instancesService: mocks.instancesService,
}));

vi.mock("vue-sonner", () => ({
  toast: mocks.toast,
}));

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}));

import InstanceTag from "./InstanceTag.vue";
import { useInstancesStore } from "@/stores/instances";

describe("InstanceTag", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("renders the fallback label when the instance is not in the store", () => {
    const wrapper = mount(InstanceTag, {
      props: {
        instanceId: "instance-hidden",
        fallbackLabel: "201234567890",
        placement: "sidebar",
      },
    });

    expect(wrapper.text()).toContain("201234567890");
    expect(wrapper.attributes("title")).toBe("201234567890");
  });

  it("prefers the resolved instance label when the instance exists", () => {
    const instancesStore = useInstancesStore();
    instancesStore.instances = [
      {
        id: "instance-visible",
        name: "Sales Line",
        phone_number: "201111111111",
        status: "connected",
        is_default: false,
        auto_read_receipt: true,
        organization_id: "org-1",
        created_at: "2026-03-09T07:00:00.000Z",
        updated_at: "2026-03-09T07:00:00.000Z",
      },
    ];

    const wrapper = mount(InstanceTag, {
      props: {
        instanceId: "instance-visible",
        fallbackLabel: "201234567890",
        placement: "sidebar",
      },
    });

    expect(wrapper.text()).toContain("Sales Line");
    expect(wrapper.attributes("title")).toBe("Sales Line");
  });
});
