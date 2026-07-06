<script setup lang="ts">
import { computed } from 'vue'
import { useInstancesStore } from '@/stores/instances'
import {
  getInstanceTagLabel,
  getInstanceTagPresetByKey,
  resolveInstanceTagDisplayMode,
  resolveInstanceTagColorKey,
  type InstanceTagDisplayMode
} from '@/lib/instance-tag'

const props = withDefaults(defineProps<{
  instanceId?: string
  fallbackLabel?: string
  direction?: 'incoming' | 'outgoing'
  displayMode?: InstanceTagDisplayMode
  placement?: 'message' | 'sidebar'
}>(), {
  direction: 'incoming',
  placement: 'message'
})

const instancesStore = useInstancesStore()

const instanceIndex = computed(() => {
  if (!props.instanceId) return 0
  return instancesStore.instances.findIndex(item => item.id === props.instanceId)
})

const instance = computed(() => {
  if (!props.instanceId) return null
  return instancesStore.instances.find(item => item.id === props.instanceId) || null
})

const colorPreset = computed(() => {
  const fallbackIndex = instanceIndex.value >= 0 ? instanceIndex.value : 0
  const key = resolveInstanceTagColorKey(instance.value, fallbackIndex)
  return getInstanceTagPresetByKey(key)
})

const label = computed(() => {
  if (!props.instanceId) return ''
  const fallbackLabel = typeof props.fallbackLabel === 'string'
    ? props.fallbackLabel.trim()
    : ''
  if (!instance.value) {
    if (fallbackLabel) return fallbackLabel
    return `Instance ${props.instanceId.slice(0, 8)}`
  }
  const resolvedDisplayMode = props.displayMode || resolveInstanceTagDisplayMode(instance.value, 'name')
  return getInstanceTagLabel(instance.value, resolvedDisplayMode)
})

const showTag = computed(() => Boolean(props.instanceId && label.value))
</script>

<template>
  <div
    v-if="showTag"
    data-instance-tag="true"
    :data-placement="props.placement"
    :title="label"
    :class="[
      'inline-flex items-center gap-1 rounded-full border font-medium',
      colorPreset.badgeClass,
      props.placement === 'sidebar'
        ? 'max-w-[130px] px-1.5 py-0 text-[9px] shrink-0'
        : 'max-w-[220px] px-2 py-0.5 text-[10px] mb-1.5',
      props.placement === 'message'
        ? (props.direction === 'outgoing'
          ? 'shadow-inner shadow-black/10'
          : 'shadow-sm shadow-black/10 light:shadow-none')
        : ''
    ]"
  >
    <span :class="['h-1.5 w-1.5 shrink-0 rounded-full', colorPreset.dotClass]" />
    <span class="truncate">{{ label }}</span>
  </div>
</template>
