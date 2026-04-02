<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { useNotesStore } from "@/stores/notes";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "vue-sonner";
import { useInfiniteScroll } from "@/composables/useInfiniteScroll";
import { getInitials, getAvatarGradient } from "@/lib/utils";
import { DeleteConfirmDialog } from "@/components/shared";
import {
  StickyNote,
  Pencil,
  Trash2,
  X,
  Check,
  Loader2,
  Send,
} from "lucide-vue-next";

const props = defineProps<{
  contactId: string;
}>();

const emit = defineEmits<{
  close: [];
}>();

const { t } = useI18n();
const notesStore = useNotesStore();
const authStore = useAuthStore();

const newNoteContent = ref("");
const editingNoteId = ref<string | null>(null);
const editingContent = ref("");
const isSaving = ref(false);
const deleteDialogOpen = ref(false);
const noteToDeleteId = ref<string | null>(null);
const notesEndRef = ref<HTMLElement | null>(null);

// Infinite scroll for older notes (scroll up to load more)
const notesScroll = useInfiniteScroll({
  direction: "top",
  onLoadMore: async () => {
    await notesScroll.preserveScrollPosition(async () => {
      await notesStore.fetchOlderNotes(props.contactId);
      await nextTick();
    });
  },
  hasMore: computed(() => notesStore.hasMore),
  isLoading: computed(() => notesStore.isLoadingOlder),
});

function scrollToBottom(instant = false) {
  nextTick(() => {
    if (notesEndRef.value) {
      notesEndRef.value.scrollIntoView({
        behavior: instant ? "instant" : "smooth",
        block: "end",
      });
    }
  });
}

onMounted(async () => {
  if (props.contactId) {
    // Only fetch if not already loaded for this contact (ChatView pre-fetches for badge count)
    if (notesStore.currentContactId !== props.contactId) {
      await notesStore.fetchNotes(props.contactId);
    }
    await nextTick();
    // Delay setup like messages do to ensure ScrollArea is fully rendered
    setTimeout(() => {
      scrollToBottom(true);
      notesScroll.setup();
    }, 50);
  }
});

watch(
  () => props.contactId,
  async (newId) => {
    if (newId) {
      notesScroll.cleanup();
      if (notesStore.currentContactId !== newId) {
        await notesStore.fetchNotes(newId);
      }
      await nextTick();
      setTimeout(() => {
        scrollToBottom(true);
        notesScroll.setup();
      }, 50);
    }
  },
);

// Auto-scroll when new notes are added at the bottom
watch(
  () => notesStore.notes.length,
  (_newLen, oldLen) => {
    if (oldLen !== undefined && _newLen > oldLen) {
      scrollToBottom();
    }
  },
);

async function addNote() {
  if (!newNoteContent.value.trim()) return;
  isSaving.value = true;
  try {
    await notesStore.createNote(props.contactId, newNoteContent.value.trim());
    newNoteContent.value = "";
    toast.success(t("chat.noteAdded"));
    scrollToBottom();
  } catch {
    toast.error(t("chat.noteAddFailed"));
  } finally {
    isSaving.value = false;
  }
}

function startEditing(noteId: string, content: string) {
  editingNoteId.value = noteId;
  editingContent.value = content;
}

function cancelEditing() {
  editingNoteId.value = null;
  editingContent.value = "";
}

async function saveEdit(noteId: string) {
  if (!editingContent.value.trim()) return;
  isSaving.value = true;
  try {
    await notesStore.updateNote(
      props.contactId,
      noteId,
      editingContent.value.trim(),
    );
    editingNoteId.value = null;
    editingContent.value = "";
    toast.success(t("chat.noteUpdated"));
  } catch {
    toast.error(t("chat.noteUpdateFailed"));
  } finally {
    isSaving.value = false;
  }
}

function requestDeleteNote(noteId: string) {
  noteToDeleteId.value = noteId;
  deleteDialogOpen.value = true;
}

function closeDeleteDialog() {
  deleteDialogOpen.value = false;
  noteToDeleteId.value = null;
}

async function confirmDeleteNote() {
  if (!noteToDeleteId.value) return;
  try {
    await notesStore.deleteNote(props.contactId, noteToDeleteId.value);
    toast.success(t("chat.noteDeleted"));
    closeDeleteDialog();
  } catch {
    toast.error(t("chat.noteDeleteFailed"));
  }
}

function formatNoteTime(dateStr: string) {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

function normalizeID(value: string | null | undefined): string {
  return (value || "").trim().toLowerCase();
}

function canManageNote(note: { created_by_id: string }): boolean {
  const noteCreatorID = normalizeID(note.created_by_id);
  const currentUserID = normalizeID(authStore.user?.id);
  return (
    (noteCreatorID !== "" && noteCreatorID === currentUserID) ||
    authStore.hasPermission("chat", "delete")
  );
}

watch(deleteDialogOpen, (open) => {
  if (!open) {
    noteToDeleteId.value = null;
  }
});
</script>

<template>
  <div
    id="notes-panel"
    class="flex w-80 flex-col border-l border-border bg-card/95 text-foreground"
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between border-b border-border px-4 py-3"
    >
      <div class="flex items-center gap-2">
        <div
          class="h-7 w-7 rounded-lg bg-amber-500/15 flex items-center justify-center"
        >
          <StickyNote class="h-4 w-4 text-primary" />
        </div>
        <span class="text-sm font-semibold text-foreground">{{
          t("chat.internalNotes")
        }}</span>
        <Badge
          v-if="notesStore.notes.length > 0"
          class="border-0 bg-primary/12 px-1.5 py-0 text-[10px] text-primary"
        >
          {{ notesStore.notes.length }}
        </Badge>
      </div>
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7 text-muted-foreground hover:bg-accent hover:text-foreground"
        @click="emit('close')"
      >
        <X class="h-4 w-4" />
      </Button>
    </div>

    <!-- Notes list -->
    <ScrollArea
      :ref="(el: any) => (notesScroll.scrollAreaRef.value = el)"
      class="flex-1 p-3"
    >
      <div class="space-y-3">
        <!-- Loading older notes -->
        <div v-if="notesStore.isLoadingOlder" class="flex justify-center py-2">
          <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
        </div>

        <!-- Initial loading state -->
        <div v-if="notesStore.isLoading" class="flex justify-center py-8">
          <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
        </div>

        <!-- Notes (chronological: oldest first, latest last) -->
        <template v-else-if="notesStore.notes.length > 0">
          <div
            v-for="note in notesStore.notes"
            :key="note.id"
            class="group relative rounded-[calc(var(--radius)+0.1rem)] border border-border bg-background/80 p-3 transition-all duration-200 hover:bg-accent/55"
          >
            <!-- Gradient accent line -->
            <div
              class="absolute top-0 left-3 right-3 h-[2px] rounded-full bg-gradient-to-r from-amber-500/60 via-orange-500/40 to-transparent"
            />

            <!-- Editing mode -->
            <template v-if="editingNoteId === note.id">
              <Textarea
                v-model="editingContent"
                class="mt-1 min-h-[60px] max-h-[100px] resize-none text-sm"
                :rows="2"
                @keydown.meta.enter.prevent="saveEdit(note.id)"
                @keydown.ctrl.enter.prevent="saveEdit(note.id)"
              />
              <div class="flex justify-end gap-1.5 mt-2">
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-7 text-xs"
                  @click="cancelEditing"
                >
                  {{ t("common.cancel") }}
                </Button>
                <Button
                  size="sm"
                  class="h-7 text-xs"
                  :disabled="!editingContent.trim() || isSaving"
                  @click="saveEdit(note.id)"
                >
                  <Loader2 v-if="isSaving" class="h-3 w-3 mr-1 animate-spin" />
                  <Check v-else class="h-3 w-3 mr-1" />
                  {{ t("common.save") }}
                </Button>
              </div>
            </template>

            <!-- Display mode -->
            <template v-else>
              <div class="mt-1 flex items-start gap-2.5">
                <Avatar class="h-6 w-6 shrink-0 ring-1 ring-border">
                  <AvatarFallback
                    :class="
                      'text-[10px] bg-gradient-to-br text-white ' +
                      getAvatarGradient(note.created_by_name)
                    "
                  >
                    {{ getInitials(note.created_by_name) }}
                  </AvatarFallback>
                </Avatar>
                <div class="flex-1 min-w-0">
                  <div class="mb-1 flex items-center justify-between">
                    <span class="text-xs font-medium text-foreground/80">{{
                      note.created_by_name
                    }}</span>
                    <span class="text-[10px] text-muted-foreground">{{
                      formatNoteTime(note.created_at)
                    }}</span>
                  </div>
                  <p
                    class="text-[13px] leading-relaxed whitespace-pre-wrap break-words text-foreground/80"
                  >
                    {{ note.content }}
                  </p>
                </div>
              </div>

              <!-- Note actions (creator or users with chat delete permission) -->
              <div
                v-if="canManageNote(note)"
                class="absolute top-2 right-2 flex gap-0.5 opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity"
              >
                <button
                  class="flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  @click="startEditing(note.id, note.content)"
                >
                  <Pencil class="h-3 w-3" />
                </button>
                <button
                  class="flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                  @click="requestDeleteNote(note.id)"
                >
                  <Trash2 class="h-3 w-3" />
                </button>
              </div>
            </template>
          </div>
        </template>

        <!-- Empty state -->
        <div
          v-else
          class="flex flex-col items-center justify-center py-12 text-center"
        >
          <div
            class="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10"
          >
            <StickyNote class="h-6 w-6 text-primary/60" />
          </div>
          <p class="mb-1 text-sm font-medium text-muted-foreground">
            {{ t("chat.noNotes") }}
          </p>
          <p class="text-xs text-muted-foreground/80">
            {{ t("chat.writeNote") }}
          </p>
        </div>

        <!-- Scroll anchor -->
        <div ref="notesEndRef" />
      </div>
    </ScrollArea>

    <!-- Add note input -->
    <div class="border-t border-border p-4">
      <div
        class="flex items-center gap-2 rounded-xl border border-border bg-background/80 p-2"
      >
        <textarea
          v-model="newNoteContent"
          :placeholder="t('chat.writeNote') + '...'"
          class="min-h-[36px] max-h-[120px] flex-1 resize-none overflow-y-auto bg-transparent py-2 text-[14px] text-foreground placeholder:text-muted-foreground focus:outline-none"
          rows="1"
          @keydown.enter.exact.prevent="addNote"
        />
        <button
          class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          :disabled="!newNoteContent.trim() || isSaving"
          @click="addNote"
        >
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin text-white" />
          <Send v-else class="h-4 w-4 text-white" />
        </button>
      </div>
    </div>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="t('common.delete')"
      :description="t('chat.confirmDeleteNote')"
      :confirm-label="t('common.delete')"
      :cancel-label="t('common.cancel')"
      @confirm="confirmDeleteNote"
      @cancel="closeDeleteDialog"
    />
  </div>
</template>
