<script setup lang="ts">
import {
  ScrollAreaRoot,
  ScrollAreaViewport,
  ScrollAreaScrollbar,
  ScrollAreaThumb,
  ScrollAreaCorner
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
  orientation?: 'vertical' | 'horizontal' | 'both'
  /**
   * Reka ScrollArea type. Defaults to 'hover' (scrollbar appears only on
   * hover/scroll). Pass 'always' for lists where the scrollbar must stay
   * visible, e.g. the WhatsApp account access lists on /settings/users.
   */
  type?: 'auto' | 'always' | 'scroll' | 'hover'
}>()
</script>

<template>
  <ScrollAreaRoot :type="props.type ?? 'hover'" :class="cn('relative overflow-hidden', props.class)">
    <ScrollAreaViewport class="h-full w-full rounded-[inherit]">
      <slot />
    </ScrollAreaViewport>
    <ScrollAreaScrollbar
      v-if="orientation !== 'horizontal'"
      orientation="vertical"
      class="flex touch-none select-none transition-colors h-full w-2.5 border-l border-l-transparent p-[1px]"
    >
      <ScrollAreaThumb :class="cn('relative flex-1 rounded-full', props.type === 'always' ? 'bg-muted-foreground/50' : 'bg-border')" />
    </ScrollAreaScrollbar>
    <ScrollAreaScrollbar
      v-if="orientation !== 'vertical'"
      orientation="horizontal"
      class="flex touch-none select-none transition-colors flex-col h-2.5 border-t border-t-transparent p-[1px]"
    >
      <ScrollAreaThumb :class="cn('relative flex-1 rounded-full', props.type === 'always' ? 'bg-muted-foreground/50' : 'bg-border')" />
    </ScrollAreaScrollbar>
    <ScrollAreaCorner />
  </ScrollAreaRoot>
</template>
