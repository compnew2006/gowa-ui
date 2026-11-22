/**
 * Task 12: the switcher shows the current org's name from the same lists
 * the dropdown renders (organizations for super admins, myOrganizations
 * otherwise) — including the collapsed view, which used to show
 * "All Organizations" for a non-super-admin because selectedOrganization
 * is null for them.
 *
 * $t is mocked to the identity function, so assertions use i18n keys.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const t = (key: string) => key

const orgsMock = vi.hoisted(() => ({
  organizations: [] as Array<{ id: string; name: string }>,
  myOrganizations: [] as Array<{
    organization_id: string
    name: string
    slug: string
    role_name: string
    is_default: boolean
  }>,
  selectedOrgId: null as string | null,
  loading: false,
  error: null as string | null,
  get isMultiOrg() {
    return this.myOrganizations.length > 1
  },
  init: vi.fn(),
  fetchOrganizations: vi.fn(),
  fetchMyOrganizations: vi.fn(),
  selectOrganization: vi.fn(),
}))

const authMock = vi.hoisted(() => ({
  user: null as { id: string; organization_id: string; is_super_admin: boolean } | null,
  refreshUserData: vi.fn(),
  switchOrg: vi.fn(),
  hasPermission: () => true,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t }),
}))

vi.mock('vue-sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

vi.mock('lucide-vue-next', () => ({
  Building2: { name: 'Building2', template: '<span />' },
  Plus: { name: 'Plus', template: '<span />' },
  Loader2: { name: 'Loader2', template: '<span />' },
}))

vi.mock('@/components/ui/button', () => ({
  Button: { name: 'Button', template: '<button @click="$emit(\'click\', $event)"><slot /></button>' },
}))

vi.mock('@/components/ui/select', () => ({
  Select: {
    name: 'Select',
    props: ['modelValue'],
    template: '<div><slot /></div>',
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

vi.mock('@/components/ui/input', () => ({
  Input: {
    name: 'Input',
    props: ['modelValue'],
    template:
      '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}))

vi.mock('@/components/ui/dialog', () => ({
  Dialog: { name: 'Dialog', props: ['open'], template: '<div><slot /></div>' },
  DialogContent: { name: 'DialogContent', template: '<div><slot /></div>' },
  DialogDescription: { name: 'DialogDescription', template: '<div><slot /></div>' },
  DialogFooter: { name: 'DialogFooter', template: '<div><slot /></div>' },
  DialogHeader: { name: 'DialogHeader', template: '<div><slot /></div>' },
  DialogTitle: { name: 'DialogTitle', template: '<div><slot /></div>' },
}))

vi.mock('@/services/api', () => ({
  organizationsService: { create: vi.fn() },
}))

vi.mock('@/stores/organizations', () => ({
  useOrganizationsStore: () => orgsMock,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authMock,
}))

import OrganizationSwitcher from './OrganizationSwitcher.vue'

function mountSwitcher(props: { collapsed?: boolean } = {}) {
  return mount(OrganizationSwitcher, {
    props,
    global: { mocks: { $t: t } },
  })
}

describe('OrganizationSwitcher current org name (Task 12)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    orgsMock.organizations = []
    orgsMock.myOrganizations = []
    orgsMock.selectedOrgId = null
    orgsMock.loading = false
    orgsMock.error = null
    orgsMock.fetchOrganizations.mockResolvedValue(undefined)
    orgsMock.fetchMyOrganizations.mockResolvedValue(undefined)
    authMock.refreshUserData.mockResolvedValue(undefined)
    authMock.user = { id: 'user-1', organization_id: 'org-1', is_super_admin: false }
  })

  it('collapsed view shows the current org name for a multi-org regular user', async () => {
    orgsMock.myOrganizations = [
      { organization_id: 'org-1', name: 'Default Organization', slug: 'default', role_name: 'admin', is_default: true },
      { organization_id: 'org-2', name: 'Second Org', slug: 'second', role_name: 'admin', is_default: false },
    ]
    const wrapper = mountSwitcher({ collapsed: true })
    await flushPromises()

    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.attributes('title')).toBe('Default Organization')
  })

  it('collapsed view shows the selected org name for a super admin', async () => {
    authMock.user = { id: 'user-1', organization_id: 'org-1', is_super_admin: true }
    orgsMock.organizations = [
      { id: 'org-1', name: 'Default Organization' },
      { id: 'org-2', name: 'Second Org' },
    ]
    orgsMock.selectedOrgId = 'org-2'
    const wrapper = mountSwitcher({ collapsed: true })
    await flushPromises()

    expect(wrapper.find('button').attributes('title')).toBe('Second Org')
  })

  it('expanded view lists org names from the same source', async () => {
    orgsMock.myOrganizations = [
      { organization_id: 'org-1', name: 'Default Organization', slug: 'default', role_name: 'admin', is_default: true },
      { organization_id: 'org-2', name: 'Second Org', slug: 'second', role_name: 'admin', is_default: false },
    ]
    const wrapper = mountSwitcher()
    await flushPromises()

    expect(wrapper.text()).toContain('Default Organization')
    expect(wrapper.text()).toContain('Second Org')
  })
})
