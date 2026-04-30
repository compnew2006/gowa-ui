<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { UserCheck } from "lucide-vue-next";
import { PageHeader } from "@/components/shared";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useContactsStore, type Contact } from "@/stores/contacts";

const { t } = useI18n();
const router = useRouter();
const contactsStore = useContactsStore();
const isLoading = ref(false);
const searchQuery = ref("");
const pageSize = 100;
const maxPages = 10;

const filteredAssignedChats = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return contactsStore.assignedChats;
  return contactsStore.assignedChats.filter(
    (chat) =>
      (chat.name || "").toLowerCase().includes(query) ||
      (chat.phone_number || "").toLowerCase().includes(query) ||
      (chat.assigned_user_id || "").toLowerCase().includes(query),
  );
});

function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString();
}

async function loadAssignedChats() {
  isLoading.value = true;
  try {
    for (let page = 1; page <= maxPages; page += 1) {
      const chats = await contactsStore.fetchAssignedChats({
        limit: pageSize,
        page,
        append: page > 1,
      });
      if (chats.length < pageSize) break;
    }
  } finally {
    isLoading.value = false;
  }
}

function openAssignedChat(chat: Contact) {
  router.push({
    name: "chat-conversation",
    params: { contactId: chat.id },
    query: { tab: "assigned" },
  });
}

onMounted(loadAssignedChats);
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="t('settings.assignedChatsTitle')"
      :subtitle="t('settings.assignedChatsSubtitle')"
      :icon="UserCheck"
      icon-gradient="bg-gradient-to-br from-blue-500 to-sky-600 shadow-blue-500/20"
    />

    <div class="p-6 space-y-4">
      <div class="flex items-center gap-2">
        <Input
          v-model="searchQuery"
          :placeholder="t('settings.searchAssignedChats')"
          class="max-w-md bg-white/[0.04] border-white/[0.1] text-white placeholder:text-white/40 light:bg-white light:border-gray-200 light:text-gray-900 light:placeholder:text-gray-400"
        />
        <Button
          variant="outline"
          :disabled="isLoading"
          @click="loadAssignedChats"
        >
          {{ isLoading ? t('common.refreshing') : t('common.refresh') }}
        </Button>
      </div>

      <div
        class="rounded-[calc(var(--radius)+0.25rem)] overflow-hidden border border-border bg-card/95 shadow-sm"
      >
        <table class="w-full text-sm">
          <thead class="bg-white/[0.04] light:bg-gray-50">
            <tr>
              <th
                class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700"
              >
                {{ t('settings.contactName') }}
              </th>
              <th
                class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700"
              >
                {{ t('settings.assignedUser') }}
              </th>
              <th
                class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700"
              >
                {{ t('settings.lastActivity') }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="isLoading">
              <td
                colspan="3"
                class="px-4 py-8 text-center text-white/50 light:text-gray-500"
              >
                {{ t('settings.loadingAssignedChats') }}
              </td>
            </tr>
            <tr
              v-for="chat in filteredAssignedChats"
              :key="chat.id"
              class="border-t border-white/[0.06] light:border-gray-100 hover:bg-white/[0.04] light:hover:bg-gray-50 cursor-pointer"
              @click="openAssignedChat(chat)"
            >
              <td class="px-4 py-3 text-white light:text-gray-900">
                <div class="font-medium">
                  {{ chat.name || chat.profile_name || chat.phone_number }}
                </div>
                <div class="text-xs text-white/50 light:text-gray-500">
                  {{ chat.phone_number }}
                </div>
              </td>
              <td class="px-4 py-3 text-white/80 light:text-gray-700">
                {{ chat.assigned_user_id || "—" }}
              </td>
              <td class="px-4 py-3 text-white/80 light:text-gray-700">
                {{ formatDate(chat.last_message_at || chat.updated_at) }}
              </td>
            </tr>
            <tr v-if="!isLoading && filteredAssignedChats.length === 0">
              <td
                colspan="3"
                class="px-4 py-8 text-center text-white/50 light:text-gray-500"
              >
                {{ t('settings.noAssignedChats') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
