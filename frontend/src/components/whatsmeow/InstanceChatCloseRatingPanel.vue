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
import { Label } from "@/components/ui/label";
import {
  cloneInstanceChatCloseRatingSettings,
  normalizeInstanceChatCloseRatingFollowupWindowMinutes,
  sanitizeInstanceChatCloseRatingSettings,
  type InstanceChatCloseRatingSettings,
} from "@/lib/instance-chat-close-rating";
import { Loader2, Settings2 } from "lucide-vue-next";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

const props = defineProps<{
  settings: InstanceChatCloseRatingSettings;
  saving?: boolean;
}>();

const emit = defineEmits<{
  (e: "save", value: InstanceChatCloseRatingSettings): void;
}>();

const dialogOpen = ref(false);
const localSettings = ref<InstanceChatCloseRatingSettings>(
  cloneInstanceChatCloseRatingSettings(props.settings),
);
const followupWindowInput = ref(
  String(localSettings.value.followup_window_minutes),
);

function syncLocalSettings(value: InstanceChatCloseRatingSettings) {
  localSettings.value = cloneInstanceChatCloseRatingSettings(value);
  followupWindowInput.value = String(
    localSettings.value.followup_window_minutes,
  );
}

watch(
  () => props.settings,
  (value) => {
    syncLocalSettings(value);
  },
  { immediate: true, deep: true },
);

watch(dialogOpen, (isOpen) => {
  if (!isOpen) {
    return;
  }

  syncLocalSettings(props.settings);
});

function handleSave() {
  emit(
    "save",
    sanitizeInstanceChatCloseRatingSettings({
      ...localSettings.value,
      followup_window_minutes:
        normalizeInstanceChatCloseRatingFollowupWindowMinutes(
          followupWindowInput.value,
        ),
    }),
  );
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
    {{ $t("instances.chat_close_rating.configureButton") }}
  </Button>

  <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
    <DialogContent class="sm:max-w-[640px]">
      <DialogHeader>
        <DialogTitle>{{ $t("instances.chat_close_rating.title") }}</DialogTitle>
        <DialogDescription class="text-muted-foreground">
          {{ $t("instances.chat_close_rating.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2 max-h-[70vh] overflow-y-auto pr-1">
        <div class="space-y-2">
          <Label>{{
            $t("settings.chatCloseRatingFollowupWindowMinutes")
          }}</Label>
          <p class="text-xs text-muted-foreground">
            {{ $t("settings.chatCloseRatingFollowupWindowMinutesDesc") }}
          </p>
          <Input
            v-model="followupWindowInput"
            type="number"
            min="1"
            max="1440"
            step="1"
            class="bg-background"
          />
        </div>

        <div
          class="space-y-3 rounded-xl border border-border/70 bg-muted/20 p-4"
        >
          <p class="text-xs text-muted-foreground">
            {{ $t("settings.chatCloseRatingTemplatesDesc") }}
          </p>

          <div v-for="lang in ['ar', 'en', 'es']" :key="lang" class="space-y-2">
            <Label>{{
              $t(
                `settings.chatCloseRatingTemplate${lang.charAt(0).toUpperCase() + lang.slice(1)}`,
              )
            }}</Label>
            <Textarea
              v-model="localSettings.templates[lang]"
              :placeholder="$t('settings.chatCloseRatingTemplatePlaceholder')"
              :rows="4"
              class="bg-background font-mono text-sm resize-y"
              dir="auto"
            />
          </div>

          <p class="font-mono text-[11px] text-muted-foreground">
            {{ $t("settings.chatCloseRatingPlaceholders") }}
          </p>
        </div>
      </div>

      <DialogFooter class="gap-2">
        <Button variant="outline" @click="dialogOpen = false">{{
          $t("common.cancel")
        }}</Button>
        <Button :disabled="saving" @click="handleSave">
          <Loader2 v-if="saving" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t("common.save") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
