<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";
import { ChevronDown, Plus, RefreshCw } from "lucide-vue-next";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import NotificationBell from "@/components/NotificationBell.vue";
import {
  statusesService,
  type WhatsAppStatusGroup,
  type WhatsAppStatusesListPayload,
} from "@/services/api";
import StatusComposerDialog from "./StatusComposerDialog.vue";
import StatusViewerDialog from "./StatusViewerDialog.vue";

type InstanceOption = {
  id: string;
  name: string;
  status?: string;
};

const props = defineProps<{
  instances: InstanceOption[];
}>();

const { t } = useI18n();

const groups = ref<WhatsAppStatusGroup[]>([]);
const viewerGroups = ref<WhatsAppStatusGroup[]>([]);
const isLoading = ref(false);
const isComposerOpen = ref(false);
const isViewerOpen = ref(false);
let refreshTimer: ReturnType<typeof setInterval> | null = null;

const connectedInstances = computed(() =>
  props.instances.filter((instance) => {
    const status = String(instance.status || "").toLowerCase();
    return status === "connected" || status === "connecting";
  }),
);

const selfGroup = computed(
  () => groups.value.find((group) => group.is_self) || null,
);
const otherGroups = computed(() =>
  groups.value.filter((group) => !group.is_self),
);
const totalGroups = computed(() => groups.value.length);
const totalGroupsLabel = computed(() =>
  totalGroups.value > 9 ? "9+" : String(totalGroups.value),
);
const isDrawerOpen = ref(false);

function getGroupInstanceLabel(group: WhatsAppStatusGroup): string {
  const instanceName = String(group.instance_name || "").trim();
  if (instanceName) return instanceName;
  const instanceID = String(group.instance_id || "").trim();
  return instanceID ? `Instance ${instanceID.slice(0, 8)}` : "Unknown Instance";
}

function toggleDrawerFromBar(event: MouseEvent) {
  const target = event.target as HTMLElement | null;
  if (target?.closest('[data-status-interactive="true"]')) {
    return;
  }
  isDrawerOpen.value = !isDrawerOpen.value;
}

async function loadStatuses() {
  isLoading.value = true;
  try {
    const response = await statusesService.list();
    const payload: WhatsAppStatusesListPayload =
      "data" in response.data ? response.data.data : response.data;
    groups.value = Array.isArray(payload.groups) ? payload.groups : [];
  } catch (error: any) {
    const message =
      error?.response?.data?.message ||
      error?.message ||
      t("chat.statusLoadFailed");
    toast.error(message);
  } finally {
    isLoading.value = false;
  }
}

function openViewer(group: WhatsAppStatusGroup) {
  const remaining = groups.value.filter(
    (entry) => entry.group_id !== group.group_id,
  );
  viewerGroups.value = [group, ...remaining];
  isViewerOpen.value = true;
}

function openComposer() {
  if (connectedInstances.value.length === 0) {
    toast.error(t("chat.noConnectedInstances"));
    return;
  }
  isComposerOpen.value = true;
}

onMounted(() => {
  void loadStatuses();
  refreshTimer = setInterval(() => {
    void loadStatuses();
  }, 45000);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
});
</script>

<template>
  <div
    class="border-b border-sidebar-border px-2 py-2"
    data-testid="status-stories-bar"
    @click="toggleDrawerFromBar"
  >
    <Collapsible v-model:open="isDrawerOpen">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <p class="text-xs font-medium text-sidebar-foreground/65">
            {{ $t("chat.statuses") }}
          </p>
          <span
            v-if="totalGroups > 0"
            class="inline-grid h-6 w-6 shrink-0 place-items-center rounded-full bg-primary text-[9px] font-semibold leading-none text-primary-foreground shadow-sm"
            data-testid="status-groups-count"
          >
            {{ totalGroupsLabel }}
          </span>
        </div>
        <div class="flex items-center gap-1">
          <Button
            size="icon"
            variant="ghost"
            class="h-6 w-6 text-sidebar-foreground/60 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            :aria-label="$t('common.refresh')"
            data-status-interactive="true"
            @click="loadStatuses"
          >
            <RefreshCw
              class="h-3.5 w-3.5"
              :class="{ 'animate-spin': isLoading }"
            />
          </Button>
          <div data-status-interactive="true">
            <NotificationBell compact />
          </div>
          <CollapsibleTrigger
            class="inline-flex h-6 w-6 items-center justify-center rounded-md text-sidebar-foreground/60 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            :aria-label="$t('chat.statuses')"
            data-testid="status-drawer-toggle"
            data-status-interactive="true"
          >
            <ChevronDown
              class="h-3.5 w-3.5 transition-transform duration-200"
              :class="isDrawerOpen ? 'rotate-180' : ''"
            />
          </CollapsibleTrigger>
        </div>
      </div>

      <CollapsibleContent>
        <div class="mt-2 flex items-center gap-2 overflow-x-auto pb-1">
          <button
            class="group shrink-0"
            :aria-label="$t('chat.createStatus')"
            data-testid="status-create-button"
            data-status-interactive="true"
            @click="openComposer"
          >
            <div class="relative">
              <Avatar class="h-10 w-10 ring-2 ring-dashed ring-primary/70">
                <AvatarFallback class="bg-primary/12 text-primary">
                  <Plus class="h-4 w-4" />
                </AvatarFallback>
              </Avatar>
            </div>
            <p
              class="mt-1 w-12 truncate text-center text-[10px] text-sidebar-foreground/65"
            >
              {{ selfGroup ? $t("chat.myStatus") : $t("chat.addStatus") }}
            </p>
          </button>

          <button
            v-for="group in otherGroups"
            :key="group.group_id"
            class="group shrink-0"
            data-testid="status-story-button"
            data-status-interactive="true"
            :title="`${group.sender_name || group.sender_jid} • ${getGroupInstanceLabel(group)}`"
            @click="openViewer(group)"
          >
            <Avatar
              class="h-10 w-10 ring-2 ring-primary/75 ring-offset-1 ring-offset-transparent"
            >
              <AvatarFallback
                class="bg-sidebar-accent text-xs text-sidebar-foreground"
              >
                {{
                  (group.sender_name || group.sender_jid)
                    .slice(0, 1)
                    .toUpperCase()
                }}
              </AvatarFallback>
            </Avatar>
            <div class="mt-1 w-16 text-center leading-tight">
              <p class="truncate text-[10px] text-sidebar-foreground/80">
                {{ group.sender_name || group.sender_jid }}
              </p>
              <p class="truncate text-[9px] text-sidebar-foreground/55">
                {{ getGroupInstanceLabel(group) }}
              </p>
            </div>
          </button>
        </div>
      </CollapsibleContent>
    </Collapsible>

    <StatusComposerDialog
      v-model:open="isComposerOpen"
      :instances="connectedInstances"
      @submitted="loadStatuses"
    />
    <StatusViewerDialog
      v-model:open="isViewerOpen"
      :groups="viewerGroups"
      @refresh="loadStatuses"
    />
  </div>
</template>
