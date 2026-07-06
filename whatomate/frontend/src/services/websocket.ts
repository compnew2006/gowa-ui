import { useContactsStore } from "@/stores/contacts";
import { useTransfersStore } from "@/stores/transfers";
import { useAuthStore } from "@/stores/auth";
import { useNotesStore } from "@/stores/notes";
import { clearMissingMediaPrefetch } from "@/lib/media_prefetch_cache";
import { maybeAutoDownloadIncomingMedia } from "@/lib/incoming_media_autodownload";
import { canUserAccessInstance } from "@/lib/instance-access";
import { toast } from "vue-sonner";
import router from "@/router";

// Notification sound
let notificationSound: HTMLAudioElement | null = null;
let notificationSoundPending = false;
let interactionListenerAttached = false;
let notificationSourceIndex = 0;
let activeNotificationSound: NotificationSoundKey | null = null;

type NotificationSoundKey = "notification1" | "notification2" | "notification";
const DEFAULT_NOTIFICATION_SOUND: NotificationSoundKey = "notification1";

const rawBasePath =
  typeof window !== "undefined"
    ? ((window as any).__BASE_PATH__ ?? import.meta.env.BASE_URL ?? "/")
    : (import.meta.env.BASE_URL ?? "/");
const normalizedBasePath = String(rawBasePath).replace(/\/$/, "");

function normalizeNotificationSound(value: unknown): NotificationSoundKey {
  if (value === "notification2") return "notification2";
  if (value === "notification") return "notification";
  return DEFAULT_NOTIFICATION_SOUND;
}

function getSelectedNotificationSound(): NotificationSoundKey {
  try {
    const authStore = useAuthStore();
    return normalizeNotificationSound(
      authStore.userSettings.notification_sound,
    );
  } catch {
    return DEFAULT_NOTIFICATION_SOUND;
  }
}

function buildNotificationSources(sound: NotificationSoundKey): string[] {
  const preferredSources = [
    normalizedBasePath ? `${normalizedBasePath}/${sound}.mp3` : `/${sound}.mp3`,
    `/${sound}.mp3`,
  ];
  const legacyFallbackSources = [
    normalizedBasePath
      ? `${normalizedBasePath}/notification.mp3`
      : "/notification.mp3",
    "/notification.mp3",
  ];

  return [...new Set([...preferredSources, ...legacyFallbackSources])];
}

function cleanupNotificationInteractionListeners() {
  if (typeof window === "undefined") return;
  if (!interactionListenerAttached) return;
  window.removeEventListener("pointerdown", retryPendingNotificationSound);
  window.removeEventListener("keydown", retryPendingNotificationSound);
  window.removeEventListener("touchstart", retryPendingNotificationSound);
  interactionListenerAttached = false;
}

function addNotificationInteractionListeners() {
  if (typeof window === "undefined") return;
  if (interactionListenerAttached) return;
  window.addEventListener("pointerdown", retryPendingNotificationSound);
  window.addEventListener("keydown", retryPendingNotificationSound);
  window.addEventListener("touchstart", retryPendingNotificationSound);
  interactionListenerAttached = true;
}

function ensureNotificationSound(): HTMLAudioElement {
  const selectedSound = getSelectedNotificationSound();
  const notificationSources = buildNotificationSources(selectedSound);

  if (!notificationSound || activeNotificationSound !== selectedSound) {
    notificationSourceIndex = 0;
    activeNotificationSound = selectedSound;
    notificationSound = new Audio(notificationSources[notificationSourceIndex]);
    notificationSound.volume = 0.5;
    notificationSound.preload = "auto";
    notificationSound.addEventListener("error", () => {
      if (notificationSourceIndex < notificationSources.length - 1) {
        notificationSourceIndex += 1;
        notificationSound!.src = notificationSources[notificationSourceIndex];
        notificationSound!.load();
      }
    });
  }
  return notificationSound;
}

function retryPendingNotificationSound() {
  if (!notificationSoundPending) {
    cleanupNotificationInteractionListeners();
    return;
  }

  const audio = ensureNotificationSound();
  audio.currentTime = 0;
  audio
    .play()
    .then(() => {
      notificationSoundPending = false;
      cleanupNotificationInteractionListeners();
    })
    .catch(() => {
      // Keep listeners active and retry on next user interaction.
    });
}

function playNotificationSound() {
  const audio = ensureNotificationSound();
  audio.currentTime = 0;
  audio
    .play()
    .then(() => {
      notificationSoundPending = false;
      cleanupNotificationInteractionListeners();
    })
    .catch(() => {
      // Browser may block autoplay until user interacts. Retry after interaction.
      notificationSoundPending = true;
      addNotificationInteractionListeners();
    });
}

function openConversationFromNotification(contactId: string) {
  const contactsStore = useContactsStore();
  void contactsStore.markConversationAsRead(contactId);
  router.push(`/chat/${contactId}`);
}

// Show toast notification with click handler
function showNotification(title: string, body: string, contactId: string) {
  toast.info(title, {
    description: body,
    duration: 5000,
    action: {
      label: "View",
      onClick: () => {
        openConversationFromNotification(contactId);
      },
      actionButtonStyle: {
        background: "transparent",
        border: "1px solid #e5e7eb",
        color: "#3b82f6",
        fontWeight: "500",
      },
    },
  });
}

// WebSocket message types
const WS_TYPE_AUTH = "auth";
const WS_TYPE_NEW_MESSAGE = "new_message";
const WS_TYPE_MESSAGE_MEDIA_UPDATED = "message_media_updated";
const WS_TYPE_STATUS_UPDATE = "status_update";
const WS_TYPE_CONTACT_UPDATE = "contact_update";
const WS_TYPE_SET_CONTACT = "set_contact";
const WS_TYPE_PING = "ping";
const WS_TYPE_PONG = "pong";

// Reaction types
const WS_TYPE_REACTION_UPDATE = "reaction_update";

// Poll vote types
const WS_TYPE_POLL_VOTE_UPDATED = "poll_vote_updated";

// Typing presence (contact composing/paused)
const WS_TYPE_TYPING = "typing";

// Agent transfer types
const WS_TYPE_AGENT_TRANSFER = "agent_transfer";
const WS_TYPE_AGENT_TRANSFER_RESUME = "agent_transfer_resume";
const WS_TYPE_AGENT_TRANSFER_ASSIGN = "agent_transfer_assign";
const WS_TYPE_TRANSFER_ESCALATION = "transfer_escalation";

// Campaign types
const WS_TYPE_CAMPAIGN_STATS_UPDATE = "campaign_stats_update";

// Permission types
const WS_TYPE_PERMISSIONS_UPDATED = "permissions_updated";

// Conversation note types
const WS_TYPE_CONVERSATION_NOTE_CREATED = "conversation_note_created";
const WS_TYPE_CONVERSATION_NOTE_UPDATED = "conversation_note_updated";
const WS_TYPE_CONVERSATION_NOTE_DELETED = "conversation_note_deleted";

// Facebook comment types
const WS_TYPE_FACEBOOK_COMMENT_CREATED = "facebook_comment_created";
const WS_TYPE_FACEBOOK_COMMENT_UPDATED = "facebook_comment_updated";

interface WSMessage {
  type: string;
  payload: any;
}

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private isManualDisconnect = false;
  private reconnectDelay = 1000;
  private pingInterval: number | null = null;
  private isConnected = false;
  private hasConnectedBefore = false;
  private campaignStatsCallbacks: ((payload: any) => void)[] = [];
  private getTokenFn: (() => Promise<string | null>) | null = null;
  private eventSubscribers: Record<string, Array<(payload: any) => void>> = {};

  private async shouldNotifyIncomingMessageForUser(options: {
    contactId: string;
    currentUserId: string;
    userRole?: string;
    isSuperAdmin?: boolean;
    payloadAssignedUserId?: string;
    payloadStatus?: string;
    localAssignedUserId?: string;
    fetchLatestContact?: () => Promise<{
      assigned_user_id?: string;
      status?: string;
    } | null>;
  }): Promise<boolean> {
    const {
      contactId,
      currentUserId,
      userRole,
      isSuperAdmin,
      payloadAssignedUserId,
      payloadStatus,
      localAssignedUserId,
      fetchLatestContact,
    } = options;

    if (!contactId || !currentUserId) return false;

    const normalizedRole =
      typeof userRole === "string" ? userRole.trim().toLowerCase() : "";
    if (
      isSuperAdmin === true ||
      normalizedRole === "admin" ||
      normalizedRole === "manager"
    ) {
      return true;
    }

    const normalizedPayloadAssignedUserId =
      typeof payloadAssignedUserId === "string" &&
      payloadAssignedUserId.trim() !== ""
        ? payloadAssignedUserId.trim()
        : undefined;
    const normalizedLocalAssignedUserId =
      typeof localAssignedUserId === "string" &&
      localAssignedUserId.trim() !== ""
        ? localAssignedUserId.trim()
        : undefined;
    const normalizedPayloadStatus =
      typeof payloadStatus === "string"
        ? payloadStatus.trim().toLowerCase()
        : "";

    if (normalizedPayloadAssignedUserId !== undefined) {
      if (normalizedPayloadAssignedUserId !== currentUserId) return false;
      return normalizedPayloadStatus !== "pending";
    }

    if (normalizedLocalAssignedUserId !== undefined) {
      if (normalizedLocalAssignedUserId !== currentUserId) return false;
      if (normalizedPayloadStatus !== "pending") return true;
    }

    try {
      const contact = await fetchLatestContact?.();
      const latestAssignedUserId =
        typeof contact?.assigned_user_id === "string" &&
        contact.assigned_user_id.trim() !== ""
          ? contact.assigned_user_id.trim()
          : undefined;
      const latestStatus =
        typeof contact?.status === "string"
          ? contact.status.trim().toLowerCase()
          : "";

      return (
        latestAssignedUserId === currentUserId && latestStatus !== "pending"
      );
    } catch {
      // Fall back to local assignment snapshot if API check fails.
      return (
        normalizedLocalAssignedUserId === currentUserId &&
        normalizedPayloadStatus !== "pending"
      );
    }
  }

  async connect(getToken?: () => Promise<string | null>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }
    this.isManualDisconnect = false;

    // Store the token function for reconnects
    if (getToken) {
      this.getTokenFn = getToken;
    }

    // Get a fresh short-lived WS token
    const token = this.getTokenFn ? await this.getTokenFn() : null;
    if (!token) {
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    const basePath = ((window as any).__BASE_PATH__ ?? "").replace(/\/$/, "");
    const url = `${protocol}//${host}${basePath}/ws`;

    try {
      this.ws = new WebSocket(url, ["whm.v1", `auth.${token}`]);

      this.ws.onopen = () => {
        // Keep message auth for backward compatibility with existing WS flow.
        this.send({ type: WS_TYPE_AUTH, payload: { token } });

        const isReconnection = this.hasConnectedBefore;
        this.isConnected = true;
        this.hasConnectedBefore = true;
        this.reconnectAttempts = 0;
        this.startPing();

        // Force refresh data after reconnection to sync any missed updates
        if (isReconnection) {
          this.refreshStaleData();
        }
      };

      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };

      this.ws.onclose = () => {
        this.isConnected = false;
        this.stopPing();
        this.handleReconnect();
      };

      this.ws.onerror = () => {
        // Error handled by onclose
      };
    } catch {
      this.handleReconnect();
    }
  }

  disconnect() {
    this.isManualDisconnect = true;
    this.stopPing();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.isConnected = false;
  }

  private handleMessage(data: string) {
    try {
      const message: WSMessage = JSON.parse(data);
      const store = useContactsStore();
      this.emit(message.type, message.payload);

      switch (message.type) {
        case WS_TYPE_NEW_MESSAGE:
          void this.handleNewMessage(store, message.payload);
          break;
        case WS_TYPE_MESSAGE_MEDIA_UPDATED:
          this.handleMessageMediaUpdated(store, message.payload);
          break;
        case WS_TYPE_STATUS_UPDATE:
          this.handleStatusUpdate(store, message.payload);
          break;
        case WS_TYPE_CONTACT_UPDATE:
          this.handleContactUpdate(store, message.payload);
          break;
        case WS_TYPE_AGENT_TRANSFER:
          this.handleAgentTransfer(message.payload);
          break;
        case WS_TYPE_AGENT_TRANSFER_RESUME:
          this.handleAgentTransferResume(message.payload);
          break;
        case WS_TYPE_AGENT_TRANSFER_ASSIGN:
          this.handleAgentTransferAssign(message.payload);
          break;
        case WS_TYPE_TRANSFER_ESCALATION:
          this.handleTransferEscalation(message.payload);
          break;
        case WS_TYPE_REACTION_UPDATE:
          this.handleReactionUpdate(store, message.payload);
          break;
        case WS_TYPE_POLL_VOTE_UPDATED:
          this.handlePollVoteUpdated(store, message.payload);
          break;
        case WS_TYPE_TYPING:
          this.handleTyping(store, message.payload);
          break;
        case WS_TYPE_PONG:
          // Pong received, connection is alive
          break;
        case WS_TYPE_CAMPAIGN_STATS_UPDATE:
          this.handleCampaignStatsUpdate(message.payload);
          break;
        case WS_TYPE_PERMISSIONS_UPDATED:
          this.handlePermissionsUpdated();
          break;
        case WS_TYPE_CONVERSATION_NOTE_CREATED:
          useNotesStore().addNote(message.payload);
          break;
        case WS_TYPE_CONVERSATION_NOTE_UPDATED:
          useNotesStore().onNoteUpdated(message.payload);
          break;
        case WS_TYPE_CONVERSATION_NOTE_DELETED:
          useNotesStore().onNoteDeleted(message.payload.id);
          break;
        default:
          // Unknown message type, ignore
          break;
      }
    } catch {
      // Failed to parse message, ignore
    }
  }

  private async handleNewMessage(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    if (payload?.message_type === "poll" && payload?.interactive_data?.type === "poll_vote") {
      return;
    }

    // Check if this message is for the current contact
    const currentContact = store.currentContact;
    const currentConversationId =
      typeof currentContact?.conversation_id === "string"
        ? currentContact.conversation_id
        : "";
    const currentInstanceId =
      typeof currentContact?.instance_id === "string"
        ? currentContact.instance_id
        : "";
    const payloadInstanceId =
      typeof payload.instance_id === "string" ? payload.instance_id : "";

    const isViewingGroupConversation =
      !!currentContact &&
      currentContact.is_group_chat === true &&
      currentConversationId !== "" &&
      payload.conversation_id === currentConversationId &&
      (currentInstanceId === "" ||
        payloadInstanceId === "" ||
        payloadInstanceId === currentInstanceId);

    const isViewingThisContact =
      !!currentContact &&
      (payload.contact_id === currentContact.id || isViewingGroupConversation);
    const contactId =
      typeof payload.contact_id === "string" ? payload.contact_id : "";
    const knownContact = contactId
      ? store.contacts.find((contact) => contact.id === contactId)
      : undefined;
    const authStore = useAuthStore();
    let unknownContactFetchPromise: Promise<
      Awaited<ReturnType<typeof store.fetchContact>>
    > | null = null;
    const ensureLatestUnknownContact = () => {
      if (knownContact) {
        return Promise.resolve(knownContact);
      }
      if (!contactId) {
        return Promise.resolve(null);
      }
      if (!canUserAccessInstance(authStore.user, payloadInstanceId)) {
        return Promise.resolve(null);
      }
      if (!unknownContactFetchPromise) {
        unknownContactFetchPromise = store.fetchContact(contactId);
      }
      return unknownContactFetchPromise;
    };

    const incomingMessage = {
      id: payload.id,
      contact_id: payload.contact_id,
      conversation_id: payload.conversation_id,
      is_group_chat: payload.is_group_chat,
      sender_phone: payload.sender_phone,
      sender_push_name: payload.sender_push_name,
      direction: payload.direction,
      message_type: payload.message_type,
      content: payload.content,
      media_url: payload.media_url,
      media_mime_type: payload.media_mime_type,
      media_filename: payload.media_filename,
      interactive_data: payload.interactive_data,
      status: payload.status,
      wamid: payload.wamid,
      error_message: payload.error_message,
      is_reply: payload.is_reply,
      reply_to_message_id: payload.reply_to_message_id,
      reply_to_message: payload.reply_to_message,
      instance_id: payload.instance_id,
      metadata: payload.metadata,
      reactions: payload.reactions,
      created_at: payload.created_at,
      updated_at: payload.updated_at,
    };

    const isNewMessage = store.addMessage(incomingMessage, {
      appendToActiveThread: isViewingThisContact,
    });

    maybeAutoDownloadIncomingMedia(payload);

    const hasContactStatus = typeof payload.contact_status === "string";
    const hasAssignedUserField = typeof payload.assigned_user_id === "string";
    const normalizedAssignedUserId =
      hasAssignedUserField && payload.assigned_user_id.trim() !== ""
        ? payload.assigned_user_id
        : undefined;
    const knownAssignedUserId =
      typeof knownContact?.assigned_user_id === "string" &&
      knownContact.assigned_user_id.trim() !== ""
        ? knownContact.assigned_user_id
        : undefined;
    const normalizedContactStatus = hasContactStatus
      ? payload.contact_status.trim().toLowerCase()
      : undefined;
    if (
      typeof payload.contact_id === "string" &&
      (hasContactStatus || hasAssignedUserField)
    ) {
      store.patchContact({
        id: payload.contact_id,
        status: hasContactStatus ? payload.contact_status : undefined,
        assigned_user_id: normalizedAssignedUserId,
        assigned_user_name:
          typeof payload.assigned_user_name === "string"
            ? payload.assigned_user_name
            : undefined,
      });
    }

    // Notifications are for newly-added incoming messages only.
    if (isNewMessage && payload.direction === "incoming") {
      const currentUserId = authStore.user?.id;
      const settings = authStore.userSettings;

      // Check if new message alerts are enabled (default to true if not set)
      const alertsEnabled = settings.new_message_alerts !== false;

      const shouldNotifyCurrentUser = currentUserId
        ? await this.shouldNotifyIncomingMessageForUser({
            contactId,
            currentUserId,
            userRole: authStore.user?.role?.name,
            isSuperAdmin: authStore.user?.is_super_admin,
            payloadAssignedUserId: normalizedAssignedUserId,
            payloadStatus: normalizedContactStatus,
            localAssignedUserId: knownAssignedUserId,
            fetchLatestContact: ensureLatestUnknownContact,
          })
        : false;

      if (alertsEnabled && shouldNotifyCurrentUser) {
        const senderName = payload.profile_name || "Unknown";
        const messagePreview =
          typeof payload.content === "string"
            ? payload.content
            : payload.content?.body || "New message";
        const preview =
          messagePreview.length > 50
            ? messagePreview.substring(0, 50) + "..."
            : messagePreview;
        const contactId = payload.contact_id;

        // Always play sound for eligible incoming notifications.
        playNotificationSound();
        // Keep toast popups suppressed while actively viewing this chat.
        if (!isViewingThisContact) {
          showNotification(senderName, preview, contactId);
        }
      }
    }

    // Avoid full list refetch on every message; it can evict already-loaded rows when pagination is active.
    // If this message belongs to a contact not present in the store, fetch just that contact.
    if (contactId && !knownContact) {
      void ensureLatestUnknownContact();
    }
  }

  private handleStatusUpdate(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    store.updateMessageStatus(
      payload.message_id,
      payload.status,
      payload.error_message,
    );
  }

  private handleMessageMediaUpdated(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    const messageID = typeof payload?.id === "string" ? payload.id : "";
    if (!messageID) return;
    if (typeof payload?.media_url === "string" && payload.media_url.trim() !== "") {
      clearMissingMediaPrefetch(messageID);
    }

    const existing = store.messages.find((message) => message.id === messageID);
    const contactID =
      typeof payload?.contact_id === "string"
        ? payload.contact_id
        : existing?.contact_id;
    if (!contactID) return;

    store.patchMessage({
      id: messageID,
      contact_id: contactID,
      direction: existing?.direction ?? "incoming",
      message_type: existing?.message_type ?? "document",
      content: payload?.content ?? existing?.content ?? { body: "" },
      status: existing?.status ?? "received",
      created_at:
        typeof payload?.created_at === "string"
          ? payload.created_at
          : (existing?.created_at ?? new Date().toISOString()),
      updated_at:
        typeof payload?.updated_at === "string"
          ? payload.updated_at
          : new Date().toISOString(),
      media_url:
        typeof payload?.media_url === "string"
          ? payload.media_url
          : existing?.media_url,
      media_mime_type:
        typeof payload?.media_mime_type === "string"
          ? payload.media_mime_type
          : existing?.media_mime_type,
      media_filename:
        typeof payload?.media_filename === "string"
          ? payload.media_filename
          : existing?.media_filename,
      metadata: payload?.metadata ?? existing?.metadata,
      interactive_data: payload?.interactive_data ?? existing?.interactive_data,
      error_message:
        typeof payload?.error_message === "string"
          ? payload.error_message
          : existing?.error_message,
    });
  }

  private handleContactUpdate(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    const id = typeof payload?.id === "string" ? payload.id : "";
    if (!id) return;
    store.patchContact(payload);

    const shouldNotifyAssignment = payload?.notify_assignment === true;
    if (!shouldNotifyAssignment) return;

    const authStore = useAuthStore();
    const currentUserId = authStore.user?.id;
    if (!currentUserId || payload.assigned_user_id !== currentUserId) return;
    const hasContactInStore = store.contacts.some(
      (contact) => contact.id === id,
    );
    if (!hasContactInStore) {
      void store.fetchContact(id);
    }

    const contactName =
      typeof payload.profile_name === "string" &&
      payload.profile_name.trim() !== ""
        ? payload.profile_name
        : "A chat";

    playNotificationSound();
    toast.info("Chat Assigned", {
      description: `${contactName} has been assigned to you`,
      duration: 5000,
      action: {
        label: "View",
        onClick: () =>
          router.push({
            name: "chat-conversation",
            params: { contactId: id },
            query: { tab: "assigned" },
          }),
      },
    });
  }

  private handleReactionUpdate(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    // Update the message reactions if we're viewing the contact
    const currentContact = store.currentContact;
    if (currentContact && payload.contact_id === currentContact.id) {
      store.updateMessageReactions(payload.message_id, payload.reactions);
    }
  }

  // handleTyping applies an inbound typing-presence event (contact composing /
  // paused) to the contacts store. The store owns the self-clearing state; the
  // UI reads store.isContactTyping(contactID).
  private handleTyping(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    const contactID =
      typeof payload?.contact_id === "string" ? payload.contact_id : "";
    const state =
      typeof payload?.state === "string" ? payload.state : "";
    if (!contactID) return;
    store.setTyping(contactID, state);
  }

  private handlePollVoteUpdated(
    store: ReturnType<typeof useContactsStore>,
    payload: any,
  ) {
    const messageID = typeof payload?.id === "string" ? payload.id : "";
    if (!messageID) return;

    const existing = store.messages.find(
      (message) => message.id === messageID,
    );
    const contactID =
      typeof payload?.contact_id === "string"
        ? payload.contact_id
        : existing?.contact_id;
    if (!contactID) return;

    store.patchMessage({
      id: messageID,
      contact_id: contactID,
      direction: existing?.direction ?? "incoming",
      message_type: existing?.message_type ?? "document",
      content: payload?.content ?? existing?.content ?? { body: "" },
      status: existing?.status ?? "received",
      created_at:
        typeof payload?.created_at === "string"
          ? payload.created_at
          : (existing?.created_at ?? new Date().toISOString()),
      updated_at:
        typeof payload?.updated_at === "string"
          ? payload.updated_at
          : new Date().toISOString(),
      media_url: existing?.media_url,
      media_mime_type: existing?.media_mime_type,
      media_filename: existing?.media_filename,
      metadata: existing?.metadata,
      interactive_data:
        payload?.interactive_data ?? existing?.interactive_data,
      error_message: existing?.error_message,
    });
  }

  private handleAgentTransfer(payload: any) {
    const transfersStore = useTransfersStore();
    const authStore = useAuthStore();

    // Add transfer to store with default SLA values
    transfersStore.addTransfer({
      id: payload.id,
      contact_id: payload.contact_id,
      contact_name: payload.contact_name || payload.phone_number,
      phone_number: payload.phone_number,
      whatsapp_account: payload.whatsapp_account,
      status: payload.status,
      source: payload.source || "manual",
      agent_id: payload.agent_id,
      team_id: payload.team_id,
      notes: payload.notes,
      transferred_at: payload.transferred_at,
      // Default SLA values - will be updated on next fetch
      sla_breached: false,
      escalation_level: 0,
    });

    // Refresh to get complete data including SLA fields
    transfersStore.fetchTransfers({ status: "active" });

    // Show toast notification for admin/manager or assigned agent
    const userRole = authStore.user?.role?.name;
    const currentUserId = authStore.user?.id;
    const isAssignedToMe = payload.agent_id === currentUserId;

    if (userRole === "admin" || userRole === "manager" || isAssignedToMe) {
      const contactName = payload.contact_name || payload.phone_number;
      toast.info("New Transfer", {
        description: `${contactName} has been transferred to ${isAssignedToMe ? "you" : "agent queue"}`,
        duration: 5000,
        action: {
          label: "View",
          onClick: () => router.push("/chatbot/transfers"),
        },
      });
    }
  }

  private handleAgentTransferResume(payload: any) {
    const transfersStore = useTransfersStore();

    const updated = transfersStore.updateTransfer(payload.id, {
      status: payload.status,
      resumed_at: payload.resumed_at,
      resumed_by: payload.resumed_by,
    });

    // If transfer wasn't found in store, refresh to get latest data
    if (!updated) {
      transfersStore.fetchTransfers();
    }
  }

  private handleAgentTransferAssign(payload: any) {
    const transfersStore = useTransfersStore();
    const authStore = useAuthStore();

    // Try to update existing transfer
    transfersStore.updateTransfer(payload.id, {
      agent_id: payload.agent_id,
      team_id: payload.team_id,
    });

    // Always refresh to ensure UI is in sync (queue counts, etc.)
    transfersStore.fetchTransfers();

    // Notify if assigned to current user
    const currentUserId = authStore.user?.id;
    if (payload.agent_id === currentUserId) {
      toast.info("Transfer Assigned", {
        description: "A transfer has been assigned to you",
        duration: 5000,
        action: {
          label: "View",
          onClick: () => router.push("/chatbot/transfers"),
        },
      });
    }
  }

  private handleTransferEscalation(payload: any) {
    const authStore = useAuthStore();
    const currentUserId = authStore.user?.id;

    // Check if current user should be notified
    const notifyIds: string[] = payload.escalation_notify_ids || [];
    const shouldNotify = notifyIds.includes(currentUserId || "");

    // Also notify admins/managers
    const userRole = authStore.user?.role?.name;
    const isAdminOrManager = userRole === "admin" || userRole === "manager";

    if (shouldNotify || isAdminOrManager) {
      const levelName =
        payload.level_name === "critical" ? "Critical" : "Warning";
      const contactName = payload.contact_name || payload.phone_number;

      // Play notification sound
      playNotificationSound();

      // Show urgent toast
      toast.warning(`SLA Escalation: ${levelName}`, {
        description: `${contactName} has been waiting since ${new Date(payload.waiting_since).toLocaleTimeString()}`,
        duration: 10000,
        action: {
          label: "View",
          onClick: () => router.push("/chatbot/transfers"),
        },
      });
    }
  }

  private handleCampaignStatsUpdate(payload: any) {
    // Notify all registered callbacks
    this.campaignStatsCallbacks.forEach((callback) => callback(payload));
  }

  private async handlePermissionsUpdated() {
    const authStore = useAuthStore();

    // Refresh user data from server
    const success = await authStore.refreshUserData();

    if (success) {
      toast.info("Permissions Updated", {
        description:
          "Your permissions have been updated. The page will refresh.",
        duration: 3000,
      });

      // Reload the page after a short delay to apply new permissions
      setTimeout(() => {
        window.location.reload();
      }, 1500);
    }
  }

  onCampaignStatsUpdate(callback: (payload: any) => void) {
    this.campaignStatsCallbacks.push(callback);
    // Return unsubscribe function
    return () => {
      const index = this.campaignStatsCallbacks.indexOf(callback);
      if (index > -1) {
        this.campaignStatsCallbacks.splice(index, 1);
      }
    };
  }

  subscribe(eventType: string, callback: (payload: any) => void) {
    if (!this.eventSubscribers[eventType]) {
      this.eventSubscribers[eventType] = [];
    }
    this.eventSubscribers[eventType].push(callback);
  }

  unsubscribe(eventType: string, callback: (payload: any) => void) {
    const callbacks = this.eventSubscribers[eventType];
    if (!callbacks) {
      return;
    }
    this.eventSubscribers[eventType] = callbacks.filter(
      (cb) => cb !== callback,
    );
  }

  // Test-only helper to trigger websocket subscribers from E2E.
  emitForTest(eventType: string, payload: any) {
    this.emit(eventType, payload);
  }

  private emit(eventType: string, payload: any) {
    const callbacks = this.eventSubscribers[eventType];
    if (!callbacks || callbacks.length === 0) {
      return;
    }
    callbacks.forEach((callback) => callback(payload));
  }

  private handleReconnect() {
    if (this.isManualDisconnect) {
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), 30000);

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  setCurrentContact(contactId: string | null) {
    this.send({
      type: WS_TYPE_SET_CONTACT,
      payload: { contact_id: contactId || "" },
    });
  }

  private send(message: WSMessage) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  private startPing() {
    this.stopPing();
    this.pingInterval = window.setInterval(() => {
      this.send({ type: WS_TYPE_PING, payload: {} });
    }, 30000); // Ping every 30 seconds
  }

  private stopPing() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private refreshStaleData() {
    // Refresh contacts list
    const contactsStore = useContactsStore();
    contactsStore.fetchContacts();

    // Refresh transfers
    const transfersStore = useTransfersStore();
    transfersStore.fetchTransfers();

    // Show subtle notification
    toast.info("Connection restored", {
      description: "Data has been refreshed",
      duration: 3000,
    });
  }

  getIsConnected() {
    return this.isConnected;
  }
}

// Export singleton instance
export const wsService = new WebSocketService();

export {
  WS_TYPE_FACEBOOK_COMMENT_CREATED,
  WS_TYPE_FACEBOOK_COMMENT_UPDATED,
};

declare global {
  interface Window {
    __WHM_WS_TEST_EMIT__?: (eventType: string, payload: any) => void;
  }
}

if (
  typeof window !== "undefined" &&
  (import.meta.env.DEV || import.meta.env.MODE === "test")
) {
  window.__WHM_WS_TEST_EMIT__ = (eventType: string, payload: any) => {
    wsService.emitForTest(eventType, payload);
  };
}
