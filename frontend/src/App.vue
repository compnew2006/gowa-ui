<script setup lang="ts">
import { computed } from 'vue'
import { RouterView } from 'vue-router'
import { Toaster } from 'vue-sonner'
import { ConfigProvider } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useColorMode } from '@/composables/useColorMode'
import { getLocaleDirection } from '@/i18n'

// Initialize color mode early
const { colorMode } = useColorMode()

// reka-ui primitives (Tabs, Select, DropdownMenu, ...) take their direction from
// ConfigProvider, not from <html dir>. Feed it the locale direction so they
// mirror in Arabic and follow runtime locale switches.
const { locale } = useI18n()
const direction = computed(() => getLocaleDirection(locale.value))
</script>

<template>
  <ConfigProvider :dir="direction">
    <TooltipProvider>
      <div class="min-h-screen bg-background font-sans antialiased">
        <RouterView />
        <Toaster position="top-right" richColors :theme="colorMode" offset="72px" />
      </div>
    </TooltipProvider>
  </ConfigProvider>
</template>
