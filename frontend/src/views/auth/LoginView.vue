<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/services/api";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { toast } from "vue-sonner";
import { MessageSquare, Loader2 } from "lucide-vue-next";

const { t } = useI18n();

interface SSOProvider {
  provider: string;
  name: string;
  organization_slug?: string;
}

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const email = ref("");
const password = ref("");
const rememberCredentials = ref(false);
const isLoading = ref(false);
const ssoProviders = ref<SSOProvider[]>([]);
const REMEMBER_CREDENTIALS_KEY = "auth:remembered_login_credentials";

interface RememberedCredentials {
  email: string;
  password: string;
}

// SSO provider icons (using simple SVG paths)
const providerIcons: Record<string, string> = {
  google:
    "M12.545,10.239v3.821h5.445c-0.712,2.315-2.647,3.972-5.445,3.972c-3.332,0-6.033-2.701-6.033-6.032s2.701-6.032,6.033-6.032c1.498,0,2.866,0.549,3.921,1.453l2.814-2.814C17.503,2.988,15.139,2,12.545,2C7.021,2,2.543,6.477,2.543,12s4.478,10,10.002,10c8.396,0,10.249-7.85,9.426-11.748L12.545,10.239z",
  microsoft: "M11 11H3V3h8v8zm10 0h-8V3h8v8zM11 21H3v-8h8v8zm10 0h-8v-8h8v8z",
  github:
    "M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z",
  facebook:
    "M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z",
  custom:
    "M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z",
};

const providerColors: Record<string, string> = {
  google: "hover:border-[#ea4335]/35",
  microsoft: "hover:border-[#2563eb]/35",
  github: "hover:border-foreground/25",
  facebook: "hover:border-[#1877f2]/35",
  custom: "hover:border-primary/35",
};

onMounted(async () => {
  const rememberedRaw = localStorage.getItem(REMEMBER_CREDENTIALS_KEY);
  if (rememberedRaw) {
    try {
      const remembered = JSON.parse(
        rememberedRaw,
      ) as Partial<RememberedCredentials>;
      if (
        typeof remembered.email === "string" &&
        typeof remembered.password === "string"
      ) {
        email.value = remembered.email;
        password.value = remembered.password;
        rememberCredentials.value = true;
      } else {
        localStorage.removeItem(REMEMBER_CREDENTIALS_KEY);
      }
    } catch {
      localStorage.removeItem(REMEMBER_CREDENTIALS_KEY);
    }
  }

  // Check for SSO error in query params
  const ssoError = route.query.sso_error as string;
  if (ssoError) {
    toast.error(decodeURIComponent(ssoError));
    // Clear the error from URL
    router.replace({ query: { ...route.query, sso_error: undefined } });
  }

  // Fetch enabled SSO providers
  try {
    const response = await api.get("/auth/sso/providers");
    ssoProviders.value = response.data.data || [];
  } catch {
    ssoProviders.value = [];
  }
});

watch(rememberCredentials, (enabled) => {
  if (!enabled) {
    localStorage.removeItem(REMEMBER_CREDENTIALS_KEY);
  }
});

const handleLogin = async () => {
  if (!email.value || !password.value) {
    toast.error(t("auth.enterEmailPassword"));
    return;
  }

  isLoading.value = true;

  try {
    await authStore.login(email.value, password.value);
    if (rememberCredentials.value) {
      const rememberedCredentials: RememberedCredentials = {
        email: email.value,
        password: password.value,
      };
      localStorage.setItem(
        REMEMBER_CREDENTIALS_KEY,
        JSON.stringify(rememberedCredentials),
      );
    } else {
      localStorage.removeItem(REMEMBER_CREDENTIALS_KEY);
    }

    toast.success(t("auth.loginSuccess"));

    // Only honor a same-origin, non-/login redirect target to avoid open
    // redirects and login loops; otherwise go to the default landing route.
    const redirect = route.query.redirect as string;
    const safeRedirect =
      redirect &&
      redirect.startsWith("/") &&
      !redirect.startsWith("//") &&
      redirect !== "/login"
        ? redirect
        : "/";
    router.push(safeRedirect);
  } catch (error: any) {
    const message =
      error.response?.data?.message || t("auth.invalidCredentials");
    toast.error(message);
  } finally {
    isLoading.value = false;
  }
};

const initiateSSO = (provider: string, organizationSlug?: string) => {
  const basePath = ((window as any).__BASE_PATH__ ?? "").replace(/\/$/, "");
  const params = new URLSearchParams();
  if (organizationSlug) {
    params.set("org", organizationSlug);
  }
  const query = params.toString();
  window.location.href = `${basePath}/api/auth/sso/${provider}/init${query ? `?${query}` : ""}`;
};
</script>

<template>
  <div class="auth-shell">
    <div class="auth-card">
      <div class="p-8 space-y-1 text-center">
        <div class="flex justify-center mb-4">
          <div class="brand-badge">
            <MessageSquare class="h-7 w-7 text-white" />
          </div>
        </div>
        <h2 class="text-2xl font-bold tracking-tight text-foreground">
          {{ $t("auth.welcomeTitle") }}
        </h2>
        <p class="text-muted-foreground">
          {{ $t("auth.welcomeSubtitle") }}
        </p>
      </div>

      <form @submit.prevent="handleLogin">
        <div class="px-8 pb-4 space-y-4">
          <div class="space-y-2">
            <Label for="email" class="text-foreground/80">{{
              $t("common.email")
            }}</Label>
            <Input
              id="email"
              v-model="email"
              type="email"
              :placeholder="$t('auth.emailPlaceholder')"
              :disabled="isLoading"
              autocomplete="username"
            />
          </div>
          <div class="space-y-2">
            <Label for="password" class="text-foreground/80">{{
              $t("auth.password")
            }}</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              :placeholder="$t('auth.passwordPlaceholder')"
              :disabled="isLoading"
              autocomplete="current-password"
            />
          </div>
          <div class="flex items-center space-x-2">
            <Checkbox
              id="remember-credentials"
              :checked="rememberCredentials"
              :disabled="isLoading"
              @update:checked="rememberCredentials = $event === true"
            />
            <Label
              for="remember-credentials"
              class="cursor-pointer text-sm font-normal text-foreground/80"
            >
              {{ $t("auth.rememberMe") }}
            </Label>
          </div>
          <Button type="submit" class="w-full" :disabled="isLoading">
            <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("auth.signIn") }}
          </Button>
        </div>
      </form>

      <!-- SSO Section -->
      <div v-if="ssoProviders.length > 0" class="px-8 pb-4 space-y-3">
        <div class="relative my-2">
          <Separator class="bg-border" />
          <span
            class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-card px-2 text-xs text-muted-foreground"
          >
            {{ $t("auth.orContinueWith") }}
          </span>
        </div>

        <Button
          v-for="provider in ssoProviders"
          :key="provider.provider"
          variant="outline"
          class="w-full justify-start gap-3 bg-background/70 text-foreground transition-colors hover:bg-accent hover:text-foreground"
          :class="providerColors[provider.provider] || providerColors.custom"
          @click="initiateSSO(provider.provider, provider.organization_slug)"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
            <path
              :d="providerIcons[provider.provider] || providerIcons.custom"
            />
          </svg>
          {{ provider.name }}
        </Button>
      </div>

      <div class="px-8 pb-8">
        <p class="text-center text-sm text-muted-foreground">
          {{ $t("auth.noAccount") }}
          <RouterLink to="/register" class="text-primary hover:underline">
            {{ $t("auth.signUp") }}
          </RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>
