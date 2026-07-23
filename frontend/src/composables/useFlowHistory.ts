import { ref, computed } from 'vue'
import type { SimulationSnapshot, SimulationMessage } from '@/types/flow-preview'

const MAX_HISTORY_SIZE = 50

export function useFlowHistory() {
  const history = ref<SimulationSnapshot[]>([])
  const historyIndex = ref(-1)

  const canUndo = computed(() => historyIndex.value > 0)
  const canRedo = computed(() => historyIndex.value < history.value.length - 1)

  const currentSnapshot = computed(() => {
    if (historyIndex.value >= 0 && historyIndex.value < history.value.length) {
      return history.value[historyIndex.value]
    }
    return null
  })

  /**
   * Save a snapshot of the current state
   */
  function saveSnapshot(
    stepIndex: number,
    stepName: string,
    variables: Record<string, any>,
    messages: SimulationMessage[],
    retryCount: number = 0
  ): void {
    const snapshot: SimulationSnapshot = {
      stepIndex,
      stepName,
      variables: { ...variables },
      messages: messages.map(m => ({ ...m })),
      retryCount,
      timestamp: new Date()
    }

    // If we're not at the end, truncate future history
    if (historyIndex.value < history.value.length - 1) {
      history.value = history.value.slice(0, historyIndex.value + 1)
    }

    // Add new snapshot
    history.value.push(snapshot)

    // Limit history size
    if (history.value.length > MAX_HISTORY_SIZE) {
      history.value = history.value.slice(-MAX_HISTORY_SIZE)
    }

    historyIndex.value = history.value.length - 1
  }

  /**
   * Go back to previous state
   */
  function undo(): SimulationSnapshot | null {
    if (!canUndo.value) return null

    historyIndex.value--
    return currentSnapshot.value
  }

  /**
   * Go forward to next state
   */
  function redo(): SimulationSnapshot | null {
    if (!canRedo.value) return null

    historyIndex.value++
    return currentSnapshot.value
  }

  /**
   * Jump to a specific point in history
   */
  function goToSnapshot(index: number): SimulationSnapshot | null {
    if (index < 0 || index >= history.value.length) return null

    historyIndex.value = index
    return currentSnapshot.value
  }

  /**
   * Clear all history
   */
  function clearHistory(): void {
    history.value = []
    historyIndex.value = -1
  }

  /**
   * Get history entries for display (e.g., in a timeline)
   */
  function getHistoryEntries(): Array<{
    index: number
    stepName: string
    timestamp: Date
    isCurrent: boolean
  }> {
    return history.value.map((snapshot, index) => ({
      index,
      stepName: snapshot.stepName,
      timestamp: snapshot.timestamp,
      isCurrent: index === historyIndex.value
    }))
  }

  return {
    history,
    historyIndex,
    canUndo,
    canRedo,
    currentSnapshot,
    saveSnapshot,
    undo,
    redo,
    goToSnapshot,
    clearHistory,
    getHistoryEntries
  }
}
