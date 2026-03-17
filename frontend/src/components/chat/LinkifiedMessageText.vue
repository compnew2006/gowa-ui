<script setup lang="ts">
import { computed } from "vue";

import { segmentMessageLinks } from "@/lib/message-linkify";

const props = defineProps<{
  text: string;
}>();

const segments = computed(() => segmentMessageLinks(props.text));
</script>

<template>
  <template v-for="(segment, index) in segments" :key="`${segment.type}-${index}`">
    <a
      v-if="segment.type === 'link'"
      :href="segment.href"
      target="_blank"
      rel="noopener noreferrer"
      class="break-all rounded-sm underline underline-offset-2 transition-opacity hover:opacity-80 focus-visible:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-current"
      style="color: inherit"
    >
      {{ segment.text }}
    </a>
    <span v-else>{{ segment.text }}</span>
  </template>
</template>
