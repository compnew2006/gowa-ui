// @vitest-environment happy-dom

import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { defineComponent } from "vue";
import type { Contact } from "@/types/contacts";

const mocks = vi.hoisted(() => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
  contactsService: {
    listCollaborators: vi.fn(),
    softDelete: vi.fn(),
  },
  tagsService: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
  usersService: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
  wsService: {
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
  contactsService: mocks.contactsService,
  tagsService: mocks.tagsService,
  usersService: mocks.usersService,
}));

vi.mock("@/services/websocket", () => ({
  wsService: mocks.wsService,
}));

vi.mock("vue-sonner", () => ({
  toast: mocks.toast,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "en" },
    t: (key: string) => key,
  }),
}));

vi.mock("@/i18n/locale-direction", () => ({
  localeDirectionManager: {
    isRTL: () => false,
  },
}));

vi.mock("@/lib/instance-access", () => ({
  canUserAccessInstance: () => true,
}));

import ContactInfoPanel from "./ContactInfoPanel.vue";
import { useAuthStore } from "@/stores/auth";
import { useTagsStore } from "@/stores/tags";
import { useUsersStore } from "@/stores/users";

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

const commandItemStub = defineComponent({
  emits: ["select"],
  template: `<div v-bind="$attrs" @click="$emit('select', $event)"><slot /></div>`,
});

function buildContact(): Contact {
  return {
    id: "contact-1",
    phone_number: "+12025550100",
    instance_id: "instance-1",
    name: "Alice",
    profile_name: "Alice",
    status: "open",
    tags: [],
    metadata: {},
    unread_count: 0,
    created_at: "2026-04-07T08:00:00Z",
    updated_at: "2026-04-07T08:05:00Z",
  };
}

function mountPanel() {
  return mount(ContactInfoPanel, {
    props: {
      contact: buildContact(),
      sessionData: null,
    },
    global: {
      mocks: {
        $t: (key: string) => key,
      },
      stubs: {
        Avatar: passthroughStub,
        AvatarFallback: passthroughStub,
        AvatarImage: passthroughStub,
        Badge: passthroughStub,
        Button: ButtonStub,
        Input: InputStub,
        ScrollArea: passthroughStub,
        Collapsible: passthroughStub,
        CollapsibleContent: passthroughStub,
        CollapsibleTrigger: passthroughStub,
        Dialog: passthroughStub,
        DialogContent: passthroughStub,
        DialogDescription: passthroughStub,
        DialogHeader: passthroughStub,
        DialogTitle: passthroughStub,
        Popover: passthroughStub,
        PopoverContent: passthroughStub,
        PopoverTrigger: passthroughStub,
        Command: passthroughStub,
        CommandEmpty: passthroughStub,
        CommandGroup: passthroughStub,
        CommandInput: InputStub,
        CommandItem: commandItemStub,
        CommandList: passthroughStub,
        TagBadge: passthroughStub,
        MetadataSection: passthroughStub,
        X: passthroughStub,
        ChevronDown: passthroughStub,
        Phone: passthroughStub,
        User: passthroughStub,
        Plus: passthroughStub,
        Check: passthroughStub,
        Tags: passthroughStub,
        Loader2: passthroughStub,
        Trash2: passthroughStub,
        Archive: passthroughStub,
        UserPlus: passthroughStub,
        Search: passthroughStub,
      },
    },
  });
}

describe("ContactInfoPanel soft delete", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    vi.stubGlobal("confirm", vi.fn().mockReturnValue(true));

    mocks.contactsService.listCollaborators.mockResolvedValue({
      data: {
        data: {
          collaborators: [],
        },
      },
    });
    mocks.contactsService.softDelete.mockResolvedValue({
      data: {
        success: true,
      },
    });

    const authStore = useAuthStore();
    authStore.user = {
      id: "user-1",
      email: "agent@example.com",
      full_name: "Agent",
      organization_id: "org-1",
      settings: {},
      role: {
        id: "role-1",
        name: "agent",
        permissions: [],
      },
    } as any;

    const tagsStore = useTagsStore();
    tagsStore.tags = [{ name: "vip", color: "green" }] as any;

    const usersStore = useUsersStore();
    usersStore.users = [];
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("hides the soft-delete action when the user lacks permission", async () => {
    const wrapper = mountPanel();
    await flushPromises();

    expect(wrapper.find('button[title="chat.softDeleteChat"]').exists()).toBe(
      false,
    );
  });

  it("calls the soft-delete endpoint and emits deletion when permitted", async () => {
    const authStore = useAuthStore();
    authStore.user = {
      ...authStore.user,
      role: {
        id: "role-1",
        name: "agent",
        permissions: ["contacts:soft_delete"],
      },
    } as any;

    const wrapper = mountPanel();
    await flushPromises();

    await wrapper.get('button[title="chat.softDeleteChat"]').trigger("click");
    await flushPromises();

    expect(window.confirm).toHaveBeenCalledWith("chat.softDeleteConfirm");
    expect(mocks.contactsService.softDelete).toHaveBeenCalledWith("contact-1");
    expect(mocks.toast.success).toHaveBeenCalledWith("chat.softDeleteSuccess");
    expect(wrapper.emitted("deleted")).toEqual([["contact-1"]]);
  });
});
