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
import { localeDirectionManager } from "@/i18n/locale-direction";
import { wsService } from "@/services/websocket";
import {
  contactsService,
  chatbotService,
  messagesService,
  customActionsService,
  cannedResponsesService,
  type CustomAction,
  type ActionResult,
  type CannedResponseAttachment,
} from "@/services/api";
import { useTagsStore } from "@/stores/tags";
import { TagBadge } from "@/components/ui/tag-badge";
import { getTagColorClass } from "@/lib/constants";
import { canUserAccessInstance } from "@/lib/instance-access";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { normalizeRenderableAvatarURL } from "@/components/ui/avatar/avatar-url";
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
// Lazy-load emoji picker to reduce initial bundle size
const EmojiPicker = defineAsyncComponent(() => {
  return import("vue3-emoji-picker").then((module) => {
    // Import CSS when component loads
    import("vue3-emoji-picker/css");
    return module.default;
  });
});
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "vue-sonner";
import {
  Search,
  Send,
  Paperclip,
  FileText,
  Download,
  Printer,
  Image as ImageIcon,
  Smile,
  Phone,
  Check,
  CheckCheck,
  Clock,
  AlertCircle,
  User,
  UserPlus,
  UserMinus,
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
  Zap,
  Ticket,
  BarChart,
  Link,
  Mail,
  Globe,
  Pin,
  Code,
  RotateCw,
  Filter,
  StickyNote,
  Video,
  Music,
  RefreshCw,
} from "lucide-vue-next";
import { getInitials, getAvatarGradient } from "@/lib/utils";
import { getMessageSenderPhone, isGroupContact } from "@/lib/group-chat";
import {
  downloadMessageMedia,
  resolveMediaFilename,
} from "@/lib/media-actions";
import { getErrorMessage } from "@/lib/api-utils";
import {
  getCachedMediaBlob,
  prefetchMediaBlob,
  storeMediaBlobInPersistentCache,
  clearMissingMediaPrefetch,
} from "@/lib/media_prefetch_cache";
import {
  resolveWhatsAppMediaCategoryForFile,
  validateWhatsAppMediaFile,
  type WhatsAppMediaCategory,
} from "@/lib/whatsapp-media-policy";
import { resolvePreferredOutboundInstanceID } from "@/lib/chat-outbound-instance";
import { mergePhotosAndPdfsAndOpenPrintDialog } from "@/lib/media-merge-print";
import {
  isMergePrintableBubbleMessage,
  toMergePrintableFile,
} from "@/lib/chat-bubble-merge-print";
import {
  isMessagePrintSupported,
  openPrintDialogForSingleMessage,
} from "@/lib/single-media-print";
import {
  ChatSidebarUnifier,
  type ChatSidebarViewMode,
  type SidebarContactEntry,
} from "@/lib/chat-sidebar-unifier";
import { MentionContactResolver } from "@/lib/mention-contact-resolver";
import { MessageHistoryNavigator } from "@/lib/message-history-navigator";
import { useColorMode } from "@/composables/useColorMode";
import { useInfiniteScroll } from "@/composables/useInfiniteScroll";
import CannedResponsePicker from "@/components/chat/CannedResponsePicker.vue";
import ContactInfoPanel from "@/components/chat/ContactInfoPanel.vue";
import ConversationNotes from "@/components/chat/ConversationNotes.vue";
import InstanceTag from "@/components/chat/InstanceTag.vue";
import LinkifiedMessageText from "@/components/chat/LinkifiedMessageText.vue";
import MediaGroupBar from "@/components/chat/MediaGroupBar.vue";
import StatusStoriesBar from "@/components/chat/status/StatusStoriesBar.vue";
import { useInstancesStore } from "@/stores/instances";
import { useNotesStore } from "@/stores/notes";
import { CreateContactDialog } from "@/components/shared";
import { Info } from "lucide-vue-next";
import { EmptyState } from "@/components/ui/empty-state";
import { useMediaGroups } from "@/composables/useMediaGroups";
import { resolveChatBackgroundStyle } from "@/lib/chat-backgrounds";

const { t, locale } = useI18n();
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
    authStore.hasPermission("chat.bypass_claim", "read"),
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
const isRTL = computed(() =>
  localeDirectionManager.isRTL(String(locale.value)),
);

type TypingPresenceState = "composing" | "paused";

const TYPING_COMPOSING_THROTTLE_MS = 2500;
const TYPING_IDLE_PAUSE_MS = 3500;

const messageInput = ref("");
const messagesEndRef = ref<HTMLElement | null>(null);
const messageInputRef = ref<HTMLTextAreaElement | null>(null);
const isSending = ref(false);
const showSendConfirm = ref(false);
let typingPauseTimer: ReturnType<typeof setTimeout> | null = null;
const typingLastComposeAt = ref(0);
const typingLastState = ref<TypingPresenceState | null>(null);
const typingLastContactID = ref<string | null>(null);
const isAssignDialogOpen = ref(false);
const isAssigning = ref(false);
const isTransferring = ref(false);
const isResuming = ref(false);
const isInfoPanelOpen = ref(false);
const isNotesPanelOpen = ref(false);
const contactSessionData = ref<any>(null);

// Multi-account state
const selectedAccount = ref<string | null>(null);
const contactAccounts = ref<string[]>([]);
const chatSidebarUnifier = new ChatSidebarUnifier();
const chatSidebarViewMode = ref<ChatSidebarViewMode>(
  ChatSidebarUnifier.readViewMode(),
);
const ACCOUNT_TOGGLE_PREFIX = "acct:";
const CONTACT_TOGGLE_PREFIX = "contact:";

function toAccountToggleKey(accountName: string): string {
  return `${ACCOUNT_TOGGLE_PREFIX}${accountName}`;
}

function toContactToggleKey(contactID: string): string {
  return `${CONTACT_TOGGLE_PREFIX}${contactID}`;
}

function accountFromToggleKey(toggleKey?: string | null): string {
  if (!toggleKey || !toggleKey.startsWith(ACCOUNT_TOGGLE_PREFIX)) {
    return "";
  }
  return toggleKey.slice(ACCOUNT_TOGGLE_PREFIX.length).trim();
}

function contactIDFromToggleKey(toggleKey?: string | null): string {
  if (!toggleKey || !toggleKey.startsWith(CONTACT_TOGGLE_PREFIX)) {
    return "";
  }
  return toggleKey.slice(CONTACT_TOGGLE_PREFIX.length).trim();
}

function selectedAccountFilter(toggleKey?: string | null): string | undefined {
  const account = accountFromToggleKey(toggleKey);
  return account || undefined;
}

function resolveSourceContactForToggle(
  entry: SidebarContactEntry | null,
  toggleKey?: string | null,
): Contact | null {
  if (!entry || !toggleKey) return null;

  const contactID = contactIDFromToggleKey(toggleKey);
  if (contactID) {
    return findSidebarEntrySourceContact(entry, contactID);
  }

  const accountName = accountFromToggleKey(toggleKey);
  if (accountName && entry.contactsByAccount[accountName]) {
    return entry.contactsByAccount[accountName];
  }

  return null;
}

function resolveSelectedSourceContact(contact: Contact | null): Contact | null {
  if (!contact) return null;
  const entry = currentSidebarEntry.value;
  const selected = resolveSourceContactForToggle(entry, selectedAccount.value);
  if (selected) return selected;
  return contact;
}

function resolveExplicitSourceContact(contact: Contact | null): Contact | null {
  if (!contact) return null;
  return resolveSourceContactForToggle(
    currentSidebarEntry.value,
    selectedAccount.value,
  );
}

function clearTypingPauseTimer() {
  if (typingPauseTimer) {
    clearTimeout(typingPauseTimer);
    typingPauseTimer = null;
  }
}

function resetTypingPresenceState() {
  typingLastComposeAt.value = 0;
  typingLastState.value = null;
  typingLastContactID.value = null;
}

function isTypingPresenceEligibleContact(contact: Contact | null): boolean {
  if (!contact) return false;
  if (contact.is_group_chat === true) return false;

  const metadata = contact.metadata || {};
  if (metadata.is_group_chat === true || metadata.is_channel_chat === true) {
    return false;
  }

  const phone = String(contact.phone_number || "")
    .trim()
    .toLowerCase();
  if (!phone) return false;
  if (phone.endsWith("@g.us") || phone.endsWith("@newsletter")) return false;

  return true;
}

function resolveTypingInstanceID(contact: Contact): string | undefined {
  return resolvePreferredOutboundInstanceID({
    messages: contactsStore.messages,
    selectedSourceContact: resolveExplicitSourceContact(contact),
    currentContact: contact,
    selectedInstanceID: contactsStore.selectedInstanceId,
  });
}

async function sendTypingPresenceForContact(
  contact: Contact | null,
  state: TypingPresenceState,
  options?: { force?: boolean },
) {
  if (!isTypingPresenceEligibleContact(contact)) return;
  if (!contact) return;

  const contactID = contact.id;
  const force = options?.force === true;

  if (!force) {
    if (
      state === "paused" &&
      typingLastState.value === "paused" &&
      typingLastContactID.value === contactID
    ) {
      return;
    }

    if (
      state === "composing" &&
      typingLastContactID.value === contactID &&
      Date.now() - typingLastComposeAt.value < TYPING_COMPOSING_THROTTLE_MS
    ) {
      return;
    }
  }

  if (state === "composing") {
    typingLastComposeAt.value = Date.now();
  }
  typingLastState.value = state;
  typingLastContactID.value = contactID;

  try {
    await messagesService.sendTyping(contactID, {
      state,
      instance_id: resolveTypingInstanceID(contact),
    });
  } catch {
    // Typing presence is best-effort and should not interrupt chat UX.
  }
}

function scheduleTypingPaused(contact: Contact | null) {
  clearTypingPauseTimer();
  if (!isTypingPresenceEligibleContact(contact)) return;

  typingPauseTimer = setTimeout(() => {
    void sendTypingPresenceForContact(contact, "paused");
    clearTypingPauseTimer();
  }, TYPING_IDLE_PAUSE_MS);
}

function stopTypingForContact(
  contact: Contact | null,
  options?: { force?: boolean },
) {
  clearTypingPauseTimer();
  void sendTypingPresenceForContact(contact, "paused", options);
}

function findSidebarEntrySourceContact(
  entry: SidebarContactEntry | null,
  contactID: string,
): Contact | null {
  if (!entry || !contactID) return null;
  return (
    entry.sourceContacts.find((contact) => contact.id === contactID) || null
  );
}

function resolveInstanceToggleLabel(instanceID?: string): string {
  if (!instanceID) return "";
  const instance = instancesStore.instances.find(
    (item) => item.id === instanceID,
  );
  if (!instance) return "";
  if (typeof instance.name === "string" && instance.name.trim() !== "") {
    return instance.name.trim();
  }
  if (
    typeof (instance as Record<string, unknown>).phone_number === "string" &&
    String((instance as Record<string, unknown>).phone_number).trim() !== ""
  ) {
    return String((instance as Record<string, unknown>).phone_number).trim();
  }
  return "";
}

function resolveSidebarEntryInstanceIDs(entry: SidebarContactEntry): string[] {
  const instanceIDs: string[] = [];
  const seen = new Set<string>();

  const appendInstanceID = (rawValue?: string) => {
    const instanceID = (rawValue || "").trim();
    if (!instanceID || seen.has(instanceID)) return;
    seen.add(instanceID);
    instanceIDs.push(instanceID);
  };

  for (const sourceContact of entry.sourceContacts || []) {
    appendInstanceID(
      typeof sourceContact.instance_id === "string"
        ? sourceContact.instance_id
        : "",
    );
  }

  for (const instanceID of entry.sourceInstanceIDs || []) {
    appendInstanceID(instanceID);
  }

  if (instanceIDs.length === 0 && entry.displayContact.instance_id) {
    appendInstanceID(entry.displayContact.instance_id);
  }

  return instanceIDs;
}

function getSidebarEntryInstanceCount(entry: SidebarContactEntry): number {
  return resolveSidebarEntryInstanceIDs(entry).length;
}

function hasSidebarEntryMultipleInstances(entry: SidebarContactEntry): boolean {
  return getSidebarEntryInstanceCount(entry) > 1;
}

function getSidebarEntryPrimaryInstanceID(
  entry: SidebarContactEntry,
): string | undefined {
  return resolveSidebarEntryInstanceIDs(entry)[0];
}

function resolveSidebarEntryInstanceLabel(
  entry: SidebarContactEntry,
  instanceID?: string,
): string {
  const normalizedInstanceID = (instanceID || "").trim();
  if (!normalizedInstanceID) return "";

  for (const sourceContact of entry.sourceContacts || []) {
    const sourceInstanceID =
      typeof sourceContact.instance_id === "string"
        ? sourceContact.instance_id.trim()
        : "";
    if (sourceInstanceID !== normalizedInstanceID) {
      continue;
    }
    const accountLabel =
      typeof sourceContact.whatsapp_account === "string"
        ? sourceContact.whatsapp_account.trim()
        : "";
    if (accountLabel) {
      return accountLabel;
    }
  }

  const displayInstanceID =
    typeof entry.displayContact.instance_id === "string"
      ? entry.displayContact.instance_id.trim()
      : "";
  if (displayInstanceID === normalizedInstanceID) {
    const displayAccount =
      typeof entry.displayContact.whatsapp_account === "string"
        ? entry.displayContact.whatsapp_account.trim()
        : "";
    if (displayAccount) {
      return displayAccount;
    }
  }

  return "";
}

function getSidebarEntryPrimaryInstanceLabel(
  entry: SidebarContactEntry,
): string {
  return resolveSidebarEntryInstanceLabel(
    entry,
    getSidebarEntryPrimaryInstanceID(entry),
  );
}

function formatAccountToggleLabel(toggleKey: string): string {
  const account = accountFromToggleKey(toggleKey);
  if (account) {
    return account;
  }

  const contactID = contactIDFromToggleKey(toggleKey);
  if (contactID) {
    const sourceContact =
      findSidebarEntrySourceContact(currentSidebarEntry.value, contactID) ||
      contactsStore.contacts.find((contact) => contact.id === contactID) ||
      null;
    if (sourceContact) {
      const instanceLabel = resolveInstanceToggleLabel(
        sourceContact.instance_id,
      );
      if (instanceLabel) {
        return instanceLabel;
      }
      const contactAccount = (sourceContact.whatsapp_account || "").trim();
      if (contactAccount) {
        return contactAccount;
      }
      if (sourceContact.instance_id) {
        return sourceContact.instance_id;
      }
      if (sourceContact.phone_number) {
        return sourceContact.phone_number;
      }
    }
  }

  return toggleKey;
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

interface PendingMediaUpload {
  id: string;
  file: File;
  category: WhatsAppMediaCategory;
  previewUrl: string | null;
}

// File upload state
const fileInputRef = ref<HTMLInputElement | null>(null);
const selectedMediaUploads = ref<PendingMediaUpload[]>([]);
const activeMediaPreviewID = ref<string | null>(null);
const isMediaDialogOpen = ref(false);
type ChatMediaViewerType = "image" | "video" | "audio" | "document";
const isChatMediaViewerOpen = ref(false);
const chatMediaViewerURL = ref("");
const chatMediaViewerType = ref<ChatMediaViewerType>("image");
const chatMediaViewerTitle = ref("");
const isProfilePhotoDialogOpen = ref(false);
const profilePhotoContact = ref<Contact | null>(null);
const profilePhotoImageFailed = ref(false);
const activeProfilePhotoURL = computed(() =>
  normalizeRenderableAvatarURL(profilePhotoContact.value?.avatar_url),
);
const mediaCaption = ref("");
const isUploadingMedia = ref(false);
const mediaUploadProgress = ref<{ current: number; total: number } | null>(
  null,
);
const isPreparingBatchPrint = ref(false);
const isBatchPrintSelectionMode = ref(false);
const selectedBatchPrintMessageIds = ref<string[]>([]);
const selectedMediaCount = computed(() => selectedMediaUploads.value.length);
const activeMediaUpload = computed<PendingMediaUpload | null>(() => {
  if (activeMediaPreviewID.value) {
    const matchedUpload = selectedMediaUploads.value.find(
      (upload) => upload.id === activeMediaPreviewID.value,
    );
    if (matchedUpload) {
      return matchedUpload;
    }
  }

  return selectedMediaUploads.value[0] ?? null;
});
const canApplyMediaCaption = computed(
  () =>
    selectedMediaCount.value === 1 &&
    activeMediaUpload.value?.category !== "audio",
);
const mediaDialogDescription = computed(() => {
  if (selectedMediaCount.value === 0) {
    return "";
  }
  if (selectedMediaCount.value === 1) {
    return activeMediaUpload.value?.file.name ?? "";
  }
  return t("chat.mediaFilesSelected", { count: selectedMediaCount.value });
});
const mediaSendButtonLabel = computed(() =>
  selectedMediaCount.value > 1 ? t("chat.sendFiles") : t("chat.send"),
);
const mediaUploadingLabel = computed(() => {
  if (selectedMediaCount.value > 1 && mediaUploadProgress.value) {
    return t("chat.mediaSendingProgress", mediaUploadProgress.value);
  }
  return `${t("chat.sending")}...`;
});

// Cache for media blob URLs (message_id -> blob URL)
const mediaBlobUrls = ref<Record<string, string>>({});
const mediaLoadingStates = ref<Record<string, boolean>>({});
const mediaBlobCache = new Map<string, Blob>();
const MAX_MEDIA_LOAD_CONCURRENCY = 4;
const pendingMediaQueue: Message[] = [];
const queuedMediaMessageIDs = new Set<string>();
const inFlightMediaRequests = new Map<string, AbortController>();
let activeMediaLoadCount = 0;
let mediaLoadGeneration = 0;

// Canned responses slash command state
const cannedPickerOpen = ref(false);
const cannedSearchQuery = ref("");
const pendingCannedResponse = ref<{
  id: string;
  attachments: CannedResponseAttachment[];
} | null>(null);

// Sticky date header state
const stickyDate = ref("");
const showStickyDate = ref(false);
let stickyDateTimeout: ReturnType<typeof setTimeout> | null = null;
let quoteHighlightTimeout: ReturnType<typeof setTimeout> | null = null;
const isQuoteNavigationInProgress = ref(false);
const QUOTE_NAVIGATION_MAX_HISTORY_REQUESTS = 64;

// Emoji picker state
const emojiPickerOpen = ref(false);

// Custom actions state
const customActions = ref<CustomAction[]>([]);
const executingActionId = ref<string | null>(null);

// Tags filter state
const isTagFilterOpen = ref(false);

// Service window state
const isServiceWindowExpired = computed(() => {
  const contact = contactsStore.currentContact;
  if (!contact) return false;
  if (configStore.isWhatsmeow) return false;
  return contact.service_window_open === false;
});

const hasPendingCannedAttachments = computed(() => {
  return (pendingCannedResponse.value?.attachments.length ?? 0) > 0;
});

const canSendMessage = computed(() => {
  return (
    Boolean(messageInput.value.trim()) || hasPendingCannedAttachments.value
  );
});

const selectedBatchPrintCount = computed(
  () => selectedBatchPrintMessageIds.value.length,
);

const hasMergePrintableBubbles = computed(() =>
  contactsStore.messages.some((message) =>
    isMergePrintableBubbleMessage(message),
  ),
);

const canMergeSelectedBubbleFiles = computed(
  () => selectedBatchPrintCount.value >= 2,
);

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
const CONTACTS_SIDEBAR_WIDTH_STORAGE_KEY = "chat.contactsSidebarWidth";
const CONTACTS_SIDEBAR_MIN_WIDTH = 280;
const CONTACTS_SIDEBAR_MAX_WIDTH = 500;
const CONTACTS_SIDEBAR_DEFAULT_WIDTH = 320;

function clampContactsSidebarWidth(value: number): number {
  return Math.min(
    CONTACTS_SIDEBAR_MAX_WIDTH,
    Math.max(CONTACTS_SIDEBAR_MIN_WIDTH, value),
  );
}

function readContactsSidebarWidth(): number {
  try {
    const stored = Number(
      localStorage.getItem(CONTACTS_SIDEBAR_WIDTH_STORAGE_KEY),
    );
    if (Number.isFinite(stored) && stored > 0) {
      return clampContactsSidebarWidth(stored);
    }
  } catch {
    // Ignore localStorage errors
  }
  return CONTACTS_SIDEBAR_DEFAULT_WIDTH;
}

const contactsSidebarWidth = ref(readContactsSidebarWidth());
const isContactsSidebarResizing = ref(false);
const isContactsSidebarCompact = computed(
  () => contactsSidebarWidth.value <= 320,
);
const isContactsSidebarWide = computed(() => contactsSidebarWidth.value >= 420);
let contactsSidebarResizeStartX = 0;
let contactsSidebarResizeStartWidth = contactsSidebarWidth.value;

function setContactsSidebarWidth(value: number) {
  const nextWidth = clampContactsSidebarWidth(value);
  contactsSidebarWidth.value = nextWidth;
  try {
    localStorage.setItem(CONTACTS_SIDEBAR_WIDTH_STORAGE_KEY, String(nextWidth));
  } catch {
    // Ignore localStorage errors
  }
}

function onContactsSidebarResizeMove(event: MouseEvent) {
  if (!isContactsSidebarResizing.value) return;
  const deltaX = isRTL.value
    ? contactsSidebarResizeStartX - event.clientX
    : event.clientX - contactsSidebarResizeStartX;
  setContactsSidebarWidth(contactsSidebarResizeStartWidth + deltaX);
}

function stopContactsSidebarResize() {
  if (!isContactsSidebarResizing.value) return;
  isContactsSidebarResizing.value = false;
  window.removeEventListener("mousemove", onContactsSidebarResizeMove);
  window.removeEventListener("mouseup", stopContactsSidebarResize);
}

function startContactsSidebarResize(event: MouseEvent) {
  if (window.innerWidth < 768) return;
  isContactsSidebarResizing.value = true;
  contactsSidebarResizeStartX = event.clientX;
  contactsSidebarResizeStartWidth = contactsSidebarWidth.value;
  window.addEventListener("mousemove", onContactsSidebarResizeMove);
  window.addEventListener("mouseup", stopContactsSidebarResize);
  event.preventDefault();
}

function openAddContactDialog() {
  isAddContactOpen.value = true;
}

async function onContactCreated(contact: any) {
  // Refresh contacts and select the new one
  await refreshContactsSidebar();
  if (contact?.id) {
    router.push({
      name: "chat-conversation",
      params: { contactId: contact.id },
    });
  }
}

// Infinite scroll for contacts (load more at bottom)
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
const isSidebarUnifiedMode = computed(
  () => chatSidebarViewMode.value === "unified",
);

function refreshChatSidebarViewModePreference() {
  chatSidebarViewMode.value = ChatSidebarUnifier.readViewMode();
}

function getSidebarEntryPreferredContact(entry: SidebarContactEntry): Contact {
  const selectedAccountName = accountFromToggleKey(selectedAccount.value);
  if (selectedAccountName && entry.contactsByAccount[selectedAccountName]) {
    return entry.contactsByAccount[selectedAccountName];
  }

  const selectedContactID = contactIDFromToggleKey(selectedAccount.value);
  if (selectedContactID) {
    const selectedContact = findSidebarEntrySourceContact(
      entry,
      selectedContactID,
    );
    if (selectedContact) {
      return selectedContact;
    }
  }

  const displayAccount =
    typeof entry.displayContact.whatsapp_account === "string"
      ? entry.displayContact.whatsapp_account.trim()
      : "";
  if (displayAccount && entry.contactsByAccount[displayAccount]) {
    return entry.contactsByAccount[displayAccount];
  }

  if (entry.accountNames.length > 0) {
    const fallbackContact = entry.contactsByAccount[entry.accountNames[0]];
    if (fallbackContact) {
      return fallbackContact;
    }
  }

  return entry.displayContact;
}

function isSidebarEntryActive(entry: SidebarContactEntry): boolean {
  const currentContactID = contactsStore.currentContact?.id;
  if (!currentContactID) return false;
  return entry.sourceContactIDs.includes(currentContactID);
}

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
const isClaimingCurrentChat = ref(false);
const isClosingCurrentChat = ref(false);
const isReopeningCurrentChat = ref(false);
const isUpdatingCurrentChatPublic = ref(false);
const deletedMessageText = "(This message was deleted)";
const legacyDeletedMessageText = "This message was deleted";
const mentionContactResolver = new MentionContactResolver();
const mentionResolutionVersion = ref(0);

function getGroupSenderPhone(message: Message): string {
  return getMessageSenderPhone(message);
}

function isGroupMessage(message: Message): boolean {
  if (message.is_group_chat === true) {
    return true;
  }
  if (
    typeof message.conversation_id === "string" &&
    message.conversation_id.endsWith("@g.us")
  ) {
    return true;
  }
  return isCurrentGroupChat.value;
}

function shouldShowGroupSenderPhone(message: Message): boolean {
  if (message.direction !== "incoming" || !isGroupMessage(message)) {
    return false;
  }
  return getGroupSenderPhone(message) !== "";
}

function normalizeDeletedMessageText(content: string): string {
  if (content.trim().toLowerCase() === legacyDeletedMessageText.toLowerCase()) {
    return deletedMessageText;
  }
  return content;
}

function preloadMentionResolverFromKnownContacts(): void {
  const changed = mentionContactResolver.preloadContacts([
    ...contactsStore.contacts,
    ...contactsStore.pendingChats,
    ...contactsStore.assignedChats,
    ...contactsStore.closedChats,
  ]);

  if (changed) {
    mentionResolutionVersion.value += 1;
  }
}

function applyMentionDisplayNames(content: string): string {
  if (!content || !content.includes("@")) {
    return content;
  }

  // Keep render reactive to async lookup results.
  const revision = mentionResolutionVersion.value;
  if (revision < 0) {
    return content;
  }

  return mentionContactResolver.replaceMentions(content);
}

async function resolveMentionsForCurrentMessages(): Promise<void> {
  preloadMentionResolverFromKnownContacts();

  const texts: string[] = [];
  for (const message of contactsStore.messages) {
    const raw = getMessageContentRaw(message);
    if (raw && raw.includes("@")) {
      texts.push(raw);
    }
  }

  if (texts.length === 0) {
    return;
  }

  const changed = await mentionContactResolver.resolveMentionsInTexts(texts);
  if (changed) {
    mentionResolutionVersion.value += 1;
  }
}

function isDeletedMessage(message: Message): boolean {
  if (message.content && typeof message.content === "object") {
    const metadata = (message.content as { metadata?: Record<string, any> })
      .metadata;
    if (metadata?.revoked === true) {
      return true;
    }
  }

  const body = getMessageContent(message).trim();
  if (!body) {
    return false;
  }

  return (
    body.includes(deletedMessageText) || body.includes(legacyDeletedMessageText)
  );
}

function isSystemEventMessage(message: Message): boolean {
  const rawValue = message.metadata?.system_event;
  return (
    rawValue === true ||
    rawValue === "true" ||
    rawValue === 1 ||
    rawValue === "1"
  );
}

// Check if current user can assign contacts (permission-based)
const canAssignContacts = computed(() => {
  return (
    authStore.hasPermission("chat.assign", "write") ||
    authStore.hasPermission("contacts", "write")
  );
});

const canReadCustomActions = computed(() => {
  return authStore.hasPermission("custom_actions", "read");
});

// Get list of users for assignment
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

// Icon mapping for custom actions
const actionIconMap: Record<string, any> = {
  ticket: Ticket,
  user: User,
  "bar-chart": BarChart,
  link: Link,
  phone: Phone,
  mail: Mail,
  "file-text": FileText,
  "external-link": ExternalLink,
  zap: Zap,
  globe: Globe,
  code: Code,
};

function getActionIcon(iconName: string) {
  return actionIconMap[iconName] || Zap;
}

async function fetchCustomActions() {
  try {
    const response = await customActionsService.list();
    const data = (response.data as any).data || response.data;
    customActions.value = (data.custom_actions || []).filter(
      (a: CustomAction) => a.is_active,
    );
  } catch (error) {
    // Silently fail - custom actions are optional
    console.error("Failed to fetch custom actions:", error);
  }
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

async function executeCustomAction(action: CustomAction) {
  if (!contactsStore.currentContact || executingActionId.value) return;

  executingActionId.value = action.id;
  try {
    const response = await customActionsService.execute(
      action.id,
      contactsStore.currentContact.id,
    );
    let result: ActionResult = (response.data as any).data || response.data;

    // JavaScript actions are now executed server-side via goja.
    // The response already contains structured result fields (toast, clipboard, redirect_url, message).

    // Handle different result types
    if (result.redirect_url) {
      // Open URL action result - prepend base path for relative URLs
      let redirectUrl = result.redirect_url;
      if (redirectUrl.startsWith("/api/")) {
        const basePath = ((window as any).__BASE_PATH__ ?? "").replace(
          /\/$/,
          "",
        );
        redirectUrl = basePath + redirectUrl;
      }
      try {
        const parsed = new URL(redirectUrl, window.location.origin);
        if (parsed.protocol === "http:" || parsed.protocol === "https:") {
          window.open(parsed.href, "_blank", "noopener,noreferrer");
        }
      } catch {
        // Invalid URL, ignore
      }
    }

    if (result.clipboard) {
      // Copy to clipboard
      await navigator.clipboard.writeText(result.clipboard);
      toast.success(t("common.copiedToClipboard"));
    }

    if (result.toast) {
      // Show toast notification
      if (result.toast.type === "success") {
        toast.success(result.toast.message);
      } else if (result.toast.type === "error") {
        toast.error(result.toast.message);
      } else {
        toast.info(result.toast.message);
      }
    } else if (result.success && !result.redirect_url && !result.clipboard) {
      // Default success message
      toast.success(result.message || t("chat.actionExecuted"));
    } else if (!result.success) {
      toast.error(result.message || t("chat.actionFailed"));
    }
  } catch (error: any) {
    const message = error.response?.data?.message || "Failed to execute action";
    toast.error(message);
  } finally {
    executingActionId.value = null;
  }
}

// Search state for assignment dialog
const assignSearchQuery = ref("");
const debouncedAssignSearchQuery = ref("");
let assignSearchTimer: ReturnType<typeof setTimeout> | null = null;

watch(assignSearchQuery, (val) => {
  if (assignSearchTimer) clearTimeout(assignSearchTimer);
  assignSearchTimer = setTimeout(() => {
    debouncedAssignSearchQuery.value = val;
  }, 150);
});

// Filtered users for assignment dialog
const filteredAssignableUsers = computed(() => {
  const query = debouncedAssignSearchQuery.value.toLowerCase().trim();
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
    usersStore.fetchUsers({ limit: 100 }).catch(() => {
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
  stopTypingForContact(activeContact, { force: true });
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
  // Clean up blob URLs to prevent memory leaks
  Object.values(mediaBlobUrls.value).forEach((url) => {
    URL.revokeObjectURL(url);
  });
  mediaBlobUrls.value = {};
  mediaBlobCache.clear();
  // Clear sticky date timeout
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
  stopTypingForContact(previousContact, { force: true });
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
      stopTypingForContact(contactsStore.currentContact, { force: true });
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
      stopTypingForContact(contactsStore.currentContact, { force: true });
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
  profilePhotoImageFailed.value = false;
  isProfilePhotoDialogOpen.value = true;
}

function handleProfilePhotoDialogOpenChange(open: boolean) {
  isProfilePhotoDialogOpen.value = open;
  if (!open) {
    profilePhotoContact.value = null;
    profilePhotoImageFailed.value = false;
  }
}

function handleProfilePhotoImageError() {
  profilePhotoImageFailed.value = true;
}

async function handleContactDeleted(contactId: string) {
  if (contactsStore.currentContact?.id === contactId) {
    stopTypingForContact(contactsStore.currentContact, { force: true });
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

  stopTypingForContact(contactsStore.currentContact);
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
    showSendConfirm.value = true;
    setTimeout(() => { showSendConfirm.value = false; }, 800);
    await nextTick();
    scrollToBottom();
  } catch (error: any) {
    const message = resolveSendErrorMessage(error, t("chat.sendMessageFailed"));
    toast.error(message);
  } finally {
    isSending.value = false;
  }
}

const retryingMessageId = ref<string | null>(null);
const revokingMessageId = ref<string | null>(null);

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

function getReplyPreviewContent(message: Message): string {
  if (!message.reply_to_message) return "";
  const reply = message.reply_to_message;
  if (reply.message_type === "text") {
    const rawBody =
      typeof reply.content === "string"
        ? reply.content
        : reply.content?.body || "";
    const body = applyMentionDisplayNames(normalizeDeletedMessageText(rawBody));
    return body.length > 50 ? body.substring(0, 50) + "..." : body;
  }
  if (reply.message_type === "button_reply") {
    const body =
      typeof reply.content === "string"
        ? reply.content
        : reply.content?.body || "";
    const displayBody = applyMentionDisplayNames(body);
    return displayBody.length > 50
      ? displayBody.substring(0, 50) + "..."
      : displayBody;
  }
  if (reply.message_type === "interactive") {
    const body =
      typeof reply.content === "string"
        ? reply.content
        : (reply as any).interactive_data?.body || reply.content?.body || "";
    const displayBody = applyMentionDisplayNames(body);
    return displayBody.length > 50
      ? displayBody.substring(0, 50) + "..."
      : displayBody;
  }
  if (reply.message_type === "template") {
    const body = reply.content?.body || "";
    const displayBody = applyMentionDisplayNames(body);
    return displayBody.length > 50
      ? displayBody.substring(0, 50) + "..."
      : displayBody;
  }
  if (reply.message_type === "image") {
    const body =
      typeof reply.content === "string"
        ? reply.content
        : reply.content?.body || "";
    const displayBody = applyMentionDisplayNames(body);
    if (displayBody.trim() !== "") {
      return displayBody.length > 50
        ? displayBody.substring(0, 50) + "..."
        : displayBody;
    }
    return "[Photo]";
  }
  if (reply.message_type === "video") return "[Video]";
  if (reply.message_type === "audio") return "[Audio]";
  if (reply.message_type === "document") return "[Document]";
  if (reply.message_type === "location") return "[Location]";
  if (reply.message_type === "contacts" || reply.message_type === "contact")
    return "[Contact]";
  if (reply.message_type === "sticker") return "[Sticker]";
  return "[Message]";
}

function getReplyPreviewMediaURL(message: Message): string {
  const rawURL =
    typeof message.reply_to_message?.media_url === "string"
      ? message.reply_to_message.media_url.trim()
      : "";
  if (!rawURL) return "";

  const lower = rawURL.toLowerCase();
  if (
    lower.startsWith("http://") ||
    lower.startsWith("https://") ||
    lower.startsWith("data:") ||
    rawURL.startsWith("/")
  ) {
    return rawURL;
  }
  return "";
}

function shouldShowReplyPreviewThumbnail(message: Message): boolean {
  return (
    message.reply_to_message?.message_type === "image" &&
    getReplyPreviewMediaURL(message) !== ""
  );
}

function resolveReplyPreviewMediaType(message: Message): ChatMediaViewerType {
  const type = message.reply_to_message?.message_type;
  if (type === "video") return "video";
  if (type === "audio") return "audio";
  if (type === "document") return "document";
  return "image";
}

function openChatMediaViewer(
  url: string,
  type: ChatMediaViewerType,
  title?: string,
): void {
  const normalizedURL = typeof url === "string" ? url.trim() : "";
  if (!normalizedURL) return;
  chatMediaViewerURL.value = normalizedURL;
  chatMediaViewerType.value = type;
  chatMediaViewerTitle.value = (title || "").trim();
  isChatMediaViewerOpen.value = true;
}

function closeChatMediaViewer(): void {
  isChatMediaViewerOpen.value = false;
  chatMediaViewerURL.value = "";
  chatMediaViewerType.value = "image";
  chatMediaViewerTitle.value = "";
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
    message.reply_to_message?.media_filename,
  );
}

function handleReplyPreviewThumbnailError(event: Event): void {
  const target = event.target as HTMLImageElement | null;
  if (!target) return;
  target.style.display = "none";
}

function getReplyAuthorLabel(message: Message): string {
  if (!message.reply_to_message) {
    return "You";
  }
  if (message.reply_to_message.direction === "outgoing") {
    return "You";
  }
  if (
    (isGroupMessage(message) || isCurrentGroupChat.value) &&
    message.reply_to_message.sender_phone
  ) {
    return message.reply_to_message.sender_phone;
  }
  return (
    contactsStore.currentContact?.profile_name ||
    contactsStore.currentContact?.name ||
    "Customer"
  );
}

function getReplyingToAuthorLabel(message: Message | null): string {
  if (!message) {
    return "Yourself";
  }
  if (message.direction === "outgoing") {
    return "Yourself";
  }
  if (isGroupMessage(message) || isCurrentGroupChat.value) {
    const senderPhone = getGroupSenderPhone(message);
    if (senderPhone) {
      return senderPhone;
    }
  }
  return (
    contactsStore.currentContact?.profile_name ||
    contactsStore.currentContact?.name ||
    "Customer"
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
  if (payload.attachments && payload.attachments.length > 0) {
    pendingCannedResponse.value = {
      id: payload.id,
      attachments: [...payload.attachments],
    };
  } else {
    pendingCannedResponse.value = null;
  }
  cannedPickerOpen.value = false;
  cannedSearchQuery.value = "";
}

function closeCannedPicker() {
  cannedPickerOpen.value = false;
  cannedSearchQuery.value = "";
}

function clearPendingCannedAttachments() {
  pendingCannedResponse.value = null;
}

function removePendingCannedAttachment(index: number) {
  if (!pendingCannedResponse.value) return;
  pendingCannedResponse.value.attachments =
    pendingCannedResponse.value.attachments.filter(
      (_, currentIndex) => currentIndex !== index,
    );
  if (pendingCannedResponse.value.attachments.length === 0) {
    pendingCannedResponse.value = null;
  }
}

function getPendingAttachmentIcon(type: string) {
  return type === "video" ? Play : ImageIcon;
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

// Reaction handling
const reactionPickerMessageId = ref<string | null>(null);
const quickReactionEmojis = ["👍", "❤️", "😂", "😮", "😢", "🙏"];

async function sendReaction(messageId: string, emoji: string) {
  if (!contactsStore.currentContact) return;

  try {
    const response = await messagesService.sendReaction(
      contactsStore.currentContact.id,
      messageId,
      emoji,
    );
    // Update will come via WebSocket, but we can update locally for immediate feedback
    const data = response.data.data || response.data;
    contactsStore.updateMessageReactions(messageId, data.reactions);
  } catch (error) {
    toast.error(t("chat.reactionFailed"));
  }
  reactionPickerMessageId.value = null;
}

function _toggleReactionPicker(messageId: string) {
  if (reactionPickerMessageId.value === messageId) {
    reactionPickerMessageId.value = null;
  } else {
    reactionPickerMessageId.value = messageId;
  }
}
void _toggleReactionPicker; // Suppress unused warning

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
    stopTypingForContact(activeContact);
  } else if (cannedPickerOpen.value) {
    // Close picker if user removes the /
    cannedPickerOpen.value = false;
    cannedSearchQuery.value = "";
  }

  if (!activeContact) {
    clearTypingPauseTimer();
    resetTypingPresenceState();
    return;
  }

  if (val.startsWith("/")) {
    return;
  }

  if (isCurrentChatSendRestricted.value || isCurrentChatClosed.value) {
    stopTypingForContact(activeContact);
    return;
  }

  if (val.trim() === "") {
    stopTypingForContact(activeContact);
    return;
  }

  void sendTypingPresenceForContact(activeContact, "composing");
  scheduleTypingPaused(activeContact);
});

async function assignContactToUser(userId: string | null) {
  if (!contactsStore.currentContact || isAssigning.value) return;
  isAssigning.value = true;

  try {
    await contactsService.assign(contactsStore.currentContact.id, userId);
    if (userId) {
      const assignedUser = filteredAssignableUsers.value.find(u => u.id === userId);
      toast.success(
        assignedUser
          ? t("chat.assignedToName", { name: assignedUser.full_name })
          : t("chat.contactAssigned"),
      );
    } else {
      toast.success(t("chat.contactUnassigned"));
    }
    // Update current contact with new assignment
    contactsStore.currentContact = {
      ...contactsStore.currentContact,
      assigned_user_id: userId || undefined,
      status: userId ? "open" : "pending",
    };
    // Refresh contacts list
    await refreshContactsSidebar();
  } catch (error: any) {
    const message = error.response?.data?.message || t("chat.assignFailed");
    toast.error(message);
  } finally {
    isAssigning.value = false;
  }
}

async function claimCurrentChat() {
  if (!contactsStore.currentContact || isClaimingCurrentChat.value) return;
  isClaimingCurrentChat.value = true;
  try {
    const updated = await contactsStore.claimChat(
      contactsStore.currentContact.id,
    );
    if (!updated) {
      toast.error(t("chat.chatClaimFailed"));
      return;
    }
    toast.success(t("chat.chatClaimedSuccess"));
    contactsStore.setActiveChatTab("assigned");
    await refreshContactsSidebar();
    const selectionSequence = ++contactSelectionSequence;
    resetMediaLoadingPipeline();
    await selectContact(updated.id, selectionSequence);
  } catch (error: any) {
    const message = error?.response?.data?.message || "Failed to claim chat";
    toast.error(message);
  } finally {
    isClaimingCurrentChat.value = false;
  }
}

async function closeCurrentChat() {
  if (!contactsStore.currentContact || isClosingCurrentChat.value) return;
  isClosingCurrentChat.value = true;
  try {
    const updated = await contactsStore.closeChat(
      contactsStore.currentContact.id,
    );
    if (!updated) {
      toast.error("Failed to close chat");
      return;
    }
    toast.success("Chat closed");
    await refreshContactsSidebar();
    stopTypingForContact(contactsStore.currentContact, { force: true });
    resetTypingPresenceState();
    wsService.setCurrentContact(null);
    contactsStore.setCurrentContact(null);
    contactsStore.clearMessages();
    router.push("/chat");
  } catch (error: any) {
    const message = error?.response?.data?.message || "Failed to close chat";
    toast.error(message);
  } finally {
    isClosingCurrentChat.value = false;
  }
}

async function reopenCurrentChat() {
  if (!contactsStore.currentContact || isReopeningCurrentChat.value) return;
  isReopeningCurrentChat.value = true;
  try {
    const updated = await contactsStore.reopenChat(
      contactsStore.currentContact.id,
    );
    if (!updated) {
      toast.error("Failed to reopen chat");
      return;
    }
    toast.success("Chat reopened and moved to pending queue");
    contactsStore.setActiveChatTab("pending");
    await refreshContactsSidebar();
    const selectionSequence = ++contactSelectionSequence;
    resetMediaLoadingPipeline();
    await selectContact(updated.id, selectionSequence);
  } catch (error: any) {
    const message = error?.response?.data?.message || "Failed to reopen chat";
    toast.error(message);
  } finally {
    isReopeningCurrentChat.value = false;
  }
}

async function toggleCurrentChatPublicVisibility() {
  if (
    !contactsStore.currentContact ||
    !canToggleCurrentChatPublic.value ||
    isUpdatingCurrentChatPublic.value
  ) {
    return;
  }

  isUpdatingCurrentChatPublic.value = true;
  const nextIsPublic = contactsStore.currentContact.is_public !== true;
  try {
    const updated = await contactsStore.setChatPublic(
      contactsStore.currentContact.id,
      nextIsPublic,
    );
    if (!updated) {
      toast.error("Failed to update public chat setting");
      return;
    }
    toast.success(
      nextIsPublic ? t("chat.publicChatEnabled") : t("chat.publicChatDisabled"),
    );

    if (
      nextIsPublic &&
      contactsStore.currentContact?.id === updated.id &&
      contactsStore.isMessageAccessRestricted
    ) {
      const activeAccountFilter = selectedAccountFilter(selectedAccount.value);
      await contactsStore.fetchMessages(
        updated.id,
        activeAccountFilter ? { account: activeAccountFilter } : undefined,
      );
    }

    await refreshContactsSidebar();
  } catch (error: any) {
    const message =
      error?.response?.data?.message || t("chat.publicChatUpdateFailed");
    toast.error(message);
  } finally {
    isUpdatingCurrentChatPublic.value = false;
  }
}

async function transferToAgent() {
  if (!contactsStore.currentContact) return;

  isTransferring.value = true;
  try {
    const currentContact = contactsStore.currentContact as any;
    const activeAccountFilter = selectedAccountFilter(selectedAccount.value);
    const fallbackAccount =
      typeof currentContact.whatsapp_account === "string"
        ? currentContact.whatsapp_account.trim()
        : "";
    await chatbotService.createTransfer({
      contact_id: currentContact.id,
      whatsapp_account: activeAccountFilter || fallbackAccount,
      source: "manual",
    });
    toast.success(t("chat.transferSuccess"), {
      description: t("chat.transferSuccessDesc"),
    });
    // Refresh transfers store (WebSocket will also update, but this ensures immediate sync)
    await transfersStore.fetchTransfers({ status: "active" });
  } catch (error: any) {
    const message = error.response?.data?.message || t("chat.transferFailed");
    toast.error(message);
  } finally {
    isTransferring.value = false;
  }
}

async function resumeChatbot() {
  if (!activeTransferId.value) return;

  const currentContactId = contactsStore.currentContact?.id;
  isResuming.value = true;
  try {
    await chatbotService.resumeTransfer(activeTransferId.value);
    toast.success(t("chat.resumeSuccess"), {
      description: t("chat.resumeSuccessDesc"),
    });
    // Refresh transfers store to update UI
    await transfersStore.fetchTransfers({ status: "active" });
    // Refresh contacts list (assignment may have changed)
    await refreshContactsSidebar();

    // Check if current contact is still in the list (may have been unassigned)
    if (currentContactId) {
      const stillExists = contactsStore.contacts.some(
        (c) => c.id === currentContactId,
      );
      if (!stillExists) {
        // Contact no longer visible to this user, navigate away
        stopTypingForContact(contactsStore.currentContact, { force: true });
        resetTypingPresenceState();
        contactsStore.setCurrentContact(null);
        contactsStore.clearMessages();
        router.push("/chat");
      }
    }
  } catch (error: any) {
    const message = error.response?.data?.message || t("chat.resumeFailed");
    toast.error(message);
  } finally {
    isResuming.value = false;
  }
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

function formatMessageTime(dateStr: string) {
  const date = new Date(dateStr);
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
  });
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

function getDateLabel(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const messageDate = new Date(
    date.getFullYear(),
    date.getMonth(),
    date.getDate(),
  );
  const diffDays = Math.floor(
    (today.getTime() - messageDate.getTime()) / 86400000,
  );

  if (diffDays === 0) {
    return "Today";
  } else if (diffDays === 1) {
    return "Yesterday";
  }
  return date.toLocaleDateString("en-US", {
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
  });
}

function shouldShowDateSeparator(index: number): boolean {
  const messages = contactsStore.messages;
  if (index === 0) return true;

  const currentDate = new Date(messages[index].created_at);
  const prevDate = new Date(messages[index - 1].created_at);

  return currentDate.toDateString() !== prevDate.toDateString();
}

function getMessageContentRaw(message: Message): string {
  if (message.message_type === "text") {
    if (typeof message.content === "string") {
      return message.content;
    }
    return message.content?.body || "";
  }
  if (message.message_type === "button_reply") {
    // Button reply stores the selected button title in content
    if (typeof message.content === "string") {
      return message.content;
    }
    return message.content?.body || "";
  }
  if (message.message_type === "interactive") {
    // Interactive messages store body text in content (string) or content.body or interactive_data.body
    if (typeof message.content === "string") {
      return message.content;
    }
    if (message.interactive_data?.body) {
      return message.interactive_data.body;
    }
    return message.content?.body || "[Interactive Message]";
  }
  // For media messages, return caption if available (media is displayed inline)
  if (
    message.message_type === "image" ||
    message.message_type === "video" ||
    message.message_type === "sticker"
  ) {
    if (typeof message.content === "string") {
      return message.content;
    }
    return message.content?.body || "";
  }
  if (message.message_type === "audio") {
    return ""; // Audio doesn't have captions
  }
  if (message.message_type === "document") {
    if (typeof message.content === "string") {
      return message.content;
    }
    return message.content?.body || "";
  }
  if (message.message_type === "template") {
    // Show actual content if available (campaign messages), otherwise fallback
    if (typeof message.content === "string") {
      return message.content;
    }
    return message.content?.body || "[Template Message]";
  }
  if (message.message_type === "location") {
    return ""; // Location is displayed as a map/card, not text
  }
  if (
    message.message_type === "contacts" ||
    message.message_type === "contact"
  ) {
    return ""; // Contacts are displayed as a card, not text
  }
  if (message.message_type === "unsupported") {
    return ""; // Displayed as a visual card, not text
  }
  return "[Message]";
}

function getMessageContent(message: Message): string {
  const rawContent = getMessageContentRaw(message);
  const normalizedContent = normalizeDeletedMessageText(rawContent);
  return applyMentionDisplayNames(normalizedContent);
}

interface LocationData {
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
}

interface ContactData {
  name: string;
  phones?: string[];
}

function getLocationData(message: Message): LocationData | null {
  if (message.message_type !== "location") return null;
  try {
    // Content is stored as JSON string in body
    const body = message.content?.body || message.content;
    if (typeof body === "string") {
      return JSON.parse(body);
    }
    return body as LocationData;
  } catch {
    return null;
  }
}

function getContactsData(message: Message): ContactData[] {
  if (message.message_type !== "contacts" && message.message_type !== "contact")
    return [];
  try {
    // Content is stored as JSON string in body
    const body = message.content?.body || message.content;
    if (typeof body === "string") {
      return JSON.parse(body);
    }
    return body as ContactData[];
  } catch {
    return [];
  }
}

function getGoogleMapsUrl(location: LocationData): string {
  return `https://www.google.com/maps?q=${location.latitude},${location.longitude}`;
}

function getInteractiveButtons(
  message: Message,
): Array<{ id: string; title: string }> {
  if (!message.interactive_data) {
    return [];
  }
  // Support both interactive and template messages with buttons
  if (
    message.message_type !== "interactive" &&
    message.message_type !== "template"
  ) {
    return [];
  }
  // Handle both "buttons" (<=3) and "rows" (>3 list format)
  const items =
    message.interactive_data.buttons || message.interactive_data.rows;
  if (!items || !Array.isArray(items)) {
    return [];
  }
  return items.map((btn: any) => ({
    id: btn.reply?.id || btn.id || "",
    title: btn.reply?.title || btn.title || btn.text || "",
  }));
}

interface CTAUrlData {
  type: "cta_url";
  body: string;
  button_text: string;
  url: string;
}

function sanitizeCTAUrl(raw: unknown): string {
  const candidate = typeof raw === "string" ? raw.trim() : "";
  if (!candidate) return "";
  try {
    const parsed = new URL(candidate);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return "";
    }
    return parsed.toString();
  } catch {
    return "";
  }
}

function getCTAUrlData(message: Message): CTAUrlData | null {
  if (message.message_type !== "interactive" || !message.interactive_data) {
    return null;
  }
  if (message.interactive_data.type !== "cta_url") {
    return null;
  }
  const safeURL = sanitizeCTAUrl((message.interactive_data as any).url);
  if (!safeURL) {
    return null;
  }
  return {
    type: "cta_url",
    body: message.interactive_data.body || "",
    button_text: (message.interactive_data as any).button_text || "Open",
    url: safeURL,
  };
}

function isMediaMessage(message: Message): boolean {
  return ["image", "video", "audio", "document", "sticker"].includes(
    message.message_type,
  );
}

function getMediaBlobUrl(message: Message): string {
  return mediaBlobUrls.value[message.id] || "";
}

function isMediaLoading(message: Message): boolean {
  return mediaLoadingStates.value[message.id] || false;
}

function canRetryMediaDownload(message: Message): boolean {
  if (!message.metadata) return false;
  const mediaID = (message.metadata as Record<string, unknown>)
    ?.legacy_media_recovery_media_id;
  return !!mediaID;
}

async function retryMediaDownload(message: Message) {
  mediaLoadingStates.value[message.id] = true;
  try {
    const resp = await contactsService.retryMediaDownload(message.id);
    if (resp.status !== "success") {
      toast.error(resp.message || t("common.mediaDownloadExpired"));
      return;
    }
    // Re-fetch the media blob now that the file is restored
    clearMissingMediaPrefetch(message.id);
    delete mediaBlobUrls.value[message.id];
    await loadMediaForMessage(message);
  } catch (error: any) {
    const msg = error?.response?.data?.message;
    toast.error(msg || t("common.mediaDownloadExpired"));
  } finally {
    mediaLoadingStates.value[message.id] = false;
  }
}

function getAttachmentFilename(message: Message): string {
  return resolveMediaFilename(message);
}

function isModifiedPointerEvent(event?: MouseEvent): boolean {
  if (!event) return false;
  return event.metaKey || event.ctrlKey || event.shiftKey || event.altKey;
}

function downloadAttachment(message: Message, event?: MouseEvent) {
  if (isBatchPrintSelectionMode.value) return;
  if (isModifiedPointerEvent(event)) return;
  const mediaUrl = getMediaBlobUrl(message);
  if (!mediaUrl) return;
  downloadMessageMedia(mediaUrl, message);
}

function printAttachment(message: Message, event?: MouseEvent) {
  if (isBatchPrintSelectionMode.value) return;
  if (isModifiedPointerEvent(event)) return;
  const mediaUrl = getMediaBlobUrl(message);
  if (!mediaUrl) return;
  if (!isMessagePrintSupported(message)) {
    toast.error(t("chat.printDialogFailed"), {
      description: t("chat.batchPrintUnsupportedDesc"),
    });
    return;
  }
  void (async () => {
    const opened = await openPrintDialogForSingleMessage({
      message,
      mediaUrl,
      resolveBlob: () => resolveMessageBlobForBatchPrint(message),
    });
    if (!opened) {
      toast.error(t("chat.printDialogFailed"));
    }
  })();
}

function resetMediaLoadingPipeline() {
  mediaLoadGeneration++;
  pendingMediaQueue.length = 0;
  queuedMediaMessageIDs.clear();
  for (const controller of inFlightMediaRequests.values()) {
    controller.abort();
  }
  inFlightMediaRequests.clear();
}

function enqueueMediaForBackgroundLoad(message: Message) {
  if (queuedMediaMessageIDs.has(message.id)) return;
  queuedMediaMessageIDs.add(message.id);
  pendingMediaQueue.push(message);
}

function pumpMediaLoadQueue() {
  while (
    activeMediaLoadCount < MAX_MEDIA_LOAD_CONCURRENCY &&
    pendingMediaQueue.length > 0
  ) {
    const message = pendingMediaQueue.shift();
    if (!message) continue;
    queuedMediaMessageIDs.delete(message.id);

    if (
      !message.media_url ||
      mediaBlobUrls.value[message.id] ||
      mediaLoadingStates.value[message.id]
    ) {
      continue;
    }

    const generationAtStart = mediaLoadGeneration;
    activeMediaLoadCount++;
    void loadMediaForMessage(message, generationAtStart).finally(() => {
      activeMediaLoadCount = Math.max(0, activeMediaLoadCount - 1);
      pumpMediaLoadQueue();
    });
  }
}

async function loadMediaForMessage(
  message: Message,
  generation: number = mediaLoadGeneration,
) {
  if (
    generation !== mediaLoadGeneration ||
    !message.media_url ||
    mediaLoadingStates.value[message.id]
  ) {
    return;
  }

  const cachedBlob = mediaBlobCache.get(message.id);
  if (cachedBlob) {
    if (!mediaBlobUrls.value[message.id]) {
      mediaBlobUrls.value[message.id] = URL.createObjectURL(cachedBlob);
    }
    return;
  }

  const persistentCachedBlob = await getCachedMediaBlob(message.id);
  if (persistentCachedBlob) {
    mediaBlobCache.set(message.id, persistentCachedBlob);
    if (generation !== mediaLoadGeneration) {
      return;
    }
    if (!mediaBlobUrls.value[message.id]) {
      mediaBlobUrls.value[message.id] =
        URL.createObjectURL(persistentCachedBlob);
    }
    return;
  }

  if (mediaBlobUrls.value[message.id]) {
    return;
  }

  const controller = new AbortController();
  inFlightMediaRequests.set(message.id, controller);
  mediaLoadingStates.value[message.id] = true;

  try {
    const blob = await prefetchMediaBlob(message.id, {
      signal: controller.signal,
    });
    if (!blob) {
      throw new Error("Failed to load media: empty response");
    }
    mediaBlobCache.set(message.id, blob);
    void storeMediaBlobInPersistentCache(message.id, blob);
    if (generation !== mediaLoadGeneration) {
      return;
    }
    if (mediaBlobUrls.value[message.id]) {
      URL.revokeObjectURL(mediaBlobUrls.value[message.id]);
    }
    const blobUrl = URL.createObjectURL(blob);
    mediaBlobUrls.value[message.id] = blobUrl;
  } catch (error: any) {
    if (error?.name === "AbortError") {
      return;
    }
    console.error("Failed to load media:", error, "message_id:", message.id);
  } finally {
    if (inFlightMediaRequests.get(message.id) === controller) {
      inFlightMediaRequests.delete(message.id);
    }
    mediaLoadingStates.value[message.id] = false;
  }
}

// Load media for all messages that have media_url
function loadMediaForMessages() {
  try {
    for (const message of contactsStore.messages) {
      if (message.media_url && !mediaBlobUrls.value[message.id]) {
        enqueueMediaForBackgroundLoad(message);
      }
    }
    pumpMediaLoadQueue();
  } catch (e) {
    console.error("Error in loadMediaForMessages:", e);
  }
}

function openMediaPreview(message: Message, event?: MouseEvent) {
  if (isBatchPrintSelectionMode.value) {
    handleMessageBubbleClickForBatchPrint(message, event);
    return;
  }
  if (isModifiedPointerEvent(event)) return;
  const url = getMediaBlobUrl(message);
  if (!url) return;
  openChatMediaViewer(
    url,
    message.message_type === "video" ? "video" : "image",
    getAttachmentFilename(message),
  );
}

function handleImageError(event: Event) {
  const img = event.target as HTMLImageElement;
  img.style.display = "none";
}

function handleMediaError(event: Event, mediaType: string) {
  console.error(`Failed to load ${mediaType}:`, event);
}

// File upload functions
function openFilePicker() {
  fileInputRef.value?.click();
}

function resetBatchPrintSelection() {
  isBatchPrintSelectionMode.value = false;
  selectedBatchPrintMessageIds.value = [];
}

function cancelBatchPrintSelection() {
  resetBatchPrintSelection();
}

function isBatchPrintBubbleSelectable(message: Message): boolean {
  return isMergePrintableBubbleMessage(message);
}

function isBatchPrintBubbleSelected(messageId: string): boolean {
  return selectedBatchPrintMessageIds.value.includes(messageId);
}

function toggleBatchPrintMessageSelection(messageId: string) {
  if (isBatchPrintBubbleSelected(messageId)) {
    selectedBatchPrintMessageIds.value =
      selectedBatchPrintMessageIds.value.filter((id) => id !== messageId);
    return;
  }
  selectedBatchPrintMessageIds.value = [
    ...selectedBatchPrintMessageIds.value,
    messageId,
  ];
}

function handleMessageBubbleClickForBatchPrint(
  message: Message,
  event?: MouseEvent,
) {
  if (!isBatchPrintSelectionMode.value) return;
  if (!isBatchPrintBubbleSelectable(message)) return;
  if (isModifiedPointerEvent(event)) return;
  event?.preventDefault();
  event?.stopPropagation();
  toggleBatchPrintMessageSelection(message.id);
}

async function resolveMessageBlobForBatchPrint(
  message: Message,
): Promise<Blob> {
  const cachedBlob = mediaBlobCache.get(message.id);
  if (cachedBlob) {
    return cachedBlob;
  }

  const persistentCachedBlob = await getCachedMediaBlob(message.id);
  if (persistentCachedBlob) {
    mediaBlobCache.set(message.id, persistentCachedBlob);
    if (!mediaBlobUrls.value[message.id]) {
      mediaBlobUrls.value[message.id] =
        URL.createObjectURL(persistentCachedBlob);
    }
    return persistentCachedBlob;
  }

  await loadMediaForMessage(message);
  const loadedBlob = mediaBlobCache.get(message.id);
  if (loadedBlob) {
    return loadedBlob;
  }

  const fallbackBlob = await prefetchMediaBlob(message.id);
  if (!fallbackBlob) {
    throw new Error("Failed to load media: empty response");
  }
  mediaBlobCache.set(message.id, fallbackBlob);
  void storeMediaBlobInPersistentCache(message.id, fallbackBlob);
  if (!mediaBlobUrls.value[message.id]) {
    mediaBlobUrls.value[message.id] = URL.createObjectURL(fallbackBlob);
  }
  return fallbackBlob;
}

async function mergeSelectedMessageBubblesAndPrint() {
  if (!canMergeSelectedBubbleFiles.value) {
    toast.error(t("chat.batchPrintMinSelection"), {
      description: t("chat.batchPrintMinSelectionDesc"),
    });
    return;
  }

  const selectedMessageIDs = new Set(selectedBatchPrintMessageIds.value);
  const selectedMessages = contactsStore.messages.filter(
    (message) =>
      selectedMessageIDs.has(message.id) &&
      isBatchPrintBubbleSelectable(message),
  );

  if (selectedMessages.length < 2) {
    toast.error(t("chat.batchPrintMinSelection"), {
      description: t("chat.batchPrintMinSelectionDesc"),
    });
    return;
  }

  isPreparingBatchPrint.value = true;
  toast.info(t("chat.batchPrintPreparing"));
  try {
    const files: File[] = [];
    for (const message of selectedMessages) {
      const blob = await resolveMessageBlobForBatchPrint(message);
      const file = toMergePrintableFile(message, blob, files.length);
      if (!file) {
        throw new Error(`Unsupported selected message: ${message.id}`);
      }
      files.push(file);
    }

    const opened = await mergePhotosAndPdfsAndOpenPrintDialog(files);
    if (!opened) {
      toast.error(t("chat.printDialogFailed"));
      return;
    }
    resetBatchPrintSelection();
  } catch (error) {
    console.error("Failed to merge selected bubbles for printing:", error);
    const errorDescription =
      error instanceof Error && error.message
        ? error.message
        : t("chat.batchPrintFailedDesc");
    toast.error(t("chat.batchPrintFailed"), {
      description: errorDescription,
    });
  } finally {
    isPreparingBatchPrint.value = false;
  }
}

function openBatchPrintPicker() {
  if (isPreparingBatchPrint.value) return;

  if (!isBatchPrintSelectionMode.value) {
    if (!hasMergePrintableBubbles.value) {
      toast.error(t("chat.batchPrintNoBubbleFiles"), {
        description: t("chat.batchPrintNoBubbleFilesDesc"),
      });
      return;
    }
    resetBatchPrintSelection();
    isBatchPrintSelectionMode.value = true;
    toast.info(t("chat.batchPrintSelectionMode"), {
      description: t("chat.batchPrintSelectionModeDesc"),
    });
    return;
  }

  void mergeSelectedMessageBubblesAndPrint();
}

function getMediaSizeErrorKey(category: WhatsAppMediaCategory) {
  if (category === "image") {
    return "chat.fileTooLargeImageDesc";
  }
  if (category === "video") {
    return "chat.fileTooLargeVideoDesc";
  }
  if (category === "audio") {
    return "chat.fileTooLargeAudioDesc";
  }
  return "chat.fileTooLargeDocumentDesc";
}

function buildPendingMediaUpload(
  file: File,
  index: number,
): PendingMediaUpload {
  const category = resolveWhatsAppMediaCategoryForFile(file);
  const shouldPreview = category === "image" || category === "video";

  return {
    id: `${file.name}-${file.size}-${file.lastModified}-${index}`,
    file,
    category,
    previewUrl: shouldPreview ? URL.createObjectURL(file) : null,
  };
}

function revokePendingMediaUpload(upload: PendingMediaUpload) {
  if (upload.previewUrl) {
    URL.revokeObjectURL(upload.previewUrl);
  }
}

function formatMediaUploadSize(sizeBytes: number) {
  const kilobyte = 1024;
  const megabyte = kilobyte * 1024;
  if (sizeBytes >= megabyte) {
    return `${(sizeBytes / megabyte).toFixed(1)} MB`;
  }
  if (sizeBytes >= kilobyte) {
    return `${(sizeBytes / kilobyte).toFixed(1)} KB`;
  }
  return `${sizeBytes} B`;
}

function setActiveMediaPreview(uploadID: string) {
  activeMediaPreviewID.value = uploadID;
}

function removeSelectedMediaUpload(uploadID: string) {
  const removedUpload = selectedMediaUploads.value.find(
    (upload) => upload.id === uploadID,
  );
  if (!removedUpload) return;

  revokePendingMediaUpload(removedUpload);

  const remainingUploads = selectedMediaUploads.value.filter(
    (upload) => upload.id !== uploadID,
  );
  selectedMediaUploads.value = remainingUploads;

  if (remainingUploads.length === 0) {
    closeMediaDialog();
    return;
  }

  if (activeMediaPreviewID.value === uploadID) {
    activeMediaPreviewID.value = remainingUploads[0].id;
  }

  if (remainingUploads.length > 1) {
    mediaCaption.value = "";
  }
}

function handleMediaDialogOpenChange(open: boolean) {
  if (open) {
    isMediaDialogOpen.value = true;
    return;
  }

  if (isUploadingMedia.value) {
    isMediaDialogOpen.value = true;
    return;
  }

  closeMediaDialog();
}

function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  if (files.length === 0) return;

  const acceptedUploads: PendingMediaUpload[] = [];

  files.forEach((file, index) => {
    const validation = validateWhatsAppMediaFile(file);
    if (!validation.isValid) {
      toast.error(t("chat.fileTooLarge"), {
        description: `${file.name}: ${t(getMediaSizeErrorKey(validation.category))}`,
      });
      return;
    }

    acceptedUploads.push(buildPendingMediaUpload(file, index));
  });

  input.value = "";

  if (acceptedUploads.length === 0) {
    return;
  }

  selectedMediaUploads.value.forEach(revokePendingMediaUpload);
  selectedMediaUploads.value = acceptedUploads;
  activeMediaPreviewID.value = acceptedUploads[0]?.id ?? null;
  mediaCaption.value = "";
  isMediaDialogOpen.value = true;
}

function closeMediaDialog() {
  selectedMediaUploads.value.forEach(revokePendingMediaUpload);
  selectedMediaUploads.value = [];
  activeMediaPreviewID.value = null;
  mediaUploadProgress.value = null;
  mediaCaption.value = "";
  isMediaDialogOpen.value = false;
}

async function sendMediaMessage() {
  if (isCurrentChatSendRestricted.value || isCurrentChatClosed.value) return;
  if (
    selectedMediaUploads.value.length === 0 ||
    !contactsStore.currentContact
  ) {
    return;
  }

  const uploads = [...selectedMediaUploads.value];
  const outboundInstanceID = resolveOutboundInstanceID(
    contactsStore.currentContact,
  );
  const accountFilter = resolveOutboundWhatsAppAccount(
    contactsStore.currentContact,
  );
  const shouldApplyCaption =
    uploads.length === 1 && uploads[0].category !== "audio";
  const caption = shouldApplyCaption ? mediaCaption.value : "";
  const sentMessages: Message[] = [];
  const successfulUploadIDs = new Set<string>();
  let firstError: unknown = null;

  isUploadingMedia.value = true;
  try {
    for (const [index, upload] of uploads.entries()) {
      mediaUploadProgress.value = {
        current: index + 1,
        total: uploads.length,
      };

      try {
        const response = await messagesService.sendMedia({
          contactId: contactsStore.currentContact.id,
          file: upload.file,
          type: upload.category,
          caption,
          instance_id: outboundInstanceID,
          whatsapp_account: accountFilter,
        });
        const result = response.data.data || response.data;
        successfulUploadIDs.add(upload.id);
        if (result) {
          sentMessages.push(result);
        }
      } catch (error) {
        if (!firstError) {
          firstError = error;
        }
      }
    }

    sentMessages.forEach((message) => contactsStore.addMessage(message));

    if (sentMessages.length > 0) {
      scrollToBottom();
      await nextTick();
      sentMessages.forEach((message) => {
        if (message.media_url) {
          loadMediaForMessage(message);
        }
      });
    }

    if (successfulUploadIDs.size === uploads.length) {
      toast.success(
        uploads.length > 1
          ? t("chat.mediaBatchSent", { count: uploads.length })
          : t("chat.mediaSent"),
      );
      closeMediaDialog();
      return;
    }

    if (successfulUploadIDs.size === 0) {
      throw firstError;
    }

    const failedUploads = uploads.filter(
      (upload) => !successfulUploadIDs.has(upload.id),
    );

    uploads
      .filter((upload) => successfulUploadIDs.has(upload.id))
      .forEach(revokePendingMediaUpload);

    selectedMediaUploads.value = failedUploads;
    activeMediaPreviewID.value = failedUploads[0]?.id ?? null;
    mediaCaption.value = "";

    toast.warning(t("chat.mediaBatchPartialFailed"), {
      description: t("chat.mediaBatchPartialFailedDesc", {
        sent: successfulUploadIDs.size,
        failed: failedUploads.length,
      }),
    });
  } catch (error: any) {
    toast.error(t("chat.mediaFailed"), {
      description: getErrorMessage(error, t("chat.mediaFailedDesc")),
    });
  } finally {
    mediaUploadProgress.value = null;
    isUploadingMedia.value = false;
  }
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
              'group flex cursor-pointer items-center gap-2 px-3 py-2 transition-all duration-150 hover:bg-sidebar-accent/80',
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

          <EmptyState
            v-if="sidebarContacts.length === 0"
            :icon="User"
            size="compact"
            variant="primary"
            animated
            :title="contactsStore.activeChatTab === 'pending'
              ? $t('chat.noPendingChats')
              : $t('chat.noAssignedChats')"
            :description="contactsStore.activeChatTab === 'pending'
              ? $t('chat.noPendingChatsDesc')
              : $t('chat.noAssignedChatsDesc')"
          />
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
      <div
        v-if="!contactsStore.currentContact"
        class="flex flex-1 items-center justify-center text-muted-foreground"
      >
        <EmptyState
          :icon="Send"
          size="hero"
          variant="primary"
          animated
          :title="$t('chat.selectConversation')"
          :description="$t('chat.chooseContact')"
          class="flex-1 items-center justify-center"
        >
          <template #action>
            <p class="text-xs text-muted-foreground/60">
              {{ $t("chat.selectConversationHint") }}
            </p>
          </template>
        </EmptyState>
      </div>

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
                  class="h-5 border-0 bg-primary/10 text-[10px] text-primary"
                >
                  {{ $t("chat.pendingStatus") }}
                </Badge>
                <Badge
                  v-if="contactsStore.currentContact.status === 'closed'"
                  class="h-5 border-0 bg-muted text-[10px] text-muted-foreground"
                >
                  {{ $t("chat.closedStatus") }}
                </Badge>
                <Badge
                  v-if="activeTransferId"
                  class="h-5 border-0 bg-accent text-[10px] text-accent-foreground"
                >
                  {{ $t("chat.pausedStatus") }}
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
                        class="w-[200px] h-[150px] bg-muted rounded-lg flex flex-col items-center justify-center gap-1"
                      >
                        <ImageIcon class="h-5 w-5 text-muted-foreground" />
                        <span
                          class="text-sm text-muted-foreground cursor-pointer hover:text-foreground underline decoration-dotted"
                          @click.stop="retryMediaDownload(message)"
                        >{{ $t("common.mediaExpired") }}</span>
                        <Button
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="retryMediaDownload(message)"
                        >
                          <RefreshCw class="h-3.5 w-3.5 mr-1" />
                          {{ $t("common.retryDownload") }}
                        </Button>
                      </div>
                      <div
                        v-if="
                          getMediaBlobUrl(message) &&
                          (configStore.showPrintButtons ||
                            configStore.showDownloadButtons)
                        "
                        class="mt-1 flex flex-wrap items-center gap-1.5"
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
                        class="w-[128px] h-[128px] bg-muted rounded-lg flex flex-col items-center justify-center gap-1"
                      >
                        <ImageIcon class="h-5 w-5 text-muted-foreground" />
                        <span
                          class="text-xs text-muted-foreground cursor-pointer hover:text-foreground underline decoration-dotted text-center"
                          @click.stop="retryMediaDownload(message)"
                        >{{ $t("common.mediaExpired") }}</span>
                        <Button
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="retryMediaDownload(message)"
                        >
                          <RefreshCw class="h-3.5 w-3.5 mr-1" />
                          {{ $t("common.retryDownload") }}
                        </Button>
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
                        class="w-[200px] h-[150px] bg-muted rounded-lg flex flex-col items-center justify-center gap-1"
                      >
                        <Video class="h-5 w-5 text-muted-foreground" />
                        <span
                          class="text-sm text-muted-foreground cursor-pointer hover:text-foreground underline decoration-dotted"
                          @click.stop="retryMediaDownload(message)"
                        >{{ $t("common.mediaExpired") }}</span>
                        <Button
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="retryMediaDownload(message)"
                        >
                          <RefreshCw class="h-3.5 w-3.5 mr-1" />
                          {{ $t("common.retryDownload") }}
                        </Button>
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
                      <div v-else class="flex items-center gap-2 flex-wrap">
                        <Music class="h-4 w-4 text-muted-foreground" />
                        <span
                          class="text-sm text-muted-foreground cursor-pointer hover:text-foreground underline decoration-dotted"
                          @click.stop="retryMediaDownload(message)"
                        >{{ $t("common.mediaExpired") }}</span>
                        <Button
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="retryMediaDownload(message)"
                        >
                          <RefreshCw class="h-3.5 w-3.5 mr-1" />
                          {{ $t("common.retryDownload") }}
                        </Button>
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
                        <span
                          class="text-sm text-muted-foreground cursor-pointer hover:text-foreground underline decoration-dotted"
                          @click.stop="retryMediaDownload(message)"
                        >{{
                          $t("common.mediaExpired")
                        }}</span>
                        <Button
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="retryMediaDownload(message)"
                        >
                          <RefreshCw class="h-3.5 w-3.5 mr-1" />
                          {{ $t("common.retryDownload") }}
                        </Button>
                      </div>
                    </div>
                    <!-- Deleted/expired media fallback (media type but no media_url) -->
                    <div
                      v-else-if="isMediaMessage(message) && !message.media_url"
                      class="mb-2"
                    >
                      <div
                        class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg"
                      >
                        <component
                          :is="
                            message.message_type === 'image' || message.message_type === 'sticker'
                              ? ImageIcon
                              : message.message_type === 'video'
                                ? Video
                                : message.message_type === 'audio'
                                  ? Music
                                  : FileText
                          "
                          class="h-5 w-5 text-muted-foreground"
                        />
                        <span
                          class="text-sm text-muted-foreground cursor-pointer hover:text-foreground underline decoration-dotted"
                          @click.stop="retryMediaDownload(message)"
                        >{{ $t("common.mediaExpired") }}</span>
                        <Button
                          variant="ghost"
                          size="xs"
                          class="h-7 px-2 text-[11px]"
                          @click.stop="retryMediaDownload(message)"
                        >
                          <RefreshCw class="h-3.5 w-3.5 mr-1" />
                          {{ $t("common.retryDownload") }}
                        </Button>
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
                        message.error_message || $t("chat.failedToSend")
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
                        message.error_message || $t("chat.failedToSend")
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
                      :title="$t('chat.retrySending')"
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
              <EmptyState
                v-if="!contactsStore.isLoadingMessages && contactsStore.messages.length === 0"
                :icon="Send"
                size="compact"
                variant="primary"
                animated
                :title="$t('chat.noMessagesYet')"
                :description="$t('chat.noMessagesHint')"
              />
              <div ref="messagesEndRef" />
            </div>
          </ScrollArea>
        </div>

        <div
          v-if="isCurrentChatClosed && !isCurrentChatRestricted"
          class="flex items-center justify-between gap-3 border-t border-border bg-muted/55 px-4 py-2 text-xs text-muted-foreground"
        >
          <span>{{ $t("chat.chatClosedReadonly") }}</span>
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
            {{ $t("chat.reopenChat") }}
          </Button>
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
              class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-all duration-200 hover:bg-primary/90 active:scale-[0.93] disabled:opacity-50"
              :disabled="
                isCurrentChatSendRestricted || !canSendMessage || isSending
              "
            >
              <transition
                enter-active-class="transition-all duration-150"
                enter-from-class="scale-0 opacity-0"
                enter-to-class="scale-100 opacity-100"
                leave-active-class="transition-all duration-100"
                leave-from-class="scale-100 opacity-100"
                leave-to-class="scale-0 opacity-0"
                mode="out-in"
              >
                <Check v-if="showSendConfirm" key="check" class="w-4 h-4" />
                <Send v-else key="send" class="w-4 h-4" />
              </transition>
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
    <Dialog
      v-model:open="isAssignDialogOpen"
      @update:open="(open) => { if (!open) { assignSearchQuery = ''; debouncedAssignSearchQuery = ''; } }"
    >
      <DialogContent class="w-[calc(100vw-2rem)] max-w-md sm:max-w-lg" resizable>
        <DialogHeader>
          <DialogTitle>{{ $t("chat.assignContact") }}</DialogTitle>
          <DialogDescription>
            {{ $t("chat.assignContactDesc") }}
          </DialogDescription>
        </DialogHeader>
        <div class="pt-1 pb-2 space-y-2">
          <!-- Search input -->
          <div class="relative">
            <Search
              class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground"
            />
            <Input
              v-model="assignSearchQuery"
              :placeholder="$t('chat.searchUsers') + '...'"
              class="pl-9 h-9 bg-muted/50"
              :aria-label="$t('chat.searchUsers')"
            />
          </div>
          <div
            v-if="contactsStore.currentContact?.assigned_user_id"
            class="space-y-2"
          >
            <Button
              variant="outline"
              class="w-full justify-start text-destructive hover:text-destructive hover:bg-destructive/10"
              :disabled="isAssigning"
              @click="
                assignContactToUser(null);
                isAssignDialogOpen = false;
              "
            >
              <UserMinus class="mr-2 h-4 w-4" />
              {{ $t("chat.unassignContact") }}
            </Button>
            <Separator />
          </div>
          <p class="text-xs text-muted-foreground font-medium px-1 mt-2">
            {{ $t('chat.usersAvailable', { count: filteredAssignableUsers.length }) }}
          </p>
          <ScrollArea class="max-h-[420px]">
            <div class="space-y-0.5" role="listbox" :aria-label="$t('chat.assignContact')">
              <Button
                v-for="user in filteredAssignableUsers"
                :key="user.id"
                :variant="
                  contactsStore.currentContact?.assigned_user_id === user.id
                    ? 'secondary'
                    : 'ghost'
                "
                class="w-full justify-start h-auto py-2.5 px-3 transition-colors"
                :class="
                  contactsStore.currentContact?.assigned_user_id === user.id
                    ? 'bg-primary/10 border border-primary/20'
                    : 'hover:bg-muted'
                "
                role="option"
                :aria-selected="contactsStore.currentContact?.assigned_user_id === user.id"
                :disabled="isAssigning"
                @click="
                  assignContactToUser(user.id);
                  isAssignDialogOpen = false;
                "
              >
                <div class="flex items-center w-full gap-3">
                  <div
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/8 text-xs font-medium text-primary"
                    aria-hidden="true"
                  >
                    {{ user.full_name?.charAt(0)?.toUpperCase() || '?' }}
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium truncate">{{ user.full_name }}</div>
                    <div class="text-xs text-muted-foreground truncate">{{ user.email }}</div>
                  </div>
                  <Check
                    v-if="
                      contactsStore.currentContact?.assigned_user_id === user.id
                    "
                    class="h-4 w-4 text-primary shrink-0"
                  />
                  <Badge v-else variant="outline" class="text-xs shrink-0">
                    {{ user.role?.name || $t("chat.noRole") }}
                  </Badge>
                </div>
              </Button>
              <p
                v-if="filteredAssignableUsers.length === 0"
                class="text-sm text-muted-foreground text-center py-8"
              >
                {{ $t("chat.noUsersFound") }}
              </p>
            </div>
          </ScrollArea>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Chat Media Viewer Dialog -->
    <Dialog
      v-model:open="isChatMediaViewerOpen"
      @update:open="(open) => !open && closeChatMediaViewer()"
    >
      <DialogContent
        class="max-w-4xl p-0 overflow-hidden border-white/10 light:border-gray-200"
      >
        <div
          class="bg-black/95 light:bg-black/90 p-3 space-y-3"
          data-testid="chat-media-viewer-dialog"
        >
          <div class="flex items-center justify-between gap-3">
            <DialogTitle class="text-sm font-medium text-white truncate">
              {{ chatMediaViewerTitle || t("chat.mediaPreview") }}
            </DialogTitle>
            <Button
              variant="ghost"
              size="icon"
              class="h-7 w-7 text-white hover:text-white"
              @click="closeChatMediaViewer"
            >
              <X class="h-4 w-4" />
            </Button>
          </div>

          <div
            class="flex items-center justify-center min-h-[220px] max-h-[80vh]"
          >
            <img
              v-if="chatMediaViewerType === 'image' && chatMediaViewerURL"
              :src="chatMediaViewerURL"
              alt="Media preview"
              class="max-w-full max-h-[76vh] rounded-md object-contain"
              data-testid="chat-media-viewer-image"
            />
            <video
              v-else-if="chatMediaViewerType === 'video' && chatMediaViewerURL"
              :src="chatMediaViewerURL"
              controls
              autoplay
              class="max-w-full max-h-[76vh] rounded-md"
              data-testid="chat-media-viewer-video"
            />
            <audio
              v-else-if="chatMediaViewerType === 'audio' && chatMediaViewerURL"
              :src="chatMediaViewerURL"
              controls
              class="w-full max-w-md"
              data-testid="chat-media-viewer-audio"
            />
            <a
              v-else-if="chatMediaViewerURL"
              :href="chatMediaViewerURL"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm text-white underline underline-offset-4"
            >
              {{ $t("chat.openMediaNewTab") }}
            </a>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Media Preview Dialog -->
    <Dialog
      v-model:open="isMediaDialogOpen"
      @update:open="handleMediaDialogOpenChange"
    >
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t("chat.sendMedia") }}</DialogTitle>
          <DialogDescription>
            {{ mediaDialogDescription }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-4">
          <div
            v-if="
              activeMediaUpload?.category === 'image' &&
              activeMediaUpload.previewUrl
            "
            class="flex justify-center"
          >
            <img
              :src="activeMediaUpload.previewUrl"
              :alt="activeMediaUpload.file.name"
              class="max-w-full max-h-[300px] rounded-lg object-contain"
            />
          </div>
          <div
            v-else-if="
              activeMediaUpload?.category === 'video' &&
              activeMediaUpload.previewUrl
            "
            class="flex justify-center"
          >
            <video
              :src="activeMediaUpload.previewUrl"
              controls
              class="max-w-full max-h-[300px] rounded-lg"
            />
          </div>
          <div
            v-else-if="activeMediaUpload?.category === 'audio'"
            class="flex justify-center"
          >
            <div class="flex items-center gap-3 px-4 py-3 bg-muted rounded-lg">
              <div
                class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center"
              >
                <Paperclip class="h-5 w-5 text-primary" />
              </div>
              <div>
                <p class="font-medium text-sm">
                  {{ activeMediaUpload.file.name }}
                </p>
                <p class="text-xs text-muted-foreground">
                  {{ $t("chat.audioFile") }}
                </p>
              </div>
            </div>
          </div>
          <div v-else-if="activeMediaUpload" class="flex justify-center">
            <div class="flex items-center gap-3 px-4 py-3 bg-muted rounded-lg">
              <div
                class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center"
              >
                <FileText class="h-5 w-5 text-primary" />
              </div>
              <div>
                <p class="font-medium text-sm truncate max-w-[200px]">
                  {{ activeMediaUpload.file.name }}
                </p>
                <p class="text-xs text-muted-foreground">
                  {{ formatMediaUploadSize(activeMediaUpload.file.size) }}
                </p>
              </div>
            </div>
          </div>

          <div v-if="selectedMediaCount > 1" class="space-y-2">
            <ScrollArea class="max-h-[220px] pr-3">
              <div class="space-y-2">
                <div
                  v-for="upload in selectedMediaUploads"
                  :key="upload.id"
                  class="flex items-center gap-2"
                >
                  <button
                    type="button"
                    class="flex flex-1 items-center gap-3 rounded-lg border px-3 py-2 text-left transition-colors"
                    :class="
                      activeMediaUpload?.id === upload.id
                        ? 'border-primary/45 bg-primary/10'
                        : 'border-border bg-muted/30 hover:bg-muted/60'
                    "
                    :disabled="isUploadingMedia"
                    @click="setActiveMediaPreview(upload.id)"
                  >
                    <div
                      class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10"
                    >
                      <ImageIcon
                        v-if="upload.category === 'image'"
                        class="h-5 w-5 text-primary"
                      />
                      <Play
                        v-else-if="upload.category === 'video'"
                        class="h-5 w-5 text-primary"
                      />
                      <Paperclip
                        v-else-if="upload.category === 'audio'"
                        class="h-5 w-5 text-primary"
                      />
                      <FileText v-else class="h-5 w-5 text-primary" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-sm font-medium">
                        {{ upload.file.name }}
                      </p>
                      <p class="text-xs text-muted-foreground">
                        {{ $t(`chat.${upload.category}`) }} ·
                        {{ formatMediaUploadSize(upload.file.size) }}
                      </p>
                    </div>
                  </button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    class="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
                    :disabled="isUploadingMedia"
                    :aria-label="`${$t('common.remove')} ${upload.file.name}`"
                    @click="removeSelectedMediaUpload(upload.id)"
                  >
                    <X class="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </ScrollArea>
            <p class="text-sm text-muted-foreground">
              {{ $t("chat.mediaBatchCaptionHint") }}
            </p>
          </div>

          <div v-else-if="canApplyMediaCaption">
            <Textarea
              v-model="mediaCaption"
              :placeholder="$t('chat.mediaCaption') + '...'"
              class="min-h-[60px] max-h-[100px] resize-none"
              :rows="2"
            />
          </div>

          <div class="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              @click="closeMediaDialog"
              :disabled="isUploadingMedia"
            >
              {{ $t("common.cancel") }}
            </Button>
            <Button
              type="button"
              @click="sendMediaMessage"
              :disabled="isUploadingMedia || selectedMediaCount === 0"
            >
              <Send v-if="!isUploadingMedia" class="mr-2 h-4 w-4" />
              <span v-if="isUploadingMedia">{{ mediaUploadingLabel }}</span>
              <span v-else>{{ mediaSendButtonLabel }}</span>
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Contact Profile Photo Dialog -->
    <Dialog
      v-model:open="isProfilePhotoDialogOpen"
      @update:open="handleProfilePhotoDialogOpenChange"
    >
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t("resources.ProfilePhoto") }}</DialogTitle>
          <DialogDescription>
            {{
              profilePhotoContact?.name ||
              profilePhotoContact?.phone_number ||
              $t("chat.customer")
            }}
          </DialogDescription>
        </DialogHeader>
        <div class="flex items-center justify-center py-2">
          <img
            v-if="activeProfilePhotoURL !== '' && !profilePhotoImageFailed"
            :src="activeProfilePhotoURL"
            :alt="
              profilePhotoContact?.name ||
              profilePhotoContact?.phone_number ||
              $t('resources.ProfilePhoto')
            "
            class="max-h-[70vh] max-w-full rounded-lg object-contain"
            @error="handleProfilePhotoImageError"
          />
          <div
            v-else
            class="flex h-48 w-48 items-center justify-center rounded-full bg-gradient-to-br from-sky-500 to-blue-600 text-4xl font-semibold text-white"
          >
            {{
              getInitials(
                profilePhotoContact?.name ||
                  profilePhotoContact?.phone_number ||
                  $t("chat.customer"),
              )
            }}
          </div>
        </div>
      </DialogContent>
    </Dialog>

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
