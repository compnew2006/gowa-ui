<script setup lang="ts">
import { computed, ref, watch } from "vue";
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
  getAutoCampaignEvaluationSchedule,
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
const schedule = computed(() =>
  getAutoCampaignEvaluationSchedule(localSettings.value),
);

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

function formatScheduleValue(value?: string) {
  if (!value) {
    return t("instances.auto_campaign.pendingFirstRun");
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return t("instances.auto_campaign.pendingFirstRun");
  }

  return parsed.toLocaleString();
}
</script>

<template>
  <Button
    variant="outline"
    size="sm"
    class="h-auto min-h-9 w-full justify-center rounded-xl border-dashed bg-background/60 px-3 py-2 text-center leading-4"
    :disabled="saving || uploading"
    @click="dialogOpen = true"
  >
    <Loader2 v-if="saving || uploading" class="h-3.5 w-3.5 mr-2 animate-spin" />
    <Settings2 v-else class="h-3.5 w-3.5 mr-2" />
    {{ $t("instances.auto_campaign.configureButton") }}
  </Button>

  <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
    <DialogContent class="sm:max-w-[640px]">
      <DialogHeader>
        <DialogTitle>{{ $t("instances.auto_campaign.title") }}</DialogTitle>
        <DialogDescription class="text-muted-foreground">
          {{ $t("instances.auto_campaign.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2 max-h-[70vh] overflow-y-auto pr-1">
        <div
          class="space-y-2 rounded-lg border border-primary/20 bg-primary/5 p-3 text-sm"
        >
          <p class="text-foreground">
            {{ $t("instances.auto_campaign.eligibilityHint") }}
          </p>
          <div class="grid gap-2 text-xs text-muted-foreground md:grid-cols-2">
            <p>
              <span class="font-medium text-foreground">{{
                $t("instances.auto_campaign.lastEvaluation")
              }}</span>
              {{ formatScheduleValue(schedule.lastEvaluationAt) }}
            </p>
            <p>
              <span class="font-medium text-foreground">{{
                $t("instances.auto_campaign.nextEvaluation")
              }}</span>
              {{ formatScheduleValue(schedule.nextEvaluationAt) }}
            </p>
          </div>
        </div>

        <div
          class="space-y-3 rounded-xl border border-border/70 bg-muted/20 p-4"
        >
          <div class="flex items-center justify-between gap-2">
            <div>
              <Label>{{ $t("instances.auto_campaign.enable") }}</Label>
              <p class="text-xs text-muted-foreground">
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
              class="bg-background"
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
              class="bg-background"
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
              class="bg-background"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t("instances.auto_campaign.delayToMinutes") }}</Label>
            <Input
              v-model.number="localSettings.max_delay_minutes"
              type="number"
              min="0"
              step="1"
              class="bg-background"
            />
          </div>
        </div>
        <p class="text-xs text-muted-foreground">
          {{ $t("instances.auto_campaign.delayHint") }}
        </p>

        <div class="space-y-2">
          <Label>{{ $t("instances.auto_campaign.targetStatus") }}</Label>
          <Select v-model="localSettings.target_status">
            <SelectTrigger class="bg-background">
              <SelectValue
                :placeholder="
                  $t('common.selectPlaceholder', {
                    resource: $t('instances.auto_campaign.targetStatus'),
                  })
                "
              />
            </SelectTrigger>
            <SelectContent>
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
          <p class="text-xs text-muted-foreground">
            {{ $t("instances.auto_campaign.placeholderHint") }}
          </p>
        </div>

        <div
          class="space-y-3 rounded-xl border border-border/70 bg-muted/20 p-4"
        >
          <div class="flex items-center justify-between gap-2">
            <div>
              <Label>{{ $t("instances.auto_campaign.media") }}</Label>
              <p class="text-xs text-muted-foreground">
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
                class="border-destructive/30 text-destructive hover:bg-destructive/10 hover:text-destructive"
                :disabled="uploading"
                @click="clearMedia"
              >
                <Trash2 class="h-3.5 w-3.5 mr-2" />
                {{ $t("instances.auto_campaign.removeMedia") }}
              </Button>
            </div>
          </div>
          <p class="text-xs text-muted-foreground">
            {{ $t("instances.auto_campaign.mediaHint") }}
          </p>

          <div
            v-if="localSettings.media_local_path"
            class="rounded-xl border border-border/70 bg-background/80 p-3 text-xs text-foreground"
          >
            <div class="flex items-center gap-2">
              <Paperclip class="h-3.5 w-3.5" />
              <span class="truncate">{{
                localSettings.media_filename || localSettings.media_local_path
              }}</span>
            </div>
            <p class="mt-1 text-muted-foreground">
              {{ localSettings.media_mime_type }}
            </p>
          </div>
        </div>
      </div>

      <DialogFooter class="gap-2">
        <Button variant="outline" @click="dialogOpen = false">{{
          $t("common.cancel")
        }}</Button>
        <Button :disabled="saving" @click="handleSave">
          <Loader2 v-if="saving" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t("instances.auto_campaign.saveSettings") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
