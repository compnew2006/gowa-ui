import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ScrollArea from './ScrollArea.vue'

function mountScrollArea(props: Record<string, unknown> = {}) {
  return mount(ScrollArea as any, {
    props,
    slots: { default: '<div style="height: 600px">tall content</div>' },
  })
}

describe('ScrollArea', () => {
  it('hides the scrollbar until hover/scroll by default (no visible scrollbar on mount)', () => {
    const wrapper = mountScrollArea()
    expect(wrapper.find('[data-state="visible"]').exists()).toBe(false)
  })

  it('renders an always-visible scrollbar with a visible thumb when type="always"', () => {
    const wrapper = mountScrollArea({ type: 'always' })
    const scrollbar = wrapper.find('[data-state="visible"]')
    expect(scrollbar.exists()).toBe(true)
    const thumb = wrapper.find('.rounded-full')
    expect(thumb.exists()).toBe(true)
    expect(thumb.classes()).toContain('bg-muted-foreground/50')
    expect(thumb.classes()).not.toContain('bg-border')
  })
})
