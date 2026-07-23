import { computed, toRef, type ComputedRef, type MaybeRefOrGetter, type Ref } from 'vue'
import type { Message } from '@/stores/contacts'

/**
 * Message types that carry a downloadable media file. Note this intentionally
 * includes 'sticker' (a media-bearing type the chat renderer treats as media)
 * and 'template' when it has a media_url (header media).
 */
const BURST_MEDIA_TYPES = ['image', 'document'] as const

export interface UseMediaBurstOptions {
  /**
   * Lookback window (ms) — all incoming media files within this time from now
   * are collected. Defaults to 2 minutes.
   */
  maxGapMs?: MaybeRefOrGetter<number>
  /** Minimum number of files for a burst to be considered "collectible". */
  minBurstSize?: number
}

export interface UseMediaBurstResult {
  /** All incoming media files within the lookback window. Chronological order. */
  recentBurst: ComputedRef<Message[]>
  /** True when the burst is large enough to surface a collect action. */
  isCollectible: ComputedRef<boolean>
  /** Number of files in the recent burst. */
  burstCount: ComputedRef<number>
}

/**
 * Collects all incoming media files whose timestamp falls within a reactive
 * lookback window (default 2 min from now). The burst grows/shrinks in
 * real-time as the user adjusts the time window or new messages arrive.
 *
 * @example
 * ```ts
 * const burstTimeMs = ref(120_000)
 * const { recentBurst, isCollectible, burstCount } = useMediaBurst(
 *   computed(() => contactsStore.messages),
 *   { maxGapMs: burstTimeMs }
 * )
 * ```
 */
export function useMediaBurst(
  messages: Ref<Message[]> | ComputedRef<Message[]>,
  options: UseMediaBurstOptions = {}
): UseMediaBurstResult {
  const maxGapMs = toRef(options.maxGapMs ?? 120_000)
  const minBurstSize = options.minBurstSize ?? 1

  const incomingMedia = computed(() =>
    messages.value.filter(
      (m) => m.direction === 'incoming' && isBurstMedia(m)
    )
  )

  const recentBurst = computed<Message[]>(() => {
    const all = incomingMedia.value
    if (all.length === 0) return []

    const windowMs = maxGapMs.value
    // Use the newest file as anchor so the window slides with the data,
    // not wall-clock time — prevents empty bursts on page reload.
    const newest = all[all.length - 1]
    const newestTime = Date.parse(newest.created_at)
    if (Number.isNaN(newestTime)) return []

    const cutoff = newestTime - windowMs

    return all.filter((m) => {
      const t = Date.parse(m.created_at)
      return !Number.isNaN(t) && t >= cutoff
    })
  })

  const burstCount = computed(() => recentBurst.value.length)
  const isCollectible = computed(() => burstCount.value >= minBurstSize)

  return { recentBurst, isCollectible, burstCount }
}

/** Whether a message counts as media for burst purposes. */
function isBurstMedia(message: Message): boolean {
  return (BURST_MEDIA_TYPES as readonly string[]).includes(message.message_type)
}
