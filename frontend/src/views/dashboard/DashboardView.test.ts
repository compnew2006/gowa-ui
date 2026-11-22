/**
 * Task 8: Dashboard charts + Recent Messages.
 *
 * Mounts DashboardView with mocked services (the view is a thin shell; the
 * mocks below mirror the real service shapes) and asserts the three
 * Task 8 behaviours at the rendered-template level:
 *  - a widget with dataSource `messages` renders a time-series chart fed by
 *    the messages series (chart_data), NOT the contacts series
 *    (data_points) — the Agent Comparison bug was exactly this kind of
 *    mislabelled series;
 *  - Recent Messages dedupes rows by message id;
 *  - contact names truncate with ellipsis and expose the full name on
 *    hover (title), and a message whose content is "." renders as-is.
 *
 * $t is mocked to the identity function, so assertions use i18n keys.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const t = (key: string) => key

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

// Chart.js components: stubbed to render their `data` prop as JSON so the
// test can pin exactly which series feeds the chart.
vi.mock('@/lib/charts', () => {
  const stub = (testId: string) => ({
    name: `Stub${testId}`,
    props: ['data', 'options'],
    template: `<div data-testid="${testId}">{{ JSON.stringify(data) }}</div>`,
  })
  return {
    Line: stub('line-chart'),
    Bar: stub('bar-chart'),
    Pie: stub('pie-chart'),
  }
})

vi.mock('grid-layout-plus', () => ({
  GridLayout: { name: 'GridLayout', template: '<div><slot /></div>' },
  GridItem: { name: 'GridItem', template: '<div><slot /></div>' },
}))

vi.mock('@/components/shared', () => ({
  DateRangePicker: { name: 'DateRangePicker', template: '<div />' },
}))

vi.mock('@/composables/useDateRange', () => ({
  useDateRange: () => ({
    selectedRange: ref('30days'),
    customDateRange: ref({ start: undefined, end: undefined }),
    isDatePickerOpen: ref(false),
    dateRange: ref({ from: '2026-11-01', to: '2026-11-22' }),
    formatDateRangeDisplay: ref(''),
    applyCustomRange: vi.fn(),
  }),
}))

vi.mock('@/composables/useAppToast', () => ({
  useAppToast: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () =>
    reactive({
      user: { id: 'user-me' },
      userSettings: {},
      isAuthenticated: true,
      hasPermission: () => true,
      restoreSession: vi.fn(),
    }),
}))

const baseWidget = (overrides: Record<string, unknown>) => ({
  id: 'w-1',
  name: 'Widget',
  description: '',
  data_source: 'messages',
  metric: 'count',
  field: '',
  filters: [],
  display_type: 'number',
  chart_type: '',
  group_by_field: '',
  show_change: true,
  color: 'blue',
  size: 'small',
  display_order: 1,
  grid_x: 0,
  grid_y: 0,
  grid_w: 6,
  grid_h: 5,
  config: {},
  is_shared: true,
  is_default: false,
  is_owner: true,
  created_by: 'user-me',
  created_at: '2026-11-01T00:00:00Z',
  updated_at: '2026-11-01T00:00:00Z',
  ...overrides,
})

const chartWidget = baseWidget({
  id: 'w-chart',
  name: 'Messages Over Time',
  display_type: 'chart',
  chart_type: 'line',
  grid_x: 0,
  grid_y: 0,
  grid_w: 6,
  grid_h: 5,
})

const recentWidget = baseWidget({
  id: 'w-recent',
  name: 'Recent Messages',
  display_type: 'table',
  grid_x: 0,
  grid_y: 5,
  grid_w: 6,
  grid_h: 8,
})

// Messages series (chart_data) vs a contacts-shaped decoy series
// (data_points): the chart must render the former, never the latter.
const widgetData = {
  'w-chart': {
    widget_id: 'w-chart',
    value: 46,
    change: 0,
    prev_value: 0,
    chart_data: [
      { label: 'Nov 01', value: 12 },
      { label: 'Nov 02', value: 34 },
    ],
    data_points: [{ label: 'contacts-decoy', value: 999 }],
  },
  'w-recent': {
    widget_id: 'w-recent',
    value: 0,
    change: 0,
    prev_value: 0,
    chart_data: [],
    data_points: [],
    table_rows: [
      {
        id: 'm-1',
        label: 'A very long contact name that should be truncated with ellipsis',
        sub_label: '.',
        status: 'read',
        direction: 'incoming',
        created_at: new Date().toISOString(),
      },
      // Same message id as the row above: must render only once.
      {
        id: 'm-1',
        label: 'Duplicate of m-1',
        sub_label: 'dup',
        status: 'read',
        direction: 'incoming',
        created_at: new Date().toISOString(),
      },
      {
        id: 'm-2',
        label: 'Second Contact',
        sub_label: 'hello',
        status: 'delivered',
        direction: 'outgoing',
        created_at: new Date().toISOString(),
      },
    ],
  },
}

vi.mock('@/services/api', () => ({
  widgetsService: {
    list: vi.fn(),
    getAllData: vi.fn(),
    getDataSources: vi.fn(),
    saveLayout: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

import DashboardView from './DashboardView.vue'
import { widgetsService } from '@/services/api'

const widgetsServiceMock = widgetsService as unknown as Record<string, ReturnType<typeof vi.fn>>

async function mountDashboard() {
  widgetsServiceMock.list.mockResolvedValue({
    data: { data: { widgets: [chartWidget, recentWidget] } },
  })
  widgetsServiceMock.getAllData.mockResolvedValue({
    data: { data: { data: widgetData } },
  })
  widgetsServiceMock.getDataSources.mockResolvedValue({
    data: { data_sources: [], metrics: [], display_types: [], operators: [] },
  })
  const wrapper = mount(DashboardView, {
    global: {
      mocks: { $t: t },
      stubs: { transition: false },
    },
  })
  await flushPromises()
  await nextTick()
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('DashboardView — Task 8 charts + Recent Messages', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('feeds the messages chart from the messages series (chart_data), not the contacts series', async () => {
    const wrapper = mountDashboard()
    const line = (await wrapper).find('[data-testid="line-chart"]')
    expect(line.exists()).toBe(true)
    const data = JSON.parse(line.text())
    expect(data.labels).toEqual(['Nov 01', 'Nov 02'])
    expect(data.datasets[0].data).toEqual([12, 34])
    expect(JSON.stringify(data)).not.toContain('contacts-decoy')
    expect(JSON.stringify(data)).not.toContain('999')
  })

  it('dedupes Recent Messages rows by message id', async () => {
    const wrapper = await mountDashboard()
    expect(wrapper.text()).not.toContain('Duplicate of m-1')
    expect(wrapper.text()).toContain('Second Contact')
    // The deduped first row still renders once.
    expect(wrapper.text()).toContain('A very long contact name that should be truncated with ellipsis')
  })

  it('truncates long contact names with ellipsis and shows the full name on hover', async () => {
    const wrapper = await mountDashboard()
    const nameEl = wrapper.find('p[title="A very long contact name that should be truncated with ellipsis"]')
    expect(nameEl.exists()).toBe(true)
    expect(nameEl.classes()).toContain('truncate')
  })

  it('renders a message whose content is "." as-is', async () => {
    const wrapper = await mountDashboard()
    const dots = wrapper.findAll('p').filter((p) => p.text() === '.')
    expect(dots.length).toBe(1)
  })
})
