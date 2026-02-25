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
import { Textarea } from "@/components/ui/textarea";
import {
  bypassContactsFromEditorValue,
  bypassContactsToEditorValue,
  cloneAutoRejectSettings,
  normalizeAutoRejectCallSettings,
  type AutoRejectCallSettings,
} from "@/lib/instance-auto-reject";
import { Loader2, Settings2 } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useI18n } from "vue-i18n";
import WhatsAppRichTextEditor from "@/components/chat/WhatsAppRichTextEditor.vue";

const { t } = useI18n();

const props = defineProps<{
  settings: AutoRejectCallSettings;
  saving?: boolean;
}>();

const emit = defineEmits<{
  (e: "save", value: AutoRejectCallSettings): void;
}>();

const dialogOpen = ref(false);
const localSettings = ref<AutoRejectCallSettings>(
  cloneAutoRejectSettings(props.settings),
);
const bypassEditor = ref("");

const dayOptions = [
  { value: 0, label: "instances.auto_reject.days.sun" },
  { value: 1, label: "instances.auto_reject.days.mon" },
  { value: 2, label: "instances.auto_reject.days.tue" },
  { value: 3, label: "instances.auto_reject.days.wed" },
  { value: 4, label: "instances.auto_reject.days.thu" },
  { value: 5, label: "instances.auto_reject.days.fri" },
  { value: 6, label: "instances.auto_reject.days.sat" },
];
const autoRejectPlaceholderTokens = [
  "{customer_name}",
  "{chat_id}",
  "{agent_name}",
  "{organization_name}",
  "{contact_name}",
  "{phone_number}",
];

watch(
  () => props.settings,
  (value) => {
    localSettings.value = cloneAutoRejectSettings(
      normalizeAutoRejectCallSettings(value),
    );
    bypassEditor.value = bypassContactsToEditorValue(
      localSettings.value.bypass_contacts,
    );
  },
  { immediate: true, deep: true },
);

function isDaySelected(day: number): boolean {
  return localSettings.value.schedule.days.includes(day);
}

function toggleDay(day: number) {
  const daySet = new Set(localSettings.value.schedule.days);
  if (daySet.has(day)) {
    daySet.delete(day);
  } else {
    daySet.add(day);
  }
  localSettings.value.schedule.days = [...daySet].sort((a, b) => a - b);
}

function appendAutoRejectPlaceholder(token: string) {
  localSettings.value.message = `${localSettings.value.message || ""}${token}`;
}

function handleSave() {
  localSettings.value.bypass_contacts = bypassContactsFromEditorValue(
    bypassEditor.value,
  );

  if (
    localSettings.value.mode === "with_message" &&
    !localSettings.value.message.trim()
  ) {
    toast.error(t("instances.auto_reject.validation.messageRequired"));
    return;
  }

  if (localSettings.value.schedule.type === "custom_hours") {
    const timePattern = /^([01]\d|2[0-3]):[0-5]\d$/;
    if (
      !timePattern.test(localSettings.value.schedule.start) ||
      !timePattern.test(localSettings.value.schedule.end)
    ) {
      toast.error(t("instances.auto_reject.validation.invalidTime"));
      return;
    }
    if (localSettings.value.schedule.days.length === 0) {
      toast.error(t("instances.auto_reject.validation.dayRequired"));
      return;
    }
  }

  emit("save", normalizeAutoRejectCallSettings(localSettings.value));
  dialogOpen.value = false;
}
</script>

<template>
  <Button
    variant="outline"
    size="sm"
    class="w-full border-white/10 text-white/70 hover:bg-white/5 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
    :disabled="saving"
    @click="dialogOpen = true"
  >
    <Loader2 v-if="saving" class="h-3.5 w-3.5 mr-2 animate-spin" />
    <Settings2 v-else class="h-3.5 w-3.5 mr-2" />
    {{ $t("instances.auto_reject.configureButton") }}
  </Button>

  <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
    <DialogContent
      class="sm:max-w-[640px] bg-[#1a1a1c] border-white/10 text-white light:bg-white light:border-gray-200 light:text-gray-900"
    >
      <DialogHeader>
        <DialogTitle>{{ $t("instances.auto_reject.title") }}</DialogTitle>
        <DialogDescription class="text-white/50 light:text-gray-500">
          {{ $t("instances.auto_reject.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2 max-h-[70vh] overflow-y-auto pr-1">
        <div class="rounded-lg border border-white/10 light:border-gray-200 p-3 space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div>
              <Label class="text-white/80 light:text-gray-800">{{
                $t("instances.auto_reject.enable")
              }}</Label>
              <p class="text-xs text-white/45 light:text-gray-500">
                {{ $t("instances.auto_reject.enableDesc") }}
              </p>
            </div>
            <Switch v-model:checked="localSettings.enabled" />
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <div class="space-y-1.5">
              <Label>{{ $t("instances.auto_reject.rejectIndividual") }}</Label>
              <Switch v-model:checked="localSettings.reject_individual_calls" />
            </div>
            <div class="space-y-1.5">
              <Label>{{ $t("instances.auto_reject.rejectGroup") }}</Label>
              <Switch v-model:checked="localSettings.reject_group_calls" />
            </div>
          </div>
        </div>

        <div class="space-y-2">
          <Label>{{ $t("instances.auto_reject.replyMode") }}</Label>
          <Select v-model="localSettings.mode">
            <SelectTrigger class="bg-white/5 border-white/10 light:bg-white light:border-gray-300">
              <SelectValue
                :placeholder="
                  $t('common.selectPlaceholder', {
                    resource: $t('instances.auto_reject.replyMode'),
                  })
                "
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="without_message">{{
                $t("instances.auto_reject.modeWithoutMessage")
              }}</SelectItem>
              <SelectItem value="with_message">{{
                $t("instances.auto_reject.modeWithMessage")
              }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div v-if="localSettings.mode === 'with_message'" class="space-y-2">
          <Label>{{ $t("instances.auto_reject.automatedMessage") }}</Label>
          <div class="flex flex-wrap items-center gap-1.5">
            <Button
              v-for="token in autoRejectPlaceholderTokens"
              :key="token"
              type="button"
              variant="outline"
              size="sm"
              class="h-7 px-2 text-xs"
              :disabled="saving"
              @click="appendAutoRejectPlaceholder(token)"
            >
              {{ token }}
            </Button>
          </div>
          <WhatsAppRichTextEditor
            v-model="localSettings.message"
            :rows="3"
            :placeholder="$t('instances.auto_reject.messagePlaceholder')"
          />
          <p class="text-xs text-white/45 light:text-gray-500">
            {{ $t("instances.auto_reject.placeholderHint") }}
          </p>
        </div>

        <div class="space-y-2">
          <Label>{{ $t("instances.auto_reject.schedule") }}</Label>
          <Select v-model="localSettings.schedule.type">
            <SelectTrigger class="bg-white/5 border-white/10 light:bg-white light:border-gray-300">
              <SelectValue
                :placeholder="
                  $t('common.selectPlaceholder', {
                    resource: $t('instances.auto_reject.schedule'),
                  })
                "
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="always">{{
                $t("instances.auto_reject.scheduleAlways")
              }}</SelectItem>
              <SelectItem value="custom_hours">{{
                $t("instances.auto_reject.scheduleCustom")
              }}</SelectItem>
              <SelectItem value="while_in_other_calls">{{
                $t("instances.auto_reject.scheduleOtherCalls")
              }}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div
          v-if="localSettings.schedule.type === 'custom_hours'"
          class="rounded-lg border border-white/10 light:border-gray-200 p-3 space-y-3"
        >
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label>{{ $t("instances.auto_reject.start") }}</Label>
              <Input
                v-model="localSettings.schedule.start"
                type="time"
                class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
              />
            </div>
            <div class="space-y-1.5">
              <Label>{{ $t("instances.auto_reject.end") }}</Label>
              <Input
                v-model="localSettings.schedule.end"
                type="time"
                class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
              />
            </div>
          </div>

          <div class="space-y-1.5">
            <Label>{{ $t("instances.auto_reject.timezone") }}</Label>
            <Input
              v-model="localSettings.schedule.timezone"
              class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
              :placeholder="$t('instances.auto_reject.timezonePlaceholder')"
            />
          </div>

          <div class="space-y-1.5">
            <Label>{{ $t("instances.auto_reject.activeDays") }}</Label>
            <div class="flex flex-wrap gap-2">
              <Button
                v-for="day in dayOptions"
                :key="day.value"
                type="button"
                variant="outline"
                size="sm"
                class="border-white/15"
                :class="
                  isDaySelected(day.value)
                    ? 'bg-emerald-500/20 text-emerald-300 border-emerald-400/40'
                    : 'text-white/60 light:text-gray-600'
                "
                @click="toggleDay(day.value)"
              >
                {{ $t(day.label) }}
              </Button>
            </div>
          </div>
        </div>

        <div class="space-y-2">
          <Label>{{ $t("instances.auto_reject.bypassContacts") }}</Label>
          <Textarea
            v-model="bypassEditor"
            :rows="3"
            class="bg-white/5 border-white/10 light:bg-white light:border-gray-300"
            :placeholder="$t('instances.auto_reject.bypassPlaceholder')"
          />
          <p class="text-xs text-white/45 light:text-gray-500">
            {{ $t("instances.auto_reject.bypassDesc") }}
          </p>
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
          {{ $t("instances.auto_reject.saveSettings") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
