<script setup lang="ts">
import { X } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";

defineProps<{
  open: boolean;
  url: string | null;
  type: "image" | "video" | "audio" | "document" | null;
  title: string | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  close: [];
}>();

function handleClose() {
  emit("close");
  emit("update:open", false);
}
</script>

<template>
  <Dialog :open="open" @update:open="(v) => !v && handleClose()">
    <DialogContent
      class="max-w-4xl p-0 overflow-hidden border-white/10 light:border-gray-200"
    >
      <div
        class="bg-black/95 light:bg-black/90 p-3 space-y-3"
        data-testid="chat-media-viewer-dialog"
      >
        <div class="flex items-center justify-between gap-3">
          <DialogTitle class="text-sm font-medium text-white truncate">
            {{ title || "Media Preview" }}
          </DialogTitle>
          <Button
            variant="ghost"
            size="icon"
            class="h-7 w-7 text-white hover:text-white"
            @click="handleClose"
          >
            <X class="h-4 w-4" />
          </Button>
        </div>

        <div
          class="flex items-center justify-center min-h-[220px] max-h-[80vh]"
        >
          <img
            v-if="type === 'image' && url"
            :src="url"
            alt="Media preview"
            class="max-w-full max-h-[76vh] rounded-md object-contain"
            data-testid="chat-media-viewer-image"
          />
          <video
            v-else-if="type === 'video' && url"
            :src="url"
            controls
            autoplay
            class="max-w-full max-h-[76vh] rounded-md"
            data-testid="chat-media-viewer-video"
          />
          <audio
            v-else-if="type === 'audio' && url"
            :src="url"
            controls
            class="w-full max-w-md"
            data-testid="chat-media-viewer-audio"
          />
          <a
            v-else-if="url"
            :href="url"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-white underline underline-offset-4"
          >
            Open media in new tab
          </a>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
