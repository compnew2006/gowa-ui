<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import {
  Popover,
  PopoverContent,
  PopoverTrigger
} from '@/components/ui/popover'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command'
import { X, ChevronDown, Phone, User, Plus, Check, Tags, Loader2, Trash2, UserPlus, Search } from 'lucide-vue-next'
import { TagBadge } from '@/components/ui/tag-badge'
import MetadataSection from '@/components/chat/MetadataSection.vue'
import { getInitials, getAvatarGradient, formatLabel } from '@/lib/utils'
import { getTagColorClass } from '@/lib/constants'
import { useTagsStore } from '@/stores/tags'
import { useAuthStore } from '@/stores/auth'
import { useUsersStore } from '@/stores/users'
import { localeDirectionManager } from '@/i18n/locale-direction'
import { contactsService, type Tag } from '@/services/api'
import { wsService } from '@/services/websocket'
import { toast } from 'vue-sonner'
import type { Contact } from '@/stores/contacts'

interface PanelFieldConfig {
  key: string
  label: string
  order: number
  display_type?: 'text' | 'badge' | 'tag'
  color?: 'default' | 'success' | 'warning' | 'error' | 'info'
}

interface PanelSection {
  id: string
  label: string
  columns: number
  collapsible: boolean
  default_collapsed: boolean
  order: number
  fields: PanelFieldConfig[]
}

interface PanelConfig {
  sections: PanelSection[]
}

interface SessionData {
  session_id?: string
  flow_id?: string
  flow_name?: string
  session_data: Record<string, any>
  panel_config: PanelConfig
}

const props = defineProps<{
  contact: Contact
  sessionData?: SessionData | null
}>()

const emit = defineEmits<{
  close: []
  tagsUpdated: [tags: string[]]
  deleted: [contactId: string]
}>()

const tagsStore = useTagsStore()
const authStore = useAuthStore()
const usersStore = useUsersStore()
const { locale, t } = useI18n()
const isRTL = computed(() => localeDirectionManager.isRTL(String(locale.value)))

const collapsedSections = ref<Record<string, boolean>>({})
const tagSelectorOpen = ref(false)
const isUpdatingTags = ref(false)

interface ContactCollaborator {
  id: string
  contact_id: string
  user_id: string
  user_name?: string
  role: string
  status: string
  invited_by_user_id: string
  invited_by_name?: string
  invited_at: string
  accepted_at?: string | null
}

const collaborators = ref<ContactCollaborator[]>([])
const isLoadingCollaborators = ref(false)
const isInviteDialogOpen = ref(false)
const inviteSearchQuery = ref('')
const isInvitingCollaborator = ref(false)
const collaboratorActionId = ref<string | null>(null)

// Resizable panel state
const MIN_WIDTH = 280
const MAX_WIDTH = 500
const panelWidth = ref(MAX_WIDTH) // Default to max width
const isResizing = ref(false)

// Check if user can edit tags
const canEditTags = computed(() => authStore.hasPermission('contacts', 'write'))
const canDeleteChats = computed(() => authStore.hasPermission('contacts', 'delete'))
const canInviteCollaborators = computed(() => authStore.hasPermission('chat.collaborators', 'write'))
const currentUserId = computed(() => authStore.user?.id || '')
const isDeletingChat = ref(false)

const handleCollaboratorUpdate = (payload: any) => {
  if (!payload?.contact_id || payload.contact_id !== props.contact.id) return
  void fetchCollaborators()
}
const handleCollaboratorInvite = (payload: any) => {
  if (!payload?.contact_id || payload.contact_id !== props.contact.id) return
  void fetchCollaborators()
}

// Fetch tags on mount
onMounted(async () => {
  if (tagsStore.tags.length === 0) {
    try {
      await tagsStore.fetchTags()
    } catch (e) {
      // Silently fail - tags just won't be available
    }
  }

  await fetchCollaborators()

  wsService.subscribe('chat_collaborator_update', handleCollaboratorUpdate)
  wsService.subscribe('chat_collaborator_invite', handleCollaboratorInvite)
})

onUnmounted(() => {
  wsService.unsubscribe('chat_collaborator_update', handleCollaboratorUpdate)
  wsService.unsubscribe('chat_collaborator_invite', handleCollaboratorInvite)
})

function startResize(e: MouseEvent) {
  isResizing.value = true
  const startX = e.clientX
  const startWidth = panelWidth.value
  const isRTLDirection = isRTL.value

  function onMouseMove(e: MouseEvent) {
    // In RTL the panel is mirrored, so drag direction must be inverted.
    const delta = isRTLDirection ? e.clientX - startX : startX - e.clientX
    const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth + delta))
    panelWidth.value = newWidth
  }

  function onMouseUp() {
    isResizing.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// Initialize collapsed state when session data changes
watch(() => props.sessionData, (newData) => {
  if (newData?.panel_config?.sections) {
    for (const section of newData.panel_config.sections) {
      if (collapsedSections.value[section.id] === undefined) {
        collapsedSections.value[section.id] = section.default_collapsed
      }
    }
  }
}, { immediate: true })

watch(() => props.contact.id, () => {
  void fetchCollaborators()
})

watch(isInviteDialogOpen, (open) => {
  if (!open) return
  if (usersStore.users.length > 0) return
  usersStore.fetchUsers().catch(() => {})
})

function toggleSection(sectionId: string) {
  collapsedSections.value[sectionId] = !collapsedSections.value[sectionId]
}

function isSectionCollapsed(sectionId: string): boolean {
  return collapsedSections.value[sectionId] ?? false
}

function getFieldValue(key: string): string {
  if (!props.sessionData?.session_data) return '-'
  const value = props.sessionData.session_data[key]
  if (value === undefined || value === null || value === '') return '-'
  return String(value)
}

function getColorClass(color?: string): string {
  switch (color) {
    case 'success':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'warning':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'error':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'info':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

// Sort sections by order
const sortedSections = computed(() => {
  if (!props.sessionData?.panel_config?.sections) return []
  return [...props.sessionData.panel_config.sections].sort((a, b) => a.order - b.order)
})

// Get tags from contact
const contactTags = computed(() => {
  if (!props.contact.tags || !Array.isArray(props.contact.tags)) return []
  return props.contact.tags as string[]
})

// Contact metadata
const hasMetadata = computed(() => {
  const md = props.contact.metadata
  return md && typeof md === 'object' && Object.keys(md).length > 0
})

const metadataPrimitives = computed(() => {
  if (!hasMetadata.value) return []
  return Object.entries(props.contact.metadata).filter(
    ([, v]) => v === null || typeof v !== 'object'
  )
})

const metadataSections = computed(() => {
  if (!hasMetadata.value) return []
  return Object.entries(props.contact.metadata).filter(
    ([, v]) => v !== null && typeof v === 'object'
  )
})

// Get tag details for display
function getTagDetails(tagName: string): Tag | undefined {
  return tagsStore.getTagByName(tagName)
}

function normalizeAllowedInstanceIDs(values: unknown): string[] {
  if (!Array.isArray(values)) return []
  return Array.from(
    new Set(
      values
        .map(value => (typeof value === 'string' ? value.trim() : ''))
        .filter(Boolean)
    )
  )
}

function getAllowedInstanceIDsForUser(user: { settings?: unknown }): string[] {
  const settings = user?.settings
  if (!settings || typeof settings !== 'object') return []
  const sendRestrictions = (settings as Record<string, unknown>).send_restrictions
  if (!sendRestrictions || typeof sendRestrictions !== 'object') return []

  const raw = sendRestrictions as Record<string, unknown>
  const fromArray = normalizeAllowedInstanceIDs(raw.allowed_instance_ids)
  if (fromArray.length > 0) {
    return fromArray
  }
  const legacy = typeof raw.allowed_instance_id === 'string'
    ? raw.allowed_instance_id.trim()
    : ''
  return legacy ? [legacy] : []
}

function canUserSeeContactInstance(user: { settings?: unknown }, instanceId?: string): boolean {
  if (!instanceId) return true
  const allowedInstanceIDs = getAllowedInstanceIDsForUser(user)
  if (allowedInstanceIDs.length === 0) return true
  return allowedInstanceIDs.includes(instanceId)
}

// Check if a tag is selected
function isTagSelected(tagName: string): boolean {
  return contactTags.value.includes(tagName)
}

// Toggle tag on contact
async function toggleTag(tagName: string) {
  if (isUpdatingTags.value) return

  const currentTags = [...contactTags.value]
  let newTags: string[]

  if (currentTags.includes(tagName)) {
    newTags = currentTags.filter(t => t !== tagName)
  } else {
    newTags = [...currentTags, tagName]
  }

  await updateContactTags(newTags)
}

// Remove tag from contact
async function removeTag(tagName: string) {
  if (isUpdatingTags.value) return

  const newTags = contactTags.value.filter(t => t !== tagName)
  await updateContactTags(newTags)
}

// Update contact tags via API
async function updateContactTags(tags: string[]) {
  isUpdatingTags.value = true
  try {
    await contactsService.updateTags(props.contact.id, tags)
    emit('tagsUpdated', tags)
    toast.success('Tags updated')
  } catch (e: any) {
    toast.error(e.response?.data?.message || 'Failed to update tags')
  } finally {
    isUpdatingTags.value = false
  }
}

const inviteCandidates = computed(() => {
  const instanceId = typeof props.contact.instance_id === 'string'
    ? props.contact.instance_id.trim()
    : ''
  const query = inviteSearchQuery.value.trim().toLowerCase()

  return usersStore.users
    .filter(user => user.is_active !== false)
    .filter(user => user.id !== currentUserId.value)
    .filter(user => canUserSeeContactInstance(user, instanceId || undefined))
    .filter(user => {
      const existing = collaborators.value.find(c => c.user_id === user.id)
      if (!existing) return true
      return existing.status === 'declined'
    })
    .filter(user => {
      if (!query) return true
      const name = (user.full_name || '').toLowerCase()
      const email = (user.email || '').toLowerCase()
      return name.includes(query) || email.includes(query)
    })
})

async function fetchCollaborators() {
  if (!props.contact?.id) return
  isLoadingCollaborators.value = true
  try {
    const response = await contactsService.listCollaborators(props.contact.id)
    const data = response.data.data || response.data
    collaborators.value = Array.isArray(data.collaborators) ? data.collaborators : []
  } catch (e: any) {
    collaborators.value = []
  } finally {
    isLoadingCollaborators.value = false
  }
}

async function inviteCollaborator(userId: string) {
  if (!props.contact?.id || isInvitingCollaborator.value) return
  isInvitingCollaborator.value = true
  try {
    await contactsService.inviteCollaborator(props.contact.id, { user_id: userId })
    toast.success(t('chat.collaboratorInvitedSuccess'))
    inviteSearchQuery.value = ''
    await fetchCollaborators()
  } catch (e: any) {
    toast.error(e.response?.data?.message || t('chat.collaboratorInviteFailed'))
  } finally {
    isInvitingCollaborator.value = false
  }
}

async function acceptInvite(collab: ContactCollaborator) {
  if (!props.contact?.id || collaboratorActionId.value) return
  collaboratorActionId.value = collab.user_id
  try {
    await contactsService.acceptCollaborator(props.contact.id, collab.user_id)
    toast.success(t('chat.collaboratorAccepted'))
    await fetchCollaborators()
  } catch (e: any) {
    toast.error(e.response?.data?.message || t('chat.collaboratorAcceptFailed'))
  } finally {
    collaboratorActionId.value = null
  }
}

async function declineInvite(collab: ContactCollaborator) {
  if (!props.contact?.id || collaboratorActionId.value) return
  collaboratorActionId.value = collab.user_id
  try {
    await contactsService.declineCollaborator(props.contact.id, collab.user_id)
    toast.success(t('chat.collaboratorDeclined'))
    await fetchCollaborators()
  } catch (e: any) {
    toast.error(e.response?.data?.message || t('chat.collaboratorDeclineFailed'))
  } finally {
    collaboratorActionId.value = null
  }
}

async function removeCollaborator(collab: ContactCollaborator) {
  if (!props.contact?.id || collaboratorActionId.value) return
  collaboratorActionId.value = collab.user_id
  try {
    await contactsService.removeCollaborator(props.contact.id, collab.user_id)
    toast.success(t('chat.collaboratorRemoved'))
    await fetchCollaborators()
  } catch (e: any) {
    toast.error(e.response?.data?.message || t('chat.collaboratorRemoveFailed'))
  } finally {
    collaboratorActionId.value = null
  }
}

async function deleteChat() {
  if (isDeletingChat.value || !canDeleteChats.value) return

  const confirmed = window.confirm('Delete this chat conversation?')
  if (!confirmed) return

  isDeletingChat.value = true
  try {
    await contactsService.delete(props.contact.id)
    toast.success('Chat deleted')
    emit('deleted', props.contact.id)
  } catch (e: any) {
    toast.error(e.response?.data?.message || 'Failed to delete chat')
  } finally {
    isDeletingChat.value = false
  }
}

</script>

<template>
  <div
    class="flex flex-col bg-card h-full relative"
    :style="{ width: `${panelWidth}px` }"
  >
    <!-- Resize Handle -->
    <div
      class="absolute top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/20 active:bg-primary/30 z-10"
      :class="[isRTL ? 'right-0 border-r' : 'left-0 border-l', { 'bg-primary/30': isResizing }]"
      @mousedown="startResize"
    />

    <!-- Header -->
    <div class="h-12 px-3 border-b flex items-center justify-between">
      <h3 class="font-medium text-sm">{{ $t('chat.contactInfo') }}</h3>
      <div class="flex items-center gap-1">
        <Button
          v-if="canDeleteChats"
          variant="ghost"
          size="icon"
          class="h-8 w-8 text-red-500 hover:text-red-600"
          :disabled="isDeletingChat"
          title="Delete chat"
          @click="deleteChat"
        >
          <Loader2 v-if="isDeletingChat" class="h-4 w-4 animate-spin" />
          <Trash2 v-else class="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" class="h-8 w-8" @click="emit('close')">
          <X class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <ScrollArea class="flex-1">
      <div class="p-4 space-y-4">
        <!-- Contact Header -->
        <div class="flex flex-col items-center text-center pb-4 border-b">
          <Avatar class="h-16 w-16 mb-3">
            <AvatarImage :src="contact.avatar_url" />
            <AvatarFallback :class="'text-lg bg-gradient-to-br text-white ' + getAvatarGradient(contact.name || contact.phone_number)">
              {{ getInitials(contact.name || contact.phone_number) }}
            </AvatarFallback>
          </Avatar>
          <h4 class="font-medium">
            {{ contact.name || contact.phone_number }}
          </h4>
          <div class="flex items-center gap-1 text-sm text-muted-foreground mt-1">
            <Phone class="h-3 w-3" />
            <span>{{ contact.phone_number }}</span>
          </div>
        </div>

        <!-- Tags Section (always shown) -->
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
            <Loader2 v-if="isUpdatingTags" class="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        </div>

        <!-- Collaborators Section -->
        <div class="pb-4 border-b">
          <div class="flex items-center justify-between py-2">
            <h5 class="text-sm font-medium flex items-center gap-2">
              <UserPlus class="h-4 w-4 text-muted-foreground" />
              {{ $t('chat.collaborators') }}
            </h5>
            <Button
              v-if="canInviteCollaborators"
              variant="ghost"
              size="sm"
              class="h-7 px-2"
              @click="isInviteDialogOpen = true"
            >
              <Plus class="h-3.5 w-3.5" />
            </Button>
          </div>

          <div v-if="isLoadingCollaborators" class="text-sm text-muted-foreground">
            {{ $t('chat.collaboratorLoading') }}
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="collab in collaborators"
              :key="collab.id"
              class="flex items-start justify-between gap-2 rounded-md border border-border/60 p-2"
            >
              <div class="min-w-0">
                <p class="text-sm font-medium truncate">
                  {{ collab.user_name || collab.user_id }}
                </p>
                <p class="text-xs text-muted-foreground">
                  {{ collab.role }} · {{ collab.status }}
                </p>
              </div>
              <div class="flex items-center gap-1">
                <Button
                  v-if="collab.status === 'invited' && collab.user_id === currentUserId"
                  variant="outline"
                  size="sm"
                  :disabled="collaboratorActionId === collab.user_id"
                  @click="acceptInvite(collab)"
                >
                  {{ $t('chat.collaboratorAccept') }}
                </Button>
                <Button
                  v-if="collab.status === 'invited' && collab.user_id === currentUserId"
                  variant="ghost"
                  size="sm"
                  :disabled="collaboratorActionId === collab.user_id"
                  @click="declineInvite(collab)"
                >
                  {{ $t('chat.collaboratorDecline') }}
                </Button>
                <Button
                  v-if="collab.status === 'declined' && canInviteCollaborators"
                  variant="outline"
                  size="sm"
                  :disabled="collaboratorActionId === collab.user_id"
                  @click="inviteCollaborator(collab.user_id)"
                >
                  {{ $t('chat.collaboratorReinvite') }}
                </Button>
                <Button
                  v-if="canInviteCollaborators || collab.user_id === currentUserId"
                  variant="ghost"
                  size="icon"
                  class="h-7 w-7"
                  :disabled="collaboratorActionId === collab.user_id"
                  @click="removeCollaborator(collab)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>

            <p v-if="collaborators.length === 0" class="text-sm text-muted-foreground">
              {{ $t('chat.collaboratorNone') }}
            </p>
          </div>
        </div>

        <!-- Contact Metadata -->
        <div v-if="hasMetadata" class="space-y-3">
          <!-- General section: top-level primitives -->
          <MetadataSection
            v-if="metadataPrimitives.length > 0"
            label="General"
            :data="Object.fromEntries(metadataPrimitives)"
          />
          <!-- Object / array sections -->
          <MetadataSection
            v-for="[key, val] in metadataSections"
            :key="key"
            :label="formatLabel(key)"
            :data="val"
          />
        </div>

        <!-- No Session Data or no panel config -->
        <div v-if="!props.sessionData || sortedSections.length === 0" class="text-center py-6 text-muted-foreground border-t">
          <User class="h-8 w-8 mx-auto mb-2 opacity-50" />
          <p class="text-sm">No data configured</p>
          <p class="text-xs mt-1">Configure panel display in the chatbot flow settings.</p>
        </div>

        <!-- Session Data with panel config -->
        <template v-else>
          <!-- Flow Name Badge -->
          <div v-if="props.sessionData?.flow_name" class="flex items-center gap-2">
            <Badge variant="outline" class="text-xs">
              {{ props.sessionData?.flow_name }}
            </Badge>
          </div>

          <!-- Dynamic Sections -->
          <div v-for="section in sortedSections" :key="section.id" class="space-y-2 border-t pt-4">
            <Collapsible
              v-if="section.collapsible"
              :open="!isSectionCollapsed(section.id)"
              @update:open="toggleSection(section.id)"
            >
              <CollapsibleTrigger class="flex items-center justify-between w-full py-2 text-sm font-medium hover:text-primary transition-colors">
                <span>{{ section.label }}</span>
                <ChevronDown
                  :class="[
                    'h-4 w-4 transition-transform',
                    isSectionCollapsed(section.id) ? '-rotate-90' : ''
                  ]"
                />
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div
                  :class="[
                    'grid gap-2 pt-2',
                    section.columns === 2 ? 'grid-cols-2' : 'grid-cols-1'
                  ]"
                >
                  <div
                    v-for="field in section.fields.sort((a, b) => a.order - b.order)"
                    :key="field.key"
                    class="bg-muted/50 rounded-md px-3 py-2"
                  >
                    <p class="text-[10px] uppercase tracking-wide text-muted-foreground font-medium">{{ field.label }}</p>
                    <!-- Badge display -->
                    <span
                      v-if="field.display_type === 'badge'"
                      :class="['inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold mt-1', getColorClass(field.color)]"
                    >
                      {{ getFieldValue(field.key) }}
                    </span>
                    <!-- Tag display -->
                    <span
                      v-else-if="field.display_type === 'tag'"
                      :class="['inline-flex items-center rounded-md px-2 py-1 text-xs font-medium mt-1', getColorClass(field.color)]"
                    >
                      {{ getFieldValue(field.key) }}
                    </span>
                    <!-- Default text display -->
                    <p v-else class="text-sm font-semibold break-words mt-0.5">{{ getFieldValue(field.key) }}</p>
                  </div>
                </div>
              </CollapsibleContent>
            </Collapsible>

            <!-- Non-collapsible section -->
            <div v-else>
              <h5 class="py-2 text-sm font-medium">{{ section.label }}</h5>
              <div
                :class="[
                  'grid gap-2',
                  section.columns === 2 ? 'grid-cols-2' : 'grid-cols-1'
                ]"
              >
                <div
                  v-for="field in section.fields.sort((a, b) => a.order - b.order)"
                  :key="field.key"
                  class="bg-muted/50 rounded-md px-3 py-2"
                >
                  <p class="text-[10px] uppercase tracking-wide text-muted-foreground font-medium">{{ field.label }}</p>
                  <!-- Badge display -->
                  <span
                    v-if="field.display_type === 'badge'"
                    :class="['inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold mt-1', getColorClass(field.color)]"
                  >
                    {{ getFieldValue(field.key) }}
                  </span>
                  <!-- Tag display -->
                  <span
                    v-else-if="field.display_type === 'tag'"
                    :class="['inline-flex items-center rounded-md px-2 py-1 text-xs font-medium mt-1', getColorClass(field.color)]"
                  >
                    {{ getFieldValue(field.key) }}
                  </span>
                  <!-- Default text display -->
                  <p v-else class="text-sm font-semibold break-words mt-0.5">{{ getFieldValue(field.key) }}</p>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </ScrollArea>

    <!-- Invite Collaborator Dialog -->
    <Dialog v-model:open="isInviteDialogOpen">
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle>{{ $t('chat.collaboratorInvite') }}</DialogTitle>
          <DialogDescription>
            {{ $t('chat.collaboratorInviteDesc') }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-4 space-y-3">
          <div class="relative">
            <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              v-model="inviteSearchQuery"
              :placeholder="$t('chat.collaboratorSearchPlaceholder')"
              class="pl-9 h-9"
            />
          </div>
          <ScrollArea class="max-h-[280px]">
            <div class="space-y-1">
              <Button
                v-for="user in inviteCandidates"
                :key="user.id"
                variant="ghost"
                class="w-full justify-start"
                :disabled="isInvitingCollaborator"
                @click="inviteCollaborator(user.id)"
              >
                <User class="mr-2 h-4 w-4" />
                <span>{{ user.full_name || user.email }}</span>
              </Button>
              <p
                v-if="inviteCandidates.length === 0"
                class="text-sm text-muted-foreground text-center py-4"
              >
                {{ $t('chat.collaboratorNoEligible') }}
              </p>
            </div>
          </ScrollArea>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>
