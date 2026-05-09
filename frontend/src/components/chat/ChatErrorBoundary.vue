<script setup lang="ts">
import { ref, onErrorCaptured } from "vue";
import { useI18n } from "vue-i18n";
import { Button } from "@/components/ui/button";
import { AlertCircle } from "lucide-vue-next";

const { t } = useI18n();

const hasError = ref(false);
const errorMessage = ref("");

onErrorCaptured((err, _instance, info) => {
  console.error("[ChatErrorBoundary] caught error:", err);
  console.error("[ChatErrorBoundary] error info:", info);

  hasError.value = true;
  errorMessage.value =
    err instanceof Error ? err.message : String(err);

  return false;
});

function retry() {
  hasError.value = false;
  errorMessage.value = "";
}
</script>

<template>
  <div v-if="hasError" class="flex h-full items-center justify-center bg-background p-8">
    <div class="flex max-w-md flex-col items-center gap-4 text-center">
      <div class="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
        <AlertCircle class="h-7 w-7 text-destructive" />
      </div>
      <div class="space-y-1.5">
        <h3 class="text-lg font-semibold text-foreground">
          {{ t('chat.errorBoundary.title') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ t('chat.errorBoundary.description') }}
        </p>
        <p
          v-if="errorMessage"
          class="mt-2 rounded-md bg-muted px-3 py-2 font-mono text-xs text-muted-foreground break-all"
        >
          {{ errorMessage }}
        </p>
      </div>
      <Button variant="outline" :aria-label="t('chat.errorBoundary.retry')" @click="retry">
        {{ t('chat.errorBoundary.retry') }}
      </Button>
    </div>
  </div>
  <slot v-else />
</template>
