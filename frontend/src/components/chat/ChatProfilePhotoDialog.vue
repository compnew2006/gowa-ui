<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { getInitials } from "@/lib/utils";
import { normalizeRenderableAvatarURL } from "@/components/ui/avatar/avatar-url";
import type { Contact } from "@/stores/contacts";

const props = defineProps<{
  open: boolean;
  contact: Contact | null;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
}>();

const { t } = useI18n();
const imageFailed = ref(false);

const avatarUrl = () => {
  if (!props.contact) return "";
  return normalizeRenderableAvatarURL(props.contact.avatar_url);
};

function handleImageError() {
  imageFailed.value = true;
}

function handleOpenChange(open: boolean) {
  if (open) imageFailed.value = false;
  emit("update:open", open);
}
</script>

<template>
  <Dialog :open="open" @update:open="handleOpenChange">
    <DialogContent class="max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("resources.ProfilePhoto") }}</DialogTitle>
        <DialogDescription>
          {{
            contact?.name ||
            contact?.phone_number ||
            t("chat.customer")
          }}
        </DialogDescription>
      </DialogHeader>
      <div class="flex items-center justify-center py-2">
        <img
          v-if="avatarUrl() !== '' && !imageFailed"
          :src="avatarUrl()"
          :alt="
            contact?.name ||
            contact?.phone_number ||
            t('resources.ProfilePhoto')
          "
          class="max-h-[70vh] max-w-full rounded-lg object-contain"
          @error="handleImageError"
        />
        <div
          v-else
          class="flex h-48 w-48 items-center justify-center rounded-full bg-gradient-to-br from-sky-500 to-blue-600 text-4xl font-semibold text-white"
        >
          {{
            getInitials(
              contact?.name ||
                contact?.phone_number ||
                t("chat.customer"),
            )
          }}
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
