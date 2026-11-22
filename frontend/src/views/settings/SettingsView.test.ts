/**
 * Task 12: org name / timezone — one source (the server response).
 *
 * Mounts SettingsView with mocked services and asserts at the rendered
 * level:
 *  - the General tab shows the name and timezone the server returned
 *    (including a stored timezone outside the static dropdown list);
 *  - a failing user-settings fetch does not wipe the org settings;
 *  - a failing org-settings fetch shows an error, never the hardcoded
 *    "My Organization" / "UTC" defaults;
 *  - saving applies the server's update response and refreshes the org
 *    store, so the switcher shows the renamed org.
 *
 * $t is mocked to the identity function, so assertions use i18n keys.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const t = (key: string) => key

const getSettingsMock = vi.fn()
const updateSettingsMock = vi.fn()
const meMock = vi.fn()
const fetchMyOrganizationsMock = vi.fn()
const fetchOrganizationsMock = vi.fn()

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
  Settings: { name: 'Settings', template: '<span />' },
  Bell: { name: 'Bell', template: '<span />' },
  Loader2: { name: 'Loader2', template: '<span />' },
  Globe: { name: 'Globe', template: '<span />' },
}))

vi.mock('@/components/ui/button', () => ({
  Button: { name: 'Button', template: '<button @click="$emit(\'click\', $event)"><slot /></button>' },
}))

vi.mock('@/components/ui/input', () => ({
  Input: {
    name: 'Input',
    props: ['modelValue'],
    template:
      '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}))

vi.mock('@/components/ui/label', () => ({
  Label: { name: 'Label', template: '<label><slot /></label>' },
}))

vi.mock('@/components/ui/scroll-area', () => ({
  ScrollArea: { name: 'ScrollArea', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/separator', () => ({
  Separator: { name: 'Separator', template: '<div />' },
}))

vi.mock('@/components/ui/switch', () => ({
  Switch: {
    name: 'Switch',
    props: ['checked'],
    template:
      '<input type="checkbox" :checked="checked" @change="$emit(\'update:checked\', $event.target.checked)" />',
  },
}))

vi.mock('@/components/ui/tabs', () => ({
  Tabs: { name: 'Tabs', template: '<div><slot /></div>' },
  TabsContent: { name: 'TabsContent', template: '<div><slot /></div>' },
  TabsList: { name: 'TabsList', template: '<div><slot /></div>' },
  TabsTrigger: { name: 'TabsTrigger', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/select', () => ({
  Select: {
    name: 'Select',
    props: ['modelValue'],
    template: '<div data-testid="tz-select" :data-value="modelValue"><slot /></div>',
  },
  SelectContent: { name: 'SelectContent', template: '<div><slot /></div>' },
  SelectItem: {
    name: 'SelectItem',
    props: ['value'],
    template: '<div class="select-item" :data-value="value"><slot /></div>',
  },
  SelectTrigger: { name: 'SelectTrigger', template: '<div><slot /></div>' },
  SelectValue: { name: 'SelectValue', template: '<div><slot /></div>' },
}))

vi.mock('@/components/shared', () => ({
  PageHeader: { name: 'PageHeader', template: '<div />' },
  AuditLogPanel: { name: 'AuditLogPanel', template: '<div />' },
}))

vi.mock('@/components/LanguageSwitcher.vue', () => ({
  default: { name: 'LanguageSwitcher', template: '<div />' },
}))

vi.mock('@/services/api', () => ({
  organizationService: {
    getSettings: (...args: unknown[]) => getSettingsMock(...args),
    updateSettings: (...args: unknown[]) => updateSettingsMock(...args),
  },
  usersService: {
    me: (...args: unknown[]) => meMock(...args),
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () =>
    reactive({
      user: { id: 'user-1', is_super_admin: false },
      organizationId: 'org-1',
      hasPermission: () => true,
    }),
}))

vi.mock('@/stores/organizations', () => ({
  useOrganizationsStore: () => ({
    fetchMyOrganizations: fetchMyOrganizationsMock,
    fetchOrganizations: fetchOrganizationsMock,
  }),
}))

import SettingsView from './SettingsView.vue'

const orgEnvelope = (name: string, timezone: string) => ({
  data: {
    status: 'success',
    data: {
      name,
      settings: { timezone, date_format: 'YYYY-MM-DD', mask_phone_numbers: false },
    },
  },
})

const userEnvelope = () => ({
  data: { data: { settings: { email_notifications: true } } },
})

function mountView() {
  return mount(SettingsView, {
    global: {
      mocks: { $t: t },
      stubs: { transition: false },
    },
  })
}

function timezoneSelect(wrapper: ReturnType<typeof mountView>) {
  const selects = wrapper.findAllComponents({ name: 'Select' })
  expect(selects.length).toBeGreaterThan(0)
  return selects[0]
}

describe('SettingsView general settings (Task 12)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getSettingsMock.mockResolvedValue(orgEnvelope('Default Organization', 'Africa/Cairo'))
    meMock.mockResolvedValue(userEnvelope())
    updateSettingsMock.mockResolvedValue(
      orgEnvelope('Renamed Organization', 'Europe/London'),
    )
    fetchMyOrganizationsMock.mockResolvedValue(undefined)
    fetchOrganizationsMock.mockResolvedValue(undefined)
  })

  it('shows a loading state first, then the server name and timezone', async () => {
    const wrapper = mountView()
    expect(wrapper.text()).toContain('common.loading')

    await flushPromises()

    const nameInput = wrapper.find('#org_name')
    expect(nameInput.exists()).toBe(true)
    expect((nameInput.element as HTMLInputElement).value).toBe('Default Organization')
    // Africa/Cairo is not in the static dropdown list, yet the stored value
    // must be shown — as the selected value and as an option.
    expect(timezoneSelect(wrapper).props('modelValue')).toBe('Africa/Cairo')
    const optionValues = wrapper
      .findAll('.select-item')
      .map((item) => item.attributes('data-value'))
    expect(optionValues).toContain('Africa/Cairo')
    expect(wrapper.text()).not.toContain('My Organization')
  })

  it('a failing user-settings fetch does not wipe the org settings', async () => {
    meMock.mockRejectedValue(new Error('me failed'))
    const wrapper = mountView()
    await flushPromises()

    const nameInput = wrapper.find('#org_name')
    expect((nameInput.element as HTMLInputElement).value).toBe('Default Organization')
    expect(timezoneSelect(wrapper).props('modelValue')).toBe('Africa/Cairo')
  })

  it('a failing org-settings fetch shows an error, never fake defaults', async () => {
    getSettingsMock.mockRejectedValue(new Error('org failed'))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('common.loadErrorTitle')
    // The form is gated: no input showing a fabricated name or timezone.
    expect(wrapper.find('#org_name').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('My Organization')
    expect(wrapper.text()).not.toContain('UTC')
  })

  it('save applies the server response and refreshes the org store', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#org_name').setValue('Renamed Organization')

    const saveButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('settings.save'))
    expect(saveButton).toBeDefined()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(updateSettingsMock).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Renamed Organization' }),
    )
    // The form now reflects the server's stored values, not the local input.
    expect((wrapper.find('#org_name').element as HTMLInputElement).value).toBe(
      'Renamed Organization',
    )
    expect(timezoneSelect(wrapper).props('modelValue')).toBe('Europe/London')
    // The switcher reads names from the org store: it must be refreshed so
    // the renamed org shows up there too.
    expect(fetchMyOrganizationsMock).toHaveBeenCalled()
    expect(fetchOrganizationsMock).not.toHaveBeenCalled()
  })
})
