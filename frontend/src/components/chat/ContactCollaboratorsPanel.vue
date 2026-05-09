<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { User, Plus, Trash2, UserPlus, Search } from "lucide-vue-next";
import { canUserAccessInstance } from "@/lib/instance-access";
import { useAuthStore } from "@/stores/auth";
import { useUsersStore } from "@/stores/users";
import { contactsService } from "@/services/api";
import { wsService } from "@/services/websocket";
import { toast } from "vue-sonner";

interface ContactCollaborator {
  id: string;
  contact_id: string;
  user_id: string;
  user_name?: string;
  role: string;
  status: string;
  invited_by_user_id: string;
  invited_by_name?: string;
  invited_at: string;
  accepted_at?: string | null;
}

const props = defineProps<{
  contactId: string;
  instanceId?: string;
}>();

const { t } = useI18n();
const authStore = useAuthStore();
const usersStore = useUsersStore();

const collaborators = ref<ContactCollaborator[]>([]);
const isLoadingCollaborators = ref(false);
const isInviteDialogOpen = ref(false);
const inviteSearchQuery = ref("");
const isInvitingCollaborator = ref(false);
const collaboratorActionId = ref<string | null>(null);

const canInviteCollaborators = computed(() =>
  authStore.hasPermission("chat.collaborators", "write"),
);
const currentUserId = computed(() => authStore.user?.id || "");

const inviteCandidates = computed(() => {
  const instanceId =
    typeof props.instanceId === "string" ? props.instanceId.trim() : "";
  const query = inviteSearchQuery.value.trim().toLowerCase();

  return usersStore.users
    .filter((user) => user.is_active !== false)
    .filter((user) => user.id !== currentUserId.value)
    .filter((user) => canUserAccessInstance(user, instanceId || undefined))
    .filter((user) => {
      const existing = collaborators.value.find((c) => c.user_id === user.id);
      if (!existing) return true;
      return existing.status === "declined";
    })
    .filter((user) => {
      if (!query) return true;
      const name = (user.full_name || "").toLowerCase();
      const email = (user.email || "").toLowerCase();
      return name.includes(query) || email.includes(query);
    });
});

const handleCollaboratorUpdate = (payload: any) => {
  if (!payload?.contact_id || payload.contact_id !== props.contactId) return;
  void fetchCollaborators();
};
const handleCollaboratorInvite = (payload: any) => {
  if (!payload?.contact_id || payload.contact_id !== props.contactId) return;
  void fetchCollaborators();
};

onMounted(async () => {
  await fetchCollaborators();
  wsService.subscribe("chat_collaborator_update", handleCollaboratorUpdate);
  wsService.subscribe("chat_collaborator_invite", handleCollaboratorInvite);
});

onUnmounted(() => {
  wsService.unsubscribe("chat_collaborator_update", handleCollaboratorUpdate);
  wsService.unsubscribe("chat_collaborator_invite", handleCollaboratorInvite);
});

watch(
  () => props.contactId,
  () => {
    void fetchCollaborators();
  },
);

watch(isInviteDialogOpen, (open) => {
  if (!open) return;
  if (usersStore.users.length > 0) return;
  usersStore.fetchUsers().catch(() => {});
});

async function fetchCollaborators() {
  if (!props.contactId) return;
  isLoadingCollaborators.value = true;
  try {
    const response = await contactsService.listCollaborators(props.contactId);
    const data = response.data.data || response.data;
    collaborators.value = Array.isArray(data.collaborators)
      ? data.collaborators
      : [];
  } catch (e: any) {
    collaborators.value = [];
  } finally {
    isLoadingCollaborators.value = false;
  }
}

async function inviteCollaborator(userId: string) {
  if (!props.contactId || isInvitingCollaborator.value) return;
  isInvitingCollaborator.value = true;
  try {
    await contactsService.inviteCollaborator(props.contactId, {
      user_id: userId,
    });
    toast.success(t("chat.collaboratorInvitedSuccess"));
    inviteSearchQuery.value = "";
    await fetchCollaborators();
  } catch (e: any) {
    toast.error(
      e.response?.data?.message || t("chat.collaboratorInviteFailed"),
    );
  } finally {
    isInvitingCollaborator.value = false;
  }
}

async function acceptInvite(collab: ContactCollaborator) {
  if (!props.contactId || collaboratorActionId.value) return;
  collaboratorActionId.value = collab.user_id;
  try {
    await contactsService.acceptCollaborator(props.contactId, collab.user_id);
    toast.success(t("chat.collaboratorAccepted"));
    await fetchCollaborators();
  } catch (e: any) {
    toast.error(
      e.response?.data?.message || t("chat.collaboratorAcceptFailed"),
    );
  } finally {
    collaboratorActionId.value = null;
  }
}

async function declineInvite(collab: ContactCollaborator) {
  if (!props.contactId || collaboratorActionId.value) return;
  collaboratorActionId.value = collab.user_id;
  try {
    await contactsService.declineCollaborator(props.contactId, collab.user_id);
    toast.success(t("chat.collaboratorDeclined"));
    await fetchCollaborators();
  } catch (e: any) {
    toast.error(
      e.response?.data?.message || t("chat.collaboratorDeclineFailed"),
    );
  } finally {
    collaboratorActionId.value = null;
  }
}

async function removeCollaborator(collab: ContactCollaborator) {
  if (!props.contactId || collaboratorActionId.value) return;
  collaboratorActionId.value = collab.user_id;
  try {
    await contactsService.removeCollaborator(props.contactId, collab.user_id);
    toast.success(t("chat.collaboratorRemoved"));
    await fetchCollaborators();
  } catch (e: any) {
    toast.error(
      e.response?.data?.message || t("chat.collaboratorRemoveFailed"),
    );
  } finally {
    collaboratorActionId.value = null;
  }
}
</script>

<template>
  <div class="pb-4 border-b">
    <div class="flex items-center justify-between py-2">
      <h5 class="text-sm font-medium flex items-center gap-2">
        <UserPlus class="h-4 w-4 text-muted-foreground" />
        {{ $t("chat.collaborators") }}
      </h5>
      <Button
        v-if="canInviteCollaborators"
        variant="ghost"
        size="sm"
        class="h-7 px-2"
        :aria-label="$t('chat.collaboratorInvite')"
        @click="isInviteDialogOpen = true"
      >
        <Plus class="h-3.5 w-3.5" />
      </Button>
    </div>

    <div v-if="isLoadingCollaborators" class="text-sm text-muted-foreground">
      {{ $t("chat.collaboratorLoading") }}
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="collab in collaborators"
        :key="collab.id"
        class="flex items-start justify-between gap-2 rounded-md border border-border/60 p-2"
      >
        <div class="min-w-0">
          <p class="text-sm font-medium truncate">
            {{ collab.user_name || collab.user_id }}
          </p>
          <p class="text-xs text-muted-foreground">
            {{ collab.role }} · {{ collab.status }}
          </p>
        </div>
        <div class="flex items-center gap-1">
          <Button
            v-if="collab.status === 'invited' && collab.user_id === currentUserId"
            variant="outline"
            size="sm"
            :disabled="collaboratorActionId === collab.user_id"
            :aria-label="`${$t('chat.collaboratorAccept')} ${collab.user_name || collab.user_id}`"
            @click="acceptInvite(collab)"
          >
            {{ $t("chat.collaboratorAccept") }}
          </Button>
          <Button
            v-if="collab.status === 'invited' && collab.user_id === currentUserId"
            variant="ghost"
            size="sm"
            :disabled="collaboratorActionId === collab.user_id"
            :aria-label="`${$t('chat.collaboratorDecline')} ${collab.user_name || collab.user_id}`"
            @click="declineInvite(collab)"
          >
            {{ $t("chat.collaboratorDecline") }}
          </Button>
          <Button
            v-if="collab.status === 'declined' && canInviteCollaborators"
            variant="outline"
            size="sm"
            :disabled="collaboratorActionId === collab.user_id"
            :aria-label="`${$t('chat.collaboratorReinvite')} ${collab.user_name || collab.user_id}`"
            @click="inviteCollaborator(collab.user_id)"
          >
            {{ $t("chat.collaboratorReinvite") }}
          </Button>
          <Button
            v-if="canInviteCollaborators || collab.user_id === currentUserId"
            variant="ghost"
            size="icon"
            class="h-7 w-7 text-destructive/80 hover:bg-destructive/10 hover:text-destructive"
            :disabled="collaboratorActionId === collab.user_id"
            :aria-label="`${$t('common.remove')} ${collab.user_name || collab.user_id}`"
            @click="removeCollaborator(collab)"
          >
            <Trash2 class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <p v-if="collaborators.length === 0" class="text-sm text-muted-foreground">
        {{ $t("chat.collaboratorNone") }}
      </p>
    </div>
  </div>

  <Dialog v-model:open="isInviteDialogOpen">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ $t("chat.collaboratorInvite") }}</DialogTitle>
        <DialogDescription>
          {{ $t("chat.collaboratorInviteDesc") }}
        </DialogDescription>
      </DialogHeader>
      <div class="py-4 space-y-3">
        <div class="relative">
          <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            v-model="inviteSearchQuery"
            :placeholder="$t('chat.collaboratorSearchPlaceholder')"
            class="pl-9 h-9"
          />
        </div>
        <ScrollArea class="max-h-[280px]">
          <div class="space-y-1">
            <Button
              v-for="user in inviteCandidates"
              :key="user.id"
              variant="ghost"
              class="w-full justify-start"
              :disabled="isInvitingCollaborator"
              @click="inviteCollaborator(user.id)"
            >
              <User class="mr-2 h-4 w-4" />
              <span>{{ user.full_name || user.email }}</span>
            </Button>
            <p
              v-if="inviteCandidates.length === 0"
              class="text-sm text-muted-foreground text-center py-4"
            >
              {{ $t("chat.collaboratorNoEligible") }}
            </p>
          </div>
        </ScrollArea>
      </div>
    </DialogContent>
  </Dialog>
</template>
