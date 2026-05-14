<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import type { Component } from "vue"
import { cn } from "@/lib/utils"

interface EmptyStateProps {
  class?: HTMLAttributes["class"]
  icon?: Component
  title?: string
  description?: string
  /** Visual density: compact for sidebars/inline, default for standard, hero for full-page */
  size?: "compact" | "default" | "hero"
  /** Icon circle style: muted for neutral, primary for brand warmth */
  variant?: "muted" | "primary"
  /** Animate the icon with a gentle float */
  animated?: boolean
}

const props = withDefaults(defineProps<EmptyStateProps>(), {
  size: "default",
  variant: "muted",
  animated: false,
})
</script>

<template>
  <div
    :class="cn(
      'flex flex-col items-center text-center',
      size === 'compact' ? 'gap-2 p-6' : 'py-12 px-4',
      props.class,
    )"
    role="status"
    aria-live="polite"
  >
    <div
      v-if="props.icon || $slots.icon"
      :class="cn(
        'flex items-center justify-center rounded-full',
        size === 'hero'
          ? 'mb-5 h-16 w-16 rounded-2xl'
          : size === 'compact'
            ? 'h-10 w-10'
            : 'mb-4 h-12 w-12',
        variant === 'primary'
          ? size === 'hero' ? 'bg-primary/12' : 'bg-primary/8'
          : 'bg-muted',
        animated && 'animate-float',
      )"
    >
      <slot name="icon">
        <component
          :is="props.icon"
          :class="cn(
            size === 'hero' ? 'h-8 w-8' : size === 'compact' ? 'h-5 w-5' : 'h-6 w-6',
            variant === 'primary'
              ? size === 'hero' ? 'text-primary' : 'text-primary/40'
              : 'text-muted-foreground',
          )"
        />
      </slot>
    </div>
    <h3
      v-if="props.title || $slots.title"
      :class="cn(
        size === 'hero'
          ? 'mb-1.5 text-lg font-medium text-foreground'
          : 'text-sm font-medium text-muted-foreground',
        size === 'default' && 'text-lg font-semibold text-foreground',
      )"
    >
      <slot name="title">{{ props.title }}</slot>
    </h3>
    <p
      v-if="props.description || $slots.description"
      :class="cn(
        'max-w-sm',
        size === 'compact'
          ? 'text-xs text-muted-foreground/70'
          : 'mt-1 text-sm text-muted-foreground',
      )"
    >
      <slot name="description">{{ props.description }}</slot>
    </p>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>
