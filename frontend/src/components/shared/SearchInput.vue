<script setup lang="ts">
import { Input } from '@/components/ui/input'
import { Search, X } from 'lucide-vue-next'

const model = defineModel<string>({ default: '' })

withDefaults(defineProps<{
  placeholder?: string
  class?: string
}>(), {
  placeholder: 'Search...',
})

function clear() {
  model.value = ''
}
</script>

<template>
  <div class="relative" :class="$props.class">
    <Search class="absolute start-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" aria-hidden="true" />
    <Input
      v-model="model"
      :placeholder="placeholder"
      :aria-label="placeholder"
      class="ps-9 pe-8"
    />
    <button
      v-if="model"
      type="button"
      class="absolute end-2 top-1/2 -translate-y-1/2 h-5 w-5 rounded-full hover:bg-muted flex items-center justify-center"
      :aria-label="$t('common.clear')"
      @click="clear"
    >
      <X class="h-3 w-3 text-muted-foreground" aria-hidden="true" />
    </button>
  </div>
</template>
