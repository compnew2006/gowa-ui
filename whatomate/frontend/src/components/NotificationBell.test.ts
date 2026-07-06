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
  contactsService: {
    list: vi.fn(),
  },
  chatsService: {
    list: vi.fn(),
  },
  messagesService: {
    list: vi.fn(),
  },
  notificationsService: {
    list: vi.fn(),
    dismiss: vi.fn(),
  },
  wsService: {
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
  router: {
    push: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
  contactsService: mocks.contactsService,
  chatsService: mocks.chatsService,
  messagesService: mocks.messagesService,
  notificationsService: mocks.notificationsService,
}));

vi.mock("@/services/websocket", () => ({
  wsService: mocks.wsService,
}));

vi.mock("vue-sonner", () => ({
  toast: mocks.toast,
}));

vi.mock("vue-router", () => ({
  useRouter: () => mocks.router,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      switch (key) {
        case "chat.chatDeletedByUserNotification":
          return `${params?.user} deleted chat ${params?.chat}`;
        case "chat.unknownChat":
          return "Unknown chat";
        case "chat.unknownUser":
          return "Unknown user";
        default:
          return key;
      }
    },
  }),
}));

import NotificationBell from "./NotificationBell.vue";
import { useAuthStore } from "@/stores/auth";
import { useContactsStore } from "@/stores/contacts";

const ButtonStub = defineComponent({
  emits: ["click"],
  template: `<button v-bind="$attrs" @click="$emit('click', $event)"><slot /></button>`,
});

const passthroughStub = defineComponent({
  template: `<div v-bind="$attrs"><slot /></div>`,
});

function buildContact() {
  return {
    id: "contact-42",
    phone_number: "+12025550100",
    instance_id: "instance-1",
    name: "Alice",
    profile_name: "Alice",
    status: "open",
    tags: [],
    metadata: {},
    unread_count: 0,
    last_message_preview: "Latest message",
    last_message_at: "2026-04-07T09:00:00Z",
    created_at: "2026-04-07T08:00:00Z",
    updated_at: "2026-04-07T09:00:00Z",
  } as const;
}

function mountBell() {
  return mount(NotificationBell, {
    global: {
      stubs: {
        Badge: passthroughStub,
        Button: ButtonStub,
        Popover: passthroughStub,
        PopoverContent: passthroughStub,
        PopoverTrigger: passthroughStub,
        ScrollArea: passthroughStub,
        Bell: passthroughStub,
        Loader2: passthroughStub,
        X: passthroughStub,
      },
    },
  });
}

describe("NotificationBell soft-delete notifications", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();

    const authStore = useAuthStore();
    authStore.user = {
      id: "admin-1",
      email: "admin@example.com",
      full_name: "Admin",
      organization_id: "org-1",
      settings: {},
      role: {
        id: "role-1",
        name: "admin",
        permissions: ["contacts:soft_delete"],
      },
    } as any;

    const contactsStore = useContactsStore();
    contactsStore.contacts = [buildContact()] as any;

    mocks.notificationsService.list.mockResolvedValue({
      data: {
        data: {
          notifications: [
            {
              id: "notification-1",
              organization_id: "org-1",
              instance_id: "instance-1",
              event_type: "chat_deleted_by_user",
              message: "ignored backend message",
              is_dismissed: false,
              created_at: "2026-04-07T09:30:00Z",
              updated_at: "2026-04-07T09:30:00Z",
              metadata: {
                contact_id: "contact-42",
                contact_name: "Alice",
                contact_phone: "+12025550100",
                actor_name: "Moderator",
              },
            },
          ],
        },
      },
    });
  });

  it("formats chat_deleted_by_user messages and navigates using metadata contact ids", async () => {
    const wrapper = mountBell();
    await flushPromises();

    const message =
      "Moderator deleted chat Alice (+12025550100)";
    const notificationButton = wrapper
      .findAll("button")
      .find((candidate) => candidate.text().includes(message));

    expect(notificationButton).toBeDefined();
    expect(notificationButton?.text()).toContain(message);

    await notificationButton?.trigger("click");

    expect(mocks.router.push).toHaveBeenCalledWith("/chat/contact-42");
  });
});
