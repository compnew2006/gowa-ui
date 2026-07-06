<script setup lang="ts">
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  CrudFormDialog,
} from "@/components/shared";
import { Upload, X } from "lucide-vue-next";

const props = defineProps<{
  isEditing: boolean;
  categories: string[];
}>();

const open = defineModel<boolean>("open", { default: false });
const emit = defineEmits<{
  saved: [payload: { name: string; body: string; category: string; mediaFile: File | null }];
}>();

const { t } = useI18n();

const isSubmitting = ref(false);
const name = ref("");
const body = ref("");
const category = ref("__none__");
const mediaFile = ref<File | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);
const fileInputKey = ref(0);
const existingCategories = computed(() => props.categories);

const allowedMediaTypes = [
  "image/jpeg", "image/png", "image/webp",
  "video/mp4", "video/3gpp",
  "audio/aac", "audio/mp4", "audio/mpeg", "audio/ogg",
  "application/pdf",
];

const maxMediaSize = 16 * 1024 * 1024;

const variableTokens = [
  "{name}",
  "{phone}",
  "{date}",
  "{time}",
  "{group_name}",
  "{customer_name}",
  "{contact_name}",
  "{agent_name}",
  "{organization_name}",
  "{phone_number}",
  "{chat_id}",
];

const detectedVariables = computed(() => {
  const matches = body.value.match(/\{[a-zA-Z_][a-zA-Z0-9_]*\}/g);
  if (!matches) return [];
  return [...new Set(matches)];
});

const charCount = computed(() => body.value.length);
const estimatedSegments = computed(() => {
  if (!body.value) return 0;
  return Math.ceil(body.value.length / 1024);
});

function insertVariable(token: string) {
  const textarea = document.querySelector(
    'textarea[data-content-body]',
  ) as HTMLTextAreaElement | null;
  if (textarea) {
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    body.value =
      body.value.substring(0, start) + token + body.value.substring(end);
    setTimeout(() => {
      textarea.focus();
      textarea.setSelectionRange(
        start + token.length,
        start + token.length,
      );
    }, 0);
  } else {
    body.value += token;
  }
}

function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  if (file.size > maxMediaSize) {
    alert(t("savedContents.mediaTooLarge"));
    clearFileInput();
    return;
  }

  if (!allowedMediaTypes.includes(file.type)) {
    alert(t("savedContents.mediaUnsupported"));
    clearFileInput();
    return;
  }

  mediaFile.value = file;
}

function clearFileInput() {
  mediaFile.value = null;
  fileInputKey.value++;
}

function triggerFilePicker() {
  fileInput.value?.click();
}

function resetForm() {
  name.value = "";
  body.value = "";
  category.value = "__none__";
  clearFileInput();
}

function openForCreate() {
  resetForm();
  open.value = true;
}

function openForEdit(item: { name: string; body: string; category: string; media_filename?: string }) {
  name.value = item.name;
  body.value = item.body;
  category.value = item.category || "__none__";
  mediaFile.value = null;
  open.value = true;
}

async function handleSubmit() {
  if (!name.value.trim() || !body.value.trim()) {
    return;
  }
  isSubmitting.value = true;
  try {
    emit("saved", {
      name: name.value.trim(),
      body: body.value.trim(),
      category: category.value === "__none__" ? "" : category.value.trim(),
      mediaFile: mediaFile.value,
    });
  } finally {
    isSubmitting.value = false;
  }
}

defineExpose({ openForCreate, openForEdit, resetForm });
</script>

<template>
  <CrudFormDialog
    v-model:open="open"
    :is-editing="isEditing"
    :is-submitting="isSubmitting"
    :edit-title="t('savedContents.editTitle')"
    :create-title="t('savedContents.createTitle')"
    :edit-description="t('savedContents.editDesc')"
    :create-description="t('savedContents.createDesc')"
    max-width="max-w-2xl"
    @submit="handleSubmit"
  >
    <div class="space-y-4">
      <div class="space-y-2">
        <Label>{{ t("savedContents.name") }}</Label>
        <Input
          v-model="name"
          :placeholder="t('savedContents.namePlaceholder')"
        />
      </div>

      <div class="space-y-2">
        <Label>{{ t("savedContents.body") }}</Label>
        <div
          class="flex flex-wrap gap-1 mb-1"
        >
          <Button
            v-for="token in variableTokens"
            :key="token"
            variant="outline"
            size="sm"
            class="h-6 text-xs font-mono"
            @click="insertVariable(token)"
          >
            {{ token }}
          </Button>
        </div>
        <Textarea
          v-model="body"
          data-content-body
          :placeholder="t('savedContents.bodyPlaceholder')"
          :rows="6"
          class="font-mono text-sm"
        />
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span>{{ charCount }} {{ t("savedContents.characters") }}</span>
          <span v-if="body">{{ estimatedSegments }} {{ t("savedContents.segments") }}</span>
        </div>
        <div v-if="detectedVariables.length > 0" class="flex flex-wrap items-center gap-1">
          <span class="text-xs text-muted-foreground">{{ t("savedContents.variablesDetected") }}:</span>
          <Badge
            v-for="v in detectedVariables"
            :key="v"
            variant="secondary"
            class="text-xs font-mono"
          >
            {{ v }}
          </Badge>
        </div>
      </div>

      <div class="space-y-2">
        <Label>{{ t("savedContents.mediaFile") }}</Label>
        <div class="flex items-center gap-2">
          <input
            :key="fileInputKey"
            ref="fileInput"
            type="file"
            :accept="allowedMediaTypes.join(',')"
            class="hidden"
            @change="handleFileSelect"
          />
          <Button variant="outline" type="button" @click="triggerFilePicker">
            <Upload class="h-4 w-4 mr-2" />
            {{ t("savedContents.chooseFile") }}
          </Button>
          <span v-if="mediaFile" class="text-sm text-muted-foreground truncate max-w-[200px]">
            {{ mediaFile.name }}
          </span>
          <Button
            v-if="mediaFile"
            variant="ghost"
            size="icon"
            class="h-8 w-8"
            @click="clearFileInput"
          >
            <X class="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div class="space-y-2">
        <Label>{{ t("savedContents.category") }}</Label>
        <div class="flex gap-2">
          <Select v-model="category">
            <SelectTrigger class="w-[200px]">
              <SelectValue :placeholder="t('savedContents.selectCategory')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">{{ t("savedContents.uncategorized") }}</SelectItem>
              <SelectItem
                v-for="cat in existingCategories"
                :key="cat"
                :value="cat"
              >
                {{ cat }}
              </SelectItem>
            </SelectContent>
          </Select>
          <Input
            v-model="category"
            :placeholder="t('savedContents.newCategoryPlaceholder')"
            class="flex-1"
          />
        </div>
      </div>
    </div>
  </CrudFormDialog>
</template>
