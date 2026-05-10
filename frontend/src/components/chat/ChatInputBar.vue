<script setup lang="ts">
import { ref, defineAsyncComponent } from "vue";
import {
  Send,
  Paperclip,
  Smile,
  Check,
  Clock,
  Download,
  X,
  Loader2,
  Printer,
  RotateCw,
} from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import CannedResponsePicker from "@/components/chat/CannedResponsePicker.vue";
import type { Contact, Message } from "@/stores/contacts";
import type { CannedResponseAttachment } from "@/services/api";

const EmojiPicker = defineAsyncComponent(() => {
  return import("vue3-emoji-picker").then((module) => {
    import("vue3-emoji-picker/css");
    return module.default;
  });
});

defineProps<{
  isChatClosed: boolean;
  isChatRestricted: boolean;
  isChatSendRestricted: boolean;
  canReopen: boolean;
  isReopening: boolean;
  isServiceWindowExpired: boolean;
  isBatchPrintSelectionMode: boolean;
  canMergeSelectedBubbleFiles: boolean;
  selectedBatchPrintCount: number;
  isPreparingBatchPrint: boolean;
  showPrintButtons: boolean;
  replyingTo: Message | null;
  replyAuthorLabel: string;
  replyContent: string;
  pendingCannedResponse: {
    id: string;
    attachments: CannedResponseAttachment[];
  } | null;
  hasPendingCannedAttachments: boolean;
  isCurrentChatSendRestricted: boolean;
  canSendMessage: boolean;
  isSending: boolean;
  isDark: boolean;
  currentContact: Contact | null;
  cannedPickerOpen: boolean;
  cannedSearchQuery: string;
}>();

const emit = defineEmits<{
  send: [];
  "send-media": [];
  "open-file-picker": [];
  "handle-file-select": [event: Event];
  "insert-emoji": [emoji: string];
  "insert-canned-response": [payload: { id: string; content: string; attachments: CannedResponseAttachment[] }];
  "close-canned-picker": [];
  "open-batch-print-picker": [];
  "cancel-batch-print": [];
  "clear-replying": [];
  "clear-canned-attachments": [];
  "remove-canned-attachment": [index: number];
  reopen: [];
  "auto-resize": [];
}>();

const messageInput = defineModel<string>("messageInput", { required: true });
const messageInputRef = ref<HTMLTextAreaElement | null>(null);
const fileInputRef = ref<HTMLInputElement | null>(null);

function openFilePicker() {
  fileInputRef.value?.click();
}

function onFileSelected(event: Event) {
  emit("handle-file-select", event);
}

function onEmojiSelect(emoji: any) {
  emit("insert-emoji", emoji.i || emoji);
}

function autoResizeTextarea() {
  const textarea = messageInputRef.value;
  if (!textarea) return;
  textarea.style.height = "auto";
  textarea.style.height = Math.min(textarea.scrollHeight, 120) + "px";
}

defineExpose({ messageInputRef, autoResizeTextarea });
</script>

<template>
  <!-- Closed chat banner -->
  <div
    v-if="isChatClosed && !isChatRestricted"
    class="flex items-center justify-between gap-3 border-t border-border bg-muted/55 px-4 py-2 text-xs text-muted-foreground"
  >
    <span>This chat is closed. You can view message history in read-only mode.</span>
    <Button
      v-if="canReopen"
      size="sm"
      variant="outline"
      class="h-7 px-2.5 text-xs"
      :disabled="isReopening"
      :aria-label="isReopening ? $t('chat.reopeningChat') : $t('chat.reopenChat')"
      @click="emit('reopen')"
    >
      <Loader2 v-if="isReopening" class="mr-1.5 h-3 w-3 animate-spin" />
      <RotateCw v-else class="mr-1.5 h-3 w-3" />
      Reopen Chat
    </Button>
  </div>

  <!-- Service window expired banner -->
  <div
    v-if="isServiceWindowExpired"
    class="flex items-center gap-2 border-t border-destructive/20 bg-destructive/10 px-4 py-2.5"
  >
    <Clock class="h-4 w-4 text-red-500 shrink-0" />
    <span class="text-sm text-red-500 flex-1">{{ $t("chat.serviceWindowExpired") }}</span>
  </div>

  <!-- Reply indicator -->
  <div
    v-if="replyingTo && !isChatClosed && !isChatRestricted && !isChatSendRestricted"
    class="flex items-center justify-between border-t border-border bg-card/80 px-4 py-2"
  >
    <div class="flex-1 min-w-0">
      <p class="text-xs font-medium text-muted-foreground">
        Replying to {{ replyAuthorLabel }}
      </p>
      <p class="truncate text-sm text-foreground/80">{{ replyContent || "[Media]" }}</p>
    </div>
    <button
      class="flex h-6 w-6 shrink-0 items-center justify-center rounded transition-colors hover:bg-accent"
      :aria-label="$t('chat.cancelReply')"
      @click="emit('clear-replying')"
    >
      <X class="h-4 w-4 text-muted-foreground" />
    </button>
  </div>

  <!-- Message Input -->
  <div v-if="!isChatClosed && !isChatRestricted" class="border-t border-border bg-card/95 p-4">
    <div
      v-if="isChatSendRestricted"
      class="mb-2 rounded-lg border border-accent px-3 py-2 text-xs text-muted-foreground"
    >
      This chat can be viewed without claim, but sending is blocked until you claim it.
    </div>

    <div
      v-if="pendingCannedResponse?.attachments?.length"
      class="mb-2 rounded-lg border border-primary/20 bg-primary/8 p-2"
    >
      <div class="mb-1 flex items-center justify-between">
        <p class="text-xs text-primary">
          {{ pendingCannedResponse.attachments.length }} canned media attachment(s) ready
        </p>
        <button
          type="button"
          class="text-xs text-primary hover:text-foreground"
          aria-label="Clear canned attachments"
          @click="emit('clear-canned-attachments')"
        >
          Clear
        </button>
      </div>
      <div class="flex flex-wrap gap-1.5">
        <div
          v-for="(attachment, index) in pendingCannedResponse.attachments"
          :key="attachment.id"
          class="inline-flex items-center gap-1.5 rounded-md bg-primary/12 px-2 py-1 text-xs text-primary"
        >
          <Download class="h-3.5 w-3.5" />
          <span class="max-w-[200px] truncate">{{ attachment.file_name }}</span>
          <button type="button" class="inline-flex items-center" :aria-label="`Remove attachment: ${attachment.file_name}`" @click="emit('remove-canned-attachment', index)">
            <X class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
    </div>

    <div
      v-if="isBatchPrintSelectionMode"
      class="mb-2 rounded-lg border border-primary/20 bg-primary/8 px-3 py-2"
    >
      <div class="flex items-center justify-between gap-3">
        <p class="text-xs text-primary">{{ $t("chat.batchPrintSelectionModeDesc") }}</p>
        <div class="flex items-center gap-2">
          <span class="text-xs text-primary">
            {{ $t("chat.batchPrintSelectedCount", { count: selectedBatchPrintCount }) }}
          </span>
          <Button
            variant="ghost"
            size="xs"
            class="h-7 px-2 text-[11px] text-primary hover:bg-primary/12 hover:text-foreground"
            :aria-label="$t('common.cancel')"
            @click="emit('cancel-batch-print')"
          >
            {{ $t("common.cancel") }}
          </Button>
        </div>
      </div>
    </div>

    <form
      @submit.prevent="emit('send')"
      role="search"
      :aria-label="$t('chat.sendMessage')"
      class="flex items-center gap-2 rounded-xl border border-border bg-background/80 p-2"
      :class="isChatSendRestricted && 'opacity-70'"
    >
      <Tooltip>
        <TooltipTrigger as-child>
          <span>
      <Popover>
        <PopoverTrigger as-child>
          <button
            type="button"
            :disabled="isChatSendRestricted"
            :aria-label="$t('chat.emojiPicker')"
            class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:bg-accent"
          >
            <Smile class="h-[18px] w-[18px] text-muted-foreground" />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" side="top" class="w-auto p-0 border-0 shadow-none">
          <EmojiPicker
            :native="true"
            :hide-search="false"
            :hide-group-icons="false"
            :hide-group-names="false"
            :static-texts="{ placeholder: 'Search...' }"
            :theme="isDark ? 'dark' : 'light'"
            @select="onEmojiSelect"
          />
        </PopoverContent>
      </Popover>
          </span>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.emojiPicker") }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <span>
            <CannedResponsePicker
              :contact="currentContact"
              :external-open="cannedPickerOpen"
              :external-search="cannedSearchQuery"
              :aria-label="$t('chat.cannedResponses')"
              :class="isChatSendRestricted && 'pointer-events-none opacity-60'"
              @select="emit('insert-canned-response', $event)"
              @close="emit('close-canned-picker')"
            />
          </span>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.cannedResponses") }}</TooltipContent>
      </Tooltip>
      <input
        ref="fileInputRef"
        type="file"
        multiple
        accept="image/jpeg,image/png,image/webp,video/mp4,video/3gpp,audio/aac,audio/amr,audio/mp3,audio/mp4,audio/ogg,.mp3,.m4a,.ogg,.aac,.amr,.mp4,.3gp,.jpg,.jpeg,.png,.webp,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.zip,.rar"
        class="hidden"
        @change="onFileSelected"
      />
      <Tooltip>
        <TooltipTrigger as-child>
          <button
            type="button"
            :disabled="isChatSendRestricted"
            :aria-label="$t('chat.attachFile')"
            class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:bg-accent"
            @click="openFilePicker"
          >
            <Paperclip class="h-[18px] w-[18px] text-muted-foreground" />
          </button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.attachFile") }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="showPrintButtons">
        <TooltipTrigger as-child>
          <button
            type="button"
            class="relative flex h-9 w-9 items-center justify-center rounded-lg transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="isPreparingBatchPrint || (isBatchPrintSelectionMode && !canMergeSelectedBubbleFiles)"
            :aria-label="isBatchPrintSelectionMode ? $t('chat.batchPrintConfirmAction') : $t('chat.mergePrint')"
            @click="emit('open-batch-print-picker')"
          >
            <Loader2 v-if="isPreparingBatchPrint" class="h-[18px] w-[18px] animate-spin text-muted-foreground" />
            <Check v-else-if="isBatchPrintSelectionMode" class="h-[18px] w-[18px] text-primary" />
            <Printer v-else class="h-[18px] w-[18px] text-muted-foreground" />
            <span
              v-if="isBatchPrintSelectionMode && selectedBatchPrintCount > 0"
              class="absolute -right-1 -top-1 min-w-4 rounded-full bg-primary px-1 text-center text-[10px] font-semibold leading-4 text-primary-foreground"
            >
              {{ selectedBatchPrintCount }}
            </span>
          </button>
        </TooltipTrigger>
        <TooltipContent>
          {{ isBatchPrintSelectionMode ? $t("chat.batchPrintConfirmAction") : $t("chat.mergePrint") }}
        </TooltipContent>
      </Tooltip>
      <textarea
        ref="messageInputRef"
        v-model="messageInput"
        :placeholder="$t('chat.typeMessage') + '...'"
        :aria-label="$t('chat.typeMessage')"
        rows="1"
        class="min-h-[36px] max-h-[120px] flex-1 resize-none overflow-y-auto bg-transparent py-2 text-[14px] text-foreground placeholder:text-muted-foreground focus:outline-none"
        :disabled="isChatSendRestricted || isSending"
        @keydown.enter.exact.prevent="emit('send')"
        @input="autoResizeTextarea"
      />
      <button
        type="submit"
        :aria-label="$t('chat.send')"
        class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        :disabled="isChatSendRestricted || !canSendMessage || isSending"
      >
        <Send class="w-4 h-4 text-white" />
      </button>
    </form>
  </div>
</template>
