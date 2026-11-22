import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { navigationSections } from '@/components/layout/navigation'
import { useAppToast } from '@/composables/useAppToast'
import { i18n } from '@/i18n'

// Permission-based route meta type
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    permission?: string // Resource permission required (e.g., 'analytics', 'chat')
    title?: string // Shown in the browser tab as "<title> | Gowa-UI"
  }
}

// Get base path from server-injected config or fallback to Vite's BASE_URL
const basePath = (window as any).__BASE_PATH__ ?? import.meta.env.BASE_URL ?? '/'
const normalizedBasePath = basePath.endsWith('/') ? basePath : basePath + '/'

const router = createRouter({
  history: createWebHistory(normalizedBasePath),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/auth/RegisterView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/auth/sso/callback',
      name: 'sso-callback',
      component: () => import('@/views/auth/SSOCallbackView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      component: () => import('@/components/layout/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/dashboard/DashboardView.vue'),
          meta: { permission: 'analytics', title: 'Dashboard' }
        },
        {
          path: 'chat/:contactId?',
          name: 'chat-conversation',
          component: () => import('@/views/chat/ChatView.vue'),
          props: true,
          meta: { permission: 'chat', stableKey: true, title: 'Chat' }
        },
        {
          path: 'profile',
          name: 'profile',
          component: () => import('@/views/profile/ProfileView.vue')
          // All roles can access profile
        },
        {
          path: 'campaigns',
          name: 'campaigns',
          component: () => import('@/views/settings/CampaignsView.vue'),
          meta: { permission: 'campaigns', title: 'Campaigns' }
        },
        {
          path: 'campaigns/:id',
          name: 'campaign-detail',
          component: () => import('@/views/settings/CampaignDetailView.vue'),
          meta: { permission: 'campaigns' }
        },
        {
          path: 'analytics/agents',
          name: 'agent-analytics',
          component: () => import('@/views/analytics/AgentAnalyticsView.vue'),
          meta: { permission: 'analytics.agents', title: 'Agent Analytics' }
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/settings/SettingsView.vue'),
          meta: { permission: 'settings.general', title: 'Settings' }
        },
        {
          path: 'settings/general',
          redirect: '/settings'
        },
        {
          path: 'settings/accounts',
          name: 'accounts',
          component: () => import('@/views/settings/AccountsView.vue'),
          meta: { permission: 'accounts' }
        },
        {
          path: 'settings/accounts/:id',
          name: 'account-detail',
          component: () => import('@/views/settings/AccountDetailView.vue'),
          meta: { permission: 'accounts' }
        },
        {
          path: 'settings/gowa-servers',
          name: 'gowa-servers',
          component: () => import('@/views/settings/GowaServersView.vue'),
          meta: { permission: 'gowa_instances' }
        },
        {
          path: 'settings/gowa-servers/:id',
          name: 'gowa-server-detail',
          component: () => import('@/views/settings/GowaServerDetailView.vue'),
          meta: { permission: 'gowa_instances' }
        },
        {
          path: 'settings/canned-responses',
          name: 'canned-responses',
          component: () => import('@/views/settings/CannedResponsesView.vue'),
          meta: { permission: 'canned_responses' }
        },
        {
          path: 'settings/canned-responses/:id',
          name: 'canned-response-detail',
          component: () => import('@/views/settings/CannedResponseDetailView.vue'),
          // stableKey reuses the component instance when :id flips from "new"
          // to the new UUID after create, so the locally-set response (and the
          // resulting Save-button reactivity) survives the route change.
          meta: { permission: 'canned_responses', stableKey: true }
        },
        {
          path: 'settings/contacts',
          name: 'contacts',
          component: () => import('@/views/settings/ContactsView.vue'),
          meta: { permission: 'contacts.manage' }
        },
        {
          path: 'settings/contacts/:id',
          name: 'contact-detail',
          component: () => import('@/views/settings/ContactDetailView.vue'),
          meta: { permission: 'contacts.manage' }
        },
        {
          path: 'settings/tags',
          name: 'tags',
          component: () => import('@/views/settings/TagsView.vue'),
          meta: { permission: 'tags' }
        },
        {
          path: 'settings/templates',
          name: 'templates',
          component: () => import('@/views/settings/TemplatesView.vue'),
          meta: { permission: 'templates' }
        },
        {
          path: 'settings/users',
          name: 'users',
          component: () => import('@/views/settings/UsersView.vue'),
          meta: { permission: 'users' }
        },
        {
          path: 'settings/users/:id',
          name: 'user-detail',
          component: () => import('@/views/settings/UserDetailView.vue'),
          meta: { permission: 'users' }
        },
        {
          path: 'settings/roles',
          name: 'roles',
          component: () => import('@/views/settings/RolesView.vue'),
          meta: { permission: 'roles' }
        },
        {
          path: 'settings/roles/:id',
          name: 'role-detail',
          component: () => import('@/views/settings/RoleDetailView.vue'),
          meta: { permission: 'roles' }
        },
        {
          path: 'settings/teams',
          name: 'teams',
          component: () => import('@/views/settings/TeamsView.vue'),
          meta: { permission: 'teams' }
        },
        {
          path: 'settings/teams/:id',
          name: 'team-detail',
          component: () => import('@/views/settings/TeamDetailView.vue'),
          meta: { permission: 'teams' }
        },
        {
          path: 'settings/api-keys',
          name: 'api-keys',
          component: () => import('@/views/settings/APIKeysView.vue'),
          meta: { permission: 'api_keys' }
        },
        {
          path: 'settings/api-keys/:id',
          name: 'api-key-detail',
          component: () => import('@/views/settings/APIKeyDetailView.vue'),
          meta: { permission: 'api_keys' }
        },
        {
          path: 'settings/webhooks',
          name: 'webhooks',
          component: () => import('@/views/settings/WebhooksView.vue'),
          meta: { permission: 'webhooks' }
        },
        {
          path: 'settings/webhooks/:id',
          name: 'webhook-detail',
          component: () => import('@/views/settings/WebhookDetailView.vue'),
          meta: { permission: 'webhooks' }
        },
        {
          path: 'settings/sso',
          name: 'sso-settings',
          component: () => import('@/views/settings/SSOSettingsView.vue'),
          meta: { permission: 'settings.sso' }
        },
        {
          path: 'settings/custom-actions',
          name: 'custom-actions',
          component: () => import('@/views/settings/CustomActionsView.vue'),
          meta: { permission: 'custom_actions' }
        },
        {
          path: 'settings/audit-logs',
          name: 'audit-logs',
          component: () => import('@/views/settings/AuditLogsView.vue'),
          meta: { permission: 'audit_logs' }
        },
        {
          path: 'settings/audit-logs/:id',
          name: 'audit-log-detail',
          component: () => import('@/views/settings/AuditLogDetailView.vue'),
          meta: { permission: 'audit_logs' }
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue')
    }
  ]
})

// Navigation items with permissions in priority order, flattened from the
// sidebar's navigation sections (AppLayout.vue) so the sidebar and the
// first-accessible-route fallback can never drift apart. Used to find the
// first accessible route for a user.
const navigationOrder = navigationSections.flatMap((section) =>
  section.items.map((item) => ({
    path: item.path,
    permission: item.permission ?? '',
    childPaths: item.children?.map((child) => ({
      path: child.path,
      permission: child.permission ?? ''
    }))
  }))
)

// Find the first accessible route for the user
function getFirstAccessibleRoute(authStore: ReturnType<typeof useAuthStore>): string {
  for (const item of navigationOrder) {
    // Check if user has permission for this item
    if (authStore.hasPermission(item.permission, 'read')) {
      return item.path
    }
    // Check child paths if available
    if (item.childPaths) {
      for (const child of item.childPaths) {
        if (authStore.hasPermission(child.permission, 'read')) {
          return child.path
        }
      }
    }
  }
  // Fallback to profile (always accessible)
  return '/profile'
}

// Sidebar entries (items + children) the denied page's name comes from,
// so the notice names the page exactly as the sidebar does.
const navigationLabels = navigationSections.flatMap((section) =>
  section.items.flatMap((item) => [
    { path: item.path, name: item.name },
    ...(item.children?.map((child) => ({ path: child.path, name: child.name })) ?? []),
  ]),
)

/**
 * Display name of the page a route points at, for the permission-denied
 * notice. Matches the longest sidebar entry that is the route's path or
 * its parent (so /settings/users/:id reads as "Users"), then falls back
 * to the route title and finally the raw path.
 */
export function resolveDeniedPageLabel(to: { path: string; meta?: { title?: string } }): string {
  let best: { path: string; name: string } | undefined
  for (const entry of navigationLabels) {
    if (to.path === entry.path || to.path.startsWith(entry.path + '/')) {
      if (!best || entry.path.length > best.path.length) best = entry
    }
  }
  if (best) return i18n.global.t(best.name)
  return to.meta?.title ?? to.path
}

/**
 * Explicit permission-denied notice (toast) naming the page the user tried
 * to open. Best-effort only: when no notice can render, the caller still
 * redirects (the redirect is the fallback), and navigation must never
 * break over a toast.
 */
function showPermissionDeniedNotice(to: { path: string; meta?: { title?: string } }): void {
  try {
    const page = resolveDeniedPageLabel(to)
    const { error } = useAppToast()
    error(
      i18n.global.t('router.permissionDeniedTitle', { page }),
      i18n.global.t('router.permissionDeniedDescription', { page }),
    )
  } catch {
    // The redirect below stays the fallback when no notice can render.
  }
}

// Navigation guard
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  // Check if route requires auth
  if (to.meta.requiresAuth !== false) {
    if (!authStore.isAuthenticated) {
      // Try to restore session from localStorage
      const restored = authStore.restoreSession()
      if (!restored) {
        return next({ name: 'login', query: { redirect: to.fullPath } })
      }
    }

    // Check permission-based access
    const requiredPermission = to.meta.permission
    if (requiredPermission) {
      if (!authStore.hasPermission(requiredPermission, 'read')) {
        // Explicit notice first; the redirect stays as the fallback.
        showPermissionDeniedNotice(to)
        return next({ path: getFirstAccessibleRoute(authStore) })
      }
    }
  } else {
    // Redirect to appropriate page if already logged in
    if (authStore.isAuthenticated && (to.name === 'login' || to.name === 'register')) {
      return next({ path: getFirstAccessibleRoute(authStore) })
    }
  }

  next()
})

// Browser tab title: "<route title> | Gowa-UI", falling back to "Gowa-UI"
// for routes without a title (login, 404, profile, ...).
router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} | Gowa-UI` : 'Gowa-UI'
})

export default router
