<script setup lang="ts">
import {
  ref,
  watch,
  onMounted,
  onUnmounted,
  nextTick,
  computed,
  type ComponentPublicInstance,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  useContactsStore,
  type Contact,
  type Message,
  type ChatTypeFilter,
} from "@/stores/contacts";
import { useAuthStore } from "@/stores/auth";
import { useUsersStore } from "@/stores/users";
import { useTransfersStore } from "@/stores/transfers";
import { useConfigStore } from "@/stores/config";
import { wsService } from "@/services/websocket";
import {
  contactsService,
  messagesService,
  cannedResponsesService,
  type CannedResponseAttachment,
} from "@/services/api";
import { useTagsStore } from "@/stores/tags";
import { TagBadge } from "@/components/ui/tag-badge";
import { getTagColorClass } from "@/lib/constants";
import { canUserAccessInstance } from "@/lib/instance-access";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { toast } from "vue-sonner";
import {
  Search,
  Check,
  CheckCheck,
  Clock,
  AlertCircle,
  User,
  UserPlus,
  X,
  Archive,
  Loader2,
  Filter,
} from "lucide-vue-next";
import { getInitials, getAvatarGradient } from "@/lib/utils";
import { isGroupContact } from "@/lib/group-chat";
import { resolveMediaFilename } from "@/lib/media-actions";
import { resolvePreferredOutboundInstanceID } from "@/lib/chat-outbound-instance";
import type { SidebarContactEntry } from "@/lib/chat-sidebar-unifier";
import { MessageHistoryNavigator } from "@/lib/message-history-navigator";
import { useColorMode } from "@/composables/useColorMode";
import { useInfiniteScroll } from "@/composables/useInfiniteScroll";
import { useVirtualMessageList } from "@/composables/useVirtualMessageList";
import { useMessageContent } from "@/composables/useMessageContent";
import { useTypingPresence } from "@/composables/useTypingPresence";
import { useChatSidebar } from "@/composables/useChatSidebar";
import { useChatMedia } from "@/composables/useChatMedia";
import { useBatchPrint } from "@/composables/useBatchPrint";
import { useChatActions } from "@/composables/useChatActions";
import { useChatMessaging } from "@/composables/useChatMessaging";
import { useChatKeyboardNav } from "@/composables/useChatKeyboardNav";
import ContactInfoPanel from "@/components/chat/ContactInfoPanel.vue";
import ConversationNotes from "@/components/chat/ConversationNotes.vue";
import InstanceTag from "@/components/chat/InstanceTag.vue";
import MediaGroupBar from "@/components/chat/MediaGroupBar.vue";
import StatusStoriesBar from "@/components/chat/status/StatusStoriesBar.vue";
import ChatEmptyState from "@/components/chat/ChatEmptyState.vue";
import ChatTypingIndicator from "@/components/chat/ChatTypingIndicator.vue";
import ChatLoadError from "@/components/chat/ChatLoadError.vue";
import ChatHeader from "@/components/chat/ChatHeader.vue";
import ChatMessageBubble from "@/components/chat/ChatMessageBubble.vue";
import ChatInputBar from "@/components/chat/ChatInputBar.vue";
import ChatAssignDialog from "@/components/chat/ChatAssignDialog.vue";
import ChatMediaViewerDialog from "@/components/chat/ChatMediaViewerDialog.vue";
import ChatMediaSendDialog from "@/components/chat/ChatMediaSendDialog.vue";
import type { PendingMediaUpload } from "@/components/chat/ChatMediaSendDialog.vue";
import ChatProfilePhotoDialog from "@/components/chat/ChatProfilePhotoDialog.vue";
import ConnectionStatusBanner from "@/components/chat/ConnectionStatusBanner.vue";
import { useInstancesStore } from "@/stores/instances";
import { useNotesStore } from "@/stores/notes";
import { CreateContactDialog } from "@/components/shared";
import { useMediaGroups } from "@/composables/useMediaGroups";
import { resolveChatBackgroundStyle } from "@/lib/chat-backgrounds";
import { isMessagePrintSupported } from "@/lib/single-media-print";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const contactsStore = useContactsStore();
const authStore = useAuthStore();
const usersStore = useUsersStore();
const transfersStore = useTransfersStore();
const configStore = useConfigStore();
const tagsStore = useTagsStore();
const notesStore = useNotesStore();
const instancesStore = useInstancesStore();
const { isDark } = useColorMode();
const chatBackgroundStyle = computed(() =>
  resolveChatBackgroundStyle(authStore.user?.settings?.chat_background, {
    theme: isDark.value ? "dark" : "light",
    variant: "chat",
  }),
);

// Media grouping for batch download
const { getGroupForMessage, isGroupLeader, isGroupTail, isGroupMember } =
  useMediaGroups(computed(() => contactsStore.messages));

/** Resolve full Message objects for a group (for download) */
function getGroupMessages(groupId: string): Message[] {
  const group = getGroupForMessage(groupId);
  if (!group) return [];
  return group.messageIds
    .map((mid) => contactsStore.messages.find((m: Message) => m.id === mid))
    .filter((m): m is Message => !!m);
}

const isAdminUser = computed(
  () =>
    authStore.user?.is_super_admin === true ||
    (authStore.userRole || "").toLowerCase() === "admin",
);
const isAgentUser = computed(
  () => (authStore.userRole || "").toLowerCase() === "agent",
);
const canSoftDeleteChats = computed(() =>
  authStore.hasPermission("contacts", "soft_delete"),
);
const canShowAddContact = computed(
  () => isAdminUser.value || authStore.hasPermission("contacts", "write"),
);
const canRevokeMessages = computed(() =>
  authStore.hasPermission("chat", "delete"),
);
const canManageTransfers = computed(() =>
  authStore.hasPermission("transfers", "write"),
);
const chatSidebar = useChatSidebar();
const {
  isRTL,
  chatSidebarUnifier,
  chatSidebarViewMode,
  selectedAccount,
  contactAccounts,
  contactsSidebarWidth,
  isContactsSidebarResizing,
  isContactsSidebarCompact,
  isContactsSidebarWide,
  isSidebarUnifiedMode,
  startContactsSidebarResize,
  stopContactsSidebarResize,
  refreshChatSidebarViewModePreference,
  toAccountToggleKey,
  toContactToggleKey,
  contactIDFromToggleKey,
  selectedAccountFilter,
  findSidebarEntrySourceContact,
  resolveSourceContactForToggle,
  resolveSidebarEntryInstanceIDs,
  getSidebarEntryInstanceCount,
  hasSidebarEntryMultipleInstances,
  getSidebarEntryPrimaryInstanceID,
  resolveSidebarEntryInstanceLabel,
  getSidebarEntryPrimaryInstanceLabel,
  isSidebarEntryActive,
  getSidebarEntryPreferredContact,
} = chatSidebar;

const chatMedia = useChatMedia(computed(() => contactsStore.messages));
const {
  mediaBlobUrls,
  mediaBlobCache,
  isChatMediaViewerOpen,
  chatMediaViewerURL,
  chatMediaViewerType,
  chatMediaViewerTitle,
  resetMediaLoadingPipeline,
  loadMediaForMessage,
  loadMediaForMessages,
  getMediaBlobUrl,
  isMediaLoading,
  openChatMediaViewer,
  closeChatMediaViewer,
  cleanupBlobUrls,
} = chatMedia;

const batchPrint = useBatchPrint(
  computed(() => contactsStore.messages),
  mediaBlobCache,
  mediaBlobUrls,
  getMediaBlobUrl,
  loadMediaForMessage,
);
const {
  isPreparingBatchPrint,
  isBatchPrintSelectionMode,
  selectedBatchPrintMessageIds,
  selectedBatchPrintCount,
  canMergeSelectedBubbleFiles,
  resetBatchPrintSelection,
  cancelBatchPrintSelection,
  isBatchPrintBubbleSelectable,
  isBatchPrintBubbleSelected,
  toggleBatchPrintMessageSelection,
  handleMessageBubbleClickForBatchPrint,
  resolveMessageBlobForBatchPrint,
  openBatchPrintPicker,
  isModifiedPointerEvent,
} = batchPrint;

function resolveSelectedSourceContact(contact: Contact | null): Contact | null {
  return chatSidebar.resolveSelectedSourceContact(contact, currentSidebarEntry.value);
}

function resolveExplicitSourceContact(contact: Contact | null): Contact | null {
  return chatSidebar.resolveExplicitSourceContact(contact, currentSidebarEntry.value);
}

function formatAccountToggleLabel(toggleKey: string): string {
  return chatSidebar.formatAccountToggleLabel(toggleKey, currentSidebarEntry.value, contactsStore.contacts);
}

const messageInput = ref("");
const messagesEndRef = ref<HTMLElement | null>(null);
const messageInputRef = ref<HTMLTextAreaElement | null>(null);
const isSending = ref(false);
const isInfoPanelOpen = ref(false);
const isNotesPanelOpen = ref(false);
const contactSessionData = ref<any>(null);

const TYPING_INDICATOR_TIMEOUT_MS = 5000;
const typingContactId = ref<string | null>(null);
const typingContactName = ref("");
let typingIndicatorTimer: ReturnType<typeof setTimeout> | null = null;

const typingPresence = useTypingPresence();
const {
  resetTypingPresenceState,
  sendTypingPresenceForContact,
  scheduleTypingPaused,
  stopTypingForContact,
} = typingPresence;

function typingContext() {
  const contact = contactsStore.currentContact;
  return {
    messages: contactsStore.messages,
    selectedSourceContact: resolveExplicitSourceContact(contact),
    selectedInstanceID: contactsStore.selectedInstanceId,
  };
}

function resolveOutboundInstanceID(
  contact: Contact | null,
): string | undefined {
  return resolvePreferredOutboundInstanceID({
    messages: contactsStore.messages,
    selectedSourceContact: resolveExplicitSourceContact(contact),
    currentContact: contact,
    selectedInstanceID: contactsStore.selectedInstanceId,
  });
}

function resolveOutboundWhatsAppAccount(
  contact: Contact | null,
): string | undefined {
  const selectedFilter = selectedAccountFilter(selectedAccount.value);
  if (selectedFilter) return selectedFilter;
  const selectedSource = resolveSelectedSourceContact(contact);
  const accountName =
    typeof selectedSource?.whatsapp_account === "string"
      ? selectedSource.whatsapp_account.trim()
      : "";
  return accountName || undefined;
}

const isProfilePhotoDialogOpen = ref(false);
const profilePhotoContact = ref<Contact | null>(null);

const stickyDate = ref("");
const showStickyDate = ref(false);
let stickyDateTimeout: ReturnType<typeof setTimeout> | null = null;
let quoteHighlightTimeout: ReturnType<typeof setTimeout> | null = null;
const isQuoteNavigationInProgress = ref(false);
const QUOTE_NAVIGATION_MAX_HISTORY_REQUESTS = 64;

const isTagFilterOpen = ref(false);

// Service window state
const isServiceWindowExpired = computed(() => {
  const contact = contactsStore.currentContact;
  if (!contact) return false;
  if (configStore.isWhatsmeow) return false;
  return contact.service_window_open === false;
});

const canSendMessage = computed(() => {
  return (
    Boolean(messageInput.value.trim()) || hasPendingCannedAttachments.value
  );
});

// Add contact dialog state
const isAddContactOpen = ref(false);
const deletingSidebarEntryKey = ref<string | null>(null);
const pendingSidebarDeleteEntryKey = ref<string | null>(null);
const softDeletingSidebarEntryKey = ref<string | null>(null);
const pendingSidebarSoftDeleteEntryKey = ref<string | null>(null);
const SIDEBAR_DELETE_CONFIRM_TIMEOUT_MS = 5000;
let pendingSidebarDeleteTimeout: ReturnType<typeof setTimeout> | null = null;
let pendingSidebarSoftDeleteTimeout: ReturnType<typeof setTimeout> | null =
  null;
let contactSelectionSequence = 0;
const contactsScroll = useInfiniteScroll({
  direction: "bottom",
  onLoadMore: () => contactsStore.loadMoreContacts(),
  hasMore: computed(() => contactsStore.hasMoreContacts),
  isLoading: computed(() => contactsStore.isLoadingMoreContacts),
});

const CONTACTS_SIDEBAR_PREFILL_MAX_REQUESTS = 48;
const CONTACTS_SIDEBAR_PREFILL_TIMEOUT_MS = 5000;
const CONTACTS_SEARCH_REFRESH_DEBOUNCE_MS = 280;
const CONTACTS_SEARCH_PREFETCH_DEBOUNCE_MS = 220;
const CONTACTS_SEARCH_PREFETCH_MAX_REQUESTS = 24;
const CONTACTS_SEARCH_PREFETCH_TIMEOUT_MS = 4000;
let contactsSearchRefreshTimer: number | null = null;
let contactsSearchPrefetchTimer: number | null = null;
let contactsSearchPrefetchRunToken = 0;

async function hydrateContactsSidebarUntilScrollable() {
  await nextTick();
  const viewport = contactsScroll.getViewport();
  if (!viewport) return;

  const startedAt = Date.now();
  let requests = 0;
  while (
    contactsStore.hasMoreContacts &&
    !contactsStore.isLoadingMoreContacts
  ) {
    const hasOverflow = viewport.scrollHeight > viewport.clientHeight + 1;
    if (hasOverflow) {
      break;
    }

    if (requests >= CONTACTS_SIDEBAR_PREFILL_MAX_REQUESTS) {
      break;
    }
    if (Date.now() - startedAt >= CONTACTS_SIDEBAR_PREFILL_TIMEOUT_MS) {
      break;
    }

    const beforeRawCount = contactsStore.contacts.length;
    await contactsStore.loadMoreContacts();
    await nextTick();
    requests++;

    if (contactsStore.contacts.length === beforeRawCount) {
      break;
    }
  }
}

async function refreshContactsSidebar() {
  await contactsStore.fetchChats({
    search: contactsStore.searchQuery || undefined,
    assigned_to: isAgentUser.value ? "me" : undefined,
  });
  await hydrateContactsSidebarUntilScrollable();
}

async function prefetchSearchResultsIfNeeded(query: string, token: number) {
  const trimmedQuery = query.trim();
  if (!trimmedQuery) return;

  await nextTick();

  const startedAt = Date.now();
  let requests = 0;

  while (
    token === contactsSearchPrefetchRunToken &&
    contactsStore.searchQuery.trim() === trimmedQuery &&
    contactsStore.sortedContacts.length === 0 &&
    contactsStore.hasMoreContacts &&
    !contactsStore.isLoadingMoreContacts
  ) {
    if (requests >= CONTACTS_SEARCH_PREFETCH_MAX_REQUESTS) {
      break;
    }
    if (Date.now() - startedAt >= CONTACTS_SEARCH_PREFETCH_TIMEOUT_MS) {
      break;
    }

    const beforeRawCount = contactsStore.contacts.length;
    await contactsStore.loadMoreContacts();
    await nextTick();
    requests++;

    if (contactsStore.contacts.length === beforeRawCount) {
      break;
    }
  }
}

async function switchChatTab(tab: "pending" | "assigned") {
  if (contactsStore.activeChatTab === tab) return;
  contactsStore.setActiveChatTab(tab);
  await refreshContactsSidebar();
}

function resolveRouteChatTab(): "pending" | "assigned" {
  return route.query.tab === "pending" ? "pending" : "assigned";
}

// Infinite scroll for messages (load older at top)
const messagesScroll = useInfiniteScroll({
  direction: "top",
  onLoadMore: async () => {
    if (!contactsStore.currentContact) return;
    await messagesScroll.preserveScrollPosition(async () => {
      const accountFilter = selectedAccountFilter(selectedAccount.value);
      await contactsStore.fetchOlderMessages(
        contactsStore.currentContact!.id,
        accountFilter,
      );
      await nextTick();
      // Load media for any new messages
      try {
        loadMediaForMessages();
      } catch (e) {
        console.error("Error loading media:", e);
      }
    });
  },
  hasMore: computed(() => contactsStore.hasMoreMessages),
  isLoading: computed(() => contactsStore.isLoadingOlderMessages),
  onScroll: (event) => updateStickyDate(event.target as HTMLElement),
});

const virtualMessages = useVirtualMessageList({
  items: computed(() => contactsStore.messages),
  getViewport: () => messagesScroll.getViewport(),
  estimatedHeight: 80,
  buffer: 15,
});

const virtualResizeObserver = new ResizeObserver((entries) => {
  for (const entry of entries) {
    const id = (entry.target as HTMLElement).dataset.virtualId;
    if (id) {
      const h = entry.borderBoxSize?.[0]?.blockSize ?? entry.contentRect.height;
      virtualMessages.reportHeight(id, h);
    }
  }
});

function observeVirtualItem(el: Element | ComponentPublicInstance | null, _id: string) {
  if (el instanceof HTMLElement) {
    virtualResizeObserver.observe(el);
  }
}

const contactId = computed(() => route.params.contactId as string | undefined);
const sidebarContacts = computed(() =>
  chatSidebarUnifier.buildEntries(
    contactsStore.sortedContacts,
    chatSidebarViewMode.value,
  ),
);
const currentSidebarEntry = computed(() => {
  if (!contactsStore.currentContact) return null;
  return (
    chatSidebarUnifier.findEntryByContactID(
      sidebarContacts.value,
      contactsStore.currentContact.id,
    ) || null
  );
});


// Get active transfer for current contact from the store (reactive)
const activeTransfer = computed(() => {
  if (!contactsStore.currentContact) return null;
  return transfersStore.getActiveTransferForContact(
    contactsStore.currentContact.id,
  );
});

const activeTransferId = computed(() => activeTransfer.value?.id || null);
const isCurrentGroupChat = computed(() =>
  isGroupContact(contactsStore.currentContact),
);

const messageContentHelpers = useMessageContent(
  computed(() => contactsStore.contacts),
  computed(() => contactsStore.pendingChats),
  computed(() => contactsStore.assignedChats),
  computed(() => contactsStore.closedChats),
  computed(() => contactsStore.messages),
  computed(() => contactsStore.currentContact),
  isCurrentGroupChat,
);
const {
  isDeletedMessage,
  isSystemEventMessage,
  shouldShowGroupSenderPhone,
  getGroupSenderPhone,
  getMessageContent,
  getLocationData,
  getContactsData,
  getInteractiveButtons,
  getCTAUrlData,
  isMediaMessage,
  shouldShowDateSeparator,
  getDateLabel,
  formatMessageTime,
  getReplyAuthorLabel,
  getReplyingToAuthorLabel,
  shouldShowReplyPreviewThumbnail,
  getReplyPreviewMediaURL,
  getReplyPreviewContent,
  resolveReplyPreviewMediaType,
  resolveMentionsForCurrentMessages,
  preloadMentionResolverFromKnownContacts,
} = messageContentHelpers;
const isCurrentChatClosed = computed(
  () => contactsStore.currentContact?.status === "closed",
);

const currentUserUnclaimedAccess = computed(() => {
  const restrictions = authStore.user?.settings?.send_restrictions || {};
  const allowSend = restrictions.allow_unclaimed_chat_send === true;
  const allowView =
    restrictions.allow_unclaimed_chat_view === true || allowSend;
  return {
    allowView,
    allowSend,
  };
});

const isCurrentChatPendingUnassigned = computed(() => {
  if (!contactsStore.currentContact) return false;
  if (contactsStore.currentContact.is_public === true) return false;
  if (contactsStore.currentContact.is_collaborator === true) return false;
  return (
    contactsStore.currentContact.status === "pending" ||
    !contactsStore.currentContact.assigned_user_id
  );
});

const isCurrentChatRestricted = computed(() => {
  if (!contactsStore.currentContact) return false;
  if (isAdminUser.value) return false;
  if (contactsStore.isMessageAccessRestricted) return true;
  if (!isCurrentChatPendingUnassigned.value) return false;
  return !currentUserUnclaimedAccess.value.allowView;
});
const isCurrentChatSendRestricted = computed(() => {
  if (!contactsStore.currentContact) return false;
  if (isAdminUser.value) return false;
  if (!isCurrentChatPendingUnassigned.value) return false;
  return !currentUserUnclaimedAccess.value.allowSend;
});
const canClaimCurrentChat = computed(() => {
  if (!contactsStore.currentContact || isAdminUser.value) return false;
  return isCurrentChatPendingUnassigned.value;
});
const canCloseCurrentChat = computed(() => {
  const contact = contactsStore.currentContact;
  if (!contact || contact.status !== "open") return false;
  if (isAdminUser.value) return true;
  return !!authStore.user?.id && contact.assigned_user_id === authStore.user.id;
});
const canReopenCurrentChat = computed(() => {
  const contact = contactsStore.currentContact;
  if (!contact || contact.status !== "closed") return false;
  if (isAdminUser.value) return true;
  return (
    authStore.hasPermission("chat.assign", "write") ||
    authStore.hasPermission("contacts", "write") ||
    authStore.hasPermission("chat", "write")
  );
});
const canToggleCurrentChatPublic = computed(() => {
  if (!contactsStore.currentContact) return false;
  if (isAdminUser.value) return true;
  return (
    authStore.hasPermission("chat.assign", "write") ||
    authStore.hasPermission("contacts", "write") ||
    authStore.hasPermission("chat", "write")
  );
});

const chatMessaging = useChatMessaging(
  computed(() => contactsStore.currentContact),
  {
    isCurrentChatSendRestricted,
    isCurrentChatClosed,
    resolveOutboundInstanceID,
    resolveOutboundWhatsAppAccount,
    scrollToBottom,
    addMessage: (message: Message) => contactsStore.addMessage(message),
    loadMediaForMessage,
    openChatMediaViewer,
    resolveMessageBlobForBatchPrint,
    isBatchPrintSelectionMode,
    isBatchPrintBubbleSelectable,
    isModifiedPointerEvent,
    handleMessageBubbleClickForBatchPrint,
  },
);
const {
  selectedMediaUploads,
  isMediaDialogOpen,
  mediaCaption,
  isUploadingMedia,
  cannedPickerOpen,
  cannedSearchQuery,
  pendingCannedResponse,
  emojiPickerOpen,
  reactionPickerMessageId,
  quickReactionEmojis,
  retryingMessageId,
  revokingMessageId,
  selectedMediaCount,
  activeMediaUpload,
  canApplyMediaCaption,
  mediaDialogDescription,
  mediaSendButtonLabel,
  mediaUploadingLabel,
  hasPendingCannedAttachments,
  closeCannedPicker,
  clearPendingCannedAttachments,
  removePendingCannedAttachment,
  sendReaction,
  openFilePicker,
  handleFileSelect,
  closeMediaDialog,
  handleMediaDialogOpenChange,
  sendMediaMessage,
  setActiveMediaPreview,
  removeSelectedMediaUpload,
} = chatMessaging;

const typedActiveMediaUpload = computed(() =>
  activeMediaUpload.value as PendingMediaUpload | null,
);
const typedSelectedMediaUploads = computed(() =>
  selectedMediaUploads.value as PendingMediaUpload[],
);

const canAssignContacts = computed(() => {
  return (
    authStore.hasPermission("chat.assign", "write") ||
    authStore.hasPermission("contacts", "write")
  );
});

const canReadCustomActions = computed(() => {
  return authStore.hasPermission("custom_actions", "read");
});

const chatActions = useChatActions(
  computed(() => contactsStore.currentContact),
  {
    isAdminUser,
    refreshContactsSidebar,
    canReadCustomActions,
  },
);
const {
  customActions,
  executingActionId,
  isTransferring,
  isResuming,
  isAssignDialogOpen,
  assignSearchQuery,
  isClaimingCurrentChat,
  isClosingCurrentChat,
  isReopeningCurrentChat,
  isUpdatingCurrentChatPublic,
  fetchCustomActions,
  executeCustomAction,
  transferToAgent,
  assignContactToUser,
  claimCurrentChat,
  closeCurrentChat,
  reopenCurrentChat,
} = chatActions;

const { focusedSidebarIndex } = useChatKeyboardNav({
  sidebarContacts,
  isInfoPanelOpen,
  isNotesPanelOpen,
  isAssignDialogOpen,
  isMediaDialogOpen,
  isAddContactOpen,
  isProfilePhotoDialogOpen,
  isChatMediaViewerOpen,
  cannedPickerOpen,
  emojiPickerOpen,
  replyingTo: computed(() => contactsStore.replyingTo),
  onContactSelect: handleContactClick,
  onEscapeReply: () => contactsStore.clearReplyingTo(),
  onEscapeCanned: () => {
    cannedPickerOpen.value = false;
    cannedSearchQuery.value = "";
  },
  onEscapeEmoji: () => {
    emojiPickerOpen.value = false;
  },
});

const assignableUsers = computed(() => {
  const instanceId = contactsStore.currentContact?.instance_id?.trim();
  return usersStore.users
    .filter((u) => u.is_active !== false)
    .filter((u) => canUserAccessInstance(u, instanceId));
});

function getAssignedAgentName(contact: Contact): string {
  const providedName =
    typeof contact.assigned_user_name === "string"
      ? contact.assigned_user_name.trim()
      : "";
  if (providedName) {
    return providedName;
  }

  const assignedUserID =
    typeof contact.assigned_user_id === "string"
      ? contact.assigned_user_id.trim()
      : "";
  if (!assignedUserID) {
    return t("chat.unassigned");
  }

  const assignedUser = usersStore.users.find(
    (user) => user.id === assignedUserID,
  );
  if (assignedUser?.full_name) {
    return assignedUser.full_name;
  }

  return assignedUserID;
}

async function toggleTagFilter(tagName: string) {
  const index = contactsStore.selectedTags.indexOf(tagName);
  if (index === -1) {
    contactsStore.selectedTags.push(tagName);
  } else {
    contactsStore.selectedTags.splice(index, 1);
  }
  // Refetch contacts with new filter
  await refreshContactsSidebar();
}

async function toggleChatTypeFilter(chatType: ChatTypeFilter) {
  const index = contactsStore.selectedChatTypes.indexOf(chatType);
  if (index === -1) {
    contactsStore.selectedChatTypes.push(chatType);
  } else {
    contactsStore.selectedChatTypes.splice(index, 1);
  }
  await refreshContactsSidebar();
}

async function updateInstanceFilter(event: Event) {
  const target = event.target as HTMLSelectElement;
  contactsStore.selectedInstanceId = target.value;
  await refreshContactsSidebar();
}

async function clearTagFilter() {
  contactsStore.selectedTags = [];
  contactsStore.selectedChatTypes = [];
  contactsStore.selectedInstanceId = "";
  await refreshContactsSidebar();
}

async function clearInstanceFilter() {
  contactsStore.selectedInstanceId = "";
  await refreshContactsSidebar();
}

function getChatTypeLabel(chatType: ChatTypeFilter): string {
  if (chatType === "group") return t("chat.groupChats");
  if (chatType === "channel") return t("chat.channelChats");
  return t("chat.privateChats");
}

const activeFilterCount = computed(() => {
  return (
    contactsStore.selectedTags.length +
    contactsStore.selectedChatTypes.length +
    (contactsStore.selectedInstanceId ? 1 : 0)
  );
});

const selectedInstanceName = computed(() => {
  if (!contactsStore.selectedInstanceId) return "";
  const selected = instancesStore.instances.find(
    (item) => item.id === contactsStore.selectedInstanceId,
  );
  return selected?.name || contactsStore.selectedInstanceId;
});

const filteredAssignableUsers = computed(() => {
  const query = assignSearchQuery.value.toLowerCase().trim();
  if (!query) return assignableUsers.value;
  return assignableUsers.value.filter(
    (u) =>
      u.full_name.toLowerCase().includes(query) ||
      u.email.toLowerCase().includes(query),
  );
});

function handleCollaboratorInvite(payload: any) {
  const contactId =
    typeof payload?.contact_id === "string" ? payload.contact_id : "";
  if (!contactId) return;
  toast.info(t("chat.collaboratorInviteToastTitle"), {
    description: t("chat.collaboratorInviteToastDesc"),
  });
  contactsStore.fetchContact(contactId).catch(() => {});
  void refreshContactsSidebar();
}

function handleCollaboratorUpdate(payload: any) {
  const contactId =
    typeof payload?.contact_id === "string" ? payload.contact_id : "";
  if (!contactId) return;
  if (contactsStore.currentContact?.id === contactId) {
    contactsStore.fetchContact(contactId).catch(() => {});
  }
  void refreshContactsSidebar();
}

// Fetch contacts on mount (WebSocket is connected in AppLayout)
onMounted(async () => {
  refreshChatSidebarViewModePreference();
  window.addEventListener("storage", refreshChatSidebarViewModePreference);

  // Ensure auth session is restored
  if (!authStore.isAuthenticated) {
    await authStore.restoreSession();
  }

  contactsStore.setActiveChatTab(resolveRouteChatTab());
  await contactsStore.fetchChats({
    assigned_to: isAgentUser.value ? "me" : undefined,
  });

  // Fetch instances for sidebar instance tags
  instancesStore.fetchInstances();

  // Setup infinite scroll for contacts list
  await nextTick();
  contactsScroll.setup();
  await hydrateContactsSidebarUntilScrollable();

  // Fetch transfers to track active transfers
  transfersStore.fetchTransfers({ status: "active" });

  // Fetch users if can assign contacts
  if (canAssignContacts.value) {
    usersStore.fetchUsers().catch(() => {
      // Silently fail if user list can't be loaded
    });
  }

  // Fetch custom actions when permitted
  if (canReadCustomActions.value) {
    fetchCustomActions();
  }

  // Fetch available tags for filtering (if not already loaded)
  if (tagsStore.tags.length === 0) {
    tagsStore.fetchTags().catch(() => {});
  }

  wsService.subscribe("chat_collaborator_invite", handleCollaboratorInvite);
  wsService.subscribe("chat_collaborator_update", handleCollaboratorUpdate);
  wsService.subscribe("contact_typing", handleContactTyping);

  if (contactId.value) {
    const selectionSequence = ++contactSelectionSequence;
    await selectContact(contactId.value, selectionSequence);
  }
});

watch(
  () => route.query.tab,
  async () => {
    const nextTab = resolveRouteChatTab();
    if (contactsStore.activeChatTab === nextTab) {
      return;
    }

    contactsStore.setActiveChatTab(nextTab);
    await refreshContactsSidebar();
  },
);

onUnmounted(() => {
  wsService.unsubscribe("chat_collaborator_invite", handleCollaboratorInvite);
  wsService.unsubscribe("chat_collaborator_update", handleCollaboratorUpdate);
  wsService.unsubscribe("contact_typing", handleContactTyping);
  if (typingIndicatorTimer) {
    clearTimeout(typingIndicatorTimer);
    typingIndicatorTimer = null;
  }
  const activeContact = contactsStore.currentContact;
  stopTypingForContact(activeContact, { force: true, ...typingContext() });
  resetTypingPresenceState();
  contactsSearchPrefetchRunToken++;
  if (contactsSearchRefreshTimer !== null) {
    window.clearTimeout(contactsSearchRefreshTimer);
    contactsSearchRefreshTimer = null;
  }
  if (contactsSearchPrefetchTimer !== null) {
    window.clearTimeout(contactsSearchPrefetchTimer);
    contactsSearchPrefetchTimer = null;
  }
  resetBatchPrintSelection();
  stopContactsSidebarResize();
  clearPendingSidebarDeleteConfirmation();
  virtualMessages.cleanup();
  virtualResizeObserver.disconnect();
  window.removeEventListener("storage", refreshChatSidebarViewModePreference);
  wsService.setCurrentContact(null);
  // Clear current contact when leaving chat view so notifications work on other pages
  contactsStore.setCurrentContact(null);
  notesStore.clearNotes();
  resetMediaLoadingPipeline();
  cleanupBlobUrls();
  if (stickyDateTimeout) clearTimeout(stickyDateTimeout);
  if (quoteHighlightTimeout) clearTimeout(quoteHighlightTimeout);
});

function updateStickyDate(scrollContainer: HTMLElement) {
  // Find all date separator elements
  const dateSeparators = scrollContainer.querySelectorAll(
    "[data-date-separator]",
  );
  if (dateSeparators.length === 0) return;

  const containerRect = scrollContainer.getBoundingClientRect();
  const containerTop = containerRect.top + 60; // Offset for sticky header position

  // Find the last date separator that's above the viewport top
  let currentDate = "";
  for (const separator of dateSeparators) {
    const rect = separator.getBoundingClientRect();
    if (rect.top < containerTop) {
      currentDate = separator.getAttribute("data-date-separator") || "";
    } else {
      break;
    }
  }

  // Show sticky date if we have scrolled past at least one date separator
  if (currentDate && scrollContainer.scrollTop > 50) {
    stickyDate.value = currentDate;
    showStickyDate.value = true;

    // Hide after scrolling stops
    if (stickyDateTimeout) clearTimeout(stickyDateTimeout);
    stickyDateTimeout = setTimeout(() => {
      showStickyDate.value = false;
    }, 1500);
  } else {
    showStickyDate.value = false;
  }
}

// Watch for route changes
watch(contactId, (newId) => {
  const previousContact = contactsStore.currentContact;
  stopTypingForContact(previousContact, { force: true, ...typingContext() });
  resetTypingPresenceState();
  const selectionSequence = ++contactSelectionSequence;
  resetBatchPrintSelection();
  resetMediaLoadingPipeline();
  contactSessionData.value = null;
  isInfoPanelOpen.value = false;

  if (newId) {
    notesStore.clearNotes();
    const prefetchedContact =
      contactsStore.contacts.find((contact) => contact.id === newId) || null;
    contactsStore.setCurrentContact(prefetchedContact);
    contactsStore.clearMessages();
    void selectContact(newId, selectionSequence);
  } else {
    wsService.setCurrentContact(null);
    contactsStore.setCurrentContact(null);
    contactsStore.clearMessages();
    notesStore.clearNotes();
  }
});

async function selectContact(id: string, selectionSequence: number) {
  const isSelectionStale = () => selectionSequence !== contactSelectionSequence;
  resetMediaLoadingPipeline();
  resetBatchPrintSelection();
  let contact = contactsStore.contacts.find((c) => c.id === id);
  if (!contact) {
    const fetched = await contactsStore.fetchContact(id);
    if (isSelectionStale()) return;
    if (!fetched) return;
    contact = fetched;
  }
  if (isSelectionStale()) return;
  const sidebarEntry = chatSidebarUnifier.findEntryByContactID(
    sidebarContacts.value,
    contact.id,
  );
  const hasUnifiedAccountGroup = Boolean(
    isSidebarUnifiedMode.value &&
    sidebarEntry &&
    (sidebarEntry.sourceContactIDs.length > 1 ||
      sidebarEntry.accountNames.length > 1),
  );
  let activeContact = contact;

  // Remove old scroll listener before switching contacts
  messagesScroll.cleanup();
  virtualMessages.cleanup();

  // Reset account selection when switching contacts
  selectedAccount.value = null;
  contactAccounts.value = [];
  contactsStore.setAccountFilter(null);

  if (
    isSidebarUnifiedMode.value &&
    sidebarEntry &&
    sidebarEntry.sourceContactIDs.length > 1
  ) {
    contactAccounts.value = Array.from(
      new Set(
        sidebarEntry.sourceContactIDs.map((sourceID) =>
          toContactToggleKey(sourceID),
        ),
      ),
    );
  } else if (
    isSidebarUnifiedMode.value &&
    sidebarEntry &&
    sidebarEntry.accountNames.length > 0
  ) {
    contactAccounts.value = sidebarEntry.accountNames.map((accountName) =>
      toAccountToggleKey(accountName),
    );
  }

  if (hasUnifiedAccountGroup && sidebarEntry) {
    const requestedContactKey = toContactToggleKey(contact.id);
    const requestedAccount =
      typeof contact.whatsapp_account === "string"
        ? contact.whatsapp_account.trim()
        : "";
    const requestedAccountKey = requestedAccount
      ? toAccountToggleKey(requestedAccount)
      : "";

    const selected =
      (requestedContactKey &&
        contactAccounts.value.includes(requestedContactKey) &&
        requestedContactKey) ||
      (requestedAccountKey &&
        contactAccounts.value.includes(requestedAccountKey) &&
        requestedAccountKey) ||
      contactAccounts.value[0];
    selectedAccount.value = selected || null;

    const selectedSourceContact = resolveSourceContactForToggle(
      sidebarEntry,
      selectedAccount.value,
    );
    if (selectedSourceContact) {
      activeContact = selectedSourceContact;
    }
    contactsStore.setAccountFilter(
      selectedAccountFilter(selectedAccount.value) || null,
    );
  }

  contactsStore.setCurrentContact(activeContact);

  const allowViewWithoutClaim = (() => {
    const restrictions = authStore.user?.settings?.send_restrictions || {};
    const allowSend = restrictions.allow_unclaimed_chat_send === true;
    return restrictions.allow_unclaimed_chat_view === true || allowSend;
  })();
  const isRestrictedForNonAdmin =
    !isAdminUser.value &&
    activeContact.is_public !== true &&
    (activeContact.status === "pending" || !activeContact.assigned_user_id) &&
    !allowViewWithoutClaim;
  if (isRestrictedForNonAdmin) {
    contactsStore.clearMessages();
    wsService.setCurrentContact(null);
    return;
  }

  const initialAccountFilter = selectedAccountFilter(selectedAccount.value);
  await contactsStore.fetchMessages(
    activeContact.id,
    initialAccountFilter
      ? {
          account: initialAccountFilter,
        }
      : undefined,
  );
  if (isSelectionStale()) return;

  if (isSidebarUnifiedMode.value) {
    // Fallback for legacy per-contact multi-account histories.
    if (!hasUnifiedAccountGroup) {
      const accounts = new Set<string>();
      for (const msg of contactsStore.messages) {
        if (msg.whatsapp_account) accounts.add(msg.whatsapp_account);
      }
      contactAccounts.value = Array.from(accounts)
        .sort()
        .map((accountName) => toAccountToggleKey(accountName));

      // Auto-select account if multi-account contact
      if (contactAccounts.value.length > 1) {
        // Find account of the most recent incoming message
        for (let i = contactsStore.messages.length - 1; i >= 0; i--) {
          const msg = contactsStore.messages[i];
          if (msg.direction === "incoming" && msg.whatsapp_account) {
            selectedAccount.value = toAccountToggleKey(msg.whatsapp_account);
            break;
          }
        }
        // Fallback to contact's default account
        if (!selectedAccount.value) {
          const preferredAccount =
            (activeContact.whatsapp_account || "").trim() ||
            selectedAccountFilter(contactAccounts.value[0]) ||
            "";
          if (preferredAccount) {
            selectedAccount.value = toAccountToggleKey(preferredAccount);
          }
        }
        // Re-fetch messages filtered by selected account
        const refreshedAccountFilter = selectedAccountFilter(
          selectedAccount.value,
        );
        if (refreshedAccountFilter) {
          contactsStore.setAccountFilter(refreshedAccountFilter);
          await contactsStore.fetchMessages(activeContact.id, {
            account: refreshedAccountFilter,
          });
          if (isSelectionStale()) return;
        }
      }
    }
  } else {
    // Separate mode should always focus a single account thread.
    const discoveredAccounts = Array.from(
      new Set(
        contactsStore.messages
          .map((msg) => msg.whatsapp_account || "")
          .filter(Boolean),
      ),
    ).sort();

    if (discoveredAccounts.length > 0) {
      const preferredFromContact =
        typeof activeContact.whatsapp_account === "string"
          ? activeContact.whatsapp_account.trim()
          : "";
      let preferredAccount = discoveredAccounts.includes(preferredFromContact)
        ? preferredFromContact
        : "";

      if (!preferredAccount) {
        for (let i = contactsStore.messages.length - 1; i >= 0; i--) {
          const msg = contactsStore.messages[i];
          if (msg.direction === "incoming" && msg.whatsapp_account) {
            preferredAccount = msg.whatsapp_account;
            break;
          }
        }
      }

      if (!preferredAccount) {
        preferredAccount = discoveredAccounts[0];
      }

      selectedAccount.value = toAccountToggleKey(preferredAccount);
      contactAccounts.value = [toAccountToggleKey(preferredAccount)];
      contactsStore.setAccountFilter(preferredAccount);
      await contactsStore.fetchMessages(activeContact.id, {
        account: preferredAccount,
      });
      if (isSelectionStale()) return;
    } else {
      selectedAccount.value = null;
      contactAccounts.value = [];
      contactsStore.setAccountFilter(null);
    }
  }

  if (isSelectionStale()) return;
  // Tell WebSocket server which contact we're viewing.
  wsService.setCurrentContact(activeContact.id);
  // Wait for DOM to render messages before scrolling
  await nextTick();
  if (isSelectionStale()) return;
  // Load media for messages after messages are fetched
  try {
    loadMediaForMessages();
  } catch (e) {
    console.error("Error loading media:", e);
  }
  // Scroll after a brief delay to ensure content is rendered (instant on initial load)
  setTimeout(() => {
    if (isSelectionStale()) return;
    scrollToBottom(true);
    // Setup scroll listener for infinite scroll after initial scroll
    messagesScroll.setup();
    // Setup virtual scroll on same viewport
    virtualMessages.setup();
  }, 50);

  // Fetch notes for badge count
  void notesStore.fetchNotes(activeContact.id);

  // Fetch session data and auto-open panel if configured
  try {
    const response = await contactsService.getSessionData(activeContact.id);
    if (isSelectionStale()) return;
    contactSessionData.value = response.data.data || response.data;
    if (contactSessionData.value?.panel_config?.sections?.length > 0) {
      isInfoPanelOpen.value = true;
    }
  } catch {
    if (isSelectionStale()) return;
    contactSessionData.value = null;
  }
}

async function retryLoadMessages() {
  const contact = contactsStore.currentContact;
  if (!contact) return;
  const accountFilterValue = selectedAccountFilter(selectedAccount.value);
  await contactsStore.fetchMessages(
    contact.id,
    accountFilterValue ? { account: accountFilterValue } : undefined,
  );
  if (contactsStore.messageLoadError) return;
  loadMediaForMessages();
  await nextTick();
  scrollToBottom(true);
}

// Watch for new messages to auto-scroll and load media
watch(
  () => contactsStore.messages.length,
  () => {
    void resolveMentionsForCurrentMessages();
    if (
      contactsStore.isLoadingOlderMessages ||
      isQuoteNavigationInProgress.value
    ) {
      return;
    }
    scrollToBottom();
    try {
      loadMediaForMessages();
    } catch (e) {
      console.error("Error loading media:", e);
    }
  },
);

// Watch for messages changes to load media
watch(
  () => contactsStore.messages,
  () => {
    void resolveMentionsForCurrentMessages();
    if (selectedBatchPrintMessageIds.value.length > 0) {
      const availableMessageIDs = new Set(
        contactsStore.messages.map((message) => message.id),
      );
      selectedBatchPrintMessageIds.value =
        selectedBatchPrintMessageIds.value.filter((id) =>
          availableMessageIDs.has(id),
        );
    }
    try {
      loadMediaForMessages();
    } catch (e) {
      console.error("Error loading media:", e);
    }
  },
  { deep: true },
);

watch(
  () => [
    contactsStore.contacts,
    contactsStore.pendingChats,
    contactsStore.assignedChats,
    contactsStore.closedChats,
  ],
  () => {
    preloadMentionResolverFromKnownContacts();
  },
  { deep: true, immediate: true },
);

watch(
  () => contactsStore.searchQuery,
  (value) => {
    contactsSearchPrefetchRunToken++;

    if (contactsSearchRefreshTimer !== null) {
      window.clearTimeout(contactsSearchRefreshTimer);
      contactsSearchRefreshTimer = null;
    }
    if (contactsSearchPrefetchTimer !== null) {
      window.clearTimeout(contactsSearchPrefetchTimer);
      contactsSearchPrefetchTimer = null;
    }

    const trimmedValue = value.trim();
    const token = contactsSearchPrefetchRunToken;
    contactsSearchRefreshTimer = window.setTimeout(() => {
      contactsSearchRefreshTimer = null;
      void (async () => {
        if (token !== contactsSearchPrefetchRunToken) return;
        await refreshContactsSidebar();
        if (token !== contactsSearchPrefetchRunToken) return;

        if (!trimmedValue) {
          return;
        }

        contactsSearchPrefetchTimer = window.setTimeout(() => {
          contactsSearchPrefetchTimer = null;
          void prefetchSearchResultsIfNeeded(trimmedValue, token);
        }, CONTACTS_SEARCH_PREFETCH_DEBOUNCE_MS);
      })();
    }, CONTACTS_SEARCH_REFRESH_DEBOUNCE_MS);
  },
);

async function switchAccount(accountToggleKey: string) {
  if (!isSidebarUnifiedMode.value) return;
  if (
    !contactsStore.currentContact ||
    accountToggleKey === selectedAccount.value
  ) {
    return;
  }

  const sidebarEntry = currentSidebarEntry.value;
  const accountFilter = selectedAccountFilter(accountToggleKey);
  const selectedContactID = contactIDFromToggleKey(accountToggleKey);

  if (sidebarEntry) {
    if (accountFilter && sidebarEntry.contactsByAccount[accountFilter]) {
      const targetContact = sidebarEntry.contactsByAccount[accountFilter];
      selectedAccount.value = accountToggleKey;
      contactsStore.setAccountFilter(accountFilter);
      if (targetContact.id !== contactsStore.currentContact.id) {
        await router.push(`/chat/${targetContact.id}`);
        return;
      }
    } else if (selectedContactID) {
      const targetContact = findSidebarEntrySourceContact(
        sidebarEntry,
        selectedContactID,
      );
      if (targetContact) {
        selectedAccount.value = accountToggleKey;
        contactsStore.setAccountFilter(null);
        if (targetContact.id !== contactsStore.currentContact.id) {
          await router.push(`/chat/${targetContact.id}`);
          return;
        }
      }
    }
  }

  selectedAccount.value = accountToggleKey;
  contactsStore.setAccountFilter(accountFilter || null);
  resetMediaLoadingPipeline();
  await contactsStore.fetchMessages(
    contactsStore.currentContact.id,
    accountFilter ? { account: accountFilter } : undefined,
  );
  await nextTick();
  try {
    loadMediaForMessages();
  } catch (e) {
    console.error("Error loading media:", e);
  }
  scrollToBottom(true);
}

function handleContactClick(entry: SidebarContactEntry) {
  clearPendingSidebarDeleteConfirmation();
  clearPendingSidebarSoftDeleteConfirmation();
  const target = getSidebarEntryPreferredContact(entry);
  router.push(`/chat/${target.id}`);
}

function handleContactTyping(payload: any) {
  const contactId = typeof payload?.contact_id === "string" ? payload.contact_id : "";
  const state = typeof payload?.state === "string" ? payload.state : "";
  const currentContact = contactsStore.currentContact;
  if (!contactId || !currentContact || contactId !== currentContact.id) return;

  if (state === "composing") {
    typingContactId.value = contactId;
    typingContactName.value = currentContact.profile_name || currentContact.name || currentContact.phone_number || "";
    if (typingIndicatorTimer) clearTimeout(typingIndicatorTimer);
    typingIndicatorTimer = setTimeout(() => {
      typingContactId.value = null;
      typingContactName.value = "";
      typingIndicatorTimer = null;
    }, TYPING_INDICATOR_TIMEOUT_MS);
  } else {
    typingContactId.value = null;
    typingContactName.value = "";
  }
}

function clearPendingSidebarDeleteConfirmation() {
  pendingSidebarDeleteEntryKey.value = null;
  if (pendingSidebarDeleteTimeout) {
    clearTimeout(pendingSidebarDeleteTimeout);
    pendingSidebarDeleteTimeout = null;
  }
}

function clearPendingSidebarSoftDeleteConfirmation() {
  pendingSidebarSoftDeleteEntryKey.value = null;
  if (pendingSidebarSoftDeleteTimeout) {
    clearTimeout(pendingSidebarSoftDeleteTimeout);
    pendingSidebarSoftDeleteTimeout = null;
  }
}

function armSidebarDeleteConfirmation(entryKey: string) {
  pendingSidebarDeleteEntryKey.value = entryKey;
  if (pendingSidebarDeleteTimeout) {
    clearTimeout(pendingSidebarDeleteTimeout);
  }
  pendingSidebarDeleteTimeout = setTimeout(() => {
    pendingSidebarDeleteEntryKey.value = null;
    pendingSidebarDeleteTimeout = null;
  }, SIDEBAR_DELETE_CONFIRM_TIMEOUT_MS);
}

function armSidebarSoftDeleteConfirmation(entryKey: string) {
  pendingSidebarSoftDeleteEntryKey.value = entryKey;
  if (pendingSidebarSoftDeleteTimeout) {
    clearTimeout(pendingSidebarSoftDeleteTimeout);
  }
  pendingSidebarSoftDeleteTimeout = setTimeout(() => {
    pendingSidebarSoftDeleteEntryKey.value = null;
    pendingSidebarSoftDeleteTimeout = null;
  }, SIDEBAR_DELETE_CONFIRM_TIMEOUT_MS);
}

async function deleteSidebarEntry(entry: SidebarContactEntry) {
  if (!isAdminUser.value || deletingSidebarEntryKey.value) return;
  clearPendingSidebarSoftDeleteConfirmation();

  const contactIDs = Array.from(
    new Set(entry.sourceContactIDs.filter((id): id is string => Boolean(id))),
  );
  if (contactIDs.length === 0) return;
  if (pendingSidebarDeleteEntryKey.value !== entry.key) {
    armSidebarDeleteConfirmation(entry.key);
    return;
  }
  clearPendingSidebarDeleteConfirmation();

  deletingSidebarEntryKey.value = entry.key;
  try {
    await Promise.all(contactIDs.map((id) => contactsService.delete(id)));

    const currentContactID = contactsStore.currentContact?.id || null;
    const deletedCurrentContact = currentContactID
      ? contactIDs.includes(currentContactID)
      : false;

    if (deletedCurrentContact) {
      stopTypingForContact(contactsStore.currentContact, { force: true, ...typingContext() });
      resetTypingPresenceState();
      wsService.setCurrentContact(null);
      contactsStore.setCurrentContact(null);
      contactsStore.clearMessages();
      notesStore.clearNotes();
      contactSessionData.value = null;
      isInfoPanelOpen.value = false;
      isNotesPanelOpen.value = false;
    }

    await refreshContactsSidebar();

    if (
      deletedCurrentContact ||
      (contactId.value && contactIDs.includes(contactId.value))
    ) {
      await router.push("/chat");
    }

    toast.success(
      t("common.deletedSuccess", {
        resource:
          contactIDs.length > 1
            ? t("resources.contacts")
            : t("resources.Contact"),
      }),
    );
  } catch (error: any) {
    const message =
      error?.response?.data?.message ||
      t("common.failedDelete", { resource: t("resources.contact") });
    toast.error(message);
  } finally {
    deletingSidebarEntryKey.value = null;
  }
}

async function softDeleteSidebarEntry(entry: SidebarContactEntry) {
  if (!canSoftDeleteChats.value || softDeletingSidebarEntryKey.value) return;
  clearPendingSidebarDeleteConfirmation();

  const contactIDs = Array.from(
    new Set(entry.sourceContactIDs.filter((id): id is string => Boolean(id))),
  );
  if (contactIDs.length === 0) return;
  if (pendingSidebarSoftDeleteEntryKey.value !== entry.key) {
    armSidebarSoftDeleteConfirmation(entry.key);
    return;
  }
  clearPendingSidebarSoftDeleteConfirmation();

  softDeletingSidebarEntryKey.value = entry.key;
  try {
    await Promise.all(contactIDs.map((id) => contactsService.softDelete(id)));

    const currentContactID = contactsStore.currentContact?.id || null;
    const deletedCurrentContact = currentContactID
      ? contactIDs.includes(currentContactID)
      : false;

    if (deletedCurrentContact) {
      stopTypingForContact(contactsStore.currentContact, { force: true, ...typingContext() });
      resetTypingPresenceState();
      wsService.setCurrentContact(null);
      contactsStore.setCurrentContact(null);
      contactsStore.clearMessages();
      notesStore.clearNotes();
      contactSessionData.value = null;
      isInfoPanelOpen.value = false;
      isNotesPanelOpen.value = false;
    }

    await refreshContactsSidebar();

    if (
      deletedCurrentContact ||
      (contactId.value && contactIDs.includes(contactId.value))
    ) {
      await router.push("/chat");
    }

    toast.success(t("chat.softDeleteSuccess"));
  } catch (error: any) {
    const message =
      error?.response?.data?.message || t("chat.softDeleteFailed");
    toast.error(message);
  } finally {
    softDeletingSidebarEntryKey.value = null;
  }
}

function openProfilePhotoDialog(contact: Contact | null) {
  if (!contact) return;
  profilePhotoContact.value = contact;
  isProfilePhotoDialogOpen.value = true;
}

function handleProfilePhotoDialogOpenChange(open: boolean) {
  isProfilePhotoDialogOpen.value = open;
  if (!open) {
    profilePhotoContact.value = null;
  }
}

async function handleContactDeleted(contactId: string) {
  if (contactsStore.currentContact?.id === contactId) {
    stopTypingForContact(contactsStore.currentContact, { force: true, ...typingContext() });
    resetTypingPresenceState();
    wsService.setCurrentContact(null);
    contactsStore.setCurrentContact(null);
    contactsStore.clearMessages();
    notesStore.clearNotes();
    isInfoPanelOpen.value = false;
    await refreshContactsSidebar();
    router.push("/chat");
    return;
  }

  await refreshContactsSidebar();
}

async function onContactCreated() {
  isAddContactOpen.value = false;
  await refreshContactsSidebar();
}

async function sendMessage() {
  if (isCurrentChatSendRestricted.value || isCurrentChatClosed.value) return;
  if (!canSendMessage.value || !contactsStore.currentContact) return;

  stopTypingForContact(contactsStore.currentContact, typingContext());
  isSending.value = true;
  try {
    const outboundInstanceID = resolveOutboundInstanceID(
      contactsStore.currentContact,
    );
    const activeAccountFilter = resolveOutboundWhatsAppAccount(
      contactsStore.currentContact,
    );
    if (
      pendingCannedResponse.value &&
      pendingCannedResponse.value.attachments.length > 0
    ) {
      const response = await cannedResponsesService.send(
        pendingCannedResponse.value.id,
        {
          contact_id: contactsStore.currentContact.id,
          content: messageInput.value,
          instance_id: outboundInstanceID,
          reply_to_message_id: contactsStore.replyingTo?.id,
          whatsapp_account: activeAccountFilter,
        },
      );
      const payload = (response.data as any).data || response.data;
      const sentMessages = Array.isArray(payload.messages)
        ? payload.messages
        : [];
      for (const sentMessage of sentMessages) {
        contactsStore.addMessage(sentMessage);
      }
    } else {
      await contactsStore.sendMessage(
        contactsStore.currentContact.id,
        "text",
        { body: messageInput.value },
        contactsStore.replyingTo?.id,
        activeAccountFilter,
        outboundInstanceID,
      );
    }

    messageInput.value = "";
    pendingCannedResponse.value = null;
    contactsStore.clearReplyingTo();
    resetTextareaHeight();
    await nextTick();
    scrollToBottom();
  } catch (error: any) {
    const message = resolveSendErrorMessage(error, t("chat.sendMessageFailed"));
    toast.error(message);
  } finally {
    isSending.value = false;
  }
}


async function retryMessage(message: Message) {
  if (!contactsStore.currentContact || retryingMessageId.value) return;

  retryingMessageId.value = message.id;
  try {
    // Get the message content based on type
    const content = message.content || {};

    await contactsStore.sendMessage(
      contactsStore.currentContact.id,
      message.message_type,
      content,
      undefined,
      message.whatsapp_account ||
        resolveOutboundWhatsAppAccount(contactsStore.currentContact),
      message.instance_id ||
        resolveOutboundInstanceID(contactsStore.currentContact),
    );

    // Remove the failed message from the list after successful retry
    const messages = (contactsStore.messages as any).get?.(
      contactsStore.currentContact.id,
    ) as Message[] | undefined;
    if (messages) {
      const index = messages.findIndex((m: Message) => m.id === message.id);
      if (index !== -1) {
        messages.splice(index, 1);
      }
    }

    toast.success(t("chat.messageSent"));
  } catch (error: any) {
    const message = resolveSendErrorMessage(error, t("chat.sendMessageFailed"));
    toast.error(message);
  } finally {
    retryingMessageId.value = null;
  }
}

function isRevokedMessage(message: Message): boolean {
  if (message.metadata?.revoked === true) {
    return true;
  }
  return isDeletedMessage(message);
}

function canRevokeMessage(message: Message): boolean {
  if (!canRevokeMessages.value) return false;
  if (message.direction !== "outgoing") return false;
  if (message.status === "failed") return false;
  return !isRevokedMessage(message);
}

async function revokeMessage(message: Message) {
  if (
    !contactsStore.currentContact ||
    revokingMessageId.value ||
    !canRevokeMessage(message)
  ) {
    return;
  }

  revokingMessageId.value = message.id;
  try {
    const response = await messagesService.revoke(
      contactsStore.currentContact.id,
      message.id,
    );
    const updatedMessage = (response.data?.data || response.data) as Message;
    contactsStore.patchMessage(updatedMessage);
    toast.success("Message revoked");
  } catch (error: any) {
    const messageText =
      error?.response?.data?.message || "Failed to revoke message";
    toast.error(messageText);
  } finally {
    revokingMessageId.value = null;
  }
}

function resetTextareaHeight() {
  const textarea = messageInputRef.value;
  if (!textarea) return;
  textarea.style.height = "auto";
}

function openReplyPreviewMedia(message: Message, event?: MouseEvent): void {
  if (isBatchPrintSelectionMode.value) return;
  if (isModifiedPointerEvent(event)) return;
  const mediaURL = getReplyPreviewMediaURL(message);
  if (!mediaURL) return;
  event?.preventDefault();
  event?.stopPropagation();
  openChatMediaViewer(
    mediaURL,
    resolveReplyPreviewMediaType(message),
    message.reply_to_message?.media_filename ?? "",
  );
}

function scrollAndHighlightMessageElement(messageEl: HTMLElement): void {
  messageEl.scrollIntoView({ behavior: "smooth", block: "center" });
  messageEl.classList.add("highlight-message");
  if (quoteHighlightTimeout) {
    clearTimeout(quoteHighlightTimeout);
  }
  quoteHighlightTimeout = setTimeout(() => {
    messageEl.classList.remove("highlight-message");
    quoteHighlightTimeout = null;
  }, 2000);
}

async function scrollToMessage(messageId: string | undefined) {
  if (!messageId) return;

  const targetElement = () =>
    document.getElementById(`message-${messageId}`) as HTMLElement | null;

  const existing = targetElement();
  if (existing) {
    scrollAndHighlightMessageElement(existing);
    return;
  }

  // If the message exists in the array but isn't rendered, make it visible
  const msgIndex = contactsStore.messages.findIndex(
    (m: Message) => m.id === messageId,
  );
  if (msgIndex >= 0) {
    await virtualMessages.ensureVisible(msgIndex);
    await nextTick();
    const el = targetElement();
    if (el) {
      scrollAndHighlightMessageElement(el);
      return;
    }
  }

  if (
    !contactsStore.currentContact ||
    isQuoteNavigationInProgress.value ||
    !contactsStore.hasMoreMessages
  ) {
    return;
  }

  const activeContactId = contactsStore.currentContact.id;
  const messageHistoryNavigator = new MessageHistoryNavigator({
    hasMessage: (targetMessageId: string) =>
      contactsStore.currentContact?.id === activeContactId &&
      contactsStore.messages.some(
        (message: Message) => message.id === targetMessageId,
      ),
    hasMoreMessages: () =>
      contactsStore.currentContact?.id === activeContactId &&
      contactsStore.hasMoreMessages,
    getBoundaryToken: () =>
      contactsStore.currentContact?.id === activeContactId
        ? contactsStore.messages[0]?.id || null
        : null,
    loadOlderMessages: async () => {
      if (contactsStore.currentContact?.id !== activeContactId) {
        return;
      }
      const accountFilter = selectedAccountFilter(selectedAccount.value);
      await messagesScroll.preserveScrollPosition(async () => {
        await contactsStore.fetchOlderMessages(activeContactId, accountFilter);
        await nextTick();
      });
    },
  });

  isQuoteNavigationInProgress.value = true;
  try {
    const found = await messageHistoryNavigator.loadUntilMessage(messageId, {
      maxRequests: QUOTE_NAVIGATION_MAX_HISTORY_REQUESTS,
    });
    if (!found) {
      return;
    }

    // Ensure the message is in the virtual window
    const foundIndex = contactsStore.messages.findIndex(
      (m: Message) => m.id === messageId,
    );
    if (foundIndex >= 0) {
      await virtualMessages.ensureVisible(foundIndex);
    }

    await nextTick();
    const loadedElement = targetElement();
    if (!loadedElement) {
      return;
    }
    scrollAndHighlightMessageElement(loadedElement);
  } finally {
    isQuoteNavigationInProgress.value = false;
  }
}

function insertCannedResponse(payload: {
  id: string;
  content: string;
  attachments: CannedResponseAttachment[];
}) {
  messageInput.value = payload.content;
  chatMessaging.insertCannedResponse(payload);
}

function insertEmoji(emoji: string) {
  messageInput.value += emoji;
  emojiPickerOpen.value = false;
}

function resolveSendErrorMessage(error: any, fallbackMessage: string): string {
  const responseData = error?.response?.data || {};
  const details = responseData?.details || responseData?.data?.details || {};
  const reasonCode = String(
    details?.reason_code ||
      responseData?.reason_code ||
      responseData?.data?.reason_code ||
      "",
  ).trim();

  const baseMessage = responseData?.message || fallbackMessage;
  if (reasonCode === "POLICY_NO_INBOUND") {
    return `Cannot send: inbound-only policy is active (${reasonCode})`;
  }
  if (reasonCode === "INSTANCE_BLOCKED") {
    return `Cannot send: selected instance is blocked (${reasonCode})`;
  }
  if (reasonCode === "INSTANCE_NOT_CONNECTED") {
    return `Cannot send: selected instance is not connected (${reasonCode})`;
  }
  if (reasonCode === "POLICY_DRAFT_ONLY") {
    return `Cannot send: campaign draft-only policy is active (${reasonCode})`;
  }
  if (reasonCode !== "") {
    return `${baseMessage} (${reasonCode})`;
  }
  return baseMessage;
}

function _toggleReactionPicker(messageId: string) {
  if (reactionPickerMessageId.value === messageId) {
    reactionPickerMessageId.value = null;
  } else {
    reactionPickerMessageId.value = messageId;
  }
}
void _toggleReactionPicker;

function replyToMessage(message: Message) {
  contactsStore.setReplyingTo(message);
  nextTick(() => {
    messageInputRef.value?.focus();
  });
}

// Watch for slash commands in message input
watch(messageInput, (val) => {
  const activeContact = contactsStore.currentContact;

  if (val.startsWith("/")) {
    const query = val.slice(1); // Remove the leading /
    cannedSearchQuery.value = query;
    cannedPickerOpen.value = true;
    stopTypingForContact(activeContact, typingContext());
  } else if (cannedPickerOpen.value) {
    // Close picker if user removes the /
    cannedPickerOpen.value = false;
    cannedSearchQuery.value = "";
  }

  if (!activeContact) {
    resetTypingPresenceState();
    return;
  }

  if (val.startsWith("/")) {
    return;
  }

  if (isCurrentChatSendRestricted.value || isCurrentChatClosed.value) {
    stopTypingForContact(activeContact, typingContext());
    return;
  }

  if (val.trim() === "") {
    stopTypingForContact(activeContact, typingContext());
    return;
  }

  const ctx = typingContext();
  void sendTypingPresenceForContact(activeContact, "composing", ctx);
  scheduleTypingPaused(activeContact, ctx);
});

function toggleCurrentChatPublicVisibility() {
  return chatActions.toggleCurrentChatPublicVisibility(canToggleCurrentChatPublic.value);
}

function resumeChatbot() {
  return chatActions.resumeChatbot(activeTransferId.value);
}

function scrollToBottom(instant = false) {
  nextTick(() => {
    if (messagesEndRef.value) {
      messagesEndRef.value.scrollIntoView({
        behavior: instant ? "instant" : "smooth",
        block: "end",
      });
    }
  });
}

function handleImageLoad() {
  const viewport = messagesScroll.getViewport();
  if (viewport) {
    // If the user is within 250px of the bottom, keep them pinned to the bottom.
    // This prevents layout jumps when images load asynchronously after opening a chat.
    const isNearBottom =
      viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight < 250;
    if (isNearBottom) {
      scrollToBottom(true);
    }
  }
}

function getMessageStatusIcon(status: string) {
  switch (status) {
    case "sent":
      return Check;
    case "delivered":
      return CheckCheck;
    case "read":
      return CheckCheck;
    case "failed":
      return AlertCircle;
    default:
      return Clock;
  }
}

function getMessageStatusClass(status: string) {
  switch (status) {
    case "read":
      return "text-blue-400"; // Bright blue for read
    case "failed":
      return "text-destructive";
    default:
      return "text-muted-foreground"; // Gray for sent/delivered
  }
}

function formatContactTime(dateStr?: string) {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const now = new Date();
  const diffDays = Math.floor((now.getTime() - date.getTime()) / 86400000);

  if (diffDays === 0) {
    return date.toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
    });
  } else if (diffDays === 1) {
    return "Yesterday";
  } else if (diffDays < 7) {
    return date.toLocaleDateString("en-US", { weekday: "short" });
  }
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

function getAttachmentFilename(message: Message): string {
  return resolveMediaFilename(message);
}

function downloadAttachment(message: Message, event?: MouseEvent) {
  chatMessaging.downloadAttachment(message, event, getMediaBlobUrl);
}

function printAttachment(message: Message, event?: MouseEvent) {
  chatMessaging.printAttachment(message, event, getMediaBlobUrl);
}

function openMediaPreview(message: Message, event?: MouseEvent) {
  chatMessaging.openMediaPreview(message, event, getMediaBlobUrl);
}

function handleImageError(event: Event) {
  const img = event.target as HTMLImageElement;
  img.style.display = "none";
}

function handleMediaError(event: Event, mediaType: string) {
  console.error(`Failed to load ${mediaType}:`, event);
}
</script>

<template>
  <div class="flex h-full bg-background text-foreground">
    <!-- Contacts List -->
    <aside
      data-contacts-sidebar="true"
      role="complementary"
      :aria-label="t('chat.contacts')"
      class="relative flex min-h-0 shrink-0 flex-col bg-sidebar text-sidebar-foreground"
      :class="
        isRTL
          ? 'border-l border-sidebar-border'
          : 'border-r border-sidebar-border'
      "
      :style="{ width: `${contactsSidebarWidth}px` }"
    >
      <StatusStoriesBar :instances="instancesStore.instances" />

      <!-- Search Header -->
      <div class="border-b border-sidebar-border p-2">
        <div class="flex items-center gap-2">
          <div class="relative flex-1">
            <Search
              :class="
                'absolute top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-sidebar-foreground/50 ' +
                (isRTL ? 'right-2.5' : 'left-2.5')
              "
            />
            <Input
              v-model="contactsStore.searchQuery"
              :placeholder="$t('chat.searchContacts') + '...'"
              :aria-label="t('chat.searchContacts')"
              :class="
                'h-8 border-sidebar-border bg-sidebar-accent/65 text-sidebar-foreground placeholder:text-sidebar-foreground/50 ' +
                (isRTL ? 'pr-8 text-right' : 'pl-8 text-left')
              "
            />
          </div>
          <!-- Add Contact -->
          <Tooltip v-if="canShowAddContact">
            <TooltipTrigger as-child>
              <Button
                variant="ghost"
                size="icon"
                class="h-8 w-8 shrink-0 text-sidebar-foreground/60 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                :aria-label="t('chat.addContact')"
                @click="isAddContactOpen = true"
              >
                <UserPlus class="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{{ $t("chat.addContact") }}</TooltipContent>
          </Tooltip>
          <!-- Tag Filter -->
          <Popover v-model:open="isTagFilterOpen">
            <PopoverTrigger as-child>
              <Button
                variant="ghost"
                size="icon"
                class="h-8 w-8 shrink-0 relative"
                :aria-label="t('chat.filterByTags')"
                :class="
                  activeFilterCount > 0
                    ? 'bg-primary/12 text-primary'
                    : 'text-sidebar-foreground/60 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
                "
              >
                <Filter class="h-4 w-4" />
                <span
                  v-if="activeFilterCount > 0"
                  class="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground"
                >
                  {{ activeFilterCount }}
                </span>
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" class="w-72 p-2">
              <div class="space-y-2">
                <div class="flex items-center justify-between px-1">
                  <span class="text-sm font-medium">{{
                    $t("chat.filterByTags")
                  }}</span>
                  <Button
                    v-if="activeFilterCount > 0"
                    variant="ghost"
                    size="sm"
                    class="h-6 px-2 text-xs"
                    :aria-label="$t('chat.clearFilters')"
                    @click="clearTagFilter"
                  >
                    {{ $t("chat.clearFilters") }}
                  </Button>
                </div>
                <Separator />
                <div class="space-y-1.5 px-1">
                  <span class="text-xs font-medium text-muted-foreground">{{
                    $t("chat.filterByInstance")
                  }}</span>
                  <select
                    :value="contactsStore.selectedInstanceId"
                    :aria-label="t('chat.filterByInstance')"
                    class="h-8 w-full rounded-md border border-input bg-input px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring/30"
                    @change="updateInstanceFilter"
                  >
                    <option value="">{{ $t("chat.allInstances") }}</option>
                    <option
                      v-for="instance in instancesStore.instances"
                      :key="instance.id"
                      :value="instance.id"
                    >
                      {{ instance.name }}
                    </option>
                  </select>
                </div>
                <Separator />
                <div class="space-y-1 px-1">
                  <span class="text-xs font-medium text-muted-foreground">{{
                    $t("chat.chatType")
                  }}</span>
                  <button
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent"
                    :class="
                      contactsStore.selectedChatTypes.includes('private') &&
                      'bg-accent text-accent-foreground'
                    "
                    aria-label="Filter by: private"
                    @click="toggleChatTypeFilter('private')"
                  >
                    <span
                      :class="['flex-1', isRTL ? 'text-right' : 'text-left']"
                      >{{ $t("chat.privateChats") }}</span
                    >
                    <Check
                      v-if="contactsStore.selectedChatTypes.includes('private')"
                      class="h-4 w-4 shrink-0 text-primary"
                    />
                  </button>
                  <button
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent"
                    :class="
                      contactsStore.selectedChatTypes.includes('group') &&
                      'bg-accent text-accent-foreground'
                    "
                    aria-label="Filter by: group"
                    @click="toggleChatTypeFilter('group')"
                  >
                    <span
                      :class="['flex-1', isRTL ? 'text-right' : 'text-left']"
                      >{{ $t("chat.groupChats") }}</span
                    >
                    <Check
                      v-if="contactsStore.selectedChatTypes.includes('group')"
                      class="h-4 w-4 shrink-0 text-primary"
                    />
                  </button>
                  <button
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent"
                    :class="
                      contactsStore.selectedChatTypes.includes('channel') &&
                      'bg-accent text-accent-foreground'
                    "
                    aria-label="Filter by: channel"
                    @click="toggleChatTypeFilter('channel')"
                  >
                    <span
                      :class="['flex-1', isRTL ? 'text-right' : 'text-left']"
                      >{{ $t("chat.channelChats") }}</span
                    >
                    <Check
                      v-if="contactsStore.selectedChatTypes.includes('channel')"
                      class="h-4 w-4 shrink-0 text-primary"
                    />
                  </button>
                </div>
                <Separator />
                <div class="space-y-1">
                  <span
                    class="px-1 text-xs font-medium text-muted-foreground"
                    >{{ $t("chat.tags") }}</span
                  >
                  <div
                    v-if="tagsStore.tags.length === 0"
                    class="py-2 text-center text-sm text-muted-foreground"
                  >
                    {{ $t("chat.noTagsAvailable") }}
                  </div>
                  <div v-else class="space-y-1 max-h-48 overflow-y-auto">
                  <button
                    v-for="tag in tagsStore.tags"
                    :key="tag.name"
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-accent"
                    :class="[
                      contactsStore.selectedTags.includes(tag.name) &&
                      'bg-accent text-accent-foreground',
                    ]"
                    :aria-label="`Filter by tag: ${tag.name}`"
                    @click="toggleTagFilter(tag.name)"
                  >
                      <span
                        :class="[
                          'w-2 h-2 rounded-full shrink-0',
                          getTagColorClass(tag.color).split(' ')[0],
                        ]"
                      />
                      <span
                        :class="[
                          'flex-1 truncate',
                          isRTL ? 'text-right' : 'text-left',
                        ]"
                        >{{ tag.name }}</span
                      >
                      <Check
                        v-if="contactsStore.selectedTags.includes(tag.name)"
                        class="h-4 w-4 shrink-0 text-primary"
                      />
                    </button>
                  </div>
                </div>
              </div>
            </PopoverContent>
          </Popover>
        </div>
        <!-- Active tag filters -->
        <div v-if="activeFilterCount > 0" class="flex flex-wrap gap-1 mt-2">
          <TagBadge
            v-if="contactsStore.selectedInstanceId"
            color="blue"
            class="cursor-pointer hover:opacity-80"
            :aria-label="`${$t('chat.clearFilters')}: ${selectedInstanceName}`"
            @click="clearInstanceFilter"
          >
            {{ $t("chat.instance") }}: {{ selectedInstanceName }}
            <X class="h-3 w-3 ml-1" />
          </TagBadge>
          <TagBadge
            v-for="chatType in contactsStore.selectedChatTypes"
            :key="chatType"
            color="blue"
            class="cursor-pointer hover:opacity-80"
            :aria-label="`Remove filter: ${getChatTypeLabel(chatType)}`"
            @click="toggleChatTypeFilter(chatType)"
          >
            {{ getChatTypeLabel(chatType) }}
            <X class="h-3 w-3 ml-1" />
          </TagBadge>
          <TagBadge
            v-for="tagName in contactsStore.selectedTags"
            :key="`tag-${tagName}`"
            :color="tagsStore.getTagByName(tagName)?.color"
            class="cursor-pointer hover:opacity-80"
            :aria-label="`Remove filter: ${tagName}`"
            @click="toggleTagFilter(tagName)"
          >
            {{ tagName }}
            <X class="h-3 w-3 ml-1" />
          </TagBadge>
        </div>
        <div
          class="mt-2 grid grid-cols-2 gap-1 rounded-lg bg-sidebar-accent/75 p-1"
        >
          <button
            role="tab"
            aria-label="Assigned chats"
            :aria-selected="contactsStore.activeChatTab === 'assigned'"
            class="rounded-md px-2 py-1.5 text-xs font-medium transition-colors"
            :class="
              contactsStore.activeChatTab === 'assigned'
                ? 'bg-primary/12 text-primary'
                : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
            "
            @click="switchChatTab('assigned')"
          >
            Assigned ({{ contactsStore.assignedChats.length }})
          </button>
          <button
            role="tab"
            aria-label="Pending chats"
            :aria-selected="contactsStore.activeChatTab === 'pending'"
            class="rounded-md px-2 py-1.5 text-xs font-medium transition-colors"
            :class="
              contactsStore.activeChatTab === 'pending'
                ? 'bg-accent text-accent-foreground'
                : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
            "
            @click="switchChatTab('pending')"
          >
            Pending ({{ contactsStore.pendingChats.length }})
          </button>
        </div>
      </div>

      <!-- Contacts -->
      <ScrollArea
        :ref="(el: any) => (contactsScroll.scrollAreaRef.value = el)"
        role="list"
        :aria-label="t('chat.contactsList')"
        class="flex-1 min-h-0"
      >
        <div class="py-1">
          <div
            v-for="(entry, entryIndex) in sidebarContacts"
            :key="entry.key"
            role="listitem"
            :aria-label="`Chat with ${entry.displayContact.name || entry.displayContact.phone_number}`"
            :aria-current="isSidebarEntryActive(entry) ? 'true' : undefined"
            :class="[
              'group flex cursor-pointer items-center gap-2 px-3 py-2 transition-colors hover:bg-sidebar-accent/80',
              isSidebarEntryActive(entry) &&
                'bg-sidebar-accent text-sidebar-accent-foreground',
              focusedSidebarIndex === entryIndex &&
                !isSidebarEntryActive(entry) &&
                'bg-sidebar-accent/50',
            ]"
            data-testid="chat-sidebar-entry"
            :data-sidebar-entry-key="entry.key"
            @click="handleContactClick(entry); focusedSidebarIndex = -1"
          >
            <button
              type="button"
              class="shrink-0 rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              :aria-label="`${t('resources.ProfilePhoto')}: ${entry.displayContact.name || entry.displayContact.phone_number}`"
              @click.stop="openProfilePhotoDialog(entry.displayContact)"
            >
              <Avatar class="h-9 w-9 ring-2 ring-sidebar-border">
                <AvatarImage :src="entry.displayContact.avatar_url" />
                <AvatarFallback
                  :class="
                    'text-xs bg-gradient-to-br text-white ' +
                    getAvatarGradient(
                      entry.displayContact.name ||
                        entry.displayContact.phone_number,
                    )
                  "
                >
                  {{
                    getInitials(
                      entry.displayContact.name ||
                        entry.displayContact.phone_number,
                    )
                  }}
                </AvatarFallback>
              </Avatar>
            </button>
            <div
              :class="['flex-1 min-w-0', isRTL ? 'text-right' : 'text-left']"
            >
              <div class="flex min-w-0 items-center justify-between gap-2">
                <p class="truncate text-sm font-medium text-sidebar-foreground">
                  {{
                    entry.displayContact.name ||
                    entry.displayContact.phone_number
                  }}
                </p>
                <span
                  v-if="!isContactsSidebarCompact"
                  class="shrink-0 text-[11px] text-sidebar-foreground/55"
                >
                  {{ formatContactTime(entry.displayContact.last_message_at) }}
                </span>
              </div>
              <div class="flex items-center justify-between gap-1.5">
                <div class="min-w-0">
                  <div class="flex min-w-0 items-center gap-1.5">
                    <p
                      v-if="!isContactsSidebarCompact"
                      class="truncate text-xs text-sidebar-foreground/60"
                    >
                      {{ entry.displayContact.phone_number }}
                    </p>
                    <div
                      v-if="
                        isSidebarUnifiedMode &&
                        hasSidebarEntryMultipleInstances(entry)
                      "
                      class="flex min-w-0 flex-wrap items-center gap-1"
                      data-testid="sidebar-multi-instance-tags"
                      :data-instance-count="
                        String(getSidebarEntryInstanceCount(entry))
                      "
                    >
                      <InstanceTag
                        v-for="instanceID in resolveSidebarEntryInstanceIDs(
                          entry,
                        )"
                        :key="`sidebar-instance-tag-${entry.key}-${instanceID}`"
                        :fallback-label="
                          resolveSidebarEntryInstanceLabel(entry, instanceID)
                        "
                        :instance-id="instanceID"
                        placement="sidebar"
                      />
                    </div>
                    <InstanceTag
                      v-else-if="getSidebarEntryPrimaryInstanceID(entry)"
                      :fallback-label="
                        getSidebarEntryPrimaryInstanceLabel(entry)
                      "
                      :instance-id="getSidebarEntryPrimaryInstanceID(entry)"
                      :class="[
                        isContactsSidebarCompact ? 'max-w-[92px]' : '',
                        isContactsSidebarWide ? 'max-w-[190px]' : '',
                      ]"
                      placement="sidebar"
                    />
                  </div>
                  <div
                    v-if="
                      isSidebarUnifiedMode &&
                      entry.accountNames.length > 1 &&
                      !hasSidebarEntryMultipleInstances(entry) &&
                      !isContactsSidebarCompact
                    "
                    class="mt-1 flex flex-wrap items-center gap-1"
                  >
                    <span
                      v-for="acct in entry.accountNames"
                      :key="`sidebar-account-${entry.key}-${acct}`"
                      class="inline-flex max-w-[130px] items-center rounded-full border border-sidebar-border bg-sidebar-accent/70 px-2 py-0.5 text-[10px] leading-none text-sidebar-foreground/75"
                      :title="acct"
                    >
                      <span class="truncate">{{ acct }}</span>
                    </span>
                  </div>
                  <p
                    v-if="
                      entry.displayContact.assigned_user_id &&
                      entry.displayContact.status !== 'closed' &&
                      !isContactsSidebarCompact
                    "
                    class="mt-0.5 truncate text-[11px] text-primary"
                  >
                    {{ $t("chat.assignedTo") }}:
                    {{ getAssignedAgentName(entry.displayContact) }}
                  </p>
                </div>
                <div class="ml-2 flex items-center gap-1">
                  <Badge
                    v-if="entry.displayContact.is_public"
                    class="h-5 border-0 bg-primary/12 text-[10px] text-primary"
                  >
                    {{ $t("chat.publicShort") }}
                  </Badge>
                  <Badge
                    v-if="entry.displayContact.status === 'closed'"
                    class="h-5 border-0 bg-destructive/10 text-[10px] uppercase text-destructive"
                  >
                    {{ entry.displayContact.status }}
                  </Badge>
                  <Badge
                    v-if="entry.displayContact.unread_count > 0"
                    class="h-5 border-0 bg-primary/12 text-[10px] text-primary"
                  >
                    {{ entry.displayContact.unread_count }}
                  </Badge>
                  <button
                    v-if="canSoftDeleteChats"
                    type="button"
                    class="inline-flex h-5 w-5 items-center justify-center rounded-md border transition-colors"
                    :class="
                      pendingSidebarSoftDeleteEntryKey === entry.key
                        ? 'border-primary bg-primary text-primary-foreground'
                        : 'border-primary/20 bg-primary/10 text-primary hover:bg-primary/20 hover:text-primary'
                    "
                    :aria-label="
                      pendingSidebarSoftDeleteEntryKey === entry.key
                        ? `${$t('chat.softDeleteConfirmLabel')}: ${entry.displayContact.name || entry.displayContact.phone_number}`
                        : `${$t('chat.softDeleteChat')}: ${entry.displayContact.name || entry.displayContact.phone_number}`
                    "
                    :disabled="softDeletingSidebarEntryKey === entry.key"
                    @click.stop="softDeleteSidebarEntry(entry)"
                  >
                    <Loader2
                      v-if="softDeletingSidebarEntryKey === entry.key"
                      class="h-3.5 w-3.5 animate-spin"
                    />
                    <Check
                      v-else-if="pendingSidebarSoftDeleteEntryKey === entry.key"
                      class="h-3.5 w-3.5"
                    />
                    <Archive v-else class="h-3.5 w-3.5" />
                  </button>
                  <button
                    v-if="isAdminUser"
                    type="button"
                    class="inline-flex h-5 w-5 items-center justify-center rounded-md border transition-colors"
                    :class="
                      pendingSidebarDeleteEntryKey === entry.key
                        ? 'border-destructive bg-destructive text-destructive-foreground'
                        : 'border-destructive/20 bg-destructive/10 text-destructive hover:bg-destructive/20 hover:text-destructive'
                    "
                    :aria-label="
                      pendingSidebarDeleteEntryKey === entry.key
                        ? `Confirm delete chat: ${entry.displayContact.name || entry.displayContact.phone_number}`
                        : `Delete chat: ${entry.displayContact.name || entry.displayContact.phone_number}`
                    "
                    :disabled="deletingSidebarEntryKey === entry.key"
                    @click.stop="deleteSidebarEntry(entry)"
                  >
                    <Loader2
                      v-if="deletingSidebarEntryKey === entry.key"
                      class="h-3.5 w-3.5 animate-spin"
                    />
                    <Check
                      v-else-if="pendingSidebarDeleteEntryKey === entry.key"
                      class="h-3.5 w-3.5"
                    />
                    <Trash2 v-else class="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Loading indicator for infinite scroll -->
          <div
            v-if="contactsStore.isLoadingMoreContacts"
            class="p-3 text-center"
          >
            <Loader2
              class="mx-auto h-5 w-5 animate-spin text-muted-foreground"
            />
          </div>

          <div
            v-if="sidebarContacts.length === 0"
            class="p-3 text-center text-muted-foreground"
          >
            <User class="h-6 w-6 mx-auto mb-1.5 opacity-50" />
            <p class="text-sm">
              {{
                contactsStore.activeChatTab === "pending"
                  ? "No pending chats"
                  : "No assigned chats"
              }}
            </p>
          </div>
        </div>
      </ScrollArea>
      <div
        data-contacts-sidebar-resize-handle="true"
        class="absolute top-0 bottom-0 hidden md:block w-1 z-20 cursor-col-resize transition-colors"
        :class="[
          isRTL ? 'left-0' : 'right-0',
          isContactsSidebarResizing ? 'bg-primary/25' : 'hover:bg-primary/15',
        ]"
        @mousedown="startContactsSidebarResize"
      />
    </aside>

    <!-- Chat Area -->
    <main class="flex min-w-0 flex-1 flex-col bg-background">
      <!-- No Contact Selected -->
      <ChatEmptyState v-if="!contactsStore.currentContact" />

      <template v-else>
        <ConnectionStatusBanner />
        <!-- Chat Header -->
        <ChatHeader
          :contact="contactsStore.currentContact"
          :active-transfer-id="activeTransferId"
          :is-updating-public="isUpdatingCurrentChatPublic"
          :is-claiming="isClaimingCurrentChat"
          :is-closing="isClosingCurrentChat"
          :is-transferring="isTransferring"
          :is-resuming="isResuming"
          :is-notes-panel-open="isNotesPanelOpen"
          :is-info-panel-open="isInfoPanelOpen"
          :notes-count="notesStore.notes.length"
          :can-assign="canAssignContacts"
          :can-toggle-public="canToggleCurrentChatPublic"
          :can-claim="canClaimCurrentChat && !isCurrentChatRestricted"
          :can-close="canCloseCurrentChat"
          :can-manage-transfers="canManageTransfers"
          :custom-actions="customActions"
          :executing-action-id="executingActionId"
          executing-action-label=""
          @open-profile-photo="openProfilePhotoDialog"
          @assign="isAssignDialogOpen = true"
          @toggle-public="toggleCurrentChatPublicVisibility"
          @claim="claimCurrentChat"
          @close="closeCurrentChat"
          @transfer="transferToAgent"
          @resume="resumeChatbot"
          @execute-action="executeCustomAction"
          @toggle-notes="isNotesPanelOpen = !isNotesPanelOpen"
          @toggle-info="isInfoPanelOpen = !isInfoPanelOpen"
        />


        <!-- Account Tabs (shown when contact has messages from multiple WhatsApp accounts) -->
        <div
          v-if="isSidebarUnifiedMode && contactAccounts.length > 1"
          data-testid="chat-account-tabs"
          class="flex-shrink-0 border-b border-border bg-card/70 px-4 py-2"
        >
          <div
            role="tablist"
            class="inline-flex items-center gap-1 rounded-lg bg-accent/80 p-1"
          >
            <button
              v-for="acct in contactAccounts"
              :key="acct"
              role="tab"
              data-testid="chat-account-tab"
              :data-account-tab-key="acct"
              :data-account-tab-active="
                acct === selectedAccount ? 'true' : 'false'
              "
              :aria-selected="acct === selectedAccount"
              :aria-label="`Switch to account: ${formatAccountToggleLabel(acct)}`"
              :class="[
                'rounded-md px-3 py-1 text-xs font-medium whitespace-nowrap transition-all',
                acct === selectedAccount
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-background/90 hover:text-foreground',
              ]"
              @click="switchAccount(acct)"
            >
              {{ formatAccountToggleLabel(acct) }}
            </button>
          </div>
        </div>

        <div
          v-if="isCurrentChatRestricted"
          class="flex-1 min-h-0 flex items-center justify-center px-6"
        >
          <div class="widget-surface max-w-md w-full p-6 text-center">
            <h3 class="text-lg font-semibold text-foreground">
              Restricted View
            </h3>
            <p class="mt-2 text-sm text-muted-foreground">
              This chat is currently unassigned. You must claim it to view
              messages.
            </p>
            <Button
              v-if="canClaimCurrentChat"
              class="mt-4 w-full"
              :disabled="isClaimingCurrentChat"
              :aria-label="isClaimingCurrentChat ? 'Claiming chat...' : 'Claim chat'"
              @click="claimCurrentChat"
            >
              <Loader2
                v-if="isClaimingCurrentChat"
                class="mr-2 h-4 w-4 animate-spin"
              />
              <span>{{
                isClaimingCurrentChat ? "Claiming..." : "Claim Chat"
              }}</span>
            </Button>
          </div>
        </div>

        <!-- Messages -->
        <div v-else class="relative flex-1 min-h-0 overflow-hidden">
          <!-- Sticky date header -->
          <Transition name="sticky-date">
            <div
              v-if="showStickyDate"
              class="absolute left-1/2 top-2 z-10 -translate-x-1/2 rounded-full border border-border bg-card/90 px-3 py-1 text-[11px] font-medium text-muted-foreground shadow-sm backdrop-blur-sm"
            >
              {{ stickyDate }}
            </div>
          </Transition>

          <div
            v-if="
              contactsStore.isLoadingMessages &&
              contactsStore.messages.length === 0
            "
            class="flex h-full items-center justify-center text-muted-foreground"
          >
            <div class="flex items-center gap-2 text-sm">
              <Loader2 class="h-4 w-4 animate-spin" />
              <span>{{ $t("common.loading") }}...</span>
            </div>
          </div>
          <ChatLoadError
            v-else-if="contactsStore.messageLoadError && contactsStore.messages.length === 0"
            :is-retrying="contactsStore.isLoadingMessages"
            @retry="retryLoadMessages"
          />
          <ScrollArea
            v-else
            :ref="(el: any) => (messagesScroll.scrollAreaRef.value = el)"
            class="h-full p-3 chat-background"
            :style="chatBackgroundStyle"
            role="log"
            :aria-label="t('chat.messages')"
            aria-live="polite"
            data-testid="chat-message-area"
          >
            <div class="space-y-2">
              <!-- Loading indicator for older messages -->
              <div
                v-if="contactsStore.isLoadingOlderMessages"
                class="flex justify-center py-2"
              >
                <div
                  class="flex items-center gap-2 text-sm text-muted-foreground"
                >
                  <div
                    class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
                  />
                  <span>{{ $t("chat.loadingOlderMessages") }}...</span>
                </div>
              </div>

              <!-- Virtual scroll: top spacer for messages above viewport -->
              <div
                v-if="virtualMessages.topSpacer.value > 0"
                :style="{ height: virtualMessages.topSpacer.value + 'px' }"
                aria-hidden="true"
              />

              <template
                v-for="item in virtualMessages.virtualItems.value"
                :key="item.data.id"
              >
                <div
                  :ref="(el: any) => observeVirtualItem(el, item.data.id)"
                  :data-virtual-id="item.data.id"
                  class="virtual-message-row"
                >
                  <!-- Date separator -->
                  <div
                    v-if="shouldShowDateSeparator(item.index)"
                    class="flex items-center justify-center my-4"
                    :data-date-separator="getDateLabel(item.data.created_at)"
                  >
                    <div
                      class="rounded-full border border-border bg-card/90 px-3 py-1 text-[11px] font-medium text-muted-foreground"
                    >
                      {{ getDateLabel(item.data.created_at) }}
                    </div>
                  </div>

                  <!-- Media group start bar -->
                  <MediaGroupBar
                    v-if="isGroupLeader(item.data.id)"
                    variant="start"
                    :group="getGroupForMessage(item.data.id)!"
                    :messages="getGroupMessages(item.data.id)"
                    :blob-urls="mediaBlobUrls"
                  />

                  <!-- Message bubble -->
                  <ChatMessageBubble
                    :message="item.data"
                    :is-batch-print-selection-mode="isBatchPrintSelectionMode"
                    :is-batch-print-selectable="isBatchPrintBubbleSelectable(item.data)"
                    :is-batch-print-selected="isBatchPrintBubbleSelected(item.data.id)"
                    :is-media-group-member="isGroupMember(item.data.id)"
                    :can-revoke="canRevokeMessage(item.data)"
                    :is-revoking="revokingMessageId === item.data.id"
                    :is-retrying="retryingMessageId === item.data.id"
                    :reaction-picker-message-id="reactionPickerMessageId"
                    :quick-reaction-emojis="quickReactionEmojis"
                    :is-media-loading="isMediaLoading(item.data)"
                    :media-blob-url="getMediaBlobUrl(item.data)"
                    :is-deleted="isDeletedMessage(item.data)"
                    :is-system-event="isSystemEventMessage(item.data)"
                    :show-group-sender-phone="shouldShowGroupSenderPhone(item.data)"
                    :group-sender-phone="getGroupSenderPhone(item.data)"
                    :message-content="getMessageContent(item.data)"
                    :is-media-message="isMediaMessage(item.data)"
                    :formatted-time="formatMessageTime(item.data.created_at)"
                    :status-icon="getMessageStatusIcon(item.data.status)"
                    :status-class="getMessageStatusClass(item.data.status)"
                    :reply-author-label="getReplyAuthorLabel(item.data)"
                    :reply-content="getReplyPreviewContent(item.data)"
                    :show-reply-thumbnail="shouldShowReplyPreviewThumbnail(item.data)"
                    :reply-thumbnail-url="getReplyPreviewMediaURL(item.data)"
                    :interactive-buttons="getInteractiveButtons(item.data)"
                    :cta-url-data="getCTAUrlData(item.data)"
                    :location-data="getLocationData(item.data)"
                    :contacts-data="getContactsData(item.data)"
                    :attachment-filename="getAttachmentFilename(item.data)"
                    :show-print-button="configStore.showPrintButtons"
                    :show-download-button="configStore.showDownloadButtons"
                    :is-print-supported="!!isMessagePrintSupported(item.data)"
                    :is-reply-message="!!item.data.is_reply"
                    :reply-to-message-id="item.data.reply_to_message_id"
                    :has-media-url="!!item.data.media_url"
                    :media-type="item.data.message_type"
                    @click-bubble="handleMessageBubbleClickForBatchPrint"
                    @toggle-batch-select="toggleBatchPrintMessageSelection"
                    @click-media-preview="openMediaPreview"
                    @click-reply-preview-media="openReplyPreviewMedia"
                    @image-error="handleImageError"
                    @image-load="handleImageLoad"
                    @media-error="handleMediaError"
                    @send-reaction="sendReaction"
                    @reply="replyToMessage"
                    @revoke="revokeMessage"
                    @retry="retryMessage"
                    @scroll-to-message="scrollToMessage"
                    @download="downloadAttachment"
                    @print="printAttachment"
                    @open-reaction-picker="(id: string) => reactionPickerMessageId = id"
                    @close-reaction-picker="reactionPickerMessageId = null"
                    @reply-preview-thumb-error="() => {}"
                  />

                  <!-- Media group end bar -->
                  <MediaGroupBar
                    v-if="isGroupTail(item.data.id)"
                    variant="end"
                    :group="getGroupForMessage(item.data.id)!"
                    :messages="getGroupMessages(item.data.id)"
                    :blob-urls="mediaBlobUrls"
                  />
                </div>
              </template>

              <!-- Virtual scroll: bottom spacer for messages below viewport -->
              <div
                v-if="virtualMessages.bottomSpacer.value > 0"
                :style="{ height: virtualMessages.bottomSpacer.value + 'px' }"
                aria-hidden="true"
              />

              <div ref="messagesEndRef" />
            </div>
          </ScrollArea>

          <ChatTypingIndicator
            v-if="typingContactId && typingContactName"
            :contact-name="typingContactName"
          />
        </div>

        <!-- Chat Input Bar -->
        <ChatInputBar
          v-model:message-input="messageInput"
          :is-chat-closed="isCurrentChatClosed"
          :is-chat-restricted="isCurrentChatRestricted"
          :is-chat-send-restricted="isCurrentChatSendRestricted"
          :can-reopen="canReopenCurrentChat"
          :is-reopening="isReopeningCurrentChat"
          :is-service-window-expired="isServiceWindowExpired"
          :is-batch-print-selection-mode="isBatchPrintSelectionMode"
          :can-merge-selected-bubble-files="canMergeSelectedBubbleFiles"
          :selected-batch-print-count="selectedBatchPrintCount"
          :is-preparing-batch-print="isPreparingBatchPrint"
          :show-print-buttons="configStore.showPrintButtons"
          :replying-to="contactsStore.replyingTo"
          :reply-author-label="contactsStore.replyingTo ? getReplyingToAuthorLabel(contactsStore.replyingTo) : ''"
          :reply-content="contactsStore.replyingTo ? getMessageContent(contactsStore.replyingTo) : ''"
          :pending-canned-response="pendingCannedResponse"
          :has-pending-canned-attachments="hasPendingCannedAttachments"
          :is-current-chat-send-restricted="isCurrentChatSendRestricted"
          :can-send-message="canSendMessage"
          :is-sending="isSending"
          :is-dark="isDark"
          :current-contact="contactsStore.currentContact"
          :canned-picker-open="cannedPickerOpen"
          :canned-search-query="cannedSearchQuery"
          @send="sendMessage"
          @open-file-picker="openFilePicker"
          @handle-file-select="handleFileSelect"
          @insert-emoji="insertEmoji"
          @insert-canned-response="insertCannedResponse"
          @close-canned-picker="closeCannedPicker"
          @open-batch-print-picker="openBatchPrintPicker"
          @cancel-batch-print="cancelBatchPrintSelection"
          @clear-replying="contactsStore.clearReplyingTo"
          @clear-canned-attachments="clearPendingCannedAttachments"
          @remove-canned-attachment="removePendingCannedAttachment"
          @reopen="reopenCurrentChat"
        />
      </template>
    </main>

    <!-- Notes Side Panel -->
    <ConversationNotes
      v-if="contactsStore.currentContact && isNotesPanelOpen"
      :contact-id="contactsStore.currentContact.id"
      @close="isNotesPanelOpen = false"
    />

    <!-- Contact Info Panel -->
    <ContactInfoPanel
      v-if="contactsStore.currentContact && isInfoPanelOpen"
      :contact="contactsStore.currentContact"
      :session-data="contactSessionData"
      @close="isInfoPanelOpen = false"
      @tags-updated="
        (tags) =>
          contactsStore.updateContactTags(
            contactsStore.currentContact!.id,
            tags,
          )
      "
      @deleted="handleContactDeleted"
    />

    <!-- Assign Contact Dialog -->
    <ChatAssignDialog
      :open="isAssignDialogOpen"
      :assigned-user-id="contactsStore.currentContact?.assigned_user_id ?? null"
      :users="filteredAssignableUsers"
      @update:open="isAssignDialogOpen = $event"
      @assign="(userId) => { assignContactToUser(userId); isAssignDialogOpen = false; }"
    />

    <!-- Chat Media Viewer Dialog -->
    <ChatMediaViewerDialog
      :open="isChatMediaViewerOpen"
      :url="chatMediaViewerURL"
      :type="chatMediaViewerType"
      :title="chatMediaViewerTitle"
      @update:open="(v) => !v && closeChatMediaViewer()"
      @close="closeChatMediaViewer()"
    />

    <!-- Media Preview Dialog -->
    <ChatMediaSendDialog
      :open="isMediaDialogOpen"
      :active-media-upload="typedActiveMediaUpload"
      :selected-media-uploads="typedSelectedMediaUploads"
      :selected-media-count="selectedMediaCount"
      :is-uploading-media="isUploadingMedia"
      :media-caption="mediaCaption"
      :can-apply-media-caption="canApplyMediaCaption"
      :media-dialog-description="mediaDialogDescription"
      :media-uploading-label="mediaUploadingLabel"
      :media-send-button-label="mediaSendButtonLabel"
      @update:open="handleMediaDialogOpenChange"
      @update:media-caption="mediaCaption = $event"
      @set-active-preview="setActiveMediaPreview"
      @remove-upload="removeSelectedMediaUpload"
      @close="closeMediaDialog"
      @send="sendMediaMessage"
    />

    <!-- Contact Profile Photo Dialog -->
    <ChatProfilePhotoDialog
      :open="isProfilePhotoDialogOpen"
      :contact="profilePhotoContact"
      @update:open="handleProfilePhotoDialogOpenChange"
    />

    <!-- Add Contact Dialog -->
    <CreateContactDialog
      v-model:open="isAddContactOpen"
      mode="chat"
      @created="onContactCreated"
    />
  </div>
</template>

<style scoped>
/* Virtual scroll: skip rendering off-screen message rows */
.virtual-message-row {
  content-visibility: auto;
  contain-intrinsic-size: 0 80px;
}

/* Media group visual connector */
.media-group-member {
  border-left: 2px solid rgb(var(--primary) / 0.26);
}

:root.light .media-group-member {
  border-left-color: rgb(var(--primary) / 0.2);
}

.batch-print-selectable-bubble {
  cursor: pointer;
}

.batch-print-selectable-bubble:hover {
  box-shadow: 0 0 0 1px rgb(var(--primary) / 0.28);
}

.batch-print-selected-bubble {
  box-shadow: 0 0 0 2px rgb(var(--primary) / 0.48);
}

:root.light .batch-print-selected-bubble {
  box-shadow: 0 0 0 2px rgb(var(--primary) / 0.42);
}

.batch-print-bubble-marker {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 20px;
  height: 20px;
  border-radius: 9999px;
  border: 1px solid rgba(255, 255, 255, 0.6);
  background: rgba(15, 15, 16, 0.9);
  color: transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
  transition: all 0.2s ease;
}

.batch-print-bubble-marker--selected {
  color: rgb(236, 253, 245);
  border-color: rgb(var(--primary) / 0.95);
  background: rgb(var(--primary));
}

:root.light .batch-print-bubble-marker {
  background: rgba(255, 255, 255, 0.95);
  border-color: rgba(107, 114, 128, 0.55);
}

.sticky-date-enter-active,
.sticky-date-leave-active {
  transition: opacity 0.3s ease;
}

.sticky-date-enter-from,
.sticky-date-leave-to {
  opacity: 0;
}
</style>
