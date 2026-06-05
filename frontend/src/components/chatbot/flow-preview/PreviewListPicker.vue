<script setup lang="ts">
import { ref } from 'vue'
import type { ButtonConfig } from '@/types/flow-preview'
import { ExternalLink, X, List } from 'lucide-vue-next'

defineProps<{
  buttons: ButtonConfig[]
  disabled?: boolean
}>()

const emit = defineEmits<{
  select: [button: ButtonConfig]
}>()

const isOpen = ref(false)

function handleSelect(button: ButtonConfig) {
  emit('select', button)
  isOpen.value = false
}
</script>

<template>
  <div class="mt-1">
    <!-- Trigger Button -->
    <button
      class="w-full bg-wa-panel-bubble-in text-wa-panel-accent text-sm font-medium py-2.5 px-4 rounded-lg shadow-sm border-0 flex items-center justify-center gap-2 transition-colors"
      :class="{
        'hover:bg-wa-panel-hover cursor-pointer': !disabled,
        'opacity-50 cursor-not-allowed': disabled
      }"
      :disabled="disabled"
      @click="isOpen = !isOpen"
    >
      <List class="h-4 w-4" />
      Select an option
    </button>

    <!-- List Picker Overlay - renders via slot in parent -->
    <Teleport to="#preview-phone-frame" :disabled="!isOpen">
      <div
        v-if="isOpen"
        class="absolute inset-0 z-10 flex flex-col"
      >
        <!-- Backdrop -->
        <div
          class="flex-1 bg-black/50"
          @click="isOpen = false"
        />

        <!-- Panel -->
        <div class="bg-wa-panel-sheet rounded-t-2xl overflow-hidden animate-slide-up">
          <!-- Header -->
          <div class="bg-wa-panel-header dark:bg-wa-panel-accent text-white px-4 py-3 flex items-center justify-between">
            <button
              class="p-1 hover:bg-white/10 rounded transition-colors"
              @click="isOpen = false"
            >
              <X class="h-5 w-5" />
            </button>
            <span class="font-medium text-sm">Select an option</span>
            <div class="w-7" />
          </div>

          <!-- Options List -->
          <div class="max-h-[250px] overflow-y-auto">
            <div
              v-for="(btn, idx) in buttons"
              :key="btn.id"
              class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 last:border-0 hover:bg-wa-panel-hover cursor-pointer flex items-center gap-3 transition-colors"
              @click="handleSelect(btn)"
            >
              <div
                v-if="btn.type === 'url'"
                class="w-5 h-5 flex items-center justify-center flex-shrink-0 text-wa-panel-accent"
              >
                <ExternalLink class="h-4 w-4" />
              </div>
              <div
                v-else
                class="w-5 h-5 rounded-full border-2 border-wa-panel-accent flex items-center justify-center flex-shrink-0"
              >
                <span class="text-[10px] text-wa-panel-accent font-medium">{{ idx + 1 }}</span>
              </div>
              <span class="text-sm text-gray-800 dark:text-gray-200 flex-1">
                {{ btn.title || `Option ${idx + 1}` }}
              </span>
              <ExternalLink v-if="btn.type === 'url'" class="h-3 w-3 text-gray-400" />
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
@keyframes slide-up {
  from {
    transform: translateY(100%);
  }
  to {
    transform: translateY(0);
  }
}

.animate-slide-up {
  animation: slide-up 0.2s ease-out;
}
</style>
