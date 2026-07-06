// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { defineComponent } from "vue";

const mocks = vi.hoisted(() => ({
  api: {
    get: vi.fn(),
  },
  contactsService: {
    create: vi.fn(),
  },
  accountsService: {
    list: vi.fn(),
  },
  instancesService: {
    list: vi.fn(),
  },
  tagsService: {
    list: vi.fn(),
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
  contactsService: mocks.contactsService,
  accountsService: mocks.accountsService,
  instancesService: mocks.instancesService,
  tagsService: mocks.tagsService,
}));

vi.mock("vue-sonner", () => ({
  toast: mocks.toast,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

import CreateContactDialog from "./CreateContactDialog.vue";
import { useConfigStore } from "@/stores/config";

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button v-bind="$attrs" @click="$emit('click', $event)"><slot /></button>`,
});

const InputStub = defineComponent({
  props: {
    modelValue: {
      type: String,
      default: "",
    },
  },
  emits: ["update:modelValue"],
  template: `<input v-bind="$attrs" :value="modelValue" @input="$emit('update:modelValue', $event.target.value)" />`,
});

const passthroughStub = defineComponent({
  template: `<div v-bind="$attrs"><slot /></div>`,
});

describe("CreateContactDialog", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();

    const configStore = useConfigStore();
    configStore.config = {
      whatsapp_provider: "whatsmeow",
      features: {
        templates: false,
        flows: false,
        catalog: false,
        business_profile: false,
        campaigns: true,
        meta_insights: false,
      },
    };

    mocks.accountsService.list.mockResolvedValue({
      data: {
        data: {
          accounts: [],
        },
      },
    });
    mocks.instancesService.list.mockResolvedValue({
      data: {
        data: [
          {
            id: "instance-1",
            name: "Sales Line",
            status: "connected",
            is_default: true,
          },
        ],
      },
    });
    mocks.tagsService.list.mockResolvedValue({
      data: {
        data: {
          tags: [],
          total: 0,
          page: 1,
          limit: 100,
        },
      },
    });
    mocks.contactsService.create.mockResolvedValue({
      data: {
        data: {
          id: "contact-1",
        },
      },
    });
  });

  it("submits start_chat payload in whatsmeow chat mode", async () => {
    const wrapper = mount(CreateContactDialog, {
      props: {
        open: false,
        mode: "chat",
      },
      global: {
        mocks: {
          $t: (key: string) => key,
        },
        stubs: {
          Button: ButtonStub,
          Input: InputStub,
          Label: passthroughStub,
          Dialog: passthroughStub,
          DialogContent: passthroughStub,
          DialogHeader: passthroughStub,
          DialogTitle: passthroughStub,
          DialogDescription: passthroughStub,
          Select: passthroughStub,
          SelectContent: passthroughStub,
          SelectItem: passthroughStub,
          SelectTrigger: passthroughStub,
          SelectValue: passthroughStub,
          Popover: passthroughStub,
          PopoverContent: passthroughStub,
          PopoverTrigger: passthroughStub,
          Command: passthroughStub,
          CommandEmpty: passthroughStub,
          CommandGroup: passthroughStub,
          CommandInput: passthroughStub,
          CommandItem: passthroughStub,
          CommandList: passthroughStub,
          TagBadge: passthroughStub,
          Loader2: passthroughStub,
          Check: passthroughStub,
          ChevronsUpDown: passthroughStub,
          X: passthroughStub,
        },
      },
    });

    await wrapper.setProps({ open: true });
    await flushPromises();

    await wrapper
      .get('[data-testid="create-contact-phone"]')
      .setValue("+1 (202) 555-0100");
    await wrapper.get('[data-testid="create-contact-submit"]').trigger("click");
    await flushPromises();

    expect(mocks.contactsService.create).toHaveBeenCalledWith({
      phone_number: "+12025550100",
      profile_name: undefined,
      whatsapp_account: undefined,
      instance_id: "instance-1",
      start_chat: true,
      tags: undefined,
    });
    expect(wrapper.emitted("created")).toBeTruthy();
  });
});
