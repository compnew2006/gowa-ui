<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { savedContentsService, type SavedContent } from "@/services/api";
import { Search, FileText, Check } from "lucide-vue-next";
import { useDebounceFn } from "@vueuse/core";
import { toast } from "vue-sonner";
import { getErrorMessage } from "@/lib/api-utils";

const props = defineProps<{
  categoryFilter?: string;
}>();

const open = defineModel<boolean>("open", { default: false });
const emit = defineEmits<{
  select: [content: SavedContent];
}>();

const { t } = useI18n();

const items = ref<SavedContent[]>([]);
const categories = ref<string[]>([]);
const isLoading = ref(false);
const searchQuery = ref("");
const selectedCategory = ref(props.categoryFilter || "all");
const selectedId = ref<string | null>(null);

async function fetchCategories() {
  try {
    const response = await savedContentsService.categories();
    const data = (response.data as any).data || response.data;
    categories.value = data.categories || [];
  } catch {
    // ignore
  }
}

async function fetchItems() {
  isLoading.value = true;
  try {
    const response = await savedContentsService.list({
      search: searchQuery.value || undefined,
      category:
        selectedCategory.value !== "all" ? selectedCategory.value : undefined,
      limit: 50,
    });
    const data = (response.data as any).data || response.data;
    items.value = data.saved_contents || [];
  } catch (error) {
    toast.error(
      getErrorMessage(error, t("savedContents.failedLoad")),
    );
  } finally {
    isLoading.value = false;
  }
}

const debouncedSearch = useDebounceFn(() => fetchItems(), 300);
watch(searchQuery, () => debouncedSearch());
watch(selectedCategory, () => fetchItems());

function handleOpen(val: boolean) {
  open.value = val;
  if (val) {
    fetchCategories();
    fetchItems();
    selectedId.value = null;
  }
}

function selectItem(item: SavedContent) {
  selectedId.value = item.id;
}

function confirmSelection() {
  const item = items.value.find((i) => i.id === selectedId.value);
  if (item) {
    emit("select", item);
    open.value = false;
    toast.success(t("savedContents.contentLoaded"));
  }
}

onMounted(() => {
  fetchCategories();
});
</script>

<template>
  <Dialog :open="open" @update:open="handleOpen">
    <DialogContent class="max-w-2xl max-h-[80vh]">
      <DialogHeader>
        <DialogTitle>{{ t("savedContents.pickerTitle") }}</DialogTitle>
        <DialogDescription>{{ t("savedContents.pickerDesc") }}</DialogDescription>
      </DialogHeader>

      <div class="space-y-3">
        <div class="flex gap-2">
          <div class="relative flex-1">
            <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              v-model="searchQuery"
              :placeholder="t('savedContents.searchPlaceholder')"
              class="pl-8"
            />
          </div>
          <Select v-model="selectedCategory">
            <SelectTrigger class="w-[160px]">
              <SelectValue :placeholder="t('savedContents.allCategories')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{{ t("savedContents.allCategories") }}</SelectItem>
              <SelectItem
                v-for="cat in categories"
                :key="cat"
                :value="cat"
              >
                {{ cat }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <ScrollArea class="h-[400px]">
          <div v-if="isLoading" class="flex items-center justify-center py-8 text-muted-foreground">
            {{ t("common.loading") }}
          </div>
          <div v-else-if="items.length === 0" class="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <FileText class="h-8 w-8 mb-2" />
            <p>{{ t("savedContents.noContentsFound") }}</p>
          </div>
          <div v-else class="space-y-1">
            <button
              v-for="item in items"
              :key="item.id"
              class="w-full text-left p-3 rounded-lg border transition-colors hover:bg-accent"
              :class="selectedId === item.id ? 'border-primary bg-accent' : 'border-transparent'"
              @click="selectItem(item)"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <span class="font-medium truncate">{{ item.name }}</span>
                    <Badge v-if="item.category" variant="outline" class="text-xs shrink-0">
                      {{ item.category }}
                    </Badge>
                  </div>
                  <p class="text-sm text-muted-foreground mt-1 line-clamp-2">
                    {{ item.preview || item.body }}
                  </p>
                  <div v-if="item.variables && item.variables.length > 0" class="flex flex-wrap gap-1 mt-1">
                    <Badge
                      v-for="v in item.variables"
                      :key="v"
                      variant="secondary"
                      class="text-xs font-mono"
                    >
                      {{ v }}
                    </Badge>
                  </div>
                </div>
                <Check
                  v-if="selectedId === item.id"
                  class="h-4 w-4 text-primary shrink-0 mt-1"
                />
              </div>
            </button>
          </div>
        </ScrollArea>
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <Button variant="outline" size="sm" @click="open = false">
          {{ t("common.cancel") }}
        </Button>
        <Button
          size="sm"
          :disabled="!selectedId"
          @click="confirmSelection"
        >
          {{ t("savedContents.loadContent") }}
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
