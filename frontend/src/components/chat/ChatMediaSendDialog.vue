<script setup lang="ts">
import { useI18n } from "vue-i18n";
import {
  Send,
  Paperclip,
  FileText,
  Image as ImageIcon,
  Play,
  X,
} from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export interface PendingMediaUpload {
  id: string;
  file: File;
  category: string;
  previewUrl: string | null;
}

const props = defineProps<{
  open: boolean;
  activeMediaUpload: PendingMediaUpload | null;
  selectedMediaUploads: PendingMediaUpload[];
  selectedMediaCount: number;
  isUploadingMedia: boolean;
  mediaCaption: string;
  canApplyMediaCaption: boolean;
  mediaDialogDescription: string;
  mediaUploadingLabel: string;
  mediaSendButtonLabel: string;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  "update:mediaCaption": [value: string];
  setActivePreview: [uploadId: string];
  removeUpload: [uploadId: string];
  close: [];
  send: [];
}>();

const { t } = useI18n();

function formatMediaUploadSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1048576).toFixed(1)} MB`;
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t("chat.sendMedia") }}</DialogTitle>
        <DialogDescription>
          {{ mediaDialogDescription }}
        </DialogDescription>
      </DialogHeader>
      <div class="py-4 space-y-4">
        <div
          v-if="
            activeMediaUpload?.category === 'image' &&
            activeMediaUpload.previewUrl
          "
          class="flex justify-center"
        >
          <img
            :src="activeMediaUpload.previewUrl"
            :alt="activeMediaUpload.file.name"
            class="max-w-full max-h-[300px] rounded-lg object-contain"
          />
        </div>
        <div
          v-else-if="
            activeMediaUpload?.category === 'video' &&
            activeMediaUpload.previewUrl
          "
          class="flex justify-center"
        >
          <video
            :src="activeMediaUpload.previewUrl"
            controls
            class="max-w-full max-h-[300px] rounded-lg"
          />
        </div>
        <div
          v-else-if="activeMediaUpload?.category === 'audio'"
          class="flex justify-center"
        >
          <div class="flex items-center gap-3 px-4 py-3 bg-muted rounded-lg">
            <div
              class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center"
            >
              <Paperclip class="h-5 w-5 text-primary" />
            </div>
            <div>
              <p class="font-medium text-sm">
                {{ activeMediaUpload.file.name }}
              </p>
              <p class="text-xs text-muted-foreground">
                {{ t("chat.audioFile") }}
              </p>
            </div>
          </div>
        </div>
        <div v-else-if="activeMediaUpload" class="flex justify-center">
          <div class="flex items-center gap-3 px-4 py-3 bg-muted rounded-lg">
            <div
              class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center"
            >
              <FileText class="h-5 w-5 text-primary" />
            </div>
            <div>
              <p class="font-medium text-sm truncate max-w-[200px]">
                {{ activeMediaUpload.file.name }}
              </p>
              <p class="text-xs text-muted-foreground">
                {{ formatMediaUploadSize(activeMediaUpload.file.size) }}
              </p>
            </div>
          </div>
        </div>

        <div v-if="selectedMediaCount > 1" class="space-y-2">
          <ScrollArea class="max-h-[220px] pr-3">
            <div class="space-y-2">
              <div
                v-for="upload in selectedMediaUploads"
                :key="upload.id"
                class="flex items-center gap-2"
              >
                <button
                  type="button"
                  class="flex flex-1 items-center gap-3 rounded-lg border px-3 py-2 text-left transition-colors"
                  :class="
                    activeMediaUpload?.id === upload.id
                      ? 'border-primary/45 bg-primary/10'
                      : 'border-border bg-muted/30 hover:bg-muted/60'
                  "
                  :disabled="isUploadingMedia"
                  @click="emit('setActivePreview', upload.id)"
                >
                  <div
                    class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10"
                  >
                    <ImageIcon
                      v-if="upload.category === 'image'"
                      class="h-5 w-5 text-primary"
                    />
                    <Play
                      v-else-if="upload.category === 'video'"
                      class="h-5 w-5 text-primary"
                    />
                    <Paperclip
                      v-else-if="upload.category === 'audio'"
                      class="h-5 w-5 text-primary"
                    />
                    <FileText v-else class="h-5 w-5 text-primary" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">
                      {{ upload.file.name }}
                    </p>
                    <p class="text-xs text-muted-foreground">
                      {{ t(`chat.${upload.category}`) }} ·
                      {{ formatMediaUploadSize(upload.file.size) }}
                    </p>
                  </div>
                </button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
                  :disabled="isUploadingMedia"
                  :aria-label="`${t('common.remove')} ${upload.file.name}`"
                  @click="emit('removeUpload', upload.id)"
                >
                  <X class="h-4 w-4" />
                </Button>
              </div>
            </div>
          </ScrollArea>
          <p class="text-sm text-muted-foreground">
            {{ t("chat.mediaBatchCaptionHint") }}
          </p>
        </div>

        <div v-else-if="canApplyMediaCaption">
          <Textarea
            :model-value="mediaCaption"
            :placeholder="t('chat.mediaCaption') + '...'"
            class="min-h-[60px] max-h-[100px] resize-none"
            :rows="2"
            @update:model-value="emit('update:mediaCaption', $event)"
          />
        </div>

        <div class="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            :disabled="isUploadingMedia"
            :aria-label="$t('common.cancel')"
            @click="emit('close')"
          >
            {{ t("common.cancel") }}
          </Button>
          <Button
            type="button"
            :disabled="isUploadingMedia || selectedMediaCount === 0"
            :aria-label="$t('chat.send')"
            @click="emit('send')"
          >
            <Send v-if="!isUploadingMedia" class="mr-2 h-4 w-4" />
            <span v-if="isUploadingMedia">{{ mediaUploadingLabel }}</span>
            <span v-else>{{ mediaSendButtonLabel }}</span>
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
