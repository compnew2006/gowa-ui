// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

const mocks = vi.hoisted(() => ({
  contactsService: {
    list: vi.fn(),
    get: vi.fn(),
  },
  chatsService: {
    list: vi.fn(),
    claim: vi.fn(),
    close: vi.fn(),
    reopen: vi.fn(),
    setPublic: vi.fn(),
    listMessages: vi.fn(),
  },
  messagesService: {
    list: vi.fn(),
    send: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  contactsService: mocks.contactsService,
  chatsService: mocks.chatsService,
  messagesService: mocks.messagesService,
}));

import { useAuthStore } from "./auth";
import { useContactsStore, type Contact } from "./contacts";

function setAuthenticatedUser(roleName: string, overrides?: Partial<any>) {
  const authStore = useAuthStore();
  authStore.user = {
    id: "agent-1",
    email: "agent@example.com",
    full_name: "Agent One",
    organization_id: "org-1",
    role: {
      id: `${roleName}-role`,
      name: roleName,
      is_system: true,
      permissions: [],
    },
    settings: {
      send_restrictions: {
        allowed_instance_ids: ["instance-a"],
      },
    },
    ...overrides,
  };
  return authStore;
}

function makeContact(overrides?: Partial<Contact>): Contact {
  const now = "2026-03-09T10:00:00.000Z";
  return {
    id: overrides?.id ?? "contact-1",
    phone_number: overrides?.phone_number ?? "+15550000001",
    name: overrides?.name ?? "Contact One",
    profile_name: overrides?.profile_name ?? "Contact One",
    status: overrides?.status ?? "pending",
    tags: overrides?.tags ?? [],
    metadata: overrides?.metadata ?? {},
    unread_count: overrides?.unread_count ?? 0,
    created_at: overrides?.created_at ?? now,
    updated_at: overrides?.updated_at ?? now,
    ...overrides,
  };
}

describe("useContactsStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    mocks.chatsService.list.mockResolvedValue({
      data: { data: { contacts: [], total: 0 } },
    });
  });

  it("omits the implicit single-instance filter for assigned_to=me", async () => {
    setAuthenticatedUser("agent");
    const store = useContactsStore();

    await store.fetchAssignedChats({ assigned_to: "me" });

    expect(mocks.chatsService.list).toHaveBeenCalledTimes(1);
    expect(mocks.chatsService.list.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        status: "open",
        assigned_to: "me",
      }),
    );
    expect(mocks.chatsService.list.mock.calls[0][0].instance_id).toBeUndefined();

    await store.fetchPendingChats();

    expect(mocks.chatsService.list.mock.calls[1][0]).toEqual(
      expect.objectContaining({
        status: "pending",
        instance_id: "instance-a",
      }),
    );
  });

  it("renders a self-assigned chat even when it belongs to an out-of-scope instance", async () => {
    setAuthenticatedUser("agent");
    const store = useContactsStore();
    const assignedContact = makeContact({
      id: "contact-self-assigned",
      status: "open",
      instance_id: "instance-b",
      assigned_user_id: "agent-1",
      assigned_user_name: "Agent One",
    });

    mocks.chatsService.list.mockResolvedValueOnce({
      data: {
        data: {
          contacts: [assignedContact],
          total: 1,
        },
      },
    });

    await store.fetchAssignedChats({ assigned_to: "me" });

    expect(store.assignedChats).toHaveLength(1);
    expect(store.assignedChats[0].id).toBe("contact-self-assigned");
    expect(store.filteredContacts).toHaveLength(1);
    expect(store.filteredContacts[0].id).toBe("contact-self-assigned");
  });

  it("hides chats assigned to other agents while keeping own assigned chats visible", () => {
    setAuthenticatedUser("agent");
    const store = useContactsStore();
    store.contacts = [makeContact({ id: "contact-1", status: "pending" })];

    store.patchContact({
      id: "contact-1",
      status: "open",
      assigned_user_id: "agent-2",
      assigned_user_name: "Other Agent",
    });
    expect(store.assignedChats).toHaveLength(0);

    store.patchContact({
      id: "contact-1",
      status: "open",
      assigned_user_id: "agent-1",
      assigned_user_name: "Agent One",
    });
    expect(store.assignedChats).toHaveLength(1);
    expect(store.assignedChats[0].id).toBe("contact-1");
  });

  it("deduplicates concurrent fetchContact calls for the same id", async () => {
    setAuthenticatedUser("agent");
    const store = useContactsStore();
    const fetchedContact = makeContact({ id: "contact-fetch" });

    let resolveRequest!: (value: unknown) => void;
    mocks.contactsService.get.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveRequest = resolve;
      }),
    );

    const firstRequest = store.fetchContact("contact-fetch");
    const secondRequest = store.fetchContact("contact-fetch");

    expect(mocks.contactsService.get).toHaveBeenCalledTimes(1);
    expect(mocks.contactsService.get).toHaveBeenCalledWith("contact-fetch");

    resolveRequest({ data: { data: fetchedContact } });

    await expect(firstRequest).resolves.toEqual(
      expect.objectContaining({ id: "contact-fetch" }),
    );
    await expect(secondRequest).resolves.toEqual(
      expect.objectContaining({ id: "contact-fetch" }),
    );
  });

  it("reuses a recent fetchContact result during the cooldown window", async () => {
    setAuthenticatedUser("agent");
    const store = useContactsStore();
    const fetchedContact = makeContact({ id: "contact-cached" });

    mocks.contactsService.get.mockResolvedValueOnce({
      data: { data: fetchedContact },
    });

    await expect(store.fetchContact("contact-cached")).resolves.toEqual(
      expect.objectContaining({ id: "contact-cached" }),
    );
    await expect(store.fetchContact("contact-cached")).resolves.toEqual(
      expect.objectContaining({ id: "contact-cached" }),
    );

    expect(mocks.contactsService.get).toHaveBeenCalledTimes(1);
  });

  it("cools down repeated fetchContact retries after a 404 without logging an error", async () => {
    setAuthenticatedUser("agent");
    const store = useContactsStore();
    const consoleErrorSpy = vi
      .spyOn(console, "error")
      .mockImplementation(() => undefined);

    mocks.contactsService.get.mockRejectedValueOnce({
      isAxiosError: true,
      response: {
        status: 404,
      },
    });

    await expect(store.fetchContact("contact-missing")).resolves.toBeNull();
    await expect(store.fetchContact("contact-missing")).resolves.toBeNull();

    expect(mocks.contactsService.get).toHaveBeenCalledTimes(1);
    expect(consoleErrorSpy).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });
});
