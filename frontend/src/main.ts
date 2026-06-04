import { createApp } from "vue";
import { createPinia } from "pinia";
import { VueQueryPlugin } from "@tanstack/vue-query";
import { toast } from "vue-sonner";

import App from "./App.vue";
import router from "./router";
import { i18n } from "./i18n";
import { initPostHog } from "./lib/posthog";
import {
  setLicenseLockedHandler,
  setSessionExpiredHandler,
} from "@/services/api";
import { useAuthStore } from "@/stores/auth";
import { useLicenseStore } from "@/stores/license";

import "./assets/fonts.css";
import "./assets/index.css";

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);
app.use(VueQueryPlugin);
app.use(i18n);

const licenseStore = useLicenseStore(pinia);
setLicenseLockedHandler((error) => {
  const payload = (error.response?.data || {}) as Record<string, any>;
  const code =
    payload?.data?.code || payload?.error?.code || payload?.code || "";

  if (code === "license_quota_overage") {
    void licenseStore.fetchBootstrap(true).catch(() => {});
    if (router.currentRoute.value.name !== "license-cleanup") {
      void router.push("/license-cleanup");
    }
    return;
  }

  licenseStore.markLocked();
  if (router.currentRoute.value.name !== "activate") {
    void router.push("/activate");
  }
});

setSessionExpiredHandler(() => {
  // Clear Pinia auth state so the rest of the app sees the user as logged out
  // (handles both 401 from /api/me and from any other protected endpoint).
  const authStore = useAuthStore(pinia);
  authStore.clearAuth();

  // Show localized "session expired" toast.
  toast.warning(i18n.global.t("auth.sessionExpired"));

  // Soft redirect to /login (no hard reload, preserves the SPA).
  if (router.currentRoute.value.name !== "login") {
    void router.push("/login");
  }
});

initPostHog(app);

router.afterEach((to) => {
  const posthog = app.config.globalProperties.$posthog;
  if (posthog) {
    posthog.capture("$pageview", {
      $current_url: window.location.origin + to.fullPath,
    });
  }
});

app.mount("#app");
