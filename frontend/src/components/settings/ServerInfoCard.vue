<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useConnectionStore, fetchAppInfo, type AppInfo } from '@/stores/connection'
import { formatBytes } from '@/lib/utils'
import { Server, Loader2, RefreshCw, AlertCircle, FileText, Image as ImageIcon, Video } from 'lucide-vue-next'

const connectionStore = useConnectionStore()

const appInfo = ref<AppInfo | null>(null)
const isLoading = ref(false)
const errorMsg = ref<string | null>(null)

async function loadInfo() {
  if (connectionStore.status !== 'connected' || !connectionStore.baseUrl) {
    appInfo.value = null
    errorMsg.value = null
    return
  }
  isLoading.value = true
  errorMsg.value = null
  try {
    appInfo.value = await fetchAppInfo(
      connectionStore.baseUrl,
      connectionStore.username || undefined,
      connectionStore.password || undefined
    )
  } catch (e: any) {
    appInfo.value = null
    errorMsg.value = 'This server does not expose /app/info yet (needs the cross-origin enablers update).'
  } finally {
    isLoading.value = false
  }
}

watch(
  () => [connectionStore.status, connectionStore.baseUrl],
  () => {
    loadInfo()
  }
)

onMounted(() => {
  loadInfo()
})
</script>

<template>
  <Card class="border-border/60 shadow-sm">
    <CardHeader class="pb-4">
      <div class="flex items-center justify-between">
        <div>
          <CardTitle class="text-lg font-semibold flex items-center gap-2">
            <Server class="h-5 w-5 text-blue-500" />
            Server
          </CardTitle>
          <CardDescription>
            Reported by GET /app/info
          </CardDescription>
        </div>
        <Button
          v-if="connectionStore.status === 'connected'"
          variant="ghost"
          size="icon"
          class="h-8 w-8 text-muted-foreground hover:text-foreground"
          :disabled="isLoading"
          @click="loadInfo"
        >
          <RefreshCw class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
        </Button>
      </div>
    </CardHeader>
    <CardContent>
      <div v-if="connectionStore.status !== 'connected'" class="text-xs text-muted-foreground py-4 text-center">
        Connect to a server to view information
      </div>

      <div v-else-if="isLoading" class="flex items-center justify-center py-6 text-muted-foreground text-xs gap-2">
        <Loader2 class="h-4 w-4 animate-spin" />
        Loading server info...
      </div>

      <div v-else-if="errorMsg" class="rounded-md bg-amber-500/10 border border-amber-500/20 p-3 flex items-start gap-2.5 text-xs text-amber-500">
        <AlertCircle class="h-4 w-4 flex-shrink-0 mt-0.5" />
        <span>{{ errorMsg }}</span>
      </div>

      <div v-else-if="appInfo" class="space-y-3">
        <div class="grid grid-cols-2 gap-3 text-xs">
          <div class="bg-muted/40 rounded-lg p-2.5 space-y-1">
            <span class="text-muted-foreground text-[11px]">Version</span>
            <p class="font-mono font-medium text-foreground">{{ appInfo.version || 'v1.0.0' }}</p>
          </div>
          <div class="bg-muted/40 rounded-lg p-2.5 space-y-1">
            <span class="text-muted-foreground text-[11px]">Device OS</span>
            <p class="font-mono font-medium text-foreground">{{ appInfo.os || 'GOWA' }}</p>
          </div>
        </div>

        <div class="space-y-2 pt-1">
          <div class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Limits</div>
          <div class="grid grid-cols-3 gap-2 text-xs">
            <div class="flex flex-col items-center justify-center bg-muted/30 rounded-md p-2 text-center">
              <ImageIcon class="h-3.5 w-3.5 text-emerald-500 mb-1" />
              <span class="text-[10px] text-muted-foreground">Image</span>
              <span class="font-mono text-[11px] font-medium mt-0.5">
                {{ formatBytes(appInfo.max_image_size) }}
              </span>
            </div>
            <div class="flex flex-col items-center justify-center bg-muted/30 rounded-md p-2 text-center">
              <FileText class="h-3.5 w-3.5 text-blue-500 mb-1" />
              <span class="text-[10px] text-muted-foreground">File</span>
              <span class="font-mono text-[11px] font-medium mt-0.5">
                {{ formatBytes(appInfo.max_file_size) }}
              </span>
            </div>
            <div class="flex flex-col items-center justify-center bg-muted/30 rounded-md p-2 text-center">
              <Video class="h-3.5 w-3.5 text-purple-500 mb-1" />
              <span class="text-[10px] text-muted-foreground">Video</span>
              <span class="font-mono text-[11px] font-medium mt-0.5">
                {{ formatBytes(appInfo.max_video_size) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
