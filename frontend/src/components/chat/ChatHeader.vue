<script setup lang="ts">
import { useI18n } from "vue-i18n";
import {
  Check,
  Loader2,
  Pin,
  Play,
  UserPlus,
  UserX,
  StickyNote,
} from "lucide-vue-next";
import { Info } from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getInitials, getAvatarGradient } from "@/lib/utils";
import type { Contact } from "@/stores/contacts";
import type { CustomAction } from "@/services/api";

defineProps<{
  contact: Contact;
  activeTransferId: string | null;
  isUpdatingPublic: boolean;
  isClaiming: boolean;
  isClosing: boolean;
  isTransferring: boolean;
  isResuming: boolean;
  isNotesPanelOpen: boolean;
  isInfoPanelOpen: boolean;
  notesCount: number;
  canAssign: boolean;
  canTogglePublic: boolean;
  canClaim: boolean;
  canClose: boolean;
  canManageTransfers: boolean;
  customActions: CustomAction[];
  executingActionId: string | null;
  executingActionLabel: string;
}>();

const emit = defineEmits<{
  "open-profile-photo": [contact: Contact];
  assign: [];
  "toggle-public": [];
  claim: [];
  close: [];
  transfer: [];
  resume: [];
  "execute-action": [action: CustomAction];
  "toggle-notes": [];
  "toggle-info": [];
}>();

const { t } = useI18n();

function getActionIcon(iconName: string) {
  const iconMap: Record<string, any> = {};
  return iconMap[iconName] || null;
}
</script>

<template>
  <div
    role="banner"
    :aria-label="t('chat.chatHeader')"
    class="flex h-14 flex-shrink-0 items-center justify-between border-b border-border bg-card/95 px-4 backdrop-blur"
  >
    <div class="flex items-center gap-2">
      <button
        type="button"
        class="rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        :aria-label="`${t('resources.ProfilePhoto')}: ${contact.name || contact.phone_number}`"
        @click="emit('open-profile-photo', contact)"
      >
        <Avatar class="h-8 w-8 ring-2 ring-border">
          <AvatarImage :src="contact.avatar_url" />
          <AvatarFallback
            :class="
              'text-xs bg-gradient-to-br text-white ' +
              getAvatarGradient(contact.name || contact.phone_number)
            "
          >
            {{ getInitials(contact.name || contact.phone_number) }}
          </AvatarFallback>
        </Avatar>
      </button>
      <div>
        <div class="flex items-center gap-1.5">
          <p class="text-sm font-medium text-foreground">
            {{ contact.name || contact.phone_number }}
          </p>
          <Badge
            v-if="contact.is_public"
            class="h-5 border-0 bg-primary/12 text-[10px] text-primary"
          >
            {{ $t("chat.publicChat") }}
          </Badge>
          <Badge
            v-if="contact.status === 'pending'"
            class="h-5 border-0 bg-accent text-[10px] text-accent-foreground"
          >
            Pending
          </Badge>
          <Badge
            v-if="contact.status === 'closed'"
            class="h-5 border-0 bg-muted text-[10px] text-muted-foreground"
          >
            Closed
          </Badge>
          <Badge
            v-if="activeTransferId"
            class="h-5 border-0 bg-accent text-[10px] text-primary"
          >
            Paused
          </Badge>
        </div>
        <p class="text-[11px] text-muted-foreground">
          {{ contact.phone_number }}
        </p>
      </div>
    </div>
    <div class="flex items-center gap-1">
      <Tooltip v-if="canAssign">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :aria-label="t('chat.assignToAgent')"
            @click="emit('assign')"
          >
            <UserPlus class="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.assignToAgent") }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="canTogglePublic">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :disabled="isUpdatingPublic"
            :aria-label="contact?.is_public ? t('chat.removePublicChat') : t('chat.makePublicChat')"
            @click="emit('toggle-public')"
          >
            <Loader2
              v-if="isUpdatingPublic"
              class="h-4 w-4 animate-spin"
            />
            <Pin
              v-else
              class="h-4 w-4"
              :class="contact?.is_public ? 'text-primary' : ''"
            />
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {{
            contact?.is_public
              ? $t("chat.removePublicChat")
              : $t("chat.makePublicChat")
          }}
        </TooltipContent>
      </Tooltip>
      <Tooltip v-if="canClaim">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :disabled="isClaiming"
            :aria-label="t('chat.claimChat')"
            @click="emit('claim')"
          >
            <Loader2 v-if="isClaiming" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.claimChat") }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="canClose">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :disabled="isClosing"
            :aria-label="t('chat.closeChat')"
            @click="emit('close')"
          >
            <Loader2 v-if="isClosing" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Close Chat</TooltipContent>
      </Tooltip>
      <Tooltip v-if="canManageTransfers && !activeTransferId">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :disabled="isTransferring"
            :aria-label="t('chat.transferToAgent')"
            @click="emit('transfer')"
          >
            <UserX class="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.transferToAgent") }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="canManageTransfers && activeTransferId">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :disabled="isResuming"
            :aria-label="t('chat.resumeChatbot')"
            @click="emit('resume')"
          >
            <Play class="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.resumeChatbot") }}</TooltipContent>
      </Tooltip>
      <Tooltip v-for="action in customActions" :key="action.id">
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :disabled="executingActionId === action.id"
            :aria-label="action.name"
            @click="emit('execute-action', action)"
          >
            <Loader2
              v-if="executingActionId === action.id"
              class="h-4 w-4 animate-spin"
            />
            <component
              v-else
              :is="getActionIcon(action.icon)"
              class="h-4 w-4"
            />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ action.name }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            id="notes-button"
            class="relative h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :class="isNotesPanelOpen && 'bg-accent text-accent-foreground'"
            :aria-label="t('chat.internalNotes')"
            @click="emit('toggle-notes')"
          >
            <StickyNote class="h-4 w-4" />
            <span
              v-if="notesCount > 0 && !isNotesPanelOpen"
              id="notes-badge"
              class="absolute -top-0.5 -right-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-primary px-1 text-[10px] text-primary-foreground"
            >
              {{ notesCount }}
            </span>
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.internalNotes") }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            variant="ghost"
            size="icon"
            id="info-button"
            class="h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground"
            :class="isInfoPanelOpen && 'bg-accent text-accent-foreground'"
            :aria-label="t('chat.contactInfo')"
            @click="emit('toggle-info')"
          >
            <Info class="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t("chat.contactInfo") }}</TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>
