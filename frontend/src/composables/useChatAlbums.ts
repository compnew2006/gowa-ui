import { computed } from 'vue'
import type { ComputedRef } from 'vue'
import type { Message } from '@/stores/contacts'

/**
 * WhatsApp-style media albums.
 *
 * GOWA (like the WhatsApp protocol) delivers every photo of a batch as its own
 * message — there is no album field in the webhook payload. The official
 * clients group consecutive caption-less media from one sender into a single
 * album bubble purely at render time, and so do we: this composable folds
 * `contactsStore.messages` into render items where a run of eligible messages
 * collapses into one album group. Grouping is presentation-only — every member
 * stays its own row in the DB (its own id, wamid, media URL), which keeps
 * scroll anchors, select-mode downloads and lazy media recovery per photo.
 *
 * Eligibility mirrors WhatsApp semantics:
 *   - type image/video only (stickers/documents/audio never join);
 *   - no caption (a captioned photo is its own bubble);
 *   - not a reply, not revoked/failed (those render their own states);
 *   - consecutive, same direction, same sender (group chats), same account;
 *   - ≤ ALBUM_GAP_MS apart and within the same calendar day;
 *   - at least 2 members (a lone photo renders as before).
 */
export interface AlbumGroup {
  kind: 'album'
  messages: Message[]
  /** Index of the FIRST member in the source messages array — the date
   *  separator / unread-divider logic in ChatView stays index-based. */
  firstIndex: number
}

export interface SingleMessageItem {
  kind: 'message'
  message: Message
  firstIndex: number
}

export type MessageRenderItem = AlbumGroup | SingleMessageItem

/** Max time between consecutive members to still join one album. WhatsApp
 *  sends a batch within ~1s; 3s leaves headroom for slow webhook relays. */
export const ALBUM_GAP_MS = 3000

export interface UseChatAlbumsOptions {
  /** Render-eligibility predicate (shouldRenderMedia from useMessageFormat) —
   *  a message that would not render as media on its own must not join an
   *  album either (status/newsletter contacts have unrecoverable media). */
  shouldRenderMedia: (message: Message) => boolean
}

export function useChatAlbums(
  messages: ComputedRef<Message[]>,
  options: UseChatAlbumsOptions
) {
  const renderItems = computed<MessageRenderItem[]>(() =>
    buildRenderItems(messages.value, options.shouldRenderMedia)
  )
  return { renderItems }
}

/** Album membership test for a single message (excluding run/gap checks). */
function isAlbumCandidate(message: Message, shouldRenderMedia: (m: Message) => boolean): boolean {
  if (message.message_type !== 'image' && message.message_type !== 'video') return false
  if (message.is_reply) return false
  if (message.status === 'revoked' || message.status === 'failed') return false
  // Albums are caption-less batches; a captioned photo renders as its own
  // bubble (content.body is the caption for media messages).
  if (message.content?.body) return false
  return shouldRenderMedia(message)
}

/** Whether two consecutive candidates belong to the same album side. */
function sameAlbumSide(a: Message, b: Message): boolean {
  if (a.direction !== b.direction) return false
  // A contact messaged on two org accounts is two parallel conversations —
  // never merge across the account filter's seam.
  if ((a.whatsapp_account || '') !== (b.whatsapp_account || '')) return false
  // In group chats each sender is its own bubble owner.
  if ((a.sender_phone || a.sender_push_name || '') !== (b.sender_phone || b.sender_push_name || '')) {
    return false
  }
  return true
}

/** Timestamp gap check: messages arrive oldest→newest; enforce a same-day,
 *  bounded gap. Unparseable timestamps (e.g. a streamed message not yet
 *  carrying created_at) must not group — never guess. */
function withinAlbumGap(prev: Message, next: Message): boolean {
  const t1 = Date.parse(prev.created_at)
  const t2 = Date.parse(next.created_at)
  if (Number.isNaN(t1) || Number.isNaN(t2)) return false
  if (new Date(t1).toDateString() !== new Date(t2).toDateString()) return false
  const gap = t2 - t1
  return gap >= 0 && gap <= ALBUM_GAP_MS
}

/** Fold a flat, oldest→newest message list into render items. Exported for
 *  tests — pure, no reactive dependencies. */
export function buildRenderItems(
  messages: Message[],
  shouldRenderMedia: (m: Message) => boolean
): MessageRenderItem[] {
  const items: MessageRenderItem[] = []
  let group: Message[] = []
  let groupStart = -1

  const flush = () => {
    if (group.length === 0) return
    if (group.length >= 2) {
      items.push({ kind: 'album', messages: group, firstIndex: groupStart })
    } else {
      items.push({ kind: 'message', message: group[0], firstIndex: groupStart })
    }
    group = []
    groupStart = -1
  }

  for (let i = 0; i < messages.length; i++) {
    const message = messages[i]
    const candidate = isAlbumCandidate(message, shouldRenderMedia)
    const joinsRun =
      candidate &&
      group.length > 0 &&
      sameAlbumSide(group[group.length - 1], message) &&
      withinAlbumGap(group[group.length - 1], message)

    if (joinsRun) {
      group.push(message)
      continue
    }

    flush()
    if (candidate) {
      group = [message]
      groupStart = i
    } else {
      items.push({ kind: 'message', message, firstIndex: i })
    }
  }
  flush()

  return items
}
