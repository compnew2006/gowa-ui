/**
 * Task 11: AppLayout wiring — the navigation skeleton from
 * useRouteLoading() must actually render in the layout while a slow child
 * route chunk loads, and disappear once the route resolves.
 *
 * (The TDD RED for the loading state itself lives in
 * useRouteLoading.test.ts; this file covers the AppLayout wiring, which
 * the composable test cannot see.)
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import AppLayout from './AppLayout.vue'

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: null,
    isAuthenticated: false,
    hasPermission: () => true,
    refreshUserData: vi.fn(),
    logout: vi.fn(),
  }),
}))

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } })

const stubs = {
  GlobalSearch: { template: '<div />' },
  OrganizationSwitcher: { template: '<div />' },
  UserMenu: { template: '<div />' },
  ScrollArea: { template: '<div><slot /></div>' },
  ScrollToTop: { template: '<div />' },
  NavItem: { template: '<div />' },
}

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div>Home page</div>' } },
      {
        path: '/slow',
        component: () =>
          new Promise((resolve) => {
            setTimeout(() => resolve({ template: '<div>Slow page</div>' }), 400)
          }) as never,
      },
    ],
  })
}

describe('AppLayout navigation skeleton', () => {
  it('shows a skeleton while a slow child route loads, then the page', async () => {
    const router = makeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper = mount(AppLayout, {
      attachTo: document.body,
      global: { plugins: [router, i18n], stubs },
    })
    expect(wrapper.find('[data-testid="route-loading-skeleton"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Home page')

    const pending = router.push('/slow')
    await new Promise((resolve) => setTimeout(resolve, 250))
    await nextTick()
    expect(wrapper.find('[data-testid="route-loading-skeleton"]').exists()).toBe(true)

    await pending
    await nextTick()
    expect(wrapper.find('[data-testid="route-loading-skeleton"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Slow page')
  })
})
