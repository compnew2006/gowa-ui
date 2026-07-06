<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";

const props = defineProps<{
  modelValue?: string;
  placeholder?: string;
  disabled?: boolean;
  rows?: number;
  class?: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const modelValue = computed({
  get: () => props.modelValue ?? "",
  set: (value) => emit("update:modelValue", value),
});
</script>

<template>
  <textarea
    v-model="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :rows="rows ?? 3"
    :class="
      cn(
        'flex min-h-[96px] w-full rounded-xl border border-input bg-input px-3.5 py-2.5 text-sm text-foreground shadow-sm ring-offset-background transition-[background-color,border-color,box-shadow] placeholder:text-muted-foreground hover:border-ring/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/15 focus-visible:border-ring/70 disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
