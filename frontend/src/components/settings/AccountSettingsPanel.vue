<script setup lang="ts">
// Shared scaffold for the per-account settings panels (call auto-reject, daily
// chat reset). Renders the Card + header, the enable toggle row, a separator,
// the panel-specific body (default slot), and the save-button footer. The
// owning panel keeps its own load/save logic and passes state in via props.
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Loader2 } from 'lucide-vue-next'

defineProps<{
  title: string
  description: string
  enableLabel: string
  enableDesc: string
  enabled: boolean
  canWrite: boolean
  isSubmitting: boolean
}>()

const emit = defineEmits<{
  (e: 'update:enabled', value: boolean): void
  (e: 'save'): void
}>()
</script>

<template>
  <div class="space-y-4">
    <Card>
      <CardHeader class="pb-3">
        <CardTitle class="text-sm font-medium">{{ title }}</CardTitle>
        <CardDescription class="text-xs">{{ description }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <Label class="text-xs">{{ enableLabel }}</Label>
            <p class="text-[11px] text-muted-foreground">{{ enableDesc }}</p>
          </div>
          <Switch
            :checked="enabled"
            @update:checked="emit('update:enabled', $event)"
            :disabled="!canWrite"
          />
        </div>
        <Separator />
        <slot />
        <div v-if="canWrite" class="flex justify-end">
          <Button variant="outline" size="sm" @click="emit('save')" :disabled="isSubmitting">
            <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t('settings.save') }}
          </Button>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
