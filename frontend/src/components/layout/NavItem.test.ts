/**
 * Task 11: collapsed sidebar icons must expose a label.
 *
 * When the sidebar is collapsed the NavItem label is visually hidden
 * (md:sr-only), so a mouse/keyboard user sees only an icon. The item must
 * still expose its label on hover/focus (tooltip) and to assistive tech
 * (aria-label). Expanded items keep their visible text label.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createI18n } from 'vue-i18n'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { LayoutDashboard } from 'lucide-vue-next'
import NavItem from './NavItem.vue'
import { TooltipProvider } from '@/components/ui/tooltip'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: { 'nav.dashboard': 'Dashboard' } },
})

function makeRouter() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div>Home</div>' } },
      { path: '/dashboard', component: { template: '<div>Dashboard page</div>' } },
    ],
  })
  return router
}

const Host = defineComponent({
  components: { NavItem, TooltipProvider },
  props: { isCollapsed: { type: Boolean, default: true } },
  setup() {
    const item = {
      name: 'nav.dashboard',
      path: '/dashboard',
      icon: LayoutDashboard,
      active: false,
    }
    return { item }
  },
  template: `<TooltipProvider><NavItem :item="item" :is-collapsed="isCollapsed" /></TooltipProvider>`,
})

async function mountNav(isCollapsed: boolean): Promise<VueWrapper> {
  const router = makeRouter()
  await router.push('/')
  await router.isReady()
  return mount(Host, {
    props: { isCollapsed },
    attachTo: document.body,
    global: { plugins: [router, i18n] },
  })
}

describe('NavItem collapsed labels', () => {
  let wrapper: VueWrapper | null = null

  beforeEach(() => {
    wrapper = null
    document.body.innerHTML = ''
  })

  it('collapsed: the link exposes its label via aria-label', async () => {
    wrapper = await mountNav(true)
    const link = wrapper.find('a')
    expect(link.exists()).toBe(true)
    expect(link.attributes('aria-label')).toBe('Dashboard')
  })

  it('collapsed: focusing the icon shows a tooltip with the label', async () => {
    wrapper = await mountNav(true)
    const link = wrapper.find('a')
    await link.trigger('focus')
    await nextTick()
    await new Promise((resolve) => setTimeout(resolve, 100))
    const tooltip = document.body.querySelector('[role="tooltip"]')
    expect(tooltip?.textContent).toBe('Dashboard')
  })

  it('expanded: the label stays visible as text (no sr-only)', async () => {
    wrapper = await mountNav(false)
    const link = wrapper.find('a')
    expect(link.text()).toContain('Dashboard')
    expect(link.find('span').classes()).not.toContain('sr-only')
  })
})
