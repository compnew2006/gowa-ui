<script setup lang="ts">
import { Handle, Position } from '@vue-flow/core'

interface OutputHandle {
  id: string
  label?: string
  title?: string
}

withDefaults(
  defineProps<{
    label: string
    headerClass?: string
    hasInput?: boolean
    outputHandles?: OutputHandle[]
  }>(),
  {
    headerClass: 'bg-slate-600',
    hasInput: true,
  }
)
</script>

<template>
  <div class="rounded-lg border bg-card text-card-foreground shadow-sm min-w-[180px] max-w-[240px] text-xs">
    <!-- Input Handle -->
    <Handle
      v-if="hasInput"
      type="target"
      :position="Position.Left"
      class="!bg-muted-foreground !w-2.5 !h-2.5 !-left-1.5"
    />

    <!-- Header -->
    <div
      class="px-3 py-1.5 rounded-t-lg font-medium text-white flex items-center justify-between gap-2"
      :class="headerClass"
    >
      <div class="flex items-center gap-1.5 truncate">
        <slot name="icon" />
        <span class="truncate">{{ label }}</span>
      </div>
    </div>

    <!-- Body -->
    <div class="p-2.5 space-y-1">
      <slot />
    </div>

    <!-- Output Handles -->
    <template v-if="outputHandles && outputHandles.length > 0">
      <div class="border-t px-2 py-1 space-y-1 bg-muted/20 rounded-b-lg">
        <div
          v-for="handle in outputHandles"
          :key="handle.id"
          class="relative flex items-center justify-end h-5 text-[10px] text-muted-foreground font-medium"
        >
          <span class="mr-2 truncate" :title="handle.title || handle.label">{{ handle.label || handle.id }}</span>
          <Handle
            type="source"
            :id="handle.id"
            :position="Position.Right"
            class="!bg-muted-foreground !w-2.5 !h-2.5 !-right-3.5"
          />
        </div>
      </div>
    </template>
    <template v-else-if="outputHandles === undefined">
      <!-- Default single source handle -->
      <Handle
        type="source"
        id="default"
        :position="Position.Right"
        class="!bg-muted-foreground !w-2.5 !h-2.5 !-right-1.5"
      />
    </template>
  </div>
</template>
