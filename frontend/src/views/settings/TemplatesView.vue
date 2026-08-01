<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader, SearchInput, DataTable, CrudFormDialog, DeleteConfirmDialog, IconButton, ErrorState, type Column } from '@/components/shared'
import type { Template } from '@/services/api'
import { useTemplatesStore } from '@/stores/templates'
import { useAuthStore } from '@/stores/auth'
import { useCrudState } from '@/composables/useCrudState'
import { toast } from 'vue-sonner'
import { Plus, LayoutTemplate, Pencil, Trash2 } from 'lucide-vue-next'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate } from '@/lib/utils'
import { useSearchPagination } from '@/composables/useSearchPagination'

const { t } = useI18n()
const templatesStore = useTemplatesStore()
const authStore = useAuthStore()
const canWriteTemplates = computed(() => authStore.hasPermission('templates', 'write'))
const canDeleteTemplates = computed(() => authStore.hasPermission('templates', 'delete'))

interface TemplateFormData {
  name: string
  display_name: string
  language: string
  category: string
  whatsapp_account: string
  header_type: string
  header_content: string
  body_content: string
  footer_content: string
}

const defaultFormData: TemplateFormData = {
  name: '',
  display_name: '',
  language: 'en',
  category: 'MARKETING',
  whatsapp_account: '',
  header_type: 'NONE',
  header_content: '',
  body_content: '',
  footer_content: ''
}

const CATEGORIES = ['MARKETING', 'UTILITY', 'AUTHENTICATION']
const LANGUAGES = ['en', 'ar', 'es', 'fr', 'de', 'it', 'pt', 'hi', 'ur', 'id', 'tr']
const HEADER_TYPES = ['NONE', 'TEXT', 'IMAGE', 'DOCUMENT', 'VIDEO']

const templates = ref<Template[]>([])
const isLoading = ref(false)
const isDeleting = ref(false)
const error = ref(false)
const {
  isSubmitting, isDialogOpen, editingItem: editingTemplate, deleteDialogOpen, itemToDelete: templateToDelete,
  formData, openCreateDialog, openEditDialog: baseOpenEditDialog, openDeleteDialog, closeDialog, closeDeleteDialog,
} = useCrudState<Template, TemplateFormData>(defaultFormData)
const { searchQuery, currentPage, totalItems, pageSize, handlePageChange } = useSearchPagination({
  fetchFn: () => fetchTemplates(),
})

// Sorting state
const sortKey = ref('updated_at')
const sortDirection = ref<'asc' | 'desc'>('desc')

const columns = computed<Column<Template>[]>(() => [
  { key: 'name', label: t('templates.template'), sortable: true },
  { key: 'category', label: t('templates.category'), sortable: true },
  { key: 'language', label: t('templates.language'), sortable: true },
  { key: 'whatsapp_account', label: t('templates.account'), sortable: true },
  { key: 'updated_at', label: t('templates.updated'), sortable: true },
  { key: 'actions', label: t('common.actions'), align: 'right' },
])

function openEditDialog(template: Template) {
  baseOpenEditDialog(template, (t) => ({
    name: t.name,
    display_name: t.display_name || '',
    language: t.language || 'en',
    category: t.category || 'MARKETING',
    whatsapp_account: t.whatsapp_account || '',
    header_type: t.header_type || 'NONE',
    header_content: t.header_content || '',
    body_content: t.body_content || '',
    footer_content: t.footer_content || '',
  }))
}

async function fetchTemplates() {
  isLoading.value = true
  error.value = false
  try {
    const response = await templatesStore.fetchTemplates({
      search: searchQuery.value || undefined,
      page: currentPage.value,
      limit: pageSize
    })
    templates.value = response.templates
    totalItems.value = response.total
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedLoad', { resource: t('resources.templates') })))
    error.value = true
  } finally {
    isLoading.value = false
  }
}

onMounted(() => fetchTemplates())

async function saveTemplate() {
  if (!formData.value.name.trim()) {
    toast.error(t('templates.nameRequired'))
    return
  }
  if (!formData.value.body_content.trim()) {
    toast.error(t('templates.bodyRequired'))
    return
  }
  if (!formData.value.whatsapp_account.trim()) {
    toast.error(t('templates.accountRequired'))
    return
  }
  isSubmitting.value = true
  try {
    // Omit header_content unless header_type is TEXT (matches backend expectation)
    const payload: TemplateFormData = { ...formData.value }
    if (payload.header_type !== 'TEXT') {
      payload.header_content = ''
    }
    if (editingTemplate.value) {
      await templatesStore.updateTemplate(editingTemplate.value.id, payload)
      toast.success(t('common.updatedSuccess', { resource: t('resources.Template') }))
    } else {
      await templatesStore.createTemplate(payload)
      toast.success(t('common.createdSuccess', { resource: t('resources.Template') }))
    }
    closeDialog()
    await fetchTemplates()
  } catch (error) {
    toast.error(getErrorMessage(error, t('common.failedSave', { resource: t('resources.template') })))
  } finally {
    isSubmitting.value = false
  }
}

async function confirmDelete() {
  if (!templateToDelete.value) return
  isDeleting.value = true
  try {
    await templatesStore.deleteTemplate(templateToDelete.value.id)
    toast.success(t('common.deletedSuccess', { resource: t('resources.Template') }))
    closeDeleteDialog()
    await fetchTemplates()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.template') })))
  } finally {
    isDeleting.value = false
  }
}

const categoryBadgeClass = (category: string): string => {
  switch ((category || '').toUpperCase()) {
    case 'MARKETING': return 'bg-blue-500/15 text-blue-400'
    case 'UTILITY': return 'bg-green-500/15 text-green-400'
    case 'AUTHENTICATION': return 'bg-amber-500/15 text-amber-400'
    default: return 'bg-muted text-muted-foreground'
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader :title="$t('templates.title')" :subtitle="$t('templates.subtitle')" :icon="LayoutTemplate" icon-gradient="bg-gradient-to-br from-cyan-500 to-blue-600 shadow-cyan-500/20" back-link="/settings">
      <template v-if="canWriteTemplates" #actions>
        <Button variant="outline" size="sm" @click="openCreateDialog"><Plus class="h-4 w-4 mr-2" />{{ $t('templates.addTemplate') }}</Button>
      </template>
    </PageHeader>

    <!-- Error State -->
    <ErrorState
      v-if="error && !isLoading"
      :title="$t('common.loadErrorTitle')"
      :description="$t('common.loadErrorDescription')"
      :retry-label="$t('common.retryLoad')"
      class="flex-1"
      @retry="fetchTemplates"
    />

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <div class="max-w-6xl mx-auto">
          <Card>
            <CardHeader>
              <div class="flex items-center justify-between flex-wrap gap-4">
                <div>
                  <CardTitle>{{ $t('templates.organizationTemplates') }}</CardTitle>
                  <CardDescription>{{ $t('templates.organizationTemplatesDesc') }}</CardDescription>
                </div>
                <SearchInput v-model="searchQuery" :placeholder="$t('templates.searchTemplates') + '...'" class="w-64" />
              </div>
            </CardHeader>
            <CardContent>
              <DataTable
                :items="templates"
                :columns="columns"
                :is-loading="isLoading"
                :empty-icon="LayoutTemplate"
                :empty-title="searchQuery ? $t('templates.noMatchingTemplates') : $t('templates.noTemplatesYet')"
                :empty-description="searchQuery ? $t('templates.noMatchingTemplatesDesc') : $t('templates.noTemplatesYetDesc')"
                v-model:sort-key="sortKey"
                v-model:sort-direction="sortDirection"
                server-pagination
                :current-page="currentPage"
                :total-items="totalItems"
                :page-size="pageSize"
                item-name="templates"
                @page-change="handlePageChange"
              >
                <template #cell-name="{ item: template }">
                  <div class="flex flex-col">
                    <span class="font-medium">{{ template.display_name || template.name }}</span>
                    <span v-if="template.display_name" class="text-xs text-muted-foreground">{{ template.name }}</span>
                  </div>
                </template>
                <template #cell-category="{ item: template }">
                  <span :class="['inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium', categoryBadgeClass(template.category)]">
                    {{ (template.category || '—').toUpperCase() }}
                  </span>
                </template>
                <template #cell-language="{ item: template }">
                  <span class="text-muted-foreground">{{ (template.language || '—').toUpperCase() }}</span>
                </template>
                <template #cell-whatsapp_account="{ item: template }">
                  <span class="text-muted-foreground">{{ template.whatsapp_account || '—' }}</span>
                </template>
                <template #cell-updated_at="{ item: template }">
                  <span class="text-muted-foreground">{{ formatDate(template.updated_at || template.created_at) }}</span>
                </template>
                <template #cell-actions="{ item: template }">
                  <div class="flex items-center justify-end gap-1">
                    <IconButton v-if="canWriteTemplates" :icon="Pencil" :label="$t('templates.editTemplate')" class="h-8 w-8" @click="openEditDialog(template)" />
                    <IconButton v-if="canDeleteTemplates" :label="$t('templates.deleteTemplate')" class="h-8 w-8" @click="openDeleteDialog(template)">
                      <Trash2 class="h-4 w-4 text-destructive" />
                    </IconButton>
                  </div>
                </template>
                <template v-if="canWriteTemplates" #empty-action>
                  <Button variant="outline" size="sm" @click="openCreateDialog">
                    <Plus class="h-4 w-4 mr-2" />
                    {{ $t('templates.addTemplate') }}
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
      :is-editing="!!editingTemplate"
      :is-submitting="isSubmitting"
      :edit-title="$t('templates.editTemplateTitle')"
      :create-title="$t('templates.createTemplateTitle')"
      :edit-description="$t('templates.editTemplateDesc')"
      :create-description="$t('templates.createTemplateDesc')"
      max-width="max-w-2xl"
      @submit="saveTemplate"
    >
      <div class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('templates.name') }} <span class="text-destructive">*</span></Label>
            <Input v-model="formData.name" :placeholder="$t('templates.namePlaceholder')" maxlength="100" />
            <p class="text-xs text-muted-foreground">{{ $t('templates.nameHint') }}</p>
          </div>
          <div class="space-y-2">
            <Label>{{ $t('templates.displayName') }}</Label>
            <Input v-model="formData.display_name" :placeholder="$t('templates.displayNamePlaceholder')" maxlength="100" />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('templates.category') }} <span class="text-destructive">*</span></Label>
            <Select v-model="formData.category" :default-value="formData.category">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="cat in CATEGORIES" :key="cat" :value="cat">{{ cat }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>{{ $t('templates.language') }} <span class="text-destructive">*</span></Label>
            <Select v-model="formData.language" :default-value="formData.language">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="lang in LANGUAGES" :key="lang" :value="lang">{{ lang.toUpperCase() }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>{{ $t('templates.account') }} <span class="text-destructive">*</span></Label>
            <Input v-model="formData.whatsapp_account" :placeholder="$t('templates.accountPlaceholder')" maxlength="100" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('templates.headerType') }}</Label>
            <Select v-model="formData.header_type" :default-value="formData.header_type">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="ht in HEADER_TYPES" :key="ht" :value="ht">{{ ht === 'NONE' ? $t('templates.none') : ht }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div v-if="formData.header_type === 'TEXT'" class="space-y-2">
            <Label>{{ $t('templates.headerContent') }}</Label>
            <Input v-model="formData.header_content" :placeholder="$t('templates.headerContentPlaceholder')" maxlength="60" />
          </div>
        </div>

        <div class="space-y-2">
          <Label>{{ $t('templates.bodyContent') }} <span class="text-destructive">*</span></Label>
          <Textarea v-model="formData.body_content" :placeholder="$t('templates.bodyPlaceholder')" :rows="4" :maxlength="1024" />
          <p class="text-xs text-muted-foreground">{{ $t('templates.bodyHint') }}</p>
        </div>

        <div class="space-y-2">
          <Label>{{ $t('templates.footerContent') }}</Label>
          <Input v-model="formData.footer_content" :placeholder="$t('templates.footerPlaceholder')" maxlength="60" />
        </div>
      </div>
    </CrudFormDialog>

    <DeleteConfirmDialog v-model:open="deleteDialogOpen" :title="$t('templates.deleteTemplate')" :item-name="templateToDelete?.display_name || templateToDelete?.name" :is-submitting="isDeleting" @confirm="confirmDelete">
      <p class="text-sm text-muted-foreground">{{ $t('templates.deleteWarning') }}</p>
    </DeleteConfirmDialog>
  </div>
</template>
