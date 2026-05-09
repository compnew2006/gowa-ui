<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown, User } from "lucide-vue-next";

interface PanelFieldConfig {
  key: string;
  label: string;
  order: number;
  display_type?: "text" | "badge" | "tag";
  color?: "default" | "success" | "warning" | "error" | "info";
}

interface PanelSection {
  id: string;
  label: string;
  columns: number;
  collapsible: boolean;
  default_collapsed: boolean;
  order: number;
  fields: PanelFieldConfig[];
}

interface PanelConfig {
  sections: PanelSection[];
}

interface SessionData {
  session_id?: string;
  flow_id?: string;
  flow_name?: string;
  session_data: Record<string, any>;
  panel_config: PanelConfig;
}

const props = defineProps<{
  sessionData?: SessionData | null;
}>();

const collapsedSections = ref<Record<string, boolean>>({});

watch(
  () => props.sessionData,
  (newData) => {
    if (newData?.panel_config?.sections) {
      for (const section of newData.panel_config.sections) {
        if (collapsedSections.value[section.id] === undefined) {
          collapsedSections.value[section.id] = section.default_collapsed;
        }
      }
    }
  },
  { immediate: true },
);

function toggleSection(sectionId: string) {
  collapsedSections.value[sectionId] = !collapsedSections.value[sectionId];
}

function isSectionCollapsed(sectionId: string): boolean {
  return collapsedSections.value[sectionId] ?? false;
}

function getFieldValue(key: string): string {
  if (!props.sessionData?.session_data) return "-";
  const value = props.sessionData.session_data[key];
  if (value === undefined || value === null || value === "") return "-";
  return String(value);
}

function getColorClass(color?: string): string {
  switch (color) {
    case "success":
      return "bg-primary/12 text-primary";
    case "warning":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400";
    case "error":
      return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";
    case "info":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400";
    default:
      return "bg-muted text-muted-foreground";
  }
}

const sortedSections = computed(() => {
  if (!props.sessionData?.panel_config?.sections) return [];
  return [...props.sessionData.panel_config.sections].sort(
    (a, b) => a.order - b.order,
  );
});
</script>

<template>
  <div>
    <div
      v-if="!sessionData || sortedSections.length === 0"
      class="text-center py-6 text-muted-foreground border-t"
    >
      <User class="h-8 w-8 mx-auto mb-2 opacity-50" />
      <p class="text-sm">No data configured</p>
      <p class="text-xs mt-1">
        Configure panel display in the chatbot flow settings.
      </p>
    </div>

    <template v-else>
      <div v-if="sessionData?.flow_name" class="flex items-center gap-2">
        <Badge variant="outline" class="text-xs">
          {{ sessionData?.flow_name }}
        </Badge>
      </div>

      <div
        v-for="section in sortedSections"
        :key="section.id"
        class="space-y-2 border-t pt-4"
      >
        <Collapsible
          v-if="section.collapsible"
          :open="!isSectionCollapsed(section.id)"
          @update:open="toggleSection(section.id)"
        >
          <CollapsibleTrigger
            class="flex items-center justify-between w-full py-2 text-sm font-medium hover:text-primary transition-colors"
            :aria-label="`Toggle section: ${section.label}`"
          >
            <span>{{ section.label }}</span>
            <ChevronDown
              :class="[
                'h-4 w-4 transition-transform',
                isSectionCollapsed(section.id) ? '-rotate-90' : '',
              ]"
            />
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div
              :class="[
                'grid gap-2 pt-2',
                section.columns === 2 ? 'grid-cols-2' : 'grid-cols-1',
              ]"
            >
              <div
                v-for="field in section.fields.sort((a, b) => a.order - b.order)"
                :key="field.key"
                class="bg-muted/50 rounded-md px-3 py-2"
              >
                <p class="text-[10px] uppercase tracking-wide text-muted-foreground font-medium">
                  {{ field.label }}
                </p>
                <span
                  v-if="field.display_type === 'badge'"
                  :class="[
                    'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold mt-1',
                    getColorClass(field.color),
                  ]"
                >
                  {{ getFieldValue(field.key) }}
                </span>
                <span
                  v-else-if="field.display_type === 'tag'"
                  :class="[
                    'inline-flex items-center rounded-md px-2 py-1 text-xs font-medium mt-1',
                    getColorClass(field.color),
                  ]"
                >
                  {{ getFieldValue(field.key) }}
                </span>
                <p v-else class="text-sm font-semibold break-words mt-0.5">
                  {{ getFieldValue(field.key) }}
                </p>
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>

        <div v-else>
          <h5 class="py-2 text-sm font-medium">{{ section.label }}</h5>
          <div
            :class="[
              'grid gap-2',
              section.columns === 2 ? 'grid-cols-2' : 'grid-cols-1',
            ]"
          >
            <div
              v-for="field in section.fields.sort((a, b) => a.order - b.order)"
              :key="field.key"
              class="bg-muted/50 rounded-md px-3 py-2"
            >
              <p class="text-[10px] uppercase tracking-wide text-muted-foreground font-medium">
                {{ field.label }}
              </p>
              <span
                v-if="field.display_type === 'badge'"
                :class="[
                  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold mt-1',
                  getColorClass(field.color),
                ]"
              >
                {{ getFieldValue(field.key) }}
              </span>
              <span
                v-else-if="field.display_type === 'tag'"
                :class="[
                  'inline-flex items-center rounded-md px-2 py-1 text-xs font-medium mt-1',
                  getColorClass(field.color),
                ]"
              >
                {{ getFieldValue(field.key) }}
              </span>
              <p v-else class="text-sm font-semibold break-words mt-0.5">
                {{ getFieldValue(field.key) }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
