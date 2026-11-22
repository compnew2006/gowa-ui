/**
 * Task 16 — agents may NOT delete contacts (decision recorded 2026-11-21
 * in the execution ledger: deletion stays admin/manager-only).
 *
 * The server already 403s agent deletes (see
 * internal/handlers/task16_delete_decision_test.go), but the Contacts list
 * rendered its per-row delete button for every viewer — the live check
 * (2026-11-21, salama@arkan.com) saw it enabled. The button must be hidden
 * for viewers without `contacts:delete`, like ContactDetailView,
 * AccountsView and the dashboard already do.
 *
 * Mounts ContactsView with mocked services and asserts at the rendered
 * level: without `contacts:delete` no delete button renders in a row;
 * with it, the button renders. $t is mocked to the identity function, so
 * assertions use i18n keys (the delete button's aria-label is
 * `common.delete`).
 */
import { describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const t = (key: string) => key

// Mutable permission set, swapped per test case.
const perms = vi.hoisted(() => ({ set: new Set<string>() }))

const contactsListMock = vi.fn()

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
  Plus: { name: 'Plus', template: '<span />' },
  Users: { name: 'Users', template: '<span />' },
  Pencil: { name: 'Pencil', template: '<span />' },
  Trash2: { name: 'Trash2', template: '<span />' },
  MessageSquare: { name: 'MessageSquare', template: '<span />' },
  Download: { name: 'Download', template: '<span />' },
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

vi.mock('@/components/ui/tag-badge', () => ({
  TagBadge: { name: 'TagBadge', template: '<span><slot /></span>' },
}))

vi.mock('@/components/ui/select', () => ({
  Select: { name: 'Select', template: '<div><slot /></div>' },
  SelectContent: { name: 'SelectContent', template: '<div><slot /></div>' },
  SelectItem: { name: 'SelectItem', template: '<div><slot /></div>' },
  SelectTrigger: { name: 'SelectTrigger', template: '<div><slot /></div>' },
  SelectValue: { name: 'SelectValue', template: '<span />' },
}))

vi.mock('@/components/shared', () => ({
  PageHeader: { name: 'PageHeader', template: '<div><slot /><slot name="actions" /></div>' },
  SearchInput: {
    name: 'SearchInput',
    props: ['modelValue'],
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  DataTable: {
    name: 'DataTable',
    props: ['items', 'columns', 'isLoading', 'sortKey', 'sortDirection'],
    template:
      '<div><div v-for="item in items" :key="item.id" class="row"><slot name="cell-actions" :item="item" /></div></div>',
  },
  DeleteConfirmDialog: { name: 'DeleteConfirmDialog', template: '<div />' },
  CreateContactDialog: { name: 'CreateContactDialog', template: '<div />' },
  ImportExportDialog: { name: 'ImportExportDialog', template: '<div />' },
  IconButton: {
    name: 'IconButton',
    props: ['label'],
    template: '<button :aria-label="label" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  ErrorState: { name: 'ErrorState', template: '<div><slot /></div>' },
}))

vi.mock('@/services/api', () => ({
  contactsService: {
    list: (...args: unknown[]) => contactsListMock(...args),
    delete: vi.fn(),
  },
  accountsService: {
    list: vi.fn().mockResolvedValue({ data: { accounts: [] } }),
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () =>
    reactive({
      hasPermission: (resource: string, action: string) => perms.set.has(`${resource}:${action}`),
    }),
}))

vi.mock('@/composables/useSearchPagination', () => ({
  useSearchPagination: () => ({
    searchQuery: ref(''),
    currentPage: ref(1),
    totalItems: ref(1),
    pageSize: 20,
    handlePageChange: vi.fn(),
  }),
}))

vi.mock('@/lib/api-utils', () => ({
  getErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('@/lib/utils', () => ({
  formatDate: (value: string) => value,
}))

import ContactsView from './ContactsView.vue'

const oneContact = () => ({
  data: {
    contacts: [
      {
        id: 'c1',
        phone_number: '966500000001',
        profile_name: 'Test Contact',
        name: '',
        whatsapp_account: '',
        tags: [],
        metadata: {},
        assigned_user_id: null,
        last_message_at: null,
        last_message_preview: '',
        unread_count: 0,
        created_at: '2026-11-21T00:00:00Z',
        updated_at: '2026-11-21T00:00:00Z',
      },
    ],
    total: 1,
  },
})

const mountView = async () => {
  const wrapper = mount(ContactsView, {
    global: {
      mocks: { $t: t },
    },
  })
  await flushPromises()
  return wrapper
}

describe('ContactsView row delete button (Task 16)', () => {
  it('hides the delete button for a viewer without contacts:delete (agent)', async () => {
    perms.set = new Set(['contacts:read', 'contacts.manage:read'])
    contactsListMock.mockResolvedValue(oneContact())

    const w = await mountView()

    // The row still renders (chat + edit actions), but no delete button.
    expect(w.findAll('.row').length).toBe(1)
    expect(w.find('button[aria-label="common.delete"]').exists()).toBe(false)
    expect(w.find('button[aria-label="contacts.openChat"]').exists()).toBe(true)
  })

  it('shows the delete button for a viewer with contacts:delete (admin/manager)', async () => {
    perms.set = new Set(['contacts:read', 'contacts.manage:read', 'contacts:delete'])
    contactsListMock.mockResolvedValue(oneContact())

    const wrapper = await mountView()

    expect(wrapper.findAll('.row').length).toBe(1)
    expect(wrapper.find('button[aria-label="common.delete"]').exists()).toBe(true)
  })
})
