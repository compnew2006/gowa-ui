<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { PageHeader, DataTable, DeleteConfirmDialog, ErrorState, type Column } from '@/components/shared'
import { useGowaServersStore } from '@/stores/gowaServers'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationsStore } from '@/stores/organizations'
import { toast } from 'vue-sonner'
import { getErrorMessage } from '@/lib/api-utils'
import { Server, Plus, Pencil, Trash2, CheckCircle2 } from 'lucide-vue-next'
import type { GowaServer } from '@/services/api'

const { t } = useI18n()
const store = useGowaServersStore()
const authStore = useAuthStore()
const organizationsStore = useOrganizationsStore()

const dialogOpen = ref(false)
const editing = ref<GowaServer | null>(null)
const submitting = ref(false)
const deleteOpen = ref(false)
const serverToDelete = ref<GowaServer | null>(null)
const isDeleting = ref(false)

const form = ref({
  name: '',
  base_url: '',
  username: '',
  password: '',
  webhook_url: '',
  is_active: true,
})

const canWrite = computed(() => authStore.hasPermission('gowa_instances', 'write'))
const canDelete = computed(() => authStore.hasPermission('gowa_instances', 'delete'))
const breadcrumbs = computed(() => [
  { label: t('nav.settings'), href: '/settings' },
  { label: t('gowaServers.title', 'GOWA Servers') },
])

const columns = computed<Column<GowaServer>[]>(() => [
  { key: 'name', label: t('gowaServers.name', 'Server Name'), width: 'w-[220px]', sortable: true },
  { key: 'base_url', label: t('gowaServers.baseUrl', 'GOWA Base URL') },
  { key: 'creds', label: t('gowaServers.hasCredentials', 'Credentials') },
  { key: 'status', label: t('gowaServers.isActive', 'Active') },
  { key: 'actions', label: t('common.actions', 'Actions'), align: 'right' },
])

watch(() => organizationsStore.selectedOrgId, () => store.fetchServers())
onMounted(() => store.fetchServers())

function openCreate() {
  editing.value = null
  form.value = { name: '', base_url: '', username: '', password: '', webhook_url: '', is_active: true }
  dialogOpen.value = true
}

function openEdit(s: GowaServer) {
  editing.value = s
  form.value = { name: s.name, base_url: s.base_url, username: '', password: '', webhook_url: s.webhook_url || '', is_active: s.is_active }
  dialogOpen.value = true
}

async function submit() {
  submitting.value = true
  try {
    const payload: any = { ...form.value }
    if (editing.value && !payload.username) delete payload.username
    if (editing.value && !payload.password) delete payload.password
    if (editing.value) {
      await store.updateServer(editing.value.id, payload)
      toast.success(t('gowaServers.updatedSuccess', 'GOWA server updated'))
    } else {
      await store.createServer(payload)
      toast.success(t('gowaServers.createdSuccess', 'GOWA server created'))
    }
    dialogOpen.value = false
  } catch (e: any) {
    const msg = e?.response?.status === 502
      ? t('gowaServers.probeFailed', 'Could not reach the GOWA server with these credentials.')
      : getErrorMessage(e, t('common.failedSave', { resource: 'server' }))
    toast.error(msg)
  } finally {
    submitting.value = false
  }
}

function openDelete(s: GowaServer) {
  serverToDelete.value = s
  deleteOpen.value = true
}

async function confirmDelete() {
  if (!serverToDelete.value) return
  isDeleting.value = true
  try {
    await store.deleteServer(serverToDelete.value.id)
    toast.success(t('gowaServers.deletedSuccess', 'GOWA server deleted'))
    deleteOpen.value = false
    serverToDelete.value = null
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: 'server' })))
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('gowaServers.title', 'GOWA Servers')"
      :icon="Server"
      icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20"
      back-link="/settings"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <Button v-if="canWrite" size="sm" class="bg-blue-600 hover:bg-blue-700 text-white font-medium" @click="openCreate">
          <Plus class="h-4 w-4 mr-1.5" />
          {{ $t('gowaServers.addServer', 'Add GOWA Server') }}
        </Button>
      </template>
    </PageHeader>

    <ErrorState
      v-if="store.error && !store.loading"
      :title="$t('common.loadErrorTitle', 'Failed to load')"
      :description="store.error"
      class="flex-1"
    >
      <template #action>
        <Button size="sm" @click="store.fetchServers()">{{ $t('common.retry', 'Retry') }}</Button>
      </template>
    </ErrorState>

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <Card>
          <CardHeader>
            <CardTitle>{{ $t('gowaServers.title', 'GOWA Servers') }}</CardTitle>
            <CardDescription>{{ $t('gowaServers.subtitle', 'Manage GOWA server instances and their WhatsApp devices') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              :items="store.servers"
              :columns="columns"
              :is-loading="store.loading"
              :empty-icon="Server"
              :empty-title="$t('gowaServers.noServers', 'No GOWA servers configured')"
              :empty-description="$t('gowaServers.noServersDesc', 'Add a GOWA server to manage its WhatsApp devices.')"
              item-name="servers"
            >
              <template #empty-action>
                <Button v-if="canWrite" size="lg" class="bg-blue-600 hover:bg-blue-700 text-white" @click="openCreate">
                  <Plus class="mr-2 h-5 w-5" />
                  {{ $t('gowaServers.addServer', 'Add GOWA Server') }}
                </Button>
              </template>

              <template #cell-name="{ item: s }">
                <RouterLink :to="`/settings/gowa-servers/${s.id}`" class="flex items-center gap-3 text-inherit no-underline hover:opacity-80">
                  <div class="h-9 w-9 rounded-full bg-blue-500/10 flex items-center justify-center flex-shrink-0">
                    <Server class="h-4 w-4 text-blue-500" />
                  </div>
                  <span class="font-medium truncate text-sm">{{ s.name }}</span>
                </RouterLink>
              </template>

              <template #cell-base_url="{ item: s }">
                <code class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">{{ s.base_url }}</code>
              </template>

              <template #cell-creds="{ item: s }">
                <Badge v-if="s.has_credentials" variant="outline" class="border-green-600 text-green-600">
                  <CheckCircle2 class="h-3 w-3 mr-1" /> {{ $t('gowaServers.hasCredentials', 'Credentials set') }}
                </Badge>
                <Badge v-else variant="outline" class="border-amber-600 text-amber-600">
                  {{ $t('gowaServers.noCredentials', 'No credentials') }}
                </Badge>
              </template>

              <template #cell-status="{ item: s }">
                <Badge v-if="s.is_active" variant="outline" class="border-green-600 text-green-600">{{ $t('gowaServers.isActive', 'Active') }}</Badge>
                <Badge v-else variant="outline" class="text-muted-foreground">Inactive</Badge>
              </template>

              <template #cell-actions="{ item: s }">
                <div class="flex items-center justify-end gap-1">
                  <Button v-if="canWrite" variant="ghost" size="icon" class="h-8 w-8" @click="openEdit(s)">
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button v-if="canDelete" variant="ghost" size="icon" class="h-8 w-8" @click="openDelete(s)">
                    <Trash2 class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </template>
            </DataTable>
          </CardContent>
        </Card>
      </div>
    </ScrollArea>

    <Dialog :open="dialogOpen" @update:open="(v) => dialogOpen = v">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? $t('gowaServers.editServer', 'Edit Server') : $t('gowaServers.addServer', 'Add GOWA Server') }}</DialogTitle>
          <DialogDescription>{{ $t('gowaServers.subtitle') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.name', 'Server Name') }}</Label>
            <Input v-model="form.name" placeholder="Production GOWA" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.baseUrl', 'GOWA Base URL') }}</Label>
            <Input v-model="form.base_url" placeholder="http://gowa:8080" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.username', 'Username') }}</Label>
            <Input v-model="form.username" :placeholder="editing ? $t('common.unchanged', 'Unchanged') : 'admin'" autocomplete="off" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.password', 'Password') }}</Label>
            <Input v-model="form.password" type="password" :placeholder="editing ? $t('common.unchanged', 'Unchanged') : '••••••••'" autocomplete="new-password" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('gowaServers.webhookUrl', 'Webhook URL (optional)') }}</Label>
            <Input v-model="form.webhook_url" placeholder="https://whatomate.example.com/api/gowa/webhook" />
          </div>
          <div class="flex items-center justify-between">
            <Label>{{ $t('gowaServers.isActive', 'Active') }}</Label>
            <Switch v-model:checked="form.is_active" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="dialogOpen = false">{{ $t('common.cancel', 'Cancel') }}</Button>
          <Button :disabled="submitting" @click="submit">
            {{ editing ? $t('common.save', 'Save') : $t('gowaServers.addServer', 'Add GOWA Server') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <DeleteConfirmDialog
      v-model:open="deleteOpen"
      :title="$t('gowaServers.deleteServer', 'Delete Server')"
      :item-name="serverToDelete?.name"
      :is-submitting="isDeleting"
      @confirm="confirmDelete"
    />
  </div>
</template>
