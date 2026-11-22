/**
 * Task 7: Status conversation, Tags, reply icons.
 *
 * Mounts ChatView with mocked stores/composables (the view is a thin shell;
 * the mocks below mirror the real store shapes) and asserts the three
 * Task 7 behaviours at the rendered-template level:
 *  - the Status virtual conversation carries a visible "Status" label
 *    (sidebar row and chat header), it is never hidden;
 *  - the Tags filter shows the org's tags when they exist and a teaching
 *    empty state (linking to Settings -> Tags) when they do not;
 *  - every icon-only message action has an aria-label + tooltip.
 *
 * $t is mocked to the identity function, so assertions use i18n keys.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const t = (key: string) => key

vi.mock('vue-router', () => ({
  useRoute: () => reactive({ params: {}, query: {} }),
  useRouter: () => ({ push: vi.fn() }),
  // Stub: the real RouterLink needs a router instance; the stub keeps the
  // `to` prop observable for the Tags empty-state link assertions.
  RouterLink: {
    name: 'RouterLink',
    props: ['to'],
    template: '<a><slot /></a>',
  },
}))

// vue3-emoji-picker touches indexedDB at import time, which jsdom lacks.
// The picker only loads on demand (defineAsyncComponent); stub it out.
vi.mock('vue3-emoji-picker', () => ({
  default: { name: 'EmojiPicker', template: '<div />' },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t }),
}))

vi.mock('vue-sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
}))

vi.mock('@/services/websocket', () => ({
  wsService: { setCurrentContact: vi.fn() },
}))

vi.mock('@/services/api', () => ({
  contactsService: { list: vi.fn() },
  tagsService: { list: vi.fn() },
}))

const authPermissions: Record<string, boolean> = {}
vi.mock('@/stores/auth', () => ({
  useAuthStore: () =>
    reactive({
      user: { id: 'user-me' },
      userSettings: {},
      isAuthenticated: true,
      hasPermission: (resource: string, action: string) =>
        authPermissions[`${resource}:${action}`] ?? true,
      restoreSession: vi.fn(),
    }),
}))

vi.mock('@/stores/users', () => ({
  useUsersStore: () => ({ users: [], fetchAllUsers: vi.fn() }),
}))

vi.mock('@/stores/notes', () => ({
  useNotesStore: () => ({
    notes: [],
    hasMore: false,
    fetchNotes: vi.fn(),
    clearNotes: vi.fn(),
  }),
}))

vi.mock('@/stores/scheduledMessages', () => ({
  useScheduledMessagesStore: () => ({
    fetchForContact: vi.fn(),
    clear: vi.fn(),
  }),
}))

const contactsState = reactive({
  contacts: [] as any[],
  visibleContacts: [] as any[],
  messages: [] as any[],
  currentContact: null as any,
  searchQuery: '',
  searchHint: null as any,
  isLoadingMoreContacts: false,
  isPendingClaim: false,
  pendingMessageCount: 0,
  canCollaborate: false,
  isChatClosed: false,
  isAdminOrManager: false,
  isAssignedToOther: false,
  isCollaborator: false,
  canManageAllChats: false,
  selectedAccounts: [] as string[],
  selectedTags: [] as string[],
  hideGroupChats: false,
  hideNewsletterChats: false,
  bulkSelectMode: false,
  selectedContactIds: new Set<string>(),
  bulkReleaseInProgress: false,
  bulkCloseInProgress: false,
  activeListTab: 'all',
  hasMoreMessages: false,
  isLoadingOlderMessages: false,
  replyingTo: null as any,
  fetchContacts: vi.fn(),
  setCurrentContact: vi.fn(),
  clearMessages: vi.fn(),
  clearReplyingTo: vi.fn(),
  markInternalCounterpartsRead: vi.fn(),
  toggleBulkSelect: vi.fn(),
  clearBulkSelection: vi.fn(),
  internalConversationFor: () => undefined,
})
vi.mock('@/stores/contacts', () => ({
  useContactsStore: () => contactsState,
}))

const tagsState = reactive({
  tags: [] as any[],
  loading: false,
  error: null as string | null,
  fetchTags: vi.fn(),
  getTagByName: (name: string) => tagsState.tags.find((tag) => tag.name === name),
})
vi.mock('@/stores/tags', () => ({
  useTagsStore: () => tagsState,
}))

vi.mock('@/composables/useMessageFormat', () => ({
  useMessageFormat: () => ({
    getMessageContent: (m: any) => m.content?.body ?? '',
    getSystemMessageText: () => '',
    getSystemMessageTimestamp: () => '',
    getReplyPreviewContent: () => '',
    getMessageStatusIcon: () => 'span',
    getMessageStatusClass: () => '',
    formatMessageTime: () => '10:00',
    formatContactTime: () => '10:00',
    getDateLabel: () => 'Today',
    shouldShowDateSeparator: () => false,
    isMediaMessage: (m: any) =>
      ['image', 'video', 'audio', 'document', 'sticker'].includes(m.message_type),
    hasRevokedMedia: () => false,
    getMediaUrl: (m: any) => m.media_url ?? '',
    shouldRenderMedia: (m: any) => !!m.media_url,
    isRetentionPurged: () => false,
    getInteractiveButtons: () => [],
    getCTAUrlData: () => null,
    getLocationData: () => null,
    getContactsData: () => [],
    getGoogleMapsUrl: () => '',
  }),
}))

vi.mock('@/composables/useChatScroll', () => ({
  useChatScroll: () => ({
    newMessagesCount: ref(0),
    firstUnreadId: ref(null),
    stickyDate: ref(null),
    showStickyDate: ref(false),
    messagesScroll: { scrollAreaRef: ref(null), setup: vi.fn() },
    loadOlderMessages: vi.fn(),
    scrollToBottom: vi.fn(),
    scrollToMessage: vi.fn(),
    onUserActive: vi.fn(),
    handleMessagesLengthChange: vi.fn(),
    resetOnContactSwitch: vi.fn(),
  }),
}))

vi.mock('@/composables/useChatTyping', () => ({
  useChatTyping: () => ({
    isCurrentAccountGowa: ref(true),
    reactionPickerMessageId: ref(null),
    quickReactionEmojis: ['👍'],
    emojiPickerOpen: ref(false),
    onTypingInput: vi.fn(),
    stopTypingIndicator: vi.fn(),
    sendReaction: vi.fn(),
  }),
}))

vi.mock('@/composables/useChatMedia', () => ({
  useChatMedia: () => ({
    selectedFile: ref(null),
    selectedFiles: ref([]),
    activeFileIndex: ref(0),
    filePreviewUrl: ref(''),
    isMediaDialogOpen: ref(false),
    mediaCaption: ref(''),
    isUploadingMedia: ref(false),
    uploadProgress: ref(0),
    uploadCurrent: ref(0),
    uploadTotal: ref(0),
    brokenMediaIds: ref(new Set<string>()),
    retryMediaDownload: vi.fn(),
    markMediaBroken: vi.fn(),
    isRedownloading: () => false,
    canDownloadMedia: (m: any) => !!m.media_url,
    isDownloadingMedia: () => false,
    downloadMessageFile: vi.fn(),
    openFilePicker: vi.fn(),
    handleFileSelect: vi.fn(),
    handlePaste: vi.fn(),
    removeFile: vi.fn(),
    setActiveFile: vi.fn(),
    closeMediaDialog: vi.fn(),
    sendMediaMessage: vi.fn(),
    openMediaPreview: vi.fn(),
    handleImageError: vi.fn(),
    handleMediaError: vi.fn(),
    getMediaType: () => 'image',
  }),
}))

const albumRenderItems = ref<any[]>([])
vi.mock('@/composables/useChatAlbums', () => ({
  useChatAlbums: () => ({ renderItems: albumRenderItems }),
  loadFullAlbum: vi.fn(),
}))

vi.mock('@/composables/useChatMessaging', () => ({
  useChatMessaging: () => ({
    messageInput: ref(''),
    isSending: ref(false),
    retryingMessageId: ref(null),
    revokingMessageId: ref(null),
    revokeDialogOpen: ref(false),
    sendMessage: vi.fn(),
    sendStatusMedia: vi.fn(),
    retryMessage: vi.fn(),
    requestRevoke: vi.fn(),
    confirmRevoke: vi.fn(),
    replyToMessage: vi.fn(),
    autoResizeTextarea: vi.fn(),
    resetTextareaHeight: vi.fn(),
  }),
}))

vi.mock('@/composables/useChatCannedTemplates', () => ({
  useChatCannedTemplates: () => ({
    cannedPickerOpen: ref(false),
    cannedSearchQuery: ref(''),
    cannedDialogOpen: ref(false),
    selectedCannedResponse: ref(null),
    cannedParamNames: ref([]),
    cannedParamValues: ref({}),
    isSendingCanned: ref(false),
    cannedPreview: ref(''),
    cannedPreviewButtons: ref([]),
    handleCannedSelect: vi.fn(),
    sendCannedResponse: vi.fn(),
    closeCannedPicker: vi.fn(),
    templateDialogOpen: ref(false),
    selectedTemplate: ref(null),
    templateParamNames: ref([]),
    templateParamValues: ref({}),
    templateHeaderParamName: ref(''),
    templateHeaderParamValue: ref(''),
    templateButtonUrlParams: ref([]),
    isSendingTemplate: ref(false),
    templatePreview: ref(''),
    templateHeaderFile: ref(null),
    templateHeaderPreview: ref(''),
    templateNeedsHeaderMedia: ref(false),
    templateHeaderAccept: ref('image/*'),
    showHeaderParamInput: ref(false),
    handleTemplateWithParams: vi.fn(),
    handleTemplateHeaderFile: vi.fn(),
    clearTemplateHeaderMedia: vi.fn(),
    sendTemplateMessage: vi.fn(),
  }),
}))

vi.mock('@/composables/useChatLifecycle', () => ({
  useChatLifecycle: () => ({
    isClaiming: ref(false),
    isJoining: ref(false),
    isAssignDialogOpen: ref(false),
    isInviteDialogOpen: ref(false),
    assignSearchQuery: ref(''),
    accessGrants: ref([]),
    isLoadingAccessGrants: ref(false),
    releaseAccessGrant: vi.fn(),
    canAssignContacts: ref(false),
    filteredAssignableUsers: ref([]),
    handleClaim: vi.fn(),
    handleJoin: vi.fn(),
    handleLeave: vi.fn(),
    handleRelease: vi.fn(),
    handleClose: vi.fn(),
    confirmClose: vi.fn(),
    isCloseDialogOpen: ref(false),
    isClosing: ref(false),
    canCloseConversation: ref(false),
    closableSelectedContacts: ref([]),
    canBulkClose: ref(false),
    handleBulkClose: vi.fn(),
    confirmBulkClose: vi.fn(),
    isBulkCloseDialogOpen: ref(false),
    isBulkClosing: ref(false),
    handleReopen: vi.fn(),
    handleBulkRelease: vi.fn(),
    handleInvite: vi.fn(),
    handleRemoveCollaborator: vi.fn(),
    assignContactToUser: vi.fn(),
    isTransferDialogOpen: ref(false),
    transferTargets: ref([]),
    isLoadingTransferTargets: ref(false),
    transferTargetsFailed: ref(false),
    transferSearchQuery: ref(''),
    selectedTransferTargetId: ref(null),
    isTransferring: ref(false),
    canTransferChat: ref(false),
    filteredTransferTargets: ref([]),
    loadTransferTargets: vi.fn(),
    transferChat: vi.fn(),
  }),
}))

vi.mock('@/composables/useChatContactsList', () => ({
  useChatContactsList: () => ({
    contactAccounts: ref([]),
    sendAccountStatus: ref(null),
    refreshSendAccounts: vi.fn(),
    isFilterOpen: ref(false),
    activeFilterCount: ref(0),
    filterAccountOptions: ref([]),
    toggleTagFilter: vi.fn(),
    toggleAccountFilter: vi.fn(),
    clearAllFilters: vi.fn(),
    customActions: ref([]),
    executingActionId: ref(null),
    getActionIcon: () => null,
    fetchCustomActions: vi.fn(),
    executeCustomAction: vi.fn(),
    visibleTabOrder: () => ['all'],
    tabGridClass: () => 'grid-cols-1',
    onTabKeydown: vi.fn(),
    tabLabel: (tab: string) => tab,
    tabCount: () => 0,
    contactsScroll: { scrollAreaRef: ref(null), setup: vi.fn() },
    switchAccount: vi.fn(),
    handleContactClick: vi.fn(),
    selectContact: vi.fn(),
    scrollActiveContactIntoView: vi.fn(),
    fetchOrgAccounts: vi.fn(),
  }),
}))

vi.mock('@/composables/useChatLiveProbe', () => ({
  useChatLiveProbe: vi.fn(),
}))

vi.mock('@/composables/useMediaExport', () => ({
  useMediaExport: () => ({
    isDownloading: ref(false),
    downloadAsZip: vi.fn(),
    downloadSeparately: vi.fn(),
  }),
}))

vi.mock('@/composables/useColorMode', () => ({
  useColorMode: () => ({ isDark: ref(true) }),
}))

import ChatView from './ChatView.vue'

const STATUS_CONTACT = {
  id: '__status__',
  phone_number: 'status@broadcast',
  name: 'Status',
  status: 'open',
  tags: [],
  metadata: { is_status: true },
  unread_count: 0,
  created_at: '1970-01-01T00:00:00.000Z',
  updated_at: '1970-01-01T00:00:00.000Z',
}

const NORMAL_CONTACT = {
  id: 'c-alice',
  phone_number: '966500000001',
  name: 'Alice',
  status: 'open',
  tags: [],
  metadata: {},
  unread_count: 0,
  created_at: '2026-11-22T00:00:00.000Z',
  updated_at: '2026-11-22T00:00:00.000Z',
}

function textMessage(overrides: Record<string, unknown> = {}) {
  return {
    id: 'm-1',
    direction: 'incoming',
    status: 'sent',
    message_type: 'text',
    content: { body: 'Hello' },
    created_at: '2026-11-22T10:00:00.000Z',
    wamid: null,
    media_url: null,
    ...overrides,
  }
}

async function mountChatView() {
  const wrapper = mount(ChatView as any, {
    shallow: true,
    global: {
      // Stubs must render their default slot: the contacts list lives inside
      // <ScrollArea> and the Tags filter inside <PopoverContent>.
      renderStubDefaultSlot: true,
      mocks: { $t: t },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('Task 7: Status conversation', () => {
  beforeEach(() => {
    contactsState.visibleContacts = []
    contactsState.messages = []
    contactsState.currentContact = null
    contactsState.replyingTo = null
    contactsState.fetchContacts.mockResolvedValue(undefined)
    tagsState.tags = []
    tagsState.fetchTags.mockResolvedValue({ tags: [], total: 0, page: 1, limit: 50 })
    albumRenderItems.value = []
    for (const key of Object.keys(authPermissions)) delete authPermissions[key]
  })

  it('keeps Status as a distinct sidebar entry with a visible Status label', async () => {
    contactsState.visibleContacts = [STATUS_CONTACT, NORMAL_CONTACT]
    const wrapper = await mountChatView()
    const row = wrapper.find('[data-contact-id="__status__"]')
    expect(row.exists()).toBe(true)
    expect(row.text()).toContain('chat.statusLabel')
    wrapper.unmount()
  })

  it('labels the Status conversation in the chat header too', async () => {
    contactsState.visibleContacts = [STATUS_CONTACT]
    contactsState.currentContact = STATUS_CONTACT
    const wrapper = await mountChatView()
    expect(wrapper.text()).toContain('chat.statusLabel')
    wrapper.unmount()
  })
})

describe('Task 7: Tags filter panel', () => {
  beforeEach(() => {
    contactsState.visibleContacts = [NORMAL_CONTACT]
    contactsState.messages = []
    contactsState.currentContact = null
    contactsState.replyingTo = null
    contactsState.fetchContacts.mockResolvedValue(undefined)
    tagsState.tags = []
    tagsState.fetchTags.mockResolvedValue({ tags: [], total: 0, page: 1, limit: 50 })
    albumRenderItems.value = []
    for (const key of Object.keys(authPermissions)) delete authPermissions[key]
  })

  it('shows the org tags when they exist', async () => {
    tagsState.tags = [
      { name: 'VIP', color: 'gray' },
      { name: 'Lead', color: 'gray' },
    ]
    const wrapper = await mountChatView()
    expect(wrapper.text()).toContain('VIP')
    expect(wrapper.text()).toContain('Lead')
    expect(wrapper.text()).not.toContain('chat.noTagsYet')
    wrapper.unmount()
  })

  it('teaches instead of dead-ending when there are no tags', async () => {
    tagsState.tags = []
    const wrapper = await mountChatView()
    expect(wrapper.text()).toContain('chat.noTagsYet')
    const link = wrapper.findComponent({ name: 'RouterLink' })
    expect(link.exists()).toBe(true)
    expect(link.props('to')).toEqual({ name: 'tags' })
    expect(link.text()).toContain('chat.createTagInSettings')
    wrapper.unmount()
  })

  it('shows the teaching text without a link when the user cannot open Tags settings', async () => {
    tagsState.tags = []
    authPermissions['tags:read'] = false
    const wrapper = await mountChatView()
    expect(wrapper.text()).toContain('chat.noTagsYet')
    expect(wrapper.text()).toContain('chat.createTagInSettings')
    expect(wrapper.findComponent({ name: 'RouterLink' }).exists()).toBe(false)
    wrapper.unmount()
  })
})

describe('Task 7: icon-only message actions', () => {
  beforeEach(() => {
    contactsState.visibleContacts = [NORMAL_CONTACT]
    contactsState.currentContact = NORMAL_CONTACT
    contactsState.replyingTo = null
    contactsState.fetchContacts.mockResolvedValue(undefined)
    tagsState.tags = []
    tagsState.fetchTags.mockResolvedValue({ tags: [], total: 0, page: 1, limit: 50 })
    albumRenderItems.value = []
    for (const key of Object.keys(authPermissions)) delete authPermissions[key]
  })

  it('labels reply, reaction and copy with aria-label + tooltip', async () => {
    contactsState.messages = [textMessage()]
    const wrapper = await mountChatView()
    const reply = wrapper.find('[aria-label="chat.reply"]')
    expect(reply.exists()).toBe(true)
    expect(reply.attributes('title')).toBe('chat.reply')
    const reaction = wrapper.find('[aria-label="chat.addReaction"]')
    expect(reaction.exists()).toBe(true)
    expect(reaction.attributes('title')).toBe('chat.addReaction')
    const copy = wrapper.find('[aria-label="common.copy"]')
    expect(copy.exists()).toBe(true)
    expect(copy.attributes('title')).toBe('common.copy')
    wrapper.unmount()
  })

  it('labels download and revoke on an outgoing media message', async () => {
    contactsState.messages = [
      textMessage({
        id: 'm-2',
        direction: 'outgoing',
        message_type: 'image',
        media_url: 'https://example.invalid/photo.jpg',
        media_filename: 'photo.jpg',
        wamid: 'wamid-1',
      }),
    ]
    const wrapper = await mountChatView()
    const download = wrapper.find('[aria-label="common.download"]')
    expect(download.exists()).toBe(true)
    expect(download.attributes('title')).toBe('common.download')
    const revoke = wrapper.find('[aria-label="chat.revoke"]')
    expect(revoke.exists()).toBe(true)
    expect(revoke.attributes('title')).toBe('chat.revoke')
    wrapper.unmount()
  })

  it('labels the retry action on a failed outgoing message', async () => {
    contactsState.messages = [
      textMessage({ id: 'm-3', direction: 'outgoing', status: 'failed' }),
    ]
    const wrapper = await mountChatView()
    const retry = wrapper.find('[aria-label="chat.retry"]')
    expect(retry.exists()).toBe(true)
    expect(retry.attributes('title')).toBe('chat.retry')
    wrapper.unmount()
  })

  it('labels the album reply and download actions', async () => {
    const img1 = textMessage({
      id: 'm-a1',
      direction: 'incoming',
      message_type: 'image',
      media_url: 'https://example.invalid/a1.jpg',
      media_filename: 'a1.jpg',
      content: { body: '' },
    })
    const img2 = textMessage({
      id: 'm-a2',
      direction: 'incoming',
      message_type: 'image',
      media_url: 'https://example.invalid/a2.jpg',
      media_filename: 'a2.jpg',
      content: { body: '' },
    })
    contactsState.messages = [img1, img2]
    albumRenderItems.value = [
      { kind: 'album', messages: [img1, img2], firstIndex: 0, runStartIndex: 0, fileType: 'jpeg' },
    ]
    const wrapper = await mountChatView()
    const reply = wrapper.find('[aria-label="chat.reply"]')
    expect(reply.exists()).toBe(true)
    expect(reply.attributes('title')).toBe('chat.reply')
    const sep = wrapper.find('[aria-label="chat.downloadSeparately"]')
    expect(sep.exists()).toBe(true)
    expect(sep.attributes('title')).toBe('chat.downloadSeparately')
    const zip = wrapper.find('[aria-label="chat.downloadZip"]')
    expect(zip.exists()).toBe(true)
    expect(zip.attributes('title')).toBe('chat.downloadZip')
    wrapper.unmount()
  })

  it('labels the reply-indicator dismiss button', async () => {
    contactsState.messages = [textMessage()]
    contactsState.replyingTo = textMessage()
    const wrapper = await mountChatView()
    const dismiss = wrapper.find('button[aria-label="chat.cancelReply"]')
    expect(dismiss.exists()).toBe(true)
    expect(dismiss.attributes('title')).toBe('chat.cancelReply')
    wrapper.unmount()
  })
})
