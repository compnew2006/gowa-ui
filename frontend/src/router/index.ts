import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";

// Permission-based route meta type
declare module "vue-router" {
  interface RouteMeta {
    requiresAuth?: boolean;
    permission?: string; // Resource permission required (e.g., 'analytics', 'chat')
    metaOnly?: boolean; // Route only available when provider is "meta"
    adminOnly?: boolean; // Route only available for admin/super-admin users
    managerOrAdminOnly?: boolean; // Route only available for manager/admin/super-admin users
  }
}

// Get base path from server-injected config or fallback to Vite's BASE_URL
const basePath =
  (window as any).__BASE_PATH__ ?? import.meta.env.BASE_URL ?? "/";
const normalizedBasePath = basePath.endsWith("/") ? basePath : basePath + "/";

const router = createRouter({
  history: createWebHistory(normalizedBasePath),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("@/views/auth/LoginView.vue"),
      meta: { requiresAuth: false },
    },
    {
      path: "/register",
      name: "register",
      component: () => import("@/views/auth/RegisterView.vue"),
      meta: { requiresAuth: false },
    },
    {
      path: "/auth/sso/callback",
      name: "sso-callback",
      component: () => import("@/views/auth/SSOCallbackView.vue"),
      meta: { requiresAuth: false },
    },
    {
      path: "/pricing",
      alias: ["/plans", "/offer"],
      name: "pricing-landing",
      component: () => import("@/views/public/PricingLandingView.vue"),
      meta: { requiresAuth: false },
    },
    {
      path: "/",
      component: () => import("@/components/layout/AppLayout.vue"),
      meta: { requiresAuth: true },
      children: [
        {
          path: "",
          name: "dashboard",
          component: () => import("@/views/dashboard/DashboardView.vue"),
          meta: { permission: "analytics" },
        },
        {
          path: "chat",
          name: "chat",
          component: () => import("@/views/chat/ChatView.vue"),
          meta: { permission: "chat" },
        },
        {
          path: "chat/:contactId",
          name: "chat-conversation",
          component: () => import("@/views/chat/ChatView.vue"),
          props: true,
          meta: { permission: "chat" },
        },
        {
          path: "profile",
          name: "profile",
          component: () => import("@/views/profile/ProfileView.vue"),
          // All roles can access profile
        },
        {
          path: "activity-logs",
          redirect: "/settings/activity-logs",
        },
        {
          path: "templates",
          name: "templates",
          component: () => import("@/views/settings/TemplatesView.vue"),
          meta: { permission: "templates", metaOnly: true },
        },
        {
          path: "flows",
          name: "flows",
          component: () => import("@/views/settings/FlowsView.vue"),
          meta: { permission: "flows.whatsapp", metaOnly: true },
        },
        {
          path: "campaigns",
          name: "campaigns",
          component: () => import("@/views/settings/CampaignsView.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "chatbot",
          name: "chatbot",
          component: () => import("@/views/chatbot/ChatbotView.vue"),
          meta: { permission: "settings.chatbot" },
        },
        {
          path: "chatbot/settings",
          redirect: "/settings/chatbot",
        },
        {
          path: "chatbot/keywords",
          name: "chatbot-keywords",
          component: () => import("@/views/chatbot/KeywordsView.vue"),
          meta: { permission: "chatbot.keywords" },
        },
        {
          path: "chatbot/flows",
          name: "chatbot-flows",
          component: () => import("@/views/chatbot/ChatbotFlowsView.vue"),
          meta: { permission: "flows.chatbot" },
        },
        {
          path: "chatbot/flows/new",
          name: "chatbot-flow-new",
          component: () => import("@/views/chatbot/ChatbotFlowBuilderView.vue"),
          meta: { permission: "flows.chatbot" },
        },
        {
          path: "chatbot/flows/:id/edit",
          name: "chatbot-flow-edit",
          component: () => import("@/views/chatbot/ChatbotFlowBuilderView.vue"),
          meta: { permission: "flows.chatbot" },
        },
        {
          path: "chatbot/ai",
          name: "chatbot-ai",
          component: () => import("@/views/chatbot/AIContextsView.vue"),
          meta: { permission: "chatbot.ai" },
        },
        {
          path: "chatbot/transfers",
          name: "chatbot-transfers",
          component: () => import("@/views/chatbot/AgentTransfersView.vue"),
          meta: { permission: "transfers" },
        },
        {
          path: "analytics/agents",
          name: "agent-analytics",
          component: () => import("@/views/analytics/AgentAnalyticsView.vue"),
          meta: { permission: "analytics.agents" },
        },
        {
          path: "analytics/meta-insights",
          name: "meta-insights",
          component: () => import("@/views/analytics/MetaInsightsView.vue"),
          meta: { permission: "analytics", metaOnly: true },
        },
        {
          path: "settings",
          name: "settings",
          component: () => import("@/views/settings/SettingsView.vue"),
          meta: { permission: "settings.general" },
        },
        {
          path: "settings/chatbot",
          name: "chatbot-settings",
          component: () => import("@/views/settings/ChatbotSettingsView.vue"),
          meta: { permission: "settings.chatbot" },
        },
        {
          path: "settings/accounts",
          name: "accounts",
          component: () => import("@/views/settings/AccountsView.vue"),
          meta: { permission: "accounts", metaOnly: true },
        },
        {
          path: "settings/instances",
          name: "instances",
          component: () => import("@/views/settings/InstancesView.vue"),
          meta: { permission: "accounts" },
        },
        {
          path: "settings/instances/health",
          name: "instances-health",
          component: () => import("@/views/settings/InstanceHealthView.vue"),
          meta: { permission: "accounts" },
        },
        {
          path: "settings/canned-responses",
          name: "canned-responses",
          component: () => import("@/views/settings/CannedResponsesView.vue"),
          meta: { permission: "canned_responses" },
        },
        {
          path: "settings/contacts",
          name: "contacts",
          component: () => import("@/views/settings/ContactsView.vue"),
          meta: { permission: "contacts" },
        },
        {
          path: "settings/closed-chats",
          name: "closed-chats",
          component: () => import("@/views/settings/ClosedChatsView.vue"),
          meta: { permission: "chat" },
        },
        {
          path: "settings/tags",
          name: "tags",
          component: () => import("@/views/settings/TagsView.vue"),
          meta: { permission: "tags" },
        },
        {
          path: "settings/users",
          name: "users",
          component: () => import("@/views/settings/UsersView.vue"),
          meta: { permission: "users" },
        },
        {
          path: "settings/activity-logs",
          name: "activity-logs",
          component: () => import("@/views/activity/ActivityLogsView.vue"),
          meta: { managerOrAdminOnly: true },
        },
        {
          path: "settings/lead-requests",
          name: "lead-requests",
          component: () => import("@/views/settings/LeadRequestsView.vue"),
          meta: { permission: "settings.general" },
        },
        {
          path: "settings/roles",
          name: "roles",
          component: () => import("@/views/settings/RolesView.vue"),
          meta: { permission: "roles" },
        },
        {
          path: "settings/teams",
          name: "teams",
          component: () => import("@/views/settings/TeamsView.vue"),
          meta: { permission: "teams" },
        },
        {
          path: "settings/api-keys",
          name: "api-keys",
          component: () => import("@/views/settings/APIKeysView.vue"),
          meta: { permission: "api_keys" },
        },
        {
          path: "settings/webhooks",
          name: "webhooks",
          component: () => import("@/views/settings/WebhooksView.vue"),
          meta: { permission: "webhooks" },
        },
        {
          path: "settings/sso",
          name: "sso-settings",
          component: () => import("@/views/settings/SSOSettingsView.vue"),
          meta: { permission: "settings.sso" },
        },
        {
          path: "settings/custom-actions",
          name: "custom-actions",
          component: () => import("@/views/settings/CustomActionsView.vue"),
          meta: { permission: "custom_actions" },
        },
        {
          path: "settings/migration",
          name: "migration",
          component: () => import("@/views/settings/MigrationView.vue"),
          meta: { permission: "settings.general" },
        },
      ],
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("@/views/NotFoundView.vue"),
    },
  ],
});

// Navigation items with permissions in priority order (matches AppLayout.vue)
// Used to find the first accessible route for a user
const navigationOrder = [
  { path: "/", permission: "analytics" },
  { path: "/chat", permission: "chat" },
  {
    path: "/chatbot",
    permission: "settings.chatbot",
    childPaths: [
      { path: "/chatbot", permission: "settings.chatbot" },
      { path: "/chatbot/keywords", permission: "chatbot.keywords" },
      { path: "/chatbot/flows", permission: "flows.chatbot" },
      { path: "/chatbot/ai", permission: "chatbot.ai" },
    ],
  },
  { path: "/chatbot/transfers", permission: "transfers" },
  { path: "/analytics/agents", permission: "analytics.agents" },
  { path: "/analytics/meta-insights", permission: "analytics" },
  { path: "/templates", permission: "templates" },
  { path: "/flows", permission: "flows.whatsapp" },
  { path: "/campaigns", permission: "campaigns" },
  {
    path: "/settings",
    permission: "settings.general",
    childPaths: [
      { path: "/settings", permission: "settings.general" },
      { path: "/settings/chatbot", permission: "settings.chatbot" },
      { path: "/settings/instances", permission: "accounts" },
      { path: "/settings/accounts", permission: "accounts" },
      { path: "/settings/canned-responses", permission: "canned_responses" },
      { path: "/settings/contacts", permission: "contacts" },
      { path: "/settings/closed-chats", permission: "chat" },
      { path: "/settings/tags", permission: "tags" },
      { path: "/settings/teams", permission: "teams" },
      { path: "/settings/users", permission: "users" },
      { path: "/settings/activity-logs", permission: "settings.general" },
      { path: "/settings/lead-requests", permission: "settings.general" },
      { path: "/settings/roles", permission: "roles" },
      { path: "/settings/api-keys", permission: "api_keys" },
      { path: "/settings/webhooks", permission: "webhooks" },
      { path: "/settings/custom-actions", permission: "custom_actions" },
      { path: "/settings/migration", permission: "settings.general" },
      { path: "/settings/sso", permission: "settings.sso" },
    ],
  },
];

// Find the first accessible route for the user
function getFirstAccessibleRoute(
  authStore: ReturnType<typeof useAuthStore>,
): string {
  for (const item of navigationOrder) {
    // Check if user has permission for this item
    if (authStore.hasPermission(item.permission, "read")) {
      return item.path;
    }
    // Check child paths if available
    if (item.childPaths) {
      for (const child of item.childPaths) {
        if (authStore.hasPermission(child.permission, "read")) {
          return child.path;
        }
      }
    }
  }
  // Fallback to profile (always accessible)
  console.warn("No accessible routes found for user, falling back to profile.");
  return "/profile";
}

// Navigation guard
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore();

  // Check if route requires auth
  if (to.meta.requiresAuth !== false) {
      if (!authStore.isAuthenticated) {
        // Try to restore session from localStorage
        const restored = await authStore.restoreSession();
        if (!restored) {
          return next({ name: "login", query: { redirect: to.fullPath } });
        }
      }

    // Check permission-based access
    const requiredPermission = to.meta.permission;
    if (requiredPermission) {
      if (!authStore.hasPermission(requiredPermission, "read")) {
        const fallback = getFirstAccessibleRoute(authStore);
        console.warn(`Access denied to ${to.path}. Redirecting to ${fallback}`);
        if (fallback === to.path) {
          // Prevent infinite redirect loop
          return next({ name: "profile" });
        }
        return next({ path: fallback });
      }
    }
  } else {
    // Redirect to appropriate page if already logged in
    if (
      authStore.isAuthenticated &&
      (to.name === "login" || to.name === "register")
    ) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  // Provider-based access: block Meta-only routes when using whatsmeow
  if (to.meta.metaOnly) {
    const configStore = useConfigStore();
    if (configStore.isWhatsmeow) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  // Role-based access: block admin-only routes for non-admins
  if (to.meta.adminOnly) {
    const isAdmin =
      authStore.user?.is_super_admin === true ||
      (authStore.userRole || "").toLowerCase() === "admin";
    if (!isAdmin) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  if (to.meta.managerOrAdminOnly) {
    const roleName = (authStore.userRole || "").toLowerCase();
    const isManagerOrAdmin =
      authStore.user?.is_super_admin === true ||
      roleName === "admin" ||
      roleName === "manager";
    if (!isManagerOrAdmin) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  next();
});

export default router;
