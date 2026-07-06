<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { messagesService, notificationsService } from "@/services/api";
import { wsService } from "@/services/websocket";
import { useContactsStore, type Contact } from "@/stores/contacts";
import { useAuthStore } from "@/stores/auth";
import type { InstanceNotification } from "@/types/whatsmeow";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Bell, Loader2, X } from "lucide-vue-next";
import { toast } from "vue-sonner";

const props = withDefaults(
  defineProps<{
    compact?: boolean;
  }>(),
  {
    compact: false,
  },
);

const router = useRouter();
const { t } = useI18n();
const contactsStore = useContactsStore();
const authStore = useAuthStore();
const open = ref(false);
const loading = ref(false);
const isMarkingAllRead = ref(false);
const notifications = ref<InstanceNotification[]>([]);

const undismissedInstanceCount = computed(
  () => notifications.value.filter((item) => !item.is_dismissed).length,
);
const unreadConversations = computed(() => {
  const unread = contactsStore.sortedContacts.filter(
    (item) => (item.unread_count || 0) > 0,
  );
  const currentUserId = authStore.user?.id;
  const isAgent = authStore.userRole === "agent";

  if (!isAgent || !currentUserId) {
    return unread;
  }

  return unread.filter((item) => item.assigned_user_id === currentUserId);
});
const hasUnreadMessages = computed(() => unreadConversations.value.length > 0);
const unreadMessagesCount = computed(() =>
  unreadConversations.value.reduce(
    (total, item) => total + (item.unread_count || 0),
    0,
  ),
);
const totalUnreadCount = computed(
  () => undismissedInstanceCount.value + unreadMessagesCount.value,
);
const triggerButtonClass = computed(() =>
  props.compact
    ? "relative h-6 w-6 rounded-md text-sidebar-foreground/60 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
    : "relative h-8 w-8 text-muted-foreground hover:bg-accent hover:text-foreground",
);
const triggerIconClass = computed(() =>
  props.compact ? "h-3.5 w-3.5" : "h-4 w-4",
);
const triggerBadgeClass = computed(() =>
  props.compact
    ? "absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center border-0 bg-primary px-1 text-[9px] text-primary-foreground shadow-sm"
    : "absolute -top-1 -right-1 flex h-[18px] min-w-[18px] items-center justify-center border-0 bg-primary px-1 text-[10px] text-primary-foreground shadow-sm",
);

function normalizeNotifications(
  payload: unknown,
): InstanceNotification[] {
  if (Array.isArray(payload)) {
    return payload as InstanceNotification[];
  }

  if (payload && typeof payload === "object") {
    const notifications = (payload as { notifications?: unknown }).notifications;
    if (Array.isArray(notifications)) {
      return notifications as InstanceNotification[];
    }
  }

  return [];
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}

function resolveNotificationContactId(
  notification: InstanceNotification,
): string {
  if (notification.contact_id) return notification.contact_id;
  const metadataId = notification.metadata?.contact_id;
  return typeof metadataId === "string" ? metadataId : "";
}

function resolveNotificationChatLabel(
  notification: InstanceNotification,
): string {
  const name = String(notification.metadata?.contact_name || "").trim();
  const phone = String(notification.metadata?.contact_phone || "").trim();
  if (name && phone && name !== phone) {
    return `${name} (${phone})`;
  }
  return name || phone || t("chat.unknownChat");
}

function resolveNotificationActor(notification: InstanceNotification): string {
  const actor = String(notification.metadata?.actor_name || "").trim();
  return actor || t("chat.unknownUser");
}

function formatNotificationMessage(notification: InstanceNotification): string {
  if (notification.event_type === "chat_deleted_by_user") {
    return t("chat.chatDeletedByUserNotification", {
      user: resolveNotificationActor(notification),
      chat: resolveNotificationChatLabel(notification),
    });
  }
  return notification.message;
}

function handleNotificationClick(notification: InstanceNotification) {
  const contactId = resolveNotificationContactId(notification);
  if (!contactId) return;
  open.value = false;
  router.push(`/chat/${contactId}`);
}

function getContactLabel(contact: Contact): string {
  return contact.profile_name || contact.name || contact.phone_number;
}

function openConversation(contactId: string) {
  open.value = false;
  void contactsStore.markConversationAsRead(contactId);
  router.push(`/chat/${contactId}`);
}

async function fetchNotifications() {
  loading.value = true;
  try {
    const response = await notificationsService.list();
    const payload =
      response.data &&
      typeof response.data === "object" &&
      "data" in response.data
        ? response.data.data
        : response.data;
    notifications.value = normalizeNotifications(payload);
  } catch {
    // Silent fail for header widget
  } finally {
    loading.value = false;
  }
}

async function dismissNotification(id: string) {
  try {
    await notificationsService.dismiss(id);
    notifications.value = notifications.value.filter((item) => item.id !== id);
  } catch {
    toast.error("Failed to dismiss notification");
  }
}

async function markAllAsRead() {
  if (!hasUnreadMessages.value || isMarkingAllRead.value) return;

  isMarkingAllRead.value = true;
  const targetContacts = unreadConversations.value.map((item) => item.id);
  const targetSet = new Set(targetContacts);

  try {
    // Fetching a contact's messages triggers server-side mark-as-read flow for that conversation.
    await Promise.all(
      targetContacts.map((contactId) =>
        messagesService.list(contactId, { limit: 1 }),
      ),
    );

    for (const contact of contactsStore.contacts) {
      if (targetSet.has(contact.id)) {
        contact.unread_count = 0;
      }
    }
    if (
      contactsStore.currentContact &&
      targetSet.has(contactsStore.currentContact.id)
    ) {
      contactsStore.currentContact.unread_count = 0;
    }

    toast.success("All chats marked as read");
  } catch {
    toast.error("Failed to mark all chats as read");
  } finally {
    isMarkingAllRead.value = false;
  }
}

function handleInstanceNotification() {
  fetchNotifications();
}

onMounted(() => {
  fetchNotifications();
  if (contactsStore.contacts.length === 0) {
    contactsStore.fetchContacts();
  }
  wsService.subscribe("instance_notification", handleInstanceNotification);
});

onUnmounted(() => {
  wsService.unsubscribe("instance_notification", handleInstanceNotification);
});
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        variant="ghost"
        :size="props.compact ? 'icon-sm' : 'icon'"
        :class="triggerButtonClass"
        aria-label="Notifications"
        data-testid="notification-bell-button"
      >
        <Bell :class="triggerIconClass" />
        <Badge
          v-if="totalUnreadCount > 0"
          :class="triggerBadgeClass"
        >
          {{ totalUnreadCount > 99 ? "99+" : totalUnreadCount }}
        </Badge>
      </Button>
    </PopoverTrigger>
    <PopoverContent
      align="end"
      class="w-[360px] max-w-[92vw] overflow-hidden rounded-[calc(var(--radius)+0.35rem)] border border-border bg-card/98 p-0 text-card-foreground shadow-xl backdrop-blur-xl"
    >
      <div class="border-b border-border bg-card/95 px-4 py-3">
        <h3 class="text-sm font-medium text-foreground">Notifications</h3>
      </div>
      <div
        class="flex items-center gap-2 border-b border-border bg-muted/35 px-3 py-2"
      >
        <Button
          variant="ghost"
          size="sm"
          class="h-7 border border-primary/20 bg-primary/10 px-2 text-[11px] text-primary hover:bg-primary/20 hover:text-primary"
          :disabled="!hasUnreadMessages || isMarkingAllRead"
          @click="markAllAsRead"
        >
          <Loader2 v-if="isMarkingAllRead" class="h-3 w-3 mr-1 animate-spin" />
          Mark all as read
        </Button>
      </div>

      <div
        v-if="
          loading &&
          notifications.length === 0 &&
          unreadConversations.length === 0
        "
        class="flex items-center justify-center py-8"
      >
        <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
      </div>

      <div
        v-else-if="
          notifications.length === 0 && unreadConversations.length === 0
        "
        class="px-4 py-8 text-center text-sm text-muted-foreground"
      >
        No notifications
      </div>

      <ScrollArea v-else class="h-[360px]">
        <div class="bg-card">
          <div
            v-if="unreadConversations.length > 0"
            class="border-b border-border"
          >
            <div
              class="px-4 py-2 text-[11px] uppercase tracking-[0.18em] text-muted-foreground"
            >
              Unread Messages
            </div>
            <button
              v-for="contact in unreadConversations"
              :key="`chat-${contact.id}`"
              type="button"
              class="w-full border-t border-border/70 bg-card px-4 py-3 text-left transition-colors hover:bg-accent/80"
              @click="openConversation(contact.id)"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-foreground">
                    {{ getContactLabel(contact) }}
                  </p>
                  <p class="truncate text-xs text-muted-foreground/90">
                    {{ contact.last_message_preview || contact.phone_number }}
                  </p>
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{
                      formatDate(contact.last_message_at || contact.updated_at)
                    }}
                  </p>
                </div>
                <Badge
                  class="min-w-[18px] h-[18px] border-0 bg-primary px-1 text-[10px] text-primary-foreground"
                >
                  {{ contact.unread_count }}
                </Badge>
              </div>
            </button>
          </div>

          <div v-if="notifications.length > 0">
            <div
              class="px-4 py-2 text-[11px] uppercase tracking-[0.18em] text-muted-foreground"
            >
              System Notifications
            </div>
            <div class="divide-y divide-border">
              <button
                v-for="notification in notifications"
                :key="notification.id"
                type="button"
                class="flex w-full items-start gap-2 bg-card px-4 py-3 text-left"
                :class="
                  resolveNotificationContactId(notification)
                    ? 'transition-colors hover:bg-accent/80'
                    : ''
                "
                @click="handleNotificationClick(notification)"
              >
                <div class="flex-1 min-w-0">
                  <div
                    class="text-xs uppercase tracking-wide text-muted-foreground"
                  >
                    {{ notification.event_type.replace("_", " ") }}
                  </div>
                  <p class="mt-1 break-words text-sm text-foreground">
                    {{ formatNotificationMessage(notification) }}
                  </p>
                  <p class="mt-1 text-[11px] text-muted-foreground">
                    {{ formatDate(notification.created_at) }}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7 text-muted-foreground hover:bg-accent hover:text-foreground"
                  @click.stop="dismissNotification(notification.id)"
                >
                  <X class="h-3.5 w-3.5" />
                </Button>
              </button>
            </div>
          </div>
        </div>
      </ScrollArea>
    </PopoverContent>
  </Popover>
</template>
