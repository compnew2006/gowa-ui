/**
 * Task 9: Audit Logs flooding and duplicates.
 *
 * Mounts AuditLogsView with mocked services and asserts the "hide routine
 * contact updates" filter at the rendered-template level:
 *  - the toggle is off by default, so the initial fetch lists everything
 *    (no history is hidden unless the user asks for it);
 *  - checking it re-fetches with hide_contact_updates: true;
 *  - unchecking it re-fetches without the flag.
 *
 * $t is mocked to the identity function, so assertions use i18n keys.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
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

vi.mock('lucide-vue-next', () => ({
  ScrollText: { name: 'ScrollText', template: '<span />' },
}))

vi.mock('@/components/ui/card', () => ({
  Card: { name: 'Card', template: '<div><slot /></div>' },
  CardContent: { name: 'CardContent', template: '<div><slot /></div>' },
  CardDescription: { name: 'CardDescription', template: '<div><slot /></div>' },
  CardHeader: { name: 'CardHeader', template: '<div><slot /></div>' },
  CardTitle: { name: 'CardTitle', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/badge', () => ({
  Badge: { name: 'Badge', template: '<span><slot /></span>' },
}))

vi.mock('@/components/ui/checkbox', () => ({
  Checkbox: {
    name: 'Checkbox',
    props: ['checked'],
    template:
      '<input type="checkbox" :checked="checked" @change="$emit(\'update:checked\', $event.target.checked)" />',
  },
}))

vi.mock('@/components/ui/scroll-area', () => ({
  ScrollArea: { name: 'ScrollArea', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/select', () => ({
  Select: { name: 'Select', template: '<div><slot /></div>' },
  SelectContent: { name: 'SelectContent', template: '<div><slot /></div>' },
  SelectItem: { name: 'SelectItem', template: '<div><slot /></div>' },
  SelectTrigger: { name: 'SelectTrigger', template: '<div><slot /></div>' },
  SelectValue: { name: 'SelectValue', template: '<div><slot /></div>' },
}))

vi.mock('@/components/shared', () => ({
  PageHeader: { name: 'PageHeader', template: '<div />' },
  DataTable: { name: 'DataTable', template: '<div />' },
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

vi.mock('@/stores/users', () => ({
  useUsersStore: () => ({ users: [], fetchUsers: vi.fn().mockResolvedValue(undefined) }),
}))

vi.mock('@/services/api', () => ({
  auditLogsService: {
    list: vi.fn().mockResolvedValue({
      data: { data: { audit_logs: [], total: 0 } },
    }),
  },
}))

import { auditLogsService } from '@/services/api'
import AuditLogsView from './AuditLogsView.vue'

describe('AuditLogsView — hide routine contact updates', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  async function mountView() {
    const wrapper = mount(AuditLogsView, {
      global: {
        stubs: {
          RouterLink: { name: 'RouterLink', props: ['to'], template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()
    return wrapper
  }

  // setChecked is private on DOMWrapper in this test-utils version, so flip
  // the input's checked state directly and fire the change the Checkbox
  // stub listens for.
  async function setToggle(wrapper: any, checked: boolean) {
    const checkbox = wrapper.find('input[type="checkbox"]')
    expect(checkbox.exists()).toBe(true)
    ;(checkbox.element as HTMLInputElement).checked = checked
    await checkbox.trigger('change')
    await flushPromises()
  }

  it('fetches everything by default (toggle off)', async () => {
    await mountView()
    expect(auditLogsService.list).toHaveBeenCalled()
    const params = (auditLogsService.list as any).mock.calls[0][0]
    expect(params.hide_contact_updates).toBeUndefined()
  })

  it('checking the toggle re-fetches with hide_contact_updates: true', async () => {
    const wrapper = await mountView()
    await setToggle(wrapper, true)

    const calls = (auditLogsService.list as any).mock.calls
    expect(calls.length).toBeGreaterThan(1)
    expect(calls[calls.length - 1][0].hide_contact_updates).toBe(true)
  })

  it('unchecking the toggle re-fetches without the flag', async () => {
    const wrapper = await mountView()
    await setToggle(wrapper, true)
    await setToggle(wrapper, false)

    const calls = (auditLogsService.list as any).mock.calls
    expect(calls[calls.length - 1][0].hide_contact_updates).toBeUndefined()
  })
})
