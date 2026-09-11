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
 * scroll anchors, select-mode downloads and lazy media recovery per file.
 *
 * Eligibility:
 *   - FILE types only: image, document, sticker. Audio and video NEVER join
 *     (user decision 2026-09-09: videos play inline and audio streams, so
 *     they stay individual bubbles);
 *   - no caption (a captioned file is its own bubble);
 *   - not a reply, not revoked/failed (those render their own states);
 *   - consecutive, same direction, same sender (group chats), same account;
 *   - ≤ ALBUM_GAP_MS apart (between each two ADJACENT members, so a slowly
 *     delivered batch still groups as long as no single gap exceeds it) and
 *     within the same calendar day;
 *   - at least 2 members (a lone file renders as before).
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

/** Max time between two ADJACENT members to still join one album (2 minutes —
 * generous on purpose: batches relayed through slow webhook queues can land
 * spread out; a lone bigger gap splits the run). */
export const ALBUM_GAP_MS = 120000

export interface UseChatAlbumsOptions {
  /** Render-eligibility predicate (shouldRenderMedia from useMessageFormat) —
   *  a message that would not render as media on its own must not join an
   *  album either (status/newsletter contacts have unrecoverable media). */
  shouldRenderMedia: (message: Message) => boolean
  /** IDs of messages belonging to an already-downloaded ("sealed") album.
   *  A sealed group never absorbs newly received files: when the seal status
   *  of the run's last member and the incoming candidate differs, the run is
   *  flushed so the newcomer starts a fresh group. Re-downloading a sealed
   *  album still works — sealing only splits grouping, never blocks export.
   *  Accepts a getter so the render-items computed stays reactive to seals. */
  sealedIds?: Set<string> | (() => Set<string>)
}

export function useChatAlbums(
  messages: ComputedRef<Message[]>,
  options: UseChatAlbumsOptions
) {
  const renderItems = computed<MessageRenderItem[]>(() =>
    buildRenderItems(
      messages.value,
      options.shouldRenderMedia,
      typeof options.sealedIds === 'function' ? options.sealedIds() : options.sealedIds
    )
  )
  return { renderItems }
}

/** Album membership test for a single message (excluding run/gap checks). */
function isAlbumCandidate(message: Message, shouldRenderMedia: (m: Message) => boolean): boolean {
  // File types only — image, document, sticker. Audio and video are excluded
  // by design (they play inline/stream and keep their individual bubbles).
  if (!['image', 'document', 'sticker'].includes(message.message_type)) return false
  if (message.is_reply) return false
  if (message.status === 'revoked' || message.status === 'failed') return false
  // Albums are caption-less batches; a captioned file renders as its own
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
 *  tests — pure, no reactive dependencies. `sealedIds` holds messages of an
 *  already-downloaded album: a run split on seal-status change keeps the old
 *  (sealed) bubble intact while newly received files start their own group,
 *  even inside ALBUM_GAP_MS. */
export function buildRenderItems(
  messages: Message[],
  shouldRenderMedia: (m: Message) => boolean,
  sealedIds?: Set<string>
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
    const last = group.length > 0 ? group[group.length - 1] : undefined
    // Seal-status change splits the run: a sealed (already downloaded) group
    // never absorbs a newly received file, and vice versa. Without sealedIds
    // this is a no-op — grouping behaves exactly as before.
    const sealSplit =
      !!sealedIds &&
      !!last &&
      (sealedIds.has(last.id) !== sealedIds.has(message.id))
    const joinsRun =
      candidate &&
      group.length > 0 &&
      !sealSplit &&
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
