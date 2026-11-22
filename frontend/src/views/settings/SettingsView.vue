<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader, AuditLogPanel } from '@/components/shared'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import { toast } from 'vue-sonner'
import { Settings, Bell, Loader2, Globe } from 'lucide-vue-next'
import { usersService, organizationService } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationsStore } from '@/stores/organizations'

const { t } = useI18n()
const authStore = useAuthStore()
const organizationsStore = useOrganizationsStore()

// The active org may be overridden by the X-Organization-ID header
// (localStorage.selected_organization_id) when a super admin switches orgs.
// That override is what the backend uses for scoping, so we must read it here
// too — otherwise the activity log panel would query the user's default org
// instead of the currently-active one.
const orgID = computed(
  () => localStorage.getItem('selected_organization_id') || authStore.organizationId,
)
const userID = computed(() => authStore.user?.id || '')

const isSubmitting = ref(false)
const isLoading = ref(true)
const settingsLoadError = ref(false)

// General Settings.
//
// Task 12: the name and timezone have ONE source — the server response
// (org.Name + settings["timezone"]). The initial values are empty, not
// hardcoded "My Organization" / "UTC": the old defaults were rendered as if
// they were the real values whenever the load failed, which is how the UI
// showed "My Organization" while the switcher showed "Default Organization"
// and "UTC" while the server stored America/New_York.
const generalSettings = ref({
  organization_name: '',
  default_timezone: '',
  date_format: 'YYYY-MM-DD',
  mask_phone_numbers: false,
})

// Timezones offered in the dropdown. The stored value is always added as an
// option when it is not one of these (e.g. Africa/Cairo), so the control can
// display whatever the server stored instead of falling back to UTC.
const staticTimezones = [
  { value: 'UTC', label: 'UTC' },
  { value: 'America/New_York', label: 'Eastern Time' },
  { value: 'America/Los_Angeles', label: 'Pacific Time' },
  { value: 'Europe/London', label: 'London' },
  { value: 'Asia/Tokyo', label: 'Tokyo' },
]
const availableTimezones = computed(() => {
  const tz = generalSettings.value.default_timezone
  if (tz && !staticTimezones.some((entry) => entry.value === tz)) {
    return [...staticTimezones, { value: tz, label: tz }]
  }
  return staticTimezones
})

// Notification Settings
const notificationSettings = ref({
  email_notifications: true,
  new_message_alerts: true,
  campaign_updates: true
})

// Close-rating (CSAT) settings moved to the per-account page
// (Settings → Accounts → account → Rating panel): each WhatsApp number has
// its own prompt/thanks/lexicon.

// Bump these keys to force the AuditLogPanel to remount and refetch after a save.
// The backend writes audit entries asynchronously in a goroutine, so we delay
// the remount slightly to give the write time to hit the DB before refetching.
const generalLogKey = ref(0)
const notificationLogKey = ref(0)

function refreshActivityLog(key: typeof generalLogKey) {
  setTimeout(() => { key.value++ }, 500)
}

async function loadSettings() {
  isLoading.value = true
  settingsLoadError.value = false
  // The two fetches are independent: a failure loading the user settings
  // must not discard the organization settings (or vice versa). The old
  // Promise.all let one failure leave the form showing the hardcoded
  // defaults — "My Organization" / "UTC" — as if they were the real values.
  const orgResult = await organizationService.getSettings().then(
    (response) => ({ ok: true as const, data: response.data?.data || response.data }),
    (error: unknown) => ({ ok: false as const, error }),
  )
  if (orgResult.ok) {
    const orgData = orgResult.data
    if (orgData) {
      generalSettings.value = {
        organization_name: orgData.name || '',
        default_timezone: orgData.settings?.timezone || '',
        date_format: orgData.settings?.date_format || 'YYYY-MM-DD',
        mask_phone_numbers: orgData.settings?.mask_phone_numbers || false,
      }
    }
  } else {
    console.error('Failed to load organization settings:', orgResult.error)
    settingsLoadError.value = true
  }

  // User notification settings
  try {
    const userResponse = await usersService.me()
    const user = userResponse.data.data || userResponse.data
    if (user.settings) {
      notificationSettings.value = {
        email_notifications: user.settings.email_notifications ?? true,
        new_message_alerts: user.settings.new_message_alerts ?? true,
        campaign_updates: user.settings.campaign_updates ?? true
      }
    }
  } catch (error) {
    console.error('Failed to load user settings:', error)
  } finally {
    isLoading.value = false
  }
}

onMounted(loadSettings)

async function saveGeneralSettings() {
  isSubmitting.value = true
  try {
    const payload: any = {
      name: generalSettings.value.organization_name,
      timezone: generalSettings.value.default_timezone,
      date_format: generalSettings.value.date_format,
      mask_phone_numbers: generalSettings.value.mask_phone_numbers
    }
    const updateResponse = await organizationService.updateSettings(payload)
    // The update response echoes the stored values (same shape as GET):
    // sync the form from the server, never from the local input.
    const saved = updateResponse.data?.data || updateResponse.data
    if (saved) {
      generalSettings.value = {
        organization_name: saved.name ?? generalSettings.value.organization_name,
        default_timezone: saved.settings?.timezone ?? generalSettings.value.default_timezone,
        date_format: saved.settings?.date_format ?? generalSettings.value.date_format,
        mask_phone_numbers: saved.settings?.mask_phone_numbers ?? generalSettings.value.mask_phone_numbers,
      }
      // The switcher reads org names from the organizations store: refresh
      // it so it shows the renamed org instead of the stale cached name.
      await organizationsStore.fetchMyOrganizations()
      if (authStore.user?.is_super_admin) {
        await organizationsStore.fetchOrganizations()
      }
    }
    toast.success(t('settings.generalSaved'))
    refreshActivityLog(generalLogKey)
  } catch (error) {
    toast.error(t('common.failedSave', { resource: t('resources.settings') }))
  } finally {
    isSubmitting.value = false
  }
}

async function saveNotificationSettings() {
  isSubmitting.value = true
  try {
    await usersService.updateSettings({
      email_notifications: notificationSettings.value.email_notifications,
      new_message_alerts: notificationSettings.value.new_message_alerts,
      campaign_updates: notificationSettings.value.campaign_updates
    })
    toast.success(t('settings.notificationsSaved'))
    refreshActivityLog(notificationLogKey)
  } catch (error) {
    toast.error(t('common.failedSave', { resource: t('resources.notificationSettings') }))
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader :title="$t('settings.title')" :subtitle="$t('settings.subtitle')" :icon="Settings" icon-gradient="bg-gradient-to-br from-gray-500 to-gray-600 shadow-gray-500/20" />
    <ScrollArea class="flex-1">
      <div class="p-6 space-y-4 max-w-4xl mx-auto">
        <Tabs default-value="general" class="w-full">
          <TabsList class="grid w-full grid-cols-2 mb-6 bg-white/[0.04] border border-white/[0.08] light:bg-gray-100 light:border-gray-200">
            <TabsTrigger value="general" class="data-[state=active]:bg-white/[0.08] data-[state=active]:text-white text-white/50 light:data-[state=active]:bg-white light:data-[state=active]:text-gray-900 light:text-gray-500">
              <Settings class="h-4 w-4 mr-2" />
              {{ $t('settings.general') }}
            </TabsTrigger>
            <TabsTrigger value="notifications" class="data-[state=active]:bg-white/[0.08] data-[state=active]:text-white text-white/50 light:data-[state=active]:bg-white light:data-[state=active]:text-gray-900 light:text-gray-500">
              <Bell class="h-4 w-4 mr-2" />
              {{ $t('settings.notifications') }}
            </TabsTrigger>
          </TabsList>

          <!-- General Settings Tab -->
          <TabsContent value="general">
            <div v-if="isLoading" class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200 p-6 text-sm text-white/40 light:text-gray-500">
              {{ $t('common.loading', 'Loading...') }}
            </div>
            <div v-else-if="settingsLoadError" class="rounded-xl border border-white/[0.08] bg-white/[0.02] light:bg-white light:border-gray-200 p-6 space-y-3">
              <p class="text-sm font-medium text-white light:text-gray-900">{{ $t('common.loadErrorTitle', 'Failed to load settings') }}</p>
              <p class="text-sm text-white/40 light:text-gray-500">{{ $t('common.loadErrorDescription', 'Check your connection and try again.') }}</p>
              <Button variant="outline" size="sm" class="bg-white/[0.04] border-white/[0.1] text-white/70 hover:bg-white/[0.08] hover:text-white light:bg-white light:border-gray-200 light:text-gray-700 light:hover:bg-gray-50" @click="loadSettings">{{ $t('common.retry', 'Retry') }}</Button>
            </div>
            <template v-else>
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
                        <SelectItem v-for="tz in availableTimezones" :key="tz.value" :value="tz.value" class="text-white/70 focus:bg-white/[0.08] focus:text-white light:text-gray-700 light:focus:bg-gray-100">{{ tz.label }}</SelectItem>
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
            </template>

            <div v-if="orgID" class="mt-4">
              <AuditLogPanel :key="generalLogKey" resource-type="settings.general" :resource-id="orgID" />
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
            <div v-if="userID" class="mt-4">
              <AuditLogPanel :key="notificationLogKey" resource-type="settings.notification" :resource-id="userID" />
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </ScrollArea>
  </div>
</template>
