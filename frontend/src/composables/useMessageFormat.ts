import { Check, CheckCheck, Clock, AlertCircle } from 'lucide-vue-next'
import type { MaybeRefOrGetter } from 'vue'
import { toValue } from 'vue'
import type { Message } from '@/stores/contacts'

/** Decoded location payload stored in a `location` message. */
export interface LocationData {
  latitude: number
  longitude: number
  name?: string
  address?: string
}

/** A single contact inside a `contacts` message. */
export interface ContactData {
  name: string
  phones?: string[]
}

/** Decoded CTA URL interactive payload. */
export interface CTAUrlData {
  type: 'cta_url'
  body: string
  button_text: string
  url: string
}

export interface UseMessageFormatOptions {
  /** i18n translator — used by system-message localization. */
  t: (key: string, params?: Record<string, unknown>) => string
  /** Live messages array — used to compare adjacent dates for separators. */
  messages: MaybeRefOrGetter<Message[]>
  /** Current contact getter — used to decide if media is recoverable. */
  getCurrentContact: () => { is_newsletter?: boolean; phone_number?: string } | null
}

/**
 * Read-only message rendering helpers shared by the chat view. All functions
 * are pure (take a `message` and return a string/boolean/decoded payload) so
 * they are cheap to call inside the message `v-for`.
 *
 * @example
 * ```ts
 * const fmt = useMessageFormat({
 *   t,
 *   messages: computed(() => contactsStore.messages),
 *   getCurrentContact: () => contactsStore.currentContact,
 * })
 * ```
 */
export function useMessageFormat(options: UseMessageFormatOptions) {
  const { t, getCurrentContact } = options
  const getMessages = () => toValue(options.messages)

  // ─── System-message i18n ───
  // The six single-actor system types are localized via chat.system.* keys with
  // {agent} interpolation. collaborator_removed is deliberately excluded because
  // it carries two actors (the removed user as agent_id and the manager as
  // removed_by) — a single {agent} would silently drop the manager, so it falls
  // back to getMessageContent which preserves both names from the legacy content.
  const SYSTEM_MESSAGE_TYPES = new Set([
    'chat_claimed',
    'chat_released',
    'chat_closed',
    'chat_reopened',
    'collaborator_joined',
    'collaborator_left',
  ])

  // extractAgentFromLegacy pulls the agent name out of the legacy "🔔 <name> ..."
  // system-message content strings written before metadata.agent_name existed.
  // Returns '' when no name can be parsed so the caller can fall back to the
  // raw content (acceptable degraded behavior, never a crash).
  function extractAgentFromLegacy(content: any): string {
    let text = ''
    if (typeof content === 'string') {
      text = content
    } else if (content && typeof content === 'object') {
      text = content.body || ''
    }
    if (!text) return ''
    // Matches "🔔 Jane Doe claimed/released/closed/...". Captures the name up to
    // the verb that follows it. Tolerates the leading bell emoji + spaces.
    const m = text.match(/^🔔\s+(.+?)\s+(?:claimed|released|closed|reopened|joined|left|was|leaves)/i)
    return m ? m[1].trim() : ''
  }

  function getMessageContent(message: Message): string {
    if (message.message_type === 'text') {
      return message.content?.body || ''
    }
    if (message.message_type === 'button_reply' || message.message_type === 'nfm_reply') {
      // Button/flow reply stores the response text in content
      if (typeof message.content === 'string') {
        return message.content
      }
      return message.content?.body || ''
    }
    if (message.message_type === 'interactive' || message.message_type === 'flow') {
      // Interactive/flow messages store body text in content (string) or content.body or interactive_data.body
      if (typeof message.content === 'string') {
        return message.content
      }
      if (message.interactive_data?.body) {
        return message.interactive_data.body
      }
      return message.content?.body || '[Interactive Message]'
    }
    // For media messages, return caption if available (media is displayed inline)
    if (message.message_type === 'image' || message.message_type === 'video' || message.message_type === 'sticker') {
      return message.content?.body || ''
    }
    if (message.message_type === 'audio') {
      return '' // Audio doesn't have captions
    }
    if (message.message_type === 'document') {
      return message.content?.body || ''
    }
    if (message.message_type === 'template') {
      // Show actual content if available (campaign messages), otherwise fallback
      return message.content?.body || '[Template Message]'
    }
    if (message.message_type === 'location') {
      return '' // Location is displayed as a map/card, not text
    }
    if (message.message_type === 'contacts') {
      return '' // Contacts are displayed as a card, not text
    }
    if (message.message_type === 'unsupported') {
      return '' // Displayed as a visual card, not text
    }
    return '[Message]'
  }

  function getSystemMessageText(message: Message): string {
    const systemType = message.metadata?.system_type
    // rating_received carries {rating} instead of {agent}, so it is handled
    // outside the SYSTEM_MESSAGE_TYPES set.
    if (systemType === 'rating_received' && message.metadata?.rating != null) {
      return t('chat.system.rating_received', { rating: message.metadata.rating })
    }
    if (systemType && SYSTEM_MESSAGE_TYPES.has(systemType)) {
      const agent =
        (message.metadata?.agent_name as string | undefined) ||
        extractAgentFromLegacy(message.content) ||
        ''
      return t(`chat.system.${systemType}`, { agent })
    }
    return getMessageContent(message)
  }

  function getMessageStatusIcon(status: string) {
    switch (status) {
      case 'sent':
        return Check
      case 'delivered':
        return CheckCheck
      case 'read':
        return CheckCheck
      case 'failed':
        return AlertCircle
      default:
        return Clock
    }
  }

  function getMessageStatusClass(status: string) {
    switch (status) {
      case 'read':
        return 'text-blue-400' // Bright blue for read
      case 'failed':
        return 'text-destructive'
      default:
        return 'text-muted-foreground' // Gray for sent/delivered
    }
  }

  function formatMessageTime(dateStr: string) {
    const date = new Date(dateStr)
    return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
  }

  function formatContactTime(dateStr?: string) {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    const now = new Date()
    const diffDays = Math.floor((now.getTime() - date.getTime()) / 86400000)

    if (diffDays === 0) {
      return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
    } else if (diffDays === 1) {
      return 'Yesterday'
    } else if (diffDays < 7) {
      return date.toLocaleDateString('en-US', { weekday: 'short' })
    }
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  }

  function getDateLabel(dateStr: string): string {
    const date = new Date(dateStr)
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
    const diffDays = Math.floor((today.getTime() - messageDate.getTime()) / 86400000)

    if (diffDays === 0) {
      return 'Today'
    } else if (diffDays === 1) {
      return 'Yesterday'
    }
    return date.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })
  }

  function shouldShowDateSeparator(index: number): boolean {
    const messages = getMessages()
    if (index === 0) return true

    const currentDate = new Date(messages[index].created_at)
    const prevDate = new Date(messages[index - 1].created_at)

    return currentDate.toDateString() !== prevDate.toDateString()
  }

  function isMediaMessage(message: Message): boolean {
    return ['image', 'video', 'audio', 'document'].includes(message.message_type)
  }

  // Revoked messages keep their media_url (the backend only flips status and
  // content), so the file can stay visible under the red "deleted" overlay.
  // Broader than isMediaMessage because stickers are also overlaid.
  function hasRevokedMedia(message: Message): boolean {
    return !!message.media_url &&
      ['image', 'video', 'audio', 'document', 'sticker'].includes(message.message_type)
  }

  function getMediaUrl(message: Message): string {
    // Always point at the per-message media endpoint. History-synced messages
    // store media_url="" by design (no local file) — the backend lazily
    // downloads the bytes from the provider on first view via WhatsAppMessageID,
    // so the URL is valid even when media_url is empty. Gating on media_url
    // here would suppress the request and starve the recovery path.
    const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
    return `${basePath}/api/media/${message.id}`
  }

  /**
   * Whether this conversation's media can be lazily recovered from the provider.
   * WhatsApp Status posts and newsletters arrive as message rows during history
   * sync but their media is NOT retrievable via the chat `/message/{id}/download`
   * endpoint — GOWA rejects them with INVALID_JID. Requesting /api/media for them
   * only produces guaranteed 404s. For these contacts we skip the request and
   * render the filename directly instead.
   *
   * Also returns false for non-history-synced media with a real media_url — that
   * path is always recoverable (the file exists locally), so we never want to
   * short-circuit it.
   */
  function isMediaRecoverable(): boolean {
    const contact = getCurrentContact()
    if (!contact) return false
    // Status feed and newsletters are not chats — media is not downloadable.
    if (contact.is_newsletter) return false
    const phone = (contact.phone_number || '').toLowerCase()
    if (phone === 'status' || phone === 'broadcast' || phone.endsWith('@newsletter')) {
      return false
    }
    return true
  }

  /**
   * Whether a message should attempt to render/download its media. True when the
   * media is already local (media_url set) OR the conversation allows provider
   * recovery. False for history-synced media in status/newsletter contacts,
   * where the bytes are unreachable.
   */
  function shouldRenderMedia(message: Message): boolean {
    if (message.media_url) return true
    return isMediaRecoverable()
  }

  function getInteractiveButtons(message: Message): Array<{ id: string; title: string; type: string; url: string }> {
    if (!message.interactive_data) {
      return []
    }
    // Support both interactive and template messages with buttons
    if (message.message_type !== 'interactive' && message.message_type !== 'template') {
      return []
    }
    // Handle both "buttons" (<=3) and "rows" (>3 list format)
    const items = message.interactive_data.buttons || message.interactive_data.rows
    if (!items || !Array.isArray(items)) {
      return []
    }
    return items.map((btn: any) => ({
      id: btn.reply?.id || btn.id || '',
      title: btn.reply?.title || btn.title || btn.text || '',
      type: btn.type || 'QUICK_REPLY',
      url: btn.url || ''
    }))
  }

  function getCTAUrlData(message: Message): CTAUrlData | null {
    if (message.message_type !== 'interactive' || !message.interactive_data) {
      return null
    }
    if (message.interactive_data.type !== 'cta_url') {
      return null
    }
    return {
      type: 'cta_url',
      body: message.interactive_data.body || '',
      button_text: (message.interactive_data as any).button_text || 'Open',
      url: (message.interactive_data as any).url || ''
    }
  }

  function getLocationData(message: Message): LocationData | null {
    if (message.message_type !== 'location') return null
    try {
      // Content is stored as JSON string in body
      const body = message.content?.body || message.content
      if (typeof body === 'string') {
        return JSON.parse(body)
      }
      return body as LocationData
    } catch {
      return null
    }
  }

  function getContactsData(message: Message): ContactData[] {
    if (message.message_type !== 'contacts') return []
    try {
      // Content is stored as JSON string in body
      const body = message.content?.body || message.content
      if (typeof body === 'string') {
        return JSON.parse(body)
      }
      return body as ContactData[]
    } catch {
      return []
    }
  }

  function getGoogleMapsUrl(location: LocationData): string {
    return `https://www.google.com/maps?q=${location.latitude},${location.longitude}`
  }

  function getReplyPreviewContent(message: Message): string {
    if (!message.reply_to_message) return ''
    const reply = message.reply_to_message
    if (reply.message_type === 'text') {
      const body = reply.content?.body || ''
      return body.length > 50 ? body.substring(0, 50) + '...' : body
    }
    if (reply.message_type === 'button_reply') {
      const body = typeof reply.content === 'string' ? reply.content : (reply.content?.body || '')
      return body.length > 50 ? body.substring(0, 50) + '...' : body
    }
    if (reply.message_type === 'interactive') {
      const body = typeof reply.content === 'string' ? reply.content : ((reply as any).interactive_data?.body || reply.content?.body || '')
      return body.length > 50 ? body.substring(0, 50) + '...' : body
    }
    if (reply.message_type === 'template') {
      const body = reply.content?.body || ''
      return body.length > 50 ? body.substring(0, 50) + '...' : body
    }
    if (reply.message_type === 'image') return '[Photo]'
    if (reply.message_type === 'video') return '[Video]'
    if (reply.message_type === 'audio') return '[Audio]'
    if (reply.message_type === 'document') return '[Document]'
    if (reply.message_type === 'location') return '[Location]'
    if (reply.message_type === 'contacts') return '[Contact]'
    if (reply.message_type === 'sticker') return '[Sticker]'
    return '[Message]'
  }

  return {
    // Content decoding
    getMessageContent,
    getSystemMessageText,
    getReplyPreviewContent,
    // Status / time formatting
    getMessageStatusIcon,
    getMessageStatusClass,
    formatMessageTime,
    formatContactTime,
    getDateLabel,
    shouldShowDateSeparator,
    // Media helpers
    isMediaMessage,
    hasRevokedMedia,
    getMediaUrl,
    shouldRenderMedia,
    // Interactive / location / contacts decoding
    getInteractiveButtons,
    getCTAUrlData,
    getLocationData,
    getContactsData,
    getGoogleMapsUrl,
  }
}
