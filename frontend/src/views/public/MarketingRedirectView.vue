<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { ExternalLink, LogIn } from "lucide-vue-next";

import { Button } from "@/components/ui/button";
import {
  buildMarketingRedirectTarget,
  shouldAutoRedirect,
} from "@/lib/marketing-redirect";

const route = useRoute();
const marketingBaseUrl = import.meta.env.VITE_PUBLIC_MARKETING_BASE_URL ?? "";

const currentUrl = computed(() =>
  typeof window === "undefined" ? "" : window.location.href,
);

const targetUrl = computed(() => {
  if (typeof window === "undefined") {
    return null;
  }

  return buildMarketingRedirectTarget({
    marketingBaseUrl,
    currentPath: route.path,
    search: window.location.search,
    hash: window.location.hash,
    origin: window.location.origin,
  });
});

const autoRedirectEnabled = computed(() =>
  shouldAutoRedirect(targetUrl.value, currentUrl.value),
);

onMounted(() => {
  if (autoRedirectEnabled.value && targetUrl.value) {
    window.location.replace(targetUrl.value);
  }
});
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <div
      class="w-full max-w-2xl rounded-3xl border border-border bg-card p-8 shadow-sm"
    >
      <div class="space-y-4 text-center">
        <div
          class="inline-flex items-center rounded-full border border-amber-500/30 bg-amber-500/10 px-3 py-1 text-xs font-medium text-amber-500"
        >
          {{ $t("marketingRedirect.badge") }}
        </div>
        <h1 class="text-3xl font-semibold tracking-tight">
          {{ $t("marketingRedirect.title") }}
        </h1>
        <p class="text-sm leading-7 text-muted-foreground sm:text-base">
          {{ $t("marketingRedirect.description") }}
        </p>
        <p class="text-sm text-muted-foreground">
          {{ $t("marketingRedirect.routeHint", { route: route.path }) }}
        </p>
      </div>

      <div
        class="mt-8 space-y-4 rounded-2xl border border-dashed border-border bg-muted/30 p-5"
      >
        <p v-if="autoRedirectEnabled" class="text-sm text-foreground">
          {{ $t("marketingRedirect.redirecting") }}
        </p>
        <p v-else-if="targetUrl" class="text-sm text-muted-foreground">
          {{ $t("marketingRedirect.redirectReady") }}
        </p>
        <p v-else class="text-sm text-muted-foreground">
          {{ $t("marketingRedirect.notConfigured") }}
        </p>

        <div class="flex flex-col justify-center gap-3 sm:flex-row">
          <Button v-if="targetUrl" as-child>
            <a :href="targetUrl" rel="noreferrer">
              <ExternalLink class="mr-2 h-4 w-4" />
              {{ $t("marketingRedirect.openSite") }}
            </a>
          </Button>
          <RouterLink to="/login">
            <Button variant="outline">
              <LogIn class="mr-2 h-4 w-4" />
              {{ $t("marketingRedirect.goToLogin") }}
            </Button>
          </RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>
