import { defineStore } from "pinia";
import { ref } from "vue";
import type { ChatTypeFilter } from "@/types/contacts";

export const useChatFiltersStore = defineStore("chat-filters", () => {
  const searchQuery = ref("");
  const selectedTags = ref<string[]>([]);
  const selectedInstanceId = ref("");
  const selectedChatTypes = ref<ChatTypeFilter[]>([]);

  return {
    searchQuery,
    selectedTags,
    selectedInstanceId,
    selectedChatTypes,
  };
});
