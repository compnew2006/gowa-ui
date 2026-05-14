import { ref, onUnmounted, type Ref } from "vue"

/**
 * Minimal resize-from-corner composable for fixed-position dialogs.
 * Handles pointer events, respects min dimensions, and cleans up on unmount.
 */
export function useResizable(
  target: Ref<HTMLElement | null>,
  opts: { minWidth?: number; minHeight?: number } = {},
) {
  const { minWidth = 280, minHeight = 200 } = opts
  const isResizing = ref(false)
  let startX = 0
  let startY = 0
  let startW = 0
  let startH = 0

  function onPointerDown(e: PointerEvent) {
    if (!target.value) return
    e.preventDefault()
    isResizing.value = true
    startX = e.clientX
    startY = e.clientY
    const rect = target.value.getBoundingClientRect()
    startW = rect.width
    startH = rect.height
    document.addEventListener("pointermove", onPointerMove)
    document.addEventListener("pointerup", onPointerUp)
  }

  function onPointerMove(e: PointerEvent) {
    if (!target.value || !isResizing.value) return
    const dw = e.clientX - startX
    const dh = e.clientY - startY
    target.value.style.width = `${Math.max(minWidth, startW + dw)}px`
    target.value.style.height = `${Math.max(minHeight, startH + dh)}px`
  }

  function onPointerUp() {
    isResizing.value = false
    document.removeEventListener("pointermove", onPointerMove)
    document.removeEventListener("pointerup", onPointerUp)
  }

  onUnmounted(() => {
    document.removeEventListener("pointermove", onPointerMove)
    document.removeEventListener("pointerup", onPointerUp)
  })

  return { isResizing, onPointerDown }
}
