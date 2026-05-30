<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { Button } from '@/components/ui/button'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  rows?: number
  disabled?: boolean
  id?: string
}>(), {
  placeholder: '',
  rows: 6,
  disabled: false,
  id: undefined
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const placeholderTokens = [
  '{customer_name}',
  '{chat_id}',
  '{agent_name}',
  '{organization_name}',
  '{contact_name}',
  '{phone_number}'
]

function updateValue(value: string) {
  emit('update:modelValue', value)
}

function applyWrapFormat(wrapper: string, fallbackLabel: string) {
  const textarea = textareaRef.value
  const current = props.modelValue || ''

  const selectionStart = textarea?.selectionStart ?? current.length
  const selectionEnd = textarea?.selectionEnd ?? current.length
  const selectedText = current.slice(selectionStart, selectionEnd)
  const innerText = selectedText || fallbackLabel
  const wrappedText = `${wrapper}${innerText}${wrapper}`

  const nextValue = `${current.slice(0, selectionStart)}${wrappedText}${current.slice(selectionEnd)}`
  emit('update:modelValue', nextValue)

  void nextTick(() => {
    const input = textareaRef.value
    if (!input) return
    input.focus()
    const start = selectionStart + wrapper.length
    const end = start + innerText.length
    input.setSelectionRange(start, end)
  })
}

function insertToken(token: string) {
  const textarea = textareaRef.value
  const current = props.modelValue || ''
  const selectionStart = textarea?.selectionStart ?? current.length
  const selectionEnd = textarea?.selectionEnd ?? current.length

  const nextValue = `${current.slice(0, selectionStart)}${token}${current.slice(selectionEnd)}`
  emit('update:modelValue', nextValue)

  void nextTick(() => {
    const input = textareaRef.value
    if (!input) return
    const cursor = selectionStart + token.length
    input.focus()
    input.setSelectionRange(cursor, cursor)
  })
}
</script>

<template>
  <div class="space-y-2">
    <div class="flex flex-wrap items-center gap-1.5">
      <Button type="button" variant="outline" size="sm" class="h-7 px-2 font-semibold" :disabled="disabled" @click="applyWrapFormat('*', 'bold')">
        Bold
      </Button>
      <Button type="button" variant="outline" size="sm" class="h-7 px-2 italic" :disabled="disabled" @click="applyWrapFormat('_', 'italic')">
        Italic
      </Button>
      <Button type="button" variant="outline" size="sm" class="h-7 px-2 line-through" :disabled="disabled" @click="applyWrapFormat('~', 'strike')">
        Strike
      </Button>
      <Button type="button" variant="outline" size="sm" class="h-7 px-2 font-mono" :disabled="disabled" @click="applyWrapFormat('```', 'mono')">
        Mono
      </Button>
    </div>

    <div class="flex flex-wrap items-center gap-1.5">
      <span class="text-xs text-muted-foreground">Placeholders:</span>
      <Button
        v-for="token in placeholderTokens"
        :key="token"
        type="button"
        variant="ghost"
        size="sm"
        class="h-7 px-2 text-xs"
        :disabled="disabled"
        @click="insertToken(token)"
      >
        {{ token }}
      </Button>
    </div>

    <textarea
      ref="textareaRef"
      :id="id"
      :value="modelValue"
      :placeholder="placeholder"
      :rows="rows"
      :disabled="disabled"
      class="flex min-h-[120px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
      @input="updateValue(($event.target as HTMLTextAreaElement).value)"
    />

    <p class="text-xs text-muted-foreground">
      WhatsApp syntax: <span class="font-mono">*bold*</span>, <span class="font-mono">_italic_</span>, <span class="font-mono">~strike~</span>, <span class="font-mono">```mono```</span>
    </p>
  </div>
</template>
