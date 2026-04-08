import { createApp } from "vue";
import { createPinia } from "pinia";
import { VueQueryPlugin } from "@tanstack/vue-query";

import App from "./App.vue";
import router from "./router";
import { i18n } from "./i18n";
import { initPostHog } from "./lib/posthog";
import { setLicenseLockedHandler } from "@/services/api";
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
