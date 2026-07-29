<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import type { Component } from 'vue'

/**
 * A single sidebar navigation entry: the top-level RouterLink plus its
 * submenu children (rendered only when the item is active and the sidebar
 * is expanded). Shared by the main nav list and the bottom-pinned section.
 */
interface NavChild {
  name: string
  path: string
  icon: Component
}
interface NavItemData {
  name: string
  path: string
  icon: Component
  active?: boolean
  children?: NavChild[]
}

defineProps<{
  item: NavItemData
  isCollapsed: boolean
}>()

const emit = defineEmits<{
  (e: 'navigate'): void
}>()

const route = useRoute()
</script>

<template>
  <RouterLink
    :to="item.path"
    :class="[
      'nav-active-indicator btn-press flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-[13px] font-medium transition-all duration-200',
      item.active
        ? 'bg-white/[0.08] text-white light:bg-gray-100 light:text-gray-900'
        : 'text-white/50 hover:text-white hover:bg-white/[0.06] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-50',
      isCollapsed && 'md:justify-center md:px-2'
    ]"
    :data-active="item.active"
    role="menuitem"
    :aria-current="item.active ? 'page' : undefined"
    @click="emit('navigate')"
  >
    <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
    <span :class="isCollapsed && 'md:sr-only'">{{ $t(item.name) }}</span>
  </RouterLink>

  <!-- Submenu items -->
  <template v-if="item.children && item.active && !isCollapsed">
    <RouterLink
      v-for="child in item.children"
      :key="child.path"
      :to="child.path"
      :class="[
        'flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-[13px] font-medium transition-all duration-200 ml-4',
        route.path === child.path
          ? 'bg-white/[0.06] text-white light:bg-gray-100 light:text-gray-900'
          : 'text-white/40 hover:text-white/70 hover:bg-white/[0.04] light:text-gray-400 light:hover:text-gray-700 light:hover:bg-gray-50'
      ]"
      role="menuitem"
      :aria-current="route.path === child.path ? 'page' : undefined"
      @click="emit('navigate')"
    >
      <component :is="child.icon" class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span>{{ $t(child.name) }}</span>
    </RouterLink>
  </template>
</template>
