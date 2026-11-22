import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'

/**
 * Tracks pending router navigations so the layout can show a skeleton
 * instead of a black screen while a lazily-loaded route chunk is still
 * arriving (1–2s on a slow connection).
 *
 * The skeleton only appears after SKELETON_DELAY_MS: instant navigations
 * (already-loaded chunks) resolve before the timer fires, so they never
 * flash it. The flag is cleared on every navigation outcome — success
 * (afterEach) or error (onError) — so a failed/aborted navigation can
 * never leave the skeleton stuck on screen.
 */
const SKELETON_DELAY_MS = 150

export function useRouteLoading() {
  const router = useRouter()
  const isNavigating = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined

  const clearTimer = () => {
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  const finish = () => {
    clearTimer()
    isNavigating.value = false
  }

  const removeBefore = router.beforeEach(() => {
    clearTimer()
    timer = setTimeout(() => {
      isNavigating.value = true
    }, SKELETON_DELAY_MS)
  })
  const removeAfter = router.afterEach(finish)
  const removeError = router.onError(finish)

  onBeforeUnmount(() => {
    clearTimer()
    removeBefore()
    removeAfter()
    removeError()
  })

  return { isNavigating }
}
