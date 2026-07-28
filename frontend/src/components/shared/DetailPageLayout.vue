<script setup lang="ts">
import { ScrollArea } from '@/components/ui/scroll-area'
import PageHeader from './PageHeader.vue'
import ErrorState from './ErrorState.vue'
import { Loader2 } from 'lucide-vue-next'
import type { Component } from 'vue'

defineProps<{
  title: string
  description?: string
  icon?: Component
  iconGradient?: string
  backLink: string
  breadcrumbs?: Array<{ label: string; href?: string }>
  isLoading?: boolean
  isNotFound?: boolean
  notFoundTitle?: string
  notFoundDescription?: string
  /** When true, drops the 2/3 + 1/3 sidebar split and widens the content
   *  canvas so the page can lay out its own multi-column bento grid in the
   *  default slot. The sidebar slot still renders above the content when
   *  populated. Default false keeps the legacy 6xl + sidebar layout. */
  wide?: boolean
}>()
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader
      :title="title"
      :description="description"
      :icon="icon"
      :icon-gradient="iconGradient"
      :back-link="backLink"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <slot name="actions" />
      </template>
    </PageHeader>

    <!-- Loading -->
    <div v-if="isLoading" class="flex-1 flex items-center justify-center">
      <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
    </div>

    <!-- Not found -->
    <ErrorState
      v-else-if="isNotFound"
      :title="notFoundTitle || 'Not found'"
      :description="notFoundDescription || 'The resource you are looking for does not exist.'"
      class="flex-1"
    />

    <!-- Content -->
    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <!-- Wide canvas: page owns its own bento grid in the default slot.
             The sidebar slot (if any) stacks above the content. -->
        <div v-if="wide" class="max-w-[1600px] mx-auto space-y-6">
          <div v-if="$slots.sidebar" class="space-y-6">
            <slot name="sidebar" />
          </div>
          <slot />
        </div>

        <!-- Legacy 2/3 + 1/3 layout (default for all other detail pages). -->
        <div v-else class="max-w-6xl mx-auto grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div class="lg:col-span-2 space-y-6">
            <slot />
          </div>
          <div class="space-y-6">
            <slot name="sidebar" />
          </div>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
