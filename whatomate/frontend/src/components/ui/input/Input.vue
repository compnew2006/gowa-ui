<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";

const props = defineProps<{
  modelValue?: string | number;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  class?: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const modelValue = computed({
  get: () => props.modelValue?.toString() ?? "",
  set: (value) => emit("update:modelValue", value),
});
</script>

<template>
  <input
    v-model="modelValue"
    :type="type ?? 'text'"
    :placeholder="placeholder"
    :disabled="disabled"
    :class="
      cn(
        'flex h-10 w-full rounded-lg border border-input bg-input px-3.5 py-2 text-sm text-foreground shadow-sm ring-offset-background transition-[background-color,border-color,box-shadow] placeholder:text-muted-foreground hover:border-ring/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/15 focus-visible:border-ring/70 disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
