<script setup lang="ts">
import { ref, onErrorCaptured, watch } from "vue";
import { RouterView } from "vue-router";
import { Toaster } from "vue-sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useColorMode } from "@/composables/useColorMode";
import { useAuthStore } from "@/stores/auth";
import { AlertCircle, RefreshCw } from "lucide-vue-next";
import { Button } from "@/components/ui/button";

// Initialize color mode early
const { resolvedColorMode, hydrateFromUserSettings, initializeAppearance } =
  useColorMode();
const authStore = useAuthStore();

const hasError = ref(false);
const errorInfo = ref<string | null>(null);

initializeAppearance();

watch(
  () => authStore.user?.settings,
  (settings) => {
    if (authStore.user) {
      hydrateFromUserSettings(settings);
    }
  },
  { immediate: true },
);

onErrorCaptured((err) => {
  console.error("Captured global error:", err);
  hasError.value = true;
  errorInfo.value = err instanceof Error ? err.message : String(err);
  return false; // block error from reaching parent
});

function reloadPage() {
  window.location.reload();
}
</script>

<template>
  <TooltipProvider>
    <div class="min-h-screen bg-background font-sans antialiased">
      <div
        v-if="hasError"
        class="fixed inset-0 z-50 flex items-center justify-center bg-background p-6"
      >
        <div
          class="max-w-md w-full p-8 text-center border rounded-xl bg-card shadow-2xl space-y-6"
        >
          <div class="flex justify-center">
            <div class="p-4 rounded-full bg-destructive/10 text-destructive">
              <AlertCircle class="h-12 w-12" />
            </div>
          </div>
          <div class="space-y-2">
            <h1 class="text-2xl font-bold tracking-tight">
              Something went wrong
            </h1>
            <p class="text-muted-foreground">
              The application encountered a fatal error and could not continue.
            </p>
          </div>
          <div
            v-if="errorInfo"
            class="p-3 bg-muted rounded text-[11px] font-mono text-left overflow-auto max-h-32 opacity-70"
          >
            {{ errorInfo }}
          </div>
          <Button @click="reloadPage" class="w-full gap-2">
            <RefreshCw class="h-4 w-4" />
            Reload Application
          </Button>
        </div>
      </div>
      <RouterView v-else />
      <Toaster
        position="bottom-right"
        richColors
        closeButton
        :theme="resolvedColorMode"
      />
    </div>
  </TooltipProvider>
</template>
