<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { toast } from "vue-sonner";
import {
  Bot,
  Check,
  Clock3,
  Eye,
  Loader2,
  Pencil,
  Plus,
  RefreshCcw,
  Save,
  Trash2,
  UserPlus,
  X,
} from "lucide-vue-next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { PageHeader, DeleteConfirmDialog } from "@/components/shared";
import { useAgentSelectionStore } from "@/stores/agentSelection";
import { useInstancesStore } from "@/stores/instances";
import { useTeamsStore } from "@/stores/teams";
import { useUsersStore } from "@/stores/users";
import { getErrorMessage } from "@/lib/api-utils";
import type {
  AgentSelectionCustomAction,
  AgentSelectionOptionType,
  AgentSelectionSettings,
} from "@/services/api";
import type { User } from "@/types/auth";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const agentSelectionStore = useAgentSelectionStore();
const instancesStore = useInstancesStore();
const usersStore = useUsersStore();
const teamsStore = useTeamsStore();

const isLoading = ref(false);
const isSaving = ref(false);
const isAddingParticipant = ref(false);
const isAddingOption = ref(false);
const isTestSending = ref(false);
const isDeletingRule = ref(false);
const editingParticipantId = ref("");
const isSavingParticipantEdit = ref(false);
const testSendContactId = ref("");
const selectedRuleProfile = ref("__global__");

const participantDeleteDialogOpen = ref(false);
const participantToDelete = ref<{ id: string; name: string } | null>(null);
const optionDeleteDialogOpen = ref(false);
const optionToDelete = ref<{ id: string; name: string } | null>(null);
const selectedTab = ref("settings");

const settingsForm = reactive({
  allowed_instance_ids: [] as string[],
  enabled: false,
  trigger_mode: "first_pending_message",
  trigger_keywords: "",
  prompt_delay_minutes: 3,
  prompt_delay_min_minutes: 3,
  prompt_delay_max_minutes: 3,
  selection_timeout_minutes: 10,
  max_invalid_attempts: 3,
  menu_header_text: "من فضلك اختر من تريد التواصل معه:",
  menu_footer_text: "",
  invalid_reply_text: "اختيار غير صحيح. من فضلك ارسل رقم من القائمة.",
  timeout_response_text: "",
  unavailable_agent_text:
    "هذا الوكيل غير متاح حاليا. من فضلك اختر رقم آخر أو انتظر أحد ممثلينا.",
  custom_final_option_enabled: false,
  custom_final_option_text: "سأذهب للفرع للطباعة",
  custom_final_option_response: "تمام، يسعدنا خدمتك في الفرع.",
  custom_final_option_action: "keep_pending" as AgentSelectionCustomAction,
  custom_final_option_team_id: "",
  hide_unavailable_agents: true,
});

const participantForm = reactive({
  user_id: "",
  display_name: "",
  description: "",
  sort_order: 0,
  is_enabled: true,
  show_only_when_available: true,
  max_open_chats: "",
});

const participantEditForm = reactive({
  display_name: "",
  description: "",
  sort_order: 0,
  max_open_chats: "",
});

const optionForm = reactive({
  option_type: "team" as AgentSelectionOptionType,
  team_id: "",
  label: "",
  description: "",
  sort_order: 50,
  is_enabled: true,
  action: "",
});

const settingsId = computed(() => agentSelectionStore.settings?.id || "");
const activeRuleInstanceId = computed(() =>
  selectedRuleProfile.value === "__global__" ? "" : selectedRuleProfile.value,
);
const canConfigureLists = computed(
  () => settingsId.value && settingsId.value !== "00000000-0000-0000-0000-000000000000",
);
const availableUsers = computed(() => {
  const used = new Set(
    agentSelectionStore.participants.map((participant) => participant.user_id),
  );
  return usersStore.users.filter(
    (user) => user.is_active !== false && !used.has(user.id),
  );
});
const teamOptions = computed(() => teamsStore.teams.filter((team) => team.is_active));
const availableInstances = computed(() => instancesStore.instances);
const activeRuleInstance = computed(() =>
  availableInstances.value.find((instance) => instance.id === activeRuleInstanceId.value),
);
const activeRuleProfileLabel = computed(() => {
  if (!activeRuleInstanceId.value) return "Global/default rule";
  return activeRuleInstance.value?.name || "Selected instance rule";
});
const activeRuleIsSaved = computed(() => canConfigureLists.value);
const removeRuleLabel = computed(() =>
  activeRuleInstanceId.value ? "Remove instance rule" : "Remove global rule",
);
const selectedInstanceCount = computed(
  () => settingsForm.allowed_instance_ids.length,
);
const allInstancesSelected = computed(
  () =>
    availableInstances.value.length > 0 &&
    selectedInstanceCount.value === availableInstances.value.length,
);
const selectedInstanceScopeLabel = computed(() => {
  if (activeRuleInstanceId.value) {
    return activeRuleInstance.value?.name || "Selected instance";
  }
  const selectedCount = selectedInstanceCount.value;
  if (selectedCount === 0) return "All WhatsMeow instances";
  if (selectedCount === 1) return "1 selected instance";
  return `${selectedCount} selected instances`;
});
const selectedInstanceScopeHelp = computed(() => {
  if (activeRuleInstanceId.value) {
    return "This rule applies only to the selected WhatsMeow instance.";
  }
  if (selectedInstanceCount.value === 0) {
    return "Customer routing will listen on every WhatsMeow instance.";
  }
  return "Customer routing will only listen on the selected instances.";
});
const routingStatusLabel = computed(() =>
  settingsForm.enabled
    ? t("agentSelection.page.status.active")
    : t("agentSelection.page.status.paused"),
);
const routingStatusHelp = computed(() =>
  settingsForm.enabled
    ? t("agentSelection.page.status.activeHelp")
    : t("agentSelection.page.status.pausedHelp"),
);
const triggerModeLabel = computed(() =>
  t(`agentSelection.page.trigger.${settingsForm.trigger_mode}`),
);
const timingSummaryLabel = computed(
  () => {
    const minDelay = Number(settingsForm.prompt_delay_min_minutes) || 0;
    const maxDelay = Number(settingsForm.prompt_delay_max_minutes) || minDelay;
    const delayLabel =
      minDelay === maxDelay
        ? `${minDelay} min delay`
        : `${minDelay}-${maxDelay} min delay`;
    return `${delayLabel}, ${settingsForm.selection_timeout_minutes} min timeout`;
  },
);
const breadcrumbs = [
  { label: t("nav.settings"), href: "/settings" },
  { label: t("agentSelection.page.title") },
];

watch(
  () => agentSelectionStore.settings,
  (settings) => {
    if (settings) {
      applySettings(settings);
    }
  },
);

watch(
  () => participantForm.user_id,
  (userId) => {
    const user = usersStore.users.find((item) => item.id === userId);
    if (user && !participantForm.display_name.trim()) {
      participantForm.display_name = displayUserName(user);
    }
  },
);

onMounted(async () => {
  await loadPage();
});

async function loadPage() {
  isLoading.value = true;
  try {
    await Promise.all([
      instancesStore.fetchInstances(),
      usersStore.fetchUsers({ limit: 200 }),
      teamsStore.fetchTeams({ limit: 200 }),
    ]);
    await loadRuleSettings();
    await Promise.all([
      agentSelectionStore.fetchSessions(),
      agentSelectionStore.fetchAudit(),
    ]);
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to load customer routing"));
  } finally {
    isLoading.value = false;
  }
}

async function loadRuleSettings() {
  const settings = await agentSelectionStore.fetchSettings(
    activeRuleInstanceId.value || undefined,
  );
  applySettings(settings);
  if (settings.id && settings.id !== "00000000-0000-0000-0000-000000000000") {
    await Promise.all([
      agentSelectionStore.fetchParticipants(settings.id),
      agentSelectionStore.fetchOptions(settings.id),
      fetchPreviewForSettings(settings.id),
    ]);
  } else {
    agentSelectionStore.participants = [];
    agentSelectionStore.options = [];
    agentSelectionStore.preview = null;
  }
  return settings;
}

async function changeRuleProfile(value: unknown) {
  const nextValue = String(value || "__global__");
  if (nextValue === selectedRuleProfile.value) return;
  selectedRuleProfile.value = nextValue;
  isLoading.value = true;
  try {
    await loadRuleSettings();
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to load instance routing rule"));
  } finally {
    isLoading.value = false;
  }
}

function applySettings(settings: AgentSelectionSettings) {
  settingsForm.allowed_instance_ids = [...(settings.allowed_instance_ids || [])];
  settingsForm.enabled = settings.enabled;
  settingsForm.trigger_mode = settings.trigger_mode;
  settingsForm.trigger_keywords = (settings.trigger_keywords || []).join(", ");
  settingsForm.prompt_delay_minutes = settings.prompt_delay_minutes;
  settingsForm.prompt_delay_min_minutes =
    settings.prompt_delay_min_minutes ?? settings.prompt_delay_minutes;
  settingsForm.prompt_delay_max_minutes =
    settings.prompt_delay_max_minutes ?? settings.prompt_delay_minutes;
  settingsForm.selection_timeout_minutes = settings.selection_timeout_minutes;
  settingsForm.max_invalid_attempts = settings.max_invalid_attempts;
  settingsForm.menu_header_text = settings.menu_header_text || "";
  settingsForm.menu_footer_text = settings.menu_footer_text || "";
  settingsForm.invalid_reply_text = settings.invalid_reply_text || "";
  settingsForm.timeout_response_text = settings.timeout_response_text || "";
  settingsForm.unavailable_agent_text = settings.unavailable_agent_text || "";
  settingsForm.custom_final_option_enabled =
    settings.custom_final_option_enabled;
  settingsForm.custom_final_option_text = settings.custom_final_option_text || "";
  settingsForm.custom_final_option_response =
    settings.custom_final_option_response || "";
  settingsForm.custom_final_option_action =
    settings.custom_final_option_action || "keep_pending";
  settingsForm.custom_final_option_team_id =
    settings.custom_final_option_team_id || "";
  settingsForm.hide_unavailable_agents = settings.hide_unavailable_agents;
}

async function saveSettings() {
  isSaving.value = true;
  try {
    await persistSettings();
    toast.success("Customer routing settings saved");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to save settings"));
  } finally {
    isSaving.value = false;
  }
}

async function removeCurrentRule() {
  const id = settingsId.value;
  if (!id || id === "00000000-0000-0000-0000-000000000000") {
    toast.info("This rule has not been saved yet");
    return;
  }
  const confirmed = window.confirm(
    activeRuleInstanceId.value
      ? "Remove this instance rule? Its agents and options will be deleted, and the instance will use the global/default rule."
      : "Remove the global/default rule? Its agents and options will be deleted, and a fresh default rule will be created when needed.",
  );
  if (!confirmed) return;

  isDeletingRule.value = true;
  try {
    await agentSelectionStore.deleteSettings(id);
    toast.success("Routing rule removed");
    if (activeRuleInstanceId.value) {
      selectedRuleProfile.value = "__global__";
    }
    await loadRuleSettings();
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to remove routing rule"));
  } finally {
    isDeletingRule.value = false;
  }
}

async function addParticipant() {
  if (!canConfigureLists.value) {
    try {
      await persistSettings();
    } catch (error) {
      toast.error(getErrorMessage(error, "Failed to save settings"));
      return;
    }
  }
  if (!canConfigureLists.value) {
    toast.error("Save customer routing settings before adding agents");
    return;
  }
  if (!participantForm.user_id) {
    toast.error("Choose an agent first");
    return;
  }
  isAddingParticipant.value = true;
  try {
    await agentSelectionStore.createParticipant({
      settings_id: settingsId.value,
      user_id: participantForm.user_id,
      display_name: participantForm.display_name.trim(),
      description: participantForm.description.trim(),
      sort_order: Number(participantForm.sort_order) || 0,
      is_enabled: participantForm.is_enabled,
      show_only_when_available: participantForm.show_only_when_available,
      max_open_chats: participantForm.max_open_chats
        ? Number(participantForm.max_open_chats)
        : null,
    });
    resetParticipantForm();
    await fetchPreviewForSettings(settingsId.value);
    toast.success("Agent added to customer list");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to add agent"));
  } finally {
    isAddingParticipant.value = false;
  }
}

async function toggleParticipant(id: string, isEnabled: boolean) {
  await agentSelectionStore.updateParticipant(id, { is_enabled: isEnabled });
  await fetchPreviewForSettings(settingsId.value);
}

function startEditParticipant(participant: {
  id: string;
  display_name: string;
  description?: string;
  sort_order: number;
  max_open_chats?: number | null;
}) {
  editingParticipantId.value = participant.id;
  participantEditForm.display_name = participant.display_name || "";
  participantEditForm.description = participant.description || "";
  participantEditForm.sort_order = Number(participant.sort_order) || 0;
  participantEditForm.max_open_chats =
    participant.max_open_chats === null || participant.max_open_chats === undefined
      ? ""
      : String(participant.max_open_chats);
}

function cancelEditParticipant() {
  editingParticipantId.value = "";
  participantEditForm.display_name = "";
  participantEditForm.description = "";
  participantEditForm.sort_order = 0;
  participantEditForm.max_open_chats = "";
}

async function saveParticipantEdit(id: string) {
  if (!participantEditForm.display_name.trim()) {
    toast.error("Customer display name is required");
    return;
  }
  isSavingParticipantEdit.value = true;
  try {
    await agentSelectionStore.updateParticipant(id, {
      display_name: participantEditForm.display_name.trim(),
      description: participantEditForm.description.trim(),
      sort_order: Number(participantEditForm.sort_order) || 0,
      max_open_chats: participantEditForm.max_open_chats
        ? Number(participantEditForm.max_open_chats)
        : null,
    });
    cancelEditParticipant();
    await fetchPreviewForSettings(settingsId.value);
    toast.success("Agent updated");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to update agent"));
  } finally {
    isSavingParticipantEdit.value = false;
  }
}

function removeParticipant(participant: { id: string; display_name: string }) {
  participantToDelete.value = { id: participant.id, name: participant.display_name };
  participantDeleteDialogOpen.value = true;
}

async function confirmRemoveParticipant() {
  if (!participantToDelete.value) return;
  const id = participantToDelete.value.id;
  try {
    await agentSelectionStore.deleteParticipant(id);
    await fetchPreviewForSettings(settingsId.value);
    toast.success("Agent removed");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to remove agent"));
  } finally {
    participantDeleteDialogOpen.value = false;
    participantToDelete.value = null;
  }
}

async function addOption() {
  if (!canConfigureLists.value) {
    try {
      await persistSettings();
    } catch (error) {
      toast.error(getErrorMessage(error, "Failed to save settings"));
      return;
    }
  }
  if (!canConfigureLists.value) {
    toast.error("Save customer routing settings before adding options");
    return;
  }
  if (!optionForm.label.trim()) {
    toast.error("Option label is required");
    return;
  }
  if (optionForm.option_type === "team" && !optionForm.team_id) {
    toast.error("Choose a team for this option");
    return;
  }
  isAddingOption.value = true;
  try {
    await agentSelectionStore.createOption({
      settings_id: settingsId.value,
      option_type: optionForm.option_type,
      team_id: optionForm.option_type === "team" ? optionForm.team_id : null,
      user_id: null,
      label: optionForm.label.trim(),
      description: optionForm.description.trim(),
      sort_order: Number(optionForm.sort_order) || 0,
      is_enabled: optionForm.is_enabled,
      action: optionForm.action.trim(),
    });
    resetOptionForm();
    await fetchPreviewForSettings(settingsId.value);
    toast.success("Routing option added");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to add option"));
  } finally {
    isAddingOption.value = false;
  }
}

async function toggleOption(id: string, isEnabled: boolean) {
  await agentSelectionStore.updateOption(id, { is_enabled: isEnabled });
  await fetchPreviewForSettings(settingsId.value);
}

function removeOption(option: { id: string; label: string }) {
  optionToDelete.value = { id: option.id, name: option.label };
  optionDeleteDialogOpen.value = true;
}

async function confirmRemoveOption() {
  if (!optionToDelete.value) return;
  const id = optionToDelete.value.id;
  try {
    await agentSelectionStore.deleteOption(id);
    await fetchPreviewForSettings(settingsId.value);
    toast.success("Option removed");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to remove option"));
  } finally {
    optionDeleteDialogOpen.value = false;
    optionToDelete.value = null;
  }
}

async function refreshOps() {
  await Promise.all([
    agentSelectionStore.fetchSessions(),
    agentSelectionStore.fetchAudit(),
  ]);
}

async function cancelSession(id: string) {
  try {
    await agentSelectionStore.cancelSession(id);
    toast.success("Session cancelled");
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to cancel session"));
  }
}

const canTestSend = computed(
  () => !isTestSending.value && testSendContactId.value.trim().length > 0,
);

async function runTestSend() {
  if (!canTestSend.value) return;
  isTestSending.value = true;
  try {
    const result = await agentSelectionStore.testSend(
      testSendContactId.value.trim(),
      agentSelectionStore.settings?.id,
    );
    toast.success(
      t("agentSelection.page.testSend.sent", {
        contact: result.contact_id,
        account: result.whatsapp_account,
      }),
    );
  } catch (error) {
    toast.error(
      t("agentSelection.page.testSend.failed", {
        error: getErrorMessage(error, "unknown error"),
      }),
    );
  } finally {
    isTestSending.value = false;
  }
}

function resetParticipantForm() {
  participantForm.user_id = "";
  participantForm.display_name = "";
  participantForm.description = "";
  participantForm.sort_order = 0;
  participantForm.is_enabled = true;
  participantForm.show_only_when_available = true;
  participantForm.max_open_chats = "";
}

function resetOptionForm() {
  optionForm.option_type = "team";
  optionForm.team_id = "";
  optionForm.label = "";
  optionForm.description = "";
  optionForm.sort_order = 50;
  optionForm.is_enabled = true;
  optionForm.action = "";
}

function splitKeywords(value: string) {
  return value
    .split(",")
    .map((keyword) => keyword.trim())
    .filter(Boolean);
}

async function persistSettings() {
  const saved = await agentSelectionStore.saveSettings(buildSettingsPayload());
  await Promise.all([
    agentSelectionStore.fetchParticipants(saved.id),
    agentSelectionStore.fetchOptions(saved.id),
    fetchPreviewForSettings(saved.id),
  ]);
  return saved;
}

async function fetchPreviewForSettings(id: string) {
  try {
    return await agentSelectionStore.fetchPreview(id);
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to load menu preview"));
    return null;
  }
}

function buildSettingsPayload(): Partial<AgentSelectionSettings> {
  return {
    instance_id: activeRuleInstanceId.value || null,
    enabled: settingsForm.enabled,
    allowed_instance_ids: activeRuleInstanceId.value
      ? []
      : settingsForm.allowed_instance_ids,
    trigger_mode: settingsForm.trigger_mode as AgentSelectionSettings["trigger_mode"],
    trigger_keywords: splitKeywords(settingsForm.trigger_keywords),
    prompt_delay_minutes: settingsForm.prompt_delay_min_minutes,
    prompt_delay_min_minutes: settingsForm.prompt_delay_min_minutes,
    prompt_delay_max_minutes: settingsForm.prompt_delay_max_minutes,
    selection_timeout_minutes: settingsForm.selection_timeout_minutes,
    max_invalid_attempts: settingsForm.max_invalid_attempts,
    menu_header_text: settingsForm.menu_header_text,
    menu_footer_text: settingsForm.menu_footer_text,
    invalid_reply_text: settingsForm.invalid_reply_text,
    timeout_response_text: settingsForm.timeout_response_text,
    unavailable_agent_text: settingsForm.unavailable_agent_text,
    custom_final_option_enabled: settingsForm.custom_final_option_enabled,
    custom_final_option_text: settingsForm.custom_final_option_text,
    custom_final_option_response: settingsForm.custom_final_option_response,
    custom_final_option_action: settingsForm.custom_final_option_action,
    custom_final_option_team_id:
      settingsForm.custom_final_option_action === "assign_to_team"
        ? settingsForm.custom_final_option_team_id || null
        : null,
    hide_unavailable_agents: settingsForm.hide_unavailable_agents,
  };
}

function toggleAllowedInstance(instanceId: string, checked: boolean) {
  if (checked) {
    if (!settingsForm.allowed_instance_ids.includes(instanceId)) {
      settingsForm.allowed_instance_ids = [
        ...settingsForm.allowed_instance_ids,
        instanceId,
      ];
    }
    return;
  }

  settingsForm.allowed_instance_ids = settingsForm.allowed_instance_ids.filter(
    (id) => id !== instanceId,
  );
}

function selectAllInstances() {
  settingsForm.allowed_instance_ids = availableInstances.value.map(
    (instance) => instance.id,
  );
}

function clearAllowedInstances() {
  settingsForm.allowed_instance_ids = [];
}

function isInstanceAllowed(instanceId: string) {
  return settingsForm.allowed_instance_ids.includes(instanceId);
}

function instanceScopeRowClass(instanceId: string) {
  return isInstanceAllowed(instanceId)
    ? "border-primary/30 bg-primary/5 text-foreground shadow-sm"
    : "border-transparent bg-background/60 text-foreground hover:border-border hover:bg-muted/50";
}

function displayUserName(user: User) {
  return user.full_name?.trim() || user.email;
}

function formatDate(value?: string | null) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col bg-background">
    <PageHeader
      :title="t('agentSelection.page.title')"
      :description="t('agentSelection.page.subtitle')"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <Button variant="outline" :disabled="isLoading" @click="loadPage">
          <RefreshCcw class="mr-2 h-4 w-4" />
          Refresh
        </Button>
        <Button :disabled="isSaving" @click="saveSettings">
          <Loader2 v-if="isSaving" class="mr-2 h-4 w-4 animate-spin" />
          <Save v-else class="mr-2 h-4 w-4" />
          Save
        </Button>
      </template>
    </PageHeader>

    <div class="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
      <div v-if="isLoading" class="flex min-h-[360px] items-center justify-center">
        <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
      </div>

      <Tabs v-else v-model="selectedTab" class="mx-auto w-full max-w-7xl space-y-4 pb-8">
        <div class="sticky top-0 z-20 -mx-4 border-b bg-background/95 px-4 pb-3 pt-1 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:-mx-6 sm:px-6">
          <div class="mx-auto flex w-full max-w-7xl overflow-x-auto">
            <TabsList class="shrink-0">
              <TabsTrigger value="settings">{{ t("agentSelection.page.tabs.settings") }}</TabsTrigger>
              <TabsTrigger value="participants">{{ t("agentSelection.page.tabs.agents") }}</TabsTrigger>
              <TabsTrigger value="options">{{ t("agentSelection.page.tabs.options") }}</TabsTrigger>
              <TabsTrigger value="audit">{{ t("agentSelection.page.tabs.operations") }}</TabsTrigger>
            </TabsList>
          </div>
        </div>

      <TabsContent value="settings" class="space-y-4">
        <section
          class="grid gap-3 rounded-lg border bg-card p-3 lg:grid-cols-[minmax(0,1fr)_320px]"
          aria-label="Routing rule profile"
        >
          <div class="min-w-0">
            <p class="text-xs font-medium uppercase text-muted-foreground">
              Rule profile
            </p>
            <p class="mt-1 text-sm font-semibold">{{ activeRuleProfileLabel }}</p>
            <p class="mt-1 text-xs leading-5 text-muted-foreground">
              Choose a specific instance to configure its own agents, options, and delay.
            </p>
          </div>
          <div class="space-y-2">
            <Label for="agent-routing-rule-profile">Configure rule for</Label>
            <div class="flex gap-2">
              <Select
                :model-value="selectedRuleProfile"
                @update:model-value="changeRuleProfile"
              >
                <SelectTrigger id="agent-routing-rule-profile" class="min-w-0 flex-1">
                  <SelectValue placeholder="Choose rule profile" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__global__">Global/default rule</SelectItem>
                  <SelectItem
                    v-for="instance in availableInstances"
                    :key="instance.id"
                    :value="instance.id"
                  >
                    {{ instance.name }}
                    <span v-if="instance.phone_number" class="text-muted-foreground">
                      - {{ instance.phone_number }}
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
              <Button
                type="button"
                variant="outline"
                :disabled="isDeletingRule || !activeRuleIsSaved"
                :title="removeRuleLabel"
                @click="removeCurrentRule"
              >
                <Loader2 v-if="isDeletingRule" class="h-4 w-4 animate-spin" />
                <Trash2 v-else class="h-4 w-4" />
              </Button>
            </div>
            <p v-if="!activeRuleIsSaved" class="text-xs text-muted-foreground">
              This is an inherited draft. Press Save to add this rule.
            </p>
          </div>
        </section>

        <section
          class="grid gap-3 rounded-lg border bg-card p-3 sm:grid-cols-3"
          aria-label="Customer routing summary"
        >
          <div class="rounded-md bg-muted/35 px-3 py-2.5">
            <p class="text-xs font-medium text-muted-foreground">Status</p>
            <p class="mt-1 text-sm font-semibold">{{ routingStatusLabel }}</p>
            <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ routingStatusHelp }}</p>
          </div>
          <div class="rounded-md bg-muted/35 px-3 py-2.5">
            <p class="text-xs font-medium text-muted-foreground">Profile</p>
            <p class="mt-1 text-sm font-semibold">{{ selectedInstanceScopeLabel }}</p>
            <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ triggerModeLabel }}</p>
          </div>
          <div class="rounded-md bg-muted/35 px-3 py-2.5">
            <p class="text-xs font-medium text-muted-foreground">Timing</p>
            <p class="mt-1 text-sm font-semibold">{{ timingSummaryLabel }}</p>
            <p class="mt-1 text-xs leading-5 text-muted-foreground">
              {{ settingsForm.max_invalid_attempts }} invalid attempts allowed
            </p>
          </div>
        </section>

        <div class="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <div class="grid gap-5">
            <Card v-if="!activeRuleInstanceId">
              <CardHeader class="gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div class="space-y-1.5">
                  <CardTitle class="text-base">Instance scope</CardTitle>
                  <CardDescription>
                    {{ selectedInstanceScopeHelp }}
                  </CardDescription>
                </div>
                <div class="flex shrink-0 flex-wrap items-center gap-2">
                  <Badge variant="secondary" class="rounded-full px-2.5">
                    {{ selectedInstanceScopeLabel }}
                  </Badge>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    :disabled="availableInstances.length === 0 || allInstancesSelected"
                    @click="selectAllInstances"
                  >
                    Select all
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    :disabled="selectedInstanceCount === 0"
                    @click="clearAllowedInstances"
                  >
                    Clear
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <div class="max-h-72 overflow-y-auto rounded-lg border bg-muted/20 p-2">
                  <div
                    v-if="instancesStore.loading"
                    class="flex min-h-28 items-center justify-center text-sm text-muted-foreground"
                  >
                    Loading instances...
                  </div>
                  <div
                    v-else-if="availableInstances.length === 0"
                    class="flex min-h-28 items-center justify-center text-sm text-muted-foreground"
                  >
                    No WhatsMeow instances found.
                  </div>
                  <div v-else class="grid gap-2 sm:grid-cols-2 2xl:grid-cols-3">
                    <label
                      v-for="instance in availableInstances"
                      :key="instance.id"
                      :for="`agent-routing-instance-${instance.id}`"
                      class="flex min-h-14 min-w-0 cursor-pointer items-start gap-3 rounded-md border px-3 py-2.5 transition-[background-color,border-color,box-shadow]"
                      :class="instanceScopeRowClass(instance.id)"
                    >
                      <span class="shrink-0 pt-0.5">
                        <Checkbox
                          :id="`agent-routing-instance-${instance.id}`"
                          :name="`agent-routing-instance-${instance.id}`"
                          :checked="settingsForm.allowed_instance_ids.includes(instance.id)"
                          @update:checked="toggleAllowedInstance(instance.id, $event === true)"
                        />
                      </span>
                      <span class="min-w-0 flex-1 text-sm leading-snug">
                        <span class="block break-words font-medium">{{ instance.name }}</span>
                        <span
                          v-if="instance.phone_number"
                          class="block break-words text-xs text-muted-foreground"
                        >
                          {{ instance.phone_number }}
                        </span>
                      </span>
                    </label>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle class="text-base">Message templates</CardTitle>
                <CardDescription>
                  Text sent by WhatsMeow while the chat remains pending.
                </CardDescription>
              </CardHeader>
              <CardContent class="grid gap-4">
                <div class="space-y-2">
                  <Label for="agent-routing-menu-header">Menu header</Label>
                  <Textarea
                    id="agent-routing-menu-header"
                    v-model="settingsForm.menu_header_text"
                    name="agent-routing-menu-header"
                    :rows="2"
                  />
                </div>
                <div class="space-y-2">
                  <Label for="agent-routing-menu-footer">Menu footer</Label>
                  <Textarea
                    id="agent-routing-menu-footer"
                    v-model="settingsForm.menu_footer_text"
                    name="agent-routing-menu-footer"
                    :rows="2"
                  />
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <div class="space-y-2">
                    <Label for="agent-routing-invalid-reply">Invalid reply text</Label>
                    <Textarea
                      id="agent-routing-invalid-reply"
                      v-model="settingsForm.invalid_reply_text"
                      name="agent-routing-invalid-reply"
                      :rows="3"
                    />
                  </div>
                  <div class="space-y-2">
                    <Label for="agent-routing-unavailable-agent">Unavailable agent text</Label>
                    <Textarea
                      id="agent-routing-unavailable-agent"
                      v-model="settingsForm.unavailable_agent_text"
                      name="agent-routing-unavailable-agent"
                      :rows="3"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle class="text-base">Custom final option</CardTitle>
                <CardDescription>
                  Example: سأذهب للفرع للطباعة
                </CardDescription>
              </CardHeader>
              <CardContent class="grid gap-4 md:grid-cols-2">
                <div class="flex items-center justify-between rounded border p-3 md:col-span-2">
                  <Label for="agent-routing-custom-final-enabled">Show final option</Label>
                  <Switch
                    id="agent-routing-custom-final-enabled"
                    v-model:checked="settingsForm.custom_final_option_enabled"
                    aria-label="Show final option"
                  />
                </div>
                <div class="space-y-2">
                  <Label for="agent-routing-custom-final-text">Option text</Label>
                  <Input
                    id="agent-routing-custom-final-text"
                    v-model="settingsForm.custom_final_option_text"
                    name="agent-routing-custom-final-text"
                  />
                </div>
                <div class="space-y-2">
                  <Label for="agent-routing-custom-final-action">Action</Label>
                  <Select v-model="settingsForm.custom_final_option_action">
                    <SelectTrigger id="agent-routing-custom-final-action"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="keep_pending">Keep pending</SelectItem>
                      <SelectItem value="send_only">Send only</SelectItem>
                      <SelectItem value="close_chat">Close chat</SelectItem>
                      <SelectItem value="assign_to_team">Assign to team</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div
                  v-if="settingsForm.custom_final_option_action === 'assign_to_team'"
                  class="space-y-2"
                >
                  <Label for="agent-routing-custom-final-team">Target team</Label>
                  <Select v-model="settingsForm.custom_final_option_team_id">
                    <SelectTrigger id="agent-routing-custom-final-team">
                      <SelectValue placeholder="Choose team" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="team in teamOptions" :key="team.id" :value="team.id">
                        {{ team.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="space-y-2 md:col-span-2">
                  <Label for="agent-routing-custom-final-response">Response text</Label>
                  <Textarea
                    id="agent-routing-custom-final-response"
                    v-model="settingsForm.custom_final_option_response"
                    name="agent-routing-custom-final-response"
                    :rows="2"
                  />
                </div>
              </CardContent>
            </Card>
          </div>

          <aside class="space-y-5 xl:sticky xl:top-20">
            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2 text-base">
                  <Bot class="h-4 w-4" />
                  Activation
                </CardTitle>
                <CardDescription>
                  Start or pause the customer-facing routing flow.
                </CardDescription>
              </CardHeader>
              <CardContent class="space-y-3">
                <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/25 p-4">
                  <div class="space-y-1">
                    <Label for="agent-routing-enabled">Enable customer routing</Label>
                    <p class="max-w-[34ch] text-xs leading-5 text-muted-foreground">
                      Start the selection flow for pending chats.
                    </p>
                  </div>
                  <Switch
                    id="agent-routing-enabled"
                    v-model:checked="settingsForm.enabled"
                    aria-label="Enable customer routing"
                  />
                </div>
                <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/25 p-4">
                  <div class="space-y-1">
                    <Label for="agent-routing-hide-unavailable">Hide unavailable agents</Label>
                    <p class="max-w-[34ch] text-xs leading-5 text-muted-foreground">
                      Keep the customer list limited to agents who can reply.
                    </p>
                  </div>
                  <Switch
                    id="agent-routing-hide-unavailable"
                    v-model:checked="settingsForm.hide_unavailable_agents"
                    aria-label="Hide unavailable agents"
                  />
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2 text-base">
                  <Clock3 class="h-4 w-4" />
                  Trigger rules
                </CardTitle>
                <CardDescription>
                  Control when the menu appears and how long selection stays open.
                </CardDescription>
              </CardHeader>
              <CardContent class="grid gap-4">
                <div class="space-y-2">
                  <Label for="agent-routing-trigger-mode">Trigger mode</Label>
                  <Select v-model="settingsForm.trigger_mode">
                    <SelectTrigger id="agent-routing-trigger-mode"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="first_pending_message">{{ t("agentSelection.page.trigger.first_pending_message") }}</SelectItem>
                      <SelectItem value="keyword">{{ t("agentSelection.page.trigger.keyword") }}</SelectItem>
                      <SelectItem value="after_office_hours">{{ t("agentSelection.page.trigger.after_office_hours") }}</SelectItem>
                      <SelectItem value="chatbot_step">{{ t("agentSelection.page.trigger.chatbot_step") }}</SelectItem>
                      <SelectItem value="manual_test">{{ t("agentSelection.page.trigger.manual_test") }}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="space-y-2">
                  <Label for="agent-routing-keywords">Keywords</Label>
                  <Input
                    id="agent-routing-keywords"
                    v-model="settingsForm.trigger_keywords"
                    name="agent-routing-keywords"
                    placeholder="موظف, تحويل, agent"
                  />
                </div>
                <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
                  <div class="space-y-2">
                    <Label for="agent-routing-prompt-delay-min">Min delay</Label>
                    <Input
                      id="agent-routing-prompt-delay-min"
                      v-model.number="settingsForm.prompt_delay_min_minutes"
                      name="agent-routing-prompt-delay-min"
                      type="number"
                      min="0"
                    />
                  </div>
                  <div class="space-y-2">
                    <Label for="agent-routing-prompt-delay-max">Max delay</Label>
                    <Input
                      id="agent-routing-prompt-delay-max"
                      v-model.number="settingsForm.prompt_delay_max_minutes"
                      name="agent-routing-prompt-delay-max"
                      type="number"
                      min="0"
                    />
                  </div>
                  <div class="space-y-2">
                    <Label for="agent-routing-selection-timeout">Timeout</Label>
                    <Input
                      id="agent-routing-selection-timeout"
                      v-model.number="settingsForm.selection_timeout_minutes"
                      name="agent-routing-selection-timeout"
                      type="number"
                      min="1"
                    />
                  </div>
                  <div class="space-y-2">
                    <Label for="agent-routing-invalid-attempts">Invalid attempts</Label>
                    <Input
                      id="agent-routing-invalid-attempts"
                      v-model.number="settingsForm.max_invalid_attempts"
                      name="agent-routing-invalid-attempts"
                      type="number"
                      min="1"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2 text-base">
                  <Eye class="h-4 w-4" />
                  WhatsMeow preview
                </CardTitle>
                <CardDescription>Rendered from currently saved options.</CardDescription>
              </CardHeader>
              <CardContent>
                <pre class="max-h-[360px] min-h-40 overflow-auto whitespace-pre-wrap rounded-lg border bg-muted/45 p-4 text-sm leading-7 text-foreground shadow-inner">{{ agentSelectionStore.preview?.text || "No visible options yet." }}</pre>
              </CardContent>
            </Card>
          </aside>
        </div>
      </TabsContent>

      <TabsContent value="participants" class="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2 text-base">
              <UserPlus class="h-4 w-4" />
              Add visible agent
            </CardTitle>
          </CardHeader>
          <CardContent class="grid gap-3 md:grid-cols-6">
            <div class="space-y-2 md:col-span-2">
              <Label for="agent-routing-participant-user">Agent</Label>
              <Select v-model="participantForm.user_id">
                <SelectTrigger id="agent-routing-participant-user">
                  <SelectValue placeholder="Choose agent" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="user in availableUsers" :key="user.id" :value="user.id">
                    {{ displayUserName(user) }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2 md:col-span-2">
              <Label for="agent-routing-participant-name">Customer display name</Label>
              <Input
                id="agent-routing-participant-name"
                v-model="participantForm.display_name"
                name="agent-routing-participant-name"
              />
            </div>
            <div class="space-y-2 md:col-span-2">
              <Label for="agent-routing-participant-service">Service / work</Label>
              <Input
                id="agent-routing-participant-service"
                v-model="participantForm.description"
                name="agent-routing-participant-service"
                placeholder="خدمات الكترونيه"
              />
            </div>
            <div class="space-y-2">
              <Label for="agent-routing-participant-sort">Sort</Label>
              <Input
                id="agent-routing-participant-sort"
                v-model.number="participantForm.sort_order"
                name="agent-routing-participant-sort"
                type="number"
              />
            </div>
            <div class="space-y-2">
              <Label for="agent-routing-participant-max-open">Max open</Label>
              <Input
                id="agent-routing-participant-max-open"
                v-model="participantForm.max_open_chats"
                name="agent-routing-participant-max-open"
                type="number"
                placeholder="Any"
              />
            </div>
            <div class="flex min-h-10 items-center gap-2">
              <Switch
                id="agent-routing-participant-available"
                v-model:checked="participantForm.show_only_when_available"
                aria-label="Only when available"
              />
              <Label for="agent-routing-participant-available">Only when available</Label>
            </div>
            <div class="flex min-h-10 items-center gap-2">
              <Switch
                id="agent-routing-participant-enabled"
                v-model:checked="participantForm.is_enabled"
                aria-label="Agent enabled"
              />
              <Label for="agent-routing-participant-enabled">Enabled</Label>
            </div>
            <Button class="md:col-span-2" :disabled="isAddingParticipant" @click="addParticipant">
              <Loader2 v-if="isAddingParticipant" class="mr-2 h-4 w-4 animate-spin" />
              <Plus v-else class="mr-2 h-4 w-4" />
              Add agent
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardContent class="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Display name</TableHead>
                  <TableHead>Service / work</TableHead>
                  <TableHead>Internal user</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead class="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="participant in agentSelectionStore.participants" :key="participant.id">
                  <TableCell class="font-medium">
                    <Input
                      v-if="editingParticipantId === participant.id"
                      v-model="participantEditForm.display_name"
                      :aria-label="`Edit display name for ${participant.display_name}`"
                    />
                    <span v-else>{{ participant.display_name }}</span>
                  </TableCell>
                  <TableCell>
                    <Input
                      v-if="editingParticipantId === participant.id"
                      v-model="participantEditForm.description"
                      :aria-label="`Edit service for ${participant.display_name}`"
                    />
                    <span v-else>{{ participant.description || "-" }}</span>
                  </TableCell>
                  <TableCell>{{ participant.user?.full_name || participant.user?.email || participant.user_id }}</TableCell>
                  <TableCell>
                    <div
                      v-if="editingParticipantId === participant.id"
                      class="grid min-w-40 gap-2 sm:grid-cols-2"
                    >
                      <Input
                        v-model.number="participantEditForm.sort_order"
                        type="number"
                        aria-label="Edit sort order"
                      />
                      <Input
                        v-model="participantEditForm.max_open_chats"
                        type="number"
                        placeholder="Any"
                        aria-label="Edit max open chats"
                      />
                    </div>
                    <div v-else class="flex flex-wrap gap-2">
                      <Badge :variant="participant.is_enabled ? 'default' : 'secondary'">
                        {{ participant.is_enabled ? "Enabled" : "Hidden" }}
                      </Badge>
                      <Badge variant="outline">Sort {{ participant.sort_order }}</Badge>
                      <Badge variant="outline">
                        Max {{ participant.max_open_chats ?? "Any" }}
                      </Badge>
                      <Badge v-if="participant.user?.is_available === false" variant="secondary">
                        Unavailable
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell class="text-right">
                    <template v-if="editingParticipantId === participant.id">
                      <Button
                        size="icon"
                        variant="outline"
                        :disabled="isSavingParticipantEdit"
                        :aria-label="`Save ${participant.display_name}`"
                        @click="saveParticipantEdit(participant.id)"
                      >
                        <Loader2
                          v-if="isSavingParticipantEdit"
                          class="h-4 w-4 animate-spin"
                        />
                        <Check v-else class="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        class="ml-2"
                        :disabled="isSavingParticipantEdit"
                        :aria-label="`Cancel editing ${participant.display_name}`"
                        @click="cancelEditParticipant"
                      >
                        <X class="h-4 w-4" />
                      </Button>
                    </template>
                    <template v-else>
                      <Button
                        size="sm"
                        variant="outline"
                        @click="toggleParticipant(participant.id, !participant.is_enabled)"
                      >
                        {{ participant.is_enabled ? "Hide" : "Show" }}
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        class="ml-2"
                        :aria-label="`Edit ${participant.display_name}`"
                        @click="startEditParticipant(participant)"
                      >
                        <Pencil class="h-4 w-4" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        class="ml-2"
                        :aria-label="`Remove ${participant.display_name}`"
                        @click="removeParticipant(participant)"
                      >
                        <Trash2 class="h-4 w-4" />
                      </Button>
                    </template>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="options" class="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Team and queue options</CardTitle>
          </CardHeader>
          <CardContent class="grid gap-3 md:grid-cols-6">
            <div class="space-y-2">
              <Label for="agent-routing-option-type">Type</Label>
              <Select v-model="optionForm.option_type">
                <SelectTrigger id="agent-routing-option-type"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="team">Team</SelectItem>
                  <SelectItem value="queue">Queue</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div v-if="optionForm.option_type === 'team'" class="space-y-2 md:col-span-2">
              <Label for="agent-routing-option-team">Team</Label>
              <Select v-model="optionForm.team_id">
                <SelectTrigger id="agent-routing-option-team">
                  <SelectValue placeholder="Choose team" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="team in teamOptions" :key="team.id" :value="team.id">
                    {{ team.name }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2 md:col-span-2">
              <Label for="agent-routing-option-label">Customer label</Label>
              <Input
                id="agent-routing-option-label"
                v-model="optionForm.label"
                name="agent-routing-option-label"
                placeholder="فريق الحسابات"
              />
            </div>
            <div class="space-y-2">
              <Label for="agent-routing-option-sort">Sort</Label>
              <Input
                id="agent-routing-option-sort"
                v-model.number="optionForm.sort_order"
                name="agent-routing-option-sort"
                type="number"
              />
            </div>
            <div class="flex min-h-10 items-center gap-2">
              <Switch
                id="agent-routing-option-enabled"
                v-model:checked="optionForm.is_enabled"
                aria-label="Option enabled"
              />
              <Label for="agent-routing-option-enabled">Enabled</Label>
            </div>
            <Button class="md:col-span-2" :disabled="isAddingOption" @click="addOption">
              <Loader2 v-if="isAddingOption" class="mr-2 h-4 w-4 animate-spin" />
              <Plus v-else class="mr-2 h-4 w-4" />
              Add option
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardContent class="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Label</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead class="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="option in agentSelectionStore.options" :key="option.id">
                  <TableCell class="font-medium">{{ option.label }}</TableCell>
                  <TableCell>{{ option.option_type }}</TableCell>
                  <TableCell>
                    <Badge :variant="option.is_enabled ? 'default' : 'secondary'">
                      {{ option.is_enabled ? "Enabled" : "Hidden" }}
                    </Badge>
                  </TableCell>
                  <TableCell class="text-right">
                    <Button size="sm" variant="outline" @click="toggleOption(option.id, !option.is_enabled)">
                      {{ option.is_enabled ? "Hide" : "Show" }}
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      class="ml-2"
                      :aria-label="`Remove ${option.label}`"
                      @click="removeOption(option)"
                    >
                      <Trash2 class="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="audit" class="space-y-4">
        <div class="flex justify-end">
          <Button variant="outline" @click="refreshOps">
            <RefreshCcw class="mr-2 h-4 w-4" />
            Refresh operations
          </Button>
        </div>
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Test send</CardTitle>
            <CardDescription>
              Send the current selection menu to a real contact to verify rendering.
            </CardDescription>
          </CardHeader>
          <CardContent class="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div class="flex-1 space-y-1.5">
              <Label for="agent-routing-test-contact">Contact ID</Label>
              <Input
                id="agent-routing-test-contact"
                v-model="testSendContactId"
                placeholder="e.g. 9c2b8e14-..."
                autocomplete="off"
              />
            </div>
            <Button :disabled="!canTestSend" @click="runTestSend">
              <Bot v-if="!isTestSending" class="mr-2 h-4 w-4" />
              <Loader2 v-else class="mr-2 h-4 w-4 animate-spin" />
              {{ isTestSending ? t("agentSelection.page.testSend.sending") : t("agentSelection.page.testSend.button") }}
            </Button>
          </CardContent>
        </Card>
        <div class="grid gap-4 xl:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle class="flex items-center gap-2 text-base">
                <Clock3 class="h-4 w-4" />
                Sessions
              </CardTitle>
            </CardHeader>
            <CardContent class="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Status</TableHead>
                    <TableHead>Due</TableHead>
                    <TableHead>Account</TableHead>
                    <TableHead class="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="session in agentSelectionStore.sessions" :key="session.id">
                    <TableCell><Badge variant="secondary">{{ session.status }}</Badge></TableCell>
                    <TableCell>{{ formatDate(session.prompt_due_at) }}</TableCell>
                    <TableCell>{{ session.whatsapp_account }}</TableCell>
                    <TableCell class="text-right">
                      <Button
                        v-if="session.status === 'waiting_delay' || session.status === 'menu_sent'"
                        size="sm"
                        variant="outline"
                        @click="cancelSession(session.id)"
                      >
                        Cancel
                      </Button>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle class="text-base">Audit events</CardTitle>
            </CardHeader>
            <CardContent class="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Event</TableHead>
                    <TableHead>Actor</TableHead>
                    <TableHead>Time</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="event in agentSelectionStore.auditEvents" :key="event.id">
                    <TableCell class="font-medium">{{ event.event_type }}</TableCell>
                    <TableCell>{{ event.actor_type }}</TableCell>
                    <TableCell>{{ formatDate(event.created_at) }}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>
      </TabsContent>
      </Tabs>
    </div>

    <DeleteConfirmDialog
      v-model:open="participantDeleteDialogOpen"
      title="Remove agent"
      :item-name="participantToDelete?.name"
      @confirm="confirmRemoveParticipant"
    />
    <DeleteConfirmDialog
      v-model:open="optionDeleteDialogOpen"
      title="Remove option"
      :item-name="optionToDelete?.name"
      @confirm="confirmRemoveOption"
    />
  </div>
</template>
