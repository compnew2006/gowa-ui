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
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Loader2, Settings2 } from "lucide-vue-next";
import {
  cloneInstanceAssignedChatResetSettings,
  sanitizeInstanceAssignedChatResetSettings,
  type InstanceAssignedChatResetSettings,
} from "@/lib/instance-assigned-chat-reset";

const props = defineProps<{
  settings: InstanceAssignedChatResetSettings;
  saving?: boolean;
  organizationTimezone?: string;
}>();

const emit = defineEmits<{
  (e: "save", value: InstanceAssignedChatResetSettings): void;
}>();

const dialogOpen = ref(false);
const localSettings = ref<InstanceAssignedChatResetSettings>(
  cloneInstanceAssignedChatResetSettings(props.settings),
);

const hourOptions = computed(() =>
  Array.from({ length: 24 }, (_, hour) => ({
    value: String(hour),
    label: `${String(hour).padStart(2, "0")}:00`,
  })),
);

function syncLocalSettings(value: InstanceAssignedChatResetSettings) {
  localSettings.value = cloneInstanceAssignedChatResetSettings(value);
}

watch(
  () => props.settings,
  (value) => {
    syncLocalSettings(value);
  },
  { immediate: true, deep: true },
);

watch(dialogOpen, (isOpen) => {
  if (isOpen) {
    syncLocalSettings(props.settings);
  }
});

function handleSave() {
  emit("save", sanitizeInstanceAssignedChatResetSettings(localSettings.value));
  dialogOpen.value = false;
}
</script>

<template>
  <Button
    variant="outline"
    size="sm"
    class="h-auto min-h-9 w-full whitespace-normal border-white/10 px-3 py-2 text-center leading-4 text-white/70 hover:bg-white/5 light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
    :disabled="saving"
    @click="dialogOpen = true"
  >
    <Loader2 v-if="saving" class="h-3.5 w-3.5 mr-2 animate-spin" />
    <Settings2 v-else class="h-3.5 w-3.5 mr-2" />
    {{ $t("instances.assigned_chat_reset.configureButton") }}
  </Button>

  <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
    <DialogContent
      class="sm:max-w-[520px] bg-[#1a1a1c] border-white/10 text-white light:bg-white light:border-gray-200 light:text-gray-900"
    >
      <DialogHeader>
        <DialogTitle>{{
          $t("instances.assigned_chat_reset.title")
        }}</DialogTitle>
        <DialogDescription class="text-white/50 light:text-gray-500">
          {{ $t("instances.assigned_chat_reset.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <div class="space-y-2">
          <Label class="text-white/80 light:text-gray-800">{{
            $t("instances.assigned_chat_reset.schedule")
          }}</Label>
          <p class="text-xs text-white/45 light:text-gray-500">
            {{ $t("instances.assigned_chat_reset.scheduleDesc") }}
          </p>
          <Select v-model="localSettings.mode">
            <SelectTrigger class="w-full border-input bg-input text-foreground">
              <SelectValue
                :placeholder="$t('instances.assigned_chat_reset.schedule')"
              />
            </SelectTrigger>
            <SelectContent
              class="border-border bg-popover text-popover-foreground"
            >
              <SelectItem
                value="midnight"
                class="text-foreground/80 focus:bg-accent focus:text-foreground"
              >
                {{ $t("instances.assigned_chat_reset.defaultMidnight") }}
              </SelectItem>
              <SelectItem
                value="custom_hour"
                class="text-foreground/80 focus:bg-accent focus:text-foreground"
              >
                {{ $t("instances.assigned_chat_reset.customHourOption") }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div v-if="localSettings.mode === 'custom_hour'" class="space-y-2">
          <Label class="text-white/80 light:text-gray-800">{{
            $t("instances.assigned_chat_reset.customHour")
          }}</Label>
          <Select
            :model-value="String(localSettings.hour)"
            @update:model-value="
              (value: unknown) => {
                if (typeof value === 'string') {
                  localSettings.hour = Number(value);
                }
              }
            "
          >
            <SelectTrigger class="w-full border-input bg-input text-foreground">
              <SelectValue
                :placeholder="$t('instances.assigned_chat_reset.customHour')"
              />
            </SelectTrigger>
            <SelectContent
              class="border-border bg-popover text-popover-foreground"
            >
              <SelectItem
                v-for="option in hourOptions"
                :key="option.value"
                :value="option.value"
                class="text-foreground/80 focus:bg-accent focus:text-foreground"
              >
                {{ option.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <p class="text-xs text-white/45 light:text-gray-500">
          {{
            $t("instances.assigned_chat_reset.timezoneHint", {
              timezone: organizationTimezone || "UTC",
            })
          }}
        </p>
      </div>

      <DialogFooter class="gap-2">
        <Button
          variant="outline"
          class="border-white/10 text-white/70 light:border-gray-300 light:text-gray-700"
          @click="dialogOpen = false"
        >
          {{ $t("common.cancel") }}
        </Button>
        <Button
          class="bg-emerald-600 hover:bg-emerald-700 text-white"
          :disabled="saving"
          @click="handleSave"
        >
          <Loader2 v-if="saving" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t("common.save") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
