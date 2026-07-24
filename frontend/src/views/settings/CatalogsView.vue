<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  PageHeader, DataTable, CrudFormDialog, DeleteConfirmDialog, IconButton, ErrorState, type Column
} from '@/components/shared'
import { catalogsService, accountsService, type Catalog } from '@/services/api'
import { useCrudState } from '@/composables/useCrudState'
import { toast } from 'vue-sonner'
import { ShoppingCart, Plus, Pencil, Trash2, RefreshCw, Package } from 'lucide-vue-next'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate } from '@/lib/utils'
import { RouterLink } from 'vue-router'

interface WhatsAppAccount { id: string; name: string }

const { t } = useI18n()

interface CatalogFormData {
  whatsapp_account: string
  name: string
}
const defaultFormData: CatalogFormData = { whatsapp_account: '', name: '' }

const catalogs = ref<Catalog[]>([])
const accounts = ref<WhatsAppAccount[]>([])
const isLoading = ref(false)
const isDeleting = ref(false)
const isSyncing = ref(false)
const syncAccount = ref('')
const error = ref(false)

const {
  isSubmitting, isDialogOpen, deleteDialogOpen,
  itemToDelete: catalogToDelete, formData, openCreateDialog, closeDialog, openDeleteDialog, closeDeleteDialog,
} = useCrudState<Catalog, CatalogFormData>(defaultFormData)

const sortKey = ref('name')
const sortDirection = ref<'asc' | 'desc'>('asc')

const columns = computed<Column<Catalog>[]>(() => [
  { key: 'name', label: t('catalogs.name', 'Catalog'), sortable: true },
  { key: 'account', label: t('catalogs.account', 'Account') },
  { key: 'products', label: t('catalogs.products', 'Products'), sortable: true, sortKey: 'product_count' },
  { key: 'status', label: t('catalogs.status', 'Status') },
  { key: 'created', label: t('common.created', 'Created'), sortable: true, sortKey: 'created_at' },
  { key: 'actions', label: t('common.actions', 'Actions'), align: 'right' },
])

async function fetchCatalogs() {
  isLoading.value = true
  error.value = false
  try {
    const response = await catalogsService.list()
    catalogs.value = response.data.catalogs || []
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedLoad', { resource: t('resources.catalogs', 'catalogs') })))
    error.value = true
  } finally {
    isLoading.value = false
  }
}

async function fetchAccounts() {
  try {
    const response = await accountsService.list()
    accounts.value = response.data?.accounts || response.data?.data?.accounts || []
    // Pre-select the first account for the create form + sync
    if (accounts.value.length > 0) {
      syncAccount.value = accounts.value[0].name
    }
  } catch {
    // non-fatal; the create form will just show an empty select
  }
}

onMounted(() => {
  fetchCatalogs()
  fetchAccounts()
})

async function saveCatalog() {
  if (!formData.value.name.trim() || !formData.value.whatsapp_account) {
    toast.error(t('catalogs.nameAccountRequired', 'Name and account are required'))
    return
  }
  isSubmitting.value = true
  try {
    // editingCatalog is unsupported by the API (no UpdateCatalog route) —
    // the dialog is create-only. If editingItem is set we ignore it.
    await catalogsService.create({
      whatsapp_account: formData.value.whatsapp_account,
      name: formData.value.name.trim(),
    })
    toast.success(t('common.createdSuccess', { resource: t('resources.Catalog', 'Catalog') }))
    closeDialog()
    await fetchCatalogs()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedSave', { resource: t('resources.catalog', 'catalog') })))
  } finally {
    isSubmitting.value = false
  }
}

async function syncCatalogs() {
  if (!syncAccount.value) {
    toast.error(t('catalogs.selectAccountFirst', 'Select an account to sync'))
    return
  }
  isSyncing.value = true
  try {
    const res = await catalogsService.sync({ whatsapp_account: syncAccount.value })
    const d = res.data
    toast.success(t('catalogs.synced', { synced: d.synced, total: d.total }))
    await fetchCatalogs()
  } catch (e) {
    toast.error(getErrorMessage(e, t('catalogs.syncFailed', 'Sync failed')))
  } finally {
    isSyncing.value = false
  }
}

async function confirmDelete() {
  if (!catalogToDelete.value) return
  isDeleting.value = true
  try {
    await catalogsService.delete(catalogToDelete.value.id)
    toast.success(t('common.deletedSuccess', { resource: t('resources.Catalog', 'Catalog') }))
    closeDeleteDialog()
    await fetchCatalogs()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.catalog', 'catalog') })))
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('catalogs.title', 'Catalogs & Products')"
      :subtitle="$t('catalogs.subtitle', 'Manage WhatsApp Business catalogs and their products')"
      :icon="ShoppingCart"
      icon-gradient="bg-gradient-to-br from-amber-500 to-orange-600 shadow-amber-500/20"
      back-link="/settings"
    >
      <template #actions>
        <Button variant="outline" size="sm" :disabled="isSyncing" @click="syncCatalogs">
          <RefreshCw class="h-4 w-4 mr-2" :class="{ 'animate-spin': isSyncing }" />
          {{ $t('catalogs.sync', 'Sync from Meta') }}
        </Button>
        <Button size="sm" @click="openCreateDialog">
          <Plus class="h-4 w-4 mr-2" />{{ $t('catalogs.addCatalog', 'Add Catalog') }}
        </Button>
      </template>
    </PageHeader>

    <ErrorState
      v-if="error && !isLoading"
      :title="$t('common.loadErrorTitle', 'Failed to load catalogs')"
      :description="$t('common.loadErrorDescription', 'Check your connection and try again.')"
      :retry-label="$t('common.retryLoad', 'Retry')"
      class="flex-1"
      @retry="fetchCatalogs"
    />

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <div class="max-w-6xl mx-auto space-y-4">
          <!-- Sync account selector -->
          <div class="flex items-center gap-3 flex-wrap">
            <Label class="text-sm text-muted-foreground">{{ $t('catalogs.syncAccount', 'Sync account') }}</Label>
            <Select v-model="syncAccount">
              <SelectTrigger class="w-64"><SelectValue :placeholder="$t('catalogs.selectAccount', 'Select account')" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="a in accounts" :key="a.id" :value="a.name">{{ a.name }}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{{ $t('catalogs.yourCatalogs', 'Catalogs') }}</CardTitle>
              <CardDescription>{{ $t('catalogs.yourCatalogsDesc', 'Each catalog syncs with a WhatsApp Business account.') }}</CardDescription>
            </CardHeader>
            <CardContent>
              <DataTable
                :items="catalogs"
                :columns="columns"
                :is-loading="isLoading"
                :empty-icon="ShoppingCart"
                :empty-title="$t('catalogs.noCatalogs', 'No catalogs yet')"
                :empty-description="$t('catalogs.noCatalogsDesc', 'Sync from Meta or create a new catalog to get started.')"
                v-model:sort-key="sortKey"
                v-model:sort-direction="sortDirection"
                item-name="catalogs"
              >
                <template #cell-name="{ item: catalog }">
                  <RouterLink :to="`/settings/catalogs/${catalog.id}`" class="flex items-center gap-3 text-inherit no-underline hover:opacity-80">
                    <div class="h-9 w-9 rounded-full bg-amber-500/10 flex items-center justify-center flex-shrink-0">
                      <ShoppingCart class="h-4 w-4 text-amber-500" />
                    </div>
                    <span class="font-medium truncate text-sm">{{ catalog.name }}</span>
                  </RouterLink>
                </template>
                <template #cell-account="{ item: catalog }">
                  <code class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">{{ catalog.whatsapp_account }}</code>
                </template>
                <template #cell-products="{ item: catalog }">
                  <Badge variant="outline" class="gap-1">
                    <Package class="h-3 w-3" />{{ catalog.product_count }}
                  </Badge>
                </template>
                <template #cell-status="{ item: catalog }">
                  <Badge v-if="catalog.is_active" variant="outline" class="border-emerald-600 text-emerald-600 bg-emerald-500/10">
                    {{ $t('common.active', 'Active') }}
                  </Badge>
                  <Badge v-else variant="outline" class="border-muted-foreground text-muted-foreground">
                    {{ $t('common.inactive', 'Inactive') }}
                  </Badge>
                </template>
                <template #cell-created="{ item: catalog }">
                  <span class="text-muted-foreground text-xs">{{ formatDate(catalog.created_at) }}</span>
                </template>
                <template #cell-actions="{ item: catalog }">
                  <div class="flex items-center justify-end gap-1">
                    <RouterLink :to="`/settings/catalogs/${catalog.id}`">
                      <IconButton :icon="Pencil" :label="$t('common.open', 'Open')" class="h-8 w-8" />
                    </RouterLink>
                    <IconButton :label="$t('common.delete', 'Delete')" class="h-8 w-8" @click="openDeleteDialog(catalog)">
                      <Trash2 class="h-4 w-4 text-destructive" />
                    </IconButton>
                  </div>
                </template>
                <template #empty-action>
                  <Button variant="outline" size="sm" @click="openCreateDialog">
                    <Plus class="h-4 w-4 mr-2" />{{ $t('catalogs.addCatalog', 'Add Catalog') }}
                  </Button>
                </template>
              </DataTable>
            </CardContent>
          </Card>
        </div>
      </div>
    </ScrollArea>

    <CrudFormDialog
      v-model:open="isDialogOpen"
      :is-editing="false"
      :is-submitting="isSubmitting"
      :create-title="$t('catalogs.createTitle', 'Create Catalog')"
      :create-description="$t('catalogs.createDesc', 'Creates a catalog in Meta and links it to a WhatsApp account.')"
      max-width="max-w-md"
      @submit="saveCatalog"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <Label>{{ $t('catalogs.account', 'WhatsApp Account') }} <span class="text-destructive">*</span></Label>
          <Select v-model="formData.whatsapp_account">
            <SelectTrigger><SelectValue :placeholder="$t('catalogs.selectAccount', 'Select account')" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="a in accounts" :key="a.id" :value="a.name">{{ a.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-2">
          <Label>{{ $t('catalogs.name', 'Catalog Name') }} <span class="text-destructive">*</span></Label>
          <Input v-model="formData.name" :placeholder="$t('catalogs.namePlaceholder', 'e.g. Spring Collection')" maxlength="255" />
        </div>
      </div>
    </CrudFormDialog>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('catalogs.deleteCatalog', 'Delete Catalog')"
      :item-name="catalogToDelete?.name"
      :is-submitting="isDeleting"
      @confirm="confirmDelete"
    >
      <p class="text-sm text-muted-foreground">{{ $t('catalogs.deleteWarning', 'This deletes the catalog from Meta and removes all its products locally.') }}</p>
    </DeleteConfirmDialog>
  </div>
</template>
