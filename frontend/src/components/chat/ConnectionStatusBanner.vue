<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  useConnectionStatus,
  type ConnectionStatus,
} from "@/services/websocket";
import { WifiOff, Loader2, Wifi, X } from "lucide-vue-next";

const { t } = useI18n();
const status = useConnectionStatus();
const manuallyDismissed = ref(false);

type BannerConfig = {
  bgClass: string;
  textClass: string;
  icon: any;
  label: string;
};

const bannerConfig = computed<Partial<BannerConfig> & { label: string }>(() => {
  const configs: Record<ConnectionStatus, BannerConfig> = {
    connecting: {
      bgClass: "bg-yellow-500/10 border-yellow-500/30",
      textClass: "text-yellow-700 dark:text-yellow-400",
      icon: Loader2,
      label: t("chat.connecting"),
    },
    reconnected: {
      bgClass: "bg-green-500/10 border-green-500/30",
      textClass: "text-green-700 dark:text-green-400",
      icon: Wifi,
      label: t("chat.reconnected"),
    },
    disconnected: {
      bgClass: "bg-red-500/10 border-red-500/30",
      textClass: "text-red-700 dark:text-red-400",
      icon: WifiOff,
      label: t("chat.offline"),
    },
    connected: {
      bgClass: "",
      textClass: "",
      icon: null,
      label: "",
    },
  };
  return configs[status.value];
});

const visible = computed(() => {
  if (status.value === "connected") return false;
  if (manuallyDismissed.value && status.value === "disconnected") return false;
  return true;
});

function dismiss() {
  manuallyDismissed.value = true;
}
</script>

<template>
  <Transition
    enter-active-class="transition-all duration-200 ease-out"
    leave-active-class="transition-all duration-200 ease-in"
    enter-from-class="-translate-y-full opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="-translate-y-full opacity-0"
  >
    <div
      v-if="visible"
      :class="[
        'flex items-center justify-center gap-2 border-b px-4 py-1.5 text-xs font-medium',
        bannerConfig.bgClass,
        bannerConfig.textClass,
      ]"
      role="status"
      :aria-live="status === 'disconnected' ? 'assertive' : 'polite'"
    >
      <component
        :is="bannerConfig.icon"
        :class="[
          'h-3.5 w-3.5 shrink-0',
          status === 'connecting' ? 'animate-spin' : '',
        ]"
      />
      <span class="flex-1 text-center">{{ bannerConfig.label }}</span>
      <button
        type="button"
        class="shrink-0 rounded p-0.5 opacity-70 hover:opacity-100 transition-opacity"
        aria-label="Dismiss"
        @click="dismiss"
      >
        <X class="h-3 w-3" />
      </button>
    </div>
  </Transition>
</template>
