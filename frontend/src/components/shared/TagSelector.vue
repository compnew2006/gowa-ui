<script setup lang="ts">
// Shared tag picker used by the contact form views (create dialog + detail
// view). Renders the combobox trigger, the searchable tag list, and the
// selected-tag badges. The owning view keeps its own outer Label/wrapper and
// supplies the available tags; selection is a v-model string[] of tag names.
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { TagBadge } from '@/components/ui/tag-badge'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Check, ChevronsUpDown, X } from 'lucide-vue-next'
import { getTagColorClass } from '@/lib/constants'
import type { Tag } from '@/services/api'

const props = withDefaults(defineProps<{
  modelValue: string[]
  tags: Tag[]
  disabled?: boolean
}>(), {
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const selectorOpen = ref(false)

function toggleTag(tagName: string) {
  const next = [...props.modelValue]
  const index = next.indexOf(tagName)
  if (index === -1) {
    next.push(tagName)
  } else {
    next.splice(index, 1)
  }
  emit('update:modelValue', next)
}

function removeTag(tagName: string) {
  emit('update:modelValue', props.modelValue.filter(t => t !== tagName))
}

function isTagSelected(tagName: string): boolean {
  return props.modelValue.includes(tagName)
}

function tagColor(tagName: string): string | undefined {
  return props.tags.find(t => t.name === tagName)?.color
}
</script>

<template>
  <Popover v-model:open="selectorOpen">
    <PopoverTrigger as-child>
      <Button variant="outline" role="combobox" class="w-full justify-between" :disabled="disabled">
        <span v-if="modelValue.length === 0" class="text-muted-foreground">{{ $t('contacts.selectTags') }}</span>
        <span v-else>{{ modelValue.length }} {{ $t('contacts.tagsSelected') }}</span>
        <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-[300px] p-0">
      <Command>
        <CommandInput :placeholder="$t('contacts.searchTags')" />
        <CommandList>
          <CommandEmpty>{{ $t('contacts.noTagsFound') }}</CommandEmpty>
          <CommandGroup>
            <CommandItem
              v-for="tag in tags"
              :key="tag.name"
              :value="tag.name"
              class="flex items-center gap-2 cursor-pointer"
              @select.prevent="toggleTag(tag.name)"
            >
              <div class="flex items-center gap-2 flex-1">
                <span :class="['w-2 h-2 rounded-full', getTagColorClass(tag.color).split(' ')[0]]"></span>
                <span>{{ tag.name }}</span>
              </div>
              <Check v-if="isTagSelected(tag.name)" class="h-4 w-4 text-primary" />
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    </PopoverContent>
  </Popover>
  <div v-if="modelValue.length > 0" class="flex flex-wrap gap-1 mt-2">
    <TagBadge
      v-for="tagName in modelValue"
      :key="tagName"
      :color="tagColor(tagName)"
    >
      {{ tagName }}
      <button
        v-if="!disabled"
        type="button"
        class="ml-1 rounded-full hover:bg-black/10 dark:hover:bg-white/10 p-0.5 transition-colors"
        @click.stop="removeTag(tagName)"
      >
        <X class="h-3 w-3" />
      </button>
    </TagBadge>
  </div>
</template>
