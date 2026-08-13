<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Plus } from 'lucide-vue-next'
import { PageHeader, DataTable, DeleteConfirmDialog, ErrorState, type Column } from '@/components/shared'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { api } from '@/services/api'
import { useOrganizationsStore } from '@/stores/organizations'
import { useAuthStore } from '@/stores/auth'
import { useGowaServersStore } from '@/stores/gowaServers'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate } from '@/lib/utils'
import {
  Phone,
  Pencil,
  Trash2,
  Check,
  CheckCircle2,
  Wifi,
  WifiOff,
  Server,
  Link2
} from 'lucide-vue-next'

// NOTE: The "Server Connection" tab (ConnectionCard / ServerInfoCard) was
// removed — it made the browser probe GOWA directly (Path B), the source of
// the CORS / mixed-content / wrong-host issues. Account→GOWA connectivity is
// now managed entirely server-side via /api/gowa/servers (gowa_instances).

const { t } = useI18n()
const organizationsStore = useOrganizationsStore()
const authStore = useAuthStore()
const gowaServersStore = useGowaServersStore()

interface WhatsAppAccount {
  id: string
  name: string
  is_default_incoming: boolean
  is_default_outgoing: boolean
  status: string
  created_at: string
  gowa_jid?: string
  gowa_connected?: boolean | null
  gowa_base_url?: string
  gowa_device_id?: string
}

const accounts = ref<WhatsAppAccount[]>([])
const isLoading = ref(true)
const fetchError = ref(false)
const deleteDialogOpen = ref(false)
const accountToDelete = ref<WhatsAppAccount | null>(null)
const isDeleting = ref(false)

const canWrite = computed(() => authStore.hasPermission('accounts', 'write'))
const canDelete = computed(() => authStore.hasPermission('accounts', 'delete'))
const canWriteDevices = computed(() => authStore.hasPermission('devices', 'write'))
const breadcrumbs = computed(() => [{ label: t('nav.settings'), href: '/settings' }, { label: t('settings.accounts', 'Accounts') }])

const sortKey = ref('name')
const sortDirection = ref<'asc' | 'desc'>('asc')

// Inline connection-status summary (header icon + popover, no modal)
const connectedCount = computed(() => accounts.value.filter(a => a.gowa_connected === true).length)
const disconnectedCount = computed(() => accounts.value.length - connectedCount.value)
const allConnected = computed(() => accounts.value.length > 0 && disconnectedCount.value === 0)

const columns = computed<Column<WhatsAppAccount>[]>(() => [
  { key: 'account', label: t('accounts.account', 'Account Name'), width: 'w-[250px]', sortable: true, sortKey: 'name' },
  { key: 'base_url', label: t('accounts.gowaBaseUrl', 'GOWA Base URL') },
  { key: 'device_id', label: t('accounts.gowaDeviceId', 'Device / JID') },
  { key: 'status', label: t('accounts.status', 'Status'), sortable: true, sortKey: 'status' },
  { key: 'defaults', label: t('accounts.defaults', 'Defaults') },
  { key: 'created', label: t('common.created', 'Created'), sortable: true, sortKey: 'created_at' },
  { key: 'actions', label: t('common.actions', 'Actions'), align: 'right' },
])

watch(() => organizationsStore.selectedOrgId, () => {
  fetchAccounts()
  gowaServersStore.fetchServers().catch(() => {})
})

onMounted(() => {
  fetchAccounts()
  gowaServersStore.fetchServers().catch(() => {})
})

async function fetchAccounts() {
  isLoading.value = true
  fetchError.value = false
  try {
    const response = await api.get('/accounts')
    accounts.value = response.data.data?.accounts || []
  } catch {
    fetchError.value = true
    toast.error(t('common.failedLoad', { resource: t('resources.accounts', 'accounts') }))
  } finally {
    isLoading.value = false
  }
}

function openDeleteDialog(account: WhatsAppAccount) {
  accountToDelete.value = account
  deleteDialogOpen.value = true
}

// Deep-link target for pairing/connecting a device on the GOWA Gateway page.
// The account's base URL is matched (trailing-slash normalized) against the
// DB-managed servers; returns null for config-based/unmatched servers so the
// UI falls back to the plain gateway link.
function linkForAccount(account: WhatsAppAccount) {
  const norm = (u: string) => (u || '').replace(/\/+$/, '')
  const base = norm(account.gowa_base_url || '')
  if (!account.gowa_device_id || !base) return null
  const server = gowaServersStore.servers.find(s => norm(s.base_url) === base)
  if (!server) return null
  return {
    path: `/settings/gowa-servers/${server.id}`,
    query: { device: account.gowa_device_id, connect: '1' },
  }
}

async function confirmDelete() {
  if (!accountToDelete.value) return
  isDeleting.value = true
  try {
    await api.delete(`/accounts/${accountToDelete.value.id}`)
    toast.success(t('common.deletedSuccess', { resource: t('resources.Account', 'Account') }))
    deleteDialogOpen.value = false
    accountToDelete.value = null
    await fetchAccounts()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.account', 'account') })))
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('accounts.title', 'Accounts & Connection')"
      :icon="Phone"
      icon-gradient="bg-gradient-to-br from-emerald-500 to-green-600 shadow-emerald-500/20"
      back-link="/settings"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <Popover>
            <PopoverTrigger as-child>
              <Button variant="outline" size="sm" class="gap-1.5">
                <Wifi v-if="allConnected" class="h-4 w-4 text-emerald-500" />
                <WifiOff v-else class="h-4 w-4 text-amber-500" />
                <span class="text-xs tabular-nums">{{ connectedCount }}/{{ accounts.length }}</span>
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" class="w-72 p-0">
              <div class="px-3 py-2 border-b border-border/40">
                <p class="text-sm font-medium">{{ $t('accounts.connectionStatus', 'Connection Status') }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ connectedCount }} {{ $t('accounts.connected', 'Connected') }} · {{ disconnectedCount }} {{ $t('accounts.disconnected', 'Disconnected') }}
                </p>
              </div>
              <div class="max-h-64 overflow-y-auto py-1">
                <p v-if="accounts.length === 0" class="px-3 py-2 text-xs text-muted-foreground">
                  {{ $t('accounts.noAccounts', 'No WhatsApp accounts linked') }}
                </p>
                <div
                  v-for="account in accounts"
                  :key="account.id"
                  class="flex items-center justify-between gap-2 px-3 py-1.5 hover:bg-muted/50"
                >
                  <div class="flex items-center gap-2 min-w-0">
                    <span
                      class="h-2 w-2 rounded-full shrink-0"
                      :class="account.gowa_connected === true ? 'bg-emerald-500' : 'bg-amber-500'"
                    />
                    <span class="text-sm truncate">{{ account.name }}</span>
                  </div>
                  <Badge
                    v-if="account.gowa_connected === true"
                    variant="outline"
                    class="text-[10px] border-emerald-600 text-emerald-600 bg-emerald-500/10 shrink-0"
                  >
                    {{ $t('accounts.connected', 'Connected') }}
                  </Badge>
                  <Badge
                    v-else-if="account.gowa_connected === false"
                    variant="outline"
                    class="text-[10px] border-amber-600 text-amber-600 bg-amber-500/10 shrink-0"
                  >
                    {{ $t('accounts.disconnected', 'Disconnected') }}
                  </Badge>
                  <Badge v-else variant="outline" class="text-[10px] shrink-0">
                    {{ account.status }}
                  </Badge>
                </div>
              </div>
            </PopoverContent>
          </Popover>
          <RouterLink v-if="canWrite" to="/settings/gowa-servers">
            <Button size="sm" class="bg-emerald-600 hover:bg-emerald-700 text-white font-medium shadow-sm">
              <Plus class="h-4 w-4 mr-1.5" />
              {{ $t('accounts.addNumber', 'Add WhatsApp Number') }}
            </Button>
          </RouterLink>
        </div>
      </template>
    </PageHeader>

    <ErrorState
      v-if="fetchError && !isLoading"
      :title="$t('common.loadErrorTitle', 'Failed to load accounts')"
      :description="$t('common.loadErrorDescription', 'Check your connection and try again.')"
      class="flex-1"
    >
      <template #action>
        <Button size="sm" @click="fetchAccounts">{{ $t('common.retry', 'Retry') }}</Button>
      </template>
    </ErrorState>

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <Tabs defaultValue="accounts" class="w-full space-y-6">
          <div class="flex items-center justify-between border-b border-border/40 pb-3">
            <TabsList class="bg-muted/50 p-1">
              <TabsTrigger value="accounts" class="flex items-center gap-2 text-xs">
                <Phone class="h-3.5 w-3.5 text-emerald-500" />
                {{ $t('accounts.yourAccounts', 'WhatsApp Accounts') }}
              </TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="accounts" class="space-y-6 mt-0">
            <!-- Gateway card — the canonical entry point for device lifecycle -->
            <Card>
              <CardHeader>
                <div class="flex items-center justify-between">
                  <div>
                    <CardTitle>{{ $t('accounts.gatewayCardTitle', 'GOWA Gateway') }}</CardTitle>
                    <CardDescription>{{ $t('accounts.gatewayCardDesc', 'GOWA server instances and their WhatsApp devices.') }}</CardDescription>
                  </div>
                  <RouterLink to="/settings/gowa-servers">
                    <Button variant="outline" size="sm">
                      <Server class="h-4 w-4 mr-1.5" />
                      {{ $t('accounts.gatewayPage', 'GOWA Gateway') }}
                    </Button>
                  </RouterLink>
                </div>
              </CardHeader>
              <CardContent>
                <div v-if="gowaServersStore.loading" class="text-xs text-muted-foreground py-2">
                  {{ $t('common.loading', 'Loading...') }}
                </div>
                <p v-else-if="gowaServersStore.servers.length === 0" class="text-xs text-muted-foreground py-2">
                  {{ $t('accounts.noServersHint', 'No GOWA servers configured. Add one to start managing WhatsApp numbers.') }}
                </p>
                <ul v-else class="divide-y divide-border/50">
                  <li
                    v-for="s in gowaServersStore.servers"
                    :key="s.id"
                    class="flex items-center justify-between gap-3 py-2 first:pt-0 last:pb-0"
                  >
                    <div class="flex items-center gap-2 min-w-0">
                      <Server class="h-3.5 w-3.5 text-emerald-500 shrink-0" />
                      <span class="text-sm truncate">{{ s.name }}</span>
                      <code class="text-[10px] bg-muted px-1.5 py-0.5 rounded font-mono truncate">{{ s.base_url }}</code>
                    </div>
                    <Badge v-if="s.is_active" variant="outline" class="text-[10px] border-emerald-600 text-emerald-600 shrink-0">
                      {{ $t('gowaServers.isActive', 'Active') }}
                    </Badge>
                    <Badge v-else variant="outline" class="text-[10px] text-muted-foreground shrink-0">Inactive</Badge>
                  </li>
                </ul>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <div class="flex items-center justify-between">
                  <div>
                    <CardTitle>{{ $t('accounts.yourAccounts', 'WhatsApp Accounts') }}</CardTitle>
                    <CardDescription>{{ $t('accounts.yourAccountsDesc', 'Manage connected GOWA WhatsApp devices') }}</CardDescription>
                  </div>
                  <RouterLink to="/settings/gowa-servers" v-if="canWrite">
                    <Button variant="outline" size="sm">
                      <Plus class="h-4 w-4 mr-1.5" />
                      {{ $t('accounts.addNumber', 'Add WhatsApp Number') }}
                    </Button>
                  </RouterLink>
                </div>
              </CardHeader>
              <CardContent>
                <DataTable
                  :items="accounts"
                  :columns="columns"
                  :is-loading="isLoading"
                  :empty-icon="Phone"
                  :empty-title="$t('accounts.noAccounts', 'No WhatsApp accounts linked')"
                  :empty-description="$t('accounts.noAccountsDesc', 'Link a GOWA device to start sending and receiving WhatsApp messages.')"
                  v-model:sort-key="sortKey"
                  v-model:sort-direction="sortDirection"
                  item-name="accounts"
                >
                  <template #empty-action>
                    <div v-if="canWrite" class="flex gap-3 justify-center">
                      <RouterLink to="/settings/gowa-servers">
                        <Button size="lg" class="bg-emerald-600 hover:bg-emerald-700 text-white font-medium">
                          <Plus class="mr-2 h-5 w-5" />
                          {{ $t('accounts.addNumber', 'Add WhatsApp Number') }}
                        </Button>
                      </RouterLink>
                    </div>
                  </template>

                  <template #cell-account="{ item: account }">
                    <RouterLink :to="`/settings/accounts/${account.id}`" class="flex items-center gap-3 text-inherit no-underline hover:opacity-80">
                      <div class="h-9 w-9 rounded-full bg-emerald-500/10 flex items-center justify-center flex-shrink-0">
                        <Phone class="h-4 w-4 text-emerald-500" />
                      </div>
                      <div class="flex flex-col gap-0.5">
                        <p class="font-medium truncate text-sm">{{ account.name }}</p>
                        <span class="text-[10px] font-medium text-blue-500 dark:text-blue-400">GOWA</span>
                      </div>
                    </RouterLink>
                  </template>

                  <template #cell-base_url="{ item: account }">
                    <code v-if="account.gowa_base_url" class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">{{ account.gowa_base_url }}</code>
                    <span v-else class="text-muted-foreground text-xs">—</span>
                  </template>

                  <template #cell-device_id="{ item: account }">
                    <code v-if="account.gowa_jid" class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">{{ account.gowa_jid }}</code>
                    <code v-else-if="account.gowa_device_id" class="text-xs bg-muted/60 px-1.5 py-0.5 rounded font-mono">{{ account.gowa_device_id }}</code>
                    <span v-else class="text-xs text-muted-foreground">—</span>
                  </template>

                  <template #cell-status="{ item: account }">
                    <Badge v-if="account.gowa_connected === true" variant="outline" class="border-emerald-600 text-emerald-600 bg-emerald-500/10">
                      <CheckCircle2 class="h-3 w-3 mr-1" /> Connected
                    </Badge>
                    <Badge v-else-if="account.gowa_connected === false" variant="outline" class="border-amber-600 text-amber-600 bg-amber-500/10">
                      Disconnected
                    </Badge>
                    <Badge v-else variant="outline" :class="account.status === 'active' ? 'border-emerald-600 text-emerald-600' : ''">
                      {{ account.status }}
                    </Badge>
                  </template>

                  <template #cell-defaults="{ item: account }">
                    <div class="flex items-center gap-1.5 flex-wrap">
                      <Badge v-if="account.is_default_incoming" variant="outline" class="text-[10px]">
                        <Check class="h-2.5 w-2.5 mr-0.5" /> {{ $t('accounts.incoming', 'Incoming') }}
                      </Badge>
                      <Badge v-if="account.is_default_outgoing" variant="outline" class="text-[10px]">
                        <Check class="h-2.5 w-2.5 mr-0.5" /> {{ $t('accounts.outgoing', 'Outgoing') }}
                      </Badge>
                    </div>
                  </template>

                  <template #cell-created="{ item: account }">
                    <span class="text-muted-foreground text-xs">{{ formatDate(account.created_at) }}</span>
                  </template>

                  <template #cell-actions="{ item: account }">
                    <div class="flex items-center justify-end gap-1">
                      <Tooltip v-if="canWriteDevices && account.gowa_connected !== true">
                        <TooltipTrigger as-child>
                          <RouterLink v-if="linkForAccount(account)" :to="linkForAccount(account)!">
                            <Button variant="ghost" size="icon" class="h-8 w-8"><Link2 class="h-4 w-4" /></Button>
                          </RouterLink>
                          <RouterLink v-else to="/settings/gowa-servers">
                            <Button variant="ghost" size="icon" class="h-8 w-8"><Link2 class="h-4 w-4" /></Button>
                          </RouterLink>
                        </TooltipTrigger>
                        <TooltipContent>{{ $t('accounts.linkDevice', 'Link device') }}</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger as-child>
                          <RouterLink :to="`/settings/accounts/${account.id}`">
                            <Button variant="ghost" size="icon" class="h-8 w-8"><Pencil class="h-4 w-4" /></Button>
                          </RouterLink>
                        </TooltipTrigger>
                        <TooltipContent>{{ $t('common.edit', 'Edit') }}</TooltipContent>
                      </Tooltip>
                      <Tooltip v-if="canDelete">
                        <TooltipTrigger as-child>
                          <Button variant="ghost" size="icon" class="h-8 w-8" @click="openDeleteDialog(account)">
                            <Trash2 class="h-4 w-4 text-destructive" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{{ $t('common.delete', 'Delete') }}</TooltipContent>
                      </Tooltip>
                    </div>
                  </template>
                </DataTable>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </ScrollArea>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('accounts.deleteAccount', 'Delete Account')"
      :item-name="accountToDelete?.name"
      :is-submitting="isDeleting"
      @confirm="confirmDelete"
    />
  </div>
</template>
