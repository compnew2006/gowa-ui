<script setup lang="ts">
import {
  FileText,
  Download,
  Printer,
  Check,
  Loader2,
  Trash2,
  RotateCw,
  Reply,
  SmilePlus,
  AlertCircle,
  MapPin,
  ExternalLink,
  Phone,
  User,
} from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import LinkifiedMessageText from "@/components/chat/LinkifiedMessageText.vue";
import type { Message } from "@/stores/contacts";

const props = defineProps<{
  message: Message;
  isBatchPrintSelectionMode: boolean;
  isBatchPrintSelectable: boolean;
  isBatchPrintSelected: boolean;
  isMediaGroupMember: boolean;
  canRevoke: boolean;
  isRevoking: boolean;
  isRetrying: boolean;
  reactionPickerMessageId: string | null;
  quickReactionEmojis: string[];
  isMediaLoading: boolean;
  mediaBlobUrl: string | undefined;
  isDeleted: boolean;
  isSystemEvent: boolean;
  showGroupSenderPhone: boolean;
  groupSenderPhone: string;
  messageContent: string;
  isMediaMessage: boolean;
  formattedTime: string;
  statusIcon: any;
  statusClass: string;
  replyAuthorLabel: string;
  replyContent: string;
  showReplyThumbnail: boolean;
  replyThumbnailUrl: string | undefined;
  interactiveButtons: { id: string; title: string }[];
  ctaUrlData: { url: string; button_text: string } | null;
  locationData: { name?: string; address?: string; latitude: number; longitude: number } | null;
  contactsData: { name: string; phones?: string[] }[];
  attachmentFilename: string;
  showPrintButton: boolean;
  showDownloadButton: boolean;
  isPrintSupported: boolean;
  isReplyMessage: boolean;
  replyToMessageId: string | undefined;
  hasMediaUrl: boolean;
  mediaType: string;
}>();

const emit = defineEmits<{
  "click-bubble": [message: Message, event: MouseEvent];
  "toggle-batch-select": [messageId: string];
  "click-media-preview": [message: Message, event?: MouseEvent];
  "click-reply-preview-media": [message: Message, event?: MouseEvent];
  "image-error": [event: Event];
  "image-load": [];
  "media-error": [event: Event, mediaType: string];
  "send-reaction": [messageId: string, emoji: string];
  reply: [message: Message];
  revoke: [message: Message];
  retry: [message: Message];
  "scroll-to-message": [messageId: string | undefined];
  download: [message: Message, event?: MouseEvent];
  print: [message: Message, event?: MouseEvent];
  "open-reaction-picker": [messageId: string];
  "close-reaction-picker": [];
  "reply-preview-thumb-error": [message: Message];
}>();

function _emitReplyPreviewThumbError() {
  emit('reply-preview-thumb-error', props.message);
}
</script>

<template>
  <div
    :id="`message-${message.id}`"
    role="article"
    :aria-label="`${message.direction === 'outgoing' ? $t('chat.outgoingMessage') : $t('chat.incomingMessage')}, ${formattedTime}`"
    :class="[
      'flex group',
      isSystemEvent ? 'justify-center' : message.direction === 'outgoing' ? 'justify-end' : 'justify-start',
    ]"
  >
    <div
      :class="[
        'chat-bubble relative',
        isSystemEvent ? 'chat-bubble-system' : message.direction === 'outgoing' ? 'chat-bubble-outgoing' : 'chat-bubble-incoming',
        isDeleted ? 'chat-bubble-deleted' : '',
        isMediaGroupMember ? 'media-group-member' : '',
        isBatchPrintSelectionMode && isBatchPrintSelectable ? 'batch-print-selectable-bubble' : '',
        isBatchPrintSelected ? 'batch-print-selected-bubble' : '',
      ]"
      @click="emit('click-bubble', message, $event)"
    >
      <button
        v-if="isBatchPrintSelectionMode && isBatchPrintSelectable"
        type="button"
        class="batch-print-bubble-marker"
        :class="isBatchPrintSelected ? 'batch-print-bubble-marker--selected' : ''"
        :aria-label="isBatchPrintSelected ? $t('chat.deselectMessage') : $t('chat.selectMessage')"
        :aria-pressed="isBatchPrintSelected"
        @click.stop.prevent="emit('toggle-batch-select', message.id)"
      >
        <Check v-if="isBatchPrintSelected" class="h-3 w-3" />
      </button>
      <p v-if="showGroupSenderPhone" class="mb-1 text-[11px] font-medium text-primary">
        {{ groupSenderPhone }}
      </p>

      <!-- Reply preview -->
      <div
        v-if="isReplyMessage && message.reply_to_message"
        role="button"
        tabindex="0"
        class="reply-preview cursor-pointer text-xs"
        :aria-label="`Reply to ${replyAuthorLabel}`"
        @click="emit('scroll-to-message', replyToMessageId)"
        @keydown.enter="emit('scroll-to-message', replyToMessageId)"
      >
        <p class="font-medium">{{ replyAuthorLabel }}</p>
        <div class="reply-preview-content">
          <img
            v-if="showReplyThumbnail"
            :src="replyThumbnailUrl"
            alt="Reply image preview"
            class="reply-preview-thumb"
            @click.stop="emit('click-reply-preview-media', message, $event)"
            @error="_emitReplyPreviewThumbError"
          />
          <p class="truncate">{{ replyContent }}</p>
        </div>
      </div>

      <!-- Image message -->
      <div v-if="mediaType === 'image' && hasMediaUrl" class="mb-2">
        <div v-if="isMediaLoading" class="w-[200px] h-[150px] bg-muted rounded-lg animate-pulse flex items-center justify-center">
          <span class="text-muted-foreground text-sm">{{ $t("common.loading") }}...</span>
        </div>
        <img
          v-else-if="mediaBlobUrl"
          :src="mediaBlobUrl"
          :alt="(message.content as any)?.body || 'Image'"
          class="max-w-[280px] max-h-[300px] rounded-lg cursor-pointer object-cover"
          @click="emit('click-media-preview', message, $event)"
          @error="emit('image-error', $event)"
          @load="emit('image-load')"
        />
        <div v-else class="w-[200px] h-[150px] bg-muted rounded-lg flex items-center justify-center">
          <span class="text-muted-foreground text-sm">[Image]</span>
        </div>
        <div v-if="mediaBlobUrl && (showPrintButton || showDownloadButton)" class="mt-2 flex flex-wrap items-center gap-1.5">
          <Button v-if="showPrintButton" variant="ghost" size="xs" class="h-7 px-2 text-[11px]" :aria-label="$t('common.print')" @click.stop="emit('print', message, $event)">
            <Printer class="h-3.5 w-3.5" /> {{ $t("common.print") }}
          </Button>
          <Button v-if="showDownloadButton" variant="ghost" size="xs" class="h-7 px-2 text-[11px]" :aria-label="$t('common.download')" @click.stop="emit('download', message, $event)">
            <Download class="h-3.5 w-3.5" /> {{ $t("common.download") }}
          </Button>
        </div>
      </div>

      <!-- Sticker message -->
      <div v-else-if="mediaType === 'sticker' && hasMediaUrl" class="mb-2">
        <div v-if="isMediaLoading" class="w-[128px] h-[128px] bg-muted rounded-lg animate-pulse flex items-center justify-center">
          <span class="text-muted-foreground text-sm">{{ $t("common.loading") }}...</span>
        </div>
        <img
          v-else-if="mediaBlobUrl"
          :src="mediaBlobUrl"
          alt="Sticker"
          class="max-w-[128px] max-h-[128px] cursor-pointer"
          @click="emit('click-media-preview', message, $event)"
          @error="emit('image-error', $event)"
          @load="emit('image-load')"
        />
        <div v-else class="w-[128px] h-[128px] bg-muted rounded-lg flex items-center justify-center">
          <span class="text-muted-foreground text-sm">[Sticker]</span>
        </div>
      </div>

      <!-- Video message -->
      <div v-else-if="mediaType === 'video' && hasMediaUrl" class="mb-2">
        <div v-if="isMediaLoading" class="w-[200px] h-[150px] bg-muted rounded-lg animate-pulse flex items-center justify-center">
          <span class="text-muted-foreground text-sm">{{ $t("common.loading") }}...</span>
        </div>
        <video v-else-if="mediaBlobUrl" :src="mediaBlobUrl" controls class="max-w-[280px] max-h-[300px] rounded-lg" @error="emit('media-error', $event, 'video')" />
        <div v-else class="w-[200px] h-[150px] bg-muted rounded-lg flex items-center justify-center">
          <span class="text-muted-foreground text-sm">[Video]</span>
        </div>
      </div>

      <!-- Audio message -->
      <div v-else-if="mediaType === 'audio' && hasMediaUrl" class="mb-2">
        <div v-if="isMediaLoading" class="w-[200px] h-[40px] bg-muted rounded-lg animate-pulse"></div>
        <audio v-else-if="mediaBlobUrl" :src="mediaBlobUrl" controls class="max-w-[280px]" @error="emit('media-error', $event, 'audio')" />
        <div v-else class="text-muted-foreground text-sm">[Audio]</div>
      </div>

      <!-- Document message -->
      <div v-else-if="mediaType === 'document' && hasMediaUrl" class="mb-2">
        <div v-if="mediaBlobUrl" class="space-y-2">
          <a
            :href="mediaBlobUrl"
            :download="attachmentFilename"
            :aria-label="`${$t('common.download')} ${attachmentFilename}`"
            class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
          >
            <FileText class="h-5 w-5 text-muted-foreground" />
            <span class="text-sm truncate max-w-[200px]">{{ attachmentFilename }}</span>
          </a>
          <div v-if="showPrintButton || showDownloadButton" class="flex flex-wrap items-center gap-1.5">
            <Button v-if="showPrintButton && isPrintSupported" variant="ghost" size="xs" class="h-7 px-2 text-[11px]" :aria-label="$t('common.print')" @click.stop="emit('print', message, $event)">
              <Printer class="h-3.5 w-3.5" /> {{ $t("common.print") }}
            </Button>
            <Button v-if="showDownloadButton" variant="ghost" size="xs" class="h-7 px-2 text-[11px]" :aria-label="$t('common.download')" @click.stop="emit('download', message, $event)">
              <Download class="h-3.5 w-3.5" /> {{ $t("common.download") }}
            </Button>
          </div>
        </div>
        <div v-else-if="isMediaLoading" class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg">
          <FileText class="h-5 w-5 text-muted-foreground" />
          <span class="text-sm text-muted-foreground">{{ $t("common.loading") }}...</span>
        </div>
        <div v-else class="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg">
          <FileText class="h-5 w-5 text-muted-foreground" />
          <span class="text-sm text-muted-foreground">[Document]</span>
        </div>
      </div>

      <!-- Location message -->
      <div v-else-if="mediaType === 'location' && locationData" class="mb-2">
        <a
          :href="`https://www.google.com/maps?q=${locationData.latitude},${locationData.longitude}`"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Open in Google Maps"
          class="flex items-center gap-3 px-3 py-3 bg-background/50 rounded-lg hover:bg-background/80 transition-colors"
        >
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-destructive/10">
            <MapPin class="h-5 w-5 text-red-500" />
          </div>
          <div class="flex-1 min-w-0">
            <p v-if="locationData.name" class="text-sm font-medium truncate">{{ locationData.name }}</p>
            <p v-else class="text-sm font-medium">Location</p>
            <p v-if="locationData.address" class="text-xs text-muted-foreground truncate">{{ locationData.address }}</p>
            <p class="text-xs text-muted-foreground">
              {{ locationData.latitude.toFixed(6) }}, {{ locationData.longitude.toFixed(6) }}
            </p>
          </div>
          <ExternalLink class="h-4 w-4 text-muted-foreground shrink-0" />
        </a>
      </div>

      <!-- Contacts message -->
      <div
        v-else-if="(mediaType === 'contacts' || mediaType === 'contact') && contactsData.length > 0"
        class="mb-2 space-y-2"
      >
        <div v-for="(contact, idx) in contactsData" :key="idx" class="flex items-center gap-3 px-3 py-2 bg-background/50 rounded-lg">
          <div class="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
            <User class="h-5 w-5 text-primary" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium truncate">{{ contact.name }}</p>
            <div v-if="contact.phones?.length" class="flex items-center gap-1 text-xs text-muted-foreground">
              <Phone class="h-3 w-3" />
              <span class="truncate">{{ contact.phones.join(", ") }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Unsupported message -->
      <div v-else-if="mediaType === 'unsupported'" class="mb-2">
        <div class="flex items-center gap-2 px-3 py-2 bg-muted/50 rounded-lg text-muted-foreground">
          <AlertCircle class="h-4 w-4 shrink-0" />
          <span class="text-sm italic">This message type is not supported</span>
        </div>
      </div>

      <!-- Button reply -->
      <div v-if="message.message_type === 'button_reply'" class="button-reply-bubble">
        <span class="whitespace-pre-wrap break-words"><LinkifiedMessageText :text="messageContent" /></span>
        <span class="chat-bubble-time"><span>{{ formattedTime }}</span></span>
      </div>

      <!-- Text content -->
      <span v-else-if="messageContent" class="whitespace-pre-wrap break-words">
        <LinkifiedMessageText :text="messageContent" />
        <span class="chat-bubble-time">
          <span>{{ formattedTime }}</span>
          <component v-if="message.direction === 'outgoing' && !isSystemEvent" :is="statusIcon" :class="['h-4 w-4 status-icon', statusClass]" />
        </span>
      </span>

      <!-- Fallback for media without URL -->
      <span v-else-if="isMediaMessage && !hasMediaUrl" class="text-muted-foreground italic">
        [{{ message.message_type.charAt(0).toUpperCase() + message.message_type.slice(1) }}]
        <span class="chat-bubble-time">
          <span>{{ formattedTime }}</span>
          <component v-if="message.direction === 'outgoing' && !isSystemEvent" :is="statusIcon" :class="['h-4 w-4 status-icon', statusClass]" />
        </span>
      </span>

      <!-- Interactive buttons -->
      <div v-if="interactiveButtons.length > 0" class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t">
        <div
          v-for="(btn, index) in interactiveButtons"
          :key="btn.id"
          :class="['py-2 text-sm text-center font-medium cursor-pointer', index > 0 && 'border-t']"
        >
          {{ btn.title }}
        </div>
      </div>

      <!-- CTA URL button -->
      <a
        v-if="ctaUrlData"
        :href="ctaUrlData.url"
        target="_blank"
        rel="noopener noreferrer"
        :aria-label="`Open: ${ctaUrlData.button_text}`"
        class="interactive-buttons mt-2 -mx-2 -mb-1.5 border-t block"
      >
        <div class="py-2 text-sm text-center font-medium cursor-pointer flex items-center justify-center gap-1.5">
          <ExternalLink class="h-3.5 w-3.5" />
          {{ ctaUrlData.button_text }}
        </div>
      </a>

      <!-- Time for messages without text content -->
      <span v-if="!messageContent && !(isMediaMessage && !hasMediaUrl)" class="chat-bubble-time block clear-both">
        <span>{{ formattedTime }}</span>
        <component v-if="message.direction === 'outgoing' && !isSystemEvent" :is="statusIcon" :class="['h-4 w-4 status-icon', statusClass]" />
      </span>

      <!-- Reactions -->
      <div v-if="message.reactions && message.reactions.length > 0" class="reactions-display flex flex-wrap gap-1 mt-1">
        <span v-for="(reaction, idx) in message.reactions" :key="idx" class="reaction-badge" :title="reaction.from_phone || reaction.from_user || ''">
          {{ reaction.emoji }}
        </span>
      </div>

      <!-- Failed message error -->
      <span
        v-if="message.status === 'failed' && message.direction === 'outgoing' && message.message_type !== 'template'"
        class="flex items-center gap-1 mt-1 text-xs text-destructive"
      >
        <AlertCircle class="h-3 w-3" />
        <span>{{ message.error_message || "Failed to send" }}</span>
      </span>
      <span
        v-if="message.status === 'failed' && message.direction === 'outgoing' && message.message_type === 'template'"
        class="flex items-center gap-1 mt-1 text-xs text-destructive"
      >
        <AlertCircle class="h-3 w-3" />
        <span>{{ message.error_message || "Failed to send" }}</span>
      </span>
    </div>

    <!-- Action buttons for incoming messages -->
    <div
      v-if="message.direction === 'incoming' && !isSystemEvent"
      role="group"
      :aria-label="$t('chat.messageActions')"
      class="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity self-center ml-1"
    >
      <Popover
        :open="reactionPickerMessageId === message.id"
        @update:open="(open: boolean) => open ? emit('open-reaction-picker', message.id) : emit('close-reaction-picker')"
      >
        <PopoverTrigger as-child>
          <Button variant="ghost" size="icon" class="h-6 w-6" :aria-label="$t('chat.addReaction')" :aria-expanded="reactionPickerMessageId === message.id"><SmilePlus class="h-3 w-3" /></Button>
        </PopoverTrigger>
        <PopoverContent side="top" class="w-auto p-2">
          <div class="flex gap-1" role="group" :aria-label="$t('chat.quickReactions')">
            <button v-for="emoji in quickReactionEmojis" :key="emoji" class="text-lg hover:bg-muted p-1 rounded cursor-pointer" :aria-label="`${$t('chat.addReaction')} ${emoji}`" @click="emit('send-reaction', message.id, emoji)">
              {{ emoji }}
            </button>
          </div>
        </PopoverContent>
      </Popover>
      <Button variant="ghost" size="icon" class="h-6 w-6" :aria-label="$t('chat.reply')" @click="emit('reply', message)">
        <Reply class="h-3 w-3" />
      </Button>
    </div>

    <!-- Action buttons for outgoing messages -->
    <div
      v-if="message.direction === 'outgoing' && !isSystemEvent"
      role="group"
      :aria-label="$t('chat.messageActions')"
      class="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity self-center ml-1"
    >
      <Popover
        :open="reactionPickerMessageId === message.id"
        @update:open="(open: boolean) => open ? emit('open-reaction-picker', message.id) : emit('close-reaction-picker')"
      >
        <PopoverTrigger as-child>
          <Button variant="ghost" size="icon" class="h-6 w-6" :aria-label="$t('chat.addReaction')" :aria-expanded="reactionPickerMessageId === message.id"><SmilePlus class="h-3 w-3" /></Button>
        </PopoverTrigger>
        <PopoverContent side="top" class="w-auto p-2">
          <div class="flex gap-1" role="group" :aria-label="$t('chat.quickReactions')">
            <button v-for="emoji in quickReactionEmojis" :key="emoji" class="text-lg hover:bg-muted p-1 rounded cursor-pointer" :aria-label="emoji" @click="emit('send-reaction', message.id, emoji)">
              {{ emoji }}
            </button>
          </div>
        </PopoverContent>
      </Popover>
      <Button variant="ghost" size="icon" class="h-6 w-6" :aria-label="$t('chat.reply')" @click="emit('reply', message)">
        <Reply class="h-3 w-3" />
      </Button>
      <Button
        v-if="canRevoke"
        variant="ghost"
        size="icon"
        class="h-6 w-6 text-destructive/80 hover:bg-destructive/10 hover:text-destructive"
        :disabled="isRevoking"
        :aria-label="$t('chat.revokeMessage')"
        title="Revoke message"
        @click="emit('revoke', message)"
      >
        <Loader2 v-if="isRevoking" class="h-3 w-3 animate-spin" />
        <Trash2 v-else class="h-3 w-3" />
      </Button>
      <Button
        v-if="message.status === 'failed' && message.message_type !== 'template'"
        variant="ghost"
        size="icon"
        class="h-6 w-6 text-destructive/80 hover:bg-destructive/10 hover:text-destructive"
        :disabled="isRetrying"
        :aria-label="$t('chat.retry')"
        @click="emit('retry', message)"
      >
        <Loader2 v-if="isRetrying" class="h-3 w-3 animate-spin" />
        <RotateCw v-else class="h-3 w-3" />
      </Button>
    </div>
  </div>
</template>
