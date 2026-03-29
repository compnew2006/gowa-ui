<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  Archive,
  Check,
  ChevronsUpDown,
  Loader2,
  RotateCw,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useDebounceFn } from "@vueuse/core";
import { PageHeader } from "@/components/shared";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { useContactsStore, type Contact } from "@/stores/contacts";
import { useInstancesStore } from "@/stores/instances";
import { useUsersStore } from "@/stores/users";

const { t } = useI18n();
const router = useRouter();
const contactsStore = useContactsStore();
const instancesStore = useInstancesStore();
const usersStore = useUsersStore();
const isLoading = ref(false);
const reopeningChatId = ref<string | null>(null);
const searchQuery = ref("");
const selectedAgentId = ref("all");
const selectedInstanceId = ref("all");
const agentComboboxOpen = ref(false);
const closedFrom = ref("");
const closedTo = ref("");
const currentPage = ref(1);
const pageSize = ref("25");
const totalClosedChats = ref(0);
const pageSizeOptions = [25, 50, 100];

interface AgentOption {
  id: string;
  full_name: string;
}

const agents = ref<AgentOption[]>([]);
const availableInstances = computed(() =>
  [...instancesStore.instances].sort((left, right) =>
    left.name.localeCompare(right.name),
  ),
);

const selectedAgentName = computed(() => {
  if (selectedAgentId.value === "all") return t("closedChats.allAgents");
  const agent = agents.value.find((item) => item.id === selectedAgentId.value);
  return agent?.full_name || t("closedChats.selectAgent");
});

const pageSizeValue = computed(() => Number(pageSize.value) || 25);
const totalPages = computed(() =>
  Math.max(1, Math.ceil(totalClosedChats.value / pageSizeValue.value)),
);
const pageStart = computed(() => {
  if (totalClosedChats.value === 0) return 0;
  return (currentPage.value - 1) * pageSizeValue.value + 1;
});
const pageEnd = computed(() => {
  if (totalClosedChats.value === 0) return 0;
  return Math.min(
    currentPage.value * pageSizeValue.value,
    totalClosedChats.value,
  );
});

const hasInvalidDateRange = computed(
  () =>
    closedFrom.value !== "" &&
    closedTo.value !== "" &&
    closedFrom.value > closedTo.value,
);

function formatClosedDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString();
}

function getClosedByLabel(chat: Contact): string {
  if (chat.closed_by_name && chat.closed_by_name.trim())
    return chat.closed_by_name;
  if (chat.closed_by_user_id) return chat.closed_by_user_id;
  if (chat.assigned_user_id) return chat.assigned_user_id;
  return "—";
}

async function loadClosedChats() {
  if (hasInvalidDateRange.value) {
    toast.error(t("closedChats.invalidDateRange"));
    return;
  }

  isLoading.value = true;
  try {
    const response = await contactsStore.fetchClosedChats({
      search: searchQuery.value.trim() || undefined,
      closed_by:
        selectedAgentId.value !== "all" ? selectedAgentId.value : undefined,
      instance_id:
        selectedInstanceId.value !== "all"
          ? selectedInstanceId.value
          : undefined,
      closed_from: closedFrom.value || undefined,
      closed_to: closedTo.value || undefined,
      page: currentPage.value,
      limit: pageSizeValue.value,
    });

    totalClosedChats.value = response.total;

    if (
      response.chats.length === 0 &&
      response.total > 0 &&
      currentPage.value > 1
    ) {
      currentPage.value -= 1;
      await loadClosedChats();
      return;
    }
  } finally {
    isLoading.value = false;
  }
}

async function loadAgents() {
  try {
    await usersStore.fetchUsers({ page: 1, limit: 500 });
    agents.value = usersStore.users
      .map((user) => ({ id: user.id, full_name: user.full_name }))
      .sort((left, right) => left.full_name.localeCompare(right.full_name));
  } catch (error) {
    console.error("Failed to load users for closed chats filter:", error);
  }
}

async function loadInstances() {
  try {
    await instancesStore.fetchInstances();
  } catch (error) {
    console.error("Failed to load instances for closed chats filter:", error);
  }
}

function openReadOnlyChat(chat: Contact) {
  router.push({
    name: "chat-conversation",
    params: { contactId: chat.id },
    query: { tab: "assigned" },
  });
}

async function reopenChat(chat: Contact) {
  if (reopeningChatId.value) return;
  reopeningChatId.value = chat.id;
  try {
    const updated = await contactsStore.reopenChat(chat.id);
    if (!updated) {
      toast.error(t("closedChats.reopenFailed"));
      return;
    }
    toast.success(t("closedChats.reopenedSuccess"));
    await loadClosedChats();
    router.push({
      name: "chat-conversation",
      params: { contactId: updated.id },
      query: { tab: "pending" },
    });
  } catch (error: any) {
    const message =
      error?.response?.data?.message || t("closedChats.reopenFailed");
    toast.error(message);
  } finally {
    reopeningChatId.value = null;
  }
}

function goToPreviousPage() {
  if (currentPage.value <= 1) return;
  currentPage.value -= 1;
  void loadClosedChats();
}

function goToNextPage() {
  if (currentPage.value >= totalPages.value) return;
  currentPage.value += 1;
  void loadClosedChats();
}

const debouncedFilterLoad = useDebounceFn(() => {
  if (currentPage.value !== 1) {
    currentPage.value = 1;
  }
  void loadClosedChats();
}, 350);

watch(
  [searchQuery, selectedAgentId, selectedInstanceId, closedFrom, closedTo],
  () => {
    debouncedFilterLoad();
  },
);

watch(pageSize, () => {
  if (currentPage.value !== 1) {
    currentPage.value = 1;
  }
  void loadClosedChats();
});

onMounted(() => {
  void loadInstances();
  void loadAgents();
  void loadClosedChats();
});
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('closedChats.title')"
      :subtitle="$t('closedChats.subtitle')"
      :icon="Archive"
      icon-gradient="bg-gradient-to-br from-zinc-500 to-zinc-700 shadow-zinc-500/20"
    />

    <div class="p-6 space-y-4">
      <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-7">
        <Input
          v-model="searchQuery"
          :placeholder="$t('closedChats.searchPlaceholder')"
          class="xl:col-span-2 bg-white/[0.04] border-white/[0.1] text-white placeholder:text-white/40 light:bg-white light:border-gray-200 light:text-gray-900 light:placeholder:text-gray-400"
          data-testid="closed-chats-search"
        />
        <Popover v-model:open="agentComboboxOpen">
          <PopoverTrigger as-child>
            <Button
              variant="outline"
              role="combobox"
              :aria-expanded="agentComboboxOpen"
              class="h-10 justify-between bg-white/[0.04] border-white/[0.1] text-white hover:bg-white/[0.08] light:bg-white light:border-gray-200 light:text-gray-900"
              data-testid="closed-chats-agent-filter"
            >
              <span class="truncate text-left">{{ selectedAgentName }}</span>
              <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-[var(--radix-popover-trigger-width)] p-0">
            <Command>
              <CommandInput
                :placeholder="$t('closedChats.searchAgent')"
                data-testid="closed-chats-agent-search"
              />
              <CommandList>
                <CommandEmpty>{{
                  $t("closedChats.noAgentFound")
                }}</CommandEmpty>
                <CommandGroup>
                  <CommandItem
                    value="all"
                    data-testid="closed-chats-agent-option-all"
                    @select="
                      () => {
                        selectedAgentId = 'all';
                        agentComboboxOpen = false;
                      }
                    "
                  >
                    <Check
                      :class="[
                        'mr-2 h-4 w-4',
                        selectedAgentId === 'all' ? 'opacity-100' : 'opacity-0',
                      ]"
                    />
                    {{ $t("closedChats.allAgents") }}
                  </CommandItem>
                  <CommandItem
                    v-for="agent in agents"
                    :key="agent.id"
                    :value="`${agent.full_name} ${agent.id}`"
                    :data-testid="`closed-chats-agent-option-${agent.id}`"
                    @select="
                      () => {
                        selectedAgentId = agent.id;
                        agentComboboxOpen = false;
                      }
                    "
                  >
                    <Check
                      :class="[
                        'mr-2 h-4 w-4',
                        selectedAgentId === agent.id
                          ? 'opacity-100'
                          : 'opacity-0',
                      ]"
                    />
                    <div class="flex min-w-0 flex-col">
                      <span class="truncate">{{ agent.full_name }}</span>
                      <span class="truncate text-xs text-muted-foreground">{{
                        agent.id
                      }}</span>
                    </div>
                  </CommandItem>
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
        <Select v-model="selectedInstanceId">
          <SelectTrigger
            class="h-10 bg-white/[0.04] border-white/[0.1] text-white light:bg-white light:border-gray-200 light:text-gray-900"
            data-testid="closed-chats-instance-filter"
          >
            <SelectValue :placeholder="$t('chat.filterByInstance')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{{ $t("chat.allInstances") }}</SelectItem>
            <SelectItem
              v-for="instance in availableInstances"
              :key="instance.id"
              :value="instance.id"
            >
              {{ instance.name }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Input
          v-model="closedFrom"
          type="date"
          :placeholder="$t('closedChats.dateFrom')"
          class="bg-white/[0.04] border-white/[0.1] text-white placeholder:text-white/40 light:bg-white light:border-gray-200 light:text-gray-900 light:placeholder:text-gray-400"
          data-testid="closed-chats-date-from"
        />
        <Input
          v-model="closedTo"
          type="date"
          :placeholder="$t('closedChats.dateTo')"
          class="bg-white/[0.04] border-white/[0.1] text-white placeholder:text-white/40 light:bg-white light:border-gray-200 light:text-gray-900 light:placeholder:text-gray-400"
          data-testid="closed-chats-date-to"
        />
        <Select v-model="pageSize">
          <SelectTrigger
            class="h-10 bg-white/[0.04] border-white/[0.1] text-white light:bg-white light:border-gray-200 light:text-gray-900"
            data-testid="closed-chats-page-size"
          >
            <SelectValue :placeholder="$t('closedChats.pageSize')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="size in pageSizeOptions"
              :key="size"
              :value="String(size)"
            >
              {{ size }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div class="flex items-center justify-between gap-2">
        <p class="text-xs text-white/60 light:text-gray-500">
          {{
            $t("closedChats.resultsSummary", {
              start: pageStart,
              end: pageEnd,
              total: totalClosedChats,
            })
          }}
        </p>
        <Button
          variant="outline"
          @click="loadClosedChats"
          :disabled="isLoading"
        >
          {{
            isLoading ? $t("closedChats.refreshing") : $t("closedChats.refresh")
          }}
        </Button>
      </div>

      <div
        class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200 overflow-hidden"
      >
        <div
          class="max-h-[62vh] overflow-auto"
          data-testid="closed-chats-scroll"
        >
          <table class="w-full text-sm">
            <thead class="bg-white/[0.04] light:bg-gray-50 sticky top-0 z-10">
              <tr>
                <th
                  class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700"
                >
                  {{ $t("closedChats.contactName") }}
                </th>
                <th
                  class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700"
                >
                  {{ $t("closedChats.closedBy") }}
                </th>
                <th
                  class="text-left px-4 py-3 font-medium text-white/70 light:text-gray-700"
                >
                  {{ $t("closedChats.dateClosed") }}
                </th>
                <th
                  class="text-right px-4 py-3 font-medium text-white/70 light:text-gray-700"
                >
                  {{ $t("closedChats.actions") }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="isLoading">
                <td
                  colspan="4"
                  class="px-4 py-8 text-center text-white/50 light:text-gray-500"
                >
                  {{ $t("closedChats.loading") }}
                </td>
              </tr>
              <tr
                v-for="chat in contactsStore.closedChats"
                :key="chat.id"
                class="border-t border-white/[0.06] light:border-gray-100 hover:bg-white/[0.04] light:hover:bg-gray-50 cursor-pointer"
                @click="openReadOnlyChat(chat)"
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
                  {{ getClosedByLabel(chat) }}
                </td>
                <td class="px-4 py-3 text-white/80 light:text-gray-700">
                  {{ formatClosedDate(chat.closed_at || chat.updated_at) }}
                </td>
                <td class="px-4 py-3 text-right">
                  <Button
                    size="sm"
                    variant="outline"
                    class="h-7 px-2.5 bg-white/[0.04] border-white/[0.12] text-white/80 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50"
                    :disabled="reopeningChatId === chat.id"
                    @click.stop="reopenChat(chat)"
                  >
                    <Loader2
                      v-if="reopeningChatId === chat.id"
                      class="mr-1.5 h-3 w-3 animate-spin"
                    />
                    <RotateCw v-else class="mr-1.5 h-3 w-3" />
                    {{ $t("closedChats.reopen") }}
                  </Button>
                </td>
              </tr>
              <tr v-if="!isLoading && contactsStore.closedChats.length === 0">
                <td
                  colspan="4"
                  class="px-4 py-8 text-center text-white/50 light:text-gray-500"
                >
                  {{ $t("closedChats.empty") }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="flex items-center justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          data-testid="closed-chats-prev-page"
          :disabled="isLoading || currentPage <= 1"
          @click="goToPreviousPage"
        >
          {{ $t("closedChats.previous") }}
        </Button>
        <span
          class="text-xs text-white/70 light:text-gray-600"
          data-testid="closed-chats-page-label"
        >
          {{
            $t("closedChats.pageLabel", {
              page: currentPage,
              total: totalPages,
            })
          }}
        </span>
        <Button
          variant="outline"
          size="sm"
          data-testid="closed-chats-next-page"
          :disabled="
            isLoading || currentPage >= totalPages || totalClosedChats === 0
          "
          @click="goToNextPage"
        >
          {{ $t("closedChats.next") }}
        </Button>
      </div>
    </div>
  </div>
</template>
