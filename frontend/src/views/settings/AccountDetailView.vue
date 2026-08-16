<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useGowaServersStore } from '@/stores/gowaServers'
import { api } from '@/services/api'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { useUnsavedChangesGuard } from '@/composables/useUnsavedChangesGuard'
import DetailPageLayout from '@/components/shared/DetailPageLayout.vue'
import AuditLogPanel from '@/components/shared/AuditLogPanel.vue'
import MetadataPanel from '@/components/shared/MetadataPanel.vue'
import UnsavedChangesDialog from '@/components/shared/UnsavedChangesDialog.vue'
import AccountCloseRatingPanel from '@/components/settings/AccountCloseRatingPanel.vue'
import AccountCallRejectPanel from '@/components/settings/AccountCallRejectPanel.vue'
import AccountChatResetPanel from '@/components/settings/AccountChatResetPanel.vue'
import AccountSendPacingPanel from '@/components/settings/AccountSendPacingPanel.vue'
import AccountBusinessHoursPanel from '@/components/settings/AccountBusinessHoursPanel.vue'
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
  Loader2,
  Smartphone,
  Lightbulb,
  ServerCog,
  Route,
  Cpu,
  ExternalLink,
  Webhook,
} from 'lucide-vue-next'

// NOTE: Device lifecycle (QR/pair connect, reconnect, logout, sync, webhook
// URL/events, delete) lives on the GOWA Gateway page — the canonical source.
// This page keeps the business configuration (routing, automation) plus a
// live device-connection glance, and links out to the gateway for device ops.

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
const gowaServersStore = useGowaServersStore()

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

// Single Activity Log aggregates every audit entry tied to this account: the
// account itself plus its per-account settings blocks (close_rating,
// call_auto_reject and daily_reset), which share the account's resource_id.
// Bumped after the account or a child panel saves to remount the panel and
// refetch.
const accountLogKey = ref(0)
const accountLogResourceTypes = [
  'account',
  'settings.close_rating',
  'settings.call_auto_reject',
  'settings.chat_reset',
]
function bumpAccountLog() {
  accountLogKey.value++
}

// Live device-connection status, polled from the backend (which proxies GOWA).
// This replaces the old "connection summary" that reflected the account's
// config `status` field instead of the actual WhatsApp session state.
const liveStatus = ref<{ is_connected: boolean; is_logged_in: boolean; jid: string } | null>(null)
const liveStatusLoading = ref(false)
let liveStatusTimer: ReturnType<typeof setInterval> | null = null

async function fetchLiveStatus() {
  if (!account.value || !account.value.gowa_device_id) return
  liveStatusLoading.value = true
  try {
    const resp = await api.get(`/accounts/${account.value.id}/gowa/status`)
    liveStatus.value = resp.data.data || resp.data
  } catch {
    liveStatus.value = null
  } finally {
    liveStatusLoading.value = false
  }
}

function startLiveStatusPoll() {
  stopLiveStatusPoll()
  fetchLiveStatus()
  liveStatusTimer = setInterval(fetchLiveStatus, 5000)
}

function stopLiveStatusPoll() {
  if (liveStatusTimer) {
    clearInterval(liveStatusTimer)
    liveStatusTimer = null
  }
}

onBeforeUnmount(stopLiveStatusPoll)

const deviceStatus = computed(() => {
  if (liveStatus.value?.is_connected) {
    return { tone: 'active', label: t('accounts.connected', 'Connected') }
  }
  if (liveStatus.value) {
    return { tone: 'inactive', label: t('accounts.disconnected', 'Disconnected') }
  }
  return { tone: 'unknown', label: t('accounts.deviceStatusUnknown', 'Status unknown') }
})

// Resolve the DB-managed GOWA server for this account's base URL so device
// management can deep-link into the gateway page. Falls back to a plain link
// when the base URL is config-based (no DB server row) or unmatched.
const matchedServer = computed(() => {
  const norm = (u: string) => (u || '').replace(/\/+$/, '')
  const base = norm(account.value?.gowa_base_url || '')
  if (!base) return null
  return gowaServersStore.servers.find(s => norm(s.base_url) === base) || null
})

const manageDeviceTarget = computed(() => {
  if (!account.value?.gowa_device_id || !matchedServer.value) return null
  return {
    path: `/settings/gowa-servers/${matchedServer.value.id}`,
    query: { device: account.value.gowa_device_id, connect: '1' },
  }
})

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
    if (!isNew.value && account.value?.gowa_device_id) {
      startLiveStatusPoll()
    }
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
      // loadAccount() refetched the account; the Activity Log is keyed, so bump
      // it after a short delay to let the backend persist the audit entry.
      setTimeout(bumpAccountLog, 500)
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

onMounted(async () => {
  gowaServersStore.fetchServers().catch(() => {})
  if (isNew.value) {
    isLoading.value = false
    hasChanges.value = false
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
      wide
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <template v-if="!isNew && account && canWriteDevices">
            <RouterLink v-if="manageDeviceTarget" :to="manageDeviceTarget">
              <Button variant="outline" size="sm">
                <ExternalLink class="h-4 w-4 me-1.5" />
                {{ $t('accounts.manageDevice', 'Manage Device') }}
              </Button>
            </RouterLink>
            <RouterLink v-else to="/settings/gowa-servers">
              <Button variant="outline" size="sm">
                <ServerCog class="h-4 w-4 me-1.5" />
                {{ $t('accounts.gatewayPage', 'GOWA Gateway') }}
              </Button>
            </RouterLink>
          </template>
          <Button v-if="canWrite && (hasChanges || isNew)" size="sm" @click="save" :disabled="isSaving" class="bg-emerald-600 hover:bg-emerald-700 text-white font-medium">
            <Save class="h-4 w-4 me-1.5" /> {{ isSaving ? $t('common.saving', 'Saving...') : isNew ? $t('common.create', 'Create') : $t('common.save', 'Save') }}
          </Button>
          <Button v-if="!isNew && canDelete" variant="ghost" size="icon" class="text-destructive hover:text-destructive" @click="deleteDialogOpen = true">
            <Trash2 class="h-4 w-4" />
          </Button>
        </div>
      </template>

      <!-- ════════════ SECTION 1 — Identity & Connection ════════════ -->
      <section class="space-y-3">
        <Card v-if="isNew" class="border-emerald-500/30 bg-emerald-500/5">
          <CardContent class="py-3 flex items-center justify-between gap-3 flex-wrap">
            <p class="text-xs text-muted-foreground min-w-0">
              <Lightbulb class="h-3.5 w-3.5 inline me-1.5 text-amber-500 -mt-0.5" />
              {{ $t('accounts.tips.connect', 'Create the device in the GOWA Gateway — the account is provisioned automatically.') }}
            </p>
            <RouterLink to="/settings/gowa-servers">
              <Button variant="outline" size="sm" class="shrink-0">
                <ServerCog class="h-4 w-4 me-1.5" />
                {{ $t('accounts.gatewayPage', 'GOWA Gateway') }}
              </Button>
            </RouterLink>
          </CardContent>
        </Card>

        <header class="flex items-start gap-3">
          <div class="h-7 w-7 rounded-md bg-emerald-500/15 text-emerald-400 flex items-center justify-center shrink-0">
            <ServerCog class="h-4 w-4" />
          </div>
          <div class="min-w-0">
            <h2 class="text-sm font-semibold leading-tight">{{ $t('accounts.sectionIdentity') }}</h2>
            <p class="text-xs text-muted-foreground mt-0.5">{{ $t('accounts.sectionIdentityDesc') }}</p>
          </div>
        </header>

        <Card>
          <CardHeader class="pb-4">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <CardTitle class="text-sm font-medium">{{ $t('accounts.accountDetails') }}</CardTitle>
                <CardDescription class="text-xs">{{ $t('accounts.sectionConnectionDesc') }}</CardDescription>
              </div>
              <!-- Prominent live device-status pill -->
              <div
                v-if="account"
                class="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-full text-xs font-medium shrink-0"
                :class="deviceStatus.tone === 'active'
                  ? 'bg-emerald-500/15 text-emerald-400'
                  : deviceStatus.tone === 'inactive'
                    ? 'bg-amber-500/15 text-amber-500'
                    : 'bg-muted text-muted-foreground'"
              >
                <span
                  class="h-1.5 w-1.5 rounded-full"
                  :class="deviceStatus.tone === 'active' ? 'bg-emerald-400' : deviceStatus.tone === 'inactive' ? 'bg-amber-500' : 'bg-muted-foreground/60'"
                />
                {{ deviceStatus.label }}
              </div>
            </div>
          </CardHeader>

          <CardContent class="space-y-5">
            <!-- Account name — full width, slightly taller for identity emphasis -->
            <div class="space-y-1.5">
              <Label for="account_name" class="text-xs">
                {{ $t('accounts.accountName') }} <span class="text-destructive">*</span>
              </Label>
              <Input
                id="account_name"
                v-model="form.name"
                :placeholder="$t('accounts.accountNamePlaceholder', 'e.g. Sales WhatsApp')"
                :disabled="!canWrite"
                class="h-9"
              />
              <p class="text-[11px] text-muted-foreground">{{ $t('accounts.nameHint') }}</p>
            </div>

            <Separator />

            <!-- GOWA connection grid — unified for new + existing accounts -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <Label for="gowa_base_url" class="text-xs">
                  {{ $t('accounts.gowaBaseUrl') }} <span class="text-destructive">*</span>
                </Label>
                <Input
                  id="gowa_base_url"
                  v-model="form.gowa_base_url"
                  :placeholder="isNew ? 'https://your-gowa-server.example.com' : ''"
                  class="font-mono text-xs h-9"
                  :disabled="!canWrite"
                />
              </div>
              <div class="space-y-1.5">
                <Label for="gowa_device_id" class="text-xs">
                  {{ $t('accounts.gowaDeviceId') }} <span class="text-destructive">*</span>
                </Label>
                <Input
                  id="gowa_device_id"
                  v-model="form.gowa_device_id"
                  :placeholder="isNew ? 'e.g. device_1' : ''"
                  class="font-mono text-xs h-9"
                  :disabled="!canWrite"
                />
              </div>
              <div class="space-y-1.5">
                <Label for="gowa_username" class="text-xs">{{ $t('accounts.gowaUsernameOptional') }}</Label>
                <Input
                  id="gowa_username"
                  v-model="form.gowa_username"
                  :placeholder="isNew ? 'basic-auth user' : ''"
                  class="h-9"
                  :disabled="!canWrite"
                />
              </div>
              <div class="space-y-1.5">
                <Label for="gowa_password" class="text-xs">{{ $t('accounts.gowaPasswordOptional') }}</Label>
                <Input
                  id="gowa_password"
                  v-model="form.gowa_password"
                  type="password"
                  :placeholder="isNew ? 'basic-auth password' : ''"
                  class="h-9"
                  :disabled="!canWrite"
                />
              </div>
            </div>
            <p class="text-[11px] text-muted-foreground -mt-2">{{ $t('accounts.gowaAuthHint') }}</p>
            <p class="text-[11px] text-muted-foreground -mt-3">
              {{ $t('accounts.deviceLifecycleHint', 'Device pairing and connection controls live in the GOWA Gateway.') }}
            </p>
          </CardContent>
        </Card>
      </section>

      <!-- ════════════ Row 2 — Routing ‖ Quick-Ref ‖ Tips ════════════ -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <!-- Routing Defaults -->
        <section class="space-y-3">
          <header class="flex items-start gap-3">
            <div class="h-7 w-7 rounded-md bg-emerald-500/15 text-emerald-400 flex items-center justify-center shrink-0">
              <Route class="h-4 w-4" />
            </div>
            <div class="min-w-0">
              <h2 class="text-sm font-semibold leading-tight">{{ $t('accounts.sectionRouting') }}</h2>
              <p class="text-xs text-muted-foreground mt-0.5">{{ $t('accounts.sectionRoutingDesc') }}</p>
            </div>
          </header>
          <Card>
            <CardContent class="pt-6">
              <ul class="divide-y divide-border/50">
                <li class="flex items-center justify-between gap-4 py-3 first:pt-0">
                  <div class="min-w-0">
                    <Label class="text-xs">{{ $t('accounts.defaultIncoming') }}</Label>
                    <p class="text-[11px] text-muted-foreground mt-0.5">{{ $t('settings.incomingRoutingDesc') }}</p>
                  </div>
                  <Switch :checked="form.is_default_incoming" @update:checked="form.is_default_incoming = $event" :disabled="!canWrite" class="shrink-0" />
                </li>
                <li class="flex items-center justify-between gap-4 py-3">
                  <div class="min-w-0">
                    <Label class="text-xs">{{ $t('accounts.defaultOutgoing') }}</Label>
                    <p class="text-[11px] text-muted-foreground mt-0.5">{{ $t('settings.outgoingRoutingDesc') }}</p>
                  </div>
                  <Switch :checked="form.is_default_outgoing" @update:checked="form.is_default_outgoing = $event" :disabled="!canWrite" class="shrink-0" />
                </li>
                <li class="flex items-center justify-between gap-4 py-3 last:pb-0">
                  <div class="min-w-0">
                    <Label class="text-xs">{{ $t('accounts.autoReadReceipt') }}</Label>
                    <p class="text-[11px] text-muted-foreground mt-0.5">{{ $t('settings.readReceiptDesc') }}</p>
                  </div>
                  <Switch :checked="form.auto_read_receipt" @update:checked="form.auto_read_receipt = $event" :disabled="!canWrite" class="shrink-0" />
                </li>
              </ul>
            </CardContent>
          </Card>
        </section>

        <!-- Device Quick-Ref (existing accounts only) -->
        <section v-if="account && !isNew" class="space-y-3">
          <header class="flex items-start gap-3">
            <div class="h-7 w-7 rounded-md bg-emerald-500/15 text-emerald-400 flex items-center justify-center shrink-0">
              <Smartphone class="h-4 w-4" />
            </div>
            <div class="min-w-0">
              <h2 class="text-sm font-semibold leading-tight">{{ $t('accounts.deviceQuickRef') }}</h2>
              <p class="text-xs text-muted-foreground mt-0.5">{{ $t('accounts.deviceQuickRefDesc') }}</p>
            </div>
          </header>
          <Card>
            <CardContent class="space-y-4 pt-6">
              <!-- Live connection status row -->
              <div class="flex items-center gap-2 flex-wrap">
                <Loader2 v-if="liveStatusLoading && !liveStatus" class="h-3.5 w-3.5 animate-spin text-muted-foreground shrink-0" />
                <template v-else>
                  <span
                    class="h-2 w-2 rounded-full shrink-0"
                    :class="liveStatus?.is_connected ? 'bg-emerald-400' : 'bg-muted-foreground/50'"
                  />
                  <span class="text-sm font-medium">{{ deviceStatus.label }}</span>
                  <span v-if="liveStatus?.jid" class="text-xs text-muted-foreground font-mono truncate min-w-0" :title="liveStatus.jid">{{ liveStatus.jid }}</span>
                </template>
              </div>
              <Separator />
              <!-- key/value reference -->
              <dl class="space-y-3 text-xs">
                <div class="flex items-start justify-between gap-3">
                  <dt class="text-muted-foreground shrink-0">{{ $t('accounts.gowaDeviceId') }}</dt>
                  <dd class="font-mono text-end truncate min-w-0" :title="form.gowa_device_id">{{ form.gowa_device_id || '—' }}</dd>
                </div>
                <div class="flex items-start justify-between gap-3">
                  <dt class="text-muted-foreground shrink-0">{{ $t('accounts.gowaBaseUrl') }}</dt>
                  <dd class="font-mono text-end truncate min-w-0" :title="form.gowa_base_url">{{ form.gowa_base_url || '—' }}</dd>
                </div>
              </dl>
              <RouterLink v-if="manageDeviceTarget" :to="manageDeviceTarget">
                <Button v-if="canWriteDevices" variant="outline" size="sm" class="w-full">
                  <ExternalLink class="h-4 w-4 me-1.5" />
                  {{ $t('accounts.manageDevice', 'Manage Device') }}
                </Button>
              </RouterLink>
              <RouterLink v-else to="/settings/gowa-servers">
                <Button v-if="canWriteDevices" variant="outline" size="sm" class="w-full">
                  <ServerCog class="h-4 w-4 me-1.5" />
                  {{ $t('accounts.gatewayPage', 'GOWA Gateway') }}
                </Button>
              </RouterLink>
            </CardContent>
          </Card>
        </section>

        <!-- Tips (fills the third column for new accounts) -->
        <section class="space-y-3">
          <header class="flex items-start gap-3">
            <div class="h-7 w-7 rounded-md bg-amber-500/15 text-amber-500 flex items-center justify-center shrink-0">
              <Lightbulb class="h-4 w-4" />
            </div>
            <div class="min-w-0">
              <h2 class="text-sm font-semibold leading-tight">{{ $t('accounts.tips.title') }}</h2>
              <p class="text-xs text-muted-foreground mt-0.5">{{ $t('accounts.sectionIdentityDesc') }}</p>
            </div>
          </header>
          <Card>
            <CardContent class="pt-6">
              <ul class="space-y-2.5 text-xs text-muted-foreground leading-relaxed">
                <li class="flex gap-2">
                  <span class="text-emerald-500 mt-0.5 shrink-0">→</span>
                  <span>{{ $t('accounts.tips.name') }}</span>
                </li>
                <li class="flex gap-2">
                  <span class="text-emerald-500 mt-0.5 shrink-0">→</span>
                  <span>{{ $t('accounts.tips.connect') }}</span>
                </li>
                <li class="flex gap-2">
                  <span class="text-emerald-500 mt-0.5 shrink-0">→</span>
                  <span>{{ $t('accounts.tips.save') }}</span>
                </li>
              </ul>
            </CardContent>
          </Card>
        </section>
      </div>

      <!-- ════════════ Section 3 — Automation (3 columns) ════════════ -->
      <section v-if="account && !isNew" class="space-y-3">
        <header class="flex items-start gap-3">
          <div class="h-7 w-7 rounded-md bg-emerald-500/15 text-emerald-400 flex items-center justify-center shrink-0">
            <Cpu class="h-4 w-4" />
          </div>
          <div class="min-w-0">
            <h2 class="text-sm font-semibold leading-tight">{{ $t('accounts.sectionAutomation') }}</h2>
            <p class="text-xs text-muted-foreground mt-0.5">{{ $t('accounts.sectionAutomationDesc') }}</p>
          </div>
        </header>

        <div class="grid grid-cols-1 xl:grid-cols-3 gap-6 items-start">
          <AccountCloseRatingPanel
            :account-id="account.id"
            :can-write="canWrite"
            @saved="bumpAccountLog"
          />
          <AccountCallRejectPanel
            :account-id="account.id"
            :can-write="canWrite"
            @saved="bumpAccountLog"
          />
          <AccountChatResetPanel
            :account-id="account.id"
            :can-write="canWrite"
            @saved="bumpAccountLog"
          />
          <AccountBusinessHoursPanel
            :account-id="account.id"
            :can-write="canWrite"
            @saved="bumpAccountLog"
          />
          <AccountSendPacingPanel
            :account-id="account.id"
            :can-write="canWrite"
            @saved="bumpAccountLog"
          />
        </div>
      </section>

      <!-- ════════════ Section 4 — Webhook Configuration ════════════ -->
      <section v-if="account && !isNew" class="space-y-3">
        <header class="flex items-start gap-3">
          <div class="h-7 w-7 rounded-md bg-emerald-500/15 text-emerald-400 flex items-center justify-center shrink-0">
            <Webhook class="h-4 w-4" />
          </div>
          <div class="min-w-0">
            <h2 class="text-sm font-semibold leading-tight">{{ $t('accounts.webhookConfig', 'Webhook Configuration') }}</h2>
            <p class="text-xs text-muted-foreground mt-0.5">{{ $t('accounts.webhookConfigDesc') }}</p>
          </div>
        </header>
        <Card>
          <CardContent class="space-y-4 pt-6">
            <div class="space-y-1.5">
              <div class="flex items-center justify-between">
                <Label for="gowa_webhook_secret" class="text-xs">{{ $t('accounts.gowaWebhookSecret', 'Webhook secret') }}</Label>
                <Badge v-if="account.has_gowa_webhook_secret" variant="outline" class="border-emerald-600 text-emerald-600 text-[10px]">
                  {{ $t('accounts.configured', 'Configured') }}
                </Badge>
              </div>
              <Input
                id="gowa_webhook_secret"
                v-model="form.gowa_webhook_secret"
                type="password"
                :placeholder="$t('accounts.gowaWebhookSecretPlaceholder', 'Set a new secret (leave blank to keep current)')"
                autocomplete="new-password"
                :disabled="!canWrite"
                class="font-mono text-xs h-9"
              />
            </div>
            <p class="text-[11px] text-muted-foreground">
              {{ $t('accounts.webhookNote', 'The webhook URL and events are managed per device in the GOWA Gateway.') }}
              <RouterLink to="/settings/gowa-servers" class="text-emerald-500 hover:underline">
                {{ $t('accounts.gatewayPage', 'GOWA Gateway') }}
              </RouterLink>
            </p>
          </CardContent>
        </Card>
      </section>

      <!-- Metadata -->
      <MetadataPanel
        v-if="account && !isNew"
        :created-at="account.created_at"
        :updated-at="account.updated_at"
        :created-by-name="account.created_by_name"
        :updated-by-name="account.updated_by_name"
      />

      <!-- Activity Log (aggregated: account + per-account settings blocks) -->
      <AuditLogPanel
        v-if="account && !isNew"
        :key="accountLogKey"
        :resource-type="accountLogResourceTypes"
        :resource-id="account.id"
      />
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

    <UnsavedChangesDialog :open="showLeaveDialog" @stay="cancelLeave" @leave="confirmLeave" />
  </div>
</template>
