<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  PageHeader,
  SearchInput,
  CrudFormDialog,
  DeleteConfirmDialog,
  DataTable,
  type Column,
} from "@/components/shared";
import {
  cannedResponsesService,
  type CannedResponse,
  type CannedResponseAttachment,
} from "@/services/api";
import { useCrudState } from "@/composables/useCrudState";
import { toast } from "vue-sonner";
import {
  Plus,
  MessageSquareText,
  Pencil,
  Trash2,
  Copy,
  Image as ImageIcon,
  Video as VideoIcon,
  X,
} from "lucide-vue-next";
import { getErrorMessage } from "@/lib/api-utils";
import { CANNED_RESPONSE_CATEGORIES, getLabelFromValue } from "@/lib/constants";
import { useDebounceFn } from "@vueuse/core";
import WhatsAppRichTextEditor from "@/components/chat/WhatsAppRichTextEditor.vue";
import { useAuthStore } from "@/stores/auth";

const { t } = useI18n();
const authStore = useAuthStore();
const canWriteResponses = computed(() => authStore.hasPermission("canned_responses", "write"));
const canDeleteResponses = computed(() => authStore.hasPermission("canned_responses", "delete"));

interface CannedResponseFormData {
  name: string;
  shortcut: string;
  content: string;
  category: string;
  is_active: boolean;
}

const defaultFormData: CannedResponseFormData = {
  name: "",
  shortcut: "",
  content: "",
  category: "",
  is_active: true,
};

const cannedResponses = ref<CannedResponse[]>([]);
const existingAttachments = ref<CannedResponseAttachment[]>([]);
const newAttachmentFiles = ref<File[]>([]);
const attachmentInputRef = ref<HTMLInputElement | null>(null);
const isLoading = ref(false);
const {
  isSubmitting,
  isDialogOpen,
  editingItem: editingResponse,
  deleteDialogOpen,
  itemToDelete: responseToDelete,
  formData,
  openCreateDialog: baseOpenCreateDialog,
  openEditDialog: baseOpenEditDialog,
  openDeleteDialog,
  closeDialog,
  closeDeleteDialog,
} = useCrudState<CannedResponse, CannedResponseFormData>(defaultFormData);
const searchQuery = ref("");
const selectedCategory = ref("all");

// Pagination state
const currentPage = ref(1);
const totalItems = ref(0);
const pageSize = 20;

const columns = computed<Column<CannedResponse>[]>(() => [
  { key: "name", label: t("cannedResponses.name"), sortable: true },
  { key: "category", label: t("cannedResponses.category"), sortable: true },
  { key: "content", label: t("cannedResponses.content") },
  { key: "attachments", label: t("cannedResponses.media") },
  { key: "usage_count", label: t("cannedResponses.used"), sortable: true },
  {
    key: "status",
    label: t("cannedResponses.status"),
    sortable: true,
    sortKey: "is_active",
  },
  { key: "actions", label: t("common.actions"), align: "right" },
]);

const sortKey = ref("name");
const sortDirection = ref<"asc" | "desc">("asc");

async function fetchItems() {
  isLoading.value = true;
  try {
    const response = await cannedResponsesService.list({
      search: searchQuery.value || undefined,
      category:
        selectedCategory.value !== "all" ? selectedCategory.value : undefined,
      page: currentPage.value,
      limit: pageSize,
    });
    const data = (response.data as any).data || response.data;
    cannedResponses.value = data.canned_responses || [];
    totalItems.value = data.total ?? cannedResponses.value.length;
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedLoad", { resource: t("resources.cannedResponses") }),
      ),
    );
  } finally {
    isLoading.value = false;
  }
}

// Debounced search
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
  existingAttachments.value = [];
  newAttachmentFiles.value = [];
  baseOpenCreateDialog();
}

function openEditDialog(response: CannedResponse) {
  existingAttachments.value = [...(response.attachments || [])];
  newAttachmentFiles.value = [];
  baseOpenEditDialog(response, (r) => ({
    name: r.name,
    shortcut: r.shortcut || "",
    content: r.content,
    category: r.category || "",
    is_active: r.is_active,
  }));
}

onMounted(() => fetchItems());

function buildMultipartPayload() {
  const payload = new FormData();
  payload.append("name", formData.value.name.trim());
  payload.append("shortcut", formData.value.shortcut.trim());
  payload.append("content", formData.value.content);
  payload.append("category", formData.value.category || "");
  payload.append("is_active", String(formData.value.is_active));
  payload.append(
    "keep_attachment_ids",
    JSON.stringify(
      existingAttachments.value.map((attachment) => attachment.id),
    ),
  );
  for (const file of newAttachmentFiles.value) {
    payload.append("attachments", file);
  }
  return payload;
}

async function saveResponse() {
  const hasAnyAttachment =
    existingAttachments.value.length > 0 || newAttachmentFiles.value.length > 0;
  if (
    !formData.value.name.trim() ||
    (!formData.value.content.trim() && !hasAnyAttachment)
  ) {
    toast.error(t("cannedResponses.nameContentRequired"));
    return;
  }

  isSubmitting.value = true;
  try {
    const payload = buildMultipartPayload();
    if (editingResponse.value) {
      await cannedResponsesService.update(editingResponse.value.id, payload);
      toast.success(
        t("common.updatedSuccess", { resource: t("resources.CannedResponse") }),
      );
    } else {
      await cannedResponsesService.create(payload);
      toast.success(
        t("common.createdSuccess", { resource: t("resources.CannedResponse") }),
      );
    }
    existingAttachments.value = [];
    newAttachmentFiles.value = [];
    closeDialog();
    await fetchItems();
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedSave", { resource: t("resources.cannedResponse") }),
      ),
    );
  } finally {
    isSubmitting.value = false;
  }
}

async function confirmDelete() {
  if (!responseToDelete.value) return;
  try {
    await cannedResponsesService.delete(responseToDelete.value.id);
    toast.success(
      t("common.deletedSuccess", { resource: t("resources.CannedResponse") }),
    );
    closeDeleteDialog();
    await fetchItems();
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedDelete", { resource: t("resources.cannedResponse") }),
      ),
    );
  }
}

function copyToClipboard(content: string) {
  navigator.clipboard.writeText(content);
  toast.success(t("common.copiedToClipboard"));
}
function getCategoryLabel(category: string): string {
  return (
    getLabelFromValue(CANNED_RESPONSE_CATEGORIES, category) ||
    t("cannedResponses.uncategorized")
  );
}

function formatFileSize(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function getAttachmentFileTypeIcon(mimeType: string) {
  return mimeType.startsWith("video/") ? VideoIcon : ImageIcon;
}

function openAttachmentPicker() {
  attachmentInputRef.value?.click();
}

function removeExistingAttachment(attachmentID: string) {
  existingAttachments.value = existingAttachments.value.filter(
    (attachment) => attachment.id !== attachmentID,
  );
}

function removeNewAttachment(index: number) {
  newAttachmentFiles.value = newAttachmentFiles.value.filter(
    (_, currentIndex) => currentIndex !== index,
  );
}

function handleAttachmentSelection(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files || []);
  if (files.length === 0) return;

  for (const file of files) {
    if (!file.type.startsWith("image/") && !file.type.startsWith("video/")) {
      toast.error(t("cannedResponses.invalidMediaType"));
      continue;
    }
    if (file.size > 16 * 1024 * 1024) {
      toast.error(t("cannedResponses.mediaTooLarge"));
      continue;
    }

    const alreadyAdded = newAttachmentFiles.value.some(
      (existing) => existing.name === file.name && existing.size === file.size,
    );
    if (!alreadyAdded) {
      newAttachmentFiles.value.push(file);
    }
  }

  input.value = "";
}
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('cannedResponses.title')"
      :subtitle="$t('cannedResponses.subtitle')"
      :icon="MessageSquareText"
      icon-gradient="bg-gradient-to-br from-blue-500 to-sky-600 shadow-blue-500/20"
    >
      <template #actions>
        <Button v-if="canWriteResponses" variant="outline" size="sm" @click="openCreateDialog"
          ><Plus class="h-4 w-4 mr-2" />{{
            $t("cannedResponses.addResponse")
          }}</Button
        >
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="p-6">
        <div class="max-w-6xl mx-auto">
          <Card>
            <CardHeader>
              <div class="flex items-center justify-between flex-wrap gap-4">
                <div>
                  <CardTitle>{{
                    $t("cannedResponses.yourResponses")
                  }}</CardTitle>
                  <CardDescription>{{
                    $t("cannedResponses.yourResponsesDesc")
                  }}</CardDescription>
                </div>
                <div class="flex items-center gap-2">
                  <Select v-model="selectedCategory">
                    <SelectTrigger class="w-[150px]"
                      ><SelectValue :placeholder="$t('common.all')"
                    /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{{
                        $t("cannedResponses.allCategories")
                      }}</SelectItem>
                      <SelectItem
                        v-for="cat in CANNED_RESPONSE_CATEGORIES"
                        :key="cat.value"
                        :value="cat.value"
                        >{{ cat.label }}</SelectItem
                      >
                    </SelectContent>
                  </Select>
                  <SearchInput
                    v-model="searchQuery"
                    :placeholder="$t('cannedResponses.searchResponses') + '...'"
                    class="w-64"
                  />
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <DataTable
                :items="cannedResponses"
                :columns="columns"
                :is-loading="isLoading"
                :empty-icon="MessageSquareText"
                :empty-title="$t('cannedResponses.noResponsesFound')"
                :empty-description="$t('cannedResponses.noResponsesFoundDesc')"
                v-model:sort-key="sortKey"
                v-model:sort-direction="sortDirection"
                server-pagination
                :current-page="currentPage"
                :total-items="totalItems"
                :page-size="pageSize"
                item-name="responses"
                @page-change="handlePageChange"
              >
                <template #cell-name="{ item: response }">
                  <div>
                    <span class="font-medium">{{ response.name }}</span>
                    <p
                      v-if="response.shortcut"
                      class="text-xs font-mono text-muted-foreground"
                    >
                      /{{ response.shortcut }}
                    </p>
                  </div>
                </template>
                <template #cell-category="{ item: response }">
                  <Badge variant="outline" class="text-xs">{{
                    getCategoryLabel(response.category)
                  }}</Badge>
                </template>
                <template #cell-content="{ item: response }">
                  <div class="max-w-[300px]">
                    <p class="text-sm text-muted-foreground truncate">
                      {{ response.content }}
                    </p>
                  </div>
                </template>
                <template #cell-attachments="{ item: response }">
                  <span class="text-xs text-muted-foreground">
                    {{ response.attachments?.length || 0 }}
                  </span>
                </template>
                <template #cell-usage_count="{ item: response }">
                  <span class="text-muted-foreground">{{
                    response.usage_count
                  }}</span>
                </template>
                <template #cell-status="{ item: response }">
                  <Badge
                    v-if="response.is_active"
                    class="bg-primary/10 text-primary border-transparent text-xs"
                    >{{ $t("common.active") }}</Badge
                  >
                  <Badge v-else variant="secondary" class="text-xs">{{
                    $t("common.inactive")
                  }}</Badge>
                </template>
                <template #cell-actions="{ item: response }">
                  <div class="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8"
                      @click="copyToClipboard(response.content)"
                      title="Copy"
                    >
                      <Copy class="h-4 w-4" />
                    </Button>
                    <Button
                      v-if="canWriteResponses"
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8"
                      @click="openEditDialog(response)"
                      title="Edit"
                    >
                      <Pencil class="h-4 w-4" />
                    </Button>
                    <Button
                      v-if="canDeleteResponses"
                      variant="ghost"
                      size="icon"
                      class="h-8 w-8 text-destructive"
                      @click="openDeleteDialog(response)"
                      title="Delete"
                    >
                      <Trash2 class="h-4 w-4" />
                    </Button>
                  </div>
                </template>
                <template #empty-action>
                  <Button v-if="canWriteResponses" variant="outline" size="sm" @click="openCreateDialog">
                    <Plus class="h-4 w-4 mr-2" />{{
                      $t("cannedResponses.addResponse")
                    }}
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
      :is-editing="!!editingResponse"
      :is-submitting="isSubmitting"
      :edit-title="$t('cannedResponses.editTitle')"
      :create-title="$t('cannedResponses.createTitle')"
      :edit-description="$t('cannedResponses.editDesc')"
      :create-description="$t('cannedResponses.createDesc')"
      max-width="max-w-lg"
      @submit="saveResponse"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <Label
            >{{ $t("cannedResponses.name") }}
            <span class="text-destructive">*</span></Label
          ><Input v-model="formData.name" placeholder="Welcome Message" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label>{{ $t("cannedResponses.shortcut") }}</Label>
            <div class="relative">
              <span
                class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                >/</span
              ><Input
                v-model="formData.shortcut"
                placeholder="welcome"
                class="pl-7"
              />
            </div>
            <p class="text-xs text-muted-foreground">
              {{ $t("cannedResponses.shortcutHint") }}
            </p>
          </div>
          <div class="space-y-2">
            <Label>{{ $t("cannedResponses.category") }}</Label>
            <Select v-model="formData.category"
              ><SelectTrigger
                ><SelectValue
                  :placeholder="
                    $t('cannedResponses.category')
                  " /></SelectTrigger
              ><SelectContent
                ><SelectItem
                  v-for="cat in CANNED_RESPONSE_CATEGORIES"
                  :key="cat.value"
                  :value="cat.value"
                  >{{ cat.label }}</SelectItem
                ></SelectContent
              ></Select
            >
          </div>
        </div>
        <div class="space-y-2">
          <Label
            >{{ $t("cannedResponses.content") }}
            <span class="text-destructive">*</span></Label
          >
          <WhatsAppRichTextEditor
            v-model="formData.content"
            :placeholder="$t('cannedResponses.contentPlaceholder')"
            :rows="6"
          />
          <p class="text-xs text-muted-foreground">
            {{ $t("cannedResponses.placeholderHint") }}
          </p>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <Label>{{ $t("cannedResponses.attachments") }}</Label>
            <Button
              type="button"
              variant="outline"
              size="sm"
              @click="openAttachmentPicker"
            >
              <Plus class="h-3.5 w-3.5 mr-1.5" />
              {{ $t("cannedResponses.addMedia") }}
            </Button>
          </div>
          <input
            ref="attachmentInputRef"
            type="file"
            accept="image/*,video/*"
            multiple
            class="hidden"
            @change="handleAttachmentSelection"
          />
          <p class="text-xs text-muted-foreground">
            {{ $t("cannedResponses.attachmentsHint") }}
          </p>
          <div
            v-if="
              existingAttachments.length > 0 || newAttachmentFiles.length > 0
            "
            class="space-y-2 rounded-md border p-2"
          >
            <div
              v-for="attachment in existingAttachments"
              :key="attachment.id"
              class="flex items-center gap-2 rounded-md bg-muted/40 px-2 py-1.5"
            >
              <component
                :is="getAttachmentFileTypeIcon(attachment.mime_type)"
                class="h-4 w-4 text-muted-foreground"
              />
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm">{{ attachment.file_name }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ formatFileSize(attachment.file_size) }}
                </p>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="h-7 w-7"
                @click="removeExistingAttachment(attachment.id)"
              >
                <X class="h-4 w-4" />
              </Button>
            </div>
            <div
              v-for="(file, index) in newAttachmentFiles"
              :key="`${file.name}-${file.size}-${index}`"
              class="flex items-center gap-2 rounded-md bg-muted/40 px-2 py-1.5"
            >
              <component
                :is="getAttachmentFileTypeIcon(file.type)"
                class="h-4 w-4 text-muted-foreground"
              />
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm">{{ file.name }}</p>
                <p class="text-xs text-muted-foreground">
                  {{ formatFileSize(file.size) }}
                </p>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="h-7 w-7"
                @click="removeNewAttachment(index)"
              >
                <X class="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
        <div v-if="editingResponse" class="flex items-center justify-between">
          <Label>{{ $t("common.active") }}</Label
          ><Switch v-model:checked="formData.is_active" />
        </div>
      </div>
    </CrudFormDialog>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('cannedResponses.deleteTitle')"
      :item-name="responseToDelete?.name"
      @confirm="confirmDelete"
    />
  </div>
</template>
