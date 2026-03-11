<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { AvatarImage as AvatarImagePrimitive } from 'reka-ui'
import { cn } from '@/lib/utils'
import { hasFailedAvatarURL, markFailedAvatarURL } from './failed-avatar-urls'

const props = defineProps<{
  src?: string
  alt?: string
  class?: string
}>()

const normalizedSrc = computed(() => String(props.src ?? '').trim())
const hasLoadError = ref(false)

watch(
  normalizedSrc,
  (src) => {
    hasLoadError.value = src !== '' && hasFailedAvatarURL(src)
  },
  { immediate: true },
)

function handleImageError() {
  const src = normalizedSrc.value
  if (src === '') return
  markFailedAvatarURL(src)
  hasLoadError.value = true
}
</script>

<template>
  <AvatarImagePrimitive
    v-if="normalizedSrc !== '' && !hasLoadError"
    :src="normalizedSrc"
    :alt="alt || ''"
    :class="cn('aspect-square h-full w-full', props.class)"
    @error="handleImageError"
  />
</template>
