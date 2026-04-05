<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { WhatsAppInstance } from "@/types/whatsmeow";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Check, Loader2 } from "lucide-vue-next";
import {
  INSTANCE_TAG_COLOR_PRESETS,
  getInstanceTagLabel,
  getInstanceTagPresetByKey,
  readInstanceTagSettings,
  resolveInstanceTagColorKey,
  resolveInstanceTagDisplayMode,
  type InstanceTagColorKey,
  type InstanceTagDisplayMode,
} from "@/lib/instance-tag";

const props = withDefaults(
  defineProps<{
    instance: WhatsAppInstance;
    paletteIndex?: number;
    saving?: boolean;
  }>(),
  {
    paletteIndex: 0,
    saving: false,
  },
);

const emit = defineEmits<{
  (
    event: "save",
    payload: {
      customLabel: string;
      color: InstanceTagColorKey;
      displayMode: InstanceTagDisplayMode;
    },
  ): void;
}>();

const customLabel = ref("");
const selectedColor = ref<InstanceTagColorKey>("sky");
const selectedDisplayMode = ref<InstanceTagDisplayMode>("name");

function syncFromProps() {
  const settings = readInstanceTagSettings(props.instance);
  customLabel.value = settings.chat_tag_custom_label || "";
  selectedColor.value =
    settings.chat_tag_color ||
    resolveInstanceTagColorKey(props.instance, props.paletteIndex);
  selectedDisplayMode.value =
    settings.chat_tag_display_mode ||
    resolveInstanceTagDisplayMode(props.instance, "name");
}

watch(() => [props.instance.settings, props.paletteIndex], syncFromProps, {
  deep: true,
  immediate: true,
});

const baseCustomLabel = computed(
  () => readInstanceTagSettings(props.instance).chat_tag_custom_label || "",
);
const baseColor = computed(() => {
  const settings = readInstanceTagSettings(props.instance);
  return (
    settings.chat_tag_color ||
    resolveInstanceTagColorKey(props.instance, props.paletteIndex)
  );
});
const baseDisplayMode = computed(() =>
  resolveInstanceTagDisplayMode(props.instance, "name"),
);

const hasChanges = computed(() => {
  return (
    customLabel.value.trim() !== baseCustomLabel.value ||
    selectedColor.value !== baseColor.value ||
    selectedDisplayMode.value !== baseDisplayMode.value
  );
});

const previewLabel = computed(() => {
  const custom = customLabel.value.trim();
  if (selectedDisplayMode.value === "custom") {
    return custom || getInstanceTagLabel(props.instance, "custom");
  }
  if (selectedDisplayMode.value === "phone") {
    return (
      (props.instance.phone_number || "").trim() ||
      custom ||
      getInstanceTagLabel(props.instance, "phone")
    );
  }
  return (
    (props.instance.name || "").trim() ||
    custom ||
    getInstanceTagLabel(props.instance, "name")
  );
});

const previewPreset = computed(() =>
  getInstanceTagPresetByKey(selectedColor.value),
);

function saveSettings() {
  emit("save", {
    customLabel: customLabel.value.trim(),
    color: selectedColor.value,
    displayMode: selectedDisplayMode.value,
  });
}

function selectColor(color: InstanceTagColorKey) {
  selectedColor.value = color;
}
</script>

<template>
  <div class="mt-1 border-t border-border/70 pt-4">
    <div class="mb-2 flex items-center justify-between gap-3">
      <p class="text-xs font-medium text-foreground">
        {{ $t("instances.tags.title") }}
      </p>
      <div
        :class="[
          'inline-flex max-w-[140px] items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-medium',
          previewPreset.badgeClass,
        ]"
        :title="previewLabel"
      >
        <span
          :class="['h-1.5 w-1.5 shrink-0 rounded-full', previewPreset.dotClass]"
        />
        <span class="truncate">{{ previewLabel }}</span>
      </div>
    </div>

    <div class="instance-tag-settings-grid grid gap-3">
      <div class="space-y-1.5">
        <Label class="text-[11px] font-medium text-muted-foreground">{{
          $t("instances.tags.customLabel")
        }}</Label>
        <Input
          v-model="customLabel"
          :placeholder="$t('instances.tags.labelPlaceholder')"
          class="h-9 text-xs"
        />
      </div>

      <div class="space-y-1.5">
        <Label class="text-[11px] font-medium text-muted-foreground">{{
          $t("instances.tags.showAs")
        }}</Label>
        <Select v-model="selectedDisplayMode">
          <SelectTrigger class="h-9 text-xs">
            <SelectValue :placeholder="$t('instances.tags.showAs')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="custom">{{
              $t("instances.tags.modeCustom")
            }}</SelectItem>
            <SelectItem value="phone">{{
              $t("instances.tags.modePhone")
            }}</SelectItem>
            <SelectItem value="name">{{
              $t("instances.tags.modeName")
            }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <div class="instance-tag-settings-actions mt-3 flex flex-col gap-3">
      <div class="min-w-0 flex-1 space-y-1.5">
        <Label class="text-[11px] font-medium text-muted-foreground">{{
          $t("instances.tags.tagColor")
        }}</Label>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="preset in INSTANCE_TAG_COLOR_PRESETS"
            :key="preset.key"
            type="button"
            class="flex h-6 w-6 items-center justify-center rounded-full border-2 transition-transform hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/25 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            :class="[
              preset.swatchClass,
              selectedColor === preset.key
                ? 'border-background shadow-sm ring-2 ring-offset-0'
                : 'border-transparent',
              selectedColor === preset.key ? preset.ringClass : '',
            ]"
            :title="preset.label"
            @click="selectColor(preset.key)"
          >
            <Check
              v-if="selectedColor === preset.key"
              class="h-3.5 w-3.5 text-white"
            />
          </button>
        </div>
      </div>

      <Button
        size="sm"
        variant="outline"
        class="instance-tag-settings-save h-9 w-full text-xs"
        :disabled="!hasChanges || saving"
        @click="saveSettings"
      >
        <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
        {{ $t("instances.tags.saveSettings") }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
@container (min-width: 25rem) {
  .instance-tag-settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .instance-tag-settings-actions {
    flex-direction: row;
    flex-wrap: wrap;
    align-items: flex-end;
    justify-content: space-between;
  }

  .instance-tag-settings-save {
    min-width: 180px;
    width: auto;
  }
}
</style>
