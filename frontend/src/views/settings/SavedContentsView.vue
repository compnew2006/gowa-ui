<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  PageHeader,
  SearchInput,
  DeleteConfirmDialog,
  DataTable,
  type Column,
} from "@/components/shared";
import {
  savedContentsService,
  type SavedContent,
} from "@/services/api";
import ContentFormDialog from "@/components/ContentFormDialog.vue";
import { useCrudState } from "@/composables/useCrudState";
import { toast } from "vue-sonner";
import { Plus, Pencil, Trash2, Copy, Eye } from "lucide-vue-next";
import { getErrorMessage } from "@/lib/api-utils";
import { useDebounceFn } from "@vueuse/core";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";

const { t } = useI18n();

interface SavedContentFormData {
  name: string;
  body: string;
  category: string;
}

const defaultFormData: SavedContentFormData = {
  name: "",
  body: "",
  category: "",
};

const items = ref<SavedContent[]>([]);
const categories = ref<string[]>([]);
const isLoading = ref(false);
const {
  isSubmitting,
  isDialogOpen,
  editingItem,
  deleteDialogOpen,
  itemToDelete,
  openCreateDialog: baseOpenCreateDialog,
  openEditDialog: baseOpenEditDialog,
  openDeleteDialog,
  closeDialog,
  closeDeleteDialog,
} = useCrudState<SavedContent, SavedContentFormData>(defaultFormData);

const searchQuery = ref("");
const selectedCategory = ref("all");
const currentPage = ref(1);
const totalItems = ref(0);
const pageSize = 20;

const formDialog = ref<InstanceType<typeof ContentFormDialog> | null>(null);

// Media preview
const mediaBlobUrls = ref<Record<string, string>>({});
const previewingContent = ref<SavedContent | null>(null);
const showMediaPreview = ref(false);

const columns = computed<Column<SavedContent>[]>(() => [
  { key: "name", label: t("savedContents.name"), sortable: true },
  { key: "media", label: t("savedContents.media") },
  { key: "category", label: t("savedContents.category"), sortable: true },
  { key: "preview", label: t("savedContents.preview") },
  { key: "variables", label: t("savedContents.variables") },
  { key: "updated_at", label: t("savedContents.updated"), sortable: true },
  { key: "actions", label: t("common.actions"), align: "right" },
]);

async function loadMediaPreview(id: string) {
  if (mediaBlobUrls.value[id]) return;
  try {
    const response = await savedContentsService.getMedia(id);
    const blob = new Blob([response.data], {
      type: response.headers["content-type"],
    });
    mediaBlobUrls.value[id] = URL.createObjectURL(blob);
  } catch {
    // ignore
  }
}

function openMediaPreview(item: SavedContent) {
  previewingContent.value = item;
  showMediaPreview.value = true;
  if (item.id && !mediaBlobUrls.value[item.id]) {
    loadMediaPreview(item.id);
  }
}

async function fetchItems() {
  isLoading.value = true;
  try {
    const response = await savedContentsService.list({
      search: searchQuery.value || undefined,
      category:
        selectedCategory.value !== "all" ? selectedCategory.value : undefined,
      page: currentPage.value,
      limit: pageSize,
    });
    const data = (response.data as any).data || response.data;
    items.value = data.saved_contents || [];
    totalItems.value = data.total ?? items.value.length;
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedLoad", { resource: t("resources.savedContents") }),
      ),
    );
  } finally {
    isLoading.value = false;
  }
}

async function fetchCategories() {
  try {
    const response = await savedContentsService.categories();
    const data = (response.data as any).data || response.data;
    categories.value = data.categories || [];
  } catch {
    // ignore
  }
}

const debouncedSearch = useDebounceFn(() => {
  currentPage.value = 1;
  fetchItems();
}, 300);

watch(searchQuery, () => debouncedSearch());
watch(selectedCategory, () => {
  currentPage.value = 1;
  fetchItems();
});

function handlePageChange(page: number) {
  currentPage.value = page;
  fetchItems();
}

function openCreateDialog() {
  baseOpenCreateDialog();
  formDialog.value?.openForCreate();
}

function openEditDialog(item: SavedContent) {
  baseOpenEditDialog(item, (i) => ({
    name: i.name,
    body: i.body,
    category: i.category || "",
  }));
  formDialog.value?.openForEdit(item);
}

function handleFormClosed() {
  closeDialog();
}

async function handleFormSaved(payload: { name: string; body: string; category: string; mediaFile: File | null }) {
  isSubmitting.value = true;
  try {
    if (editingItem.value) {
      const id = editingItem.value.id;
      await savedContentsService.update(id, {
        name: payload.name,
        body: payload.body,
        category: payload.category,
      });
      if (payload.mediaFile) {
        await savedContentsService.uploadMedia(id, payload.mediaFile);
        if (mediaBlobUrls.value[id]) {
          URL.revokeObjectURL(mediaBlobUrls.value[id]);
          delete mediaBlobUrls.value[id];
        }
      }
      toast.success(
        t("common.updatedSuccess", { resource: t("resources.SavedContent") }),
      );
    } else {
      const response = await savedContentsService.create({
        name: payload.name,
        body: payload.body,
        category: payload.category,
      });
      const created = (response.data as any).data || response.data;
      const newId = created.id;
      if (payload.mediaFile && newId) {
        await savedContentsService.uploadMedia(newId, payload.mediaFile);
      }
      toast.success(
        t("common.createdSuccess", { resource: t("resources.SavedContent") }),
      );
    }
    isDialogOpen.value = false;
    formDialog.value?.resetForm();
    await Promise.all([fetchItems(), fetchCategories()]);
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedSave", { resource: t("resources.savedContent") }),
      ),
    );
  } finally {
    isSubmitting.value = false;
  }
}

async function confirmDelete() {
  if (!itemToDelete.value) return;
  try {
    await savedContentsService.delete(itemToDelete.value.id);
    toast.success(
      t("common.deletedSuccess", { resource: t("resources.SavedContent") }),
    );
    closeDeleteDialog();
    await Promise.all([fetchItems(), fetchCategories()]);
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedDelete", { resource: t("resources.savedContent") }),
      ),
    );
  }
}

function copyToClipboard(content: string) {
  navigator.clipboard.writeText(content);
  toast.success(t("common.copiedToClipboard"));
}

onMounted(() => {
  fetchItems();
  fetchCategories();
});

onUnmounted(() => {
  for (const url of Object.values(mediaBlobUrls.value)) {
    URL.revokeObjectURL(url);
  }
});
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      :title="t('savedContents.title')"
      :subtitle="t('savedContents.subtitle')"
    >
      <template #actions>
        <Button @click="openCreateDialog">
          <Plus class="h-4 w-4 mr-2" />
          {{ t("savedContents.addContent") }}
        </Button>
      </template>
    </PageHeader>

    <Card>
      <CardHeader>
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle>{{ t("savedContents.yourContents") }}</CardTitle>
            <CardDescription>{{ t("savedContents.yourContentsDesc") }}</CardDescription>
          </div>
          <div class="flex gap-2">
            <SearchInput
              v-model="searchQuery"
              :placeholder="t('savedContents.searchContents')"
            />
            <select
              v-model="selectedCategory"
              class="rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="all">{{ t("savedContents.allCategories") }}</option>
              <option v-for="cat in categories" :key="cat" :value="cat">
                {{ cat }}
              </option>
            </select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <DataTable
          :columns="columns"
          :items="items"
          :is-loading="isLoading"
          :total-items="totalItems"
          :current-page="currentPage"
          :page-size="pageSize"
          :server-pagination="true"
          :empty-title="t('savedContents.noContentsFound')"
          :empty-description="t('savedContents.noContentsFoundDesc')"
          @page-change="handlePageChange"
        >
          <template #cell-name="{ item }">
            <span class="font-medium">{{ item.name }}</span>
          </template>

          <template #cell-media="{ item }">
            <div v-if="item.media_mime_type" class="flex items-center gap-2">
              <Button variant="outline" size="sm" @click="openMediaPreview(item)">
                <Eye class="h-3 w-3 mr-1" />
                {{ t("savedContents.viewMedia") }}
              </Button>
              <span class="text-xs text-muted-foreground truncate max-w-[100px]">
                {{ item.media_filename }}
              </span>
            </div>
            <span v-else class="text-muted-foreground text-sm">&mdash;</span>
          </template>

          <template #cell-category="{ item }">
            <Badge v-if="item.category" variant="outline">
              {{ item.category }}
            </Badge>
            <span v-else class="text-muted-foreground text-sm">
              {{ t("savedContents.uncategorized") }}
            </span>
          </template>

          <template #cell-preview="{ item }">
            <p class="text-sm text-muted-foreground truncate max-w-[300px]">
              {{ item.preview || item.body }}
            </p>
          </template>

          <template #cell-variables="{ item }">
            <div v-if="item.variables && item.variables.length > 0" class="flex flex-wrap gap-1">
              <Badge
                v-for="v in item.variables"
                :key="v"
                variant="secondary"
                class="text-xs font-mono"
              >
                {{ v }}
              </Badge>
            </div>
            <span v-else class="text-muted-foreground text-sm">—</span>
          </template>

          <template #cell-updated_at="{ item }">
            <span class="text-sm text-muted-foreground">
              {{ new Date(item.updated_at).toLocaleDateString() }}
            </span>
          </template>

          <template #cell-actions="{ item }">
            <div class="flex items-center gap-1 justify-end">
              <Button variant="ghost" size="icon" class="h-8 w-8" @click="copyToClipboard(item.body)">
                <Copy class="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" class="h-8 w-8" @click="openEditDialog(item)">
                <Pencil class="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" class="h-8 w-8 text-destructive" @click="openDeleteDialog(item)">
                <Trash2 class="h-4 w-4" />
              </Button>
            </div>
          </template>
        </DataTable>
      </CardContent>
    </Card>

    <ContentFormDialog
      ref="formDialog"
      v-model:open="isDialogOpen"
      :is-editing="!!editingItem"
      :categories="categories"
      @saved="handleFormSaved"
      @update:open="(val: boolean) => { if (!val) handleFormClosed(); }"
    />

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="t('savedContents.deleteTitle')"
      :item-name="itemToDelete?.name || ''"
      :description="t('savedContents.deleteDesc')"
      @confirm="confirmDelete"
    />

    <Dialog v-model:open="showMediaPreview">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{ t("savedContents.mediaPreview") }}</DialogTitle>
          <DialogDescription>
            {{ previewingContent?.media_filename || previewingContent?.name }}
          </DialogDescription>
        </DialogHeader>
        <div class="flex items-center justify-center py-4">
          <img
            v-if="
              previewingContent?.media_mime_type?.startsWith('image/') &&
              previewingContent?.id &&
              mediaBlobUrls[previewingContent.id]
            "
            :src="mediaBlobUrls[previewingContent.id]"
            :alt="previewingContent?.media_filename"
            class="max-w-full max-h-[60vh] object-contain rounded"
          />
          <video
            v-else-if="
              previewingContent?.media_mime_type?.startsWith('video/') &&
              previewingContent?.id &&
              mediaBlobUrls[previewingContent.id]
            "
            :src="mediaBlobUrls[previewingContent.id]"
            controls
            class="max-w-full max-h-[60vh] rounded"
          />
          <div v-else class="text-sm text-muted-foreground">
            {{ previewingContent?.media_filename }}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="showMediaPreview = false">
            {{ t("common.close") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
