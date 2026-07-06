<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { TagBadge } from "@/components/ui/tag-badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  contactsService,
  accountsService,
  instancesService,
  type Tag,
} from "@/services/api";
import type { WhatsAppInstance } from "@/types/whatsmeow";
import { useConfigStore } from "@/stores/config";
import { useTagsStore } from "@/stores/tags";
import { toast } from "vue-sonner";
import { Loader2, Check, ChevronsUpDown, X } from "lucide-vue-next";
import { getErrorMessage } from "@/lib/api-utils";
import { getTagColorClass } from "@/lib/constants";

const { t } = useI18n();
const configStore = useConfigStore();
const tagsStore = useTagsStore();

interface Props {
  open: boolean;
  mode?: "contact" | "chat";
}

const props = withDefaults(defineProps<Props>(), {
  mode: "contact",
});

const emit = defineEmits<{
  "update:open": [value: boolean];
  created: [contact: any];
}>();

interface ContactFormData {
  phone_number: string;
  profile_name: string;
  whatsapp_account: string;
  instance_id: string;
  tags: string[];
}

const defaultFormData: ContactFormData = {
  phone_number: "",
  profile_name: "",
  whatsapp_account: "",
  instance_id: "",
  tags: [],
};

const formData = ref<ContactFormData>({ ...defaultFormData });
const isSubmitting = ref(false);
const tagSelectorOpen = ref(false);
const availableTags = ref<Tag[]>([]);
const availableAccounts = ref<
  { id: string; name: string; phone_number: string }[]
>([]);
const availableInstances = ref<WhatsAppInstance[]>([]);

const isChatMode = computed(() => props.mode === "chat");
const isWhatsmeowProvider = computed(() => configStore.isWhatsmeow);
const isWhatsmeowChatMode = computed(
  () => isChatMode.value && isWhatsmeowProvider.value,
);
const visibleInstances = computed(() =>
  isWhatsmeowChatMode.value
    ? availableInstances.value.filter(
        (instance) => instance.status === "connected",
      )
    : availableInstances.value,
);
const shouldShowInstanceSelector = computed(
  () => visibleInstances.value.length > 0,
);
const shouldShowAccountSelector = computed(
  () => !isWhatsmeowProvider.value && availableAccounts.value.length > 0,
);
const shouldShowTags = computed(
  () => !isChatMode.value && availableTags.value.length > 0,
);
const dialogTitle = computed(() =>
  isWhatsmeowChatMode.value
    ? t("contacts.startChatTitle")
    : t("contacts.createTitle"),
);
const dialogDescription = computed(() =>
  isWhatsmeowChatMode.value
    ? t("contacts.startChatDesc")
    : t("contacts.createDesc"),
);
const submitLabel = computed(() =>
  isWhatsmeowChatMode.value
    ? t("contacts.startChatAction")
    : t("common.create"),
);

watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return;

    formData.value = { ...defaultFormData };

    await Promise.all([
      configStore.fetchConfig().catch(() => {}),
      fetchTags(),
      fetchAccounts(),
      fetchInstances(),
    ]);

    syncDefaultInstanceSelection();
  },
);

watch(visibleInstances, () => {
  if (!props.open) return;
  syncDefaultInstanceSelection();
});

async function fetchTags() {
  try {
    const response = await tagsStore.fetchTags({ limit: 100 });
    availableTags.value = response.tags;
  } catch {
    availableTags.value = [];
  }
}

async function fetchAccounts() {
  try {
    const response = await accountsService.list();
    const data = response.data as any;
    const responseData = data.data || data;
    availableAccounts.value = responseData.accounts || [];
  } catch {
    availableAccounts.value = [];
  }
}

async function fetchInstances() {
  try {
    const response = await instancesService.list();
    const data = response.data as any;
    const responseData = data.data || data;
    availableInstances.value = Array.isArray(responseData) ? responseData : [];
  } catch {
    availableInstances.value = [];
  }
}

function resolvePreferredInstanceId(instances: WhatsAppInstance[]): string {
  if (instances.length === 0) return "";

  const connectedDefault = instances.find(
    (instance) => instance.status === "connected" && instance.is_default,
  );
  if (connectedDefault) return connectedDefault.id;

  const connected = instances.find(
    (instance) => instance.status === "connected",
  );
  if (connected) return connected.id;

  const defaultInstance = instances.find((instance) => instance.is_default);
  if (defaultInstance) return defaultInstance.id;

  return instances[0]?.id || "";
}

function syncDefaultInstanceSelection() {
  if (
    formData.value.instance_id &&
    visibleInstances.value.some(
      (instance) => instance.id === formData.value.instance_id,
    )
  ) {
    return;
  }

  formData.value.instance_id = resolvePreferredInstanceId(
    visibleInstances.value,
  );
}

function normalizePhoneNumber(value: string): string {
  const compact = value.replace(/[^\d+]/g, "");
  if (compact === "") return "";
  if (compact.startsWith("00")) {
    return `+${compact.slice(2)}`;
  }
  if (!compact.startsWith("+")) {
    return compact;
  }
  return `+${compact.slice(1).replace(/\+/g, "")}`;
}

function isInternationalPhoneNumber(value: string): boolean {
  const normalized = normalizePhoneNumber(value);
  const digits = normalized.startsWith("+") ? normalized.slice(1) : normalized;
  return /^[1-9]\d{6,14}$/.test(digits);
}

async function saveContact() {
  const normalizedPhoneNumber = normalizePhoneNumber(
    formData.value.phone_number,
  );
  if (!normalizedPhoneNumber) {
    toast.error(t("contacts.phoneRequired"));
    return;
  }
  if (
    isWhatsmeowChatMode.value &&
    !isInternationalPhoneNumber(normalizedPhoneNumber)
  ) {
    toast.error(t("contacts.phoneInvalid"));
    return;
  }
  if (isWhatsmeowChatMode.value && !formData.value.instance_id) {
    toast.error(t("contacts.instanceRequired"));
    return;
  }

  isSubmitting.value = true;
  try {
    const response = await contactsService.create({
      phone_number: normalizedPhoneNumber,
      profile_name: formData.value.profile_name.trim() || undefined,
      whatsapp_account: shouldShowAccountSelector.value
        ? formData.value.whatsapp_account || undefined
        : undefined,
      instance_id: formData.value.instance_id || undefined,
      start_chat: isWhatsmeowChatMode.value || undefined,
      tags:
        shouldShowTags.value && formData.value.tags.length > 0
          ? formData.value.tags
          : undefined,
    });
    const contact = response.data?.data || response.data;
    toast.success(
      t("common.createdSuccess", { resource: t("resources.Contact") }),
    );
    emit("update:open", false);
    emit("created", contact);
  } catch (error) {
    toast.error(
      getErrorMessage(
        error,
        t("common.failedCreate", { resource: t("resources.contact") }),
      ),
    );
  } finally {
    isSubmitting.value = false;
  }
}

function toggleTag(tagName: string) {
  const index = formData.value.tags.indexOf(tagName);
  if (index === -1) {
    formData.value.tags.push(tagName);
  } else {
    formData.value.tags.splice(index, 1);
  }
}

function removeTag(tagName: string) {
  formData.value.tags = formData.value.tags.filter((tag) => tag !== tagName);
}

function isTagSelected(tagName: string): boolean {
  return formData.value.tags.includes(tagName);
}

function getTagDetails(tagName: string): Tag | undefined {
  return availableTags.value.find((tag) => tag.name === tagName);
}

function closeDialog() {
  emit("update:open", false);
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ dialogTitle }}</DialogTitle>
        <DialogDescription>{{ dialogDescription }}</DialogDescription>
      </DialogHeader>
      <div class="space-y-4 py-4">
        <div class="space-y-2">
          <Label
            >{{ $t("contacts.phoneNumber") }}
            <span class="text-destructive">*</span></Label
          >
          <Input
            v-model="formData.phone_number"
            data-testid="create-contact-phone"
            :placeholder="$t('contacts.phonePlaceholder')"
          />
          <p class="text-xs text-muted-foreground">
            {{ $t("contacts.phoneHint") }}
          </p>
        </div>
        <div class="space-y-2">
          <Label>{{ $t("contacts.profileName") }}</Label>
          <Input
            v-model="formData.profile_name"
            :placeholder="$t('contacts.namePlaceholder')"
          />
        </div>
        <div v-if="shouldShowInstanceSelector" class="space-y-2">
          <Label>{{ $t("contacts.whatsappInstance") }}</Label>
          <Select v-model="formData.instance_id">
            <SelectTrigger data-testid="create-contact-instance">
              <SelectValue :placeholder="$t('contacts.selectInstance')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="instance in visibleInstances"
                :key="instance.id"
                :value="instance.id"
              >
                {{ instance.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="shouldShowAccountSelector" class="space-y-2">
          <Label>{{ $t("contacts.whatsappAccount") }}</Label>
          <Select v-model="formData.whatsapp_account">
            <SelectTrigger>
              <SelectValue :placeholder="$t('contacts.selectAccount')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="account in availableAccounts"
                :key="account.id"
                :value="account.name"
              >
                {{ account.name }} ({{ account.phone_number }})
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="shouldShowTags" class="space-y-2">
          <Label>{{ $t("contacts.tags") }}</Label>
          <Popover v-model:open="tagSelectorOpen">
            <PopoverTrigger as-child>
              <Button
                variant="outline"
                role="combobox"
                class="w-full justify-between"
              >
                <span
                  v-if="formData.tags.length === 0"
                  class="text-muted-foreground"
                  >{{ $t("contacts.selectTags") }}</span
                >
                <span v-else
                  >{{ formData.tags.length }}
                  {{ $t("contacts.tagsSelected") }}</span
                >
                <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
              </Button>
            </PopoverTrigger>
            <PopoverContent
              class="w-[300px] p-0"
              @interact-outside="(event) => event.preventDefault()"
            >
              <Command>
                <CommandInput :placeholder="$t('contacts.searchTags')" />
                <CommandList>
                  <CommandEmpty>{{ $t("contacts.noTagsFound") }}</CommandEmpty>
                  <CommandGroup>
                    <CommandItem
                      v-for="tag in availableTags"
                      :key="tag.name"
                      :value="tag.name"
                      class="flex items-center gap-2 cursor-pointer"
                      @select.prevent="toggleTag(tag.name)"
                    >
                      <div class="flex items-center gap-2 flex-1">
                        <span
                          :class="[
                            'w-2 h-2 rounded-full',
                            getTagColorClass(tag.color).split(' ')[0],
                          ]"
                        ></span>
                        <span>{{ tag.name }}</span>
                      </div>
                      <Check
                        v-if="isTagSelected(tag.name)"
                        class="h-4 w-4 text-primary"
                      />
                    </CommandItem>
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
          <div
            v-if="formData.tags.length > 0"
            class="mt-2 flex flex-wrap gap-1"
          >
            <TagBadge
              v-for="tagName in formData.tags"
              :key="tagName"
              :color="getTagDetails(tagName)?.color"
            >
              {{ tagName }}
              <button
                type="button"
                class="ml-1 rounded-full p-0.5 transition-colors hover:bg-black/10 dark:hover:bg-white/10"
                @click.stop="removeTag(tagName)"
              >
                <X class="h-3 w-3" />
              </button>
            </TagBadge>
          </div>
        </div>
      </div>
      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="closeDialog">{{
          $t("common.cancel")
        }}</Button>
        <Button
          data-testid="create-contact-submit"
          :disabled="isSubmitting"
          @click="saveContact"
        >
          <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
          {{ submitLabel }}
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
