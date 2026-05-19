import { ref, onUnmounted, type ComponentPublicInstance, type Ref } from "vue"

type ResizableTarget = HTMLElement | ComponentPublicInstance | null

function resolveResizableElement(target: ResizableTarget): HTMLElement | null {
  if (typeof HTMLElement !== "undefined" && target instanceof HTMLElement) {
    return target
  }

  const maybeElement = (target as ComponentPublicInstance | null)?.$el
  if (typeof HTMLElement !== "undefined" && maybeElement instanceof HTMLElement) {
    return maybeElement
  }

  return null
}

/**
 * Minimal resize-from-corner composable for fixed-position dialogs.
 * Handles pointer events, respects min dimensions, and cleans up on unmount.
 */
export function useResizable(
  target: Ref<ResizableTarget>,
  opts: { minWidth?: number; minHeight?: number } = {},
) {
  const { minWidth = 280, minHeight = 200 } = opts
  const isResizing = ref(false)
  let startX = 0
  let startY = 0
  let startW = 0
  let startH = 0

  function onPointerDown(e: PointerEvent) {
    const element = resolveResizableElement(target.value)
    if (!element) return
    e.preventDefault()
    isResizing.value = true
    startX = e.clientX
    startY = e.clientY
    const rect = element.getBoundingClientRect()
    startW = rect.width
    startH = rect.height
    document.addEventListener("pointermove", onPointerMove)
    document.addEventListener("pointerup", onPointerUp)
  }

  function onPointerMove(e: PointerEvent) {
    const element = resolveResizableElement(target.value)
    if (!element || !isResizing.value) return
    const dw = e.clientX - startX
    const dh = e.clientY - startY
    element.style.width = `${Math.max(minWidth, startW + dw)}px`
    element.style.height = `${Math.max(minHeight, startH + dh)}px`
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
