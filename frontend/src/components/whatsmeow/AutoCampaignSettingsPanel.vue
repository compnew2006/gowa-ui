<script setup lang="ts">
import { ref, watch } from "vue";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import WhatsAppRichTextEditor from "@/components/chat/WhatsAppRichTextEditor.vue";
import {
  cloneAutoCampaignSettings,
  normalizeAutoCampaignSettings,
  type AutoCampaignSettings,
} from "@/lib/instance-auto-campaign";
import { Loader2, Paperclip, Settings2, Trash2, Upload } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const props = defineProps<{
  settings: AutoCampaignSettings;
  saving?: boolean;
  uploading?: boolean;
}>();

const emit = defineEmits<{
  (e: "save", value: AutoCampaignSettings): void;
  (e: "upload-media", file: File): void;
  (e: "clear-media"): void;
}>();

const dialogOpen = ref(false);
const localSettings = ref<AutoCampaignSettings>(
  cloneAutoCampaignSettings(props.settings),
);
const fileInput = ref<HTMLInputElement | null>(null);

watch(
  () => props.settings,
  (value) => {
    localSettings.value = cloneAutoCampaignSettings(
      normalizeAutoCampaignSettings(value),
    );
  },
  { immediate: true, deep: true },
);

function handleSave() {
  if (localSettings.value.enabled && !localSettings.value.message.trim()) {
    toast.error(t("instances.auto_campaign.validation.messageRequired"));
    return;
  }

  const interval = Number(localSettings.value.interval_days);
  if (!Number.isFinite(interval) || interval < 1 || interval > 365) {
    toast.error(t("instances.auto_campaign.validation.intervalInvalid"));
    return;
  }

  const minDelay = Number(localSettings.value.min_delay_minutes);
  const maxDelay = Number(localSettings.value.max_delay_minutes);
  if (
    !Number.isFinite(minDelay) ||
    !Number.isFinite(maxDelay) ||
    minDelay < 0 ||
    maxDelay < 0 ||
    minDelay > maxDelay
  ) {
    toast.error(t("instances.auto_campaign.validation.delayInvalid"));
    return;
  }

  localSettings.value.interval_days = Math.floor(interval);
  localSettings.value.min_delay_minutes = Math.floor(minDelay);
  localSettings.value.max_delay_minutes = Math.floor(maxDelay);
  emit("save", normalizeAutoCampaignSettings(localSettings.value));
  dialogOpen.value = false;
}

function triggerFilePicker() {
  fileInput.value?.click();
}

function handleFileSelected(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }
  emit("upload-media", file);
  input.value = "";
}

function clearMedia() {
  emit("clear-media");
}
</script>

<template>
  <Button
    variant="outline"
    size="sm"
    class="w-full border-white/10 text-white/70 hover:bg-white/5 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
    :disabled="saving || uploading"
    @click="dialogOpen = true"
  >
    <Loader2
      v-if="saving || uploading"
      class="h-3.5 w-3.5 mr-2 animate-spin"
    />
    <Settings2 v-else class="h-3.5 w-3.5 mr-2" />
    {{ $t("instances.auto_campaign.configureButton") }}
  </Button>

  <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
    <DialogContent
      class="sm:max-w-[640px] bg-[#1a1a1c] border-white/10 text-white light:bg-white light:border-gray-200 light:text-gray-900"
    >
      <DialogHeader>
        <DialogTitle>{{ $t("instances.auto_campaign.title") }}</DialogTitle>
        <DialogDescription class="text-white/50 light:text-gray-500">
          {{ $t("instances.auto_campaign.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2 max-h-[70vh] overflow-y-auto pr-1">
        <div class="rounded-lg border border-white/10 light:border-gray-200 p-3 space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div>
              <Label class="text-white/80 light:text-gray-800">{{
                $t("instances.auto_campaign.enable")
              }}</Label>
              <p class="text-xs text-white/45 light:text-gray-500">
                {{ $t("instances.auto_campaign.enableDesc") }}
              </p>
            </div>
            <Switch v-model:checked="localSettings.enabled" />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <div class="space-y-2">
            <Label>{{ $t("instances.auto_campaign.namePrefix") }}</Label>
            <Input
              v-model="localSettings.name_prefix"
              class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
              :placeholder="$t('instances.auto_campaign.namePrefixPlaceholder')"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t("instances.auto_campaign.intervalDays") }}</Label>
            <Input
              v-model.number="localSettings.interval_days"
              type="number"
              min="1"
              max="365"
              class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
            />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <div class="space-y-2">
            <Label>{{ $t("instances.auto_campaign.delayFromMinutes") }}</Label>
            <Input
              v-model.number="localSettings.min_delay_minutes"
              type="number"
              min="0"
              step="1"
              class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t("instances.auto_campaign.delayToMinutes") }}</Label>
            <Input
              v-model.number="localSettings.max_delay_minutes"
              type="number"
              min="0"
              step="1"
              class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
            />
          </div>
        </div>
        <p class="text-xs text-white/45 light:text-gray-500">
          {{ $t("instances.auto_campaign.delayHint") }}
        </p>

        <div class="space-y-2">
          <Label>{{ $t("instances.auto_campaign.targetStatus") }}</Label>
          <Select v-model="localSettings.target_status">
            <SelectTrigger class="bg-white/5 border-white/10 light:bg-white light:border-gray-300">
              <SelectValue
                :placeholder="
                  $t('common.selectPlaceholder', {
                    resource: $t('instances.auto_campaign.targetStatus'),
                  })
                "
              />
            </SelectTrigger>
            <SelectContent class="bg-[#1a1a1c] border-white/10 text-white light:bg-white light:border-gray-200 light:text-gray-900">
              <SelectItem value="draft">{{
                $t("instances.auto_campaign.statusDraft")
              }}</SelectItem>
              <SelectItem value="run">{{
                $t("instances.auto_campaign.statusRun")
              }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="space-y-2">
          <Label
            >{{ $t("instances.auto_campaign.message") }}
            <span class="text-destructive">*</span></Label
          >
          <WhatsAppRichTextEditor
            v-model="localSettings.message"
            :placeholder="$t('instances.auto_campaign.messagePlaceholder')"
            :rows="5"
          />
          <p class="text-xs text-white/45 light:text-gray-500">
            {{ $t("instances.auto_campaign.placeholderHint") }}
          </p>
        </div>

        <div class="rounded-lg border border-white/10 light:border-gray-200 p-3 space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div>
              <Label class="text-white/80 light:text-gray-800">{{
                $t("instances.auto_campaign.media")
              }}</Label>
              <p class="text-xs text-white/45 light:text-gray-500">
                {{ $t("instances.auto_campaign.mediaDesc") }}
              </p>
            </div>
            <div class="flex items-center gap-2">
              <input
                ref="fileInput"
                type="file"
                class="hidden"
                accept="*/*"
                @change="handleFileSelected"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="border-white/15 text-white/70 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
                :disabled="uploading"
                @click="triggerFilePicker"
              >
                <Loader2
                  v-if="uploading"
                  class="h-3.5 w-3.5 mr-2 animate-spin"
                />
                <Upload v-else class="h-3.5 w-3.5 mr-2" />
                {{ $t("instances.auto_campaign.uploadMedia") }}
              </Button>
              <Button
                v-if="localSettings.media_local_path"
                type="button"
                variant="outline"
                size="sm"
                class="border-red-500/40 text-red-300 light:border-red-300 light:text-red-600 light:hover:bg-red-50"
                :disabled="uploading"
                @click="clearMedia"
              >
                <Trash2 class="h-3.5 w-3.5 mr-2" />
                {{ $t("instances.auto_campaign.removeMedia") }}
              </Button>
            </div>
          </div>
          <p class="text-xs text-white/45 light:text-gray-500">
            {{ $t("instances.auto_campaign.mediaHint") }}
          </p>

          <div
            v-if="localSettings.media_local_path"
            class="rounded-md border border-white/10 bg-white/5 p-2 text-xs text-white/70 light:border-gray-200 light:bg-gray-50 light:text-gray-700"
          >
            <div class="flex items-center gap-2">
              <Paperclip class="h-3.5 w-3.5" />
              <span class="truncate">{{
                localSettings.media_filename || localSettings.media_local_path
              }}</span>
            </div>
            <p class="mt-1 text-white/45 light:text-gray-500">{{ localSettings.media_mime_type }}</p>
          </div>
        </div>
      </div>

      <DialogFooter class="gap-2">
        <Button
          variant="outline"
          class="border-white/10 text-white/70 light:border-gray-300 light:text-gray-700"
          @click="dialogOpen = false"
          >{{ $t("common.cancel") }}</Button
        >
        <Button
          class="bg-emerald-600 hover:bg-emerald-700"
          :disabled="saving"
          @click="handleSave"
        >
          <Loader2 v-if="saving" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t("instances.auto_campaign.saveSettings") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
