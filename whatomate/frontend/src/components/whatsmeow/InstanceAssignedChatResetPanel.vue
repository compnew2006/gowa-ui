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
    class="h-auto min-h-9 w-full justify-center rounded-xl border-dashed bg-background/60 px-3 py-2 text-center leading-4"
    :disabled="saving"
    @click="dialogOpen = true"
  >
    <Loader2 v-if="saving" class="h-3.5 w-3.5 mr-2 animate-spin" />
    <Settings2 v-else class="h-3.5 w-3.5 mr-2" />
    {{ $t("instances.assigned_chat_reset.configureButton") }}
  </Button>

  <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
    <DialogContent class="sm:max-w-[520px]">
      <DialogHeader>
        <DialogTitle>{{
          $t("instances.assigned_chat_reset.title")
        }}</DialogTitle>
        <DialogDescription class="text-muted-foreground">
          {{ $t("instances.assigned_chat_reset.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <div class="space-y-2">
          <Label>{{ $t("instances.assigned_chat_reset.schedule") }}</Label>
          <p class="text-xs text-muted-foreground">
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
          <Label>{{ $t("instances.assigned_chat_reset.customHour") }}</Label>
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

        <p class="text-xs text-muted-foreground">
          {{
            $t("instances.assigned_chat_reset.timezoneHint", {
              timezone: organizationTimezone || "UTC",
            })
          }}
        </p>
      </div>

      <DialogFooter class="gap-2">
        <Button variant="outline" @click="dialogOpen = false">
          {{ $t("common.cancel") }}
        </Button>
        <Button :disabled="saving" @click="handleSave">
          <Loader2 v-if="saving" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t("common.save") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
