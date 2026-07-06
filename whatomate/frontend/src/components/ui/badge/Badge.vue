<script setup lang="ts">
import type { HTMLAttributes } from "vue";
import { cn } from "@/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "border-primary/10 bg-primary/10 text-primary",
        secondary: "border-transparent bg-muted text-muted-foreground",
        destructive: "border-transparent bg-destructive/10 text-destructive",
        outline: "border-border text-muted-foreground",
        success:
          "border-transparent bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
        warning:
          "border-transparent bg-amber-500/15 text-amber-600 dark:text-amber-400",
        info: "border-transparent bg-primary/10 text-primary",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type BadgeVariants = VariantProps<typeof badgeVariants>;

const props = defineProps<{
  variant?: BadgeVariants["variant"];
  class?: HTMLAttributes["class"];
}>();
</script>

<template>
  <div :class="cn(badgeVariants({ variant }), props.class)">
    <slot />
  </div>
</template>
