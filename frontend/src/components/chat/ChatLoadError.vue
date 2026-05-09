<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { AlertCircle, RefreshCw } from "lucide-vue-next";
import { Button } from "@/components/ui/button";

defineProps<{
  isRetrying?: boolean;
}>();

defineEmits<{
  retry: [];
}>();

const { t } = useI18n();
</script>

<template>
  <div class="flex h-full items-center justify-center text-muted-foreground">
    <div class="text-center">
      <div
        class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-destructive/12 shadow-sm"
      >
        <AlertCircle class="h-8 w-8 text-destructive" />
      </div>
      <h3 class="mb-1 text-lg font-medium text-foreground">
        {{ t("chat.loadErrorTitle") }}
      </h3>
      <p class="mb-4 text-sm text-muted-foreground">
        {{ t("chat.loadErrorDescription") }}
      </p>
      <Button variant="outline" size="sm" :disabled="isRetrying" :aria-label="$t('chat.retry')" @click="$emit('retry')">
        <RefreshCw class="mr-2 h-4 w-4" :class="{ 'animate-spin': isRetrying }" />
        {{ isRetrying ? t("common.loading") : t("chat.retry") }}
      </Button>
    </div>
  </div>
</template>
