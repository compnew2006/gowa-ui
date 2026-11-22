/**
 * Task 13: teaching empty states on the settings pages + SSO.
 *
 * Mounts each settings list view with mocked services/stores returning an
 * empty list and asserts at the rendered level (DataTable is stubbed to
 * render its empty-title / empty-description props and the #empty-action
 * slot, exactly the pattern the views already use):
 *  - the true-empty state explains what the feature is for (its description
 *    key) and offers exactly one action to create the first item;
 *  - a search with no match shows the no-matching copy instead and offers
 *    NO create action (the list is not empty, the query just matches
 *    nothing);
 *  - SSO shows which of the 5 providers is configured vs not, and with
 *    none configured it teaches SSO and offers one action to set up the
 *    first provider (reusing the existing setup dialog, no new flow).
 *
 * $t is mocked to the identity function, so rendered assertions use i18n
 * keys. The teaching copy itself is pinned by value in the locale-content
 * describe block (the keys stay the same, the wording is what changes).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import en from '@/i18n/locales/en.json'
import ar from '@/i18n/locales/ar.json'

const t = (key: string) => key

const cannedListMock = vi.fn()
const apiKeysListMock = vi.fn()
const webhooksListMock = vi.fn()
const customActionsListMock = vi.fn()
const apiGetMock = vi.fn()
const fetchTagsMock = vi.fn()
const fetchTemplatesMock = vi.fn()
const fetchTeamsMock = vi.fn()

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
  MessageSquareText: { name: 'MessageSquareText', template: '<span />' },
  Pencil: { name: 'Pencil', template: '<span />' },
  Trash2: { name: 'Trash2', template: '<span />' },
  Copy: { name: 'Copy', template: '<span />' },
  Check: { name: 'Check', template: '<span />' },
  Tags: { name: 'Tags', template: '<span />' },
  LayoutTemplate: { name: 'LayoutTemplate', template: '<span />' },
  Users: { name: 'Users', template: '<span />' },
  RotateCcw: { name: 'RotateCcw', template: '<span />' },
  Scale: { name: 'Scale', template: '<span />' },
  Hand: { name: 'Hand', template: '<span />' },
  Key: { name: 'Key', template: '<span />' },
  Webhook: { name: 'Webhook', template: '<span />' },
  Play: { name: 'Play', template: '<span />' },
  Zap: { name: 'Zap', template: '<span />' },
  Loader2: { name: 'Loader2', template: '<span />' },
  Globe: { name: 'Globe', template: '<span />' },
  Code: { name: 'Code', template: '<span />' },
  Ticket: { name: 'Ticket', template: '<span />' },
  User: { name: 'User', template: '<span />' },
  BarChart: { name: 'BarChart', template: '<span />' },
  Link: { name: 'Link', template: '<span />' },
  Phone: { name: 'Phone', template: '<span />' },
  Mail: { name: 'Mail', template: '<span />' },
  FileText: { name: 'FileText', template: '<span />' },
  ExternalLink: { name: 'ExternalLink', template: '<span />' },
  ShieldCheck: { name: 'ShieldCheck', template: '<span />' },
  Settings2: { name: 'Settings2', template: '<span />' },
  Info: { name: 'Info', template: '<span />' },
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

vi.mock('@/components/ui/textarea', () => ({
  Textarea: { name: 'Textarea', template: '<textarea><slot /></textarea>' },
}))

vi.mock('@/components/ui/switch', () => ({
  Switch: {
    name: 'Switch',
    props: ['checked'],
    template: '<button><slot /></button>',
  },
}))

vi.mock('@/components/ui/select', () => ({
  Select: { name: 'Select', props: ['modelValue', 'defaultValue'], template: '<div><slot /></div>' },
  SelectContent: { name: 'SelectContent', template: '<div><slot /></div>' },
  SelectItem: { name: 'SelectItem', props: ['value', 'textValue'], template: '<div><slot /></div>' },
  SelectTrigger: { name: 'SelectTrigger', template: '<div><slot /></div>' },
  SelectValue: { name: 'SelectValue', props: ['placeholder'], template: '<span><slot /></span>' },
}))

vi.mock('@/components/ui/dialog', () => ({
  Dialog: {
    name: 'Dialog',
    props: { open: { type: Boolean, default: false } },
    template: '<div v-if="open"><slot /></div>',
  },
  DialogContent: { name: 'DialogContent', template: '<div><slot /></div>' },
  DialogDescription: { name: 'DialogDescription', template: '<div><slot /></div>' },
  DialogFooter: { name: 'DialogFooter', template: '<div><slot /></div>' },
  DialogHeader: { name: 'DialogHeader', template: '<div><slot /></div>' },
  DialogTitle: { name: 'DialogTitle', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/tooltip', () => ({
  Tooltip: { name: 'Tooltip', template: '<div><slot /></div>' },
  TooltipContent: { name: 'TooltipContent', template: '<div><slot /></div>' },
  TooltipTrigger: { name: 'TooltipTrigger', template: '<div><slot /></div>' },
}))

vi.mock('@/components/ui/radio-group', () => ({
  RadioGroup: { name: 'RadioGroup', props: ['modelValue'], template: '<div><slot /></div>' },
  RadioGroupItem: { name: 'RadioGroupItem', props: ['value', 'id'], template: '<span />' },
}))

vi.mock('@/components/ui/tag-badge', () => ({
  TagBadge: { name: 'TagBadge', template: '<span><slot /></span>' },
}))

vi.mock('@/components/ui/empty-state', () => ({
  EmptyState: {
    name: 'EmptyState',
    props: ['title', 'description', 'icon'],
    template:
      '<div class="empty-state"><h3>{{ title }}</h3><p>{{ description }}</p><div class="empty-action"><slot name="action" /></div></div>',
  },
}))

vi.mock('@/components/shared', () => ({
  PageHeader: { name: 'PageHeader', template: '<div><slot /><slot name="actions" /></div>' },
  SearchInput: {
    name: 'SearchInput',
    props: ['modelValue', 'placeholder'],
    template:
      '<input class="search-input" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  DataTable: {
    name: 'DataTable',
    props: ['items', 'columns', 'isLoading', 'emptyTitle', 'emptyDescription', 'sortKey', 'sortDirection'],
    template:
      '<div><div v-if="items.length === 0" class="empty-state"><p class="empty-title">{{ emptyTitle }}</p><p class="empty-description">{{ emptyDescription }}</p><div class="empty-action"><slot name="empty-action" /></div></div><div v-else class="rows"><div v-for="item in items" :key="item.id" class="row">{{ item.name || item.id }}</div></div></div>',
  },
  DeleteConfirmDialog: { name: 'DeleteConfirmDialog', template: '<div />' },
  ConfirmDialog: { name: 'ConfirmDialog', template: '<div />' },
  CrudFormDialog: { name: 'CrudFormDialog', template: '<div />' },
  IconButton: { name: 'IconButton', template: '<button @click="$emit(\'click\', $event)"><slot /></button>' },
  ErrorState: { name: 'ErrorState', template: '<div><slot /></div>' },
}))

vi.mock('@/services/api', () => ({
  api: {
    get: (...args: unknown[]) => apiGetMock(...args),
    put: vi.fn(),
    delete: vi.fn(),
  },
  cannedResponsesService: { list: (...args: unknown[]) => cannedListMock(...args), delete: vi.fn() },
  apiKeysService: { list: (...args: unknown[]) => apiKeysListMock(...args), delete: vi.fn(), update: vi.fn() },
  webhooksService: { list: (...args: unknown[]) => webhooksListMock(...args), delete: vi.fn(), update: vi.fn(), test: vi.fn() },
  customActionsService: { list: (...args: unknown[]) => customActionsListMock(...args), delete: vi.fn(), update: vi.fn(), create: vi.fn() },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () =>
    reactive({
      hasPermission: () => true,
    }),
}))

vi.mock('@/stores/organizations', () => ({
  useOrganizationsStore: () => reactive({ selectedOrgId: ref(null) }),
}))

vi.mock('@/stores/tags', () => ({
  useTagsStore: () => ({
    tags: [],
    fetchTags: (...args: unknown[]) => fetchTagsMock(...args),
    createTag: vi.fn(),
    updateTag: vi.fn(),
    deleteTag: vi.fn(),
  }),
}))

vi.mock('@/stores/templates', () => ({
  useTemplatesStore: () => ({
    fetchTemplates: (...args: unknown[]) => fetchTemplatesMock(...args),
    createTemplate: vi.fn(),
    updateTemplate: vi.fn(),
    deleteTemplate: vi.fn(),
  }),
}))

vi.mock('@/stores/teams', () => ({
  useTeamsStore: () => ({
    fetchTeams: (...args: unknown[]) => fetchTeamsMock(...args),
    deleteTeam: vi.fn(),
  }),
}))

vi.mock('@/lib/api-utils', () => ({
  getErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('@/lib/utils', () => ({
  formatDate: (value: string) => value,
}))

vi.mock('@/lib/constants', () => ({
  CANNED_RESPONSE_CATEGORIES: [],
  TAG_COLORS: [],
  ASSIGNMENT_STRATEGIES: [],
  getLabelFromValue: () => '',
}))

import CannedResponsesView from './CannedResponsesView.vue'
import TagsView from './TagsView.vue'
import TemplatesView from './TemplatesView.vue'
import TeamsView from './TeamsView.vue'
import APIKeysView from './APIKeysView.vue'
import WebhooksView from './WebhooksView.vue'
import CustomActionsView from './CustomActionsView.vue'
import SSOSettingsView from './SSOSettingsView.vue'

const mountOptions = { global: { mocks: { $t: t } } } as const

// File-level: every describe in this file mounts views that fetch on
// mount, so the empty-list mocks are (re)set before every test.
beforeEach(() => {
  vi.clearAllMocks()
  cannedListMock.mockResolvedValue({ data: { canned_responses: [], total: 0 } })
  apiKeysListMock.mockResolvedValue({ data: { api_keys: [], total: 0 } })
  webhooksListMock.mockResolvedValue({ data: { webhooks: [], available_events: [], total: 0 } })
  customActionsListMock.mockResolvedValue({ data: { custom_actions: [], total: 0 } })
  fetchTagsMock.mockResolvedValue({ tags: [], total: 0 })
  fetchTemplatesMock.mockResolvedValue({ templates: [], total: 0 })
  fetchTeamsMock.mockResolvedValue({ teams: [], total: 0 })
  apiGetMock.mockResolvedValue({ data: { data: [] } })
})

async function typeSearch(wrapper: ReturnType<typeof mount>, query: string) {
  const input = wrapper.find('input.search-input')
  await input.setValue(query)
  // Every list view debounces search by 300ms before re-fetching.
  await new Promise((resolve) => setTimeout(resolve, 350))
  await flushPromises()
}

describe('settings empty states teach the feature + one create action (Task 13)', () => {
  it.each([
    ['canned responses', CannedResponsesView, 'cannedResponses.noResponsesFound', 'cannedResponses.noResponsesFoundDesc', 'cannedResponses.addResponse'],
    ['tags', TagsView, 'tags.noTagsYet', 'tags.noTagsYetDesc', 'tags.addTag'],
    ['templates', TemplatesView, 'templates.noTemplatesYet', 'templates.noTemplatesYetDesc', 'templates.addTemplate'],
    ['teams', TeamsView, 'teams.noTeamsYet', 'teams.noTeamsYetDesc', 'teams.addTeam'],
    ['API keys', APIKeysView, 'apiKeys.noApiKeysYet', 'apiKeys.noApiKeysYetDesc', 'apiKeys.createApiKey'],
    ['webhooks', WebhooksView, 'webhooks.noWebhooksYet', 'webhooks.noWebhooksYetDesc', 'webhooks.addWebhook'],
    ['custom actions', CustomActionsView, 'customActions.noActionsYet', 'customActions.noActionsYetDesc', 'customActions.addAction'],
  ])('%s: true-empty state explains the feature and offers one create action', async (_name, View, titleKey, descKey, actionKey) => {
    const wrapper = mount(View as never, mountOptions)
    await flushPromises()

    expect(wrapper.find('.empty-title').text()).toBe(titleKey)
    expect(wrapper.find('.empty-description').text()).toBe(descKey)
    const actionButtons = wrapper.find('.empty-action').findAll('button')
    expect(actionButtons).toHaveLength(1)
    expect(actionButtons[0].text()).toContain(actionKey)
  })

  it.each([
    ['canned responses', CannedResponsesView, 'cannedResponses.noMatchingResponses', 'cannedResponses.noMatchingResponsesDesc', 'cannedResponses.addResponse'],
    ['tags', TagsView, 'tags.noMatchingTags', 'tags.noMatchingTagsDesc', 'tags.addTag'],
    ['templates', TemplatesView, 'templates.noMatchingTemplates', 'templates.noMatchingTemplatesDesc', 'templates.addTemplate'],
    ['teams', TeamsView, 'teams.noMatchingTeams', 'teams.noMatchingTeamsDesc', 'teams.addTeam'],
    ['API keys', APIKeysView, 'apiKeys.noMatchingApiKeys', 'apiKeys.noMatchingApiKeysDesc', 'apiKeys.createApiKey'],
    ['webhooks', WebhooksView, 'webhooks.noMatchingWebhooks', 'webhooks.noMatchingWebhooksDesc', 'webhooks.addWebhook'],
    ['custom actions', CustomActionsView, 'customActions.noMatchingActions', 'customActions.noMatchingActionsDesc', 'customActions.addAction'],
  ])('%s: a search with no match shows no-matching copy and no create action', async (_name, View, titleKey, descKey, actionKey) => {
    const wrapper = mount(View as never, mountOptions)
    await flushPromises()
    await typeSearch(wrapper, 'zzz-no-such-item')

    expect(wrapper.find('.empty-title').text()).toBe(titleKey)
    expect(wrapper.find('.empty-description').text()).toBe(descKey)
    expect(wrapper.find('.empty-action').text()).not.toContain(actionKey)
  })
})

describe('teaching copy wording (Task 13)', () => {
  it('each empty description says what the feature is for, in both locales', () => {
    // Canned responses: quick replies to common questions.
    expect(en.cannedResponses.noResponsesFoundDesc).toMatch(/common questions/i)
    expect(ar.cannedResponses.noResponsesFoundDesc).toContain('الأسئلة الشائعة')
    // API keys: let an external app access the account.
    expect(en.apiKeys.noApiKeysYetDesc).toMatch(/external app/i)
    expect(ar.apiKeys.noApiKeysYetDesc).toContain('تطبيق خارجي')
    // Webhooks: send real-time events to an external system.
    expect(en.webhooks.noWebhooksYetDesc).toMatch(/real-time events/i)
    expect(ar.webhooks.noWebhooksYetDesc).toContain('الأحداث الفورية')
    // Custom actions: a quick-integration button in the chat header.
    expect(en.customActions.noActionsYetDesc).toMatch(/chat header/i)
    expect(ar.customActions.noActionsYetDesc).toContain('رأس المحادثة')
  })

  it('the canned-responses no-match copy exists in both locales', () => {
    expect(en.cannedResponses.noMatchingResponses).toBeTruthy()
    expect(en.cannedResponses.noMatchingResponsesDesc).toBeTruthy()
    expect(ar.cannedResponses.noMatchingResponses).toBeTruthy()
    expect(ar.cannedResponses.noMatchingResponsesDesc).toBeTruthy()
  })
})

describe('SSO empty state (Task 13)', () => {
  it('with no provider configured: teaches SSO, one setup action, all 5 providers marked not configured', async () => {
    const wrapper = mount(SSOSettingsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.empty-state h3').text()).toBe('sso.noProvidersYet')
    expect(wrapper.find('.empty-state p').text()).toBe('sso.noProvidersYetDesc')
    const actionButtons = wrapper.find('.empty-state .empty-action').findAll('button')
    expect(actionButtons).toHaveLength(1)
    expect(actionButtons[0].text()).toContain('sso.setUpFirstProvider')
    expect((wrapper.text().match(/sso\.notConfigured/g) || []).length).toBe(5)
    expect(wrapper.text()).not.toContain('sso.enabled')
  })

  it('with one provider configured: no teaching banner, that provider shows Enabled, the other 4 show Not configured', async () => {
    apiGetMock.mockResolvedValue({
      data: {
        data: [
          { provider: 'google', client_id: 'cid', has_secret: true, is_enabled: true, allow_auto_create: false, default_role: 'agent', allowed_domains: '' },
        ],
      },
    })
    const wrapper = mount(SSOSettingsView, mountOptions)
    await flushPromises()

    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.text()).toContain('sso.enabled')
    expect((wrapper.text().match(/sso\.notConfigured/g) || []).length).toBe(4)
  })

  it('the banner action opens the existing setup dialog for the first provider', async () => {
    const wrapper = mount(SSOSettingsView, mountOptions)
    await flushPromises()

    expect(wrapper.text()).not.toContain('sso.configure Google')
    await wrapper.find('.empty-state .empty-action').find('button').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('sso.configure Google')
  })
})
