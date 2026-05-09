import { defineStore } from "pinia";
import { ref } from "vue";
import { chatsService, messagesService } from "@/services/api";
import { unwrapResponse } from "@/lib/api-utils";
import {
  type MessagesListPayload,
  type AddMessageOptions,
  removeSyntheticPlaceholderMessages,
  getMessageBody,
} from "./helpers";
import { useContactsStore } from "./contacts";
import type {
  Message,
  Reaction,
} from "@/types/contacts";

export const useMessagesStore = defineStore("messages", () => {
  const messages = ref<Message[]>([]);
  const isLoadingMessages = ref(false);
  const isLoadingOlderMessages = ref(false);
  const isMessageAccessRestricted = ref(false);
  const hasMoreMessages = ref(false);
  const messageLoadError = ref<string | null>(null);
  let messageFetchSequence = 0;
  let latestMessageFetchSequence = 0;
  const replyingTo = ref<Message | null>(null);
  const accountFilter = ref<string | null>(null);

  async function fetchMessages(
    contactId: string,
    params?: { page?: number; limit?: number; account?: string },
  ) {
    const requestSequence = ++messageFetchSequence;
    latestMessageFetchSequence = requestSequence;
    isLoadingMessages.value = true;
    isMessageAccessRestricted.value = false;
    messageLoadError.value = null;
    messages.value = [];
    hasMoreMessages.value = false;
    try {
      const response = await chatsService.listMessages(contactId, params);
      if (requestSequence !== latestMessageFetchSequence) {
        return;
      }
      const data = unwrapResponse<MessagesListPayload>(response);
      messages.value = removeSyntheticPlaceholderMessages(data.messages || []);
      hasMoreMessages.value = data.has_more === true;

      const contactsStore = useContactsStore();
      const contact = contactsStore.contacts.find((c) => c.id === contactId);
      if (contact) {
        contact.unread_count = 0;
      }
      if (contactsStore.currentContact?.id === contactId) {
        contactsStore.currentContact.unread_count = 0;
      }
    } catch (error: any) {
      if (requestSequence !== latestMessageFetchSequence) {
        return;
      }
      if (error?.response?.status === 403) {
        messages.value = [];
        hasMoreMessages.value = false;
        isMessageAccessRestricted.value = true;
      } else {
        messageLoadError.value = error?.message || String(error);
      }
      console.error("Failed to fetch messages:", error);
    } finally {
      if (requestSequence === latestMessageFetchSequence) {
        isLoadingMessages.value = false;
      }
    }
  }

  async function fetchOlderMessages(contactId: string, account?: string) {
    if (
      isMessageAccessRestricted.value ||
      isLoadingOlderMessages.value ||
      !hasMoreMessages.value ||
      messages.value.length === 0
    ) {
      return;
    }

    isLoadingOlderMessages.value = true;
    try {
      const oldestMessageId = messages.value[0].id;
      const response = await chatsService.listMessages(contactId, {
        before_id: oldestMessageId,
        account,
      });
      const data = unwrapResponse<MessagesListPayload>(response);
      const olderMessages = data.messages || [];

      const contactsStore = useContactsStore();
      if (contactsStore.currentContact?.id !== contactId) {
        return;
      }

      if (olderMessages.length > 0) {
        messages.value = removeSyntheticPlaceholderMessages([
          ...olderMessages,
          ...messages.value,
        ]);
      }
      hasMoreMessages.value = data.has_more === true;
    } catch (error) {
      console.error("Failed to fetch older messages:", error);
    } finally {
      isLoadingOlderMessages.value = false;
    }
  }

  async function sendMessage(
    contactId: string,
    type: string,
    content: unknown,
    replyToMessageId?: string,
    whatsappAccount?: string,
    explicitInstanceID?: string,
  ) {
    try {
      const contactsStore = useContactsStore();
      const contact = contactsStore.contacts.find((item) => item.id === contactId);
      const resolvedInstanceID =
        typeof explicitInstanceID === "string" &&
        explicitInstanceID.trim() !== ""
          ? explicitInstanceID.trim()
          : contact?.instance_id;
      const response = await messagesService.send(contactId, {
        type,
        content,
        reply_to_message_id: replyToMessageId,
        instance_id: resolvedInstanceID,
        whatsapp_account: whatsappAccount,
      });
      const newMessage = unwrapResponse<Message>(response);
      addMessage(newMessage);
      return newMessage;
    } catch (error) {
      console.error("Failed to send message:", error);
      throw error;
    }
  }

  function setReplyingTo(message: Message | null) {
    replyingTo.value = message;
  }

  function clearReplyingTo() {
    replyingTo.value = null;
  }

  function addMessage(
    message: Message,
    options: AddMessageOptions = {},
  ): boolean {
    const { appendToActiveThread = true } = options;

    const contactsStore = useContactsStore();
    const contact = contactsStore.contacts.find((c) => c.id === message.contact_id);
    if (contact) {
      contact.last_message_at = message.created_at;
      if (message.direction === "incoming") {
        contact.unread_count++;
        contact.last_inbound_at = message.created_at;
        contact.service_window_open = true;
      }
    }
    if (
      contactsStore.currentContact &&
      contactsStore.currentContact.id === message.contact_id &&
      message.direction === "incoming"
    ) {
      contactsStore.currentContact.last_inbound_at = message.created_at;
      contactsStore.currentContact.service_window_open = true;
    }

    if (
      accountFilter.value &&
      message.whatsapp_account &&
      message.whatsapp_account !== accountFilter.value
    ) {
      return false;
    }

    const existingIndex = messages.value.findIndex((m) => m.id === message.id);
    if (existingIndex !== -1) {
      if (appendToActiveThread) {
        messages.value[existingIndex] = {
          ...messages.value[existingIndex],
          ...message,
        };
        messages.value = removeSyntheticPlaceholderMessages(messages.value);
      }
      return false;
    }

    if (appendToActiveThread) {
      messages.value.push(message);
      messages.value = removeSyntheticPlaceholderMessages(messages.value);
    }

    return true;
  }

  function updateMessageStatus(
    messageId: string,
    status: string,
    errorMessage?: string,
  ) {
    const index = messages.value.findIndex((m) => m.id === messageId);
    if (index !== -1) {
      messages.value[index] = {
        ...messages.value[index],
        status,
        ...(errorMessage ? { error_message: errorMessage } : {}),
      };
    }
  }

  function patchMessage(updatedMessage: Message) {
    const index = messages.value.findIndex((m) => m.id === updatedMessage.id);
    if (index !== -1) {
      messages.value[index] = { ...messages.value[index], ...updatedMessage };
      messages.value = removeSyntheticPlaceholderMessages(messages.value);
    }

    const contactsStore = useContactsStore();
    const contact = contactsStore.contacts.find(
      (c) => c.id === updatedMessage.contact_id,
    );
    if (contact) {
      const existingLastAt = contact.last_message_at
        ? new Date(contact.last_message_at).getTime()
        : 0;
      const updatedLastAt = updatedMessage.created_at
        ? new Date(updatedMessage.created_at).getTime()
        : 0;
      if (updatedLastAt >= existingLastAt) {
        contact.last_message_at = updatedMessage.created_at;
        contact.last_message_preview = getMessageBody(updatedMessage);
      }
    }
  }

  function setAccountFilter(account: string | null) {
    accountFilter.value = account;
  }

  function clearMessages() {
    latestMessageFetchSequence = ++messageFetchSequence;
    messages.value = [];
    hasMoreMessages.value = false;
    isMessageAccessRestricted.value = false;
    isLoadingMessages.value = false;
    messageLoadError.value = null;
    accountFilter.value = null;
  }

  function updateMessageReactions(messageId: string, reactions: Reaction[]) {
    const message = messages.value.find((m) => m.id === messageId);
    if (message) {
      message.reactions = reactions;
    }
  }

  return {
    messages,
    isLoadingMessages,
    isLoadingOlderMessages,
    isMessageAccessRestricted,
    messageLoadError,
    hasMoreMessages,
    replyingTo,
    fetchMessages,
    fetchOlderMessages,
    sendMessage,
    addMessage,
    updateMessageStatus,
    patchMessage,
    setReplyingTo,
    clearReplyingTo,
    setAccountFilter,
    clearMessages,
    updateMessageReactions,
  };
});
