import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import { useLicenseStore } from "@/stores/license";
import { moduleKeyForPath } from "@/modules/registry";

// Permission-based route meta type
declare module "vue-router" {
  interface RouteMeta {
    requiresAuth?: boolean;
    permission?: string; // Resource permission required (e.g., 'analytics', 'chat')
    permissionsAny?: string[];
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
      path: "/activate",
      name: "activate",
      component: () => import("@/views/public/ActivateLicenseView.vue"),
      meta: { requiresAuth: false },
    },
    {
      path: "/pricing",
      alias: ["/plans", "/offer"],
      name: "marketing-redirect",
      component: () => import("@/views/public/MarketingRedirectView.vue"),
      meta: { requiresAuth: false },
    },
    {
      path: "/",
      component: () => import("@/components/layout/AppLayout.vue"),
      meta: { requiresAuth: true },
      children: [
        {
          path: "license-cleanup",
          name: "license-cleanup",
          component: () => import("@/views/settings/LicenseCleanupView.vue"),
        },
        {
          path: "",
          redirect: "/chat",
        },
        {
          path: "dashboard",
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
          path: "whatsapp/campaigns",
          name: "campaigns",
          component: () => import("@/views/settings/CampaignsView.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "whatsapp/instances",
          name: "instances",
          component: () => import("@/views/settings/InstancesView.vue"),
          meta: { permission: "accounts" },
        },
        {
          path: "whatsapp/instances/health",
          name: "instances-health",
          component: () => import("@/views/settings/InstanceHealthView.vue"),
          meta: { permission: "accounts" },
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
          component: () => import("@/views/settings/SettingsHubView.vue"),
          meta: {
            permissionsAny: ["settings.general", "settings.uploads_cleanup"],
          },
        },
        {
          path: "settings/user",
          name: "settings-user",
          component: () => import("@/views/settings/SettingsView.vue"),
          meta: {
            permissionsAny: ["settings.general", "settings.uploads_cleanup"],
          },
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
          path: "whatsapp/canned-responses",
          name: "canned-responses",
          component: () => import("@/views/settings/CannedResponsesView.vue"),
          meta: { permission: "canned_responses" },
        },
        {
          path: "whatsapp/saved-contents",
          name: "saved-contents",
          component: () => import("@/views/settings/SavedContentsView.vue"),
          meta: { permission: "saved_contents" },
        },
        {
          path: "whatsapp/contacts",
          name: "contacts",
          component: () => import("@/views/settings/ContactsView.vue"),
          meta: { permission: "contacts" },
        },
        {
          path: "whatsapp/closed-chats",
          name: "closed-chats",
          component: () => import("@/views/settings/ClosedChatsView.vue"),
          meta: { permission: "chat" },
        },
        {
          path: "settings/canned-responses",
          redirect: "/whatsapp/canned-responses",
        },
        {
          path: "settings/contacts",
          redirect: "/whatsapp/contacts",
        },
        {
          path: "settings/closed-chats",
          redirect: "/whatsapp/closed-chats",
        },
        {
          path: "settings/whatsapp-filter",
          redirect: "/whatsapp/whatsapp-filter",
        },
        {
          path: "settings/tags",
          name: "tags",
          component: () => import("@/views/settings/TagsView.vue"),
          meta: { permission: "tags" },
        },
        {
          path: "whatsapp/whatsapp-filter",
          name: "whatsapp-filter",
          component: () => import("@/views/settings/WhatsAppFilter.vue"),
          meta: { permission: "wa_filter" },
        },
        {
          path: "settings/group-search",
          redirect: "/whatsapp/group-search",
        },
        {
          path: "whatsapp/group-search",
          name: "group-search",
          component: () => import("@/views/settings/GroupSearch.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "whatsapp/group-join-campaigns",
          name: "group-join-campaigns",
          component: () => import("@/views/settings/GroupJoinCampaignsView.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "whatsapp/group-extraction",
          name: "group-extraction",
          component: () => import("@/views/settings/GroupExtractionView.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "whatsapp/member-extraction",
          name: "member-extraction",
          component: () => import("@/views/settings/MemberExtractionView.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "whatsapp/group-participants",
          name: "group-participants",
          component: () => import("@/views/settings/GroupParticipants.vue"),
          meta: { title: "Group Members", icon: "Users", permission: "campaigns" },
        },
        {
          path: "whatsapp/extract",
          name: "extract",
          component: () => import("@/views/tools/ExtractView.vue"),
          meta: { permission: "campaigns" },
        },
        {
          path: "settings/users",
          name: "users",
          component: () => import("@/views/settings/UsersView.vue"),
          meta: { permission: "users" },
        },
        {
          path: "settings/roles",
          name: "roles",
          component: () => import("@/views/settings/RolesView.vue"),
          meta: { permission: "roles" },
        },
        {
          path: "settings/audit-log",
          name: "audit-log",
          component: () => import("@/views/settings/AuditLogView.vue"),
          meta: { permission: "audit" },
        },
        {
          path: "settings/teams",
          name: "teams",
          component: () => import("@/views/settings/TeamsView.vue"),
          meta: { permission: "teams" },
        },
        {
          path: "whatsapp/agent-selection",
          name: "agent-selection",
          component: () => import("@/views/settings/AgentSelectionView.vue"),
          meta: { permission: "agent_selection" },
        },
        {
          path: "settings/agent-selection",
          redirect: "/whatsapp/agent-selection",
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
          path: "settings/license",
          name: "license-settings",
          component: () => import("@/views/settings/LicenseSettingsView.vue"),
          meta: { adminOnly: true },
        },
        {
          path: "settings/modules",
          name: "modules-settings",
          component: () => import("@/views/settings/ModulesView.vue"),
          meta: { permission: "organizations" },
        },
        {
          path: "settings/custom-actions",
          name: "custom-actions",
          component: () => import("@/views/settings/CustomActionsView.vue"),
          meta: { permission: "custom_actions" },
        },
        {
          path: "facebook",
          name: "facebook-hub",
          component: () => import("@/views/facebook/FacebookHubView.vue"),
          meta: { permission: "chat" },
        },
        {
          path: "whatsapp",
          name: "whatsapp-hub",
          component: () => import("@/views/whatsapp/WhatsappHubView.vue"),
          meta: { permission: "chat" },
        },
        {
          path: "facebook/comments",
          name: "facebook-comments",
          component: () => import("@/views/facebook/FacebookCommentsView.vue"),
          meta: { permission: "chat" },
        },
        {
          path: "settings/facebook-comments",
          redirect: "/facebook/comments",
        },
        {
          path: "settings/facebook",
          redirect: "/facebook/comments",
        },
        {
          path: "facebook/page-search",
          name: "facebook-page-search",
          component: () => import("@/views/facebook/PageSearchView.vue"),
        },
        {
          path: "facebook/people-search",
          name: "facebook-people-search",
          component: () => import("@/views/facebook/PeopleSearchView.vue"),
        },
        {
          path: "facebook/group-search",
          name: "facebook-group-search",
          component: () => import("@/views/facebook/GroupSearchView.vue"),
        },
        {
          path: "facebook/extract-likes",
          name: "facebook-extract-likes",
          component: () => import("@/views/facebook/ExtractLikesView.vue"),
        },
        {
          path: "facebook/page-messengers",
          name: "facebook-page-messengers",
          component: () => import("@/views/facebook/PageMessengersView.vue"),
        },
        {
          path: "facebook/extract-data",
          name: "facebook-extract-data",
          component: () => import("@/views/facebook/ExtractDataView.vue"),
        },
        {
          path: "facebook/auto-share",
          name: "facebook-auto-share",
          component: () => import("@/views/facebook/AutoShareView.vue"),
        },
        {
          path: "facebook/retargeting",
          name: "facebook-retargeting",
          component: () => import("@/views/facebook/RetargetingView.vue"),
        },
        {
          path: "facebook/accounts",
          name: "facebook-accounts",
          component: () => import("@/views/facebook/FacebookAccountsView.vue"),
          meta: { permission: "accounts" },
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
  { path: "/chat", permission: "chat" },
  { path: "/dashboard", permission: "analytics" },
  {
    path: "/chatbot",
    permission: "settings.chatbot",
    childPaths: [
      { path: "/chatbot", permission: "settings.chatbot" },
      { path: "/chatbot/keywords", permission: "chatbot.keywords" },
      { path: "/chatbot/flows", permission: "flows.chatbot" },
      { path: "/chatbot/ai", permission: "chatbot.ai" },
      { path: "/chatbot/transfers", permission: "transfers" },
    ],
  },
  { path: "/analytics/agents", permission: "analytics.agents" },
  { path: "/analytics/meta-insights", permission: "analytics" },
  { path: "/templates", permission: "templates" },
  { path: "/flows", permission: "flows.whatsapp" },
  { path: "/whatsapp/campaigns", permission: "campaigns" },
  {
    path: "/whatsapp",
    permission: "campaigns",
    childPaths: [
      { path: "/whatsapp/campaigns", permission: "campaigns" },
      { path: "/whatsapp/group-join-campaigns", permission: "campaigns" },
      { path: "/whatsapp/group-extraction", permission: "campaigns" },
      { path: "/whatsapp/member-extraction", permission: "campaigns" },
      { path: "/chatbot", permission: "settings.chatbot" },
      { path: "/chatbot/keywords", permission: "chatbot.keywords" },
      { path: "/chatbot/flows", permission: "flows.chatbot" },
      { path: "/chatbot/ai", permission: "chatbot.ai" },
      { path: "/chatbot/transfers", permission: "transfers" },
      { path: "/whatsapp/contacts", permission: "contacts" },
      { path: "/whatsapp/canned-responses", permission: "canned_responses" },
      { path: "/whatsapp/saved-contents", permission: "saved_contents" },
      { path: "/whatsapp/closed-chats", permission: "chat" },
      { path: "/whatsapp/whatsapp-filter", permission: "wa_filter" },
      { path: "/whatsapp/group-search", permission: "campaigns" },
      { path: "/whatsapp/agent-selection", permission: "agent_selection" },
    ],
  },
  {
    path: "/facebook",
    permission: "chat",
    childPaths: [
      { path: "/facebook/page-search", permission: "chat" },
      { path: "/facebook/comments", permission: "chat" },
      { path: "/facebook/people-search", permission: "chat" },
      { path: "/facebook/group-search", permission: "chat" },
      { path: "/facebook/extract-likes", permission: "chat" },
      { path: "/facebook/page-messengers", permission: "chat" },
      { path: "/facebook/extract-data", permission: "chat" },
      { path: "/facebook/auto-share", permission: "chat" },
      { path: "/facebook/retargeting", permission: "chat" },
      { path: "/facebook/accounts", permission: "accounts" },
    ],
  },
  {
    path: "/settings",
    permission: "settings.general",
    childPaths: [
      { path: "/settings", permission: "settings.general" },
      { path: "/settings", permission: "settings.uploads_cleanup" },
      { path: "/whatsapp/instances", permission: "accounts" },
      { path: "/settings/accounts", permission: "accounts" },
      { path: "/settings/tags", permission: "tags" },
      { path: "/settings/teams", permission: "teams" },
      { path: "/settings/users", permission: "users" },
      { path: "/settings/roles", permission: "roles" },
      { path: "/settings/audit-log", permission: "audit" },
      { path: "/settings/api-keys", permission: "api_keys" },
      { path: "/settings/webhooks", permission: "webhooks" },
      { path: "/settings/custom-actions", permission: "custom_actions" },
      { path: "/settings/sso", permission: "settings.sso" },
    ],
  },
];

function normalizedRoleName(
  authStore: ReturnType<typeof useAuthStore>,
): string {
  return String(authStore.userRole || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "_");
}

function isAdminUser(authStore: ReturnType<typeof useAuthStore>): boolean {
  const roleName = normalizedRoleName(authStore);
  return (
    authStore.user?.is_super_admin === true ||
    roleName === "admin" ||
    roleName === "super_admin" ||
    roleName === "super-admin"
  );
}

function isManagerOrAdminUser(
  authStore: ReturnType<typeof useAuthStore>,
): boolean {
  const roleName = normalizedRoleName(authStore);
  return (
    authStore.user?.is_super_admin === true ||
    roleName === "admin" ||
    roleName === "manager" ||
    roleName === "super_admin" ||
    roleName === "super-admin"
  );
}

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
  const licenseStore = useLicenseStore();

  try {
    await licenseStore.fetchBootstrap();
  } catch {
    if (to.name !== "activate") {
      return next({ name: "activate" });
    }
  }

  const allowWhileLocked =
    to.name === "activate" ||
    to.name === "license-settings" ||
    to.name === "sso-callback";

  if (licenseStore.isLocked && !allowWhileLocked) {
    return next({ name: "activate", query: { redirect: to.fullPath } });
  }

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
    const requiredPermissionsAny = to.meta.permissionsAny;
    if (requiredPermissionsAny?.length) {
      const hasAnyRequiredPermission = requiredPermissionsAny.some(
        (permission) => authStore.hasPermission(permission, "read"),
      );
      if (!hasAnyRequiredPermission) {
        const fallback = getFirstAccessibleRoute(authStore);
        console.warn(`Access denied to ${to.path}. Redirecting to ${fallback}`);
        if (fallback === to.path) {
          return next({ name: "profile" });
        }
        return next({ path: fallback });
      }
    }

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
    if (authStore.isAuthenticated && to.name === "activate") {
      if (isAdminUser(authStore)) {
        return next({ name: "license-settings" });
      }
      return next({ path: getFirstAccessibleRoute(authStore) });
    }

    if (
      !licenseStore.isLocked &&
      authStore.isAuthenticated &&
      (to.name === "login" || to.name === "register")
    ) {
      if (licenseStore.showQuotaOverage) {
        return next({ name: "license-cleanup" });
      }
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  const allowWhileOverage =
    to.name === "activate" ||
    to.name === "license-cleanup" ||
    to.name === "sso-callback";

  if (
    licenseStore.showQuotaOverage &&
    authStore.isAuthenticated &&
    !allowWhileOverage
  ) {
    return next({ name: "license-cleanup", query: { redirect: to.fullPath } });
  }

  const configStore = useConfigStore();
  const moduleKey = moduleKeyForPath(to.path);
  if (moduleKey && authStore.isAuthenticated) {
    await configStore.fetchConfig();
    if (!configStore.isModuleEnabled(moduleKey)) {
      const fallback = getFirstAccessibleRoute(authStore);
      return next({ path: fallback === to.path ? "/profile" : fallback });
    }
  }

  // Provider-based access: block Meta-only routes when using whatsmeow
  if (to.meta.metaOnly) {
    if (configStore.isWhatsmeow) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  // Role-based access: block admin-only routes for non-admins
  if (to.meta.adminOnly) {
    if (!isAdminUser(authStore)) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  if (to.meta.managerOrAdminOnly) {
    if (!isManagerOrAdminUser(authStore)) {
      return next({ path: getFirstAccessibleRoute(authStore) });
    }
  }

  next();
});

export default router;
