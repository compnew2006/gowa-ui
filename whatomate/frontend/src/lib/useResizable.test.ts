// @vitest-environment happy-dom

import { describe, expect, it } from "vitest"
import { ref, type ComponentPublicInstance } from "vue"

import { useResizable } from "./useResizable"

function pointerEvent(type: string, clientX: number, clientY: number): PointerEvent {
  return new MouseEvent(type, {
    bubbles: true,
    cancelable: true,
    clientX,
    clientY,
  }) as PointerEvent
}

describe("useResizable", () => {
  it("resizes component refs by using their root element", () => {
    const element = document.createElement("div")
    element.getBoundingClientRect = () =>
      ({
        width: 320,
        height: 240,
        x: 0,
        y: 0,
        top: 0,
        right: 320,
        bottom: 240,
        left: 0,
        toJSON: () => ({}),
      }) as DOMRect

    const componentRef = ref({
      $el: element,
    } as ComponentPublicInstance)
    const { onPointerDown } = useResizable(componentRef, {
      minWidth: 280,
      minHeight: 200,
    })

    onPointerDown(pointerEvent("pointerdown", 10, 20))
    document.dispatchEvent(pointerEvent("pointermove", 60, 80))
    document.dispatchEvent(pointerEvent("pointerup", 60, 80))

    expect(element.style.width).toBe("370px")
    expect(element.style.height).toBe("300px")
  })

  it("ignores non-element component refs without throwing", () => {
    const componentRef = ref({
      $el: null,
    } as ComponentPublicInstance)
    const { onPointerDown } = useResizable(componentRef)

    expect(() => onPointerDown(pointerEvent("pointerdown", 10, 20))).not.toThrow()
  })
})
