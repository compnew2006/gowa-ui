<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { Send } from "lucide-vue-next";

const { t } = useI18n();
</script>

<template>
  <div class="flex flex-1 items-center justify-center text-muted-foreground">
    <div class="text-center">
      <div
        class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/12 shadow-sm"
      >
        <Send class="h-8 w-8 text-primary" />
      </div>
      <h3 class="mb-1 text-lg font-medium text-foreground">
        {{ t("chat.selectConversation") }}
      </h3>
      <p class="text-sm text-muted-foreground">
        {{ t("chat.chooseContact") }}
      </p>
    </div>
  </div>
</template>
