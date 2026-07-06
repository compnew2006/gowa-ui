import { createApp } from "vue";
import { createPinia } from "pinia";
import { VueQueryPlugin } from "@tanstack/vue-query";
import { toast } from "vue-sonner";

import App from "./App.vue";
import router from "./router";
import { i18n } from "./i18n";
import { initPostHog } from "./lib/posthog";
import {
  setSessionExpiredHandler,
} from "@/services/api";
import { useAuthStore } from "@/stores/auth";

import "./assets/fonts.css";
import "./assets/index.css";

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);
app.use(VueQueryPlugin);
app.use(i18n);

setSessionExpiredHandler(() => {
  // Record the logout server-side (writes the `logout` audit event) and clear
  // Pinia auth state. authStore.logout() already swallows errors and always
  // clears local state via clearAuth() in its finally block, so a 401 from an
  // already-dead refresh token still leaves the user logged out cleanly. The
  // backend records the audit event BEFORE the Redis revocation check, so it
  // fires even for already-revoked tokens.
  const authStore = useAuthStore(pinia);
  void authStore.logout();

  // Show localized "session expired" toast.
  toast.warning(i18n.global.t("auth.sessionExpired"));

  // Soft redirect to /login preserving the route the user was on, so a
  // successful re-login returns them where they were instead of /chat.
  if (router.currentRoute.value.name !== "login") {
    const redirectPath = router.currentRoute.value.fullPath;
    void router.push({ name: "login", query: { redirect: redirectPath } });
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
