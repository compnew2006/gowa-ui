/**
 * Task 11: navigation must show a skeleton, not a black screen, while a
 * lazily-loaded route chunk is still loading (1-2s on a slow connection).
 *
 * useRouteLoading() tracks pending router navigations. The skeleton only
 * appears after a short delay, so instant navigations (already-loaded
 * chunks) never flash it.
 */
import { describe, expect, it } from 'vitest'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { useRouteLoading } from './useRouteLoading'

const Host = defineComponent({
  setup() {
    const { isNavigating } = useRouteLoading()
    return { isNavigating }
  },
  template: `<div>{{ isNavigating }}</div>`,
})

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div>Home</div>' } },
      { path: '/fast', component: { template: '<div>Fast</div>' } },
      {
        path: '/slow',
        // Simulates a lazily-loaded route chunk arriving after ~400ms.
        component: () =>
          new Promise((resolve) => {
            setTimeout(() => resolve({ template: '<div>Slow page</div>' }), 400)
          }) as never,
      },
    ],
  })
}

describe('useRouteLoading', () => {
  it('is true while a slow route chunk loads, false once it resolves', async () => {
    const router = makeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper: VueWrapper = mount(Host, { global: { plugins: [router] } })
    expect(wrapper.text()).toBe('false')

    const pending = router.push('/slow')
    // Past the skeleton delay (150ms), before the chunk resolves (400ms).
    await new Promise((resolve) => setTimeout(resolve, 250))
    await nextTick()
    expect(wrapper.text()).toBe('true')

    await pending
    await nextTick()
    expect(wrapper.text()).toBe('false')
  })

  it('never shows for an instant navigation (cached chunk)', async () => {
    const router = makeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper: VueWrapper = mount(Host, { global: { plugins: [router] } })
    await router.push('/fast')
    await nextTick()
    // Past the skeleton delay: the flag must never have flipped on.
    await new Promise((resolve) => setTimeout(resolve, 250))
    await nextTick()
    expect(wrapper.text()).toBe('false')
  })
})
