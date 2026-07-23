import { computed, type ComputedRef, type Ref } from 'vue'
import type { Message } from '@/stores/contacts'

/**
 * Message types that carry a downloadable media file. Note this intentionally
 * includes 'sticker' (a media-bearing type the chat renderer treats as media)
 * and 'template' when it has a media_url (header media).
 */
export const BURST_MEDIA_TYPES = ['image', 'video', 'audio', 'document', 'sticker'] as const

export interface UseMediaBurstOptions {
  /**
   * Maximum gap (ms) between two consecutive incoming media messages for them
   * to be considered part of the same burst. Defaults to 2 minutes — the
   * "incoming files flurry" window.
   */
  maxGapMs?: number
  /** Minimum number of files for a burst to be considered "collectible". */
  minBurstSize?: number
}

export interface UseMediaBurstResult {
  /** The most recent burst (possibly empty). Members in chronological order. */
  recentBurst: ComputedRef<Message[]>
  /** True when the recent burst is large enough to surface a collect action. */
  isCollectible: ComputedRef<boolean>
  /** Number of files in the recent burst. */
  burstCount: ComputedRef<number>
  /** Total bytes of the recent burst if `media_size` is present (else 0). */
  burstHasVideo: ComputedRef<boolean>
}

/**
 * Detects a "burst" of incoming media files arriving close together in a live
 * chat. A burst is the longest trailing run of incoming media messages where
 * each is within `maxGapMs` of the previous one. It is a *living* object — it
 * grows as new files arrive and shrinks only when the messages array changes
 * (contact switch, etc.) — so it never needs timers.
 *
 * @example
 * ```ts
 * const { recentBurst, isCollectible, burstCount } = useMediaBurst(
 *   computed(() => contactsStore.messages)
 * )
 * ```
 */
export function useMediaBurst(
  messages: Ref<Message[]> | ComputedRef<Message[]>,
  options: UseMediaBurstOptions = {}
): UseMediaBurstResult {
  const maxGapMs = options.maxGapMs ?? 120_000
  const minBurstSize = options.minBurstSize ?? 2

  const incomingMedia = computed(() =>
    messages.value.filter(
      (m) => m.direction === 'incoming' && isBurstMedia(m) && !!m.media_url
    )
  )

  const recentBurst = computed<Message[]>(() => {
    const all = incomingMedia.value
    if (all.length === 0) return []

    // Walk backwards from the most recent, grouping consecutive messages whose
    // gap to the previous member is within the window.
    const burst: Message[] = [all[all.length - 1]]
    for (let i = all.length - 2; i >= 0; i--) {
      const prev = all[i + 1]
      const cur = all[i]
      if (gapMs(cur, prev) <= maxGapMs) {
        burst.unshift(cur)
      } else {
        break // a gap larger than the window closes the burst
      }
    }
    return burst
  })

  const burstCount = computed(() => recentBurst.value.length)
  const isCollectible = computed(() => burstCount.value >= minBurstSize)
  const burstHasVideo = computed(() =>
    recentBurst.value.some((m) => m.message_type === 'video')
  )

  return { recentBurst, isCollectible, burstCount, burstHasVideo }
}

/** Whether a message counts as media for burst purposes. */
export function isBurstMedia(message: Message): boolean {
  return (BURST_MEDIA_TYPES as readonly string[]).includes(message.message_type)
}

/** Absolute gap in milliseconds between two messages (newer - older). */
function gapMs(older: Message, newer: Message): number {
  const a = Date.parse(older.created_at)
  const b = Date.parse(newer.created_at)
  if (Number.isNaN(a) || Number.isNaN(b)) return Number.POSITIVE_INFINITY
  return Math.abs(b - a)
}
