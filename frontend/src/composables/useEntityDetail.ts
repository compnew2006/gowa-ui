import { ref, nextTick } from 'vue'
import { useUnsavedChangesGuard } from '@/composables/useUnsavedChangesGuard'

/**
 * Shared skeleton for entity detail views (load / not-found / saving / unsaved-changes).
 *
 * Owns the loading lifecycle refs and the unsaved-changes guard. The `load` helper
 * wraps a caller-supplied loader with the standard flow: flip isLoading on, clear
 * isNotFound, run the loader, reset hasChanges on the next tick (so the form watcher
 * fires first), and set isNotFound on failure. Each view keeps its own `watch` and
 * `onMounted` so per-view nuances (null-guards, extra fetches) stay explicit.
 */
export function useEntityDetail() {
  const isLoading = ref(true)
  const isNotFound = ref(false)
  const isSaving = ref(false)
  const hasChanges = ref(false)

  const { showLeaveDialog, confirmLeave, cancelLeave } = useUnsavedChangesGuard(hasChanges)

  async function load(loader: () => Promise<void>) {
    isLoading.value = true
    isNotFound.value = false
    try {
      await loader()
      // Reset after the loader so the form watcher doesn't leave hasChanges dirty.
      nextTick(() => { hasChanges.value = false })
    } catch {
      isNotFound.value = true
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading,
    isNotFound,
    isSaving,
    hasChanges,
    showLeaveDialog,
    confirmLeave,
    cancelLeave,
    load,
  }
}
