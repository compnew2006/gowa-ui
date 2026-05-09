<script setup lang="ts">
import { ref } from "vue";
import { Button } from "@/components/ui/button";
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
import { Plus, X, Check, Loader2, Tags } from "lucide-vue-next";
import { TagBadge } from "@/components/ui/tag-badge";
import { getTagColorClass } from "@/lib/constants";
import { useTagsStore } from "@/stores/tags";
import { contactsService, type Tag } from "@/services/api";
import { toast } from "vue-sonner";

const props = defineProps<{
  contactId: string;
  contactTags: string[];
  canEditTags: boolean;
}>();

const emit = defineEmits<{
  tagsUpdated: [tags: string[]];
}>();

const tagsStore = useTagsStore();
const tagSelectorOpen = ref(false);
const isUpdatingTags = ref(false);

function getTagDetails(tagName: string): Tag | undefined {
  return tagsStore.getTagByName(tagName);
}

function isTagSelected(tagName: string): boolean {
  return props.contactTags.includes(tagName);
}

async function toggleTag(tagName: string) {
  if (isUpdatingTags.value) return;
  const currentTags = [...props.contactTags];
  let newTags: string[];
  if (currentTags.includes(tagName)) {
    newTags = currentTags.filter((t) => t !== tagName);
  } else {
    newTags = [...currentTags, tagName];
  }
  await updateContactTags(newTags);
}

async function removeTag(tagName: string) {
  if (isUpdatingTags.value) return;
  const newTags = props.contactTags.filter((t) => t !== tagName);
  await updateContactTags(newTags);
}

async function updateContactTags(tags: string[]) {
  isUpdatingTags.value = true;
  try {
    await contactsService.updateTags(props.contactId, tags);
    emit("tagsUpdated", tags);
    toast.success("Tags updated");
  } catch (e: any) {
    toast.error(e.response?.data?.message || "Failed to update tags");
  } finally {
    isUpdatingTags.value = false;
  }
}
</script>

<template>
  <div class="pb-4">
    <div class="flex items-center justify-between py-2">
      <h5 class="text-sm font-medium flex items-center gap-2">
        <Tags class="h-4 w-4 text-muted-foreground" />
        Tags
      </h5>
      <Popover v-if="canEditTags" v-model:open="tagSelectorOpen">
        <PopoverTrigger as-child>
          <Button variant="ghost" size="sm" class="h-7 px-2">
            <Plus class="h-3.5 w-3.5" />
          </Button>
        </PopoverTrigger>
        <PopoverContent class="w-[200px] p-0" align="end">
          <Command>
            <CommandInput placeholder="Search tags..." />
            <CommandList>
              <CommandEmpty>
                <div class="py-4 text-center text-sm text-muted-foreground">
                  No tags found
                </div>
              </CommandEmpty>
              <CommandGroup>
                <CommandItem
                  v-for="tag in tagsStore.tags"
                  :key="tag.name"
                  :value="tag.name"
                  class="flex items-center gap-2"
                  @select="toggleTag(tag.name)"
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
                  <Check v-if="isTagSelected(tag.name)" class="h-4 w-4 text-primary" />
                </CommandItem>
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>

    <div class="flex flex-wrap gap-2 mt-2">
      <template v-if="contactTags.length > 0">
        <TagBadge
          v-for="tagName in contactTags"
          :key="tagName"
          :color="getTagDetails(tagName)?.color"
        >
          {{ tagName }}
          <button
            v-if="canEditTags"
            type="button"
            class="ml-1 rounded-full hover:bg-black/10 dark:hover:bg-white/10 p-0.5 transition-colors"
            :disabled="isUpdatingTags"
            @click.stop="removeTag(tagName)"
          >
            <X class="h-3 w-3" />
          </button>
        </TagBadge>
      </template>
      <span v-else class="text-sm text-muted-foreground">No tags</span>
      <Loader2
        v-if="isUpdatingTags"
        class="h-4 w-4 animate-spin text-muted-foreground"
      />
    </div>
  </div>
</template>
