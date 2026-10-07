<script setup lang="ts">
import { DropdownMenuRoot } from 'reka-ui'

// `open: undefined` keeps the menu uncontrolled when no v-model:open is bound.
// Without it Vue's boolean casting turns a missing prop into `false`, which
// pins DropdownMenuRoot closed so the trigger never opens it.
const props = withDefaults(defineProps<{
  open?: boolean
}>(), {
  open: undefined,
})

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()
</script>

<template>
  <DropdownMenuRoot :open="open" @update:open="emit('update:open', $event)">
    <slot />
  </DropdownMenuRoot>
</template>
