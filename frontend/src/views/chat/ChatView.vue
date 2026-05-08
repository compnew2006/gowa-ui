<script setup lang="ts">
import {
  ref,
  watch,
  onMounted,
  onUnmounted,
  nextTick,
  computed,
  defineAsyncComponent,
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
const EmojiPicker = defineAsyncComponent(() => {
  return import("vue3-emoji-picker").then((module) => {
    import("vue3-emoji-picker/css");
    return module.default;
  });
});
import { toast } from "vue-sonner";
import {
  Search,
  Send,
  Paperclip,
  FileText,
  Download,
  Printer,
  Smile,
  Phone,
  Check,
  CheckCheck,
  Clock,
  AlertCircle,
  User,
  UserPlus,
  UserX,
  Play,
  Reply,
  X,
  SmilePlus,
  Archive,
  Trash2,
  MapPin,
  ExternalLink,
  Loader2,
  Pin,
  RotateCw,
  Filter,
  StickyNote,
} from "lucide-vue-next";
import { getInitials, getAvatarGradient } from "@/lib/utils";
import { isGroupContact } from "@/lib/group-chat";
import { resolveMediaFilename } from "@/lib/media-actions";
import { resolvePreferredOutboundInstanceID } from "@/lib/chat-outbound-instance";
import type { SidebarContactEntry } from "@/lib/chat-sidebar-unifier";
import { MessageHistoryNavigator } from "@/lib/message-history-navigator";
import { useColorMode } from "@/composables/useColorMode";
import { useInfiniteScroll } from "@/composables/useInfiniteScroll";
import { useMessageContent } from "@/composables/useMessageContent";
import { useTypingPresence } from "@/composables/useTypingPresence";
import { useChatSidebar } from "@/composables/useChatSidebar";
import { useChatMedia } from "@/composables/useChatMedia";
import { useBatchPrint } from "@/composables/useBatchPrint";
import { useChatActions } from "@/composables/useChatActions";
import { useChatMessaging } from "@/composables/useChatMessaging";
import CannedResponsePicker from "@/components/chat/CannedResponsePicker.vue";
import ContactInfoPanel from "@/components/chat/ContactInfoPanel.vue";
import ConversationNotes from "@/components/chat/ConversationNotes.vue";
import InstanceTag from "@/components/chat/InstanceTag.vue";
import LinkifiedMessageText from "@/components/chat/LinkifiedMessageText.vue";
import MediaGroupBar from "@/components/chat/MediaGroupBar.vue";
import StatusStoriesBar from "@/components/chat/status/StatusStoriesBar.vue";
import ChatEmptyState from "@/components/chat/ChatEmptyState.vue";
import ChatAssignDialog from "@/components/chat/ChatAssignDialog.vue";
import ChatMediaViewerDialog from "@/components/chat/ChatMediaViewerDialog.vue";
import ChatMediaSendDialog from "@/components/chat/ChatMediaSendDialog.vue";
import type { PendingMediaUpload } from "@/components/chat/ChatMediaSendDialog.vue";
import ChatProfilePhotoDialog from "@/components/chat/ChatProfilePhotoDialog.vue";
import { useInstancesStore } from "@/stores/instances";
import { useNotesStore } from "@/stores/notes";
import { CreateContactDialog } from "@/components/shared";
import { Info } from "lucide-vue-next";
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
  getGoogleMapsUrl,
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
  handleReplyPreviewThumbnailError,
  preloadMentionResolverFromKnownContacts,
  resolveMentionsForCurrentMessages,
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
  fileInputRef,
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
  getPendingAttachmentIcon,
  sendReaction,
  openFilePicker,
  handleFileSelect,
  closeMediaDialog,
  handleMediaDialogOpenChange,
  sendMediaMessage,
  setActiveMediaPreview,
  removeSelectedMediaUpload,
} = chatMessaging;

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
  getActionIcon,
  fetchCustomActions,
  executeCustomAction,
  transferToAgent,
  assignContactToUser,
  claimCurrentChat,
  closeCurrentChat,
  reopenCurrentChat,
} = chatActions;

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

function autoResizeTextarea() {
  const textarea = messageInputRef.value;
  if (!textarea) return;
  textarea.style.height = "auto";
  textarea.style.height = Math.min(textarea.scrollHeight, 120) + "px";
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
    <div
      data-contacts-sidebar="true"
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
                @click="openAddContactDialog"
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
                      :class="
                        contactsStore.selectedTags.includes(tag.name) &&
                        'bg-accent text-accent-foreground'
                      "
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
        class="flex-1 min-h-0"
      >
        <div class="py-1">
          <div
            v-for="entry in sidebarContacts"
            :key="entry.key"
            :class="[
              'group flex cursor-pointer items-center gap-2 px-3 py-2 transition-colors hover:bg-sidebar-accent/80',
              isSidebarEntryActive(entry) &&
                'bg-sidebar-accent text-sidebar-accent-foreground',
            ]"
            data-testid="chat-sidebar-entry"
            :data-sidebar-entry-key="entry.key"
            @click="handleContactClick(entry)"
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
    </div>

    <!-- Chat Area -->
    <div class="flex min-w-0 flex-1 flex-col bg-background">
      <!-- No Contact Selected -->
      <ChatEmptyState v-if="!contactsStore.currentContact" />

      <!-- Chat Interface -->
      <template v-else>
        <!-- Chat Header -->
        <div
          class="flex h-14 flex-shrink-0 items-center justify-between border-b border-border bg-card/95 px-4 backdrop-blur"
        >
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              :aria-label="`${t('resources.ProfilePhoto')}: ${contactsStore.currentContact.name || contactsStore.currentContact.phone_number}`"
              @click="openProfilePhotoDialog(contactsStore.currentContact)"
            >
              <Avatar class="h-8 w-8 ring-2 ring-border">
                <AvatarImage :src="contactsStore.currentContact.avatar_url" />
                <AvatarFallback
                  :class="
                    'text-xs bg-gradient-to-br text-white ' +
                    getAvatarGradient(
                      contactsStore.currentContact.name ||
                        contactsStore.currentContact.phone_number,
                    )
                  "
                >
                  {{
                    getInitials(
                      contactsStore.currentContact.name ||
                        contactsStore.currentContact.phone_number,
                    )
                  }}
                </AvatarFallback>
              </Avatar>
            </button>
            <div>
              <div class="flex items-center gap-1.5">
                <p class="text-sm font-medium text-foreground">
                  {{
                    contactsStore.currentContact.name ||
                    contactsStore.currentContact.phone_number
                  }}
                </p>
                <Badge
                  v-if="contactsStore.currentContact.is_public"
                  class="h-5 border-0 bg-primary/12 text-[10px] text-primary"
                >
                  {{ $t("chat.publicChat") }}
                </Badge>
                <Badge
                  v-if="contactsStore.currentContact.status === 'pending'"
                  class="h-5 border-0 bg-accent text-[10px] text-accent-foreground"
                >
                  Pending
                </Badge>
                <Badge
                  v-if="contactsStore.currentContact.status === 'closed'"
                  class="h-5 border-0 bg-muted text-[10px] text-muted-foreground"
                >
                  Closed
                </Badge>
                <Badge
                  v-if="activeTransferId"
                  class="h-5 border-0 bg-accent text-[10px] text-primary"
                >
                  Paused
                </Badge>
              </div>
              <p class="text-[11px] text-muted-foreground">
                {{ contactsStore.currentContact.phone_number }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-1">
            <Tooltip v-if="canAssignContacts">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  @click="isAssignDialogOpen = true"
                >
                  <UserPlus class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.assignToAgent") }}</TooltipContent>
            </Tooltip>
            <Tooltip v-if="canToggleCurrentChatPublic">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :disabled="isUpdatingCurrentChatPublic"
                  @click="toggleCurrentChatPublicVisibility"
                >
                  <Loader2
                    v-if="isUpdatingCurrentChatPublic"
                    class="h-4 w-4 animate-spin"
                  />
                  <Pin
                    v-else
                    class="h-4 w-4"
                    :class="
                      contactsStore.currentContact?.is_public
                        ? 'text-primary'
                        : ''
                    "
                  />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {{
                  contactsStore.currentContact?.is_public
                    ? $t("chat.removePublicChat")
                    : $t("chat.makePublicChat")
                }}
              </TooltipContent>
            </Tooltip>
            <Tooltip v-if="canClaimCurrentChat && !isCurrentChatRestricted">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :disabled="isClaimingCurrentChat"
                  @click="claimCurrentChat"
                >
                  <Loader2
                    v-if="isClaimingCurrentChat"
                    class="h-4 w-4 animate-spin"
                  />
                  <Check v-else class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.claimChat") }}</TooltipContent>
            </Tooltip>
            <Tooltip v-if="canCloseCurrentChat">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :disabled="isClosingCurrentChat"
                  @click="closeCurrentChat"
                >
                  <Loader2
                    v-if="isClosingCurrentChat"
                    class="h-4 w-4 animate-spin"
                  />
                  <Check v-else class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Close Chat</TooltipContent>
            </Tooltip>
            <Tooltip v-if="canManageTransfers && !activeTransferId">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :disabled="isTransferring"
                  @click="transferToAgent"
                >
                  <UserX class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.transferToAgent") }}</TooltipContent>
            </Tooltip>
            <Tooltip v-if="canManageTransfers && activeTransferId">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :disabled="isResuming"
                  @click="resumeChatbot"
                >
                  <Play class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.resumeChatbot") }}</TooltipContent>
            </Tooltip>
            <!-- Custom Action Buttons -->
            <Tooltip v-for="action in customActions" :key="action.id">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :disabled="executingActionId === action.id"
                  @click="executeCustomAction(action)"
                >
                  <Loader2
                    v-if="executingActionId === action.id"
                    class="h-4 w-4 animate-spin"
                  />
                  <component
                    v-else
                    :is="getActionIcon(action.icon)"
                    class="h-4 w-4"
                  />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ action.name }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  id="notes-button"
                  class="relative h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :class="
                    isNotesPanelOpen && 'bg-accent text-accent-foreground'
                  "
                  @click="isNotesPanelOpen = !isNotesPanelOpen"
                >
                  <StickyNote class="h-4 w-4" />
                  <span
                    v-if="notesStore.notes.length > 0 && !isNotesPanelOpen"
                    id="notes-badge"
                    class="absolute -top-0.5 -right-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-primary px-1 text-[10px] text-primary-foreground"
                  >
                    {{ notesStore.notes.length }}
                  </span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.internalNotes") }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  id="info-button"
                  class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
                  :class="isInfoPanelOpen && 'bg-accent text-accent-foreground'"
                  @click="isInfoPanelOpen = !isInfoPanelOpen"
                >
                  <Info class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.contactInfo") }}</TooltipContent>
            </Tooltip>
          </div>
        </div>

        <!-- Account Tabs (shown when contact has messages from multiple WhatsApp accounts) -->
        <div
          v-if="isSidebarUnifiedMode && contactAccounts.length > 1"
          data-testid="chat-account-tabs"
          class="flex-shrink-0 border-b border-border bg-card/70 px-4 py-2"
        >
          <div
            class="inline-flex items-center gap-1 rounded-lg bg-accent/80 p-1"
          >
            <button
              v-for="acct in contactAccounts"
              :key="acct"
              data-testid="chat-account-tab"
              :data-account-tab-key="acct"
              :data-account-tab-active="
                acct === selectedAccount ? 'true' : 'false'
              "
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
          <ScrollArea
            v-else
            :ref="(el: any) => (messagesScroll.scrollAreaRef.value = el)"
            class="h-full p-3 chat-background"
            :style="chatBackgroundStyle"
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
              <template
                v-for="(message, index) in contactsStore.messages"
                :key="message.id"
              >
                <!-- Date separator -->
                <div
                  v-if="shouldShowDateSeparator(index)"
                  class="flex items-center justify-center my-4"
                  :data-date-separator="getDateLabel(message.created_at)"
                >
                  <div
                    class="rounded-full border border-border bg-card/90 px-3 py-1 text-[11px] font-medium text-muted-foreground"
                  >
                    {{ getDateLabel(message.created_at) }}
                  </div>
                </div>

                <!-- Media group start bar -->
                <MediaGroupBar
                  v-if="isGroupLeader(message.id)"
                  variant="start"
                  :group="getGroupForMessage(message.id)!"
                  :messages="getGroupMessages(message.id)"
                  :blob-urls="mediaBlobUrls"
                />

                <!-- Message bubble -->
                <div
                  :id="`message-${message.id}`"
                  :class="[
                    'flex group',
                    isSystemEventMessage(message)
                      ? 'justify-center'
                      : message.direction === 'outgoing'
                        ? 'justify-end'
                        : 'justify-start',
                  ]"
                >
                  <div
                    :class="[
                      'chat-bubble relative',
                      isSystemEventMessage(message)
                        ? 'chat-bubble-system'
                        : message.direction === 'outgoing'
                          ? 'chat-bubble-outgoing'
                          : 'chat-bubble-incoming',
                      isDeletedMessage(message) ? 'chat-bubble-deleted' : '',
                      isGroupMember(message.id) ? 'media-group-member' : '',
                      isBatchPrintSelectionMode &&
                      isBatchPrintBubbleSelectable(message)
                        ? 'batch-print-selectable-bubble'
                        : '',
                      isBatchPrintBubbleSelected(message.id)
                        ? 'batch-print-selected-bubble'
                        : '',
                    ]"
                    @click="
                      handleMessageBubbleClickForBatchPrint(message, $event)
                    "
                  >
                    <button
                      v-if="
                        isBatchPrintSelectionMode &&
                        isBatchPrintBubbleSelectable(message)
                      "
                      type="button"
                      class="batch-print-bubble-marker"
                      :class="
                        isBatchPrintBubbleSelected(message.id)
                          ? 'batch-print-bubble-marker--selected'
                          : ''
                      "
                      @click.stop.prevent="
                        toggleBatchPrintMessageSelection(message.id)
                      "
                    >
                      <Check
                        v-if="isBatchPrintBubbleSelected(message.id)"
                        class="h-3 w-3"
                      />
                    </button>
                    <p
                      v-if="shouldShowGroupSenderPhone(message)"
                      class="mb-1 text-[11px] font-medium text-primary"
                    >
                      {{ getGroupSenderPhone(message) }}
                    </p>
                    <!-- Reply preview (if this message is replying to another) -->
                    <div
                      v-if="message.is_reply && message.reply_to_message"
                      class="reply-preview cursor-pointer text-xs"
                      @click="scrollToMessage(message.reply_to_message_id)"
                    >
                      <p class="font-medium">
                        {{ getReplyAuthorLabel(message) }}
                      </p>
                      <div class="reply-preview-content">
                        <img
                          v-if="shouldShowReplyPreviewThumbnail(message)"
                          :src="getReplyPreviewMediaURL(message)"
                          alt="Reply image preview"
                          class="reply-preview-thumb"
                          @click.stop="openReplyPreviewMedia(message, $event)"
                          @error="handleReplyPreviewThumbnailError"
                        />
                        <p class="truncate">
                          {{ getReplyPreviewContent(message) }}
                        </p>
                      </div>
                    </div>
                    <!-- Image message -->
                    <div
                      v-if="
                        message.message_type === 'image' && message.media_url
                      "
                      class="mb-2"
                    >
                      <div
                        v-if="isMediaLoading(message)"
                        class="w-[200px] h-[150px] bg-muted rounded-lg animate-pulse flex items-center justify-center"
                      >
                        <span class="text-muted-foreground text-sm"
                          >{{ $t("common.loading") }}...</span
                        >
                      </div>
                      <img
                        v-else-if="getMediaBlobUrl(message)"
                        :src="getMediaBlobUrl(message)"
                        :alt="message.content?.body || 'Image'"
                        class="max-w-[280px] max-h-[300px] rounded-lg cursor-pointer object-cover"
                        @click="openMediaPreview(message, $event)"
                        @error="handleImageError($event)"
                        @load="handleImageLoad"
                      />
                      <div
                        v-else
                        class="w-[200px] h-[150px] bg-muted rounded-lg flex items-center justify-center"
                      >
                        <span class="text-muted-foreground text-sm"
                          >[Image]</span
                        >
                      </div>
                      <div
                        v-if="
                          getMediaBlobUrl(message) &&
                          (configStore.showPrintButtons ||
                            configStore.showDownloadButtons)
                        "
                        class="mt-2 flex flex-wrap items-center gap-1.5"
                      >
                        <Button
                          v-if="configStore.showPrintButtons"
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="printAttachment(message, $event)"
                        >
                          <Printer class="h-3.5 w-3.5" />
                          {{ $t("common.print") }}
                        </Button>
                        <Button
                          v-if="configStore.showDownloadButtons"
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="downloadAttachment(message, $event)"
                        >
                          <Download class="h-3.5 w-3.5" />
                          {{ $t("common.download") }}
                        </Button>
                      </div>
                    </div>
                    <!-- Sticker message -->
                    <div
                      v-else-if="
                        message.message_type === 'sticker' && message.media_url
                      "
                      class="mb-2"
                    >
                      <div
                        v-if="isMediaLoading(message)"
                        class="w-[128px] h-[128px] bg-muted rounded-lg animate-pulse flex items-center justify-center"
                      >
                        <span class="text-muted-foreground text-sm"
                          >{{ $t("common.loading") }}...</span
                        >
                      </div>
                      <img
                        v-else-if="getMediaBlobUrl(message)"
                        :src="getMediaBlobUrl(message)"
                        alt="Sticker"
                        class="max-w-[128px] max-h-[128px] cursor-pointer"
                        @click="openMediaPreview(message, $event)"
                        @error="handleImageError($event)"
                        @load="handleImageLoad"
                      />
                      <div
                        v-else
                        class="w-[128px] h-[128px] bg-muted rounded-lg flex items-center justify-center"
                      >
                        <span class="text-muted-foreground text-sm"
                          >[Sticker]</span
                        >
                      </div>
                    </div>
                    <!-- Video message -->
                    <div
                      v-else-if="
                        message.message_type === 'video' && message.media_url
                      "
                      class="mb-2"
                    >
                      <div
                        v-if="isMediaLoading(message)"
                        class="w-[200px] h-[150px] bg-muted rounded-lg animate-pulse flex items-center justify-center"
                      >
                        <span class="text-muted-foreground text-sm"
                          >{{ $t("common.loading") }}...</span
                        >
                      </div>
                      <video
                        v-else-if="getMediaBlobUrl(message)"
                        :src="getMediaBlobUrl(message)"
                        controls
                        class="max-w-[280px] max-h-[300px] rounded-lg"
                        @error="handleMediaError($event, 'video')"
                      />
                      <div
                        v-else
                        class="w-[200px] h-[150px] bg-muted rounded-lg flex items-center justify-center"
                      >
                        <span class="text-muted-foreground text-sm"
                          >[Video]</span
                        >
                      </div>
                    </div>
                    <!-- Audio message -->
                    <div
                      v-else-if="
                        message.message_type === 'audio' && message.media_url
                      "
                      class="mb-2"
                    >
                      <div
                        v-if="isMediaLoading(message)"
                        class="w-[200px] h-[40px] bg-muted rounded-lg animate-pulse"
                      ></div>
                      <audio
                        v-else-if="getMediaBlobUrl(message)"
                        :src="getMediaBlobUrl(message)"
                        controls
                        class="max-w-[280px]"
                        @error="handleMediaError($event, 'audio')"
                      />
                      <div v-else class="text-muted-foreground text-sm">
                        [Audio]
                      </div>
                    </div>
                    <!-- Document message -->
                    <div
                      v-else-if="
                        message.message_type === 'document' && message.media_url
                      "
                      class="mb-2"
                    >
                      <div v-if="getMediaBlobUrl(message)" class="space-y-2">
                        <a
                          :href="getMediaBlobUrl(message)"
                          :download="getAttachmentFilename(message)"
                          class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
                        >
                          <FileText class="h-5 w-5 text-muted-foreground" />
                          <span class="text-sm truncate max-w-[200px]">
                            {{ getAttachmentFilename(message) }}
                          </span>
                        </a>
                        <div
                          v-if="
                            configStore.showPrintButtons ||
                            configStore.showDownloadButtons
                          "
                          class="flex flex-wrap items-center gap-1.5"
                        >
                          <Button
                            v-if="
                              configStore.showPrintButtons &&
                              isMessagePrintSupported(message)
                            "
                            variant="ghost"
                            size="xs"
                            class="h-7 px-2 text-[11px]"
                            @click.stop="printAttachment(message, $event)"
                          >
                            <Printer class="h-3.5 w-3.5" />
                            {{ $t("common.print") }}
                          </Button>
                          <Button
                            v-if="configStore.showDownloadButtons"
                            variant="ghost"
                            size="xs"
                            class="h-7 px-2 text-[11px]"
                            @click.stop="downloadAttachment(message, $event)"
                          >
                            <Download class="h-3.5 w-3.5" />
                            {{ $t("common.download") }}
                          </Button>
                        </div>
                      </div>
                      <div
                        v-else-if="isMediaLoading(message)"
                        class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg"
                      >
                        <FileText class="h-5 w-5 text-muted-foreground" />
                        <span class="text-sm text-muted-foreground"
                          >{{ $t("common.loading") }}...</span
                        >
                      </div>
                      <div
                        v-else
                        class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg"
                      >
                        <FileText class="h-5 w-5 text-muted-foreground" />
                        <span class="text-sm text-muted-foreground"
                          >[Document]</span
                        >
                      </div>
                    </div>
                    <!-- Location message -->
                    <div
                      v-else-if="
                        message.message_type === 'location' &&
                        getLocationData(message)
                      "
                      class="mb-2"
                    >
                      <a
                        :href="getGoogleMapsUrl(getLocationData(message)!)"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="flex items-center gap-3 px-3 py-3 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
                      >
                        <div
                          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-destructive/10"
                        >
                          <MapPin class="h-5 w-5 text-red-500" />
                        </div>
                        <div class="flex-1 min-w-0">
                          <p
                            v-if="getLocationData(message)?.name"
                            class="text-sm font-medium truncate"
                          >
                            {{ getLocationData(message)?.name }}
                          </p>
                          <p v-else class="text-sm font-medium">Location</p>
                          <p
                            v-if="getLocationData(message)?.address"
                            class="text-xs text-muted-foreground truncate"
                          >
                            {{ getLocationData(message)?.address }}
                          </p>
                          <p class="text-xs text-muted-foreground">
                            {{ getLocationData(message)?.latitude.toFixed(6) }},
                            {{ getLocationData(message)?.longitude.toFixed(6) }}
                          </p>
                        </div>
                        <ExternalLink
                          class="h-4 w-4 text-muted-foreground shrink-0"
                        />
                      </a>
                    </div>
                    <!-- Contacts message -->
                    <div
                      v-else-if="
                        (message.message_type === 'contacts' ||
                          message.message_type === 'contact') &&
                        getContactsData(message).length > 0
                      "
                      class="mb-2 space-y-2"
                    >
                      <div
                        v-for="(contact, idx) in getContactsData(message)"
                        :key="idx"
                        class="flex items-center gap-3 px-3 py-2 bg-background/50 rounded-lg"
                      >
                        <div
                          class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center shrink-0"
                        >
                          <User class="h-5 w-5 text-primary" />
                        </div>
                        <div class="flex-1 min-w-0">
                          <p class="text-sm font-medium truncate">
                            {{ contact.name }}
                          </p>
                          <div
                            v-if="contact.phones?.length"
                            class="flex items-center gap-1 text-xs text-muted-foreground"
                          >
                            <Phone class="h-3 w-3" />
                            <span class="truncate">{{
                              contact.phones.join(", ")
                            }}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                    <!-- Unsupported message -->
                    <div
                      v-else-if="message.message_type === 'unsupported'"
                      class="mb-2"
                    >
                      <div
                        class="flex items-center gap-2 px-3 py-2 bg-muted/50 rounded-lg text-muted-foreground"
                      >
                        <AlertCircle class="h-4 w-4 shrink-0" />
                        <span class="text-sm italic"
                          >This message type is not supported</span
                        >
                      </div>
                    </div>
                    <!-- Button reply - WhatsApp style -->
                    <div
                      v-if="message.message_type === 'button_reply'"
                      class="button-reply-bubble"
                    >
                      <span class="whitespace-pre-wrap break-words"
                        ><LinkifiedMessageText
                          :text="getMessageContent(message)"
                      /></span>
                      <span class="chat-bubble-time"
                        ><span>{{
                          formatMessageTime(message.created_at)
                        }}</span></span
                      >
                    </div>
                    <!-- Text content (for text messages or captions) -->
                    <span
                      v-else-if="getMessageContent(message)"
                      class="whitespace-pre-wrap break-words"
                      ><LinkifiedMessageText
                        :text="getMessageContent(message)" />
                      <span class="chat-bubble-time"
                        ><span>{{ formatMessageTime(message.created_at) }}</span
                        ><component
                          v-if="
                            message.direction === 'outgoing' &&
                            !isSystemEventMessage(message)
                          "
                          :is="getMessageStatusIcon(message.status)"
                          :class="[
                            'h-4 w-4 status-icon',
                            getMessageStatusClass(message.status),
                          ]" /></span
                    ></span>
                    <!-- Fallback for media without URL -->
                    <span
                      v-else-if="isMediaMessage(message) && !message.media_url"
                      class="text-muted-foreground italic"
                      >[{{
                        message.message_type.charAt(0).toUpperCase() +
                        message.message_type.slice(1)
                      }}]<span class="chat-bubble-time"
                        ><span>{{ formatMessageTime(message.created_at) }}</span
                        ><component
                          v-if="
                            message.direction === 'outgoing' &&
                            !isSystemEventMessage(message)
                          "
                          :is="getMessageStatusIcon(message.status)"
                          :class="[
                            'h-4 w-4 status-icon',
                            getMessageStatusClass(message.status),
                          ]" /></span
                    ></span>
                    <!-- Interactive buttons - WhatsApp style -->
                    <div
                      v-if="getInteractiveButtons(message).length > 0"
                      class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t"
                    >
                      <div
                        v-for="(btn, index) in getInteractiveButtons(message)"
                        :key="btn.id"
                        :class="[
                          'py-2 text-sm text-center font-medium cursor-pointer',
                          index > 0 && 'border-t',
                        ]"
                      >
                        {{ btn.title }}
                      </div>
                    </div>
                    <!-- CTA URL button - WhatsApp style -->
                    <a
                      v-if="getCTAUrlData(message)"
                      :href="getCTAUrlData(message)?.url"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t block"
                    >
                      <div
                        class="py-2 text-sm text-center font-medium cursor-pointer flex items-center justify-center gap-1.5"
                      >
                        <ExternalLink class="h-3.5 w-3.5" />
                        {{ getCTAUrlData(message)?.button_text }}
                      </div>
                    </a>
                    <!-- Time for messages without text content -->
                    <span
                      v-if="
                        !getMessageContent(message) &&
                        !(isMediaMessage(message) && !message.media_url)
                      "
                      class="chat-bubble-time block clear-both"
                    >
                      <span>{{ formatMessageTime(message.created_at) }}</span>
                      <component
                        v-if="
                          message.direction === 'outgoing' &&
                          !isSystemEventMessage(message)
                        "
                        :is="getMessageStatusIcon(message.status)"
                        :class="[
                          'h-4 w-4 status-icon',
                          getMessageStatusClass(message.status),
                        ]"
                      />
                    </span>
                    <!-- Reactions display -->
                    <div
                      v-if="message.reactions && message.reactions.length > 0"
                      class="reactions-display flex flex-wrap gap-1 mt-1"
                    >
                      <span
                        v-for="(reaction, idx) in message.reactions"
                        :key="idx"
                        class="reaction-badge"
                        :title="reaction.from_phone || reaction.from_user || ''"
                      >
                        {{ reaction.emoji }}
                      </span>
                    </div>
                    <!-- Failed message error (not for template messages) -->
                    <span
                      v-if="
                        message.status === 'failed' &&
                        message.direction === 'outgoing' &&
                        message.message_type !== 'template'
                      "
                      class="flex items-center gap-1 mt-1 text-xs text-destructive"
                    >
                      <AlertCircle class="h-3 w-3" />
                      <span>{{
                        message.error_message || "Failed to send"
                      }}</span>
                    </span>
                    <!-- Failed template message indicator (no retry) -->
                    <span
                      v-if="
                        message.status === 'failed' &&
                        message.direction === 'outgoing' &&
                        message.message_type === 'template'
                      "
                      class="flex items-center gap-1 mt-1 text-xs text-destructive"
                    >
                      <AlertCircle class="h-3 w-3" />
                      <span>{{
                        message.error_message || "Failed to send"
                      }}</span>
                    </span>
                  </div>
                  <!-- Action buttons for incoming messages -->
                  <div
                    v-if="
                      message.direction === 'incoming' &&
                      !isSystemEventMessage(message)
                    "
                    class="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity self-center ml-1"
                  >
                    <Popover
                      :open="reactionPickerMessageId === message.id"
                      @update:open="
                        (open: boolean) =>
                          (reactionPickerMessageId = open ? message.id : null)
                      "
                    >
                      <PopoverTrigger as-child>
                        <Button variant="ghost" size="icon" class="h-6 w-6">
                          <SmilePlus class="h-3 w-3" />
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent side="top" class="w-auto p-2">
                        <div class="flex gap-1">
                          <button
                            v-for="emoji in quickReactionEmojis"
                            :key="emoji"
                            class="text-lg hover:bg-muted p-1 rounded cursor-pointer"
                            @click="sendReaction(message.id, emoji)"
                          >
                            {{ emoji }}
                          </button>
                        </div>
                      </PopoverContent>
                    </Popover>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-6 w-6"
                      @click="replyToMessage(message)"
                    >
                      <Reply class="h-3 w-3" />
                    </Button>
                  </div>
                  <!-- Reply button for outgoing messages (shown on hover) -->
                  <div
                    v-if="
                      message.direction === 'outgoing' &&
                      !isSystemEventMessage(message)
                    "
                    class="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity self-center ml-1"
                  >
                    <Popover
                      :open="reactionPickerMessageId === message.id"
                      @update:open="
                        (open: boolean) =>
                          (reactionPickerMessageId = open ? message.id : null)
                      "
                    >
                      <PopoverTrigger as-child>
                        <Button variant="ghost" size="icon" class="h-6 w-6">
                          <SmilePlus class="h-3 w-3" />
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent side="top" class="w-auto p-2">
                        <div class="flex gap-1">
                          <button
                            v-for="emoji in quickReactionEmojis"
                            :key="emoji"
                            class="text-lg hover:bg-muted p-1 rounded cursor-pointer"
                            @click="sendReaction(message.id, emoji)"
                          >
                            {{ emoji }}
                          </button>
                        </div>
                      </PopoverContent>
                    </Popover>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-6 w-6"
                      @click="replyToMessage(message)"
                    >
                      <Reply class="h-3 w-3" />
                    </Button>
                    <Button
                      v-if="canRevokeMessage(message)"
                      variant="ghost"
                      size="icon"
                      class="h-6 w-6 text-destructive/80 hover:bg-destructive/10 hover:text-destructive"
                      :disabled="revokingMessageId === message.id"
                      title="Revoke message"
                      @click="revokeMessage(message)"
                    >
                      <Loader2
                        v-if="revokingMessageId === message.id"
                        class="h-3 w-3 animate-spin"
                      />
                      <Trash2 v-else class="h-3 w-3" />
                    </Button>
                    <Button
                      v-if="
                        message.status === 'failed' &&
                        message.message_type !== 'template'
                      "
                      variant="ghost"
                      size="icon"
                      class="h-6 w-6 text-destructive/80 hover:bg-destructive/10 hover:text-destructive"
                      :disabled="retryingMessageId === message.id"
                      @click="retryMessage(message)"
                      title="Retry sending"
                    >
                      <Loader2
                        v-if="retryingMessageId === message.id"
                        class="h-3 w-3 animate-spin"
                      />
                      <RotateCw v-else class="h-3 w-3" />
                    </Button>
                  </div>
                </div>

                <!-- Media group end bar -->
                <MediaGroupBar
                  v-if="isGroupTail(message.id)"
                  variant="end"
                  :group="getGroupForMessage(message.id)!"
                  :messages="getGroupMessages(message.id)"
                  :blob-urls="mediaBlobUrls"
                />
              </template>
              <div ref="messagesEndRef" />
            </div>
          </ScrollArea>
        </div>

        <div
          v-if="isCurrentChatClosed && !isCurrentChatRestricted"
          class="flex items-center justify-between gap-3 border-t border-border bg-muted/55 px-4 py-2 text-xs text-muted-foreground"
        >
          <span
            >This chat is closed. You can view message history in read-only
            mode.</span
          >
          <Button
            v-if="canReopenCurrentChat"
            size="sm"
            variant="outline"
            class="h-7 px-2.5 text-xs"
            :disabled="isReopeningCurrentChat"
            @click="reopenCurrentChat"
          >
            <Loader2
              v-if="isReopeningCurrentChat"
              class="mr-1.5 h-3 w-3 animate-spin"
            />
            <RotateCw v-else class="mr-1.5 h-3 w-3" />
            Reopen Chat
          </Button>
        </div>

        <!-- Service window expired banner -->
        <div
          v-if="isServiceWindowExpired"
          class="flex items-center gap-2 border-t border-destructive/20 bg-destructive/10 px-4 py-2.5"
        >
          <Clock class="h-4 w-4 text-red-500 shrink-0" />
          <span class="text-sm text-red-500 flex-1">{{
            $t("chat.serviceWindowExpired")
          }}</span>
        </div>

        <!-- Reply indicator -->
        <div
          v-if="
            contactsStore.replyingTo &&
            !isCurrentChatClosed &&
            !isCurrentChatRestricted &&
            !isCurrentChatSendRestricted
          "
          class="flex items-center justify-between border-t border-border bg-card/80 px-4 py-2"
        >
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-muted-foreground">
              Replying to
              {{ getReplyingToAuthorLabel(contactsStore.replyingTo) }}
            </p>
            <p class="truncate text-sm text-foreground/80">
              {{ getMessageContent(contactsStore.replyingTo) || "[Media]" }}
            </p>
          </div>
          <button
            class="flex h-6 w-6 shrink-0 items-center justify-center rounded transition-colors hover:bg-accent"
            @click="contactsStore.clearReplyingTo"
          >
            <X class="h-4 w-4 text-muted-foreground" />
          </button>
        </div>

        <!-- Message Input -->
        <div
          v-if="!isCurrentChatClosed && !isCurrentChatRestricted"
          class="border-t border-border bg-card/95 p-4"
        >
          <div
            v-if="isCurrentChatSendRestricted"
            class="mb-2 rounded-lg border border-accent px-3 py-2 text-xs text-muted-foreground"
          >
            This chat can be viewed without claim, but sending is blocked until
            you claim it.
          </div>

          <div
            v-if="pendingCannedResponse?.attachments?.length"
            class="mb-2 rounded-lg border border-primary/20 bg-primary/8 p-2"
          >
            <div class="mb-1 flex items-center justify-between">
              <p class="text-xs text-primary">
                {{ pendingCannedResponse.attachments.length }} canned media
                attachment(s) ready
              </p>
              <button
                type="button"
                class="text-xs text-primary hover:text-foreground"
                @click="clearPendingCannedAttachments"
              >
                Clear
              </button>
            </div>
            <div class="flex flex-wrap gap-1.5">
              <div
                v-for="(attachment, index) in pendingCannedResponse.attachments"
                :key="attachment.id"
                class="inline-flex items-center gap-1.5 rounded-md bg-primary/12 px-2 py-1 text-xs text-primary"
              >
                <component
                  :is="getPendingAttachmentIcon(attachment.type)"
                  class="h-3.5 w-3.5"
                />
                <span class="max-w-[200px] truncate">{{
                  attachment.file_name
                }}</span>
                <button
                  type="button"
                  class="inline-flex items-center"
                  @click="removePendingCannedAttachment(index)"
                >
                  <X class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </div>

          <div
            v-if="isBatchPrintSelectionMode"
            class="mb-2 rounded-lg border border-primary/20 bg-primary/8 px-3 py-2"
          >
            <div class="flex items-center justify-between gap-3">
              <p class="text-xs text-primary">
                {{ $t("chat.batchPrintSelectionModeDesc") }}
              </p>
              <div class="flex items-center gap-2">
                <span class="text-xs text-primary">
                  {{
                    $t("chat.batchPrintSelectedCount", {
                      count: selectedBatchPrintCount,
                    })
                  }}
                </span>
                <Button
                  variant="ghost"
                  size="xs"
                  class="h-7 px-2 text-[11px] text-primary hover:bg-primary/12 hover:text-foreground"
                  @click="cancelBatchPrintSelection"
                >
                  {{ $t("common.cancel") }}
                </Button>
              </div>
            </div>
          </div>

          <form
            @submit.prevent="sendMessage"
            class="flex items-center gap-2 rounded-xl border border-border bg-background/80 p-2"
            :class="isCurrentChatSendRestricted && 'opacity-70'"
          >
            <Tooltip>
              <TooltipTrigger as-child>
                <span>
                  <Popover v-model:open="emojiPickerOpen">
                    <PopoverTrigger as-child>
                      <button
                        type="button"
                        :disabled="isCurrentChatSendRestricted"
                        class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:bg-accent"
                      >
                        <Smile
                          class="h-[18px] w-[18px] text-muted-foreground"
                        />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent side="top" align="start" class="w-auto p-0">
                      <EmojiPicker
                        :native="true"
                        :disable-skin-tones="true"
                        :theme="isDark ? 'dark' : 'light'"
                        @select="insertEmoji($event.i)"
                      />
                    </PopoverContent>
                  </Popover>
                </span>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.emoji") }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <span>
                  <CannedResponsePicker
                    :contact="contactsStore.currentContact"
                    :external-open="cannedPickerOpen"
                    :external-search="cannedSearchQuery"
                    :class="
                      isCurrentChatSendRestricted &&
                      'pointer-events-none opacity-60'
                    "
                    @select="insertCannedResponse"
                    @close="closeCannedPicker"
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.cannedResponses") }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <button
                  type="button"
                  :disabled="isCurrentChatSendRestricted"
                  class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:bg-accent"
                  @click="openFilePicker"
                >
                  <Paperclip class="h-[18px] w-[18px] text-muted-foreground" />
                </button>
              </TooltipTrigger>
              <TooltipContent>{{ $t("chat.attachFile") }}</TooltipContent>
            </Tooltip>
            <Tooltip v-if="configStore.showPrintButtons">
              <TooltipTrigger as-child>
                <button
                  type="button"
                  class="relative flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="
                    isPreparingBatchPrint ||
                    (isBatchPrintSelectionMode && !canMergeSelectedBubbleFiles)
                  "
                  @click="openBatchPrintPicker"
                >
                  <Loader2
                    v-if="isPreparingBatchPrint"
                    class="h-[18px] w-[18px] animate-spin text-muted-foreground"
                  />
                  <Check
                    v-else-if="isBatchPrintSelectionMode"
                    class="h-[18px] w-[18px] text-primary"
                  />
                  <Printer
                    v-else
                    class="h-[18px] w-[18px] text-muted-foreground"
                  />
                  <span
                    v-if="
                      isBatchPrintSelectionMode && selectedBatchPrintCount > 0
                    "
                    class="absolute -right-1 -top-1 min-w-4 rounded-full bg-primary px-1 text-center text-[10px] font-semibold leading-4 text-primary-foreground"
                  >
                    {{ selectedBatchPrintCount }}
                  </span>
                </button>
              </TooltipTrigger>
              <TooltipContent>{{
                isBatchPrintSelectionMode
                  ? $t("chat.batchPrintConfirmAction")
                  : $t("chat.mergePrint")
              }}</TooltipContent>
            </Tooltip>
            <input
              ref="fileInputRef"
              type="file"
              accept="*/*"
              multiple
              class="hidden"
              @change="handleFileSelect"
            />
            <textarea
              ref="messageInputRef"
              v-model="messageInput"
              :placeholder="$t('chat.typeMessage') + '...'"
              rows="1"
              class="min-h-[36px] max-h-[120px] flex-1 resize-none overflow-y-auto bg-transparent py-2 text-[14px] text-foreground placeholder:text-muted-foreground focus:outline-none"
              :disabled="isCurrentChatSendRestricted || isSending"
              @keydown.enter.exact.prevent="sendMessage"
              @input="autoResizeTextarea"
            />
            <button
              type="submit"
              class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
              :disabled="
                isCurrentChatSendRestricted || !canSendMessage || isSending
              "
            >
              <Send class="w-4 h-4 text-white" />
            </button>
          </form>
        </div>
      </template>
    </div>

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
      :active-media-upload="activeMediaUpload as PendingMediaUpload | null"
      :selected-media-uploads="selectedMediaUploads as PendingMediaUpload[]"
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
