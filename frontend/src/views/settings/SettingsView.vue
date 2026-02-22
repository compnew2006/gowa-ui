<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader } from '@/components/shared'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import { toast } from 'vue-sonner'
import { Settings, Bell, Loader2, Globe, MessageSquare, Play, Archive } from 'lucide-vue-next'
import { usersService, organizationService } from '@/services/api'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()

type NotificationSoundKey = 'notification1' | 'notification2' | 'notification'
const DEFAULT_NOTIFICATION_SOUND: NotificationSoundKey = 'notification1'
type AssignedChatResetMode = 'midnight' | 'custom_hour'

const isSubmitting = ref(false)
const isLoading = ref(true)
const isPreviewPlaying = ref(false)
let previewAudio: HTMLAudioElement | null = null

function normalizeNotificationSound(value: unknown): NotificationSoundKey {
  if (value === 'notification2') return 'notification2'
  if (value === 'notification') return 'notification'
  return DEFAULT_NOTIFICATION_SOUND
}

const rawBasePath = typeof window !== 'undefined'
  ? ((window as any).__BASE_PATH__ ?? import.meta.env.BASE_URL ?? '/')
  : (import.meta.env.BASE_URL ?? '/')
const normalizedBasePath = String(rawBasePath).replace(/\/$/, '')

function getNotificationSoundSources(sound: NotificationSoundKey): string[] {
  return [
    normalizedBasePath ? `${normalizedBasePath}/${sound}.mp3` : `/${sound}.mp3`,
    `/${sound}.mp3`
  ].filter((src, index, arr) => arr.indexOf(src) === index)
}

async function previewNotificationSound() {
  const selectedSound = normalizeNotificationSound(notificationSettings.value.notification_sound)
  const sources = getNotificationSoundSources(selectedSound)

  if (previewAudio) {
    previewAudio.pause()
    previewAudio.currentTime = 0
    previewAudio = null
  }

  for (const src of sources) {
    const audio = new Audio(src)
    audio.volume = 0.5
    audio.preload = 'auto'

    try {
      await audio.play()
      previewAudio = audio
      isPreviewPlaying.value = true

      const resetState = () => {
        if (previewAudio === audio) {
          previewAudio = null
          isPreviewPlaying.value = false
        }
      }

      audio.addEventListener('ended', resetState, { once: true })
      audio.addEventListener('error', resetState, { once: true })
      return
    } catch {
      // Try next source path
    }
  }

  isPreviewPlaying.value = false
}

// General Settings
const generalSettings = ref({
  organization_name: 'My Organization',
  default_timezone: 'UTC',
  date_format: 'YYYY-MM-DD',
  mask_phone_numbers: false
})

interface NotificationSettings {
  email_notifications: boolean
  new_message_alerts: boolean
  campaign_updates: boolean
  notification_sound: NotificationSoundKey
}

// Notification Settings
const notificationSettings = ref<NotificationSettings>({
  email_notifications: true,
  new_message_alerts: true,
  campaign_updates: true,
  notification_sound: DEFAULT_NOTIFICATION_SOUND
})

function normalizeAssignedChatResetMode(value: unknown): AssignedChatResetMode {
  return value === 'custom_hour' ? 'custom_hour' : 'midnight'
}

function normalizeAssignedChatResetHour(value: unknown): number {
  const parsed = typeof value === 'number'
    ? value
    : typeof value === 'string'
      ? Number(value)
      : Number.NaN
  if (!Number.isFinite(parsed)) return 0
  const rounded = Math.trunc(parsed)
  return Math.min(23, Math.max(0, rounded))
}

// Chat Preferences (localStorage-only)
const MEDIA_GROUP_WINDOW_KEY = 'chat.mediaGroupWindowSeconds'
const chatSettings = ref({
  media_group_window: 60,
  assigned_chat_reset_enabled: true,
  assigned_chat_reset_mode: 'midnight' as AssignedChatResetMode,
  assigned_chat_reset_hour: 0
})

const timezoneOptions = [
  { value: 'UTC', label: 'UTC (GMT+0)' },
  { value: 'Etc/GMT-2', label: 'UTC+2 (Fixed Offset)' },
  { value: 'Africa/Cairo', label: 'UTC+2 (Cairo)' },
  { value: 'Europe/Athens', label: 'UTC+2 (Athens)' },
  { value: 'Etc/GMT-3', label: 'UTC+3 (Fixed Offset)' },
  { value: 'Asia/Riyadh', label: 'UTC+3 (Riyadh)' },
  { value: 'Europe/Moscow', label: 'UTC+3 (Moscow)' },
  { value: 'America/New_York', label: 'UTC-5/-4 (Eastern Time)' },
  { value: 'America/Chicago', label: 'UTC-6/-5 (Central Time)' },
  { value: 'America/Denver', label: 'UTC-7/-6 (Mountain Time)' },
  { value: 'America/Los_Angeles', label: 'UTC-8/-7 (Pacific Time)' },
  { value: 'Europe/London', label: 'UTC+0/+1 (London)' },
  { value: 'Europe/Paris', label: 'UTC+1/+2 (Paris)' },
  { value: 'Asia/Dubai', label: 'UTC+4 (Dubai)' },
  { value: 'Asia/Karachi', label: 'UTC+5 (Karachi)' },
  { value: 'Asia/Kolkata', label: 'UTC+5:30 (India)' },
  { value: 'Asia/Bangkok', label: 'UTC+7 (Bangkok)' },
  { value: 'Asia/Singapore', label: 'UTC+8 (Singapore)' },
  { value: 'Asia/Tokyo', label: 'UTC+9 (Tokyo)' },
  { value: 'Australia/Sydney', label: 'UTC+10/+11 (Sydney)' }
]

const chatResetHourOptions = Array.from({ length: 24 }, (_, hour) => ({
  value: String(hour),
  label: `${String(hour).padStart(2, '0')}:00`
}))

// Load chat settings from localStorage
try {
  const stored = Number(localStorage.getItem(MEDIA_GROUP_WINDOW_KEY))
  if (Number.isFinite(stored) && stored >= 5 && stored <= 300) {
    chatSettings.value.media_group_window = stored
  }
} catch {
  // Ignore localStorage errors
}

onMounted(async () => {
  try {
    const [orgResponse, userResponse] = await Promise.all([
      organizationService.getSettings(),
      usersService.me()
    ])

    // Organization settings
    const orgData = orgResponse.data.data || orgResponse.data
    if (orgData) {
      generalSettings.value = {
        organization_name: orgData.name || 'My Organization',
        default_timezone: orgData.settings?.timezone || 'UTC',
        date_format: orgData.settings?.date_format || 'YYYY-MM-DD',
        mask_phone_numbers: orgData.settings?.mask_phone_numbers || false
      }

      const resetMode = normalizeAssignedChatResetMode(orgData.settings?.assigned_chat_reset_mode)
      const resetHour = normalizeAssignedChatResetHour(orgData.settings?.assigned_chat_reset_hour)
      chatSettings.value.assigned_chat_reset_enabled = orgData.settings?.assigned_chat_reset_enabled !== false
      chatSettings.value.assigned_chat_reset_mode = resetMode
      chatSettings.value.assigned_chat_reset_hour = resetMode === 'midnight' ? 0 : resetHour
    }

    // User notification settings
    const user = userResponse.data.data || userResponse.data
    if (user.settings) {
      notificationSettings.value = {
        email_notifications: user.settings.email_notifications ?? true,
        new_message_alerts: user.settings.new_message_alerts ?? true,
        campaign_updates: user.settings.campaign_updates ?? true,
        notification_sound: normalizeNotificationSound(user.settings.notification_sound)
      }
    }
  } catch (error) {
    console.error('Failed to load settings:', error)
  } finally {
    isLoading.value = false
  }
})

async function saveGeneralSettings() {
  isSubmitting.value = true
  try {
    await organizationService.updateSettings({
      name: generalSettings.value.organization_name,
      timezone: generalSettings.value.default_timezone,
      date_format: generalSettings.value.date_format,
      mask_phone_numbers: generalSettings.value.mask_phone_numbers
    })
    toast.success(t('settings.generalSaved'))
  } catch (error) {
    toast.error(t('common.failedSave', { resource: t('resources.settings') }))
  } finally {
    isSubmitting.value = false
  }
}

async function saveNotificationSettings() {
  isSubmitting.value = true
  try {
    const notificationSound = normalizeNotificationSound(notificationSettings.value.notification_sound)
    await usersService.updateSettings({
      email_notifications: notificationSettings.value.email_notifications,
      new_message_alerts: notificationSettings.value.new_message_alerts,
      campaign_updates: notificationSettings.value.campaign_updates,
      notification_sound: notificationSound
    })

    if (authStore.user) {
      authStore.user = {
        ...authStore.user,
        settings: {
          ...(authStore.user.settings || {}),
          email_notifications: notificationSettings.value.email_notifications,
          new_message_alerts: notificationSettings.value.new_message_alerts,
          campaign_updates: notificationSettings.value.campaign_updates,
          notification_sound: notificationSound
        }
      }
      localStorage.setItem('user', JSON.stringify(authStore.user))
    }

    toast.success(t('settings.notificationsSaved'))
  } catch (error) {
    toast.error(t('common.failedSave', { resource: t('resources.notificationSettings') }))
  } finally {
    isSubmitting.value = false
  }
}
async function saveChatSettings() {
  isSubmitting.value = true
  const clamped = Math.min(300, Math.max(5, Math.round(chatSettings.value.media_group_window)))
  const normalizedMode = normalizeAssignedChatResetMode(chatSettings.value.assigned_chat_reset_mode)
  const normalizedHour = normalizedMode === 'midnight'
    ? 0
    : normalizeAssignedChatResetHour(chatSettings.value.assigned_chat_reset_hour)

  chatSettings.value.media_group_window = clamped
  chatSettings.value.assigned_chat_reset_mode = normalizedMode
  chatSettings.value.assigned_chat_reset_hour = normalizedHour

  try {
    localStorage.setItem(MEDIA_GROUP_WINDOW_KEY, String(clamped))
    await organizationService.updateSettings({
      assigned_chat_reset_enabled: chatSettings.value.assigned_chat_reset_enabled,
      assigned_chat_reset_mode: normalizedMode,
      assigned_chat_reset_hour: normalizedHour
    })
    toast.success(t('settings.chatPreferencesSaved'))
  } catch (error) {
    toast.error(t('settings.chatPreferencesSaveFailed'))
  } finally {
    isSubmitting.value = false
  }
}

onBeforeUnmount(() => {
  if (previewAudio) {
    previewAudio.pause()
    previewAudio = null
  }
  isPreviewPlaying.value = false
})
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader :title="$t('settings.title')" :subtitle="$t('settings.subtitle')" :icon="Settings" icon-gradient="bg-gradient-to-br from-gray-500 to-gray-600 shadow-gray-500/20" />
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-4 max-w-4xl mx-auto">
        <Tabs default-value="general" class="w-full">
          <TabsList class="grid w-full grid-cols-3 mb-6 bg-white/[0.04] border border-white/[0.08] light:bg-gray-100 light:border-gray-200">
            <TabsTrigger value="general" class="data-[state=active]:bg-white/[0.08] data-[state=active]:text-white text-white/50 light:data-[state=active]:bg-white light:data-[state=active]:text-gray-900 light:text-gray-500">
              <Settings class="h-4 w-4 mr-2" />
              {{ $t('settings.general') }}
            </TabsTrigger>
            <TabsTrigger value="chat" class="data-[state=active]:bg-white/[0.08] data-[state=active]:text-white text-white/50 light:data-[state=active]:bg-white light:data-[state=active]:text-gray-900 light:text-gray-500">
              <MessageSquare class="h-4 w-4 mr-2" />
              {{ $t('settings.chat') }}
            </TabsTrigger>
            <TabsTrigger value="notifications" class="data-[state=active]:bg-white/[0.08] data-[state=active]:text-white text-white/50 light:data-[state=active]:bg-white light:data-[state=active]:text-gray-900 light:text-gray-500">
              <Bell class="h-4 w-4 mr-2" />
              {{ $t('settings.notifications') }}
            </TabsTrigger>
          </TabsList>

          <!-- General Settings Tab -->
          <TabsContent value="general">
            <div class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200">
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-white light:text-gray-900">{{ $t('settings.generalSettings') }}</h3>
                <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.generalSettingsDesc') }}</p>
              </div>
              <div class="p-6 pt-3 space-y-4">
                <div class="space-y-2">
                  <Label for="org_name" class="text-white/70 light:text-gray-700">{{ $t('settings.organizationName') }}</Label>
                  <Input
                    id="org_name"
                    v-model="generalSettings.organization_name"
                    :placeholder="$t('settings.organizationPlaceholder')"
                  />
                </div>
                <div class="grid grid-cols-2 gap-4">
                  <div class="space-y-2">
                    <Label for="timezone" class="text-white/70 light:text-gray-700">{{ $t('settings.defaultTimezone') }}</Label>
                    <Select v-model="generalSettings.default_timezone">
                      <SelectTrigger class="bg-white/[0.04] border-white/[0.1] text-white/70 light:bg-white light:border-gray-200 light:text-gray-700">
                        <SelectValue :placeholder="$t('settings.selectTimezone')" />
                      </SelectTrigger>
                      <SelectContent class="bg-[#141414] border-white/[0.08] light:bg-white light:border-gray-200">
                        <SelectItem
                          v-for="option in timezoneOptions"
                          :key="option.value"
                          :value="option.value"
                          class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100"
                        >
                          {{ option.label }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div class="space-y-2">
                    <Label for="date_format" class="text-white/70 light:text-gray-700">{{ $t('settings.dateFormat') }}</Label>
                    <Select v-model="generalSettings.date_format">
                      <SelectTrigger class="bg-white/[0.04] border-white/[0.1] text-white/70 light:bg-white light:border-gray-200 light:text-gray-700">
                        <SelectValue :placeholder="$t('settings.selectFormat')" />
                      </SelectTrigger>
                      <SelectContent class="bg-[#141414] border-white/[0.08] light:bg-white light:border-gray-200">
                        <SelectItem value="YYYY-MM-DD" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">YYYY-MM-DD</SelectItem>
                        <SelectItem value="DD/MM/YYYY" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">DD/MM/YYYY</SelectItem>
                        <SelectItem value="MM/DD/YYYY" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">MM/DD/YYYY</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div class="space-y-2">
                  <Label class="text-white/70 light:text-gray-700">
                    <Globe class="h-4 w-4 inline mr-1" />
                    {{ $t('settings.language') }}
                  </Label>
                  <LanguageSwitcher class="max-w-xs" />
                  <p class="text-xs text-white/40 light:text-gray-500">{{ $t('settings.languageDesc') }}</p>
                </div>
                <Separator class="bg-white/[0.08] light:bg-gray-200" />
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-white light:text-gray-900">{{ $t('settings.maskPhoneNumbers') }}</p>
                    <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.maskPhoneNumbersDesc') }}</p>
                  </div>
                  <Switch
                    :checked="generalSettings.mask_phone_numbers"
                    @update:checked="generalSettings.mask_phone_numbers = $event"
                  />
                </div>
                <div class="flex justify-end">
                  <Button variant="outline" size="sm" class="bg-white/[0.04] border-white/[0.1] text-white/70 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50" @click="saveGeneralSettings" :disabled="isSubmitting">
                    <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
                    {{ $t('settings.save') }}
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>

          <!-- Notification Settings Tab -->
          <TabsContent value="notifications">
            <div class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200">
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-white light:text-gray-900">{{ $t('settings.notifications') }}</h3>
                <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.notificationsDesc') }}</p>
              </div>
              <div class="p-6 pt-3 space-y-4">
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-white light:text-gray-900">{{ $t('settings.emailNotifications') }}</p>
                    <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.emailNotificationsDesc') }}</p>
                  </div>
                  <Switch
                    :checked="notificationSettings.email_notifications"
                    @update:checked="notificationSettings.email_notifications = $event"
                  />
                </div>
                <Separator class="bg-white/[0.08] light:bg-gray-200" />
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-white light:text-gray-900">{{ $t('settings.newMessageAlerts') }}</p>
                    <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.newMessageAlertsDesc') }}</p>
                  </div>
                  <Switch
                    :checked="notificationSettings.new_message_alerts"
                    @update:checked="notificationSettings.new_message_alerts = $event"
                  />
                </div>
                <Separator class="bg-white/[0.08] light:bg-gray-200" />
                <div class="space-y-2">
                  <div>
                    <p class="font-medium text-white light:text-gray-900">{{ $t('settings.notificationSound') }}</p>
                    <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.notificationSoundDesc') }}</p>
                  </div>
                  <div class="flex items-center gap-2">
                    <Select v-model="notificationSettings.notification_sound">
                      <SelectTrigger class="w-full max-w-xs bg-white/[0.04] border-white/[0.1] text-white/70 light:bg-white light:border-gray-200 light:text-gray-700">
                        <SelectValue :placeholder="$t('settings.selectNotificationSound')" />
                      </SelectTrigger>
                      <SelectContent class="bg-[#141414] border-white/[0.08] light:bg-white light:border-gray-200">
                        <SelectItem value="notification1" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">
                          {{ $t('settings.notificationSound1') }}
                        </SelectItem>
                        <SelectItem value="notification2" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">
                          {{ $t('settings.notificationSound2') }}
                        </SelectItem>
                        <SelectItem value="notification" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">
                          {{ $t('settings.notificationSoundOriginal') }}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      class="shrink-0 bg-white/[0.04] border-white/[0.1] text-white/70 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50"
                      :disabled="isPreviewPlaying"
                      @click="previewNotificationSound"
                    >
                      <Loader2 v-if="isPreviewPlaying" class="h-4 w-4 mr-1 animate-spin" />
                      <Play v-else class="h-4 w-4 mr-1" />
                      {{ $t('settings.previewSound') }}
                    </Button>
                  </div>
                </div>
                <Separator class="bg-white/[0.08] light:bg-gray-200" />
                <div class="flex items-center justify-between">
                  <div>
                    <p class="font-medium text-white light:text-gray-900">{{ $t('settings.campaignUpdates') }}</p>
                    <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.campaignUpdatesDesc') }}</p>
                  </div>
                  <Switch
                    :checked="notificationSettings.campaign_updates"
                    @update:checked="notificationSettings.campaign_updates = $event"
                  />
                </div>
                <div class="flex justify-end pt-4">
                  <Button variant="outline" size="sm" class="bg-white/[0.04] border-white/[0.1] text-white/70 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50" @click="saveNotificationSettings" :disabled="isSubmitting">
                    <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
                    {{ $t('settings.save') }}
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>

          <!-- Chat Preferences Tab -->
          <TabsContent value="chat">
            <div class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200">
              <div class="p-6 pb-3">
                <h3 class="text-lg font-semibold text-white light:text-gray-900">{{ $t('settings.chatPreferences') }}</h3>
                <p class="text-sm text-white/40 light:text-gray-500">{{ $t('settings.chatPreferencesDesc') }}</p>
              </div>
              <div class="p-6 pt-3 space-y-4">
                <div class="space-y-2">
                  <Label for="media_group_window" class="text-white/70 light:text-gray-700">{{ $t('settings.mediaGroupingWindow') }}</Label>
                  <p class="text-xs text-white/40 light:text-gray-500">{{ $t('settings.mediaGroupingWindowDesc') }}</p>
                  <Select
                    :model-value="String(chatSettings.media_group_window)"
                    @update:model-value="(v: unknown) => { if (typeof v === 'string') chatSettings.media_group_window = Number(v) }"
                  >
                    <SelectTrigger class="w-full max-w-xs bg-white/[0.04] border-white/[0.1] text-white/70 light:bg-white light:border-gray-200 light:text-gray-700">
                      <SelectValue :placeholder="$t('settings.selectGroupingWindow')" />
                    </SelectTrigger>
                    <SelectContent class="bg-[#141414] border-white/[0.08] light:bg-white light:border-gray-200">
                      <SelectItem value="15" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ $t('settings.mediaGroupingWindow15Seconds') }}</SelectItem>
                      <SelectItem value="30" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ $t('settings.mediaGroupingWindow30Seconds') }}</SelectItem>
                      <SelectItem value="60" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ $t('settings.mediaGroupingWindow60SecondsDefault') }}</SelectItem>
                      <SelectItem value="120" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ $t('settings.mediaGroupingWindow2Minutes') }}</SelectItem>
                      <SelectItem value="180" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ $t('settings.mediaGroupingWindow3Minutes') }}</SelectItem>
                      <SelectItem value="300" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ $t('settings.mediaGroupingWindow5Minutes') }}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <Separator class="bg-white/[0.08] light:bg-gray-200" />
                <div class="space-y-2">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <Label class="text-white/70 light:text-gray-700">{{ $t('settings.assignedChatResetEnabled') }}</Label>
                      <p class="text-xs text-white/40 light:text-gray-500">
                        {{ $t('settings.assignedChatResetEnabledDesc') }}
                      </p>
                    </div>
                    <Switch
                      :checked="chatSettings.assigned_chat_reset_enabled"
                      @update:checked="chatSettings.assigned_chat_reset_enabled = $event"
                    />
                  </div>
                  <Label class="text-white/70 light:text-gray-700">{{ $t('settings.assignedChatResetSchedule') }}</Label>
                  <p class="text-xs text-white/40 light:text-gray-500">
                    {{ $t('settings.assignedChatResetScheduleDesc') }}
                  </p>
                  <Select v-model="chatSettings.assigned_chat_reset_mode" :disabled="!chatSettings.assigned_chat_reset_enabled">
                    <SelectTrigger class="w-full max-w-xs bg-white/[0.04] border-white/[0.1] text-white/70 light:bg-white light:border-gray-200 light:text-gray-700">
                      <SelectValue :placeholder="$t('settings.selectResetSchedule')" />
                    </SelectTrigger>
                    <SelectContent class="bg-[#141414] border-white/[0.08] light:bg-white light:border-gray-200">
                      <SelectItem value="midnight" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">
                        {{ $t('settings.defaultMidnight') }}
                      </SelectItem>
                      <SelectItem value="custom_hour" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">
                        {{ $t('settings.customHour') }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div v-if="chatSettings.assigned_chat_reset_enabled && chatSettings.assigned_chat_reset_mode === 'custom_hour'" class="space-y-2">
                  <Label class="text-white/70 light:text-gray-700">{{ $t('settings.customResetHour') }}</Label>
                  <Select
                    :model-value="String(chatSettings.assigned_chat_reset_hour)"
                    @update:model-value="(v: unknown) => { if (typeof v === 'string') chatSettings.assigned_chat_reset_hour = Number(v) }"
                    :disabled="!chatSettings.assigned_chat_reset_enabled"
                  >
                    <SelectTrigger class="w-full max-w-xs bg-white/[0.04] border-white/[0.1] text-white/70 light:bg-white light:border-gray-200 light:text-gray-700">
                      <SelectValue :placeholder="$t('settings.selectResetHour')" />
                    </SelectTrigger>
                    <SelectContent class="bg-[#141414] border-white/[0.08] light:bg-white light:border-gray-200">
                      <SelectItem
                        v-for="option in chatResetHourOptions"
                        :key="option.value"
                        :value="option.value"
                        class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <Separator class="bg-white/[0.08] light:bg-gray-200" />
                <div class="space-y-3">
                  <div>
                    <h4 class="text-sm font-medium text-white light:text-gray-900">{{ $t('settings.chatQueues') }}</h4>
                    <p class="text-xs text-white/40 light:text-gray-500">{{ $t('settings.chatQueuesDesc') }}</p>
                  </div>
                  <div class="grid gap-2 sm:grid-cols-1">
                    <RouterLink
                      to="/settings/closed-chats"
                      class="rounded-lg border border-zinc-400/30 bg-zinc-500/10 px-3 py-2 text-xs text-zinc-100 hover:bg-zinc-500/20 light:border-zinc-200 light:bg-zinc-100 light:text-zinc-700"
                    >
                      <span class="inline-flex items-center gap-1.5 font-medium"><Archive class="h-3.5 w-3.5" /> {{ $t('settings.closedChats') }}</span>
                    </RouterLink>
                  </div>
                </div>
                <div class="flex justify-end pt-4">
                  <Button variant="outline" size="sm" class="bg-white/[0.04] border-white/[0.1] text-white/70 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50" @click="saveChatSettings" :disabled="isSubmitting">
                    <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
                    {{ $t('settings.save') }}
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </ScrollArea>
  </div>
</template>
