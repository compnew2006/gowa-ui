<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2, MessageSquare, Settings, User } from 'lucide-vue-next'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import {
  Command,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import type { UseGlobalSearch } from '@/composables/useGlobalSearch'

const props = defineProps<{ search: UseGlobalSearch }>()

const { t } = useI18n()

// Top-level setup bindings are auto-unwrapped in the template; the search
// object itself is a prop, so its nested refs would NOT unwrap there.
const {
  open,
  isLoading,
  canSearch,
  isEmpty,
  contactItems,
  conversationItems,
  settingsItems,
  contactGroupLabel,
  conversationGroupLabel,
  settingsGroupLabel,
  emptyTitle,
  emptyDescription,
  select,
} = props.search

// Ctrl+K / Cmd+K opens the palette from anywhere. The listener lives here
// (mounted once, in AppLayout) so the shortcut and the dialog share one
// search instance. Esc is handled by the Dialog itself.
function onKeyDown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    props.search.openPalette()
  }
}

onMounted(() => window.addEventListener('keydown', onKeyDown))
onUnmounted(() => window.removeEventListener('keydown', onKeyDown))

// Return focus to whatever opened the palette when it closes.
const previouslyFocused = ref<Element | null>(null)
watch(
  () => open.value,
  (isOpen) => {
    if (isOpen) {
      previouslyFocused.value = document.activeElement
    } else {
      props.search.closePalette()
      const el = previouslyFocused.value
      if (el instanceof HTMLElement) el.focus()
      previouslyFocused.value = null
    }
  },
)

// CommandInput keeps its own filter text internally; mirror it into the
// search query so the server search runs on the same keystrokes.
function onInput(event: Event) {
  props.search.query.value = (event.target as HTMLInputElement).value
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent
      class="w-[calc(100vw-2rem)] max-w-lg overflow-hidden p-0 shadow-lg"
    >
      <DialogTitle class="sr-only">{{ t('globalSearch.title') }}</DialogTitle>
      <Command
        class="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground [&_[cmdk-group]:not([hidden])_~[cmdk-group]]:pt-0 [&_[cmdk-group]]:px-2 [&_[cmdk-input-wrapper]_svg]:h-5 [&_[cmdk-input-wrapper]_svg]:w-5 [&_[cmdk-input]]:h-12 [&_[cmdk-item]]:px-2 [&_[cmdk-item]]:py-3 [&_[cmdk-item]_svg]:h-5 [&_[cmdk-item]_svg]:w-5"
      >
        <CommandInput
          :placeholder="t('globalSearch.placeholder')"
          @input="onInput"
        />
        <CommandList>
          <CommandGroup
            v-if="contactItems.length > 0"
            :heading="contactGroupLabel"
          >
            <CommandItem
              v-for="item in contactItems"
              :key="item.id"
              :value="item.id"
              @select="() => select(item)"
            >
              <User class="h-4 w-4 shrink-0 opacity-60" />
              <span class="truncate">{{ item.title }}</span>
              <span
                v-if="item.subtitle"
                class="ml-auto shrink-0 text-xs text-muted-foreground"
              >
                {{ item.subtitle }}
              </span>
            </CommandItem>
          </CommandGroup>

          <CommandGroup
            v-if="conversationItems.length > 0"
            :heading="conversationGroupLabel"
          >
            <CommandItem
              v-for="item in conversationItems"
              :key="item.id"
              :value="item.id"
              @select="() => select(item)"
            >
              <MessageSquare class="h-4 w-4 shrink-0 opacity-60" />
              <span class="truncate">{{ item.title }}</span>
              <span
                v-if="item.subtitle"
                class="ml-auto max-w-[55%] shrink-0 truncate text-xs text-muted-foreground"
              >
                {{ item.subtitle }}
              </span>
            </CommandItem>
          </CommandGroup>

          <CommandGroup
            v-if="settingsItems.length > 0"
            :heading="settingsGroupLabel"
          >
            <CommandItem
              v-for="item in settingsItems"
              :key="item.id"
              :value="item.id"
              @select="() => select(item)"
            >
              <Settings class="h-4 w-4 shrink-0 opacity-60" />
              <span class="truncate">{{ item.title }}</span>
              <span
                v-if="item.subtitle"
                class="ml-auto shrink-0 text-xs text-muted-foreground"
              >
                {{ item.subtitle }}
              </span>
            </CommandItem>
          </CommandGroup>

          <div
            v-if="isLoading && canSearch"
            class="flex items-center gap-2 px-4 py-3 text-sm text-muted-foreground"
            role="status"
          >
            <Loader2 class="h-4 w-4 animate-spin opacity-60" />
            {{ t('globalSearch.searching') }}
          </div>

          <div
            v-if="isEmpty"
            role="status"
            class="px-4 py-6 text-center"
          >
            <p class="text-sm font-medium">{{ emptyTitle }}</p>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ emptyDescription }}
            </p>
          </div>
          <div
            v-else-if="!canSearch && !isLoading"
            class="px-4 py-6 text-center text-sm text-muted-foreground"
          >
            {{ t('globalSearch.typeToSearch') }}
          </div>
        </CommandList>
        <div
          class="border-t px-3 py-2 text-xs text-muted-foreground"
        >
          {{ t('globalSearch.scopeHint') }}
        </div>
      </Command>
    </DialogContent>
  </Dialog>
</template>
