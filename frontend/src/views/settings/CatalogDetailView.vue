<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  DetailPageLayout, DataTable, CrudFormDialog, DeleteConfirmDialog, IconButton, type Column
} from '@/components/shared'
import {
  catalogsService, productsService, type Catalog, type CatalogProduct
} from '@/services/api'
import { useCrudState } from '@/composables/useCrudState'
import { toast } from 'vue-sonner'
import { ShoppingCart, Plus, Pencil, Trash2, Package, Loader2 } from 'lucide-vue-next'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate, formatPrice } from '@/lib/utils'

const { t } = useI18n()
const route = useRoute()
const catalogId = computed(() => route.params.id as string)

const catalog = ref<Catalog | null>(null)
const products = ref<CatalogProduct[]>([])
const isLoading = ref(true)
const isNotFound = ref(false)
const isDeletingProduct = ref(false)

interface ProductFormData {
  name: string
  description: string
  price: number | undefined
  currency: string
  url: string
  image_url: string
  retailer_id: string
}
const defaultProductForm: ProductFormData = {
  name: '', description: '', price: undefined, currency: 'USD', url: '', image_url: '', retailer_id: ''
}

const {
  isSubmitting, isDialogOpen, editingItem: editingProduct, deleteDialogOpen,
  itemToDelete: productToDelete, formData, openCreateDialog,
  openEditDialog: baseOpenEditDialog, openDeleteDialog, closeDialog, closeDeleteDialog,
} = useCrudState<CatalogProduct, ProductFormData>(defaultProductForm)

const sortKey = ref('name')
const sortDirection = ref<'asc' | 'desc'>('asc')

const columns = computed<Column<CatalogProduct>[]>(() => [
  { key: 'name', label: t('products.name', 'Product'), sortable: true },
  { key: 'price', label: t('products.price', 'Price'), sortable: true },
  { key: 'retailer_id', label: t('products.sku', 'SKU') },
  { key: 'status', label: t('products.status', 'Status') },
  { key: 'updated', label: t('common.updated', 'Updated'), sortable: true, sortKey: 'updated_at' },
  { key: 'actions', label: t('common.actions', 'Actions'), align: 'right' },
])

const breadcrumbs = computed(() => [
  { label: t('nav.settings', 'Settings'), href: '/settings' },
  { label: t('catalogs.title', 'Catalogs'), href: '/settings/catalogs' },
  { label: catalog.value?.name || '' },
])

async function loadCatalog() {
  isLoading.value = true
  isNotFound.value = false
  try {
    // GetCatalog returns the catalog with its products preloaded.
    const res = await catalogsService.get(catalogId.value)
    const data = res.data
    catalog.value = data
    products.value = data.products || []
  } catch {
    isNotFound.value = true
  } finally {
    isLoading.value = false
  }
}

onMounted(loadCatalog)

function openEditProduct(p: CatalogProduct) {
  baseOpenEditDialog(p, (x) => ({
    name: x.name, description: x.description, price: x.price, currency: x.currency || 'USD',
    url: x.url, image_url: x.image_url, retailer_id: x.retailer_id,
  }))
}

async function saveProduct() {
  if (!formData.value.name.trim() || !formData.value.price || formData.value.price <= 0) {
    toast.error(t('products.namePriceRequired', 'Name and price are required'))
    return
  }
  isSubmitting.value = true
  try {
    const payload = {
      name: formData.value.name.trim(),
      description: formData.value.description,
      price: formData.value.price,
      currency: formData.value.currency || 'USD',
      url: formData.value.url,
      image_url: formData.value.image_url,
      retailer_id: formData.value.retailer_id,
    }
    if (editingProduct.value) {
      await productsService.update(editingProduct.value.id, payload)
      toast.success(t('common.updatedSuccess', { resource: t('resources.Product', 'Product') }))
    } else {
      await catalogsService.createProduct(catalogId.value, payload)
      toast.success(t('common.createdSuccess', { resource: t('resources.Product', 'Product') }))
    }
    closeDialog()
    await loadCatalog()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedSave', { resource: t('resources.product', 'product') })))
  } finally {
    isSubmitting.value = false
  }
}

async function confirmDeleteProduct() {
  if (!productToDelete.value) return
  isDeletingProduct.value = true
  try {
    await productsService.delete(productToDelete.value.id)
    toast.success(t('common.deletedSuccess', { resource: t('resources.Product', 'Product') }))
    closeDeleteDialog()
    await loadCatalog()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.product', 'product') })))
  } finally {
    isDeletingProduct.value = false
  }
}
</script>

<template>
  <DetailPageLayout
    :title="catalog?.name || ''"
    :description="catalog ? $t('catalogs.detailDesc', { account: catalog.whatsapp_account }) : ''"
    :icon="ShoppingCart"
    icon-gradient="bg-gradient-to-br from-amber-500 to-orange-600 shadow-amber-500/20"
    back-link="/settings/catalogs"
    :breadcrumbs="breadcrumbs"
    :is-loading="isLoading"
    :is-not-found="isNotFound"
    :not-found-title="$t('catalogs.notFound', 'Catalog not found')"
    :not-found-description="$t('catalogs.notFoundDesc', 'It may have been deleted.')"
  >
    <!-- Products table (main slot) -->
    <Card>
      <CardHeader>
        <div class="flex items-center justify-between flex-wrap gap-4">
          <div>
            <CardTitle class="flex items-center gap-2">
              <Package class="h-5 w-5 text-amber-500" />
              {{ $t('products.title', 'Products') }}
              <Badge variant="secondary" class="ml-1">{{ products.length }}</Badge>
            </CardTitle>
            <CardDescription>{{ $t('products.subtitle', 'Products in this catalog. Changes sync to Meta.') }}</CardDescription>
          </div>
          <Button size="sm" @click="openCreateDialog">
            <Plus class="h-4 w-4 mr-2" />{{ $t('products.addProduct', 'Add Product') }}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <DataTable
          :items="products"
          :columns="columns"
          :is-loading="isLoading"
          :empty-icon="Package"
          :empty-title="$t('products.noProducts', 'No products yet')"
          :empty-description="$t('products.noProductsDesc', 'Add a product to this catalog to get started.')"
          v-model:sort-key="sortKey"
          v-model:sort-direction="sortDirection"
          item-name="products"
        >
          <template #cell-name="{ item: p }">
            <div class="flex items-center gap-3">
              <img v-if="p.image_url" :src="p.image_url" :alt="p.name" class="h-9 w-9 rounded object-cover flex-shrink-0" />
              <div v-else class="h-9 w-9 rounded bg-muted flex items-center justify-center flex-shrink-0">
                <Package class="h-4 w-4 text-muted-foreground" />
              </div>
              <div class="flex flex-col gap-0.5 min-w-0">
                <span class="font-medium truncate text-sm">{{ p.name }}</span>
                <span v-if="p.description" class="text-xs text-muted-foreground truncate">{{ p.description }}</span>
              </div>
            </div>
          </template>
          <template #cell-price="{ item: p }">
            <span class="font-medium">{{ formatPrice(p.price, p.currency) }}</span>
          </template>
          <template #cell-retailer_id="{ item: p }">
            <code v-if="p.retailer_id" class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">{{ p.retailer_id }}</code>
            <span v-else class="text-muted-foreground text-xs">—</span>
          </template>
          <template #cell-status="{ item: p }">
            <Badge v-if="p.is_active" variant="outline" class="border-emerald-600 text-emerald-600 bg-emerald-500/10">
              {{ $t('common.active', 'Active') }}
            </Badge>
            <Badge v-else variant="outline" class="border-muted-foreground text-muted-foreground">
              {{ $t('common.inactive', 'Inactive') }}
            </Badge>
          </template>
          <template #cell-updated="{ item: p }">
            <span class="text-muted-foreground text-xs">{{ formatDate(p.updated_at) }}</span>
          </template>
          <template #cell-actions="{ item: p }">
            <div class="flex items-center justify-end gap-1">
              <IconButton :icon="Pencil" :label="$t('common.edit', 'Edit')" class="h-8 w-8" @click="openEditProduct(p)" />
              <IconButton :label="$t('common.delete', 'Delete')" class="h-8 w-8" @click="openDeleteDialog(p)">
                <Trash2 class="h-4 w-4 text-destructive" />
              </IconButton>
            </div>
          </template>
          <template #empty-action>
            <Button variant="outline" size="sm" @click="openCreateDialog">
              <Plus class="h-4 w-4 mr-2" />{{ $t('products.addProduct', 'Add Product') }}
            </Button>
          </template>
        </DataTable>
      </CardContent>
    </Card>

    <!-- Sidebar: catalog metadata -->
    <template #sidebar>
      <Card>
        <CardHeader>
          <CardTitle class="text-sm">{{ $t('catalogs.metadata', 'Catalog Details') }}</CardTitle>
        </CardHeader>
        <CardContent class="space-y-3 text-sm">
          <div v-if="catalog">
            <div class="flex justify-between"><span class="text-muted-foreground">{{ $t('catalogs.account', 'Account') }}</span><code class="text-xs bg-muted px-1.5 py-0.5 rounded">{{ catalog.whatsapp_account }}</code></div>
            <div class="flex justify-between"><span class="text-muted-foreground">{{ $t('catalogs.metaId', 'Meta ID') }}</span><code class="text-xs bg-muted px-1.5 py-0.5 rounded">{{ catalog.meta_catalog_id?.slice(0, 12) || '—' }}</code></div>
            <div class="flex justify-between"><span class="text-muted-foreground">{{ $t('common.created', 'Created') }}</span><span>{{ formatDate(catalog.created_at) }}</span></div>
          </div>
          <div v-else class="flex items-center gap-2 text-muted-foreground">
            <Loader2 class="h-4 w-4 animate-spin" />
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- Product create/edit dialog -->
    <CrudFormDialog
      v-model:open="isDialogOpen"
      :is-editing="!!editingProduct"
      :is-submitting="isSubmitting"
      :create-title="$t('products.createTitle', 'Add Product')"
      :edit-title="$t('products.editTitle', 'Edit Product')"
      :create-description="$t('products.createDesc', 'Creates the product in Meta and adds it to this catalog.')"
      max-width="max-w-lg"
      @submit="saveProduct"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <Label>{{ $t('products.name', 'Name') }} <span class="text-destructive">*</span></Label>
          <Input v-model="formData.name" :placeholder="$t('products.namePlaceholder', 'Product name')" maxlength="255" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('products.price', 'Price (cents)') }} <span class="text-destructive">*</span></Label>
            <Input v-model.number="formData.price" type="number" min="1" :placeholder="$t('products.pricePlaceholder', 'e.g. 1999 = $19.99')" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('products.currency', 'Currency') }}</Label>
            <Select v-model="formData.currency">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="USD">USD</SelectItem>
                <SelectItem value="EUR">EUR</SelectItem>
                <SelectItem value="GBP">GBP</SelectItem>
                <SelectItem value="SAR">SAR</SelectItem>
                <SelectItem value="AED">AED</SelectItem>
                <SelectItem value="EGP">EGP</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div class="space-y-2">
          <Label>{{ $t('products.description', 'Description') }}</Label>
          <Textarea v-model="formData.description" :placeholder="$t('products.descriptionPlaceholder', 'Short description')" :rows="3" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('products.sku', 'Retailer ID (SKU)') }}</Label>
            <Input v-model="formData.retailer_id" :placeholder="$t('products.skuPlaceholder', 'SKU')" maxlength="100" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('products.imageUrl', 'Image URL') }}</Label>
            <Input v-model="formData.image_url" :placeholder="t('products.imageUrlPlaceholder', 'https://...')" />
          </div>
        </div>
        <div class="space-y-2">
          <Label>{{ $t('products.url', 'Product URL') }}</Label>
          <Input v-model="formData.url" :placeholder="$t('products.urlPlaceholder', 'https://...')" />
        </div>
      </div>
    </CrudFormDialog>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('products.deleteProduct', 'Delete Product')"
      :item-name="productToDelete?.name"
      :is-submitting="isDeletingProduct"
      @confirm="confirmDeleteProduct"
    >
      <p class="text-sm text-muted-foreground">{{ $t('products.deleteWarning', 'This deletes the product from Meta and locally.') }}</p>
    </DeleteConfirmDialog>
  </DetailPageLayout>
</template>
