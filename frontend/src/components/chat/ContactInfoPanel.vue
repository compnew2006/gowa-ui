<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  X,
  Phone,
  Loader2,
  Trash2,
  Archive,
} from "lucide-vue-next";
import { getInitials, getAvatarGradient } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { useTagsStore } from "@/stores/tags";
import { contactsService } from "@/services/api";
import { localeDirectionManager } from "@/i18n/locale-direction";
import { toast } from "vue-sonner";
import type { Contact } from "@/stores/contacts";

import ContactTagsPanel from "@/components/chat/ContactTagsPanel.vue";
import ContactCollaboratorsPanel from "@/components/chat/ContactCollaboratorsPanel.vue";
import ContactMetadataPanel from "@/components/chat/ContactMetadataPanel.vue";
import ContactSessionDataPanel from "@/components/chat/ContactSessionDataPanel.vue";

interface PanelFieldConfig {
  key: string;
  label: string;
  order: number;
  display_type?: "text" | "badge" | "tag";
  color?: "default" | "success" | "warning" | "error" | "info";
}

interface PanelSection {
  id: string;
  label: string;
  columns: number;
  collapsible: boolean;
  default_collapsed: boolean;
  order: number;
  fields: PanelFieldConfig[];
}

interface PanelConfig {
  sections: PanelSection[];
}

interface SessionData {
  session_id?: string;
  flow_id?: string;
  flow_name?: string;
  session_data: Record<string, any>;
  panel_config: PanelConfig;
}

const props = defineProps<{
  contact: Contact;
  sessionData?: SessionData | null;
}>();

const emit = defineEmits<{
  close: [];
  tagsUpdated: [tags: string[]];
  deleted: [contactId: string];
}>();

const authStore = useAuthStore();
const tagsStore = useTagsStore();
const { locale, t } = useI18n();
const isRTL = computed(() =>
  localeDirectionManager.isRTL(String(locale.value)),
);

const contactTags = computed(() => {
  if (!props.contact.tags || !Array.isArray(props.contact.tags)) return [];
  return props.contact.tags as string[];
});

const canEditTags = computed(() =>
  authStore.hasPermission("contacts", "write"),
);
const canDeleteChats = computed(() =>
  authStore.hasPermission("contacts", "delete"),
);
const canSoftDeleteChats = computed(() =>
  authStore.hasPermission("contacts", "soft_delete"),
);
const isDeletingChat = ref(false);
const isSoftDeletingChat = ref(false);

const MIN_WIDTH = 280;
const MAX_WIDTH = 500;
const panelWidth = ref(MAX_WIDTH);
const isResizing = ref(false);

onMounted(async () => {
  if (tagsStore.tags.length === 0) {
    try {
      await tagsStore.fetchTags();
    } catch (e) {
      // Silently fail - tags just won't be available
    }
  }
});

function startResize(e: MouseEvent) {
  isResizing.value = true;
  const startX = e.clientX;
  const startWidth = panelWidth.value;
  const isRTLDirection = isRTL.value;

  function onMouseMove(e: MouseEvent) {
    const delta = isRTLDirection ? e.clientX - startX : startX - e.clientX;
    const newWidth = Math.min(
      MAX_WIDTH,
      Math.max(MIN_WIDTH, startWidth + delta),
    );
    panelWidth.value = newWidth;
  }

  function onMouseUp() {
    isResizing.value = false;
    document.removeEventListener("mousemove", onMouseMove);
    document.removeEventListener("mouseup", onMouseUp);
  }

  document.addEventListener("mousemove", onMouseMove);
  document.addEventListener("mouseup", onMouseUp);
}

async function deleteChat() {
  if (isDeletingChat.value || !canDeleteChats.value) return;

  const confirmed = window.confirm("Delete this chat conversation?");
  if (!confirmed) return;

  isDeletingChat.value = true;
  try {
    await contactsService.delete(props.contact.id);
    toast.success("Chat deleted");
    emit("deleted", props.contact.id);
  } catch (e: any) {
    toast.error(e.response?.data?.message || "Failed to delete chat");
  } finally {
    isDeletingChat.value = false;
  }
}

async function softDeleteChat() {
  if (isSoftDeletingChat.value || !canSoftDeleteChats.value) return;

  const confirmed = window.confirm(t("chat.softDeleteConfirm"));
  if (!confirmed) return;

  isSoftDeletingChat.value = true;
  try {
    await contactsService.softDelete(props.contact.id);
    toast.success(t("chat.softDeleteSuccess"));
    emit("deleted", props.contact.id);
  } catch (e: any) {
    toast.error(e.response?.data?.message || t("chat.softDeleteFailed"));
  } finally {
    isSoftDeletingChat.value = false;
  }
}
</script>

<template>
  <div
    class="flex flex-col bg-card h-full relative"
    :style="{ width: `${panelWidth}px` }"
  >
    <!-- Resize Handle -->
    <div
      class="absolute top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/20 active:bg-primary/30 z-10"
      :class="[
        isRTL ? 'right-0 border-r' : 'left-0 border-l',
        { 'bg-primary/30': isResizing },
      ]"
      @mousedown="startResize"
    />

    <!-- Header -->
    <div class="h-12 px-3 border-b flex items-center justify-between">
      <h3 class="font-medium text-sm">{{ $t("chat.contactInfo") }}</h3>
      <div class="flex items-center gap-1">
        <Button
          v-if="canSoftDeleteChats"
          variant="ghost"
          size="icon"
          class="h-8 w-8 border border-primary/20 bg-primary/10 text-primary hover:bg-primary/20 hover:text-primary"
          :disabled="isSoftDeletingChat"
          :aria-label="$t('chat.softDeleteChat')"
          @click="softDeleteChat"
        >
          <Loader2 v-if="isSoftDeletingChat" class="h-4 w-4 animate-spin" />
          <Archive v-else class="h-4 w-4" />
        </Button>
        <Button
          v-if="canDeleteChats"
          variant="ghost"
          size="icon"
          class="h-8 w-8 border border-destructive/20 bg-destructive/10 text-destructive hover:bg-destructive/20 hover:text-destructive"
          :disabled="isDeletingChat"
          :aria-label="$t('common.delete')"
          @click="deleteChat"
        >
          <Loader2 v-if="isDeletingChat" class="h-4 w-4 animate-spin" />
          <Trash2 v-else class="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="h-8 w-8"
          :aria-label="$t('common.close')"
          @click="emit('close')"
        >
          <X class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <ScrollArea class="flex-1">
      <div class="p-4 space-y-4">
        <!-- Contact Header -->
        <div class="flex flex-col items-center text-center pb-4 border-b">
          <Avatar class="h-16 w-16 mb-3">
            <AvatarImage :src="contact.avatar_url" />
            <AvatarFallback
              :class="
                'text-lg bg-gradient-to-br text-white ' +
                getAvatarGradient(contact.name || contact.phone_number)
              "
            >
              {{ getInitials(contact.name || contact.phone_number) }}
            </AvatarFallback>
          </Avatar>
          <h4 class="font-medium">
            {{ contact.name || contact.phone_number }}
          </h4>
          <div
            class="flex items-center gap-1 text-sm text-muted-foreground mt-1"
          >
            <Phone class="h-3 w-3" />
            <span>{{ contact.phone_number }}</span>
          </div>
        </div>

        <!-- Tags -->
        <ContactTagsPanel
          :contact-id="contact.id"
          :contact-tags="contactTags"
          :can-edit-tags="canEditTags"
          @tags-updated="(tags) => emit('tagsUpdated', tags)"
        />

        <!-- Collaborators -->
        <ContactCollaboratorsPanel
          :contact-id="contact.id"
          :instance-id="contact.instance_id"
        />

        <!-- Contact Metadata -->
        <ContactMetadataPanel :metadata="contact.metadata" />

        <!-- Session Data -->
        <ContactSessionDataPanel :session-data="sessionData" />
      </div>
    </ScrollArea>
  </div>
</template>
