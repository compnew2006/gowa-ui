<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/services/api'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { useUnsavedChangesGuard } from '@/composables/useUnsavedChangesGuard'
import DetailPageLayout from '@/components/shared/DetailPageLayout.vue'
import MetadataPanel from '@/components/shared/MetadataPanel.vue'
import AuditLogPanel from '@/components/shared/AuditLogPanel.vue'
import UnsavedChangesDialog from '@/components/shared/UnsavedChangesDialog.vue'
import AccountCloseRatingPanel from '@/components/settings/AccountCloseRatingPanel.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Phone,
  Save,
  Trash2,
  Copy,
  RefreshCw,
  Loader2,
  AlertCircle,
  CheckCircle2,
  QrCode,
  Link2,
  Smartphone
} from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/ui/tabs'

interface WhatsAppAccount {
  id: string
  name: string
  gowa_base_url?: string
  gowa_device_id?: string
  gowa_username?: string
  gowa_password?: string
  has_gowa_webhook_secret?: boolean
  is_default_incoming: boolean
  is_default_outgoing: boolean
  auto_read_receipt: boolean
  status: string
  created_by_id?: string
  created_by_name?: string
  updated_by_id?: string
  updated_by_name?: string
  created_at: string
  updated_at: string
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const accountId = computed(() => route.params.id as string)
const isNew = computed(() => accountId.value === 'new')
const account = ref<WhatsAppAccount | null>(null)
const isLoading = ref(true)
const isNotFound = ref(false)
const isSaving = ref(false)
const hasChanges = ref(false)
const deleteDialogOpen = ref(false)

const { showLeaveDialog, confirmLeave, cancelLeave } = useUnsavedChangesGuard(hasChanges)

const canWrite = computed(() => authStore.hasPermission('accounts', 'write'))
const canDelete = computed(() => authStore.hasPermission('accounts', 'delete'))
const canWriteDevices = computed(() => authStore.hasPermission('devices', 'write'))

const form = ref({
  name: '',
  gowa_base_url: '',
  gowa_device_id: '',
  gowa_username: '',
  gowa_password: '',
  gowa_webhook_secret: '',
  is_default_incoming: false,
  is_default_outgoing: false,
  auto_read_receipt: false,
})

const breadcrumbs = computed(() => [
  { label: t('nav.settings'), href: '/settings' },
  { label: t('settings.accounts', 'Accounts'), href: '/settings/accounts' },
  { label: isNew.value ? t('accounts.newAccount', 'New Account') : (account.value?.name || '') },
])

// Track form changes
watch(form, () => { hasChanges.value = true }, { deep: true })

async function loadAccount() {
  isLoading.value = true
  isNotFound.value = false
  try {
    const response = await api.get(`/accounts/${accountId.value}`)
    const data = response.data.data || response.data
    account.value = data
    syncForm()
    nextTick(() => { hasChanges.value = false })
  } catch {
    isNotFound.value = true
  } finally {
    isLoading.value = false
  }
}

function syncForm() {
  if (!account.value) return
  form.value = {
    name: account.value.name,
    gowa_base_url: account.value.gowa_base_url || '',
    gowa_device_id: account.value.gowa_device_id || '',
    gowa_username: account.value.gowa_username || '',
    gowa_password: account.value.gowa_password || '',
    gowa_webhook_secret: '',
    is_default_incoming: account.value.is_default_incoming,
    is_default_outgoing: account.value.is_default_outgoing,
    auto_read_receipt: account.value.auto_read_receipt,
  }
}

async function save() {
  if (!form.value.name.trim()) {
    toast.error(t('accounts.fillRequired', 'Account name is required'))
    return
  }

  if (!form.value.gowa_base_url.trim() || !form.value.gowa_device_id.trim()) {
    toast.error(t('accounts.gowaFieldsRequired', 'GOWA Base URL and Device ID are required'))
    return
  }

  isSaving.value = true
  try {
    const payload: any = {
      name: form.value.name,
      gowa_base_url: form.value.gowa_base_url,
      gowa_device_id: form.value.gowa_device_id,
      gowa_username: form.value.gowa_username,
      gowa_password: form.value.gowa_password,
      gowa_webhook_secret: form.value.gowa_webhook_secret,
      is_default_incoming: form.value.is_default_incoming,
      is_default_outgoing: form.value.is_default_outgoing,
      auto_read_receipt: form.value.auto_read_receipt,
    }

    if (!isNew.value && !payload.gowa_webhook_secret) {
      delete payload.gowa_webhook_secret
    }

    if (isNew.value) {
      const response = await api.post('/accounts', payload)
      const created = response.data.data || response.data
      hasChanges.value = false
      toast.success(t('common.createdSuccess', { resource: t('resources.Account', 'Account') }))
      router.replace(`/settings/accounts/${created.id}`)
    } else {
      await api.put(`/accounts/${account.value!.id}`, payload)
      await loadAccount()
      hasChanges.value = false
      toast.success(t('common.updatedSuccess', { resource: t('resources.Account', 'Account') }))
    }
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedSave', { resource: t('resources.account', 'account') })))
  } finally {
    isSaving.value = false
  }
}

async function deleteAccount() {
  if (!account.value) return
  try {
    await api.delete(`/accounts/${account.value.id}`)
    toast.success(t('common.deletedSuccess', { resource: t('resources.Account', 'Account') }))
    router.push('/settings/accounts')
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.account', 'account') })))
  }
  deleteDialogOpen.value = false
}

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(t('common.copiedToClipboard', 'Copied to clipboard'))
  } catch {
    toast.error(t('common.clipboardFailed', 'Failed to copy'))
  }
}

// --- GOWA Instance & Device Creation ---
const gowaInstances = ref<Array<{ name: string; base_url: string }>>([])
const selectedGowaInstance = ref('')
const creatingDevice = ref(false)

async function fetchGowaInstances() {
  try {
    const resp = await api.get('/gowa/instances')
    gowaInstances.value = resp.data.data?.instances || []
  } catch {
    gowaInstances.value = []
  }
}

async function createGowaDevice() {
  if (!selectedGowaInstance.value) return
  creatingDevice.value = true
  try {
    const resp = await api.post('/gowa/create-device', { base_url: selectedGowaInstance.value, device_name: form.value.name || 'whatomate' })
    const data = resp.data.data || resp.data
    form.value.gowa_base_url = data.base_url
    form.value.gowa_device_id = data.device_id
    form.value.gowa_webhook_secret = data.webhook_secret
    toast.success(t('accounts.deviceCreated', 'Device created on GOWA instance'))
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.deviceCreateFailed', 'Failed to create device')))
  } finally {
    creatingDevice.value = false
  }
}

// --- GOWA Device Connection ---
const gowaConnectOpen = ref(false)
const gowaQrLink = ref('')
const gowaQrDuration = ref(30)
const gowaQrLoading = ref(false)
const gowaQrTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const gowaPairPhone = ref('')
const gowaPairCode = ref('')
const gowaPairLoading = ref(false)
const gowaStatus = ref<{ is_connected: boolean; is_logged_in: boolean; jid: string } | null>(null)
const gowaStatusLoading = ref(false)
const gowaStatusTimer = ref<ReturnType<typeof setInterval> | null>(null)

async function fetchGowaQr() {
  if (!account.value) return
  gowaQrLoading.value = true
  try {
    const resp = await api.get(`/accounts/${account.value.id}/gowa/qr`)
    const data = resp.data.data || resp.data
    if (data.already_connected) {
      gowaStatus.value = { is_connected: true, is_logged_in: true, jid: data.jid || '' }
      gowaQrLink.value = ''
      return
    }
    gowaQrLink.value = data.qr_link || ''
    gowaQrDuration.value = data.qr_duration || 30
    if (gowaQrTimer.value) clearTimeout(gowaQrTimer.value)
    if (!gowaStatus.value?.is_connected) {
      gowaQrTimer.value = setTimeout(fetchGowaQr, (gowaQrDuration.value + 2) * 1000)
    }
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.gowaQrFailed', 'Failed to get QR code')))
  } finally {
    gowaQrLoading.value = false
  }
}

async function fetchGowaPairCode() {
  if (!account.value || !gowaPairPhone.value.trim()) return
  gowaPairLoading.value = true
  gowaPairCode.value = ''
  try {
    const resp = await api.post(`/accounts/${account.value.id}/gowa/pair-code`, { phone: gowaPairPhone.value.trim() })
    const data = resp.data.data || resp.data
    gowaPairCode.value = data.pair_code || ''
  } catch (e) {
    toast.error(getErrorMessage(e, t('accounts.gowaPairFailed', 'Failed to get pair code')))
  } finally {
    gowaPairLoading.value = false
  }
}

async function fetchGowaStatus() {
  if (!account.value) return
  gowaStatusLoading.value = true
  try {
    const resp = await api.get(`/accounts/${account.value.id}/gowa/status`)
    gowaStatus.value = resp.data.data || resp.data
  } catch {
    gowaStatus.value = null
  } finally {
    gowaStatusLoading.value = false
  }
}

function openGowaConnect() {
  gowaConnectOpen.value = true
  gowaQrLink.value = ''
  gowaPairCode.value = ''
  gowaPairPhone.value = ''
  fetchGowaStatus()
  fetchGowaQr()
  gowaStatusTimer.value = setInterval(async () => {
    await fetchGowaStatus()
    if (gowaStatus.value?.is_connected) {
      clearGowaTimers()
      toast.success(t('accounts.gowaConnected', 'Device connected!'))
      gowaConnectOpen.value = false
      await loadAccount()
    }
  }, 5000)
}

function closeGowaConnect() {
  gowaConnectOpen.value = false
  clearGowaTimers()
}

function clearGowaTimers() {
  if (gowaQrTimer.value) { clearTimeout(gowaQrTimer.value); gowaQrTimer.value = null }
  if (gowaStatusTimer.value) { clearInterval(gowaStatusTimer.value); gowaStatusTimer.value = null }
}

onMounted(async () => {
  if (isNew.value) {
    isLoading.value = false
    hasChanges.value = false
    fetchGowaInstances()
  } else {
    await loadAccount()
  }
})
</script>

<template>
  <div class="h-full">
    <DetailPageLayout
      :title="isNew ? $t('accounts.newAccount', 'New Account') : (account?.name || '')"
      :icon="Phone"
      icon-gradient="bg-gradient-to-br from-emerald-500 to-green-600 shadow-emerald-500/20"
      back-link="/settings/accounts"
      :breadcrumbs="breadcrumbs"
      :is-loading="isLoading"
      :is-not-found="isNotFound"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <Button v-if="!isNew && account && canWriteDevices" variant="outline" size="sm" @click="openGowaConnect">
            <QrCode class="h-4 w-4 mr-1" />
            {{ $t('accounts.connectDevice', 'Connect Device') }}
          </Button>
          <Button v-if="canWrite && (hasChanges || isNew)" size="sm" @click="save" :disabled="isSaving" class="bg-emerald-600 hover:bg-emerald-700 text-white font-medium">
            <Save class="h-4 w-4 mr-1" /> {{ isSaving ? $t('common.saving', 'Saving...') : isNew ? $t('common.create', 'Create') : $t('common.save', 'Save') }}
          </Button>
          <Button v-if="!isNew && canDelete" variant="ghost" size="icon" class="text-destructive hover:text-destructive" @click="deleteDialogOpen = true">
            <Trash2 class="h-4 w-4" />
          </Button>
        </div>
      </template>

      <!-- Account Details Card -->
      <Card>
        <CardHeader class="pb-3">
          <div class="flex items-center justify-between">
            <div>
              <CardTitle class="text-sm font-medium">{{ $t('accounts.accountDetails', 'GOWA Account Details') }}</CardTitle>
              <CardDescription class="text-xs">{{ $t('accounts.gowaAccountDesc', 'Configure GOWA server connection parameters') }}</CardDescription>
            </div>
            <Badge v-if="account" :variant="account.status === 'active' ? 'default' : 'secondary'">
              {{ account.status }}
            </Badge>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="space-y-1.5">
            <Label class="text-xs">{{ $t('accounts.accountName', 'Account Name') }} *</Label>
            <Input v-model="form.name" placeholder="e.g. Sales WhatsApp" :disabled="!canWrite" />
          </div>

          <Separator />

          <!-- New account provisioning options -->
          <div v-if="isNew" class="space-y-4">
            <div class="space-y-1.5" v-if="gowaInstances.length > 0">
              <Label class="text-xs">{{ $t('accounts.gowaInstance', 'Select GOWA Server Instance') }}</Label>
              <select
                v-model="selectedGowaInstance"
                class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                :disabled="!canWrite"
              >
                <option value="">{{ $t('accounts.selectInstance', 'Select an instance or enter manually below...') }}</option>
                <option v-for="inst in gowaInstances" :key="inst.base_url" :value="inst.base_url">{{ inst.name }} ({{ inst.base_url }})</option>
              </select>
            </div>

            <div v-if="selectedGowaInstance && !form.gowa_device_id" class="flex items-center gap-2 p-3 bg-muted/40 rounded-lg border border-border/50">
              <Button variant="outline" size="sm" :disabled="creatingDevice" @click="createGowaDevice">
                <Loader2 v-if="creatingDevice" class="h-4 w-4 animate-spin mr-1" />
                <Smartphone v-else class="h-4 w-4 mr-1 text-emerald-500" />
                {{ $t('accounts.createDevice', 'Auto-Provision Device') }}
              </Button>
              <span class="text-xs text-muted-foreground">{{ $t('accounts.createDeviceDesc', 'Automatically provisions a new device ID on the selected server') }}</span>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-1">
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaBaseUrl', 'GOWA Base URL') }} *</Label>
                <Input v-model="form.gowa_base_url" placeholder="http://127.0.0.1:3000 or http://localhost:8080" class="font-mono text-xs" :disabled="!canWrite" />
              </div>
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaDeviceId', 'GOWA Device ID') }} *</Label>
                <Input v-model="form.gowa_device_id" placeholder="e.g. device_1 or my_wa_account" class="font-mono text-xs" :disabled="!canWrite" />
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaUsername', 'Username (Optional)') }}</Label>
                <Input v-model="form.gowa_username" placeholder="Basic auth username if required" class="text-xs" :disabled="!canWrite" />
              </div>
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaPassword', 'Password (Optional)') }}</Label>
                <Input v-model="form.gowa_password" type="password" placeholder="Basic auth password if required" class="text-xs" :disabled="!canWrite" />
              </div>
            </div>
          </div>

          <!-- Existing account params -->
          <div v-else class="space-y-3">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaBaseUrl', 'GOWA Base URL') }} *</Label>
                <Input v-model="form.gowa_base_url" class="font-mono text-xs" :disabled="!canWrite" />
              </div>
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaDeviceId', 'GOWA Device ID') }} *</Label>
                <Input v-model="form.gowa_device_id" class="font-mono text-xs" :disabled="!canWrite" />
              </div>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaUsername', 'Username (Optional)') }}</Label>
                <Input v-model="form.gowa_username" placeholder="Basic auth username if required" class="text-xs" :disabled="!canWrite" />
              </div>
              <div class="space-y-1.5">
                <Label class="text-xs">{{ $t('accounts.gowaPassword', 'Password (Optional)') }}</Label>
                <Input v-model="form.gowa_password" type="password" placeholder="Basic auth password if required" class="text-xs" :disabled="!canWrite" />
              </div>
            </div>
          </div>

          <Separator />

          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <Label class="text-xs">{{ $t('accounts.defaultIncoming', 'Default for Incoming') }}</Label>
                <p class="text-[11px] text-muted-foreground">Route unassigned incoming WhatsApp messages to this account</p>
              </div>
              <Switch :checked="form.is_default_incoming" @update:checked="form.is_default_incoming = $event" :disabled="!canWrite" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <Label class="text-xs">{{ $t('accounts.defaultOutgoing', 'Default for Outgoing') }}</Label>
                <p class="text-[11px] text-muted-foreground">Use this account for default outgoing campaign dispatch</p>
              </div>
              <Switch :checked="form.is_default_outgoing" @update:checked="form.is_default_outgoing = $event" :disabled="!canWrite" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <Label class="text-xs">{{ $t('accounts.autoReadReceipt', 'Auto Read Receipt') }}</Label>
                <p class="text-[11px] text-muted-foreground">Automatically send read receipts for incoming messages</p>
              </div>
              <Switch :checked="form.auto_read_receipt" @update:checked="form.auto_read_receipt = $event" :disabled="!canWrite" />
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Per-account chat-close rating (prompt/thanks/lexicon + CSAT stats) -->
      <AccountCloseRatingPanel
        v-if="account && !isNew"
        :account-id="account.id"
        :can-write="canWrite"
      />

      <!-- Activity Log -->
      <AuditLogPanel
        v-if="account && !isNew"
        resource-type="account"
        :resource-id="account.id"
      />

      <!-- Sidebar -->
      <template #sidebar>
        <MetadataPanel
          v-if="!isNew"
          :created-at="account?.created_at"
          :updated-at="account?.updated_at"
          :created-by-name="account?.created_by_name"
          :updated-by-name="account?.updated_by_name"
        />

        <!-- GOWA Setup Guide -->
        <Card>
          <CardHeader class="pb-3">
            <CardTitle class="text-sm font-medium">{{ $t('accounts.setupGuide', 'GOWA Connection Guide') }}</CardTitle>
          </CardHeader>
          <CardContent>
            <ol class="list-decimal list-inside space-y-2.5 text-xs text-muted-foreground leading-relaxed">
              <li>Enter an <strong>Account Name</strong> for identification.</li>
              <li>Set or select the <strong>GOWA Base URL</strong> (e.g. server URL).</li>
              <li>Provide a unique <strong>Device ID</strong>.</li>
              <li>Click <strong>Save / Create</strong> to register the account.</li>
              <li>Click <strong>Connect Device</strong> to scan QR code or enter Pair Code on WhatsApp.</li>
            </ol>
          </CardContent>
        </Card>
      </template>
    </DetailPageLayout>

    <!-- Delete Confirmation -->
    <AlertDialog v-model:open="deleteDialogOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('accounts.deleteAccount', 'Delete Account') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ $t('accounts.deleteAccountConfirm', 'Are you sure? This action cannot be undone.') }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('common.cancel', 'Cancel') }}</AlertDialogCancel>
          <AlertDialogAction @click="deleteAccount">{{ $t('common.delete', 'Delete') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- GOWA Connect Device Dialog -->
    <Dialog :open="gowaConnectOpen" @update:open="(v) => !v && closeGowaConnect()">
      <DialogContent class="max-w-lg" @escape-key-down="closeGowaConnect" @pointer-down-outside="closeGowaConnect">
        <DialogHeader>
          <DialogTitle>{{ $t('accounts.connectDevice', 'Connect Device') }}</DialogTitle>
          <DialogDescription>
            {{ $t('accounts.gowaConnectDesc', 'Scan the QR code or use a pair code to link your WhatsApp account.') }}
          </DialogDescription>
        </DialogHeader>

        <!-- Connection Status -->
        <div class="flex items-center gap-2 text-sm mb-2">
          <Loader2 v-if="gowaStatusLoading" class="h-4 w-4 animate-spin text-muted-foreground" />
          <template v-else-if="gowaStatus">
            <Badge v-if="gowaStatus.is_connected" variant="outline" class="border-green-600 text-green-600">
              <CheckCircle2 class="h-3 w-3 mr-1" /> {{ $t('accounts.connected', 'Connected') }}
            </Badge>
            <Badge v-else variant="outline" class="border-amber-600 text-amber-600">
              <AlertCircle class="h-3 w-3 mr-1" /> {{ $t('accounts.disconnected', 'Disconnected') }}
            </Badge>
            <span v-if="gowaStatus.jid" class="text-xs text-muted-foreground font-mono">{{ gowaStatus.jid }}</span>
          </template>
        </div>

        <Tabs default-value="qr">
          <TabsList class="grid w-full grid-cols-2">
            <TabsTrigger value="qr">
              <QrCode class="h-4 w-4 mr-1.5" /> {{ $t('accounts.qrCode', 'QR Code') }}
            </TabsTrigger>
            <TabsTrigger value="pair">
              <Link2 class="h-4 w-4 mr-1.5" /> {{ $t('accounts.pairCode', 'Pair Code') }}
            </TabsTrigger>
          </TabsList>

          <!-- QR Code Tab -->
          <TabsContent value="qr" class="flex flex-col items-center gap-3 py-4">
            <div class="relative w-64 h-64 bg-white rounded-lg flex items-center justify-center border border-border shadow-inner">
              <Loader2 v-if="gowaQrLoading && !gowaQrLink" class="h-8 w-8 animate-spin text-muted-foreground" />
              <img v-else-if="gowaQrLink" :src="gowaQrLink" alt="QR Code" class="w-full h-full object-contain p-2" />
              <QrCode v-else class="h-16 w-16 text-muted-foreground" />
            </div>
            <p class="text-xs text-muted-foreground text-center">
              {{ $t('accounts.qrInstructions', 'Open WhatsApp on your phone → Settings → Linked Devices → Link a Device → scan this code') }}
            </p>
            <Button variant="outline" size="sm" :disabled="gowaQrLoading" @click="fetchGowaQr">
              <RefreshCw class="h-4 w-4 mr-1" :class="{ 'animate-spin': gowaQrLoading }" />
              {{ $t('accounts.refreshQr', 'Refresh QR') }}
            </Button>
          </TabsContent>

          <!-- Pair Code Tab -->
          <TabsContent value="pair" class="space-y-4 py-4">
            <div class="space-y-2">
              <Label class="text-xs">{{ $t('accounts.phoneNumber', 'Phone Number') }}</Label>
              <div class="flex gap-2">
                <Input v-model="gowaPairPhone" placeholder="16505551234" class="flex-1" />
                <Button size="sm" :disabled="gowaPairLoading || !gowaPairPhone.trim()" @click="fetchGowaPairCode">
                  <Loader2 v-if="gowaPairLoading" class="h-4 w-4 animate-spin mr-1" />
                  {{ $t('accounts.getCode', 'Get Code') }}
                </Button>
              </div>
              <p class="text-xs text-muted-foreground">
                {{ $t('accounts.pairCodeInstructions', 'Enter your phone number with country code. You will receive an 8-digit code to enter on your phone.') }}
              </p>
            </div>
            <div v-if="gowaPairCode" class="flex flex-col items-center gap-2 p-4 bg-muted rounded-lg">
              <span class="text-xs text-muted-foreground">{{ $t('accounts.yourPairCode', 'Your Pair Code') }}</span>
              <span class="text-3xl font-bold font-mono tracking-[0.3em]">{{ gowaPairCode }}</span>
              <Button variant="ghost" size="sm" @click="copyToClipboard(gowaPairCode)">
                <Copy class="h-3 w-3 mr-1" /> {{ $t('common.copy', 'Copy') }}
              </Button>
            </div>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>

    <UnsavedChangesDialog :open="showLeaveDialog" @stay="cancelLeave" @leave="confirmLeave" />
  </div>
</template>
