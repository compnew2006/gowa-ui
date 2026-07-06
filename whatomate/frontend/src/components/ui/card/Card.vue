<script setup lang="ts">
import { cn } from "@/lib/utils";

const props = defineProps<{
  class?: string;
}>();
</script>

<template>
  <div
    :class="
      cn(
        'rounded-[calc(var(--radius)+0.35rem)] border border-border bg-card/95 text-card-foreground shadow-sm backdrop-blur-sm transition-colors',
        props.class,
      )
    "
  >
    <slot />
  </div>
</template>
