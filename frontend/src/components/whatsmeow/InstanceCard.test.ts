// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import type { WhatsAppInstance } from "@/types/whatsmeow";

const mocks = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
  },
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

import InstanceCard from "./InstanceCard.vue";

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button v-bind="$attrs" @click="$emit('click', $event)"><slot /></button>`,
});

const SwitchStub = defineComponent({
  inheritAttrs: false,
  props: {
    checked: {
      type: Boolean,
      default: false,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["update:checked"],
  template: `<button v-bind="$attrs" :disabled="disabled" @click="$emit('update:checked', !checked)"><slot /></button>`,
});

const passthroughStub = defineComponent({
  template: `<div v-bind="$attrs"><slot /></div>`,
});

function createInstance(
  settings: WhatsAppInstance["settings"] = {},
): WhatsAppInstance {
  return {
    id: "instance-1",
    name: "Primary",
    status: "connected",
    phone_number: "201000000000",
    jid: "201000000000@s.whatsapp.net",
    is_default: false,
    auto_read_receipt: true,
    organization_id: "org-1",
    settings,
    created_at: "2026-04-06T09:00:00Z",
    updated_at: "2026-04-06T09:00:00Z",
  };
}

function mountInstanceCard(instance: WhatsAppInstance) {
  return mount(InstanceCard, {
    props: {
      instance,
    },
    global: {
      mocks: {
        $t: (key: string) => key,
      },
      stubs: {
        Card: passthroughStub,
        CardHeader: passthroughStub,
        CardTitle: passthroughStub,
        CardDescription: passthroughStub,
        CardContent: passthroughStub,
        CardFooter: passthroughStub,
        Button: ButtonStub,
        Badge: passthroughStub,
        Switch: SwitchStub,
        InstanceTagSettings: passthroughStub,
        AutoRejectSettingsPanel: passthroughStub,
        AutoCampaignSettingsPanel: passthroughStub,
        InstanceChatCloseRatingPanel: passthroughStub,
        InstanceAssignedChatResetPanel: passthroughStub,
        Loader2: passthroughStub,
        Power: passthroughStub,
        Trash2: passthroughStub,
        Smartphone: passthroughStub,
        QrCode: passthroughStub,
        Pencil: passthroughStub,
      },
    },
  });
}

describe("InstanceCard quick setting toggles", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("blocks enabling auto campaign without a message", async () => {
    const wrapper = mountInstanceCard(
      createInstance({
        auto_campaign: {
          enabled: false,
          message: "",
          interval_days: 7,
          min_delay_minutes: 1,
          max_delay_minutes: 3,
          target_status: "draft",
        },
      }),
    );

    await wrapper.get('[data-testid="instance-auto-campaign-toggle"]').trigger("click");

    expect(mocks.toast.error).toHaveBeenCalledWith(
      "instances.auto_campaign.validation.messageRequired",
    );
    expect(wrapper.emitted("update-auto-campaign-settings")).toBeUndefined();
  });

  it("emits auto campaign updates when the payload is valid", async () => {
    const wrapper = mountInstanceCard(
      createInstance({
        auto_campaign: {
          enabled: false,
          message: "Hello {{name}}",
          interval_days: 7,
          min_delay_minutes: 1,
          max_delay_minutes: 3,
          target_status: "draft",
        },
      }),
    );

    await wrapper.get('[data-testid="instance-auto-campaign-toggle"]').trigger("click");

    expect(mocks.toast.error).not.toHaveBeenCalled();
    expect(wrapper.emitted("update-auto-campaign-settings")).toEqual([
      [
        "instance-1",
        expect.objectContaining({
          enabled: true,
          message: "Hello {{name}}",
          interval_days: 7,
        }),
      ],
    ]);
  });

  it("blocks enabling auto reject with message mode and an empty reply", async () => {
    const wrapper = mountInstanceCard(
      createInstance({
        auto_reject_calls: {
          enabled: false,
          mode: "with_message",
          message: "",
          reject_individual_calls: true,
          reject_group_calls: true,
          bypass_contacts: [],
          schedule: {
            type: "always",
            start: "09:00",
            end: "18:00",
            days: [1, 2, 3, 4, 5],
            timezone: "UTC",
          },
        },
      }),
    );

    await wrapper.get('[data-testid="instance-auto-reject-toggle"]').trigger("click");

    expect(mocks.toast.error).toHaveBeenCalledWith(
      "instances.auto_reject.validation.messageRequired",
    );
    expect(wrapper.emitted("update-auto-reject-settings")).toBeUndefined();
  });
});
