import { computed, ref, type Ref } from 'vue'

/**
 * A media group represents consecutive incoming media messages
 * sent within a short time window (e.g., batch photo sends).
 */
export interface MediaGroup {
  /** ID of the first message in the group (used as group key) */
  id: string
  /** Ordered array of messages that belong to this group */
  messageIds: string[]
}

/** Message shape required by the grouping logic (subset of the full Message interface) */
interface GroupableMessage {
  id: string
  direction: 'incoming' | 'outgoing'
  message_type: string
  media_url?: string
  created_at: string
}

/** Media types eligible for grouping */
const GROUPABLE_TYPES = new Set(['image', 'video', 'document'])

/** localStorage key for the configurable grouping window */
export const MEDIA_GROUP_WINDOW_KEY = 'chat.mediaGroupWindowSeconds'

/** Default grouping window in seconds */
export const MEDIA_GROUP_WINDOW_DEFAULT = 60

/** Minimum number of messages required to form a group */
const MIN_GROUP_SIZE = 2

/**
 * Read the grouping window from localStorage, with validation.
 */
export function readMediaGroupWindow(): number {
  try {
    const stored = Number(localStorage.getItem(MEDIA_GROUP_WINDOW_KEY))
    if (Number.isFinite(stored) && stored >= 5 && stored <= 300) {
      return stored
    }
  } catch {
    // Ignore localStorage errors
  }
  return MEDIA_GROUP_WINDOW_DEFAULT
}

/**
 * Save the grouping window to localStorage.
 */
export function saveMediaGroupWindow(seconds: number) {
  const clamped = Math.min(300, Math.max(5, Math.round(seconds)))
  try {
    localStorage.setItem(MEDIA_GROUP_WINDOW_KEY, String(clamped))
  } catch {
    // Ignore localStorage errors
  }
}

/**
 * Composable that groups consecutive incoming media messages by timestamp proximity.
 *
 * @param messages - Reactive ref to the chronologically sorted messages array
 * @returns Computed helpers for querying group membership
 */
export function useMediaGroups(messages: Ref<GroupableMessage[]>) {
  /** Reactive grouping window (re-read on each evaluation) */
  const groupWindowSeconds = ref(readMediaGroupWindow())

  /**
   * Core computed: walks messages once and builds group data structures.
   * Returns a tuple: [groupMap, messageToGroupMap]
   */
  const groupData = computed(() => {
    const windowMs = groupWindowSeconds.value * 1000
    const groups = new Map<string, MediaGroup>()
    const messageToGroup = new Map<string, string>()

    if (!messages.value || messages.value.length === 0) {
      return { groups, messageToGroup }
    }

    let currentGroup: MediaGroup | null = null
    let lastTimestamp: number | null = null

    for (const msg of messages.value) {
      const isGroupable =
        msg.direction === 'incoming' &&
        GROUPABLE_TYPES.has(msg.message_type) &&
        !!msg.media_url

      if (!isGroupable) {
        // Non-groupable message breaks the current run
        if (currentGroup && currentGroup.messageIds.length >= MIN_GROUP_SIZE) {
          groups.set(currentGroup.id, currentGroup)
          for (const mid of currentGroup.messageIds) {
            messageToGroup.set(mid, currentGroup.id)
          }
        }
        currentGroup = null
        lastTimestamp = null
        continue
      }

      const msgTime = new Date(msg.created_at).getTime()

      if (
        currentGroup &&
        lastTimestamp !== null &&
        Math.abs(msgTime - lastTimestamp) <= windowMs
      ) {
        // Continue the current group
        currentGroup.messageIds.push(msg.id)
      } else {
        // Finalize previous group if it qualifies
        if (currentGroup && currentGroup.messageIds.length >= MIN_GROUP_SIZE) {
          groups.set(currentGroup.id, currentGroup)
          for (const mid of currentGroup.messageIds) {
            messageToGroup.set(mid, currentGroup.id)
          }
        }
        // Start a new potential group
        currentGroup = { id: msg.id, messageIds: [msg.id] }
      }

      lastTimestamp = msgTime
    }

    // Finalize last group
    if (currentGroup && currentGroup.messageIds.length >= MIN_GROUP_SIZE) {
      groups.set(currentGroup.id, currentGroup)
      for (const mid of currentGroup.messageIds) {
        messageToGroup.set(mid, currentGroup.id)
      }
    }

    return { groups, messageToGroup }
  })

  /** All recognized media groups */
  const mediaGroups = computed(() => groupData.value.groups)

  /** Get the group a message belongs to, or undefined */
  function getGroupForMessage(messageId: string): MediaGroup | undefined {
    const groupId = groupData.value.messageToGroup.get(messageId)
    if (!groupId) return undefined
    return groupData.value.groups.get(groupId)
  }

  /** Check whether a message is the first (leader) of its group */
  function isGroupLeader(messageId: string): boolean {
    const group = getGroupForMessage(messageId)
    return !!group && group.id === messageId
  }

  /** Check whether a message is the last of its group */
  function isGroupTail(messageId: string): boolean {
    const group = getGroupForMessage(messageId)
    if (!group) return false
    return group.messageIds[group.messageIds.length - 1] === messageId
  }

  /** Check whether a message belongs to any group */
  function isGroupMember(messageId: string): boolean {
    return groupData.value.messageToGroup.has(messageId)
  }

  /** Update the grouping window and re-trigger the computed */
  function setGroupWindow(seconds: number) {
    const clamped = Math.min(300, Math.max(5, Math.round(seconds)))
    groupWindowSeconds.value = clamped
    saveMediaGroupWindow(clamped)
  }

  return {
    mediaGroups,
    groupWindowSeconds,
    getGroupForMessage,
    isGroupLeader,
    isGroupTail,
    isGroupMember,
    setGroupWindow
  }
}
