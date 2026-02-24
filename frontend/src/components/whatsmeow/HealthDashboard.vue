<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useInstancesStore } from '@/stores/instances'
import { useOrganizationsStore } from '@/stores/organizations'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Activity, AlertTriangle, Clock3, Inbox, Loader2, Send } from 'lucide-vue-next'

const instancesStore = useInstancesStore()
const organizationsStore = useOrganizationsStore()
const refreshing = ref(false)

const instances = computed(() => instancesStore.instances)

function formatUptime(totalSeconds?: number) {
  const seconds = totalSeconds || 0
  if (seconds <= 0) {
    return '0m'
  }
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

async function refreshHealth(options: { includeInstances?: boolean } = {}) {
  refreshing.value = true
  try {
    if (options.includeInstances) {
      await instancesStore.fetchInstances()
    }
    await instancesStore.fetchAllHealth()
  } finally {
    refreshing.value = false
  }
}

onMounted(async () => {
  await refreshHealth({ includeInstances: true })
  instancesStore.startHealthPolling(30000, { refreshInstances: true })
})

onUnmounted(() => {
  instancesStore.stopHealthPolling()
})

watch(
  () => organizationsStore.selectedOrgId,
  async (nextOrgID, previousOrgID) => {
    if (nextOrgID === previousOrgID) {
      return
    }
    await refreshHealth({ includeInstances: true })
  }
)
</script>

<template>
  <div class="space-y-4">
    <div class="flex justify-end">
      <Button variant="outline" class="border-white/10 hover:bg-white/5 text-white/80 light:text-gray-700" @click="refreshHealth({ includeInstances: true })">
        <Loader2 v-if="refreshing" class="h-4 w-4 mr-2 animate-spin" />
        Refresh
      </Button>
    </div>

    <div v-if="instances.length === 0" class="text-sm text-white/50 light:text-gray-500">
      No instances found.
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <Card
        v-for="instance in instances"
        :key="instance.id"
        class="bg-white/[0.04] border-white/[0.08] light:bg-white light:border-gray-200"
      >
        <CardHeader>
          <CardTitle class="text-sm text-white light:text-gray-900 flex items-center justify-between">
            <span>{{ instance.name }}</span>
            <span class="text-xs font-normal text-white/50 light:text-gray-500">{{ instance.status }}</span>
          </CardTitle>
        </CardHeader>
        <CardContent class="space-y-3 text-sm">
          <div class="flex items-center justify-between text-white/70 light:text-gray-700">
            <div class="flex items-center gap-2">
              <Clock3 class="h-4 w-4 opacity-70" />
              <span>Uptime</span>
            </div>
            <span class="font-medium text-white light:text-gray-900">{{ formatUptime(instance.health?.uptime_seconds) }}</span>
          </div>

          <div class="flex items-center justify-between text-white/70 light:text-gray-700">
            <div class="flex items-center gap-2">
              <Send class="h-4 w-4 opacity-70" />
              <span>Sent today</span>
            </div>
            <span class="font-medium text-white light:text-gray-900">{{ instance.health?.messages_sent_today ?? 0 }}</span>
          </div>

          <div class="flex items-center justify-between text-white/70 light:text-gray-700">
            <div class="flex items-center gap-2">
              <Inbox class="h-4 w-4 opacity-70" />
              <span>Received today</span>
            </div>
            <span class="font-medium text-white light:text-gray-900">{{ instance.health?.messages_received_today ?? 0 }}</span>
          </div>

          <div class="flex items-center justify-between text-white/70 light:text-gray-700">
            <div class="flex items-center gap-2">
              <AlertTriangle class="h-4 w-4 opacity-70" />
              <span>Failed today</span>
            </div>
            <span class="font-medium text-white light:text-gray-900">{{ instance.health?.messages_failed_today ?? 0 }}</span>
          </div>

          <div class="flex items-center justify-between text-white/70 light:text-gray-700">
            <div class="flex items-center gap-2">
              <Activity class="h-4 w-4 opacity-70" />
              <span>Error rate</span>
            </div>
            <span class="font-medium text-white light:text-gray-900">{{ (instance.health?.error_rate_percent ?? 0).toFixed(1) }}%</span>
          </div>

          <div class="flex items-center justify-between text-white/70 light:text-gray-700">
            <span>Queue depth</span>
            <span class="font-medium text-white light:text-gray-900">{{ instance.health?.queue_depth ?? 0 }}</span>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
