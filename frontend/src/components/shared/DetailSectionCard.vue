<script setup lang="ts">
// Shared "details" section card for the entity detail views (keyword rule,
// team, ...). Renders a Card with a header showing the section title and an
// optional status badge, plus a space-y-4 content area (default slot) for the
// entity's own form fields. Owning views keep their form state and logic and
// only pass the title/badge in.
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

defineProps<{
  title: string
  badgeLabel?: string
  badgeVariant?: 'default' | 'secondary' | 'destructive' | 'outline'
}>()
</script>

<template>
  <Card>
    <CardHeader class="pb-3">
      <div class="flex items-center justify-between">
        <CardTitle class="text-sm font-medium">{{ title }}</CardTitle>
        <Badge v-if="badgeLabel" :variant="badgeVariant">{{ badgeLabel }}</Badge>
      </div>
    </CardHeader>
    <CardContent class="space-y-4">
      <slot />
    </CardContent>
  </Card>
</template>
