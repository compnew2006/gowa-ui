<script setup lang="ts">
import { useI18n } from "vue-i18n";

const { t } = useI18n();

defineProps<{
  contactName: string;
}>();
</script>

<template>
  <div
    class="flex items-center gap-1.5 px-4 py-1.5 text-xs text-muted-foreground"
    role="status"
    aria-live="polite"
  >
    <span class="flex items-center gap-0.5">
      <span
        class="h-1 w-1 animate-bounce rounded-full bg-muted-foreground/60"
        style="animation-delay: 0ms"
      />
      <span
        class="h-1 w-1 animate-bounce rounded-full bg-muted-foreground/60"
        style="animation-delay: 150ms"
      />
      <span
        class="h-1 w-1 animate-bounce rounded-full bg-muted-foreground/60"
        style="animation-delay: 300ms"
      />
    </span>
    <span>{{ contactName }} {{ t("chat.isTyping") }}</span>
  </div>
</template>
