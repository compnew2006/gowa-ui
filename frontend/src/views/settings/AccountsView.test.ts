/**
 * Task 12: every account status renders from one enum
 * (Connected / Disconnected / Reconnecting) with a badge.
 *
 * Mounts AccountsView with mocked services and asserts at the rendered
 * level:
 *  - an account with no live probe and the legacy stored status "active"
 *    (the Arkan-6178 case) shows the Connected badge, never bare "active";
 *  - a reconnecting account gets its own badge;
 *  - the header summary counts agree with the badges.
 *
 * $t is mocked to the identity function, so assertions use i18n keys.
 * The DataTable is stubbed to render only the status cell slot per row —
 * that is the cell under test.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const t = (key: string) => key

const apiGetMock = vi.fn()
const fetchServersMock = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => reactive({ params: {}, query: {} }),
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: {
    name: 'RouterLink',
    props: ['to'],
    template: '<a><slot /></a>',
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t }),
}))

vi.mock('vue-sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

vi.mock('lucide-vue-next', () => ({
  Phone: { name: 'Phone', template: '<span />' },
  Pencil: { name: 'Pencil', template: '<span />' },
  Trash2: { name: 'Trash2', template: '<span />' },
  Check: { name: 'Check', template: '<span />' },
  CheckCircle2: { name: 'CheckCircle2', template: '<span />' },
  Wifi: { name: 'Wifi', template: '<span />' },
  WifiOff: { name: 'WifiOff', template: '<span />' },
  Server: { name: 'Server', template: '<span />' },
  Link2: { name: 'Link2', template: '<span />' },
  Plus: { name: 'Plus', template: '<span />' },
}))

vi.mock('@/components/ui/card', () => ({
  Card: { name: 'Card', template: '<div><slot /></div>' },
  CardContent: { name: 'CardContent', template: '<div><slot /></div>' },
  CardDescription: { name: 'CardDescription', template: '<div><slot /></div>' },
  CardHeader: { name: 'CardHeader', template: '<div><slot /></div>' },
  CardTitle: { name: 'CardTitle', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/scroll-area', () => ({
  ScrollArea: { name: 'ScrollArea', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/button', () => ({
  Button: { name: 'Button', template: '<button @click="$emit(\'click\', $event)"><slot /></button>' },
}))

vi.mock('@/components/ui/badge', () => ({
  Badge: { name: 'Badge', template: '<span class="badge"><slot /></span>' },
}))

vi.mock('@/components/ui/tooltip', () => ({
  Tooltip: { name: 'Tooltip', template: '<div><slot /></div>' },
  TooltipContent: { name: 'TooltipContent', template: '<div><slot /></div>' },
  TooltipTrigger: { name: 'TooltipTrigger', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/popover', () => ({
  Popover: { name: 'Popover', template: '<div><slot /></div>' },
  PopoverContent: { name: 'PopoverContent', template: '<div><slot /></div>' },
  PopoverTrigger: { name: 'PopoverTrigger', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/tabs', () => ({
  Tabs: { name: 'Tabs', template: '<div><slot /></div>' },
  TabsContent: { name: 'TabsContent', template: '<div><slot /></div>' },
  TabsList: { name: 'TabsList', template: '<div><slot /></div>' },
  TabsTrigger: { name: 'TabsTrigger', template: '<div><slot /></div>' },
}))

vi.mock('@/components/shared', () => ({
  // PageHeader renders the actions slot too: the connection-status popover
  // (and its badges) live in #actions.
  PageHeader: { name: 'PageHeader', template: '<div><slot /><slot name="actions" /></div>' },
  DataTable: {
    name: 'DataTable',
    props: ['items', 'columns', 'isLoading', 'sortKey', 'sortDirection'],
    template:
      '<div><div v-for="item in items" :key="item.id" class="row"><slot name="cell-status" :item="item" /></div></div>',
  },
  DeleteConfirmDialog: { name: 'DeleteConfirmDialog', template: '<div />' },
  ErrorState: { name: 'ErrorState', template: '<div><slot /></div>' },
}))

vi.mock('@/services/api', () => ({
  api: {
    get: (...args: unknown[]) => apiGetMock(...args),
    delete: vi.fn(),
  },
}))

vi.mock('@/stores/organizations', () => ({
  useOrganizationsStore: () => reactive({ selectedOrgId: ref(null) }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () =>
    reactive({
      hasPermission: () => true,
    }),
}))

vi.mock('@/stores/gowaServers', () => ({
  useGowaServersStore: () =>
    reactive({
      servers: [],
      loading: false,
      fetchServers: fetchServersMock,
    }),
}))

vi.mock('@/lib/api-utils', () => ({
  getErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('@/lib/utils', () => ({
  formatDate: (value: string) => value,
}))

import AccountsView from './AccountsView.vue'

const accountsResponse = () => ({
  data: {
    data: {
      accounts: [
        {
          id: '1',
          name: 'Arkan-6178',
          status: 'active',
          gowa_connected: null,
          gowa_base_url: 'http://gowa.local:3000',
          gowa_device_id: 'device-1',
          created_at: '2026-11-21T00:00:00Z',
        },
        {
          id: '2',
          name: 'Arkan-2',
          status: 'reconnecting',
          gowa_connected: null,
          gowa_base_url: 'http://gowa.local:3000',
          gowa_device_id: 'device-2',
          created_at: '2026-11-21T00:00:00Z',
        },
        {
          id: '3',
          name: 'Arkan-3',
          status: 'disconnected',
          gowa_connected: false,
          gowa_base_url: 'http://gowa.local:3000',
          gowa_device_id: 'device-3',
          created_at: '2026-11-21T00:00:00Z',
        },
      ],
    },
  },
})

describe('AccountsView status badges (Task 12)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiGetMock.mockResolvedValue(accountsResponse())
    fetchServersMock.mockResolvedValue(undefined)
  })

  it('badges every status from the canonical enum, never raw stored text', async () => {
    const wrapper = mount(AccountsView, {
      global: {
        mocks: { $t: t },
      },
    })
    await flushPromises()

    const text = wrapper.text()
    // The legacy bare "active" must be impossible to see anywhere. (The
    // reconnecting badge's i18n key contains "reconnecting", so that label
    // is pinned by the exact badge counts below instead.)
    expect(text).not.toContain('active')

    // Each account renders a badge: once in the header popover list and
    // once in the table status cell.
    const badges = wrapper.findAll('.badge').map((badge) => badge.text())
    expect(badges.filter((label) => label === 'accounts.connected')).toHaveLength(2)
    expect(badges.filter((label) => label === 'accounts.reconnecting')).toHaveLength(2)
    expect(badges.filter((label) => label === 'accounts.disconnected')).toHaveLength(2)

    // Header summary agrees with the badges: Arkan-6178 counts as
    // connected via its normalized stored status.
    expect(text).toContain('1/3')
  })
})
