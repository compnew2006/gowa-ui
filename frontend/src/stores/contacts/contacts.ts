import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { contactsService, chatsService } from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import { unwrapResponse } from "@/lib/api-utils";
import type { ChatBucketTab, ChatStatus, Contact } from "@/types/contacts";
import {
  type ContactsListPayload,
  type RecentContactFetch,
  normalizeContact,
  normalizeContacts,
  normalizeChatStatus,
  extractAllowedInstanceIDsFromUserSettings,
  contactMatchesSearch,
  contactFetchCooldownMs,
  missingContactFetchCooldownMs,
} from "./helpers";
import { useChatFiltersStore } from "./chat-filters";

export const useContactsStore = defineStore("contacts", () => {
  const authStore = useAuthStore();
  const filtersStore = useChatFiltersStore();

  const contacts = ref<Contact[]>([]);
  const pendingChats = ref<Contact[]>([]);
  const assignedChats = ref<Contact[]>([]);
  const closedChats = ref<Contact[]>([]);
  const activeChatTab = ref<ChatBucketTab>("assigned");
  const currentContact = ref<Contact | null>(null);
  const isLoading = ref(false);
  const contactsPage = ref(1);
  const contactsLimit = ref(50);
  const contactsTotal = ref(0);
  const pendingChatsTotal = ref(0);
  const assignedChatsTotal = ref(0);
  const isLoadingMoreContacts = ref(false);
  const assignedChatsAssignedToFilter = ref<"me" | string | undefined>(
    undefined,
  );
  const inFlightContactFetches = new Map<string, Promise<Contact | null>>();
  const recentContactFetches = new Map<string, RecentContactFetch>();
  let fetchChatsSequence = 0;

  const restrictedAllowedInstanceIDs = computed(() =>
    extractAllowedInstanceIDsFromUserSettings(authStore.user?.settings),
  );
  const effectiveInstanceFilterID = computed(() => {
    const selected = filtersStore.selectedInstanceId.trim();
    if (selected !== "") {
      return selected;
    }
    return restrictedAllowedInstanceIDs.value.length === 1
      ? restrictedAllowedInstanceIDs.value[0]
      : "";
  });
  const isAgentRole = computed(() => {
    if (authStore.user?.is_super_admin === true) return false;
    return (authStore.userRole || "").trim().toLowerCase() === "agent";
  });
  const currentUserID = computed(() => authStore.user?.id || "");

  function resolveListInstanceFilter(options?: {
    allowImplicitRestrictedDefault?: boolean;
  }): string | undefined {
    const selected = filtersStore.selectedInstanceId.trim();
    if (selected !== "") {
      return selected;
    }
    if (options?.allowImplicitRestrictedDefault === false) {
      return undefined;
    }
    return effectiveInstanceFilterID.value || undefined;
  }

  function buildListParams(options?: {
    allowImplicitRestrictedDefault?: boolean;
  }) {
    return {
      tags:
        filtersStore.selectedTags.length > 0
          ? filtersStore.selectedTags.join(",")
          : undefined,
      instance_id: resolveListInstanceFilter(options),
      chat_types:
        filtersStore.selectedChatTypes.length > 0
          ? filtersStore.selectedChatTypes.join(",")
          : undefined,
    };
  }

  function isVisibleAssignedChatForCurrentUser(contact: Contact) {
    if (contact.status === "closed") {
      return false;
    }
    if (contact.is_collaborator) {
      return true;
    }
    if (contact.status !== "open" || !contact.assigned_user_id) {
      return false;
    }
    if (!isAgentRole.value) {
      return true;
    }
    return (
      contact.is_public === true ||
      contact.assigned_user_id === currentUserID.value
    );
  }

  function shouldBypassImplicitRestrictedInstanceFilter(contact: Contact) {
    if (filtersStore.selectedInstanceId.trim() !== "") {
      return false;
    }
    if (activeChatTab.value !== "assigned") {
      return false;
    }
    if (assignedChatsAssignedToFilter.value !== "me") {
      return false;
    }
    return (
      typeof contact.assigned_user_id === "string" &&
      contact.assigned_user_id === currentUserID.value
    );
  }

  const hasMoreContacts = computed(() => {
    const activeCount =
      activeChatTab.value === "assigned"
        ? assignedChats.value.length
        : pendingChats.value.length;
    return activeCount < contactsTotal.value;
  });

  function rebuildChatBucketsFromContacts() {
    pendingChats.value = contacts.value.filter(
      (c) =>
        c.status === "pending" &&
        !c.assigned_user_id &&
        c.is_collaborator !== true,
    );
    assignedChats.value = contacts.value.filter((c) =>
      isVisibleAssignedChatForCurrentUser(c),
    );
  }

  function mergeContactsIntoStore(nextContacts: Contact[]) {
    const normalized = normalizeContacts(nextContacts);
    const merged = new Map<string, Contact>();

    for (const existing of contacts.value) {
      merged.set(existing.id, existing);
    }

    for (const next of normalized) {
      const current = merged.get(next.id);
      merged.set(next.id, current ? { ...current, ...next } : next);
    }

    contacts.value = Array.from(merged.values());
    rebuildChatBucketsFromContacts();
  }

  function upsertContact(contact: Contact) {
    mergeContactsIntoStore([contact]);
    if (currentContact.value?.id === contact.id) {
      currentContact.value = {
        ...currentContact.value,
        ...normalizeContact(contact),
      };
    }
  }

  function replaceContacts(nextContacts: Contact[]) {
    contacts.value = normalizeContacts(nextContacts);
    rebuildChatBucketsFromContacts();
  }

  const activeTabContacts = computed(() => {
    if (activeChatTab.value === "assigned") return assignedChats.value;
    return pendingChats.value;
  });

  const searchedContacts = computed(() => {
    const trimmedQuery = filtersStore.searchQuery.trim();
    if (!trimmedQuery) return activeTabContacts.value;

    const merged = new Map<string, Contact>();
    for (const contact of activeTabContacts.value) {
      merged.set(contact.id, contact);
    }
    for (const contact of closedChats.value) {
      merged.set(contact.id, contact);
    }

    return Array.from(merged.values()).filter((c) =>
      contactMatchesSearch(c, trimmedQuery),
    );
  });

  function getConversationId(contact: Contact): string {
    return (
      contact.conversation_id ||
      (typeof contact.metadata?.group_jid === "string"
        ? contact.metadata.group_jid
        : "") ||
      (typeof contact.metadata?.channel_jid === "string"
        ? contact.metadata.channel_jid
        : "") ||
      (contact.phone_number.endsWith("@g.us") ||
      contact.phone_number.endsWith("@newsletter")
        ? contact.phone_number
        : "")
    );
  }

  function isGroupConversation(contact: Contact): boolean {
    const conversationId = getConversationId(contact);
    return Boolean(
      contact.is_group_chat === true ||
      contact.metadata?.is_group_chat === true ||
      (conversationId && conversationId.endsWith("@g.us")),
    );
  }

  function matchesActiveFilters(contact: Contact): boolean {
    const explicitInstanceFilterID = filtersStore.selectedInstanceId.trim();
    if (
      explicitInstanceFilterID &&
      contact.instance_id !== explicitInstanceFilterID
    ) {
      return false;
    }
    if (
      !explicitInstanceFilterID &&
      !shouldBypassImplicitRestrictedInstanceFilter(contact) &&
      restrictedAllowedInstanceIDs.value.length > 0
    ) {
      const instanceID =
        typeof contact.instance_id === "string"
          ? contact.instance_id.trim()
          : "";
      if (
        !instanceID ||
        !restrictedAllowedInstanceIDs.value.includes(instanceID)
      ) {
        return false;
      }
    }

    return true;
  }

  const filteredContacts = computed(() => {
    const grouped = new Map<string, Contact>();
    for (const contact of searchedContacts.value) {
      if (!matchesActiveFilters(contact)) {
        continue;
      }

      const conversationId = getConversationId(contact);
      const isGroupChat = isGroupConversation(contact);

      const groupKey =
        isGroupChat && conversationId
          ? `group:${conversationId}:${contact.instance_id || "no-instance"}`
          : `contact:${contact.id}`;

      const existing = grouped.get(groupKey);
      if (!existing) {
        grouped.set(groupKey, { ...contact });
        continue;
      }

      const existingTime = existing.last_message_at
        ? new Date(existing.last_message_at).getTime()
        : 0;
      const contactTime = contact.last_message_at
        ? new Date(contact.last_message_at).getTime()
        : 0;
      const latest = contactTime >= existingTime ? contact : existing;

      grouped.set(groupKey, {
        ...existing,
        ...latest,
        conversation_id:
          conversationId || latest.conversation_id || existing.conversation_id,
        is_group_chat:
          isGroupChat || latest.is_group_chat || existing.is_group_chat,
        is_public:
          existing.is_public === true ||
          latest.is_public === true ||
          contact.is_public === true,
        unread_count:
          (existing.unread_count || 0) + (contact.unread_count || 0),
        tags: Array.from(
          new Set([...(existing.tags || []), ...(contact.tags || [])]),
        ),
      });
    }

    return Array.from(grouped.values());
  });

  const sortedContacts = computed(() => {
    return [...filteredContacts.value].sort((a, b) => {
      const publicA = a.is_public === true ? 1 : 0;
      const publicB = b.is_public === true ? 1 : 0;
      if (publicA !== publicB) {
        return publicB - publicA;
      }
      const dateA = a.last_message_at
        ? new Date(a.last_message_at).getTime()
        : 0;
      const dateB = b.last_message_at
        ? new Date(b.last_message_at).getTime()
        : 0;
      return dateB - dateA;
    });
  });

  function setActiveChatTab(tab: ChatBucketTab) {
    activeChatTab.value = tab;
    contactsPage.value = 1;
    contactsTotal.value =
      tab === "assigned" ? assignedChatsTotal.value : pendingChatsTotal.value;
  }

  async function fetchContacts(params?: {
    search?: string;
    page?: number;
    limit?: number;
    tags?: string;
    instance_id?: string;
    chat_types?: string;
    status?: ChatStatus;
    assigned_to?: "me" | "unassigned" | string;
  }) {
    isLoading.value = true;
    try {
      const response = await contactsService.list({
        ...buildListParams(),
        page: params?.page ?? 1,
        limit: contactsLimit.value,
        ...params,
      });
      const data = unwrapResponse<ContactsListPayload>(response);
      replaceContacts(data.contacts || []);
      contactsTotal.value = data.total ?? contacts.value.length;
      pendingChatsTotal.value = pendingChats.value.length;
      assignedChatsTotal.value = assignedChats.value.length;
      contactsPage.value = params?.page ?? 1;
    } catch (error) {
      console.error("Failed to fetch contacts:", error);
    } finally {
      isLoading.value = false;
    }
  }

  async function fetchChats(params?: {
    search?: string;
    limit?: number;
    assigned_to?: "me" | string;
  }) {
    const requestSequence = ++fetchChatsSequence;
    isLoading.value = true;
    try {
      assignedChatsAssignedToFilter.value = params?.assigned_to;
      const trimmedSearch =
        typeof params?.search === "string" ? params.search.trim() : "";
      const includeClosedInSearch = trimmedSearch !== "";
      const closedSearchLimit = Math.max(
        params?.limit ?? contactsLimit.value,
        500,
      );
      const pendingListParams = {
        ...buildListParams(),
        search: trimmedSearch || undefined,
        page: 1,
        limit: params?.limit ?? contactsLimit.value,
      };
      const assignedListParams = {
        ...buildListParams({
          allowImplicitRestrictedDefault: params?.assigned_to !== "me",
        }),
        search: trimmedSearch || undefined,
        page: 1,
        limit: params?.limit ?? contactsLimit.value,
      };

      const [pendingResponse, assignedResponse, closedResponse] =
        await Promise.all([
          chatsService.list({
            ...pendingListParams,
            status: "pending",
          }),
          chatsService.list({
            ...assignedListParams,
            status: "open",
            assigned_to: params?.assigned_to,
          }),
          includeClosedInSearch
            ? chatsService.list({
                ...pendingListParams,
                limit: closedSearchLimit,
                status: "closed",
              })
            : Promise.resolve(null),
        ]);

      if (requestSequence !== fetchChatsSequence) {
        return;
      }

      const pendingData = unwrapResponse<ContactsListPayload>(pendingResponse);
      const assignedData = unwrapResponse<ContactsListPayload>(assignedResponse);
      const pendingList = normalizeContacts(pendingData.contacts || []);
      const assignedList = normalizeContacts(assignedData.contacts || []);
      pendingChatsTotal.value = pendingData.total ?? pendingList.length;
      assignedChatsTotal.value = assignedData.total ?? assignedList.length;
      const searchedClosed =
        includeClosedInSearch && closedResponse
          ? normalizeContacts(
              unwrapResponse<ContactsListPayload>(closedResponse).contacts || [],
            )
          : null;
      if (searchedClosed) {
        closedChats.value = searchedClosed;
      }

      const retainedClosed =
        searchedClosed ?? contacts.value.filter((c) => c.status === "closed");
      const merged = new Map<string, Contact>();
      for (const item of retainedClosed) {
        merged.set(item.id, item);
      }
      for (const item of [...pendingList, ...assignedList]) {
        merged.set(item.id, item);
      }

      contacts.value = Array.from(merged.values());
      rebuildChatBucketsFromContacts();
      contactsTotal.value =
        activeChatTab.value === "assigned"
          ? assignedChatsTotal.value
          : pendingChatsTotal.value;
      contactsPage.value = 1;
    } catch (error) {
      if (requestSequence !== fetchChatsSequence) {
        return;
      }
      console.error("Failed to fetch chats:", error);
    } finally {
      if (requestSequence === fetchChatsSequence) {
        isLoading.value = false;
      }
    }
  }

  async function fetchPendingChats(params?: {
    search?: string;
    limit?: number;
    page?: number;
    append?: boolean;
  }) {
    isLoading.value = true;
    try {
      const response = await chatsService.list({
        ...buildListParams(),
        search: params?.search,
        page: params?.page ?? 1,
        limit: params?.limit ?? contactsLimit.value,
        status: "pending",
      });
      const data = unwrapResponse<ContactsListPayload>(response);
      const nextPending = normalizeContacts(data.contacts || []);

      mergeContactsIntoStore(nextPending);
      pendingChats.value = params?.append
        ? [...pendingChats.value, ...nextPending]
        : nextPending;
      pendingChatsTotal.value = data.total ?? nextPending.length;
      if (activeChatTab.value === "pending") {
        contactsTotal.value = pendingChatsTotal.value;
      }
      return nextPending;
    } catch (error) {
      console.error("Failed to fetch pending chats:", error);
      return [];
    } finally {
      isLoading.value = false;
    }
  }

  async function fetchAssignedChats(params?: {
    search?: string;
    limit?: number;
    page?: number;
    append?: boolean;
    assigned_to?: "me" | string;
  }) {
    isLoading.value = true;
    try {
      assignedChatsAssignedToFilter.value = params?.assigned_to;
      const response = await chatsService.list({
        ...buildListParams({
          allowImplicitRestrictedDefault: params?.assigned_to !== "me",
        }),
        search: params?.search,
        page: params?.page ?? 1,
        limit: params?.limit ?? contactsLimit.value,
        status: "open",
        assigned_to: params?.assigned_to,
      });
      const data = unwrapResponse<ContactsListPayload>(response);
      const nextAssigned = normalizeContacts(data.contacts || []);

      mergeContactsIntoStore(nextAssigned);
      assignedChats.value = params?.append
        ? [...assignedChats.value, ...nextAssigned]
        : nextAssigned;
      assignedChatsTotal.value = data.total ?? nextAssigned.length;
      if (activeChatTab.value === "assigned") {
        contactsTotal.value = assignedChatsTotal.value;
      }
      return nextAssigned;
    } catch (error) {
      console.error("Failed to fetch assigned chats:", error);
      return [];
    } finally {
      isLoading.value = false;
    }
  }

  async function loadMoreContacts() {
    if (isLoadingMoreContacts.value || !hasMoreContacts.value) return;

    isLoadingMoreContacts.value = true;
    try {
      const nextPage = contactsPage.value + 1;
      const response = await chatsService.list({
        ...buildListParams({
          allowImplicitRestrictedDefault:
            !(
              activeChatTab.value === "assigned" &&
              assignedChatsAssignedToFilter.value === "me"
            ),
        }),
        search: filtersStore.searchQuery || undefined,
        page: nextPage,
        limit: contactsLimit.value,
        status: activeChatTab.value === "assigned" ? "open" : "pending",
        assigned_to:
          activeChatTab.value === "assigned"
            ? assignedChatsAssignedToFilter.value
            : undefined,
      });
      const data = unwrapResponse<ContactsListPayload>(response);
      const newContacts = normalizeContacts(data.contacts || []);

      if (newContacts.length > 0) {
        mergeContactsIntoStore(newContacts);
        contactsPage.value = nextPage;
      }
      if (activeChatTab.value === "assigned") {
        assignedChatsTotal.value = data.total ?? assignedChatsTotal.value;
      } else {
        pendingChatsTotal.value = data.total ?? pendingChatsTotal.value;
      }
      contactsTotal.value =
        activeChatTab.value === "assigned"
          ? assignedChatsTotal.value
          : pendingChatsTotal.value;
    } catch (error) {
      console.error("Failed to load more contacts:", error);
    } finally {
      isLoadingMoreContacts.value = false;
    }
  }

  async function fetchContact(id: string) {
    const normalizedID = id.trim();
    if (!normalizedID) return null;

    const existingRequest = inFlightContactFetches.get(normalizedID);
    if (existingRequest) {
      return existingRequest;
    }

    const recentFetch = recentContactFetches.get(normalizedID);
    if (recentFetch && Date.now() - recentFetch.at < recentFetch.cooldownMs) {
      if (recentFetch.result) {
        if (currentContact.value?.id === normalizedID) {
          return currentContact.value;
        }
        return (
          contacts.value.find((contact) => contact.id === normalizedID) ??
          recentFetch.result
        );
      }
      return null;
    }

    const request = (async () => {
      try {
        const response = await contactsService.get(normalizedID);
        const data = normalizeContact(unwrapResponse<Contact>(response));
        upsertContact(data);
        if (currentContact.value?.id === normalizedID) {
          currentContact.value = data;
        }
        recentContactFetches.set(normalizedID, {
          at: Date.now(),
          cooldownMs: contactFetchCooldownMs,
          result: data,
        });
        return data;
      } catch (error) {
        const errorStatus =
          typeof error === "object" &&
          error !== null &&
          "isAxiosError" in error &&
          (error as { isAxiosError?: boolean }).isAxiosError === true
            ? (error as { response?: { status?: number } }).response?.status
            : undefined;
        if (errorStatus !== 404) {
          console.error("Failed to fetch contact:", error);
        }
        recentContactFetches.set(normalizedID, {
          at: Date.now(),
          cooldownMs:
            errorStatus === 404
              ? missingContactFetchCooldownMs
              : contactFetchCooldownMs,
          result: null,
        });
        return null;
      } finally {
        inFlightContactFetches.delete(normalizedID);
      }
    })();

    inFlightContactFetches.set(normalizedID, request);
    return request;
  }

  async function fetchClosedChats(params?: {
    search?: string;
    page?: number;
    limit?: number;
    closed_by?: string;
    closed_from?: string;
    closed_to?: string;
    instance_id?: string;
  }) {
    isLoading.value = true;
    try {
      const response = await chatsService.list({
        search: params?.search,
        page: params?.page ?? 1,
        limit: params?.limit ?? contactsLimit.value,
        status: "closed",
        closed_by: params?.closed_by,
        closed_from: params?.closed_from,
        closed_to: params?.closed_to,
        instance_id: params?.instance_id,
      });
      const data = unwrapResponse<ContactsListPayload>(response);
      const nextClosed = normalizeContacts(data.contacts || []);

      mergeContactsIntoStore(nextClosed);
      closedChats.value = nextClosed;
      return {
        chats: nextClosed,
        total: data.total ?? nextClosed.length,
        page: data.page ?? params?.page ?? 1,
        limit: data.limit ?? params?.limit ?? contactsLimit.value,
      };
    } catch (error) {
      console.error("Failed to fetch closed chats:", error);
      closedChats.value = [];
      return {
        chats: [],
        total: 0,
        page: params?.page ?? 1,
        limit: params?.limit ?? contactsLimit.value,
      };
    } finally {
      isLoading.value = false;
    }
  }

  async function claimChat(chatId: string) {
    try {
      const response = await chatsService.claim(chatId);
      const updated = normalizeContact(unwrapResponse<Contact>(response));
      upsertContact(updated);
      return updated;
    } catch (error) {
      console.error("Failed to claim chat:", error);
      return null;
    }
  }

  async function closeChat(chatId: string) {
    try {
      const response = await chatsService.close(chatId);
      const updated = normalizeContact(unwrapResponse<Contact>(response));
      upsertContact(updated);
      return updated;
    } catch (error) {
      console.error("Failed to close chat:", error);
      return null;
    }
  }

  async function reopenChat(chatId: string) {
    try {
      const response = await chatsService.reopen(chatId);
      const updated = normalizeContact(unwrapResponse<Contact>(response));
      upsertContact(updated);
      return updated;
    } catch (error) {
      console.error("Failed to reopen chat:", error);
      return null;
    }
  }

  async function setChatPublic(chatId: string, isPublic: boolean) {
    try {
      const response = await chatsService.setPublic(chatId, isPublic);
      const updated = normalizeContact(unwrapResponse<Contact>(response));
      upsertContact(updated);
      return updated;
    } catch (error) {
      console.error("Failed to update chat public state:", error);
      return null;
    }
  }

  function patchContact(updatedContact: Partial<Contact> & { id: string }) {
    const normalizedPartial: Partial<Contact> & { id: string } = {
      ...updatedContact,
    };
    if (
      updatedContact.status !== undefined ||
      updatedContact.assigned_user_id !== undefined
    ) {
      normalizedPartial.status = normalizeChatStatus(
        updatedContact.status,
        updatedContact.assigned_user_id,
      );
    }

    const index = contacts.value.findIndex((c) => c.id === updatedContact.id);
    if (index !== -1) {
      contacts.value[index] = {
        ...contacts.value[index],
        ...normalizedPartial,
      };
      rebuildChatBucketsFromContacts();
    }

    if (currentContact.value?.id === updatedContact.id) {
      currentContact.value = {
        ...currentContact.value,
        ...normalizedPartial,
      };
    }
  }

  function setCurrentContact(contact: Contact | null) {
    currentContact.value = contact ? normalizeContact(contact) : null;
    if (currentContact.value) {
      currentContact.value.unread_count = 0;
    }
  }

  async function markConversationAsRead(contactId: string) {
    if (!contactId) return;

    try {
      const { messagesService } = await import("@/services/api");
      await messagesService.list(contactId, { limit: 1 });
    } catch (error) {
      console.error("Failed to mark conversation as read:", error);
    }

    const contact = contacts.value.find((c) => c.id === contactId);
    if (contact) {
      contact.unread_count = 0;
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value.unread_count = 0;
    }
  }

  function updateContactTags(contactId: string, tags: string[]) {
    const contact = contacts.value.find((c) => c.id === contactId);
    if (contact) {
      contact.tags = tags;
    }
    if (currentContact.value?.id === contactId) {
      currentContact.value = { ...currentContact.value, tags };
    }
  }

  return {
    contacts,
    pendingChats,
    assignedChats,
    closedChats,
    activeChatTab,
    currentContact,
    isLoading,
    contactsTotal,
    contactsPage,
    contactsLimit,
    hasMoreContacts,
    isLoadingMoreContacts,
    filteredContacts,
    sortedContacts,
    searchedContacts,
    activeTabContacts,
    setActiveChatTab,
    fetchContacts,
    fetchChats,
    fetchPendingChats,
    fetchAssignedChats,
    fetchClosedChats,
    loadMoreContacts,
    fetchContact,
    claimChat,
    closeChat,
    reopenChat,
    setChatPublic,
    patchContact,
    setCurrentContact,
    markConversationAsRead,
    updateContactTags,
  };
});
