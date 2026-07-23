<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { PageHeader, ErrorState, DeleteConfirmDialog } from '@/components/shared'
import { useGowaServersStore } from '@/stores/gowaServers'
import { useAuthStore } from '@/stores/auth'
import { gowaServersService, type GowaDevice } from '@/services/api'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { Server, Plus, QrCode, Link2, Trash2, LogOut, RefreshCw, Webhook, Loader2, CheckCircle2, AlertCircle, Copy, Smartphone } from 'lucide-vue-next'

const route = useRoute()
const { t } = useI18n()
const store = useGowaServersStore()
const authStore = useAuthStore()

const serverId = computed(() => route.params.id as string)
const devices = ref<GowaDevice[]>([])
const loading = ref(true)
const fetchError = ref(false)

const canWriteDevices = computed(() => authStore.hasPermission('devices', 'write'))
const canDeleteDevices = computed(() => authStore.hasPermission('devices', 'delete'))

const breadcrumbs = computed(() => [
  { label: t('nav.settings'), href: '/settings' },
  { label: t('gowaServers.title', 'GOWA Servers'), href: '/settings/gowa-servers' },
  { label: store.currentServer?.name || '' },
])

const createOpen = ref(false)
const newDeviceName = ref('')
const creating = ref(false)

const connectOpen = ref(false)
const connectDevice = ref<GowaDevice | null>(null)
const qrLink = ref('')
const qrDuration = ref(30)
const qrLoading = ref(false)
const qrTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const statusPoll = ref<ReturnType<typeof setInterval> | null>(null)
const pairPhone = ref('')
const pairCode = ref('')
const pairLoading = ref(false)
const statusLoading = ref(false)

function stateClass(state: string): string {
  switch (state) {
    case 'logged_in': return 'border-emerald-600 text-emerald-600 bg-emerald-500/10'
    case 'connected': return 'border-sky-600 text-sky-600 bg-sky-500/10'
    case 'connecting': return 'border-amber-600 text-amber-600 bg-amber-500/10'
    default: return 'text-muted-foreground'
  }
}
function stateLabel(state: string): string {
  const map: Record<string, string> = {
    logged_in: t('gowaServers.connected', 'Logged in'),
    connected: t('gowaServers.connected', 'Connected'),
    connecting: t('gowaServers.connecting', 'Connecting'),
    disconnected: t('gowaServers.disconnected', 'Disconnected'),
  }
  return map[state] || state || t('gowaServers.disconnected', 'Disconnected')
}

async function onPairingSuccess() {
  clearTimers()
  toast.success(t('gowaServers.connected', 'Connected'))
  connectOpen.value = false
  await refreshDevices()
}

const webhookOpen = ref(false)
const webhookDevice = ref<GowaDevice | null>(null)
const webhookForm = ref({ webhook_url: '', webhook_events: 'message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited', webhook_insecure_skip_verify: false })
const webhookSaving = ref(false)

const deleteOpen = ref(false)
const deviceToDelete = ref<GowaDevice | null>(null)
const isDeleting = ref(false)

onMounted(load)
onBeforeUnmount(clearTimers)

async function load() {
  loading.value = true
  fetchError.value = false
  try {
    await Promise.all([store.fetchServer(serverId.value), store.fetchDevices(serverId.value)])
    devices.value = store.devices
  } catch {
    fetchError.value = true
    toast.error(t('common.failedLoad', { resource: 'server' }))
  } finally {
    loading.value = false
  }
}

async function refreshDevices() {
  try {
    await store.fetchDevices(serverId.value)
    devices.value = store.devices
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedLoad', { resource: 'devices' })))
  }
}

function openCreate() {
  newDeviceName.value = ''
  createOpen.value = true
}

async function submitCreate() {
  creating.value = true
  try {
    await gowaServersService.createDevice(serverId.value, { device_name: newDeviceName.value || 'whatomate' })
    toast.success(t('gowaServers.deviceCreated', 'Device created'))
    createOpen.value = false
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, t('gowaServers.deviceCreated', 'Failed to create device')))
  } finally {
    creating.value = false
  }
}

function openConnect(d: GowaDevice) {
  connectDevice.value = d
  connectOpen.value = true
  qrLink.value = ''
  pairCode.value = ''
  pairPhone.value = ''
  fetchQr()
  if (statusPoll.value) clearInterval(statusPoll.value)
  statusPoll.value = setInterval(fetchQr, 3000)
}

async function fetchQr() {
  if (!connectDevice.value) return
  qrLoading.value = true
  try {
    const resp = await gowaServersService.deviceQR(serverId.value, connectDevice.value.id)
    const data = (resp.data as any).data || resp.data
    if (data.already_connected) {
      qrLink.value = ''
      await onPairingSuccess()
      return
    }
    qrLink.value = data.qr_link || ''
    qrDuration.value = data.qr_duration || 30
    if (!statusPoll.value && qrTimer.value === null) {
      qrTimer.value = setTimeout(fetchQr, (qrDuration.value + 2) * 1000)
    }
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.gowaQrFailed', 'Failed to get QR code')))
  } finally {
    qrLoading.value = false
  }
}

async function fetchPair() {
  if (!connectDevice.value || !pairPhone.value.trim()) return
  pairLoading.value = true
  pairCode.value = ''
  try {
    const resp = await gowaServersService.devicePairCode(serverId.value, connectDevice.value.id, pairPhone.value.trim())
    const data = (resp.data as any).data || resp.data
    pairCode.value = data.pair_code || ''
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.gowaPairFailed', 'Failed to get pair code')))
  } finally {
    pairLoading.value = false
  }
}

async function closeConnect() {
  connectOpen.value = false
  clearTimers()
  await refreshDevices()
}

function clearTimers() {
  if (qrTimer.value) { clearTimeout(qrTimer.value); qrTimer.value = null }
  if (statusPoll.value) { clearInterval(statusPoll.value); statusPoll.value = null }
}

async function copyText(txt: string) {
  try { await navigator.clipboard.writeText(txt) } catch { /* ignore */ }
}

async function logout(d: GowaDevice) {
  try {
    await gowaServersService.deviceLogout(serverId.value, d.id)
    toast.success(t('gowaServers.logout', 'Logout'))
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  }
}

async function reconnect(d: GowaDevice) {
  statusLoading.value = true
  try {
    await gowaServersService.deviceReconnect(serverId.value, d.id)
    toast.success(t('gowaServers.reconnect', 'Reconnect'))
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  } finally {
    statusLoading.value = false
  }
}

async function openWebhook(d: GowaDevice) {
  webhookDevice.value = d
  webhookForm.value = { webhook_url: d.webhook_url || '', webhook_events: d.webhook_events || 'message,message.ack,chat_presence,connection,message.reaction,message.revoked,message.edited', webhook_insecure_skip_verify: false }
  webhookOpen.value = true
  try {
    const resp = await gowaServersService.getDeviceWebhook(serverId.value, d.id)
    const data = (resp.data as any).data || resp.data
    if (data.webhook) {
      webhookForm.value.webhook_url = data.webhook.webhook_url || webhookForm.value.webhook_url
      webhookForm.value.webhook_events = data.webhook.webhook_events || webhookForm.value.webhook_events
      webhookForm.value.webhook_insecure_skip_verify = !!data.webhook.webhook_insecure_skip_verify
    }
  } catch { /* keep defaults */ }
}

async function saveWebhook() {
  if (!webhookDevice.value) return
  webhookSaving.value = true
  try {
    await gowaServersService.setDeviceWebhook(serverId.value, webhookDevice.value.id, webhookForm.value)
    toast.success(t('gowaServers.saveWebhook', 'Saved'))
    webhookOpen.value = false
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  } finally {
    webhookSaving.value = false
  }
}

function openDelete(d: GowaDevice) {
  deviceToDelete.value = d
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!deviceToDelete.value) return
  isDeleting.value = true
  try {
    await gowaServersService.deleteDevice(serverId.value, deviceToDelete.value.id)
    toast.success(t('gowaServers.deleteDevice', 'Deleted'))
    deleteOpen.value = false
    deviceToDelete.value = null
    await refreshDevices()
  } catch (e) {
    toast.error(getErrorMessage(e, 'Failed'))
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="store.currentServer?.name || t('gowaServers.title', 'GOWA Servers')"
      :icon="Server"
      icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20"
      back-link="/settings/gowa-servers"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <Button v-if="canWriteDevices" size="sm" class="bg-blue-600 hover:bg-blue-700 text-white" @click="openCreate">
          <Plus class="h-4 w-4 mr-1.5" />
          {{ $t('gowaServers.createDevice', 'Create Device') }}
        </Button>
      </template>
    </PageHeader>

    <ErrorState
      v-if="fetchError && !loading"
      :title="$t('common.loadErrorTitle', 'Failed to load')"
      class="flex-1"
    >
      <template #action>
        <Button size="sm" @click="load">{{ $t('common.retry', 'Retry') }}</Button>
      </template>
    </ErrorState>

    <ScrollArea v-else class="flex-1">
      <div class="p-6 space-y-6">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div>
                <CardTitle>{{ $t('gowaServers.devices', 'Devices') }}</CardTitle>
                <CardDescription>{{ store.currentServer?.base_url }}</CardDescription>
              </div>
              <Button variant="ghost" size="sm" @click="refreshDevices">
                <RefreshCw class="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="!loading && devices.length === 0" class="text-center py-12 text-muted-foreground">
              <Smartphone class="h-10 w-10 mx-auto mb-3 opacity-50" />
              <p class="text-sm">{{ $t('gowaServers.noDevices', 'No devices on this server') }}</p>
              <p class="text-xs mt-1">{{ $t('gowaServers.noDevicesDesc', 'Create a device to connect a WhatsApp number.') }}</p>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <Card v-for="d in devices" :key="d.id" class="border-border/60">
                <CardContent class="pt-4 space-y-3">
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0">
                      <p class="font-medium text-sm truncate">{{ d.display_name || d.id }}</p>
                      <code class="text-[10px] bg-muted px-1.5 py-0.5 rounded font-mono block mt-1 truncate">{{ d.jid || d.phone_number || d.id }}</code>
                    </div>
                    <Badge variant="outline" :class="stateClass(d.state) + ' flex-shrink-0'">
                      <CheckCircle2 v-if="d.state === 'logged_in' || d.is_connected" class="h-3 w-3 mr-1" />
                      <AlertCircle v-else class="h-3 w-3 mr-1" />
                      {{ stateLabel(d.state) }}
                    </Badge>
                  </div>

                  <div class="flex flex-wrap gap-1.5">
                    <Button v-if="canWriteDevices" size="sm" variant="outline" @click="openConnect(d)">
                      <QrCode class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.connect', 'Connect') }}
                    </Button>
                    <Button v-if="canWriteDevices" size="sm" variant="ghost" @click="reconnect(d)" :disabled="statusLoading">
                      <RefreshCw class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.reconnect', 'Reconnect') }}
                    </Button>
                    <Button v-if="canWriteDevices" size="sm" variant="ghost" @click="logout(d)">
                      <LogOut class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.logout', 'Logout') }}
                    </Button>
                    <Button v-if="canWriteDevices" size="sm" variant="ghost" @click="openWebhook(d)">
                      <Webhook class="h-3.5 w-3.5 mr-1" /> {{ $t('gowaServers.webhook', 'Webhook') }}
                    </Button>
                    <Button v-if="canDeleteDevices" size="sm" variant="ghost" class="text-destructive" @click="openDelete(d)">
                      <Trash2 class="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          </CardContent>
        </Card>
      </div>
    </ScrollArea>

    <Dialog :open="createOpen" @update:open="(v) => createOpen = v">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('gowaServers.createDevice', 'Create Device') }}</DialogTitle>
          <DialogDescription>{{ $t('gowaServers.noDevicesDesc', 'Create a device to connect a WhatsApp number.') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-2 py-2">
          <Label>{{ $t('gowaServers.deviceName', 'Device Name') }}</Label>
          <Input v-model="newDeviceName" placeholder="sales-phone" />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createOpen = false">{{ $t('common.cancel', 'Cancel') }}</Button>
          <Button :disabled="creating" @click="submitCreate">{{ $t('gowaServers.createDevice', 'Create Device') }}</Button>
        </div>
      </DialogContent>
    </Dialog>

    <Dialog :open="connectOpen" @update:open="(v) => !v && closeConnect()">
      <DialogContent class="max-w-lg" @escape-key-down="closeConnect" @pointer-down-outside="closeConnect">
        <DialogHeader>
          <DialogTitle>{{ $t('accounts.connectDevice', 'Connect Device') }}</DialogTitle>
          <DialogDescription>{{ $t('accounts.gowaConnectDesc', 'Scan the QR code or use a pair code to link your WhatsApp account.') }}</DialogDescription>
        </DialogHeader>
        <Tabs default-value="qr">
          <TabsList class="grid w-full grid-cols-2">
            <TabsTrigger value="qr"><QrCode class="h-4 w-4 mr-1.5" /> {{ $t('gowaServers.qrCode', 'QR Code') }}</TabsTrigger>
            <TabsTrigger value="pair"><Link2 class="h-4 w-4 mr-1.5" /> {{ $t('gowaServers.pairCode', 'Pair Code') }}</TabsTrigger>
          </TabsList>
          <TabsContent value="qr" class="flex flex-col items-center gap-3 py-4">
            <div class="relative w-64 h-64 bg-white rounded-lg flex items-center justify-center border border-border shadow-inner">
              <Loader2 v-if="qrLoading && !qrLink" class="h-8 w-8 animate-spin text-muted-foreground" />
              <img v-else-if="qrLink" :src="qrLink" alt="QR Code" class="w-full h-full object-contain p-2" />
              <QrCode v-else class="h-16 w-16 text-muted-foreground" />
            </div>
            <p class="text-xs text-muted-foreground text-center">{{ $t('gowaServers.qrInstructions') }}</p>
            <Button variant="outline" size="sm" :disabled="qrLoading" @click="fetchQr">
              <RefreshCw class="h-4 w-4 mr-1" :class="{ 'animate-spin': qrLoading }" />
              {{ $t('gowaServers.refreshQr', 'Refresh QR') }}
            </Button>
          </TabsContent>
          <TabsContent value="pair" class="space-y-4 py-4">
            <div class="space-y-2">
              <Label class="text-xs">{{ $t('gowaServers.phoneNumber', 'Phone Number') }}</Label>
              <div class="flex gap-2">
                <Input v-model="pairPhone" placeholder="16505551234" class="flex-1" />
                <Button size="sm" :disabled="pairLoading || !pairPhone.trim()" @click="fetchPair">
                  <Loader2 v-if="pairLoading" class="h-4 w-4 animate-spin mr-1" />
                  {{ $t('gowaServers.getCode', 'Get Code') }}
                </Button>
              </div>
              <p class="text-xs text-muted-foreground">{{ $t('gowaServers.pairCodeInstructions') }}</p>
            </div>
            <div v-if="pairCode" class="flex flex-col items-center gap-2 p-4 bg-muted rounded-lg">
              <span class="text-xs text-muted-foreground">{{ $t('gowaServers.yourPairCode', 'Your Pair Code') }}</span>
              <span class="text-3xl font-bold font-mono tracking-[0.3em]">{{ pairCode }}</span>
              <Button variant="ghost" size="sm" @click="copyText(pairCode)">
                <Copy class="h-3 w-3 mr-1" /> {{ $t('common.copy', 'Copy') }}
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>

    <Dialog :open="webhookOpen" @update:open="(v) => webhookOpen = v">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('gowaServers.webhook', 'Webhook') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.webhookUrlLabel', 'Webhook URL') }}</Label>
            <Input v-model="webhookForm.webhook_url" placeholder="https://whatomate.example.com/api/gowa/webhook" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.webhookEventsLabel', 'Webhook Events') }}</Label>
            <Input v-model="webhookForm.webhook_events" class="font-mono text-xs" />
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="webhookOpen = false">{{ $t('common.cancel', 'Cancel') }}</Button>
          <Button :disabled="webhookSaving" @click="saveWebhook">{{ $t('gowaServers.saveWebhook', 'Save Webhook') }}</Button>
        </div>
      </DialogContent>
    </Dialog>

    <DeleteConfirmDialog
      v-model:open="deleteOpen"
      :title="$t('gowaServers.deleteDevice', 'Delete Device')"
      :item-name="deviceToDelete?.display_name || deviceToDelete?.id"
      :is-submitting="isDeleting"
      @confirm="confirmDelete"
    />
  </div>
</template>
