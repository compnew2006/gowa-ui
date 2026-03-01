<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { RouterLink } from "vue-router";
import { useInstancesStore } from "@/stores/instances";
import InstanceCard from "@/components/whatsmeow/InstanceCard.vue";
import QRCodeModal from "@/components/whatsmeow/QRCodeModal.vue";
import { DeleteConfirmDialog, PageHeader } from "@/components/shared";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Smartphone, Plus, Loader2 } from "lucide-vue-next";
import { wsService } from "@/services/websocket";
import { instancesService } from "@/services/api";
import { toast } from "vue-sonner";
import { useI18n } from "vue-i18n";
import {
  upsertInstanceTagSettings,
  type InstanceTagColorKey,
  type InstanceTagDisplayMode,
} from "@/lib/instance-tag";
import {
  normalizeAutoRejectCallSettings,
  type AutoRejectCallSettings,
} from "@/lib/instance-auto-reject";
import {
  normalizeAutoCampaignSettings,
  type AutoCampaignSettings,
} from "@/lib/instance-auto-campaign";

const instancesStore = useInstancesStore();
const { t } = useI18n();
const {
  fetchInstances,
  fetchAllHealth,
  fetchInstance,
  startHealthPolling,
  stopHealthPolling,
  createInstance,
  updateInstance,
  connectInstance,
  requestPhonePairCode,
  reconnectInstance,
  disconnectInstance,
  deleteInstance,
} = instancesStore;

interface CachedQRCode {
  code: string;
  timeoutSeconds: number;
  receivedAtMs: number;
}

const CONNECT_EVENT_TIMEOUT_MS = 15000;
const QR_SNAPSHOT_POLL_INTERVAL_MS = 1000;

// State
const qrModalOpen = ref(false);
const qrCode = ref("");
const qrTimeout = ref(20);
const currentInstanceId = ref("");
const qrCodesByInstance = ref<Record<string, CachedQRCode>>({});
const isRefreshingQR = ref(false);
const pairingCode = ref("");
const pairingPhoneNumber = ref("");
const isRequestingPairCode = ref(false);
const qrErrorMessage = ref("");
let connectWatchdogTimer: number | null = null;
let qrSnapshotPollTimer: number | null = null;
let qrSnapshotPollInFlight = false;

// Create Dialog State
const createDialogOpen = ref(false);
const newInstanceName = ref("");
const isCreating = ref(false);
const deleteDialogOpen = ref(false);
const deletingInstance = ref<{ id: string; name: string } | null>(null);
const deleteChatsWithInstance = ref(false);
const isDeleting = ref(false);
const editDialogOpen = ref(false);
const editInstanceId = ref("");
const editInstanceName = ref("");
const isUpdatingName = ref(false);
const tagSettingsSaving = ref<Record<string, boolean>>({});
const autoSyncSaving = ref<Record<string, boolean>>({});
const autoDownloadIncomingMediaSaving = ref<Record<string, boolean>>({});
const autoRejectSaving = ref<Record<string, boolean>>({});
const autoCampaignSaving = ref<Record<string, boolean>>({});
const autoCampaignUploading = ref<Record<string, boolean>>({});

// Lifecycle
onMounted(async () => {
  await fetchInstances();
  await fetchAllHealth();
  startHealthPolling(30000);
  wsService.subscribe("instance_qr_code", handleQRCode);
  wsService.subscribe("instance_connected", handleConnected);
  wsService.subscribe("instance_disconnected", handleDisconnected);
  wsService.subscribe("instance_banned", handleBanned);
  wsService.subscribe("instance_logged_out", handleLoggedOut);
  wsService.subscribe("instance_qr_timeout", handleQRTimeout);
  wsService.subscribe("instance_reconnect_failed", handleReconnectFailed);
});

onUnmounted(() => {
  clearConnectWatchdog();
  clearQRSnapshotPoll();
  stopHealthPolling();
  wsService.unsubscribe("instance_qr_code", handleQRCode);
  wsService.unsubscribe("instance_connected", handleConnected);
  wsService.unsubscribe("instance_disconnected", handleDisconnected);
  wsService.unsubscribe("instance_banned", handleBanned);
  wsService.unsubscribe("instance_logged_out", handleLoggedOut);
  wsService.unsubscribe("instance_qr_timeout", handleQRTimeout);
  wsService.unsubscribe("instance_reconnect_failed", handleReconnectFailed);
});

function cacheQRCode(instanceId: string, code: string, timeoutSeconds: number) {
  if (!code) return;
  qrCodesByInstance.value = {
    ...qrCodesByInstance.value,
    [instanceId]: {
      code,
      timeoutSeconds,
      receivedAtMs: Date.now(),
    },
  };
}

function clearCachedQRCode(instanceId: string) {
  if (!qrCodesByInstance.value[instanceId]) return;
  const next = { ...qrCodesByInstance.value };
  delete next[instanceId];
  qrCodesByInstance.value = next;
}

function getActiveCachedQRCode(instanceId: string): CachedQRCode | null {
  const cached = qrCodesByInstance.value[instanceId];
  if (!cached) return null;

  const expiresAt = cached.receivedAtMs + cached.timeoutSeconds * 1000;
  if (Date.now() >= expiresAt) {
    clearCachedQRCode(instanceId);
    return null;
  }

  return cached;
}

function clearConnectWatchdog() {
  if (connectWatchdogTimer !== null) {
    clearTimeout(connectWatchdogTimer);
    connectWatchdogTimer = null;
  }
}

function clearQRSnapshotPoll() {
  if (qrSnapshotPollTimer !== null) {
    clearInterval(qrSnapshotPollTimer);
    qrSnapshotPollTimer = null;
  }
  qrSnapshotPollInFlight = false;
}

async function fetchQRSnapshot(instanceID: string): Promise<boolean> {
  if (!instanceID) return false;

  try {
    const response = await instancesService.getQRCode(instanceID);
    const payload = (response.data?.data || response.data) as {
      instance_id?: string;
      available?: boolean;
      qr_code?: string;
      timeout_seconds?: number;
    };

    if (payload?.available !== true || !payload.qr_code) {
      return false;
    }

    const timeoutSeconds = Number(payload.timeout_seconds) || 20;
    cacheQRCode(instanceID, payload.qr_code, timeoutSeconds);

    if (instanceID === currentInstanceId.value && qrModalOpen.value) {
      clearConnectWatchdog();
      setQRError("");
      qrCode.value = payload.qr_code;
      qrTimeout.value = timeoutSeconds;
      isRefreshingQR.value = false;
    }

    return true;
  } catch {
    return false;
  }
}

function startQRSnapshotPoll(instanceID: string) {
  clearQRSnapshotPoll();

  qrSnapshotPollTimer = window.setInterval(async () => {
    if (
      qrSnapshotPollInFlight ||
      !instanceID ||
      instanceID !== currentInstanceId.value ||
      !qrModalOpen.value ||
      qrCode.value
    ) {
      if (!instanceID || instanceID !== currentInstanceId.value || !qrModalOpen.value || qrCode.value) {
        clearQRSnapshotPoll();
      }
      return;
    }

    qrSnapshotPollInFlight = true;
    try {
      const loaded = await fetchQRSnapshot(instanceID);
      if (loaded) {
        clearQRSnapshotPoll();
      }
    } finally {
      qrSnapshotPollInFlight = false;
    }
  }, QR_SNAPSHOT_POLL_INTERVAL_MS);
}

function setQRError(message: string) {
  qrErrorMessage.value = message.trim();
}

async function syncInstanceStatus(instanceID: string) {
  const latest = await fetchInstance(instanceID);
  if (!latest) {
    return null;
  }

  if (latest.status === "connected") {
    handleConnected({
      instance_id: latest.id,
      phone_number: latest.phone_number,
    });
    return latest;
  }

  instancesStore.updateInstanceStatus(
    latest.id,
    latest.status,
    latest.phone_number,
  );
  return latest;
}

function scheduleConnectWatchdog(instanceID: string) {
  clearConnectWatchdog();
  connectWatchdogTimer = window.setTimeout(async () => {
    if (instanceID !== currentInstanceId.value || !qrModalOpen.value || qrCode.value) {
      return;
    }

    if (await fetchQRSnapshot(instanceID)) {
      return;
    }

    const latest = await syncInstanceStatus(instanceID);
    if (latest?.status === "connected") {
      return;
    }

    const timeoutMessage = t("instances.qr_modal.validation.connectionTimeout");
    setQRError(timeoutMessage);
    isRefreshingQR.value = false;
    clearQRSnapshotPoll();
    toast.error(timeoutMessage);
  }, CONNECT_EVENT_TIMEOUT_MS);
}

// Handlers
function handleQRCode(payload: any) {
  const timeoutSeconds = Number(payload.timeout_seconds) || 20;
  cacheQRCode(payload.instance_id, payload.qr_code || "", timeoutSeconds);

  if (payload.instance_id !== currentInstanceId.value) return;

  clearConnectWatchdog();
  clearQRSnapshotPoll();
  setQRError("");
  if (payload.qr_code) {
    qrCode.value = payload.qr_code;
  }
  qrTimeout.value = timeoutSeconds;
  isRefreshingQR.value = false;
  qrModalOpen.value = true;
}

function handleConnected(payload: any) {
  clearConnectWatchdog();
  clearQRSnapshotPoll();
  clearCachedQRCode(payload.instance_id);
  if (payload.instance_id === currentInstanceId.value) {
    qrCode.value = "";
    pairingCode.value = "";
    pairingPhoneNumber.value = "";
    setQRError("");
    qrModalOpen.value = false;
  }
  isRefreshingQR.value = false;
  instancesStore.updateInstanceStatus(
    payload.instance_id,
    "connected",
    payload.phone_number,
  );
  instancesStore.fetchInstanceHealth(payload.instance_id);
}

function handleDisconnected(payload: any) {
  clearConnectWatchdog();
  clearQRSnapshotPoll();
  clearCachedQRCode(payload.instance_id);
  if (payload.instance_id === currentInstanceId.value) {
    qrCode.value = "";
  }
  isRefreshingQR.value = false;
  instancesStore.updateInstanceStatus(payload.instance_id, "disconnected");
  instancesStore.fetchInstanceHealth(payload.instance_id);
}

function handleBanned(payload: any) {
  clearConnectWatchdog();
  clearQRSnapshotPoll();
  clearCachedQRCode(payload.instance_id);
  if (payload.instance_id === currentInstanceId.value) {
    qrCode.value = "";
  }
  isRefreshingQR.value = false;
  instancesStore.updateInstanceStatus(payload.instance_id, "banned");
  instancesStore.fetchInstanceHealth(payload.instance_id);
}

function handleLoggedOut(payload: any) {
  clearConnectWatchdog();
  clearQRSnapshotPoll();
  clearCachedQRCode(payload.instance_id);
  if (payload.instance_id === currentInstanceId.value) {
    qrCode.value = "";
  }
  isRefreshingQR.value = false;
  instancesStore.updateInstanceStatus(payload.instance_id, "logged_out");
  instancesStore.fetchInstanceHealth(payload.instance_id);
}

function handleQRTimeout(payload: any) {
  if (!payload?.instance_id) return;

  clearConnectWatchdog();
  clearQRSnapshotPoll();
  clearCachedQRCode(payload.instance_id);
  instancesStore.updateInstanceStatus(payload.instance_id, "disconnected");
  instancesStore.fetchInstanceHealth(payload.instance_id);

  if (payload.instance_id !== currentInstanceId.value) return;

  qrCode.value = "";
  isRefreshingQR.value = false;
  const timeoutMessage =
    payload.message || t("instances.qr_modal.validation.qrTimeout");
  setQRError(timeoutMessage);
  toast.error(timeoutMessage);
}

function handleReconnectFailed(payload: any) {
  if (!payload?.instance_id) return;

  clearConnectWatchdog();
  clearQRSnapshotPoll();
  clearCachedQRCode(payload.instance_id);
  instancesStore.updateInstanceStatus(payload.instance_id, "disconnected");
  instancesStore.fetchInstanceHealth(payload.instance_id);

  const message =
    payload.message || t("instances.qr_modal.validation.connectionFailed");

  if (payload.instance_id === currentInstanceId.value) {
    qrCode.value = "";
    isRefreshingQR.value = false;
    qrModalOpen.value = true;
    setQRError(message);
  }

  toast.error(message);
}

async function handleConnect(id: string) {
  clearConnectWatchdog();
  clearQRSnapshotPoll();
  currentInstanceId.value = id;
  isRefreshingQR.value = false;
  isRequestingPairCode.value = false;
  pairingCode.value = "";
  pairingPhoneNumber.value = "";
  setQRError("");
  const cached = getActiveCachedQRCode(id);
  qrCode.value = cached?.code || "";
  qrTimeout.value = cached?.timeoutSeconds || 20;
  qrModalOpen.value = true; // Open modal immediately with existing code or loading state
  const initiated = await connectInstance(id);
  if (!initiated) {
    setQRError(t("instances.qr_modal.validation.connectionFailed"));
    return;
  }

  const latest = await syncInstanceStatus(id);
  if (latest?.status === "connected") {
    return;
  }

  if (!qrCode.value) {
    if (!(await fetchQRSnapshot(id))) {
      startQRSnapshotPoll(id);
    }
    scheduleConnectWatchdog(id);
  }
}

async function handleRegenerateQRCode() {
  if (!currentInstanceId.value || isRefreshingQR.value) return;

  clearConnectWatchdog();
  clearQRSnapshotPoll();
  isRefreshingQR.value = true;
  pairingCode.value = "";
  pairingPhoneNumber.value = "";
  setQRError("");
  clearCachedQRCode(currentInstanceId.value);
  qrCode.value = "";
  qrTimeout.value = 20;
  try {
    await reconnectInstance(currentInstanceId.value);
    if (!(await fetchQRSnapshot(currentInstanceId.value))) {
      startQRSnapshotPoll(currentInstanceId.value);
    }
    scheduleConnectWatchdog(currentInstanceId.value);
  } catch {
    setQRError(t("instances.qr_modal.validation.connectionFailed"));
    isRefreshingQR.value = false;
  }
}

async function handleRequestPairCode(phone: string) {
  const phoneNumber = phone.trim();
  if (!currentInstanceId.value || !phoneNumber || isRequestingPairCode.value)
    return;

  isRequestingPairCode.value = true;
  setQRError("");
  try {
    const payload = await requestPhonePairCode(
      currentInstanceId.value,
      phoneNumber,
    );
    pairingCode.value = payload?.pairing_code || "";
    pairingPhoneNumber.value = payload?.phone_number || phoneNumber;
    if (payload?.timeout_seconds) {
      const timeoutSeconds = Number(payload.timeout_seconds);
      if (!Number.isNaN(timeoutSeconds) && timeoutSeconds > 0) {
        qrTimeout.value = timeoutSeconds;
      }
    }
  } finally {
    isRequestingPairCode.value = false;
  }
}

function handleQRModalOpenChange(open: boolean) {
  qrModalOpen.value = open;
  if (!open) {
    clearConnectWatchdog();
    clearQRSnapshotPoll();
    setQRError("");
    isRefreshingQR.value = false;
  }
}

async function handleCreate() {
  const trimmedName = newInstanceName.value.trim();
  if (!trimmedName) return;
  isCreating.value = true;
  try {
    await createInstance({ name: trimmedName });
    await fetchAllHealth();
    createDialogOpen.value = false;
    newInstanceName.value = "";
  } finally {
    isCreating.value = false;
  }
}

function openDeleteDialog(id: string) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;
  deletingInstance.value = { id: instance.id, name: instance.name };
  deleteChatsWithInstance.value = false;
  deleteDialogOpen.value = true;
}

function closeDeleteDialog() {
  if (isDeleting.value) return;
  deleteDialogOpen.value = false;
  deletingInstance.value = null;
  deleteChatsWithInstance.value = false;
}

async function handleDeleteInstance() {
  if (!deletingInstance.value || isDeleting.value) return;

  isDeleting.value = true;
  try {
    await deleteInstance(deletingInstance.value.id, {
      deleteChats: deleteChatsWithInstance.value,
    });
    if (currentInstanceId.value === deletingInstance.value.id) {
      currentInstanceId.value = "";
      qrModalOpen.value = false;
      qrCode.value = "";
      pairingCode.value = "";
      pairingPhoneNumber.value = "";
    }
    deleteDialogOpen.value = false;
    deletingInstance.value = null;
    deleteChatsWithInstance.value = false;
  } finally {
    isDeleting.value = false;
  }
}

function openEditDialog(id: string) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;
  editInstanceId.value = instance.id;
  editInstanceName.value = instance.name;
  editDialogOpen.value = true;
}

function closeEditDialog() {
  editDialogOpen.value = false;
  editInstanceId.value = "";
  editInstanceName.value = "";
}

async function handleUpdateName() {
  const trimmedName = editInstanceName.value.trim();
  if (!editInstanceId.value || !trimmedName) return;
  isUpdatingName.value = true;
  try {
    await updateInstance(editInstanceId.value, { name: trimmedName });
    closeEditDialog();
  } finally {
    isUpdatingName.value = false;
  }
}

async function handleSaveTagSettings(
  id: string,
  payload: {
    customLabel: string;
    color: InstanceTagColorKey;
    displayMode: InstanceTagDisplayMode;
  },
) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;

  tagSettingsSaving.value[id] = true;
  try {
    const settings = upsertInstanceTagSettings(instance.settings, {
      chat_tag_custom_label: payload.customLabel,
      chat_tag_color: payload.color,
      chat_tag_display_mode: payload.displayMode,
    });
    await instancesStore.updateInstance(id, { settings });
  } finally {
    tagSettingsSaving.value[id] = false;
  }
}

async function handleAutoSyncUpdate(id: string, enabled: boolean) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;

  autoSyncSaving.value[id] = true;
  try {
    const settings = {
      ...(instance.settings || {}),
      auto_sync_history: enabled,
    };
    await instancesStore.updateInstance(id, { settings });
  } finally {
    autoSyncSaving.value[id] = false;
  }
}

async function handleAutoDownloadIncomingMediaUpdate(id: string, enabled: boolean) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;

  autoDownloadIncomingMediaSaving.value[id] = true;
  try {
    const settings = {
      ...(instance.settings || {}),
      auto_download_incoming_media: enabled,
    };
    await instancesStore.updateInstance(id, { settings });
  } finally {
    autoDownloadIncomingMediaSaving.value[id] = false;
  }
}

async function handleAutoRejectSettingsUpdate(
  id: string,
  payload: AutoRejectCallSettings,
) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;

  autoRejectSaving.value[id] = true;
  try {
    const settings = {
      ...(instance.settings || {}),
      auto_reject_calls: normalizeAutoRejectCallSettings(payload),
    };
    await instancesStore.updateInstance(id, { settings });
  } finally {
    autoRejectSaving.value[id] = false;
  }
}

async function handleAutoCampaignSettingsUpdate(
  id: string,
  payload: AutoCampaignSettings,
) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;

  autoCampaignSaving.value[id] = true;
  try {
    const current = normalizeAutoCampaignSettings(instance.settings?.auto_campaign);
    const next = normalizeAutoCampaignSettings({
      ...current,
      ...payload,
      last_generated_at: current.last_generated_at,
    });

    const settings = {
      ...(instance.settings || {}),
      auto_campaign: next,
    };
    await instancesStore.updateInstance(id, { settings });
  } finally {
    autoCampaignSaving.value[id] = false;
  }
}

async function handleAutoCampaignMediaUpload(id: string, file: File) {
  if (!file) return;

  autoCampaignUploading.value[id] = true;
  try {
    await instancesService.uploadAutoCampaignMedia(id, file);
    await fetchInstance(id);
    toast.success(t("instances.auto_campaign.mediaUploaded"));
  } catch (err: any) {
    const message =
      err.response?.data?.message ||
      t("instances.auto_campaign.mediaUploadFailed");
    toast.error(message);
  } finally {
    autoCampaignUploading.value[id] = false;
  }
}

async function handleAutoCampaignMediaClear(id: string) {
  const instance = instancesStore.instances.find((item) => item.id === id);
  if (!instance) return;

  autoCampaignSaving.value[id] = true;
  try {
    const current = normalizeAutoCampaignSettings(instance.settings?.auto_campaign);
    const settings = {
      ...(instance.settings || {}),
      auto_campaign: normalizeAutoCampaignSettings({
        ...current,
        media_local_path: "",
        media_mime_type: "",
        media_filename: "",
        last_generated_at: current.last_generated_at,
      }),
    };
    await instancesStore.updateInstance(id, { settings });
  } finally {
    autoCampaignSaving.value[id] = false;
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader
      :title="$t('settings.instances.title')"
      :subtitle="$t('settings.instances.subtitle')"
      :icon="Smartphone"
      icon-gradient="bg-gradient-to-br from-emerald-500 to-green-600 shadow-emerald-500/20"
    >
      <template #actions>
        <div class="flex gap-2">
          <RouterLink to="/settings/instances/health">
            <Button
              variant="outline"
              class="border-white/10 hover:bg-white/5 text-white/80 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
            >
              {{ $t("settings.instances.healthDashboard") }}
            </Button>
          </RouterLink>
          <Button
            class="bg-emerald-600 hover:bg-emerald-700 text-white"
            @click="createDialogOpen = true"
          >
            <Plus class="h-4 w-4 mr-2" />
            {{ $t("settings.instances.addAccount") }}
          </Button>
        </div>
      </template>
    </PageHeader>

    <div class="flex-1 p-6 overflow-y-auto">
      <div
        v-if="instancesStore.loading && instancesStore.instances.length === 0"
        class="flex justify-center items-center h-64"
      >
        <Loader2 class="h-8 w-8 text-white/20 light:text-gray-400 animate-spin" />
      </div>

      <div
        v-else-if="instancesStore.instances.length === 0"
        class="flex flex-col items-center justify-center h-64 text-center"
      >
        <div
          class="h-16 w-16 bg-white/5 light:bg-gray-100 rounded-full flex items-center justify-center mb-4"
        >
          <Smartphone class="h-8 w-8 text-white/20 light:text-gray-400" />
        </div>
        <h3 class="text-lg font-medium text-white light:text-gray-900">
          {{ $t("settings.instances.noAccounts") }}
        </h3>
        <p class="text-white/40 max-w-sm mt-2 light:text-gray-500">
          {{ $t("settings.instances.connectFirst") }}
        </p>
        <Button
          variant="outline"
          class="mt-6 border-white/10 hover:bg-white/5 text-emerald-400 light:border-gray-300 light:hover:bg-gray-100"
          @click="createDialogOpen = true"
        >
          {{ $t("settings.instances.connectFirstButton") }}
        </Button>
      </div>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <InstanceCard
          v-for="(instance, index) in instancesStore.instances"
          :key="instance.id"
          :instance="instance"
          :palette-index="index"
          :tag-settings-saving="tagSettingsSaving[instance.id] || false"
          :auto-sync-saving="autoSyncSaving[instance.id] || false"
          :auto-download-incoming-media-saving="
            autoDownloadIncomingMediaSaving[instance.id] || false
          "
          :auto-reject-saving="autoRejectSaving[instance.id] || false"
          :auto-campaign-saving="autoCampaignSaving[instance.id] || false"
          :auto-campaign-uploading="
            autoCampaignUploading[instance.id] || false
          "
          @connect="handleConnect"
          @disconnect="disconnectInstance"
          @edit="openEditDialog"
          @delete="openDeleteDialog"
          @save-tag-settings="handleSaveTagSettings"
          @update-auto-sync="handleAutoSyncUpdate"
          @update-auto-download-incoming-media="
            handleAutoDownloadIncomingMediaUpdate
          "
          @update-auto-reject-settings="handleAutoRejectSettingsUpdate"
          @update-auto-campaign-settings="handleAutoCampaignSettingsUpdate"
          @upload-auto-campaign-media="handleAutoCampaignMediaUpload"
          @clear-auto-campaign-media="handleAutoCampaignMediaClear"
        />
      </div>
    </div>

    <!-- Create Instance Dialog -->
    <Dialog :open="createDialogOpen" @update:open="createDialogOpen = $event">
      <DialogContent
        class="bg-[#1a1a1c] border-white/10 text-white light:bg-white light:border-gray-200 light:text-gray-900 sm:max-w-[425px]"
      >
        <DialogHeader>
          <DialogTitle>{{
            $t("settings.instances.dialog.addAccount")
          }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid gap-2">
            <Label htmlFor="name" class="text-white/70 light:text-gray-700">{{
              $t("settings.instances.dialog.accountName")
            }}</Label>
            <Input
              id="name"
              v-model="newInstanceName"
              :placeholder="$t('settings.instances.dialog.placeholder')"
              class="bg-white/5 border-white/10 text-white placeholder:text-white/20 light:bg-white light:border-gray-300 light:text-gray-900 light:placeholder:text-gray-400"
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            @click="createDialogOpen = false"
            class="border-white/10 hover:bg-white/5 text-white/70 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
            >{{ $t("common.cancel") }}</Button
          >
          <Button
            @click="handleCreate"
            :disabled="isCreating || !newInstanceName"
            class="bg-emerald-600 hover:bg-emerald-700"
          >
            <Loader2 v-if="isCreating" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("common.create") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Edit Instance Dialog -->
    <Dialog
      :open="editDialogOpen"
      @update:open="(open) => !open && closeEditDialog()"
    >
      <DialogContent
        class="bg-[#1a1a1c] border-white/10 text-white light:bg-white light:border-gray-200 light:text-gray-900 sm:max-w-[425px]"
      >
        <DialogHeader>
          <DialogTitle>{{
            $t("settings.instances.dialog.editAccount")
          }}</DialogTitle>
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid gap-2">
            <Label htmlFor="edit-name" class="text-white/70 light:text-gray-700">{{
              $t("settings.instances.dialog.accountName")
            }}</Label>
            <Input
              id="edit-name"
              v-model="editInstanceName"
              :placeholder="$t('settings.instances.dialog.placeholder')"
              class="bg-white/5 border-white/10 text-white placeholder:text-white/20 light:bg-white light:border-gray-300 light:text-gray-900 light:placeholder:text-gray-400"
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            @click="closeEditDialog"
            class="border-white/10 hover:bg-white/5 text-white/70 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
            >{{ $t("common.cancel") }}</Button
          >
          <Button
            @click="handleUpdateName"
            :disabled="isUpdatingName || !editInstanceName.trim()"
            class="bg-emerald-600 hover:bg-emerald-700"
          >
            <Loader2 v-if="isUpdatingName" class="mr-2 h-4 w-4 animate-spin" />
            {{ $t("common.update") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- QR Code Modal -->
    <QRCodeModal
      :open="qrModalOpen"
      @update:open="handleQRModalOpenChange"
      :qr-code="qrCode"
      :timeout="qrTimeout"
      :error-message="qrErrorMessage"
      :refreshing="isRefreshingQR"
      :pairing-code="pairingCode"
      :pairing-phone-number="pairingPhoneNumber"
      :requesting-pair-code="isRequestingPairCode"
      @refresh="handleRegenerateQRCode"
      @request-pair-code="handleRequestPairCode"
    />

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="$t('settings.instances.dialog.deleteAccount')"
      :item-name="deletingInstance?.name"
      :confirm-label="$t('settings.instances.dialog.confirmDelete')"
      :cancel-label="$t('settings.instances.dialog.cancelKeep')"
      @confirm="handleDeleteInstance"
      @cancel="closeDeleteDialog"
    >
      <template #description>
        {{ $t("settings.instances.dialog.deleteDescription") }}
      </template>
      <template #details>
        <div class="rounded-md border border-destructive/30 bg-destructive/5 p-3">
          <div class="flex items-start gap-2">
            <Checkbox
              id="delete-instance-chats"
              :checked="deleteChatsWithInstance"
              @update:checked="deleteChatsWithInstance = $event === true"
            />
            <div class="space-y-1">
              <Label
                for="delete-instance-chats"
                class="cursor-pointer text-white light:text-gray-900"
              >
                {{ $t("settings.instances.dialog.deleteRelatedChatsLabel") }}
              </Label>
              <p class="text-xs text-white/60 light:text-gray-600">
                {{ $t("settings.instances.dialog.deleteRelatedChatsHint") }}
              </p>
            </div>
          </div>
        </div>
      </template>
    </DeleteConfirmDialog>
  </div>
</template>
