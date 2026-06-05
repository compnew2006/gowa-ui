<script setup lang="ts">
import { ref, computed } from "vue";
import { Paperclip, Download, Loader2, CheckCircle } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import type { MediaGroup } from "@/composables/useMediaGroups";
import JSZip from "jszip";

interface Message {
  id: string;
  media_url?: string;
  media_filename?: string;
  media_mime_type?: string;
  message_type: string;
}

const props = defineProps<{
  /** 'start' = rendered above the first message, 'end' = rendered below the last */
  variant: "start" | "end";
  /** The media group this bar belongs to */
  group: MediaGroup;
  /** Full message objects for this group (for download) */
  messages: Message[];
  /** Already-cached blob URLs keyed by message id */
  blobUrls: Record<string, string>;
}>();

const isDownloading = ref(false);
const downloadProgress = ref(0);
const downloadDone = ref(false);

const fileCount = computed(() => props.group.messageIds.length);

const basePath = ((window as any).__BASE_PATH__ ?? "").replace(/\/$/, "");

/**
 * Fetch a single media blob, using cached blob URL if available.
 */
async function fetchMediaBlob(
  message: Message,
): Promise<{ blob: Blob; filename: string }> {
  const cachedUrl = props.blobUrls[message.id];
  let blob: Blob;

  if (cachedUrl) {
    const resp = await fetch(cachedUrl);
    blob = await resp.blob();
  } else {
    const resp = await fetch(`${basePath}/api/media/${message.id}`, {
      credentials: "include",
    });
    if (!resp.ok)
      throw new Error(
        `Failed to fetch media for ${message.id}: ${resp.status}`,
      );
    blob = await resp.blob();
  }

  // Determine filename
  let filename =
    message.media_filename || `${message.message_type}_${message.id}`;
  // Ensure extension
  if (!filename.includes(".") && message.media_mime_type) {
    const ext = mimeToExt(message.media_mime_type);
    if (ext) filename += `.${ext}`;
  }

  return { blob, filename };
}

function mimeToExt(mime: string): string {
  const map: Record<string, string> = {
    "image/jpeg": "jpg",
    "image/png": "png",
    "image/gif": "gif",
    "image/webp": "webp",
    "video/mp4": "mp4",
    "video/3gpp": "3gp",
    "application/pdf": "pdf",
    "application/msword": "doc",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
      "docx",
    "application/vnd.ms-excel": "xls",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
    "text/plain": "txt",
  };
  return map[mime] || mime.split("/")[1] || "";
}

/**
 * Deduplicate filenames within a zip to avoid collisions.
 */
function deduplicateFilename(name: string, used: Set<string>): string {
  if (!used.has(name)) return name;
  const dot = name.lastIndexOf(".");
  const base = dot > 0 ? name.slice(0, dot) : name;
  const ext = dot > 0 ? name.slice(dot) : "";
  let i = 1;
  while (used.has(`${base}_${i}${ext}`)) i++;
  return `${base}_${i}${ext}`;
}

async function downloadAll() {
  if (isDownloading.value) return;
  isDownloading.value = true;
  downloadProgress.value = 0;
  downloadDone.value = false;

  try {
    const eligibleMessages = props.messages.filter((m) => m.media_url);

    if (eligibleMessages.length === 1) {
      // Single file: direct download
      const { blob, filename } = await fetchMediaBlob(eligibleMessages[0]);
      triggerBlobDownload(blob, filename);
    } else {
      // Multiple files: zip
      const zip = new JSZip();
      const usedNames = new Set<string>();

      for (let i = 0; i < eligibleMessages.length; i++) {
        const { blob, filename } = await fetchMediaBlob(eligibleMessages[i]);
        const uniqueName = deduplicateFilename(filename, usedNames);
        usedNames.add(uniqueName);
        zip.file(uniqueName, blob);
        downloadProgress.value = Math.round(
          ((i + 1) / eligibleMessages.length) * 80,
        );
      }

      downloadProgress.value = 90;
      const zipBlob = await zip.generateAsync({ type: "blob" });
      downloadProgress.value = 100;
      triggerBlobDownload(zipBlob, `whatsapp_files_${fileCount.value}.zip`);
    }

    downloadDone.value = true;
    setTimeout(() => {
      downloadDone.value = false;
    }, 2500);
  } catch (error) {
    console.error("Batch download failed:", error);
  } finally {
    isDownloading.value = false;
    downloadProgress.value = 0;
  }
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
</script>

<template>
  <!-- Group start indicator -->
  <div
    v-if="variant === 'start'"
    class="media-group-bar media-group-bar--start"
  >
    <div class="media-group-label">
      <Paperclip class="h-3.5 w-3.5" />
      <span>{{ fileCount }} files sent together</span>
    </div>
  </div>

  <!-- Group end indicator with download button -->
  <div v-else class="media-group-bar media-group-bar--end">
    <Button
      variant="ghost"
      size="sm"
      class="media-group-download-btn"
      :disabled="isDownloading"
      @click="downloadAll"
    >
      <Loader2 v-if="isDownloading" class="h-3.5 w-3.5 animate-spin" />
      <CheckCircle v-else-if="downloadDone" class="h-3.5 w-3.5 text-primary" />
      <Download v-else class="h-3.5 w-3.5" />
      <span v-if="isDownloading && downloadProgress > 0">
        Downloading... {{ downloadProgress }}%
      </span>
      <span v-else-if="downloadDone">Done!</span>
      <span v-else>Download All ({{ fileCount }} files)</span>
    </Button>
  </div>
</template>

<style scoped>
.media-group-bar {
  display: flex;
  justify-content: flex-start;
  padding-inline-start: 12px;
  max-width: 340px;
}

.media-group-bar--start {
  margin-bottom: 2px;
}

.media-group-bar--end {
  margin-top: 2px;
  margin-bottom: 4px;
}

.media-group-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 9999px;
  font-size: 11px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.5);
  background: rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(4px);
}

:root.light .media-group-label {
  color: rgb(107, 114, 128);
  background: rgb(229, 231, 235);
}

.media-group-download-btn {
  height: 28px;
  padding: 0 10px;
  font-size: 11px;
  font-weight: 500;
  gap: 5px;
  border-radius: 9999px;
  color: rgba(255, 255, 255, 0.6);
  background: rgb(var(--primary) / 0.08);
  border: 1px solid rgb(var(--primary) / 0.16);
  transition: all 0.2s ease;
}

.media-group-download-btn:hover:not(:disabled) {
  background: rgb(var(--primary) / 0.14);
  color: rgb(var(--primary));
  border-color: rgb(var(--primary) / 0.28);
}

:root.light .media-group-download-btn {
  color: rgb(75, 85, 99);
  background: rgb(var(--primary) / 0.07);
  border-color: rgb(var(--primary) / 0.14);
}

:root.light .media-group-download-btn:hover:not(:disabled) {
  background: rgb(var(--primary) / 0.12);
  color: rgb(var(--primary));
  border-color: rgb(var(--primary) / 0.24);
}
</style>
