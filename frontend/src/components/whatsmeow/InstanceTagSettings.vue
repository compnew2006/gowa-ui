<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { WhatsAppInstance } from "@/types/whatsmeow";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
const selectedColor = ref<InstanceTagColorKey>("emerald");
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
  <div class="mt-4 border-t border-white/[0.08] pt-3 light:border-gray-200">
    <div class="mb-2 flex items-center justify-between gap-3">
      <p class="text-xs font-medium text-white/70 light:text-gray-700">
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

    <div class="space-y-1.5">
      <Label
        class="text-[11px] font-medium text-white/50 light:text-gray-500"
        >{{ $t("instances.tags.customLabel") }}</Label
      >
      <Input
        v-model="customLabel"
        :placeholder="$t('instances.tags.labelPlaceholder')"
        class="h-8 bg-white/[0.04] text-xs text-white placeholder:text-white/25 light:bg-white light:text-gray-900 light:placeholder:text-gray-400"
      />
    </div>

    <div class="mt-3 space-y-1.5">
      <Label
        class="text-[11px] font-medium text-white/50 light:text-gray-500"
        >{{ $t("instances.tags.showAs") }}</Label
      >
      <select
        v-model="selectedDisplayMode"
        class="h-8 w-full rounded-md border border-white/[0.12] bg-black/20 px-2 text-xs text-white focus:outline-none focus:ring-1 focus:ring-emerald-400 light:border-gray-300 light:bg-white light:text-gray-800"
      >
        <option value="custom">{{ $t("instances.tags.modeCustom") }}</option>
        <option value="phone">{{ $t("instances.tags.modePhone") }}</option>
        <option value="name">{{ $t("instances.tags.modeName") }}</option>
      </select>
    </div>

    <div class="mt-3 space-y-1.5">
      <Label
        class="text-[11px] font-medium text-white/50 light:text-gray-500"
        >{{ $t("instances.tags.tagColor") }}</Label
      >
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="preset in INSTANCE_TAG_COLOR_PRESETS"
          :key="preset.key"
          type="button"
          class="flex h-6 w-6 items-center justify-center rounded-full border-2 transition-transform hover:scale-105"
          :class="[
            preset.swatchClass,
            selectedColor === preset.key
              ? 'border-white ring-2 ring-offset-0'
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
      class="mt-3 h-8 w-full border-white/[0.1] text-xs text-white/80 hover:bg-white/[0.08] light:border-gray-300 light:text-gray-700 light:hover:bg-gray-100"
      :disabled="!hasChanges || saving"
      @click="saveSettings"
    >
      <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
      {{ $t("instances.tags.saveSettings") }}
    </Button>
  </div>
</template>
