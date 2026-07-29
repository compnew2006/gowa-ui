<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import TagSelector from '@/components/shared/TagSelector.vue'
import { contactsService, accountsService, type Tag } from '@/services/api'
import { useTagsStore } from '@/stores/tags'
import { toast } from 'vue-sonner'
import { Loader2 } from 'lucide-vue-next'
import { getErrorMessage } from '@/lib/api-utils'

const { t } = useI18n()
const tagsStore = useTagsStore()

interface Props {
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  'created': [contact: any]
}>()

interface ContactFormData {
  phone_number: string
  profile_name: string
  whatsapp_account: string
  tags: string[]
}

const defaultFormData: ContactFormData = { phone_number: '', profile_name: '', whatsapp_account: '', tags: [] }

const formData = ref<ContactFormData>({ ...defaultFormData })
const isSubmitting = ref(false)
const availableTags = ref<Tag[]>([])
const availableAccounts = ref<{ id: string; name: string; phone_number: string }[]>([])

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    formData.value = { ...defaultFormData }
    fetchTags()
    fetchAccounts()
  }
})

async function fetchTags() {
  try {
    const response = await tagsStore.fetchTags({ limit: 100 })
    availableTags.value = response.tags
  } catch {
    // Silently fail - tags are optional
  }
}

async function fetchAccounts() {
  try {
    const response = await accountsService.list()
    const data = response.data as any
    const responseData = data.data || data
    availableAccounts.value = responseData.accounts || []
  } catch (error) {
    // Silently fail - accounts are optional
  }
}

async function saveContact() {
  if (!formData.value.phone_number.trim()) {
    toast.error(t('contacts.phoneRequired'))
    return
  }
  isSubmitting.value = true
  try {
    const response = await contactsService.create({
      phone_number: formData.value.phone_number.trim(),
      profile_name: formData.value.profile_name.trim() || undefined,
      whatsapp_account: formData.value.whatsapp_account || undefined,
      tags: formData.value.tags.length > 0 ? formData.value.tags : undefined
    })
    const contact = response.data?.data || response.data
    toast.success(t('common.createdSuccess', { resource: t('resources.Contact') }))
    emit('update:open', false)
    emit('created', contact)
  } catch (error) {
    toast.error(getErrorMessage(error, t('common.failedCreate', { resource: t('resources.contact') })))
  } finally {
    isSubmitting.value = false
  }
}

function closeDialog() {
  emit('update:open', false)
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ $t('contacts.createTitle') }}</DialogTitle>
        <DialogDescription>{{ $t('contacts.createDesc') }}</DialogDescription>
      </DialogHeader>
      <div class="space-y-4 py-4">
        <div class="space-y-2">
          <Label>{{ $t('contacts.phoneNumber') }} <span class="text-destructive">*</span></Label>
          <Input v-model="formData.phone_number" :placeholder="$t('contacts.phonePlaceholder')" />
          <p class="text-xs text-muted-foreground">{{ $t('contacts.phoneHint') }}</p>
        </div>
        <div class="space-y-2">
          <Label>{{ $t('contacts.profileName') }}</Label>
          <Input v-model="formData.profile_name" :placeholder="$t('contacts.namePlaceholder')" />
        </div>
        <div v-if="availableAccounts.length > 0" class="space-y-2">
          <Label>{{ $t('contacts.whatsappAccount') }}</Label>
          <Select v-model="formData.whatsapp_account">
            <SelectTrigger>
              <SelectValue :placeholder="$t('contacts.selectAccount')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="account in availableAccounts" :key="account.id" :value="account.name">
                {{ account.name }} ({{ account.phone_number }})
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="availableTags.length > 0" class="space-y-2">
          <Label>{{ $t('contacts.tags') }}</Label>
          <TagSelector v-model="formData.tags" :tags="availableTags" />
        </div>
      </div>
      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="closeDialog">{{ $t('common.cancel') }}</Button>
        <Button @click="saveContact" :disabled="isSubmitting">
          <Loader2 v-if="isSubmitting" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t('common.create') }}
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
