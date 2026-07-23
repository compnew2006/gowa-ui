<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AlertCircle, Loader2, RotateCw } from 'lucide-vue-next'
import type { Message } from '@/stores/contacts'

defineProps<{
  message: Message
  isRedownloading: boolean
}>()

defineEmits<{
  (e: 'retry'): void
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex items-center gap-3 px-3 py-3 bg-background/50 rounded-lg max-w-[280px]">
    <div class="h-10 w-10 rounded-full bg-red-900/30 light:bg-red-100 flex items-center justify-center shrink-0">
      <AlertCircle class="h-5 w-5 text-red-500" />
    </div>
    <div class="flex-1 min-w-0">
      <p class="text-sm font-medium truncate">{{ t('chat.mediaMissing') }}</p>
      <button
        type="button"
        class="text-xs text-primary hover:underline flex items-center gap-1 mt-0.5 disabled:opacity-50"
        :disabled="isRedownloading"
        @click="$emit('retry')"
      >
        <Loader2 v-if="isRedownloading" class="h-3 w-3 animate-spin" />
        <RotateCw v-else class="h-3 w-3" />
        {{ isRedownloading ? t('chat.retrying') : t('chat.retryDownload') }}
      </button>
    </div>
  </div>
</template>
