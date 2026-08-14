<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick, computed, defineAsyncComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useContactsStore, type Contact } from '@/stores/contacts'
import { useAuthStore } from '@/stores/auth'
import { useUsersStore } from '@/stores/users'
import { wsService } from '@/services/websocket'
import { useTagsStore } from '@/stores/tags'
import { TagBadge } from '@/components/ui/tag-badge'
import { getTagColorClass } from '@/lib/constants'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Spinner } from '@/components/ui/spinner'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
// Lazy-load emoji picker to reduce initial bundle size
const EmojiPicker = defineAsyncComponent(() => {
  return import('vue3-emoji-picker').then(module => {
    // Import CSS when component loads
    import('vue3-emoji-picker/css')
    return module.default
  })
})
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Search,
  Send,
  Paperclip,
  FileText,
  ImageIcon,
  Video,
  Smile,
  MoreVertical,
  Check,
  CheckCheck,
  AlertCircle,
  Ban,
  User,
  UserPlus,
  UserMinus,
  Reply,
  X,
  SmilePlus,
  MapPin,
  ExternalLink,
  Loader2,
  RotateCw,
  Trash2,
  Filter,
  StickyNote,
  CalendarClock,
  Lock,
  Hand,
  Users,
  Info,
  LogOut,
  Ghost,
  Megaphone,
  RotateCcw
} from 'lucide-vue-next'
import { getInitials, getAvatarGradient, avatarSrc, linkifySegments } from '@/lib/utils'
import { useColorMode } from '@/composables/useColorMode'
import CannedResponsePicker from '@/components/chat/CannedResponsePicker.vue'
import TemplatePicker from '@/components/chat/TemplatePicker.vue'
import ContactInfoPanel from '@/components/chat/ContactInfoPanel.vue'
import ConversationNotes from '@/components/chat/ConversationNotes.vue'
import { useNotesStore } from '@/stores/notes'
import ScheduleMessageDialog from '@/components/chat/ScheduleMessageDialog.vue'
import ScheduledMessagesPanel from '@/components/chat/ScheduledMessagesPanel.vue'
import { useScheduledMessagesStore } from '@/stores/scheduledMessages'
import { CreateContactDialog, ConfirmDialog } from '@/components/shared'
import HeaderMediaUpload from '@/components/shared/HeaderMediaUpload.vue'
import { Download } from 'lucide-vue-next'
import MediaBurstDialog from '@/components/chat/MediaBurstDialog.vue'
import MediaRetryButton from '@/components/chat/MediaRetryButton.vue'
import { useMediaBurst } from '@/composables/useMediaBurst'
import { useMediaExport } from '@/composables/useMediaExport'
import { isStatusContact } from '@/lib/status'
// Extracted chat composables (keep ChatView a thin orchestration shell)
import { useMessageFormat } from '@/composables/useMessageFormat'
import { useChatScroll } from '@/composables/useChatScroll'
import { useChatTyping } from '@/composables/useChatTyping'
import { useChatMedia } from '@/composables/useChatMedia'
import { useChatCannedTemplates } from '@/composables/useChatCannedTemplates'
import { useChatMessaging } from '@/composables/useChatMessaging'
import { useChatLifecycle } from '@/composables/useChatLifecycle'
import { useChatContactsList } from '@/composables/useChatContactsList'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const contactsStore = useContactsStore()
const authStore = useAuthStore()
const usersStore = useUsersStore()
const tagsStore = useTagsStore()
const notesStore = useNotesStore()
const scheduledStore = useScheduledMessagesStore()
const { isDark } = useColorMode()

const canWriteContacts = authStore.hasPermission('contacts', 'write')
const canExportMedia = authStore.hasPermission('contacts', 'export')
const canRevokeMessages = authStore.hasPermission('chat.revoke', 'write')

const contactId = computed(() => route.params.contactId as string | undefined)

// ─── Template refs bound to DOM nodes (view-owned, passed into composables) ───
// Declared in the view so vue-tsc reliably tracks their template usage; the
// composables that need them receive them as params.
const messagesEndRef = ref<HTMLElement | null>(null)
const messageInputRef = ref<HTMLTextAreaElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const tabStripRef = ref<HTMLElement | null>(null)

// ─── Resizable contacts sidebar (pure view state) ───
const SIDEBAR_MIN_WIDTH = 260
const SIDEBAR_MAX_WIDTH = 520
const SIDEBAR_DEFAULT_WIDTH = 320 // matches the previous fixed `w-80`
// Reserve this much room for the chat panel (header + composer) so the sidebar
// can never starve the conversation area on narrow viewports.
const SIDEBAR_MIN_CHAT_ROOM = 360
const sidebarWidth = ref(SIDEBAR_DEFAULT_WIDTH)
const isResizingSidebar = ref(false)

// Effective max sidebar width for the current viewport: never wider than
// SIDEBAR_MAX_WIDTH, and never so wide that the chat panel is starved below
// SIDEBAR_MIN_CHAT_ROOM. This is what prevents the header controls from being
// pushed off-screen on narrow windows.
function sidebarEffectiveMax(): number {
  if (typeof window === 'undefined') return SIDEBAR_MAX_WIDTH
  // The chat view lives in <main>, which sits beside the app nav rail
  // (~64px collapsed / ~256px expanded). Subtract a conservative 80px for the
  // nav so the clamp holds on collapsed layouts too.
  const mainWidth = Math.max(0, window.innerWidth - 80)
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, mainWidth - SIDEBAR_MIN_CHAT_ROOM))
}

function clampSidebarToViewport() {
  const max = sidebarEffectiveMax()
  if (sidebarWidth.value > max) sidebarWidth.value = max
}

function startSidebarResize(e: MouseEvent) {
  isResizingSidebar.value = true
  const startX = e.clientX
  const startWidth = sidebarWidth.value

  function onMouseMove(e: MouseEvent) {
    // Sidebar is on the left: dragging the right edge rightward widens it.
    // Clamp against the viewport-aware max so the chat panel keeps room.
    const delta = e.clientX - startX
    const max = sidebarEffectiveMax()
    const newWidth = Math.min(max, Math.max(SIDEBAR_MIN_WIDTH, startWidth + delta))
    sidebarWidth.value = newWidth
  }

  function onMouseUp() {
    isResizingSidebar.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// ─── Right-panel toggles (pure view state) ───
const isInfoPanelOpen = ref(false)
const isNotesPanelOpen = ref(false)
const isScheduledPanelOpen = ref(false)
const isScheduleDialogOpen = ref(false)

// ─── Media burst: detect a flurry of incoming files and offer to collect them ───
const burstTimeMs = ref(1_800_000) // 30 minutes default, reactive — UI can adjust
const {
  recentBurst,
  isCollectible,
  burstCount
} = useMediaBurst(computed(() => contactsStore.messages), { maxGapMs: burstTimeMs })

// Shared media-export instance: drives both the per-message "retry download"
// (inside useChatMedia) and the burst "download as zip / separately" below.
// One instance so a single in-flight download tracks progress in one place.
const mediaExport = useMediaExport()
const {
  isDownloading: isBurstDownloading,
  progress: burstProgress,
  downloadAsZip,
  downloadSeparately,
} = mediaExport
const isBurstDialogOpen = ref(false)

// ─── Add-contact dialog (pure view state) ───
const isAddContactOpen = ref(false)
function openAddContactDialog() {
  isAddContactOpen.value = true
}
async function onContactCreated(contact: any) {
  // Refresh contacts and select the new one
  await contactsStore.fetchContacts()
  if (contact?.id) {
    router.push({ name: 'chat-conversation', params: { contactId: contact.id } })
  }
}

// ─── selectedAccount: owned by the view, shared by every chat composable ───
// Declared here (before the composables) so there is no temporal-dead-zone when
// the composables below read it. useChatContactsList mutates it on contact switch.
const selectedAccount = ref<string | null>(null)

// ─── Composables: each owns one concern; template destructures from them ───

// 1) Message formatting (stateless read helpers used in the message v-for)
const {
  getMessageContent,
  getSystemMessageText,
  getReplyPreviewContent,
  getMessageStatusIcon,
  getMessageStatusClass,
  formatMessageTime,
  formatContactTime,
  getDateLabel,
  shouldShowDateSeparator,
  isMediaMessage,
  hasRevokedMedia,
  getMediaUrl,
  shouldRenderMedia,
  getInteractiveButtons,
  getCTAUrlData,
  getLocationData,
  getContactsData,
  getGoogleMapsUrl,
} = useMessageFormat({
  t,
  messages: computed(() => contactsStore.messages),
  getCurrentContact: () => contactsStore.currentContact,
})

// 2) Scrolling + sticky date + unread pill + load-older infinite scroll
const {
  newMessagesCount,
  firstUnreadId,
  stickyDate,
  showStickyDate,
  messagesScroll,
  scrollToBottom,
  scrollToMessage,
  onUserActive,
  handleMessagesLengthChange,
  resetOnContactSwitch,
} = useChatScroll({
  contactsStore,
  selectedAccount,
  messagesEndRef,
  getFirstUnreadId: () => firstUnreadId.value,
  getCurrentContactId: () => contactsStore.currentContact?.id,
  hasMoreMessages: computed(() => contactsStore.hasMoreMessages),
  isLoadingOlderMessages: computed(() => contactsStore.isLoadingOlderMessages),
})

// 3) Typing indicator + reactions + emoji picker
const {
  isCurrentAccountGowa,
  reactionPickerMessageId,
  quickReactionEmojis,
  emojiPickerOpen,
  onTypingInput,
  stopTypingIndicator,
  sendReaction,
} = useChatTyping({
  contactsStore,
  getCurrentUserId: () => authStore.user?.id,
})

// 4 + 5) Messaging ↔ Media. These two have a circular dependency (the Status
// media send path crosses both: media upload → sendStatusMedia in messaging, and
// messaging's status send needs media's getMediaType / dialog / upload flag).
// Break it with a lazy holder so either composable may be created first; the
// callbacks only fire at user-send time, well after both are wired.
let _sendStatusMedia: (file: File, caption: string) => Promise<void> = async () => {}

// 4) Media upload + preview + per-message recovery (shares mediaExport)
const media = useChatMedia({
  t,
  contactsStore,
  selectedAccount,
  scrollToBottom,
  sendStatusMedia: (file: File, caption: string) => _sendStatusMedia(file, caption),
  isStatusContact,
  mediaExport,
  fileInputRef,
})
const {
  selectedFile,
  filePreviewUrl,
  isMediaDialogOpen,
  mediaCaption,
  isUploadingMedia,
  brokenMediaIds,
  retryMediaDownload,
  markMediaBroken,
  isRedownloading,
  openFilePicker,
  handleFileSelect,
  closeMediaDialog,
  sendMediaMessage,
  openMediaPreview,
  handleImageError,
  handleMediaError,
} = media

// 5) Messaging (send / retry / revoke / reply + status paths)
const {
  messageInput,
  isSending,
  retryingMessageId,
  revokingMessageId,
  revokeDialogOpen,
  sendMessage,
  sendStatusMedia,
  retryMessage,
  requestRevoke,
  confirmRevoke,
  replyToMessage,
  autoResizeTextarea,
  resetTextareaHeight,
} = useChatMessaging({
  t,
  contactsStore,
  selectedAccount,
  isCurrentAccountGowa,
  stopTypingIndicator,
  scrollToBottom,
  getMediaType: media.getMediaType,
  closeMediaDialog,
  getIsUploadingMedia: () => isUploadingMedia.value,
  setIsUploadingMedia: (v: boolean) => { isUploadingMedia.value = v },
  messageInputRef,
})
// Resolve the lazy holder now that messaging has produced the real sendStatusMedia.
_sendStatusMedia = sendStatusMedia

// 6) Canned responses + templates
const {
  cannedPickerOpen,
  cannedSearchQuery,
  cannedDialogOpen,
  selectedCannedResponse,
  cannedParamNames,
  cannedParamValues,
  isSendingCanned,
  cannedPreview,
  cannedPreviewButtons,
  handleCannedSelect,
  sendCannedResponse,
  closeCannedPicker,
  templateDialogOpen,
  selectedTemplate,
  templateParamNames,
  templateParamValues,
  templateHeaderParamName,
  templateHeaderParamValue,
  templateButtonUrlParams,
  isSendingTemplate,
  templatePreview,
  templateHeaderFile,
  templateHeaderPreview,
  templateNeedsHeaderMedia,
  templateHeaderAccept,
  showHeaderParamInput,
  handleTemplateWithParams,
  handleTemplateHeaderFile,
  clearTemplateHeaderMedia,
  sendTemplateMessage,
} = useChatCannedTemplates({
  t,
  contactsStore,
  selectedAccount,
  getCurrentContact: () => contactsStore.currentContact,
  getCurrentUser: () => authStore.user,
  messageInput,
  resetTextareaHeight,
  scrollToBottom,
})

// 7) Chat lifecycle (claim / join / release / close / assign)
const {
  isClaiming,
  isJoining,
  isAssignDialogOpen,
  isInviteDialogOpen,
  assignSearchQuery,
  canAssignContacts,
  filteredAssignableUsers,
  handleClaim,
  handleJoin,
  handleLeave,
  handleRelease,
  handleClose,
  handleReopen,
  handleBulkRelease,
  handleInvite,
  handleRemoveCollaborator,
  assignContactToUser,
} = useChatLifecycle({
  t,
  contactsStore,
  authStore,
  usersStore,
})

// 8) Contacts list (tabs, tag filter, custom actions, multi-account, selectContact)
const {
  // Accounts that actually have messages with the selected contact — drives
  // the per-contact account tabs (showing every org account instead would
  // surface accounts with no conversation for this number).
  contactAccounts,
  isTagFilterOpen,
  toggleTagFilter,
  clearTagFilter,
  customActions,
  executingActionId,
  getActionIcon,
  fetchCustomActions,
  executeCustomAction,
  visibleTabOrder,
  onTabKeydown,
  tabLabel,
  tabCount,
  contactsScroll,
  switchAccount,
  handleContactClick,
  selectContact,
  scrollActiveContactIntoView,
  fetchOrgAccounts,
} = useChatContactsList({
  t,
  contactsStore,
  selectedAccount,
  tabStripRef,
  onContactClick: (contact: Contact) => router.push(`/chat/${contact.id}`),
  onContactSelected: async (id: string) => {
    // Notes + scheduled messages are view-owned; fetch them after the contact
    // is selected, then scroll the room to the bottom.
    await Promise.all([
      notesStore.fetchNotes(id),
      scheduledStore.fetchForContact(id),
    ])
    setTimeout(() => {
      scrollToBottom(true)
      messagesScroll.setup()
    }, 50)
  },
  messagesScroll,
  resetUnreadOnSwitch: resetOnContactSwitch,
  scrollToBottom,
})

// Emoji insertion needs to mutate the messaging-owned messageInput ref.
function insertEmoji(emoji: string) {
  messageInput.value += emoji
  emojiPickerOpen.value = false
}

// ─── View orchestration: route/contacts/messages watchers + lifecycle ───
// This is the only place the chat view coordinates across composables. The
// composables below are pure concerns; the glue lives here.

// Watch for route changes → select the contact (or clear on /chat root).
watch(contactId, async (newId) => {
  if (newId) {
    notesStore.notes = []
    notesStore.hasMore = false
    await selectContact(newId)
  } else {
    wsService.setCurrentContact(null)
    contactsStore.setCurrentContact(null)
    contactsStore.clearMessages()
    notesStore.clearNotes()
    scheduledStore.clear()
  }
})

// Watch for new messages. WhatsApp Web style: while the browser tab is focused
// the user is "watching", so auto-scroll if at the bottom. When away, pile up
// unread and surface a divider above the first message that arrived (issue #280).
watch(() => contactsStore.messages.length, (newLen, oldLen) => {
  if (newLen <= oldLen) return
  const latest = contactsStore.messages[newLen - 1]
  const isIncoming = latest?.direction === 'incoming'
  // First unread of the batch — record its id before piling up the count.
  if (isIncoming && newMessagesCount.value === 0) {
    firstUnreadId.value = latest.id
  }
  handleMessagesLengthChange(newLen, oldLen, isIncoming)
})

// Watch for slash commands in the composer → open the canned picker.
watch(messageInput, (val) => {
  if (val.startsWith('/')) {
    cannedSearchQuery.value = val.slice(1) // Remove the leading /
    cannedPickerOpen.value = true
  } else if (cannedPickerOpen.value) {
    // Close picker if user removes the /
    cannedPickerOpen.value = false
    cannedSearchQuery.value = ''
  }
})

onMounted(async () => {
  // Ensure auth session is restored
  if (!authStore.isAuthenticated) {
    authStore.restoreSession()
  }

  // Keep the sidebar from overflowing on the current viewport.
  clampSidebarToViewport()
  window.addEventListener('resize', clampSidebarToViewport)

  await contactsStore.fetchContacts()

  // Setup infinite scroll for contacts list
  await nextTick()
  contactsScroll.setup()

  // Fetch users + custom actions if the agent can assign contacts.
  if (canAssignContacts.value) {
    usersStore.fetchUsers().catch(() => { /* Silently fail */ })
    fetchCustomActions()
  }

  // Fetch org-level WhatsApp accounts for the account tabs.
  await fetchOrgAccounts()

  // Fetch available tags for filtering (if not already loaded).
  if (tagsStore.tags.length === 0) {
    tagsStore.fetchTags().catch(() => {})
  }

  if (contactId.value) {
    await selectContact(contactId.value)
    // Keep the restored conversation visible in the sidebar after a refresh,
    // even if it sits far down the list. 'nearest' only moves the viewport
    // when the active row is off-screen, so it never fights a manual click.
    scrollActiveContactIntoView()
  }

  // Auto-scroll to unread divider + mark read when the agent returns. Covers
  // tab-switch (visibilitychange) and OS window focus (focus), since "tab
  // visible but window unfocused" is a real state and we don't want to send
  // blue-tick receipts when no one is looking. See issue #280.
  document.addEventListener('visibilitychange', onUserActive)
  window.addEventListener('focus', onUserActive)
})

onUnmounted(() => {
  wsService.setCurrentContact(null)
  // Clear current contact when leaving chat view so notifications work elsewhere
  contactsStore.setCurrentContact(null)
  notesStore.clearNotes()
  scheduledStore.clear()
  document.removeEventListener('visibilitychange', onUserActive)
  window.removeEventListener('focus', onUserActive)
  window.removeEventListener('resize', clampSidebarToViewport)
})

</script>

<template>
  <div class="flex h-full bg-[#0a0a0b] light:bg-gray-50">
    <!-- Contacts List -->
    <div
      class="chat-sidebar border-r border-white/[0.08] light:border-gray-200 flex flex-col bg-[#0a0a0b] light:bg-white relative"
      :style="{ width: `${sidebarWidth}px` }"
    >
      <!-- Resize Handle (right edge of the sidebar) -->
      <div
        class="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/20 active:bg-primary/30 z-10 -mr-px border-r border-transparent"
        :class="{ 'bg-primary/30': isResizingSidebar }"
        @mousedown="startSidebarResize"
      />
      <!-- Search Header -->
      <div class="p-2 border-b border-white/[0.08] light:border-gray-200">
        <div class="flex items-center gap-2">
          <div class="relative flex-1">
            <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-white/40 light:text-gray-400" />
            <Input
              v-model="contactsStore.searchQuery"
              :placeholder="$t('chat.searchContacts') + '...'"
              class="pl-8 pr-8 h-8 text-sm bg-white/[0.04] border-white/[0.1] text-white placeholder:text-white/40 light:bg-gray-50 light:border-gray-200 light:text-gray-900 light:placeholder:text-gray-400"
            />
            <!-- Clear search button: shown only when there is a query -->
            <button
              v-if="contactsStore.searchQuery"
              type="button"
              :aria-label="$t('chat.clearSearch')"
              class="absolute right-2 top-1/2 -translate-y-1/2 h-5 w-5 flex items-center justify-center rounded-full text-white/40 hover:text-white hover:bg-white/[0.1] light:text-gray-400 light:hover:text-gray-700 light:hover:bg-gray-200 transition-colors"
              @click="contactsStore.searchQuery = ''"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
          <!-- Bulk-select toggle (M4). Available to anyone who can release
               (chat.assign:write). Toggles multi-select mode on the list so the
               agent can release several chats at once. -->
          <Tooltip v-if="canAssignContacts">
            <TooltipTrigger as-child>
              <Button
                variant="ghost"
                size="icon"
                :aria-label="$t('chat.bulkSelect')"
                :class="[
                  'h-8 w-8 shrink-0 transition-colors',
                  contactsStore.bulkSelectMode
                    ? 'text-amber-400 bg-amber-500/10'
                    : 'text-white/40 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100'
                ]"
                @click="contactsStore.bulkSelectMode = !contactsStore.bulkSelectMode; if (!contactsStore.bulkSelectMode) contactsStore.clearBulkSelection()"
              >
                <CheckCheck class="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{{ $t('chat.bulkSelect') }}</TooltipContent>
          </Tooltip>
          <!-- Add Contact -->
          <Tooltip v-if="canWriteContacts">
            <TooltipTrigger as-child>
              <Button
                variant="ghost"
                size="icon"
                :aria-label="$t('chat.addContact')"
                class="h-8 w-8 shrink-0 text-white/40 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
                @click="openAddContactDialog"
              >
                <UserPlus class="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{{ $t('chat.addContact') }}</TooltipContent>
          </Tooltip>
          <!-- Tag Filter -->
          <Popover v-model:open="isTagFilterOpen">
            <PopoverTrigger as-child>
              <Button
                variant="ghost"
                size="icon"
                class="h-8 w-8 shrink-0 relative"
                :class="contactsStore.selectedTags.length > 0 ? 'text-emerald-400 bg-emerald-500/10' : 'text-white/40 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100'"
              >
                <Filter class="h-4 w-4" />
                <span v-if="contactsStore.selectedTags.length > 0" class="absolute -top-1 -right-1 h-4 w-4 rounded-full bg-emerald-500 text-[10px] text-white flex items-center justify-center">
                  {{ contactsStore.selectedTags.length }}
                </span>
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" class="w-56 p-2">
              <div class="space-y-2">
                <div class="flex items-center justify-between px-1">
                  <span class="text-sm font-medium">{{ $t('chat.filterByTags') }}</span>
                  <Button
                    v-if="contactsStore.selectedTags.length > 0"
                    variant="ghost"
                    size="sm"
                    class="h-6 px-2 text-xs"
                    @click="clearTagFilter"
                  >
                    Clear
                  </Button>
                </div>
                <Separator />
                <div v-if="tagsStore.tags.length === 0" class="py-2 text-center text-sm text-muted-foreground">
                  {{ $t('chat.noTagsAvailable') }}
                </div>
                <div v-else class="space-y-1 max-h-48 overflow-y-auto">
                  <button
                    v-for="tag in tagsStore.tags"
                    :key="tag.name"
                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-md text-sm hover:bg-white/[0.08] light:hover:bg-gray-100 transition-colors"
                    :class="contactsStore.selectedTags.includes(tag.name) && 'bg-white/[0.08] light:bg-gray-100'"
                    @click="toggleTagFilter(tag.name)"
                  >
                    <span :class="['w-2 h-2 rounded-full shrink-0', getTagColorClass(tag.color).split(' ')[0]]" />
                    <span class="flex-1 text-left truncate">{{ tag.name }}</span>
                    <Check
                      v-if="contactsStore.selectedTags.includes(tag.name)"
                      class="h-4 w-4 text-emerald-400 shrink-0"
                    />
                  </button>
                </div>
              </div>
            </PopoverContent>
          </Popover>
        </div>
        <!-- Active tag filters -->
        <div v-if="contactsStore.selectedTags.length > 0" class="flex flex-wrap gap-1 mt-2">
          <TagBadge
            v-for="tagName in contactsStore.selectedTags"
            :key="tagName"
            :color="tagsStore.getTagByName(tagName)?.color"
            class="cursor-pointer hover:opacity-80"
            @click="toggleTagFilter(tagName)"
          >
            {{ tagName }}
            <X class="h-3 w-3 ml-1" />
          </TagBadge>
        </div>
        <!-- Visibility toggles: show/hide group & newsletter chats -->
        <div class="flex items-center gap-4 mt-2">
          <label class="flex items-center gap-1.5 cursor-pointer select-none text-xs text-white/60 light:text-gray-600">
            <Switch
              :checked="!contactsStore.hideGroupChats"
              @update:checked="(v: boolean) => (contactsStore.hideGroupChats = !v)"
            />
            <Users class="h-3.5 w-3.5 opacity-70" />
            <span>{{ $t('chat.showGroups') }}</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer select-none text-xs text-white/60 light:text-gray-600">
            <Switch
              :checked="!contactsStore.hideNewsletterChats"
              @update:checked="(v: boolean) => (contactsStore.hideNewsletterChats = !v)"
            />
            <Megaphone class="h-3.5 w-3.5 opacity-70" />
            <span>{{ $t('chat.showNewsletters') }}</span>
          </label>
        </div>

        <!-- Me / Pending (+ Closed / All for supervisors) tab strip — drives
             the sidebar list. Full a11y (M5): role=tablist + arrow-key
             navigation with roving tabindex. -->
        <div
          ref="tabStripRef"
          role="tablist"
          aria-label="$t('chat.conversationTabs')"
          :class="[
            'grid gap-1 rounded-lg bg-white/[0.04] light:bg-gray-100 p-1 mt-2',
            contactsStore.canSeeSupervisorTabs ? 'grid-cols-4' : 'grid-cols-2'
          ]"
          @keydown="onTabKeydown"
        >
          <button
            v-for="tab in visibleTabOrder()"
            :key="tab"
            type="button"
            role="tab"
            :id="`tab-${tab}`"
            :aria-selected="contactsStore.activeListTab === tab"
            :tabindex="contactsStore.activeListTab === tab ? 0 : -1"
            :class="[
              'rounded-md py-1.5 text-xs font-medium whitespace-nowrap transition-all inline-flex items-center justify-center gap-1.5',
              contactsStore.activeListTab === tab
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-white/60 hover:text-white/90 hover:bg-white/[0.06] light:text-gray-600 light:hover:text-gray-800 light:hover:bg-gray-200'
            ]"
            @click="contactsStore.activeListTab = tab"
          >
            {{ tabLabel(tab) }}
            <span
              :class="[
                'inline-flex items-center justify-center min-w-[1.25rem] h-4 px-1 rounded-full text-[10px] font-semibold tabular-nums',
                contactsStore.activeListTab === tab
                  ? 'bg-white/25 text-white'
                  : 'bg-white/[0.08] text-white/60 light:bg-gray-300 light:text-gray-700'
              ]"
            >{{ tabCount(tab) }}</span>
          </button>
        </div>
      </div>

      <!-- Contacts -->
      <ScrollArea :ref="(el: any) => contactsScroll.scrollAreaRef.value = el" orientation="vertical" class="flex-1">
        <div class="py-1 w-full">
          <!-- Cross-tab search hint (M3): the query matched contacts in other
               tabs but not the current one. Surface where the hits live so the
               empty list is explained instead of looking broken. -->
          <div v-if="contactsStore.searchHint?.show" class="px-3 py-2 mx-1 mb-1 rounded-md bg-blue-500/10 text-blue-300 light:bg-blue-50 light:text-blue-700 text-xs flex items-center gap-1.5">
            <Search class="h-3.5 w-3.5 flex-shrink-0" />
            <span class="flex-1">{{ $t('chat.foundInTabs') }}</span>
            <button
              v-for="tab in contactsStore.searchHint.tabs"
              :key="tab"
              type="button"
              class="underline hover:text-blue-200 light:hover:text-blue-900"
              @click="contactsStore.activeListTab = tab"
            >{{ tabLabel(tab) }}</button>
          </div>

          <div
            v-for="contact in contactsStore.visibleContacts"
            :key="contact.id"
            :data-contact-id="contact.id"
            :class="[
              'flex items-center gap-2 px-3 py-2 cursor-pointer border-l-[3px] border-transparent transition-colors',
              contactsStore.currentContact?.id === contact.id
                ? 'bg-white/[0.14] light:bg-slate-200 border-primary'
                : 'hover:bg-white/[0.04] light:hover:bg-gray-50'
            ]"
            @click="contactsStore.bulkSelectMode ? contactsStore.toggleBulkSelect(contact.id) : handleContactClick(contact)"
          >
            <!-- Bulk-select checkbox (M4). Only rendered in select mode; the
                 row click handler above also toggles selection in that mode. -->
            <Checkbox
              v-if="contactsStore.bulkSelectMode"
              :model-value="contactsStore.selectedContactIds.has(contact.id)"
              class="h-4 w-4 flex-shrink-0"
              @update:model-value="contactsStore.toggleBulkSelect(contact.id)"
              @click.stop
            />
            <Avatar class="h-9 w-9 ring-2 ring-white/[0.1] light:ring-gray-200">
              <AvatarImage :src="avatarSrc(contact.avatar_url)" />
              <AvatarFallback :class="'text-xs bg-gradient-to-br text-white ' + getAvatarGradient(contact.name || contact.phone_number)">
                {{ getInitials(contact.name || contact.phone_number) }}
              </AvatarFallback>
            </Avatar>
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2">
                <p
                  class="flex-1 min-w-0 text-sm font-medium truncate text-white light:text-gray-900"
                  :title="contact.name || contact.phone_number"
                >
                  {{ contact.name || contact.phone_number }}
                  <!-- Groups and newsletters are mutually exclusive categories.
                       A @newsletter JID is never a group. Prefer the newsletter
                       badge when is_newsletter is set (legacy contacts may carry
                       both flags from before the mutual-exclusivity fix). -->
                  <Badge v-if="contact.is_newsletter" class="ml-1 h-4 text-[9px] align-middle bg-amber-500/20 text-amber-400 light:bg-amber-100 light:text-amber-700">
                    {{ $t('chat.newsletter') }}
                  </Badge>
                  <Badge v-else-if="contact.is_group_chat" class="ml-1 h-4 text-[9px] align-middle bg-blue-500/20 text-blue-400 light:bg-blue-100 light:text-blue-700">
                    {{ $t('chat.group') }}
                  </Badge>
                </p>
                <span class="flex-shrink-0 text-[11px] text-white/40 light:text-gray-500 tabular-nums">
                  {{ formatContactTime(contact.last_message_at) }}
                </span>
              </div>
              <div class="flex items-center justify-between gap-2">
                <p class="flex-1 min-w-0 text-xs text-white/50 light:text-gray-500 truncate flex items-center gap-1">
                  {{ isStatusContact(contact.id) ? $t('chat.statusHint') : contact.phone_number }}
                  <!-- M1: assigned-agent tag. Shows whenever a chat is assigned
                       to someone other than the viewer, so an admin can see who
                       owns each conversation at a glance and a fellow agent can
                       see who to collaborate with. -->
                  <span
                    v-if="contact.assigned_user_name && contact.assigned_user_id !== authStore.user?.id"
                    class="inline-flex items-center gap-0.5 ml-1 px-1.5 py-0.5 rounded text-[9px] font-medium bg-emerald-500/15 text-emerald-300 light:bg-emerald-100 light:text-emerald-700 truncate max-w-[90px]"
                    :title="$t('chat.assignedTo') + ' ' + contact.assigned_user_name"
                  >
                    <User class="h-2.5 w-2.5 flex-shrink-0" />
                    <span class="truncate">{{ contact.assigned_user_name }}</span>
                  </span>
                </p>
                <Badge v-if="contact.whatsapp_account" class="flex-shrink-0 h-4 text-[9px] bg-violet-500/20 text-violet-400 light:bg-violet-100 light:text-violet-700">
                  {{ contact.whatsapp_account }}
                </Badge>
                <Badge v-if="contact.unread_count > 0" class="flex-shrink-0 h-5 text-[10px] tabular-nums bg-emerald-500 text-white light:bg-emerald-600 light:text-white">
                  {{ contact.unread_count }}
                </Badge>
              </div>
            </div>
          </div>

          <!-- Loading indicator for infinite scroll -->
          <div v-if="contactsStore.isLoadingMoreContacts" class="p-3 text-center">
            <Loader2 class="h-5 w-5 mx-auto animate-spin text-white/40 light:text-gray-400" />
          </div>

          <div v-if="contactsStore.visibleContacts.length === 0 && !contactsStore.searchQuery.trim()" class="p-3 text-center text-white/40 light:text-gray-500">
            <User class="h-6 w-6 mx-auto mb-1.5 opacity-50" />
            <p class="text-sm">{{ $t('chat.noContacts') }}</p>
          </div>
          <div v-else-if="contactsStore.visibleContacts.length === 0 && contactsStore.searchQuery.trim()" class="p-3 text-center text-white/40 light:text-gray-500">
            <Search class="h-6 w-6 mx-auto mb-1.5 opacity-50" />
            <p class="text-sm">{{ $t('chat.noSearchResults') }}</p>
          </div>
        </div>
      </ScrollArea>

      <!-- Bulk-action bar (M4). Slides in when bulk-select mode is on. -->
      <div v-if="contactsStore.bulkSelectMode" class="flex items-center justify-between gap-2 px-3 py-2 border-t border-white/[0.08] light:border-gray-200 bg-[#0f0f10] light:bg-white">
        <span class="text-xs text-white/60 light:text-gray-600">
          {{ $t('chat.bulkSelected', { count: contactsStore.selectedContactIds.size }) }}
        </span>
        <div class="flex items-center gap-1">
          <Button size="sm" variant="ghost" class="h-7 text-xs" :disabled="contactsStore.bulkReleaseInProgress" @click="contactsStore.clearBulkSelection()">
            {{ $t('chat.cancel') }}
          </Button>
          <Button size="sm" variant="ghost" class="h-7 text-xs hover:text-amber-400 hover:bg-amber-500/10"
                  :disabled="contactsStore.selectedContactIds.size === 0 || contactsStore.bulkReleaseInProgress"
                  @click="handleBulkRelease">
            <Loader2 v-if="contactsStore.bulkReleaseInProgress" class="h-3 w-3 mr-1 animate-spin" />
            <RotateCcw v-else class="h-3 w-3 mr-1" />
            {{ $t('chat.releasedConversation') }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Chat Area -->
    <div class="flex-1 min-w-0 flex flex-col bg-[#0f0f10] light:bg-gray-50">
      <!-- No Contact Selected -->
      <div
        v-if="!contactsStore.currentContact"
        class="flex-1 flex items-center justify-center text-white/40 light:text-gray-500"
      >
        <div class="text-center">
          <div class="h-16 w-16 rounded-2xl bg-gradient-to-br from-emerald-500 to-green-600 flex items-center justify-center mx-auto mb-4 shadow-lg shadow-emerald-500/20">
            <Send class="h-8 w-8 text-white" />
          </div>
          <h3 class="font-medium text-lg mb-1 text-white light:text-gray-900">{{ $t('chat.selectConversation') }}</h3>
          <p class="text-sm text-white/50 light:text-gray-500">{{ $t('chat.chooseContact') }}</p>
        </div>
      </div>

      <!-- Chat Interface -->
      <template v-else>
        <!-- Chat Header -->
        <div class="h-14 flex-shrink-0 px-4 border-b border-white/[0.08] light:border-gray-200 flex items-center justify-between bg-[#0f0f10] light:bg-white">
          <div class="flex items-center gap-2">
            <Avatar class="h-8 w-8 ring-2 ring-white/[0.1] light:ring-gray-200">
              <AvatarImage :src="avatarSrc(contactsStore.currentContact.avatar_url)" />
              <AvatarFallback :class="'text-xs bg-gradient-to-br text-white ' + getAvatarGradient(contactsStore.currentContact.name || contactsStore.currentContact.phone_number)">
                {{ getInitials(contactsStore.currentContact.name || contactsStore.currentContact.phone_number) }}
              </AvatarFallback>
            </Avatar>
            <div>
              <div class="flex items-center gap-1.5">
                <p class="text-sm font-medium text-white light:text-gray-900">
                  {{ contactsStore.currentContact.name || contactsStore.currentContact.phone_number }}
                </p>
                <Badge v-if="contactsStore.currentContact?.assigned_user_name && !contactsStore.isPendingClaim"
                       class="text-[10px] h-5 bg-emerald-500/20 text-emerald-400 light:bg-emerald-100 light:text-emerald-700">
                  👤 {{ contactsStore.currentContact.assigned_user_name }}
                </Badge>
                <Badge v-if="contactsStore.isChatClosed"
                       class="text-[10px] h-5 bg-gray-500/20 text-gray-400 light:bg-gray-100 light:text-gray-600">
                  {{ $t('chat.conversationClosed') }}
                </Badge>
                <Badge v-if="contactsStore.currentContact?.marketing_opt_out" class="text-[10px] h-5 bg-red-500/20 text-red-400 light:bg-red-100 light:text-red-700" :title="$t('chat.marketingOptOut')">
                  {{ $t('chat.marketingOptOut') }}
                </Badge>
              </div>
              <p class="text-[11px] text-white/50 light:text-gray-500">
                {{ contactsStore.currentContact.phone_number }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-1">
            <!-- Collaborators bar (Ghost Mode: admins hidden from agents server-side) -->
            <div v-if="contactsStore.currentContact?.collaborators?.length || contactsStore.isAdminOrManager"
                 class="flex items-center gap-2">
              <div class="flex -space-x-2">
                <div v-for="collab in contactsStore.currentContact?.collaborators"
                     :key="collab.user_id"
                     class="group relative h-7 w-7 rounded-full bg-blue-500/20 ring-2 ring-background flex items-center justify-center text-xs font-medium text-blue-400"
                     :title="`${collab.name} (${collab.role})`">
                  {{ collab.name?.charAt(0)?.toUpperCase() || '?' }}
                  <!-- Admin/manager: remove (✕) button on each collaborator -->
                  <button v-if="contactsStore.isAdminOrManager && collab.user_id !== authStore.user?.id"
                          type="button"
                          :aria-label="$t('chat.removeFromConversation')"
                          class="absolute -top-1 -right-1 h-3.5 w-3.5 rounded-full bg-red-500 text-white flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                          @click.stop="handleRemoveCollaborator(collab.user_id)">
                    <X class="h-2.5 w-2.5" />
                  </button>
                </div>
                <!-- Ghost badge: admin/manager viewing is not in collaborators, show self as ghost -->
                <div v-if="contactsStore.isAdminOrManager && !contactsStore.isCollaborator"
                     class="h-7 w-7 rounded-full bg-purple-500/20 ring-2 ring-background flex items-center justify-center text-xs font-medium text-purple-300"
                     :title="$t('chat.youAreGhosting')">
                  <Ghost class="h-3.5 w-3.5" />
                </div>
              </div>
              <!-- Release button: assignee (on an open chat) OR admin/manager on any
                   non-pending chat. Closed chats are admin/manager-only per the spec
                   edge case (D3 fix): an agent assignee of a closed chat must NOT see
                   Release — they would otherwise silently transition closed → pending.
                   RotateCcw = "send back / undo claim", distinct from Leave. -->
              <Button v-if="(contactsStore.isAssignedToMe && !contactsStore.isChatClosed) || (contactsStore.isAdminOrManager && !contactsStore.isPendingClaim && !contactsStore.isChatClosed)"
                      variant="ghost" size="sm"
                      class="h-8 gap-1.5 px-3 text-xs text-white/60 hover:text-amber-400 hover:bg-amber-500/10 light:text-gray-500"
                      @click="handleRelease">
                <RotateCcw class="h-3.5 w-3.5" />
                {{ $t('chat.releasedConversation') }}
              </Button>
              <!-- Reopen + Release (admin/manager on a closed chat): closed chats
                   cannot be released directly; reopen first, then release. -->
              <Tooltip v-if="contactsStore.isChatClosed && contactsStore.isAdminOrManager">
                <TooltipTrigger as-child>
                  <Button variant="ghost" size="sm"
                          class="h-8 gap-1.5 px-3 text-xs text-white/60 hover:text-amber-400 hover:bg-amber-500/10 light:text-gray-500"
                          @click="handleReopen">
                    <RotateCcw class="h-3.5 w-3.5" />
                    {{ $t('chat.releasedConversation') }}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{{ $t('chat.releaseClosedHint') }}</TooltipContent>
              </Tooltip>
              <!-- Leave button: collaborators (not owner) OR admin/manager ghost-exit.
                   Last participant → label swaps to "Leave & Close".
                   LogOut = "exit the conversation". -->
              <Button v-if="(contactsStore.isCollaborator && !contactsStore.isAssignedToMe) || (contactsStore.isAdminOrManager && !contactsStore.isPendingClaim && !contactsStore.isChatClosed)"
                      variant="ghost" size="sm"
                      class="h-8 gap-1.5 px-3 text-xs text-white/60 hover:text-red-400 hover:bg-red-500/10 light:text-gray-500"
                      @click="handleLeave">
                <LogOut class="h-3.5 w-3.5" />
                {{ contactsStore.isLastParticipant ? $t('chat.leaveAndClose') : $t('chat.leaveConversation') }}
              </Button>
            </div>

            <!-- Divider between participant actions and chat actions -->
            <div
              v-if="(contactsStore.currentContact?.collaborators?.length || contactsStore.isAdminOrManager) && (contactsStore.isAssignedToMe || contactsStore.isCollaborator || contactsStore.isAdminOrManager)"
              class="h-5 w-px bg-white/[0.08] light:bg-gray-200 mx-1"
            />

            <!-- Invite collaborator button (managers/admins only, on open chats) -->
            <Tooltip v-if="contactsStore.canCollaborate && contactsStore.currentContact?.chat_status === 'open' && !contactsStore.isPendingClaim">
              <TooltipTrigger as-child>
                <Button variant="ghost" size="icon" class="h-8 w-8 text-white/50 hover:text-blue-400 hover:bg-blue-500/10 light:text-gray-500" @click="isInviteDialogOpen = true">
                  <UserPlus class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.inviteCollaborator') }}</TooltipContent>
            </Tooltip>

            <!-- Close conversation button.
                 Admins/managers always see it (force-close & kick all).
                 Agents see it when they are a participant (owner/collaborator). -->
            <Tooltip v-if="contactsStore.currentContact && contactsStore.currentContact.chat_status === 'open' && !contactsStore.isPendingClaim && (contactsStore.isAdminOrManager || contactsStore.isAssignedToMe || contactsStore.isCollaborator)">
              <TooltipTrigger as-child>
                <Button variant="ghost" size="icon" class="h-8 w-8 text-white/50 hover:text-red-400 hover:bg-red-500/10 light:text-gray-500" @click="handleClose">
                  <CheckCheck class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.closeConversation') }}</TooltipContent>
            </Tooltip>

            <Tooltip v-if="canAssignContacts">
              <TooltipTrigger as-child>
                <Button variant="ghost" size="icon" class="h-8 w-8 text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100" @click="isAssignDialogOpen = true">
                  <UserPlus class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.assignToAgent') }}</TooltipContent>
            </Tooltip>
            <!-- Custom Action Buttons -->
            <Tooltip v-for="action in customActions" :key="action.id">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
                  :disabled="executingActionId === action.id"
                  @click="executeCustomAction(action)"
                >
                  <Loader2 v-if="executingActionId === action.id" class="h-4 w-4 animate-spin" />
                  <component v-else :is="getActionIcon(action.icon)" class="h-4 w-4" />
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
                  class="h-8 w-8 relative text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
                  :class="isNotesPanelOpen && 'bg-amber-500/10 text-amber-400 light:bg-amber-50 light:text-amber-600'"
                  @click="isNotesPanelOpen = !isNotesPanelOpen"
                >
                  <StickyNote class="h-4 w-4" />
                  <span
                    v-if="notesStore.notes.length > 0 && !isNotesPanelOpen"
                    id="notes-badge"
                    class="absolute -top-0.5 -right-0.5 h-4 min-w-[16px] rounded-full bg-amber-500 text-[10px] text-white flex items-center justify-center px-1"
                  >
                    {{ notesStore.notes.length }}
                  </span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.internalNotes') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  id="scheduled-button"
                  class="h-8 w-8 relative text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
                  :class="isScheduledPanelOpen && 'bg-sky-500/10 text-sky-400 light:bg-sky-50 light:text-sky-600'"
                  @click="isScheduledPanelOpen = !isScheduledPanelOpen"
                >
                  <CalendarClock class="h-4 w-4" />
                  <span
                    v-if="scheduledStore.pendingCount > 0 && !isScheduledPanelOpen"
                    id="scheduled-badge"
                    class="absolute -top-0.5 -right-0.5 h-4 min-w-[16px] rounded-full bg-sky-500 text-[10px] text-white flex items-center justify-center px-1"
                  >
                    {{ scheduledStore.pendingCount }}
                  </span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.scheduledMessages') }}</TooltipContent>
            </Tooltip>
            <!-- Collect a burst of incoming files (ZIP or separate) — hidden for unclaimed/closed chats -->
            <Tooltip v-if="canExportMedia && !contactsStore.isPendingClaim && !contactsStore.isChatClosed">
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  id="collect-files-button"
                  class="relative h-8 w-8 text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
                  :disabled="!isCollectible"
                  @click="isBurstDialogOpen = true"
                >
                  <Download class="h-4 w-4" />
                  <span
                    v-if="burstCount > 0"
                    class="absolute -top-0.5 -right-0.5 h-4 min-w-[16px] rounded-full bg-primary text-[10px] text-white flex items-center justify-center px-1"
                  >
                    {{ burstCount }}
                  </span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.collectFiles') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  variant="ghost"
                  size="icon"
                  id="info-button"
                  class="h-8 w-8 text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
                  :class="isInfoPanelOpen && 'bg-white/[0.08] text-white light:bg-gray-100 light:text-gray-900'"
                  @click="isInfoPanelOpen = !isInfoPanelOpen"
                >
                  <Info class="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.contactInfo') }}</TooltipContent>
            </Tooltip>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="icon" class="h-8 w-8 text-white/50 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100">
                  <MoreVertical class="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>{{ $t('chat.contactOptions') }}</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem v-if="canAssignContacts" @click="isAssignDialogOpen = true">
                  <UserPlus class="mr-2 h-4 w-4" />
                  <span>{{ $t('chat.assignToAgent') }}</span>
                </DropdownMenuItem>
                <DropdownMenuItem @click="isInfoPanelOpen = !isInfoPanelOpen">
                  <Info class="mr-2 h-4 w-4" />
                  <span>{{ isInfoPanelOpen ? $t('chat.hideContactDetails') : $t('chat.viewContactDetails') }}</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Account Tabs (per-contact: only the accounts that have messages
             with this number, and only when more than one such account exists) -->
        <div
          v-if="contactAccounts.length > 1 && selectedAccount"
          class="flex-shrink-0 px-4 py-2 border-b border-white/[0.08] light:border-gray-200 bg-[#0a0a0b] light:bg-gray-50"
        >
          <div class="inline-flex items-center gap-1 rounded-lg bg-white/[0.06] light:bg-gray-100 p-1">
            <button
              v-for="acct in contactAccounts"
              :key="acct"
              :class="[
                'rounded-md px-3 py-1 text-xs font-medium whitespace-nowrap transition-all',
                acct === selectedAccount
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'bg-white/[0.08] text-white/70 hover:text-white/90 hover:bg-white/[0.12] light:bg-gray-200 light:text-gray-600 light:hover:text-gray-800 light:hover:bg-gray-300'
              ]"
              @click="switchAccount(acct)"
            >
              {{ acct }}
            </button>
          </div>
        </div>

        <!-- Messages -->
        <div class="relative flex-1 min-h-0 overflow-hidden">
          <!-- Loading overlay while switching contacts / loading the first page -->
          <Spinner v-if="contactsStore.isLoadingMessages" overlay />



          <!-- Sticky date header -->
          <Transition name="sticky-date">
            <div
              v-if="showStickyDate"
              class="absolute top-2 left-1/2 -translate-x-1/2 z-10 px-3 py-1 bg-white/[0.08] light:bg-gray-200 backdrop-blur-sm rounded-full text-[11px] text-white/50 light:text-gray-600 font-medium shadow-sm"
            >
              {{ stickyDate }}
            </div>
          </Transition>

          <!-- Floating "files just in" chip — appears when a burst is collectible (hidden for unclaimed chats) -->
          <Transition name="sticky-date">
            <button
              v-if="isCollectible && !contactsStore.isPendingClaim && !contactsStore.isChatClosed"
              type="button"
              class="absolute top-12 left-1/2 -translate-x-1/2 z-20 flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium shadow-lg hover:opacity-90 animate-pulse"
              @click="isBurstDialogOpen = true"
            >
              <Download class="h-3.5 w-3.5" />
              {{ $t('chat.filesJustIn', { count: burstCount }) }}
            </button>
          </Transition>

          <!-- Claim screen: pending unassigned conversation -->
          <div v-if="contactsStore.isPendingClaim"
               class="flex flex-col items-center justify-center h-full max-w-md mx-auto px-6 text-center">
            <div class="mb-5 flex h-16 w-16 items-center justify-center rounded-full bg-amber-500/10 ring-1 ring-amber-500/20">
              <Lock class="h-8 w-8 text-amber-500" />
            </div>
            <h3 class="text-base font-semibold text-foreground mb-1.5">{{ $t('chat.chatNotClaimed') }}</h3>
            <p class="text-sm text-muted-foreground mb-4 max-w-[36ch]">{{ $t('chat.claimToViewMessages') }}</p>
            <div v-if="contactsStore.pendingMessageCount > 0" class="flex items-baseline gap-1.5 mb-6">
              <span class="text-2xl font-bold text-amber-500 tabular-nums">{{ contactsStore.pendingMessageCount }}</span>
              <span class="text-xs text-muted-foreground">{{ $t('chat.messagesWaiting') }}</span>
            </div>
            <Button size="lg" :disabled="isClaiming" @click="handleClaim" class="gap-2">
              <Hand class="h-5 w-5" />
              {{ isClaiming ? $t('common.loading') : $t('chat.claimChat') }}
            </Button>
          </div>

          <!-- Closed screen: conversation is closed, show reopen button.
               Admins/managers bypass this — they can view closed content
               instantly (spec §1.A + §3: "Admin can still see content? YES"). -->
          <div v-else-if="contactsStore.isChatClosed && !contactsStore.canManageAllChats"
               class="flex flex-col items-center justify-center h-full max-w-md mx-auto px-6 text-center">
            <div class="mb-5 flex h-16 w-16 items-center justify-center rounded-full bg-gray-500/10 ring-1 ring-gray-500/20">
              <CheckCheck class="h-8 w-8 text-gray-400" />
            </div>
            <h3 class="text-base font-semibold text-foreground mb-1.5">{{ $t('chat.conversationClosed') }}</h3>
            <p class="text-sm text-muted-foreground mb-6 max-w-[36ch]">{{ $t('chat.reopenHint') }}</p>
            <Button size="lg" :disabled="isClaiming" @click="handleClaim" class="gap-2">
              <Hand class="h-5 w-5" />
              {{ isClaiming ? $t('common.loading') : $t('chat.reopenConversation') }}
            </Button>
          </div>

          <!-- Join screen: assigned to another agent — only for agents (not managers/admins who see everything) -->
          <div v-else-if="contactsStore.isAssignedToOther && !contactsStore.isCollaborator && contactsStore.canCollaborate && !contactsStore.canManageAllChats"
               class="flex flex-col items-center justify-center h-full max-w-md mx-auto px-6 text-center">
            <div class="mb-5 flex h-16 w-16 items-center justify-center rounded-full bg-blue-500/10 ring-1 ring-blue-500/20">
              <Users class="h-8 w-8 text-blue-500" />
            </div>
            <h3 class="text-base font-semibold text-foreground mb-1.5">{{ $t('chat.assignedToAnother') }}</h3>
            <p class="text-sm text-muted-foreground mb-6 max-w-[36ch]">{{ $t('chat.joinAsCollaboratorHint') }}</p>
            <Button size="lg" :disabled="isJoining" @click="handleJoin" class="gap-2">
              <UserPlus class="h-5 w-5" />
              {{ isJoining ? $t('common.loading') : $t('chat.joinAsCollaborator') }}
            </Button>
          </div>

          <ScrollArea v-else :ref="(el: any) => messagesScroll.scrollAreaRef.value = el" class="h-full p-3 chat-background">
            <div class="space-y-2">
              <!-- Loading indicator for older messages -->
              <div v-if="contactsStore.isLoadingOlderMessages" class="flex justify-center py-2">
                <div class="flex items-center gap-2 text-white/40 light:text-gray-500 text-sm">
                  <div class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                  <span>{{ $t('chat.loadingOlderMessages') }}...</span>
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
                  <div class="px-3 py-1 bg-white/[0.06] light:bg-gray-200 rounded-full text-[11px] text-white/40 light:text-gray-600 font-medium">
                    {{ getDateLabel(message.created_at) }}
                  </div>
                </div>

                <!-- Unread divider (WhatsApp-style; appears above the first
                     message that arrived while the tab was hidden) -->
                <div
                  v-if="newMessagesCount > 0 && message.id === firstUnreadId"
                  class="flex items-center justify-center my-4"
                >
                  <div class="px-3 py-1 bg-emerald-500/15 light:bg-emerald-100 rounded-full text-[11px] text-emerald-400 light:text-emerald-700 font-medium tabular-nums">
                    {{ newMessagesCount }} {{ newMessagesCount === 1 ? $t('chat.unreadMessage') : $t('chat.unreadMessages') }}
                  </div>
                </div>

              <!-- System message -->
              <div
                v-if="message.metadata?.is_system_message"
                :id="`message-${message.id}`"
                class="flex items-center justify-center my-3 w-full animate-fade-in"
              >
                <div class="px-3.5 py-1 bg-white/[0.04] light:bg-gray-200/60 rounded-full text-[11px] text-white/45 light:text-gray-500 font-medium max-w-[85%] text-center select-none border-none shadow-none">
                  {{ getSystemMessageText(message) }}
                </div>
              </div>

              <!-- Message bubble -->
              <div
                v-else
                :id="`message-${message.id}`"
                :class="[
                  'flex group',
                  message.direction === 'outgoing' ? 'justify-end' : 'justify-start'
                ]"
              >
              <div
                :class="[
                  'chat-bubble',
                  message.direction === 'outgoing' ? 'chat-bubble-outgoing' : 'chat-bubble-incoming',
                  message.status === 'revoked' && 'chat-bubble-revoked'
                ]"
              >
                <!-- Reply preview (if this message is replying to another) -->
                <!-- Sender name for group messages (who sent it inside the group) -->
                <span
                  v-if="message.direction === 'incoming' && contactsStore.currentContact?.is_group_chat && (message.sender_push_name || message.sender_phone)"
                  class="block text-xs font-medium mb-1 text-blue-400"
                >
                  {{ message.sender_push_name || message.sender_phone }}
                </span>
                <!-- Revoked (delete-for-everyone) placeholder. Both the inbound
                     message.revoked webhook and the outbound revoke handler set
                     status "revoked", so a single render path covers both.
                     Media files are kept visible under a red tint with the
                     deleted label overlaid, since the backend preserves
                     media_url on revoke. -->
                <template v-if="message.status === 'revoked'">
                  <!-- Revoked: the original content stays VISIBLE (dimmed) with a
                       small "deleted" badge on top — NOT an opaque red overlay that
                       hides it. Matches WhatsApp: you still see what you sent, just
                       marked as unsent. Applies to text, image, video, audio, document. -->
                  <div class="revoked-badge">
                    <Ban class="h-3 w-3 shrink-0" />
                    {{ $t('chat.messageRevokedPlaceholder') }}
                  </div>
                  <div class="revoked-content mt-1">
                    <!-- Original media (kept dimmed, still previewable/downloadable) -->
                    <img
                      v-if="message.message_type === 'image' || message.message_type === 'sticker'"
                      :src="getMediaUrl(message)"
                      :alt="message.media_filename || 'media'"
                      class="max-w-[280px] max-h-[300px] rounded-lg cursor-pointer object-cover"
                      @click="openMediaPreview(message)"
                      @error="handleImageError($event)"
                    />
                    <video
                      v-else-if="message.message_type === 'video'"
                      :src="getMediaUrl(message)"
                      controls
                      class="max-w-[280px] max-h-[300px] rounded-lg"
                    />
                    <audio
                      v-else-if="message.message_type === 'audio'"
                      :src="getMediaUrl(message)"
                      controls
                      class="max-w-[280px]"
                    />
                    <a
                      v-else-if="hasRevokedMedia(message)"
                      :href="getMediaUrl(message)"
                      :download="message.media_filename || 'document'"
                      class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
                    >
                      <FileText class="h-5 w-5 text-muted-foreground" />
                      <span class="text-sm truncate max-w-[200px]">{{ message.media_filename || 'Document' }}</span>
                    </a>
                    <!-- Original text (kept dimmed + line-through) -->
                    <span
                      v-else-if="getMessageContent(message)"
                      class="block whitespace-pre-wrap break-words line-through opacity-60"
                    >{{ getMessageContent(message) }}</span>
                  </div>
                </template>
                <template v-else>
                <div
                  v-if="message.is_reply && message.reply_to_message"
                  class="reply-preview cursor-pointer text-xs"
                  @click="scrollToMessage(message.reply_to_message_id)"
                >
                  <p class="font-medium">
                    {{ message.reply_to_message.direction === 'incoming' ? (contactsStore.currentContact?.profile_name || contactsStore.currentContact?.name || 'Customer') : 'You' }}
                  </p>
                  <p class="truncate">
                    {{ getReplyPreviewContent(message) }}
                  </p>
                </div>
                <!-- Template header media (image/video/document shown above template text) -->
                <div v-if="message.message_type === 'template' && message.media_url" class="mb-2">
                  <img
                    v-if="message.media_mime_type?.startsWith('image/')"
                    :src="getMediaUrl(message)"
                    alt="Template header"
                    class="max-w-[280px] max-h-[300px] rounded-lg cursor-pointer object-cover"
                    @click="openMediaPreview(message)"
                    @error="handleImageError($event)"
                  />
                  <video
                    v-else-if="message.media_mime_type?.startsWith('video/')"
                    :src="getMediaUrl(message)"
                    controls
                    class="max-w-[280px] max-h-[300px] rounded-lg"
                  />
                  <a
                    v-else
                    :href="getMediaUrl(message)"
                    :download="message.media_filename || 'document'"
                    class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
                  >
                    <FileText class="h-5 w-5 text-muted-foreground" />
                    <span class="text-sm truncate max-w-[200px]">{{ message.media_filename || 'Document' }}</span>
                  </a>
                </div>
                <!-- Image message. Rendered even when media_url is empty so the
                     <img> request hits /api/media/{id} and the backend can lazily
                     download history-synced bytes on first view. Skipped for
                     status/newsletter contacts where media is never recoverable
                     (would only produce a guaranteed 404). -->
                <div v-else-if="message.message_type === 'image' && shouldRenderMedia(message)" class="mb-2">
                  <img
                    v-if="!brokenMediaIds.has(message.id)"
                    :src="getMediaUrl(message)"
                    :alt="message.content?.body || 'Image'"
                    class="max-w-[280px] max-h-[300px] rounded-lg cursor-pointer object-cover"
                    @click="openMediaPreview(message)"
                    @error="markMediaBroken(message)"
                  />
                  <MediaRetryButton
                    v-else
                    :message="message"
                    :is-redownloading="isRedownloading(message)"
                    @retry="retryMediaDownload(message)"
                  />
                </div>
                <!-- Sticker message -->
                <div v-else-if="message.message_type === 'sticker' && shouldRenderMedia(message)" class="mb-2">
                  <img
                    v-if="!brokenMediaIds.has(message.id)"
                    :src="getMediaUrl(message)"
                    alt="Sticker"
                    class="max-w-[128px] max-h-[128px] cursor-pointer"
                    @click="openMediaPreview(message)"
                    @error="markMediaBroken(message)"
                  />
                  <MediaRetryButton
                    v-else
                    :message="message"
                    :is-redownloading="isRedownloading(message)"
                    @retry="retryMediaDownload(message)"
                  />
                </div>
                <!-- Video message. Rendered even when media_url is empty so the
                     <video> request triggers the backend lazy-recovery download.
                     Skipped for status/newsletter contacts. -->
                <div v-else-if="message.message_type === 'video' && shouldRenderMedia(message)" class="mb-2">
                  <video
                    v-if="!brokenMediaIds.has(message.id)"
                    :src="getMediaUrl(message)"
                    controls
                    class="max-w-[280px] max-h-[300px] rounded-lg"
                    @error="markMediaBroken(message)"
                  />
                  <MediaRetryButton
                    v-else
                    :message="message"
                    :is-redownloading="isRedownloading(message)"
                    @retry="retryMediaDownload(message)"
                  />
                </div>
                <!-- Audio message. Rendered even when media_url is empty so the
                     <audio> request triggers the backend lazy-recovery download.
                     Skipped for status/newsletter contacts. -->
                <div v-else-if="message.message_type === 'audio' && shouldRenderMedia(message)" class="mb-2">
                  <audio
                    :src="getMediaUrl(message)"
                    controls
                    class="max-w-[280px]"
                    @error="handleMediaError($event, 'audio')"
                  />
                </div>
                <!-- Document message. Link is always rendered so the download
                     request triggers backend lazy-recovery when needed. -->
                <div v-else-if="message.message_type === 'document'" class="mb-2">
                  <a
                    :href="getMediaUrl(message)"
                    :download="message.media_filename || 'document'"
                    class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
                  >
                    <FileText class="h-5 w-5 text-muted-foreground" />
                    <span class="text-sm truncate max-w-[200px]">
                      {{ message.media_filename || 'Document' }}
                    </span>
                  </a>
                </div>
                <!-- Location message -->
                <div v-else-if="message.message_type === 'location' && getLocationData(message)" class="mb-2">
                  <a
                    :href="getGoogleMapsUrl(getLocationData(message)!)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex items-center gap-3 px-3 py-3 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
                  >
                    <div class="h-10 w-10 rounded-full bg-red-900/30 light:bg-red-100 flex items-center justify-center shrink-0">
                      <MapPin class="h-5 w-5 text-red-500" />
                    </div>
                    <div class="flex-1 min-w-0">
                      <p v-if="getLocationData(message)?.name" class="text-sm font-medium truncate">
                        {{ getLocationData(message)?.name }}
                      </p>
                      <p v-else class="text-sm font-medium">Location</p>
                      <p v-if="getLocationData(message)?.address" class="text-xs text-muted-foreground truncate">
                        {{ getLocationData(message)?.address }}
                      </p>
                      <p class="text-xs text-muted-foreground">
                        {{ getLocationData(message)?.latitude.toFixed(6) }}, {{ getLocationData(message)?.longitude.toFixed(6) }}
                      </p>
                    </div>
                    <ExternalLink class="h-4 w-4 text-muted-foreground shrink-0" />
                  </a>
                </div>
                <!-- Contacts message -->
                <div v-else-if="message.message_type === 'contacts' && getContactsData(message).length > 0" class="mb-2 space-y-2">
                  <div
                    v-for="(contact, idx) in getContactsData(message)"
                    :key="idx"
                    class="flex items-center gap-3 px-3 py-2 bg-background/50 rounded-lg"
                  >
                    <div class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <User class="h-5 w-5 text-primary" />
                    </div>
                    <div class="flex-1 min-w-0">
                      <p class="text-sm font-medium truncate">{{ contact.name }}</p>
                      <div v-if="contact.phones?.length" class="flex items-center gap-1 text-xs text-muted-foreground">
                        <Phone class="h-3 w-3" />
                        <span class="truncate">{{ contact.phones.join(', ') }}</span>
                      </div>
                    </div>
                  </div>
                </div>
                <!-- Unsupported message -->
                <div v-else-if="message.message_type === 'unsupported'" class="mb-2">
                  <div class="flex items-center gap-2 px-3 py-2 bg-muted/50 rounded-lg text-muted-foreground">
                    <AlertCircle class="h-4 w-4 shrink-0" />
                    <span class="text-sm italic">{{ $t('chat.unsupportedMessage') }}</span>
                  </div>
                </div>
                <!-- Button reply - WhatsApp style -->
                <div v-if="message.message_type === 'button_reply'" class="button-reply-bubble">
                  <span class="whitespace-pre-wrap break-words"><template v-for="(seg, idx) in linkifySegments(getMessageContent(message))" :key="idx"><a v-if="seg.href" :href="seg.href" target="_blank" rel="noopener noreferrer" class="chat-bubble-link" @click.stop>{{ seg.text }}</a><template v-else>{{ seg.text }}</template></template></span>
                  <span class="chat-bubble-time"><span>{{ formatMessageTime(message.created_at) }}</span></span>
                </div>
                <!-- Text content (for text messages or captions) -->
                <span v-else-if="getMessageContent(message)" class="whitespace-pre-wrap break-words"><template v-for="(seg, idx) in linkifySegments(getMessageContent(message))" :key="idx"><a v-if="seg.href" :href="seg.href" target="_blank" rel="noopener noreferrer" class="chat-bubble-link" @click.stop>{{ seg.text }}</a><template v-else>{{ seg.text }}</template></template><span class="chat-bubble-time"><span>{{ formatMessageTime(message.created_at) }}</span><component v-if="message.direction === 'outgoing'" :is="getMessageStatusIcon(message.status)" :class="['h-4 w-4 status-icon', getMessageStatusClass(message.status)]" /></span></span>
                <!-- Fallback for media without URL. Reached when recovery is
                     impossible — e.g. history-synced media in WhatsApp Status or
                     newsletter contacts, where the bytes were never downloaded
                     and the provider's /message/{id}/download endpoint rejects
                     the JID. Show a neutral card with the filename instead of a
                     broken image or a guaranteed-404 request. -->
                <div v-else-if="isMediaMessage(message) && message.message_type !== 'document' && !message.media_url" class="mb-2 flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg max-w-[280px]">
                  <ImageIcon v-if="message.message_type === 'image' || message.message_type === 'sticker'" class="h-5 w-5 text-muted-foreground shrink-0" />
                  <Video v-else-if="message.message_type === 'video'" class="h-5 w-5 text-muted-foreground shrink-0" />
                  <FileText v-else class="h-5 w-5 text-muted-foreground shrink-0" />
                  <span class="text-sm text-muted-foreground truncate">
                    {{ message.media_filename || $t('chat.mediaMissing') }}
                  </span>
                </div>
                <!-- Interactive buttons - WhatsApp style -->
                <div
                  v-if="getInteractiveButtons(message).length > 0"
                  class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t"
                >
                  <template v-for="(btn, index) in getInteractiveButtons(message)" :key="btn.id">
                    <a
                      v-if="btn.type === 'URL' && btn.url"
                      :href="btn.url"
                      target="_blank"
                      rel="noopener noreferrer"
                      :class="['py-2 text-sm text-center font-medium cursor-pointer flex items-center justify-center gap-1.5', index > 0 && 'border-t']"
                    >
                      <ExternalLink class="h-3.5 w-3.5" />
                      {{ btn.title }}
                    </a>
                    <div
                      v-else
                      :class="['py-2 text-sm text-center font-medium cursor-pointer', index > 0 && 'border-t']"
                    >
                      {{ btn.title }}
                    </div>
                  </template>
                </div>
                <!-- CTA URL button - WhatsApp style -->
                <a
                  v-if="getCTAUrlData(message)"
                  :href="getCTAUrlData(message)?.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t block"
                >
                  <div class="py-2 text-sm text-center font-medium cursor-pointer flex items-center justify-center gap-1.5">
                    <ExternalLink class="h-3.5 w-3.5" />
                    {{ getCTAUrlData(message)?.button_text }}
                  </div>
                </a>

                <!-- Time for messages without text content -->
                <span v-if="!getMessageContent(message) && !(isMediaMessage(message) && !message.media_url)" class="chat-bubble-time block clear-both">
                  <span>{{ formatMessageTime(message.created_at) }}</span>
                  <component
                    v-if="message.direction === 'outgoing'"
                    :is="getMessageStatusIcon(message.status)"
                    :class="['h-4 w-4 status-icon', getMessageStatusClass(message.status)]"
                  />
                </span>
                </template>
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
                  v-if="message.status === 'failed' && message.direction === 'outgoing' && message.message_type !== 'template'"
                  class="flex items-center gap-1 mt-1 text-xs text-destructive"
                >
                  <AlertCircle class="h-3 w-3" />
                  <span>{{ message.error_message || 'Failed to send' }}</span>
                </span>
                <!-- Failed template message indicator (no retry) -->
                <span
                  v-if="message.status === 'failed' && message.direction === 'outgoing' && message.message_type === 'template'"
                  class="flex items-center gap-1 mt-1 text-xs text-destructive"
                >
                  <AlertCircle class="h-3 w-3" />
                  <span>{{ message.error_message || 'Failed to send' }}</span>
                </span>
              </div>
              <!-- Hover actions: reaction + reply for all messages; retry/revoke for outgoing -->
              <div class="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity self-center ml-1">
                <Popover :open="reactionPickerMessageId === message.id" @update:open="(open: boolean) => reactionPickerMessageId = open ? message.id : null">
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
                  v-if="message.direction === 'outgoing' && message.status === 'failed' && message.message_type !== 'template'"
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 text-destructive hover:text-destructive"
                  :disabled="retryingMessageId === message.id"
                  @click="retryMessage(message)"
                  title="Retry sending"
                >
                  <Loader2 v-if="retryingMessageId === message.id" class="h-3 w-3 animate-spin" />
                  <RotateCw v-else class="h-3 w-3" />
                </Button>
                <!-- Revoke (delete-for-everyone). GOWA-only and only for sent
                     outgoing messages that have a WhatsApp ID and aren't already
                     revoked. The backend re-validates the GOWA guard. Opens a
                     styled ConfirmDialog (requestRevoke) — never deletes inline. -->
                <Button
                  v-if="canRevokeMessages && message.direction === 'outgoing' && isCurrentAccountGowa && message.status !== 'revoked' && message.status !== 'failed' && message.wamid"
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 text-muted-foreground hover:text-destructive"
                  :disabled="revokingMessageId === message.id"
                  :title="$t('chat.revoke')"
                  @click="requestRevoke(message)"
                >
                  <Loader2 v-if="revokingMessageId === message.id" class="h-3 w-3 animate-spin" />
                  <Trash2 v-else class="h-3 w-3" />
                </Button>
              </div>
            </div>
            </template>
            <div ref="messagesEndRef" />
          </div>
        </ScrollArea>
        </div>

        <!-- Reply indicator -->
        <div
          v-if="contactsStore.replyingTo"
          class="px-4 py-2 border-t border-white/[0.08] light:border-gray-200 bg-white/[0.04] light:bg-gray-50 flex items-center justify-between"
        >
          <div class="flex-1 min-w-0">
            <p class="text-xs font-medium text-white/50 light:text-gray-500">
              Replying to {{ contactsStore.replyingTo.direction === 'incoming' ? (contactsStore.currentContact?.profile_name || contactsStore.currentContact?.name || 'Customer') : 'Yourself' }}
            </p>
            <p class="text-sm truncate text-white/70 light:text-gray-700">
              {{ getMessageContent(contactsStore.replyingTo) || '[Media]' }}
            </p>
          </div>
          <button class="w-6 h-6 rounded hover:bg-white/[0.08] light:hover:bg-gray-200 flex items-center justify-center shrink-0 transition-colors" @click="contactsStore.clearReplyingTo">
            <X class="h-4 w-4 text-white/50 light:text-gray-500" />
          </button>
        </div>

        <!-- Closed chat banner (shown when conversation is closed).
             Admins/managers get a Reopen button so they can flip it back to
             'open' without claiming ownership. -->
        <div v-if="contactsStore.isChatClosed && !contactsStore.isPendingClaim"
             class="flex items-center justify-center gap-3 p-4 border-t border-white/[0.08] light:border-gray-200 bg-[#0f0f10] light:bg-white">
          <CheckCheck class="h-4 w-4 text-muted-foreground" />
          <span class="text-sm text-muted-foreground">{{ $t('chat.conversationClosed') }}</span>
          <Button v-if="contactsStore.canManageAllChats"
                  variant="outline" size="sm" @click="handleReopen" class="ml-2 h-7 gap-1.5">
            <RotateCw class="h-3.5 w-3.5" />
            {{ $t('chat.reopenConversation') }}
          </Button>
        </div>

        <!-- Message Input (hidden for unclaimed pending and closed conversations) -->
        <div v-else-if="!contactsStore.isPendingClaim && !contactsStore.isChatClosed" class="p-4 border-t border-white/[0.08] light:border-gray-200 bg-[#0f0f10] light:bg-white">
          <form @submit.prevent="sendMessage" class="flex items-center gap-2 p-2 rounded-xl bg-white/[0.06] light:bg-gray-100 border border-white/[0.08] light:border-gray-200">
            <Tooltip>
              <TooltipTrigger as-child>
                <span>
                  <Popover v-model:open="emojiPickerOpen">
                    <PopoverTrigger as-child>
                      <button type="button" class="w-9 h-9 rounded-lg hover:bg-white/[0.08] light:hover:bg-gray-200 flex items-center justify-center transition-colors">
                        <Smile class="w-[18px] h-[18px] text-white/40 light:text-gray-500" />
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
              <TooltipContent>{{ $t('chat.emoji') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <span>
                  <CannedResponsePicker
                    :external-open="cannedPickerOpen"
                    :external-search="cannedSearchQuery"
                    @select="handleCannedSelect"
                    @close="closeCannedPicker"
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.cannedResponses') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <span>
                  <TemplatePicker
                    :selected-account="selectedAccount"
                    @select-with-params="handleTemplateWithParams"
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.sendTemplate') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <button type="button" class="w-9 h-9 rounded-lg hover:bg-white/[0.08] light:hover:bg-gray-200 flex items-center justify-center transition-colors" @click="openFilePicker">
                  <Paperclip class="w-[18px] h-[18px] text-white/40 light:text-gray-500" />
                </button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.attachFile') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <button
                  type="button"
                  id="schedule-message-button"
                  class="w-9 h-9 rounded-lg hover:bg-white/[0.08] light:hover:bg-gray-200 flex items-center justify-center transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  :disabled="!messageInput.trim()"
                  @click="isScheduleDialogOpen = true"
                >
                  <CalendarClock class="w-[18px] h-[18px] text-white/40 light:text-gray-500" />
                </button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('chat.scheduleMessageTitle') }}</TooltipContent>
            </Tooltip>
            <input
              ref="fileInputRef"
              type="file"
              accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.csv,.html,.htm,.zip,.rar,.7z,.md,.json,.xml,.rtf"
              class="hidden"
              @change="handleFileSelect"
            />
            <textarea
              ref="messageInputRef"
              v-model="messageInput"
              :placeholder="$t('chat.typeMessage') + '...'"
              rows="1"
              class="flex-1 bg-transparent text-[14px] text-white light:text-gray-900 placeholder:text-white/30 light:placeholder:text-gray-400 focus:outline-none resize-none min-h-[36px] max-h-[120px] py-2 overflow-y-auto"
              @keydown.enter.exact.prevent="sendMessage"
              @input="autoResizeTextarea(); onTypingInput()"
              @blur="stopTypingIndicator"
            />
            <button type="submit" class="w-9 h-9 rounded-lg bg-emerald-600 hover:bg-emerald-500 light:bg-emerald-500 light:hover:bg-emerald-600 flex items-center justify-center transition-colors disabled:opacity-50" :disabled="!messageInput.trim() || isSending">
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

    <!-- Scheduled Messages Side Panel -->
    <ScheduledMessagesPanel
      v-if="contactsStore.currentContact && isScheduledPanelOpen"
      :contact-id="contactsStore.currentContact.id"
      @close="isScheduledPanelOpen = false"
    />

    <!-- Schedule Message Dialog -->
    <ScheduleMessageDialog
      v-if="contactsStore.currentContact"
      v-model:open="isScheduleDialogOpen"
      :contact-id="contactsStore.currentContact.id"
      :body="messageInput"
      :whatsapp-account="selectedAccount"
      @scheduled="messageInput = ''"
    />

    <!-- Contact Info Panel -->
    <ContactInfoPanel
      v-if="contactsStore.currentContact && isInfoPanelOpen"
      :contact="contactsStore.currentContact"
      @close="isInfoPanelOpen = false"
      @tags-updated="(tags) => contactsStore.updateContactTags(contactsStore.currentContact!.id, tags)"
    />

    <!-- Template Params Dialog -->
    <Dialog v-model:open="templateDialogOpen">
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ templateParamNames.length > 0 ? $t('chat.fillParameters') : $t('chat.preview') }}</DialogTitle>
          <DialogDescription>
            {{ selectedTemplate?.display_name || selectedTemplate?.name }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-3">
          <!-- Header media upload -->
          <HeaderMediaUpload
            v-if="templateNeedsHeaderMedia"
            :file="templateHeaderFile"
            :preview-url="templateHeaderPreview"
            :accept-types="templateHeaderAccept"
            :label="selectedTemplate?.header_type === 'IMAGE' ? $t('chat.headerImage') : selectedTemplate?.header_type === 'VIDEO' ? $t('chat.headerVideo') : $t('chat.headerDocument')"
            @change="handleTemplateHeaderFile"
            @clear="clearTemplateHeaderMedia"
          />

          <div v-if="showHeaderParamInput" class="space-y-1">
            <label class="text-sm font-medium flex items-center gap-1.5">
              <span>{{ templateHeaderParamName }}</span>
              <span class="text-[10px] uppercase tracking-wider text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {{ $t('chat.headerParamBadge', 'Header') }}
              </span>
            </label>
            <Input
              v-model="templateHeaderParamValue"
              :placeholder="templateHeaderParamName ?? ''"
              class="h-9"
            />
          </div>
          <div v-for="param in templateParamNames" :key="param" class="space-y-1">
            <label class="text-sm font-medium">{{ param }}</label>
            <Input
              v-model="templateParamValues[param]"
              :placeholder="param"
              class="h-9"
            />
          </div>
          <div v-for="(btnParam, idx) in templateButtonUrlParams" :key="`btn-${btnParam.index}`" class="space-y-1">
            <label class="text-sm font-medium">
              {{ btnParam.type === 'COPY_CODE' ? `Coupon Code (${btnParam.text})` : $t('chat.urlButtonParam', { button: btnParam.text }) }}
            </label>
            <Input
              v-model="templateButtonUrlParams[idx].value"
              :placeholder="btnParam.type === 'COPY_CODE' ? 'WELCOME10' : $t('chat.urlButtonParamPlaceholder')"
              class="h-9"
            />
          </div>
          <div v-if="templatePreview" class="space-y-1">
            <label class="text-xs font-medium text-muted-foreground">{{ $t('chat.preview') }}</label>
            <div class="chat-bubble chat-bubble-outgoing ml-auto" style="max-width: 100%;">
              <img v-if="templateHeaderPreview" :src="templateHeaderPreview" class="rounded-lg mb-2 max-h-40 w-full object-cover" />
              <span class="whitespace-pre-wrap break-words text-sm">{{ templatePreview }}</span>
              <div
                v-if="selectedTemplate?.buttons?.length"
                class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t"
              >
                <div
                  v-for="(btn, index) in selectedTemplate.buttons"
                  :key="index"
                  :class="['py-2 text-sm text-center font-medium', Number(index) > 0 && 'border-t']"
                >
                  {{ btn.text }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="templateDialogOpen = false">{{ $t('common.cancel') }}</Button>
          <Button @click="sendTemplateMessage" :disabled="isSendingTemplate">
            <Loader2 v-if="isSendingTemplate" class="h-4 w-4 mr-2 animate-spin" />
            {{ $t('chat.send') }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Canned Response Preview Dialog -->
    <Dialog v-model:open="cannedDialogOpen">
      <DialogContent class="max-w-sm">
        <!-- DialogContent doesn't forward $attrs (multi-root via DialogPortal),
             so wrap the body in a div carrying the stable id used by e2e. -->
        <div id="canned-response-dialog">
        <DialogHeader>
          <DialogTitle>{{ cannedParamNames.length > 0 ? $t('chat.fillParameters') : $t('chat.preview') }}</DialogTitle>
          <DialogDescription>
            {{ selectedCannedResponse?.name }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-3">
          <div v-for="param in cannedParamNames" :key="param" class="space-y-1">
            <label class="text-sm font-medium" :for="`canned-response-param-${param}`">{{ param }}</label>
            <Input
              :id="`canned-response-param-${param}`"
              v-model="cannedParamValues[param]"
              :placeholder="param"
              class="h-9 canned-response-param"
            />
          </div>
          <div v-if="cannedPreview || cannedPreviewButtons.length" class="space-y-1">
            <label class="text-xs font-medium text-muted-foreground">{{ $t('chat.preview') }}</label>
            <div id="canned-response-preview" class="chat-bubble chat-bubble-outgoing ml-auto" style="max-width: 100%;">
              <span v-if="cannedPreview" class="whitespace-pre-wrap break-words text-sm">{{ cannedPreview }}</span>

            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button id="canned-response-cancel" variant="outline" @click="cannedDialogOpen = false">{{ $t('common.cancel') }}</Button>
          <Button id="canned-response-send" :disabled="isSendingCanned" @click="sendCannedResponse">
            <Loader2 v-if="isSendingCanned" class="h-4 w-4 mr-2 animate-spin" />
            {{ $t('chat.send') }}
          </Button>
        </div>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Assign Contact Dialog -->
    <Dialog v-model:open="isAssignDialogOpen" @update:open="(open) => !open && (assignSearchQuery = '')">
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ $t('chat.assignContact') }}</DialogTitle>
          <DialogDescription>
            {{ $t('chat.assignContactDesc') }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-3">
          <!-- Search input -->
          <div class="relative">
            <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              v-model="assignSearchQuery"
              :placeholder="$t('chat.searchUsers') + '...'"
              class="pl-9 h-9"
            />
          </div>
          <Button
            v-if="contactsStore.currentContact?.assigned_user_id"
            variant="outline"
            class="w-full justify-start"
            @click="assignContactToUser(null); isAssignDialogOpen = false"
          >
            <UserMinus class="mr-2 h-4 w-4" />
            {{ $t('chat.unassignContact') }}
          </Button>
          <Separator />
          <ScrollArea class="max-h-[280px]">
            <div class="space-y-1">
              <Button
                v-for="user in filteredAssignableUsers"
                :key="user.id"
                :variant="contactsStore.currentContact?.assigned_user_id === user.id ? 'secondary' : 'ghost'"
                class="w-full justify-start"
                @click="assignContactToUser(user.id); isAssignDialogOpen = false"
              >
                <User class="mr-2 h-4 w-4" />
                <span>{{ user.full_name }}</span>
                <Check
                  v-if="contactsStore.currentContact?.assigned_user_id === user.id"
                  class="ml-auto h-4 w-4 text-primary"
                />
                <Badge v-else variant="outline" class="ml-auto text-xs">
                  {{ user.role?.name }}
                </Badge>
              </Button>
              <p v-if="filteredAssignableUsers.length === 0" class="text-sm text-muted-foreground text-center py-4">
                {{ $t('chat.noUsersFound') }}
              </p>
            </div>
          </ScrollArea>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Invite Collaborator Dialog -->
    <Dialog v-model:open="isInviteDialogOpen">
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ $t('chat.inviteCollaborator') }}</DialogTitle>
          <DialogDescription>
            {{ $t('chat.inviteCollaboratorDesc') }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-3">
          <ScrollArea class="max-h-[280px]">
            <div class="space-y-1">
              <Button
                v-for="user in filteredAssignableUsers"
                :key="user.id"
                :variant="contactsStore.currentContact?.collaborators?.some(c => c.user_id === user.id) ? 'secondary' : 'ghost'"
                class="w-full justify-start"
                @click="handleInvite(user.id); isInviteDialogOpen = false"
              >
                <User class="mr-2 h-4 w-4" />
                <span>{{ user.full_name }}</span>
                <Check
                  v-if="contactsStore.currentContact?.collaborators?.some(c => c.user_id === user.id)"
                  class="ml-auto h-4 w-4 text-primary"
                />
                <Badge v-else variant="outline" class="ml-auto text-xs">
                  {{ user.role?.name }}
                </Badge>
              </Button>
              <p v-if="filteredAssignableUsers.length === 0" class="text-sm text-muted-foreground text-center py-4">
                {{ $t('chat.noUsersFound') }}
              </p>
            </div>
          </ScrollArea>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Media Preview Dialog -->
    <Dialog v-model:open="isMediaDialogOpen">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('chat.sendMedia') }}</DialogTitle>
          <DialogDescription>
            {{ selectedFile?.name }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-4">
          <!-- Image preview -->
          <div v-if="selectedFile?.type.startsWith('image/') && filePreviewUrl" class="flex justify-center">
            <img
              :src="filePreviewUrl"
              :alt="selectedFile.name"
              class="max-w-full max-h-[300px] rounded-lg object-contain"
            />
          </div>
          <!-- Video preview -->
          <div v-else-if="selectedFile?.type.startsWith('video/') && filePreviewUrl" class="flex justify-center">
            <video
              :src="filePreviewUrl"
              controls
              class="max-w-full max-h-[300px] rounded-lg"
            />
          </div>
          <!-- Audio preview -->
          <div v-else-if="selectedFile?.type.startsWith('audio/')" class="flex justify-center">
            <div class="flex items-center gap-3 px-4 py-3 bg-muted rounded-lg">
              <div class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                <Paperclip class="h-5 w-5 text-primary" />
              </div>
              <div>
                <p class="font-medium text-sm">{{ selectedFile.name }}</p>
                <p class="text-xs text-muted-foreground">{{ $t('chat.audioFile') }}</p>
              </div>
            </div>
          </div>
          <!-- Document preview -->
          <div v-else-if="selectedFile" class="flex justify-center">
            <div class="flex items-center gap-3 px-4 py-3 bg-muted rounded-lg">
              <div class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
                <FileText class="h-5 w-5 text-primary" />
              </div>
              <div>
                <p class="font-medium text-sm truncate max-w-[200px]">{{ selectedFile.name }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ (selectedFile.size / 1024).toFixed(1) }} KB
                </p>
              </div>
            </div>
          </div>

          <!-- Caption input (not for audio) -->
          <div v-if="selectedFile && !selectedFile.type.startsWith('audio/')">
            <Textarea
              v-model="mediaCaption"
              :placeholder="$t('chat.mediaCaption') + '...'"
              class="min-h-[60px] max-h-[100px] resize-none"
              :rows="2"
            />
          </div>

          <!-- Actions -->
          <div class="flex justify-end gap-2">
            <Button variant="outline" @click="closeMediaDialog" :disabled="isUploadingMedia">
              {{ $t('common.cancel') }}
            </Button>
            <Button @click="sendMediaMessage" :disabled="isUploadingMedia">
              <Send v-if="!isUploadingMedia" class="mr-2 h-4 w-4" />
              <span v-if="isUploadingMedia">{{ $t('chat.sending') }}...</span>
              <span v-else>{{ $t('chat.send') }}</span>
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <!-- Add Contact Dialog -->
    <CreateContactDialog v-model:open="isAddContactOpen" @created="onContactCreated" />

    <!-- Revoke (delete-for-everyone) confirmation. Styled dialog replaces the
         old native window.confirm. confirmRevoke runs the actual revoke once
         the user confirms. -->
    <ConfirmDialog
      v-model:open="revokeDialogOpen"
      :title="$t('chat.revoke')"
      variant="destructive"
      :confirm-label="$t('chat.revoke')"
      :cancel-label="$t('common.cancel')"
      :is-submitting="!!revokingMessageId"
      @confirm="confirmRevoke"
    >
      <template #description>{{ $t('chat.revokeConfirm') }}</template>
    </ConfirmDialog>

    <!-- Media burst download dialog -->
    <MediaBurstDialog
      v-model:open="isBurstDialogOpen"
      v-model:burst-time-ms="burstTimeMs"
      :messages="recentBurst"
      :is-downloading="isBurstDownloading"
      :progress="burstProgress"
      @zip="downloadAsZip(recentBurst)"
      @separate="downloadSeparately(recentBurst)"
    />
  </div>
</template>

<style scoped>
.sticky-date-enter-active,
.sticky-date-leave-active {
  transition: opacity 0.3s ease;
}

.sticky-date-enter-from,
.sticky-date-leave-to {
  opacity: 0;
}

/* The contacts sidebar must shrink with the viewport instead of pushing the
   header controls (tabs, toggles, search) off-screen on narrow windows.
   - min-width: 0 lets the flex item shrink below its content's intrinsic size.
   - flex-shrink allows it to give ground to the chat panel. The inline width
     style (sidebarWidth) is the *preferred* size, not a floor. */
.chat-sidebar {
  min-width: 0;
  flex-shrink: 1;
  /* Never let the user-driven width exceed the usable area, even mid-resize
     before the JS clamp fires. 38vw is a generous ceiling that still leaves
     the chat panel the majority of the screen. */
  max-width: 42vw;
}

@media (min-width: 1024px) {
  .chat-sidebar {
    /* On roomy desktops, the sidebar can be wider relative to the viewport. */
    max-width: 46vw;
  }
}
</style>
