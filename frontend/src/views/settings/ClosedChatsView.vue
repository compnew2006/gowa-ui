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
  Search,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useDebounceFn } from "@vueuse/core";
import { PageHeader } from "@/components/shared";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
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

const pageSizeOptions = [25, 50, 100];

// Persisted filter preferences so navigating away and back (e.g. opening a
// closed chat then returning) keeps the user's active filters. Mirrors the
// localStorage pattern used by AgentAnalyticsView/TemplatesView.
const FILTERS_STORAGE_KEY = "closed_chats_filters";

interface SavedClosedChatFilters {
  searchQuery?: string;
  selectedAgentId?: string;
  selectedInstanceId?: string;
  closedFrom?: string;
  closedTo?: string;
  pageSize?: string;
}

function loadSavedFilters(): SavedClosedChatFilters {
  try {
    const raw = localStorage.getItem(FILTERS_STORAGE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as SavedClosedChatFilters;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    // Malformed or unavailable storage; fall back to defaults silently.
    return {};
  }
}

const savedFilters = loadSavedFilters();
const isValidPageSize = (value: string | undefined) =>
  !!value && pageSizeOptions.includes(Number(value));

const searchQuery = ref(savedFilters.searchQuery ?? "");
const selectedAgentId = ref(savedFilters.selectedAgentId ?? "all");
const selectedInstanceId = ref(savedFilters.selectedInstanceId ?? "all");
const agentComboboxOpen = ref(false);
const closedFrom = ref(savedFilters.closedFrom ?? "");
const closedTo = ref(savedFilters.closedTo ?? "");
const currentPage = ref(1);
const pageSize = ref(
  isValidPageSize(savedFilters.pageSize) ? savedFilters.pageSize! : "25",
);
const totalClosedChats = ref(0);

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
const hasActiveFilters = computed(
  () =>
    searchQuery.value.trim() !== "" ||
    selectedAgentId.value !== "all" ||
    selectedInstanceId.value !== "all" ||
    closedFrom.value !== "" ||
    closedTo.value !== "",
);

function saveFilters() {
  try {
    const payload: SavedClosedChatFilters = {
      searchQuery: searchQuery.value,
      selectedAgentId: selectedAgentId.value,
      selectedInstanceId: selectedInstanceId.value,
      closedFrom: closedFrom.value,
      closedTo: closedTo.value,
      pageSize: pageSize.value,
    };
    localStorage.setItem(FILTERS_STORAGE_KEY, JSON.stringify(payload));
  } catch {
    // Storage unavailable or full; persistence is best-effort.
  }
}

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

function clearFilters() {
  searchQuery.value = "";
  selectedAgentId.value = "all";
  selectedInstanceId.value = "all";
  closedFrom.value = "";
  closedTo.value = "";
  // The watchers will persist the cleared state; no explicit save needed.
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
  } catch {
    toast.error(t("closedChats.loadAgentsFailed"));
  }
}

async function loadInstances() {
  try {
    await instancesStore.fetchInstances();
  } catch {
    toast.error(t("closedChats.loadInstancesFailed"));
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
    saveFilters();
    debouncedFilterLoad();
  },
);

watch(pageSize, () => {
  saveFilters();
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
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('closedChats.title')"
      :subtitle="$t('closedChats.subtitle')"
      :icon="Archive"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
      back-link="/whatsapp"
      :breadcrumbs="[
        { label: $t('nav.whatsappTools'), href: '/whatsapp' },
        { label: $t('closedChats.title') },
      ]"
    >
      <template #actions>
        <Button variant="outline" :disabled="isLoading" @click="loadClosedChats">
          <Loader2 v-if="isLoading" class="me-2 h-4 w-4 animate-spin" />
          <RotateCw v-else class="me-2 h-4 w-4" />
          {{ isLoading ? $t("closedChats.refreshing") : $t("closedChats.refresh") }}
        </Button>
      </template>
    </PageHeader>

    <div class="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
      <div class="mx-auto max-w-6xl space-y-5">
        <section class="rounded-xl border bg-card p-4 shadow-sm">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div class="space-y-1">
              <h2 class="text-sm font-semibold text-foreground">
                {{ $t("closedChats.filters") }}
              </h2>
              <p class="text-xs leading-5 text-muted-foreground">
                {{ $t("closedChats.filtersDesc") }}
              </p>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              :disabled="!hasActiveFilters"
              @click="clearFilters"
            >
              {{ $t("closedChats.clearFilters") }}
            </Button>
          </div>

          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-12">
            <div class="space-y-2 xl:col-span-3">
              <Label for="closed-chats-search">
                {{ $t("closedChats.search") }}
              </Label>
              <div class="relative">
                <Search class="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="closed-chats-search"
                  v-model="searchQuery"
                  name="closed-chats-search"
                  :placeholder="$t('closedChats.searchPlaceholder')"
                  class="bg-background ps-9"
                  data-testid="closed-chats-search"
                />
              </div>
            </div>

            <div class="space-y-2 xl:col-span-2">
              <Label for="closed-chats-agent-filter">
                {{ $t("closedChats.closedBy") }}
              </Label>
              <Popover v-model:open="agentComboboxOpen">
                <PopoverTrigger as-child>
                  <Button
                    id="closed-chats-agent-filter"
                    variant="outline"
                    role="combobox"
                    :aria-expanded="agentComboboxOpen"
                    class="h-10 w-full justify-between bg-background"
                    data-testid="closed-chats-agent-filter"
                  >
                    <span class="truncate text-start">{{ selectedAgentName }}</span>
                    <ChevronsUpDown class="ms-2 h-4 w-4 shrink-0 opacity-50" />
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
                              'me-2 h-4 w-4',
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
                              'me-2 h-4 w-4',
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
            </div>

            <div class="space-y-2 xl:col-span-2">
              <Label for="closed-chats-instance-filter">
                {{ $t("chat.instance") }}
              </Label>
              <Select v-model="selectedInstanceId">
                <SelectTrigger
                  id="closed-chats-instance-filter"
                  class="h-10 bg-background"
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
            </div>

            <div class="space-y-2 xl:col-span-2">
              <Label for="closed-chats-date-from">
                {{ $t("closedChats.dateFrom") }}
              </Label>
              <Input
                id="closed-chats-date-from"
                v-model="closedFrom"
                name="closed-chats-date-from"
                type="date"
                class="bg-background"
                data-testid="closed-chats-date-from"
              />
            </div>

            <div class="space-y-2 xl:col-span-2">
              <Label for="closed-chats-date-to">
                {{ $t("closedChats.dateTo") }}
              </Label>
              <Input
                id="closed-chats-date-to"
                v-model="closedTo"
                name="closed-chats-date-to"
                type="date"
                class="bg-background"
                data-testid="closed-chats-date-to"
              />
            </div>

            <div class="space-y-2 xl:col-span-1">
              <Label for="closed-chats-page-size">
                {{ $t("closedChats.pageSize") }}
              </Label>
              <Select v-model="pageSize">
                <SelectTrigger
                  id="closed-chats-page-size"
                  class="h-10 bg-background"
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
          </div>

          <p
            v-if="hasInvalidDateRange"
            class="mt-3 text-xs font-medium text-destructive"
          >
            {{ $t("closedChats.invalidDateRange") }}
          </p>
        </section>

        <div class="flex flex-wrap items-center justify-between gap-3">
          <Badge variant="secondary" class="rounded-full px-3 py-1">
            {{
              $t("closedChats.resultsSummary", {
                start: pageStart,
                end: pageEnd,
                total: totalClosedChats,
              })
            }}
          </Badge>
          <span class="text-xs text-muted-foreground" data-testid="closed-chats-page-summary">
            {{
              $t("closedChats.pageLabel", {
                page: currentPage,
                total: totalPages,
              })
            }}
          </span>
        </div>

        <div
          class="overflow-hidden rounded-xl border border-border bg-card shadow-sm"
        >
        <div class="max-h-[62vh] overflow-auto" data-testid="closed-chats-scroll">
          <table class="w-full text-sm">
            <thead class="sticky top-0 z-10 bg-muted/60 backdrop-blur">
              <tr>
                <th class="px-4 py-3 text-start font-medium text-muted-foreground">
                  {{ $t("closedChats.contactName") }}
                </th>
                <th class="px-4 py-3 text-start font-medium text-muted-foreground">
                  {{ $t("closedChats.closedBy") }}
                </th>
                <th class="px-4 py-3 text-start font-medium text-muted-foreground">
                  {{ $t("closedChats.dateClosed") }}
                </th>
                <th class="px-4 py-3 text-end font-medium text-muted-foreground">
                  {{ $t("closedChats.actions") }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="isLoading">
                <td colspan="4" class="px-4 py-12 text-center text-muted-foreground">
                  <div class="flex items-center justify-center gap-2">
                    <Loader2 class="h-4 w-4 animate-spin" />
                    {{ $t("closedChats.loading") }}
                  </div>
                </td>
              </tr>
              <tr
                v-for="chat in contactsStore.closedChats"
                :key="chat.id"
                class="cursor-pointer border-t border-border/60 transition-colors hover:bg-accent/35"
                @click="openReadOnlyChat(chat)"
              >
                <td class="px-4 py-3 text-foreground">
                  <div class="font-medium">
                    {{ chat.name || chat.profile_name || chat.phone_number }}
                  </div>
                  <div class="text-xs text-muted-foreground">
                    {{ chat.phone_number }}
                  </div>
                </td>
                <td class="px-4 py-3 text-foreground/85">
                  {{ getClosedByLabel(chat) }}
                </td>
                <td class="px-4 py-3 text-foreground/85">
                  {{ formatClosedDate(chat.closed_at || chat.updated_at) }}
                </td>
                <td class="px-4 py-3 text-end">
                  <Button
                    size="sm"
                    class="h-8 px-3"
                    :disabled="reopeningChatId === chat.id"
                    @click.stop="reopenChat(chat)"
                  >
                    <Loader2
                      v-if="reopeningChatId === chat.id"
                      class="me-1.5 h-3 w-3 animate-spin"
                    />
                    <RotateCw v-else class="me-1.5 h-3 w-3" />
                    {{ $t("closedChats.reopen") }}
                  </Button>
                </td>
              </tr>
              <tr v-if="!isLoading && contactsStore.closedChats.length === 0">
                <td colspan="4" class="px-4 py-14 text-center">
                  <div class="mx-auto flex max-w-sm flex-col items-center gap-3 text-muted-foreground">
                    <span class="rounded-full border bg-muted/40 p-3">
                      <Archive class="h-5 w-5" />
                    </span>
                    <div class="space-y-1">
                      <p class="font-medium text-foreground">
                        {{ $t("closedChats.empty") }}
                      </p>
                      <p class="text-xs leading-5">
                        {{ $t("closedChats.emptyDesc") }}
                      </p>
                    </div>
                  </div>
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
        <span class="text-xs text-muted-foreground" data-testid="closed-chats-page-label">
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
  </div>
</template>
