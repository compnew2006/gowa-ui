<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useConnectionStore, type TestResult } from '@/stores/connection'
import { toast } from 'vue-sonner'
import {
  Globe,
  User,
  Lock,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  WifiOff,
  Power,
  RefreshCw
} from 'lucide-vue-next'

const connectionStore = useConnectionStore()

const serverUrl = ref(connectionStore.baseUrl || 'http://localhost:3000')
const username = ref(connectionStore.username || '')
const password = ref(connectionStore.password || '')
const isTesting = ref(false)

watch(
  () => [connectionStore.baseUrl, connectionStore.username, connectionStore.password],
  ([newUrl, newUser, newPass]) => {
    if (newUrl) serverUrl.value = newUrl
    username.value = newUser || ''
    password.value = newPass || ''
  }
)

onMounted(async () => {
  if (connectionStore.status === 'booting') {
    await connectionStore.boot()
  }
  if (connectionStore.baseUrl) {
    serverUrl.value = connectionStore.baseUrl
  }
  if (connectionStore.username) {
    username.value = connectionStore.username
  }
  if (connectionStore.password) {
    password.value = connectionStore.password
  }
})

async function handleConnect() {
  if (!serverUrl.value.trim()) {
    toast.error('Please enter a server URL')
    return
  }
  isTesting.value = true
  try {
    const result: TestResult = await connectionStore.connect(
      serverUrl.value,
      username.value,
      password.value
    )
    if (result === 'ok') {
      toast.success('Connected to GOWA server successfully!')
    } else if (result === 'unauthorized') {
      toast.error('Unauthorized: Invalid username or password')
    } else if (result === 'not-gowa') {
      toast.error('Server responded but does not appear to be a GOWA instance')
    } else {
      toast.error('Server unreachable. Please check URL and CORS/network settings')
    }
  } catch (e: any) {
    toast.error(e?.message || 'Failed to test connection')
  } finally {
    isTesting.value = false
  }
}

function handleDisconnect() {
  connectionStore.disconnect()
  toast.info('Disconnected from server')
}
</script>

<template>
  <Card class="border-border/60 shadow-sm">
    <CardHeader class="pb-4">
      <div class="flex items-center justify-between">
        <div>
          <CardTitle class="text-lg font-semibold flex items-center gap-2">
            <Globe class="h-5 w-5 text-emerald-500" />
            Connection
          </CardTitle>
          <CardDescription>
            Where this dashboard sends its requests
          </CardDescription>
        </div>
        <div>
          <Badge
            v-if="connectionStore.status === 'connected'"
            variant="outline"
            class="border-green-600 text-green-600 bg-green-500/10 px-2.5 py-1"
          >
            <CheckCircle2 class="h-3.5 w-3.5 mr-1" />
            Connected
          </Badge>
          <Badge
            v-else-if="connectionStore.status === 'booting'"
            variant="outline"
            class="border-blue-600 text-blue-600 bg-blue-500/10 px-2.5 py-1"
          >
            <Loader2 class="h-3.5 w-3.5 mr-1 animate-spin" />
            Connecting...
          </Badge>
          <Badge
            v-else-if="connectionStore.status === 'unauthorized'"
            variant="outline"
            class="border-destructive text-destructive bg-destructive/10 px-2.5 py-1"
          >
            <AlertTriangle class="h-3.5 w-3.5 mr-1" />
            Unauthorized
          </Badge>
          <Badge
            v-else-if="connectionStore.status === 'unreachable'"
            variant="outline"
            class="border-amber-600 text-amber-600 bg-amber-500/10 px-2.5 py-1"
          >
            <WifiOff class="h-3.5 w-3.5 mr-1" />
            Unreachable
          </Badge>
          <Badge
            v-else
            variant="outline"
            class="border-muted text-muted-foreground bg-muted/20 px-2.5 py-1"
          >
            Unconfigured
          </Badge>
        </div>
      </div>
    </CardHeader>
    <CardContent class="space-y-4">
      <div class="space-y-1.5">
        <Label for="server-url" class="text-xs font-medium text-foreground">
          Server URL
        </Label>
        <div class="relative">
          <Globe class="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            id="server-url"
            v-model="serverUrl"
            placeholder="http://localhost:3000"
            class="pl-9 font-mono text-xs"
            :disabled="isTesting"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="space-y-1.5">
          <Label for="server-username" class="text-xs font-medium text-foreground">
            Username
          </Label>
          <div class="relative">
            <User class="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="server-username"
              v-model="username"
              placeholder="Username (optional)"
              class="pl-9 text-xs"
              :disabled="isTesting"
            />
          </div>
        </div>

        <div class="space-y-1.5">
          <Label for="server-password" class="text-xs font-medium text-foreground">
            Password
          </Label>
          <div class="relative">
            <Lock class="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="server-password"
              v-model="password"
              type="password"
              placeholder="Password (optional)"
              class="pl-9 text-xs"
              :disabled="isTesting"
            />
          </div>
        </div>
      </div>

      <div class="pt-2 flex items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <Button
            size="sm"
            @click="handleConnect"
            :disabled="isTesting"
            class="bg-emerald-600 hover:bg-emerald-700 text-white"
          >
            <Loader2 v-if="isTesting" class="h-4 w-4 mr-2 animate-spin" />
            <RefreshCw v-else class="h-4 w-4 mr-2" />
            {{ isTesting ? 'Connecting...' : 'Connect' }}
          </Button>
          <Button
            v-if="connectionStore.status === 'connected' || connectionStore.baseUrl"
            variant="outline"
            size="sm"
            @click="handleDisconnect"
            :disabled="isTesting"
            class="text-destructive border-destructive/30 hover:bg-destructive/10"
          >
            <Power class="h-4 w-4 mr-1.5" />
            Disconnect
          </Button>
        </div>

        <span v-if="connectionStore.baseUrl" class="text-xs text-muted-foreground truncate max-w-[200px] font-mono">
          {{ connectionStore.baseUrl }}
        </span>
      </div>
    </CardContent>
  </Card>
</template>
