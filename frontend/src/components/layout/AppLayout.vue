<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import { useLicenseStore } from "@/stores/license";
import { localeDirectionManager } from "@/i18n/locale-direction";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import {
  MessageSquare,
  Menu,
  Pin,
  PinOff,
  ShieldAlert,
  X,
} from "lucide-vue-next";
import { wsService } from "@/services/websocket";
import { authService } from "@/services/api";
import OrganizationSwitcher from "./OrganizationSwitcher.vue";
import UserMenu from "./UserMenu.vue";
import { navigationItems } from "./navigation";

const { locale, t } = useI18n(); // Enable localized banner strings in template

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const configStore = useConfigStore();
const licenseStore = useLicenseStore();
const SIDEBAR_PINNED_STORAGE_KEY = "layout.sidebarPinnedClosed";
const LEGACY_SIDEBAR_PINNED_OPEN_STORAGE_KEY = "layout.sidebarPinnedOpen";
const pinnedClosed = ref(false);
const hasDesktopHover = ref(false);
const hasDesktopFocusWithin = ref(false);
const sidebarOverlayOpenState = ref({
  organization: false,
  userMenu: false,
});

const expandedGroups = ref<Set<string>>(new Set());
const isMobileMenuOpen = ref(false);
const isRTL = computed(() => localeDirectionManager.isRTL(locale.value));
const hasDesktopOverlayOpen = computed(
  () =>
    sidebarOverlayOpenState.value.organization ||
    sidebarOverlayOpenState.value.userMenu,
);
const isDesktopSidebarExpanded = computed(
  () =>
    !pinnedClosed.value &&
    (hasDesktopHover.value ||
      hasDesktopFocusWithin.value ||
      hasDesktopOverlayOpen.value),
);
const isSidebarExpanded = computed(
  () => isMobileMenuOpen.value || isDesktopSidebarExpanded.value,
);
const showDesktopSidebarPinToggle = computed(
  () => pinnedClosed.value || isDesktopSidebarExpanded.value,
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
const normalizedRoleName = computed(() =>
  String(authStore.userRole || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "_"),
);
const isAdminUser = computed(
  () =>
    authStore.user?.is_super_admin === true ||
    normalizedRoleName.value === "admin" ||
    normalizedRoleName.value === "super_admin" ||
    normalizedRoleName.value === "super-admin",
);
const isManagerOrAdminUser = computed(
  () =>
    authStore.user?.is_super_admin === true ||
    ["admin", "manager", "super_admin", "super-admin"].includes(
      normalizedRoleName.value,
    ),
);
const licenseBannerDismissed = ref(false);
const canSeeLicenseBanner = computed(() => isAdminUser.value);
const licenseBannerStorageKey = computed(() =>
  authStore.user?.id ? `license.banner.dismissed.${authStore.user.id}` : "",
);
const showLicenseBanner = computed(() => {
  if (!canSeeLicenseBanner.value || licenseBannerDismissed.value) {
    return false;
  }
  return licenseStore.loaded && licenseStore.state.enabled;
});
const bannerVariant = computed(() => {
  if (licenseStore.isGrace || licenseStore.showQuotaOverage) {
    return "warning";
  }
  return "info";
});
const bannerPanelClass = computed(() => {
  if (bannerVariant.value === "warning") {
    return "border-yellow-500/50 bg-yellow-50 text-yellow-800 dark:border-yellow-500/30 dark:bg-yellow-950 dark:text-yellow-200";
  }
  return "border-blue-500/50 bg-blue-50 text-blue-800 dark:border-blue-500/30 dark:bg-blue-950 dark:text-blue-200";
});
const bannerTitle = computed(() => {
  if (licenseStore.isGrace) {
    return t("licenseSettings.banner.titleGrace");
  }
  if (licenseStore.showQuotaOverage) {
    return t("licenseSettings.banner.titleQuotaOverage");
  }
  if (licenseStore.showExpiryWarning) {
    return t("licenseSettings.banner.titleExpiry");
  }
  return t("licenseSettings.banner.titleActive");
});
const bannerMessage = computed(() => {
  if (licenseStore.isGrace) {
    return t("licenseSettings.banner.messageGrace", {
      deadline:
        licenseStore.state.grace_deadline ||
        t("licenseSettings.banner.theGraceDeadline"),
    });
  }
  if (licenseStore.showQuotaOverage) {
    return t("licenseSettings.banner.messageQuotaOverage");
  }
  if (licenseStore.daysUntilExpiry === null) {
    return t("licenseSettings.banner.messageExpiringSoon");
  }
  if (licenseStore.daysUntilExpiry === 0) {
    return t("licenseSettings.banner.messageExpiresToday");
  }
  if (!licenseStore.showExpiryWarning) {
    if (
      licenseStore.state.expires_at &&
      licenseStore.daysUntilExpiry !== null
    ) {
      return t("licenseSettings.banner.messageActiveWithDaysRemaining", {
        count: licenseStore.daysUntilExpiry,
      });
    }
    return t("licenseSettings.banner.messageActive");
  }
  return t("licenseSettings.banner.messageExpiresInDays", {
    count: licenseStore.daysUntilExpiry,
  });
});
const bannerLicenseMetaLabel = computed(() => {
  if (!licenseStore.state.license_kind) {
    return "";
  }
  const licenseKindLabel =
    licenseStore.state.license_kind === "trial"
      ? t("licenseSettings.licenseKind.trial")
      : licenseStore.state.license_kind === "paid"
        ? t("licenseSettings.licenseKind.paid")
        : licenseStore.state.license_kind;

  let durationLabel = licenseStore.state.duration_label || "";
  const normalizedDuration = durationLabel.trim().toLowerCase();
  if (normalizedDuration === "lifetime") {
    durationLabel = t("licenseSettings.duration.lifetime");
  } else {
    const dayMatch = normalizedDuration.match(/^(\d+)d$/);
    if (dayMatch) {
      const dayCount = Number(dayMatch[1]);
      durationLabel =
        dayCount === 1
          ? t("licenseSettings.duration.oneDay")
          : t("licenseSettings.duration.days", { count: dayCount });
    }
  }

  if (durationLabel) {
    return `${licenseKindLabel} • ${durationLabel}`;
  }
  return licenseKindLabel;
});
const bannerRemainingLabel = computed(() => {
  if (!licenseStore.state.expires_at) {
    return t("licenseSettings.duration.lifetime");
  }

  if (licenseStore.daysUntilExpiry === null) {
    return "";
  }

  if (licenseStore.daysUntilExpiry <= 0) {
    return t("licenseSettings.banner.remainingExpiresToday");
  }

  if (licenseStore.daysUntilExpiry === 1) {
    return t("licenseSettings.banner.remainingOneDay");
  }

  return t("licenseSettings.banner.remainingDays", {
    count: licenseStore.daysUntilExpiry,
  });
});
const bannerActionTo = computed(() =>
  licenseStore.showQuotaOverage ? "/license-cleanup" : "/settings/license",
);
const bannerActionLabel = computed(() =>
  licenseStore.showQuotaOverage
    ? t("licenseSettings.banner.resolveOverage")
    : t("licenseSettings.banner.manageLicense"),
);

function syncBannerDismissedState() {
  if (typeof window === "undefined") {
    licenseBannerDismissed.value = false;
    return;
  }

  const storageKey = licenseBannerStorageKey.value;
  licenseBannerDismissed.value =
    storageKey !== "" && window.sessionStorage.getItem(storageKey) === "true";
}

function dismissLicenseBanner() {
  licenseBannerDismissed.value = true;

  if (typeof window === "undefined") {
    return;
  }

  const storageKey = licenseBannerStorageKey.value;
  if (storageKey) {
    window.sessionStorage.setItem(storageKey, "true");
  }
}

watch(
  () => [licenseStore.isLocked, licenseStore.showQuotaOverage],
  ([locked, overage]) => {
    if (locked || overage) {
      wsService.disconnect();
    }
  },
);

watch(
  () => licenseBannerStorageKey.value,
  () => {
    syncBannerDismissedState();
  },
  { immediate: true },
);

// Connect WebSocket on mount using short-lived WS token
onMounted(() => {
  try {
    pinnedClosed.value =
      window.localStorage.getItem(SIDEBAR_PINNED_STORAGE_KEY) === "true";
    window.localStorage.removeItem(LEGACY_SIDEBAR_PINNED_OPEN_STORAGE_KEY);
  } catch {
    pinnedClosed.value = false;
  }

  if (authStore.isAuthenticated) {
    // Load app config (provider & feature flags)
    configStore.fetchConfig();

    if (!licenseStore.showQuotaOverage) {
      wsService.connect(async () => {
        try {
          const resp = await authService.getWSToken();
          return resp.data.data.token;
        } catch {
          return null;
        }
      });
    }
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
        if (child.path === "/whatsapp/campaigns" && !f.campaigns) return false;
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
            : item.activeMatchPaths
              ? item.activeMatchPaths.some((p) => route.path.startsWith(p))
              : route.path.startsWith(originalPath);

      return {
        ...item,
        path: effectivePath,
        active: isActive,
        children: filteredChildren,
      };
    });
});

const toggleGroup = (path: string) => {
  if (expandedGroups.value.has(path)) {
    expandedGroups.value.delete(path);
  } else {
    expandedGroups.value.add(path);
  }
};

const isGroupExpanded = (item: { path: string; children?: unknown[] }) => {
  if (expandedGroups.value.has(item.path)) return true;
  return false;
};

const handleDesktopSidebarMouseEnter = () => {
  if (pinnedClosed.value) {
    return;
  }

  hasDesktopHover.value = true;
};

const handleDesktopSidebarMouseLeave = () => {
  hasDesktopHover.value = false;
};

const handleDesktopSidebarFocusIn = () => {
  if (pinnedClosed.value) {
    return;
  }

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

const persistSidebarPinState = (nextValue: boolean) => {
  try {
    window.localStorage.setItem(SIDEBAR_PINNED_STORAGE_KEY, String(nextValue));
  } catch {
    // Ignore storage failures and keep the preference in memory.
  }
};

const toggleSidebarPin = () => {
  pinnedClosed.value = !pinnedClosed.value;

  if (pinnedClosed.value) {
    hasDesktopHover.value = false;
    hasDesktopFocusWithin.value = false;
    sidebarOverlayOpenState.value = {
      organization: false,
      userMenu: false,
    };
  }

  persistSidebarPinState(pinnedClosed.value);
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
      :data-sidebar-pinned="pinnedClosed ? 'true' : 'false'"
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
          v-if="!pinnedClosed"
          to="/"
          :class="[
            'flex min-w-0 items-center gap-2 overflow-hidden',
            isDesktopSidebarExpanded ? (isRTL ? 'pl-10' : 'pr-10') : 'mx-auto',
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
        <Button
          variant="ghost"
          size="icon"
          data-testid="sidebar-pin-toggle"
          :class="[
            'absolute inline-flex h-7 w-7 text-sidebar-foreground/60 transition-opacity duration-200 hover:bg-sidebar-accent hover:text-sidebar-foreground',
            pinnedClosed
              ? 'left-1/2 -translate-x-1/2'
              : isRTL
                ? 'left-2'
                : 'right-2',
            showDesktopSidebarPinToggle
              ? 'opacity-100'
              : 'pointer-events-none opacity-0',
          ]"
          :aria-label="
            pinnedClosed ? $t('nav.unpinSidebar') : $t('nav.pinSidebar')
          "
          :aria-hidden="!showDesktopSidebarPinToggle"
          :aria-pressed="pinnedClosed"
          :tabindex="showDesktopSidebarPinToggle ? 0 : -1"
          @click="toggleSidebarPin"
        >
          <PinOff v-if="pinnedClosed" class="h-3.5 w-3.5" />
          <Pin v-else class="h-3.5 w-3.5" />
        </Button>
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
            <component
              :is="item.children && item.children.length > 0 ? 'button' : 'RouterLink'"
              v-bind="item.children && item.children.length > 0
                ? { type: 'button' }
                : { to: item.path }
              "
              :class="[
                'flex items-center rounded-lg px-2.5 py-2 text-[13px] font-medium transition-all duration-200 w-full',
                item.active
                  ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/70 hover:text-sidebar-foreground',
                isSidebarExpanded ? 'gap-2.5' : 'justify-center gap-0',
                isRTL && isSidebarExpanded && 'text-right flex-row-reverse',
              ]"
              :role="item.children && item.children.length > 0 ? 'menuitem' : undefined"
              :aria-current="!item.children || item.children.length === 0 ? (item.active ? 'page' : undefined) : undefined"
              :aria-expanded="item.children && item.children.length > 0 ? isGroupExpanded(item) : undefined"
              :title="!isSidebarExpanded ? $t(item.name) : undefined"
              @click="item.children && item.children.length > 0 ? toggleGroup(item.path) : (isMobileMenuOpen = false)"
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
              <svg
                v-if="item.children && item.children.length > 0 && isSidebarExpanded"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                :class="[
                  'h-3.5 w-3.5 shrink-0 transition-transform duration-200',
                  isGroupExpanded(item) ? 'rotate-180' : '',
                  isRTL ? 'mr-auto' : 'ml-auto',
                ]"
              >
                <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
              </svg>
            </component>

            <!-- Submenu items -->
            <template v-if="item.children && item.children.length > 0 && isGroupExpanded(item) && isSidebarExpanded">
              <div
                :class="[
                  'px-2.5 pb-1 pt-1 text-xs font-medium text-muted-foreground/80',
                  isRTL ? 'mr-4 text-right' : 'ml-4 text-left',
                ]"
                role="presentation"
                :aria-label="$t(item.name)"
              >
                {{ $t(item.name) }}
              </div>
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
      <div
        v-if="showLicenseBanner"
        class="border-b border-border/70 bg-background/80 px-4 py-3 backdrop-blur-sm md:px-6"
      >
        <div
          :class="[
            'mx-auto flex max-w-6xl items-center gap-3 overflow-hidden rounded-lg border px-4 py-2 text-xs sm:text-sm',
            bannerPanelClass,
          ]"
        >
          <ShieldAlert class="h-4 w-4 shrink-0" />
          <div
            class="min-w-0 flex flex-1 items-center gap-2 overflow-hidden whitespace-nowrap"
          >
            <span class="shrink-0 font-medium">{{ bannerTitle }}</span>
            <Badge v-if="licenseStore.state.tier" variant="outline">
              {{ licenseStore.state.tier }}
            </Badge>
            <Badge v-if="bannerLicenseMetaLabel" variant="outline">
              {{ bannerLicenseMetaLabel }}
            </Badge>
            <Badge v-if="bannerRemainingLabel" variant="outline">
              {{ bannerRemainingLabel }}
            </Badge>
            <span class="min-w-0 truncate text-current/90">
              {{ bannerMessage }}
            </span>
          </div>
          <RouterLink
            :to="bannerActionTo"
            class="shrink-0 font-medium text-primary whitespace-nowrap underline-offset-4 hover:underline"
          >
            {{ bannerActionLabel }}
          </RouterLink>
          <Button
            variant="ghost"
            size="icon"
            class="h-7 w-7 shrink-0 text-current/70 hover:text-current"
            aria-label="Hide license banner"
            @click="dismissLicenseBanner"
          >
            <X class="h-4 w-4" />
          </Button>
        </div>
      </div>
      <RouterView />
    </main>
  </div>
</template>
