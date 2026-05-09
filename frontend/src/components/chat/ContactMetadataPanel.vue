<script setup lang="ts">
import { computed } from "vue";
import MetadataSection from "@/components/chat/MetadataSection.vue";
import { formatLabel } from "@/lib/utils";

const props = defineProps<{
  metadata: Record<string, any> | null | undefined;
}>();

const hasMetadata = computed(() => {
  const md = props.metadata;
  return md && typeof md === "object" && Object.keys(md).length > 0;
});

const metadataPrimitives = computed(() => {
  if (!hasMetadata.value) return [];
  return Object.entries(props.metadata!).filter(
    ([, v]) => v === null || typeof v !== "object",
  );
});

const metadataSections = computed(() => {
  if (!hasMetadata.value) return [];
  return Object.entries(props.metadata!).filter(
    ([, v]) => v !== null && typeof v === "object",
  );
});
</script>

<template>
  <div v-if="hasMetadata" class="space-y-3">
    <MetadataSection
      v-if="metadataPrimitives.length > 0"
      label="General"
      :data="Object.fromEntries(metadataPrimitives)"
    />
    <MetadataSection
      v-for="[key, val] in metadataSections"
      :key="key"
      :label="formatLabel(key)"
      :data="val"
    />
  </div>
</template>
