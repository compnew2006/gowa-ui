<script setup lang="ts">
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus, Trash2 } from 'lucide-vue-next'

/**
 * Generic key/value list editor backed by a plain object (v-model).
 * Used for API-call headers, webhook headers, and response mappings in the
 * flow node editor — all of which are `Record<string, string>` maps rendered
 * as an add-button + a row of key/value inputs with a delete action.
 */
const props = withDefaults(defineProps<{
  modelValue: Record<string, any>
  label: string
  addLabel?: string
  keyPlaceholder?: string
  valuePlaceholder?: string
  /** Apply font-mono to the key/value inputs (used by response mapping). */
  mono?: boolean
}>(), {
  addLabel: 'Add',
  keyPlaceholder: 'Key',
  valuePlaceholder: 'Value',
  mono: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, any>): void
}>()

function addEntry() {
  const next = { ...(props.modelValue || {}) }
  next[''] = ''
  emit('update:modelValue', next)
}

function removeEntry(key: string) {
  const next = { ...(props.modelValue || {}) }
  delete next[key]
  emit('update:modelValue', next)
}

function updateKey(oldKey: string, newKey: string) {
  if (oldKey === newKey) return
  const next = { ...(props.modelValue || {}) }
  next[newKey] = next[oldKey]
  delete next[oldKey]
  emit('update:modelValue', next)
}

function updateValue(key: string, value: string) {
  const next = { ...(props.modelValue || {}) }
  next[key] = value
  emit('update:modelValue', next)
}
</script>

<template>
  <div class="space-y-1.5">
    <div class="flex items-center justify-between">
      <Label class="text-xs">{{ label }}</Label>
      <Button variant="outline" size="sm" class="h-6 text-xs" @click="addEntry">
        <Plus class="h-3 w-3 mr-1" /> {{ addLabel }}
      </Button>
    </div>
    <slot name="description" />
    <div v-for="(val, key) in (modelValue || {})" :key="String(key)" class="flex items-center gap-1">
      <Input :model-value="String(key)" @update:model-value="(v: string) => updateKey(String(key), v)" :placeholder="keyPlaceholder" :class="mono ? 'h-7 text-xs flex-1 font-mono' : 'h-7 text-xs flex-1'" />
      <Input :model-value="String(val)" @update:model-value="(v: string) => updateValue(String(key), v)" :placeholder="valuePlaceholder" :class="mono ? 'h-7 text-xs flex-1 font-mono' : 'h-7 text-xs flex-1'" />
      <Button variant="ghost" size="icon" class="h-6 w-6" @click="removeEntry(String(key))">
        <Trash2 class="h-3 w-3 text-destructive" />
      </Button>
    </div>
  </div>
</template>
