<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import { localeDirectionManager } from "@/i18n/locale-direction";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { MessageSquare, Menu, X } from "lucide-vue-next";
import { wsService } from "@/services/websocket";
import { authService } from "@/services/api";
import OrganizationSwitcher from "./OrganizationSwitcher.vue";
import UserMenu from "./UserMenu.vue";
import { navigationItems } from "./navigation";

const { locale } = useI18n(); // Enable $t() in template

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const configStore = useConfigStore();
const hasDesktopHover = ref(false);
const hasDesktopFocusWithin = ref(false);
const sidebarOverlayOpenState = ref({
  organization: false,
  userMenu: false,
});
const isMobileMenuOpen = ref(false);
const isRTL = computed(() => localeDirectionManager.isRTL(locale.value));
const hasDesktopOverlayOpen = computed(
  () =>
    sidebarOverlayOpenState.value.organization ||
    sidebarOverlayOpenState.value.userMenu,
);
const isDesktopSidebarExpanded = computed(
  () =>
    hasDesktopHover.value ||
    hasDesktopFocusWithin.value ||
    hasDesktopOverlayOpen.value,
);
const isSidebarExpanded = computed(
  () => isMobileMenuOpen.value || isDesktopSidebarExpanded.value,
);
const mainContentOffsetClass = computed(() =>
  isRTL.value ? "md:pr-16" : "md:pl-16",
);
const desktopSidebarPositionClass = computed(() =>
  isRTL.value ? "right-0 border-l" : "left-0 border-r",
);
const mobileSidebarClosedClass = computed(() =>
  isRTL.value
    ? "translate-x-full md:translate-x-0"
    : "-translate-x-full md:translate-x-0",
);
const isAdminUser = computed(
  () =>
    authStore.user?.is_super_admin === true ||
    (authStore.userRole || "").toLowerCase() === "admin",
);
const isManagerOrAdminUser = computed(
  () =>
    authStore.user?.is_super_admin === true ||
    ["admin", "manager"].includes((authStore.userRole || "").toLowerCase()),
);

// Connect WebSocket on mount using short-lived WS token
onMounted(() => {
  if (authStore.isAuthenticated) {
    // Load app config (provider & feature flags)
    configStore.fetchConfig();

    wsService.connect(async () => {
      try {
        const resp = await authService.getWSToken();
        return resp.data.data.token;
      } catch {
        return null;
      }
    });
  }
});

// Meta-only nav paths that should be hidden when provider is whatsmeow
const metaOnlyPaths = new Set([
  "/templates",
  "/flows",
  "/analytics/meta-insights",
]);
// Meta-only child paths within settings
const metaOnlyChildPaths = new Set(["/settings/accounts"]);

// Filter navigation based on user permissions AND provider features
const navigation = computed(() => {
  const f = configStore.features;
  return navigationItems
    .filter((item) => {
      if (item.adminOnly && !isAdminUser.value) {
        return false;
      }
      if (item.managerOrAdminOnly && !isManagerOrAdminUser.value) {
        return false;
      }
      // Hide entire nav items that are Meta-only when provider is whatsmeow
      if (
        metaOnlyPaths.has(item.path) &&
        !f.templates &&
        !f.flows &&
        !f.campaigns &&
        !f.meta_insights
      ) {
        // Check specific feature per path
        if (item.path === "/templates" && !f.templates) return false;
        if (item.path === "/flows" && !f.flows) return false;
        if (item.path === "/campaigns" && !f.campaigns) return false;
        if (item.path === "/analytics/meta-insights" && !f.meta_insights)
          return false;
      } else if (metaOnlyPaths.has(item.path)) {
        if (item.path === "/templates" && !f.templates) return false;
        if (item.path === "/flows" && !f.flows) return false;
        if (item.path === "/campaigns" && !f.campaigns) return false;
        if (item.path === "/analytics/meta-insights" && !f.meta_insights)
          return false;
      }
      // Permission-based filter
      if (item.childPermissions) {
        return item.childPermissions.some((p) =>
          authStore.hasPermission(p, "read"),
        );
      }
      return (
        !item.permission || authStore.hasPermission(item.permission, "read")
      );
    })
    .map((item) => {
      // Filter children that are Meta-only
      let filteredChildren = item.children?.filter((child) => {
        if (metaOnlyChildPaths.has(child.path) && !f.business_profile)
          return false;
        if (child.adminOnly && !isAdminUser.value) return false;
        if (child.managerOrAdminOnly && !isManagerOrAdminUser.value)
          return false;
        return (
          !child.permission || authStore.hasPermission(child.permission, "read")
        );
      });

      let effectivePath = item.path;
      if (
        item.childPermissions &&
        item.permission &&
        !authStore.hasPermission(item.permission, "read") &&
        filteredChildren?.length
      ) {
        effectivePath = filteredChildren[0].path;
      }

      const originalPath = item.path;
      const isActive =
        originalPath === "/dashboard"
          ? route.name === "dashboard"
          : originalPath === "/chat"
            ? route.name === "chat" || route.name === "chat-conversation"
            : route.path.startsWith(originalPath);

      return {
        ...item,
        path: effectivePath,
        active: isActive,
        children: filteredChildren,
      };
    });
});

const handleDesktopSidebarMouseEnter = () => {
  hasDesktopHover.value = true;
};

const handleDesktopSidebarMouseLeave = () => {
  hasDesktopHover.value = false;
};

const handleDesktopSidebarFocusIn = () => {
  hasDesktopFocusWithin.value = true;
};

const handleDesktopSidebarFocusOut = (event: FocusEvent) => {
  const currentTarget = event.currentTarget as HTMLElement | null;
  const nextTarget = event.relatedTarget as Node | null;

  if (currentTarget?.contains(nextTarget)) {
    return;
  }

  hasDesktopFocusWithin.value = false;
};

const handleSidebarOverlayOpenChange = (
  key: keyof typeof sidebarOverlayOpenState.value,
  open: boolean,
) => {
  sidebarOverlayOpenState.value = {
    ...sidebarOverlayOpenState.value,
    [key]: open,
  };
};

const handleLogout = async () => {
  await authStore.logout();
  router.push("/login");
};
</script>

<template>
  <div class="relative h-screen overflow-hidden bg-background text-foreground">
    <!-- Skip link for accessibility -->
    <a href="#main-content" class="skip-link">{{ $t("nav.skipToMain") }}</a>

    <!-- Mobile header -->
    <header
      class="fixed top-0 left-0 right-0 z-50 flex h-12 items-center justify-between border-b border-border bg-background/90 px-3 backdrop-blur-sm md:hidden"
    >
      <RouterLink to="/" class="flex items-center gap-2">
        <div
          class="flex h-7 w-7 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm"
        >
          <MessageSquare class="h-4 w-4 text-white" />
        </div>
        <span class="text-sm font-semibold text-foreground">Whatomate</span>
      </RouterLink>
      <Button
        variant="ghost"
        size="icon"
        class="h-8 w-8 text-muted-foreground hover:text-foreground hover:bg-accent"
        aria-label="Toggle menu"
        :aria-expanded="isMobileMenuOpen"
        @click="isMobileMenuOpen = !isMobileMenuOpen"
      >
        <X v-if="isMobileMenuOpen" class="h-5 w-5" />
        <Menu v-else class="h-5 w-5" />
      </Button>
    </header>

    <!-- Mobile menu overlay -->
    <div
      v-if="isMobileMenuOpen"
      data-testid="mobile-menu-overlay"
      class="fixed inset-0 z-40 bg-black/60 light:bg-black/30 backdrop-blur-sm md:hidden"
      @click="isMobileMenuOpen = false"
    />

    <!-- Sidebar -->
    <aside
      data-testid="app-sidebar"
      :data-sidebar-state="isSidebarExpanded ? 'expanded' : 'collapsed'"
      :class="[
        'group/sidebar fixed inset-y-0 z-40 flex flex-col overflow-hidden border-sidebar-border bg-sidebar text-sidebar-foreground transition-[width,transform] duration-300 ease-out',
        desktopSidebarPositionClass,
        isMobileMenuOpen ? 'translate-x-0' : mobileSidebarClosedClass,
        isSidebarExpanded ? 'w-64 shadow-2xl md:shadow-none' : 'w-16',
      ]"
      role="navigation"
      aria-label="Main navigation"
      @mouseenter="handleDesktopSidebarMouseEnter"
      @mouseleave="handleDesktopSidebarMouseLeave"
      @focusin="handleDesktopSidebarFocusIn"
      @focusout="handleDesktopSidebarFocusOut"
    >
      <!-- Logo (hidden on mobile, shown in header instead) -->
      <div
        class="relative hidden h-12 items-center border-b border-sidebar-border px-3 md:flex"
      >
        <RouterLink
          to="/"
          :class="[
            'flex min-w-0 items-center gap-2 overflow-hidden',
            isSidebarExpanded ? '' : 'mx-auto',
          ]"
        >
          <div
            class="flex h-7 w-7 items-center justify-center rounded-full bg-sidebar-primary text-sidebar-primary-foreground shadow-sm"
          >
            <MessageSquare class="h-4 w-4 text-white" />
          </div>
          <span
            :class="[
              'overflow-hidden whitespace-nowrap text-sm font-semibold text-sidebar-foreground transition-[max-width,opacity] duration-200',
              isSidebarExpanded ? 'max-w-32 opacity-100' : 'max-w-0 opacity-0',
            ]"
          >
            Whatomate
          </span>
        </RouterLink>
      </div>
      <!-- Mobile logo spacer -->
      <div class="h-12 md:hidden" />

      <!-- Organization Switcher (Super Admin only) -->
      <OrganizationSwitcher
        :expanded="isSidebarExpanded"
        @overlay-open-change="
          (open) => handleSidebarOverlayOpenChange('organization', open)
        "
      />

      <!-- Navigation -->
      <ScrollArea class="flex-1 py-2">
        <nav class="space-y-0.5 px-2" role="menubar">
          <template v-for="item in navigation" :key="item.path">
            <RouterLink
              :to="item.path"
              :class="[
                'flex items-center rounded-lg px-2.5 py-2 text-[13px] font-medium transition-all duration-200',
                item.active
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/70 hover:text-sidebar-foreground',
                isSidebarExpanded ? 'gap-2.5' : 'justify-center gap-0',
                isRTL && isSidebarExpanded && 'text-right flex-row-reverse',
              ]"
              role="menuitem"
              :aria-current="item.active ? 'page' : undefined"
              :title="!isSidebarExpanded ? $t(item.name) : undefined"
              @click="isMobileMenuOpen = false"
            >
              <component
                :is="item.icon"
                class="h-4 w-4 shrink-0"
                aria-hidden="true"
              />
              <span
                :class="[
                  'overflow-hidden whitespace-nowrap transition-[max-width,opacity] duration-200',
                  isSidebarExpanded
                    ? 'max-w-40 flex-1 opacity-100'
                    : 'max-w-0 opacity-0',
                  isRTL && isSidebarExpanded ? 'text-right' : 'text-left',
                ]"
              >
                {{ $t(item.name) }}
              </span>
            </RouterLink>

            <!-- Submenu items -->
            <template v-if="item.children && item.active && isSidebarExpanded">
              <RouterLink
                v-for="child in item.children"
                :key="child.path"
                :to="child.path"
                :class="[
                  'flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-[13px] font-medium transition-all duration-200',
                  isRTL ? 'mr-4 text-right flex-row-reverse' : 'ml-4 text-left',
                  route.path === child.path
                    ? 'bg-sidebar-accent/80 text-sidebar-accent-foreground'
                    : 'text-sidebar-foreground/55 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground',
                ]"
                role="menuitem"
                :aria-current="route.path === child.path ? 'page' : undefined"
                @click="isMobileMenuOpen = false"
              >
                <component
                  :is="child.icon"
                  class="h-3.5 w-3.5 shrink-0"
                  aria-hidden="true"
                />
                <span :class="['flex-1', isRTL ? 'text-right' : 'text-left']">{{
                  $t(child.name)
                }}</span>
              </RouterLink>
            </template>
          </template>
        </nav>
      </ScrollArea>

      <!-- User Menu -->
      <UserMenu
        :expanded="isSidebarExpanded"
        @overlay-open-change="
          (open) => handleSidebarOverlayOpenChange('userMenu', open)
        "
        @logout="handleLogout"
      />
    </aside>

    <!-- Main content -->
    <main
      id="main-content"
      :class="[
        'h-full overflow-hidden bg-background pt-12 md:pt-0',
        mainContentOffsetClass,
      ]"
      role="main"
    >
      <RouterView />
    </main>
  </div>
</template>
