/**
 * Task 15a: a route the viewer lacks permission for must show an explicit
 * permission-denied notice naming the page — not a silent redirect.
 * A route the viewer CAN access still opens normally.
 *
 * One router instance is shared across tests (fresh instances are slow to
 * build and each one re-loads the layout chain). The auth store is mocked
 * with a live permission set, so each test just sets its permissions and
 * navigates. The layout + view components are stubbed: the real AppLayout
 * chain takes ~20s to load in this environment and is not what is under
 * test here — the guard, the notice, and the redirect target are.
 */
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Router } from 'vue-router'

const toastError = vi.hoisted(() => vi.fn())
const authPermissions = vi.hoisted(() => new Set<string>())

let router: Router
let resolveDeniedPageLabel: (to: { path: string; meta?: { title?: string } }) => string

/** Agent permission set from Task 14 (10 permissions, roles.go:261-276). */
const AGENT_PERMISSIONS = [
  'accounts',
  'chat',
  'chat.revoke',
  'chat.assign',
  'contacts',
  'tags',
  'analytics.agents',
  'canned_responses',
  'custom_actions',
]

function setPermissions(permissions: string[]) {
  authPermissions.clear()
  for (const p of permissions) authPermissions.add(p)
}

/** Neutral route with no permission meta, used to reset between tests. */
async function resetToProfile() {
  if (router.currentRoute.value.path !== '/profile') {
    await router.push('/profile')
  }
}

describe('router permission-denied notice (Task 15a)', () => {
  beforeAll(async () => {
    vi.doMock('@/stores/auth', () => ({
      useAuthStore: () => ({
        user: null,
        isAuthenticated: true,
        restoreSession: vi.fn(() => true),
        hasPermission: (resource: string) => authPermissions.has(resource),
      }),
    }))
    vi.doMock('@/composables/useAppToast', () => ({
      useAppToast: () => ({
        success: vi.fn(),
        error: toastError,
        info: vi.fn(),
        warning: vi.fn(),
      }),
    }))
    // Stub the layout + views the navigations land on; their real import
    // chains are out of scope for the guard under test.
    vi.doMock('@/components/layout/AppLayout.vue', () => ({
      default: { name: 'AppLayoutStub', template: '<div>layout</div>' },
    }))
    vi.doMock('@/views/profile/ProfileView.vue', () => ({
      default: { name: 'ProfileViewStub', template: '<div>profile</div>' },
    }))
    vi.doMock('@/views/chat/ChatView.vue', () => ({
      default: { name: 'ChatViewStub', template: '<div>chat</div>' },
    }))
    vi.doMock('@/views/settings/UsersView.vue', () => ({
      default: { name: 'UsersViewStub', template: '<div>users</div>' },
    }))
    vi.doMock('@/views/settings/CampaignsView.vue', () => ({
      default: { name: 'CampaignsViewStub', template: '<div>campaigns</div>' },
    }))
    const mod = await import('@/router/index')
    router = mod.default
    resolveDeniedPageLabel = mod.resolveDeniedPageLabel
    await router.push('/profile')
  })

  beforeEach(async () => {
    setPermissions(AGENT_PERMISSIONS)
    await resetToProfile()
  })

  it('shows a notice naming the page when the viewer lacks permission, instead of a silent redirect', async () => {
    await router.push('/settings/users')

    // Still lands on the first accessible route (the redirect stays),
    // but the user is told why.
    expect(router.currentRoute.value.path).toBe('/chat')
    expect(toastError).toHaveBeenCalledTimes(1)
    const [title, description] = toastError.mock.calls[0] as [string, string]
    expect(title).toContain('Users')
    expect(description).toContain('Users')
  })

  it('names the page for a detail route too (longest matching nav entry)', async () => {
    await router.push('/settings/users/00000000-0000-0000-0000-000000000000')

    expect(router.currentRoute.value.path).toBe('/chat')
    expect(toastError).toHaveBeenCalledTimes(1)
    const [title] = toastError.mock.calls[0] as [string, string]
    expect(title).toContain('Users')
  })

  it('names Campaigns for the campaigns route', async () => {
    await router.push('/campaigns')

    expect(router.currentRoute.value.path).toBe('/chat')
    expect(toastError).toHaveBeenCalledTimes(1)
    const [title] = toastError.mock.calls[0] as [string, string]
    expect(title).toContain('Campaigns')
  })

  it('still redirects when the notice itself cannot render (redirect is the fallback)', async () => {
    toastError.mockImplementationOnce(() => {
      throw new Error('toast unavailable')
    })

    await router.push('/settings/users')

    expect(router.currentRoute.value.path).toBe('/chat')
  })

  it('opens the route normally with no notice when the viewer has permission', async () => {
    setPermissions([...AGENT_PERMISSIONS, 'users'])

    await router.push('/settings/users')

    expect(router.currentRoute.value.path).toBe('/settings/users')
    expect(toastError).not.toHaveBeenCalled()
  })
})

describe('resolveDeniedPageLabel', () => {
  it('resolves detail routes to their parent page name', () => {
    expect(resolveDeniedPageLabel({ path: '/settings/users/abc', meta: {} })).toBe('Users')
    expect(resolveDeniedPageLabel({ path: '/settings/api-keys/abc', meta: {} })).toBe('API Keys')
  })

  it('falls back to the route title, then the path', () => {
    expect(resolveDeniedPageLabel({ path: '/no-such-page', meta: { title: 'Mystery' } })).toBe('Mystery')
    expect(resolveDeniedPageLabel({ path: '/no-such-page', meta: {} })).toBe('/no-such-page')
  })
})
