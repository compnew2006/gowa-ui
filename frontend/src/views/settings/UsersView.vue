<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { PageHeader, SearchInput, DataTable, CrudFormDialog, DeleteConfirmDialog, IconButton, ErrorState, type Column } from '@/components/shared'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { useUsersStore, type User } from '@/stores/users'
import { useAuthStore } from '@/stores/auth'
import { useRolesStore } from '@/stores/roles'
import { useOrganizationsStore } from '@/stores/organizations'
import { accountsService } from '@/services/api'
import { toast } from 'vue-sonner'
import { Plus, Pencil, Trash2, UserMinus, User as UserIcon, Shield, ShieldCheck, UserCog, Users, Link, UserPlus, Loader2, Smartphone } from 'lucide-vue-next'
import { useCrudState } from '@/composables/useCrudState'
import { getErrorMessage } from '@/lib/api-utils'
import { formatDate } from '@/lib/utils'
import { ROLE_BADGE_VARIANTS } from '@/lib/constants'
import { useSearchPagination } from '@/composables/useSearchPagination'

const { t } = useI18n()

const usersStore = useUsersStore()
const authStore = useAuthStore()
const rolesStore = useRolesStore()
const organizationsStore = useOrganizationsStore()

interface UserFormData {
  email: string
  password: string
  full_name: string
  role_id: string
  is_active: boolean
  is_super_admin: boolean
  whatsapp_account_ids: string[]
}

const defaultFormData: UserFormData = { email: '', password: '', full_name: '', role_id: '', is_active: true, is_super_admin: false, whatsapp_account_ids: [] }

const {
  isLoading, isSubmitting, isDialogOpen, deleteDialogOpen, itemToDelete: userToDelete,
  formData, openCreateDialog: baseOpenCreateDialog, openDeleteDialog, closeDialog, closeDeleteDialog,
} = useCrudState<User, UserFormData>(defaultFormData)

const users = ref<User[]>([])
const isDeleting = ref(false)
const error = ref(false)
const onlineCount = ref(0)

// Filters. Radix's <SelectItem> can't take value="", so use a sentinel for
// "no role filter" and translate it to undefined before hitting the API.
const ALL_ROLES = '__all'
const roleFilter = ref<string>(ALL_ROLES)
const onlineOnly = ref(false)

const { searchQuery, currentPage, totalItems, pageSize, handlePageChange } = useSearchPagination({
  fetchFn: () => fetchUsers(),
})

// Re-fetch (and reset to page 1) whenever a filter changes.
watch([roleFilter, onlineOnly], () => {
  currentPage.value = 1
  fetchUsers()
})

const columns = computed<Column<User>[]>(() => [
  { key: 'user', label: t('users.user'), width: 'w-[280px]', sortable: true, sortKey: 'full_name' },
  { key: 'email', label: t('common.email'), sortable: true, sortKey: 'email' },
  { key: 'role', label: t('users.role'), sortable: true, sortKey: 'role.name' },
  { key: 'status', label: t('users.status'), sortable: true, sortKey: 'is_active' },
  { key: 'created', label: t('users.created'), sortable: true, sortKey: 'created_at' },
  { key: 'actions', label: t('common.actions'), align: 'right' },
])

// Sorting state
const sortKey = ref('full_name')
const sortDirection = ref<'asc' | 'desc'>('asc')

const currentUserId = computed(() => authStore.user?.id)
const isSuperAdmin = computed(() => authStore.user?.is_super_admin || false)
const canWrite = computed(() => authStore.hasPermission('users', 'write'))
const breadcrumbs = computed(() => [{ label: t('nav.settings'), href: '/settings' }, { label: t('nav.users') }])
const getDefaultRoleId = () => rolesStore.roles.find(r => r.name === 'agent' && r.is_system)?.id || ''

function openCreateDialog() { formData.value.role_id = getDefaultRoleId(); baseOpenCreateDialog() }

// Org accounts for the assignment checklist (loaded best-effort)
const accounts = ref<{ id: string; name: string }[]>([])

async function loadAccounts() {
  try {
    const response = await accountsService.list()
    accounts.value = response.data.data?.accounts || response.data.accounts || []
  } catch {
    accounts.value = []
  }
}

function toggleAccount(accountId: string, checked: boolean) {
  const ids = new Set(formData.value.whatsapp_account_ids)
  if (checked) ids.add(accountId)
  else ids.delete(accountId)
  formData.value.whatsapp_account_ids = [...ids]
}

// Per-user account access dialog (replaces the assignment section in the
// user detail page): an icon in the row opens this popup without navigating.
const accessDialogOpen = ref(false)
const accessUser = ref<User | null>(null)
const accessIds = ref<string[]>([])
const isLoadingAccess = ref(false)
const isSavingAccess = ref(false)

async function openAccessDialog(user: User) {
  accessUser.value = user
  accessIds.value = []
  accessDialogOpen.value = true
  isLoadingAccess.value = true
  try {
    const full = await usersStore.fetchUser(user.id)
    accessIds.value = [...(full.whatsapp_account_ids || [])]
  } catch {
    toast.error(t('common.failedLoad', { resource: t('resources.user') }))
  } finally {
    isLoadingAccess.value = false
  }
}

function toggleAccess(accountId: string, checked: boolean) {
  const ids = new Set(accessIds.value)
  if (checked) ids.add(accountId)
  else ids.delete(accountId)
  accessIds.value = [...ids]
}

async function saveAccess() {
  if (!accessUser.value) return
  isSavingAccess.value = true
  try {
    await usersStore.updateUser(accessUser.value.id, { whatsapp_account_ids: accessIds.value })
    toast.success(t('common.updatedSuccess', { resource: t('resources.User') }))
    accessDialogOpen.value = false
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedSave', { resource: t('resources.user') })))
  } finally {
    isSavingAccess.value = false
  }
}

watch(() => organizationsStore.selectedOrgId, () => { fetchUsers(); rolesStore.fetchRoles(); loadAccounts() })
onMounted(() => { fetchUsers(); rolesStore.fetchRoles(); loadAccounts() })

async function fetchUsers() {
  isLoading.value = true
  error.value = false
  try {
    const response = await usersStore.fetchUsers({
      search: searchQuery.value || undefined,
      page: currentPage.value,
      limit: pageSize,
      role_id: roleFilter.value === ALL_ROLES ? undefined : roleFilter.value,
      online_only: onlineOnly.value || undefined,
    })
    users.value = response.users
    totalItems.value = response.total
    onlineCount.value = response.online_count
  } catch {
    toast.error(t('common.failedLoad', { resource: t('resources.users') }))
    error.value = true
  } finally { isLoading.value = false }
}

async function createUser() {
  if (!formData.value.email.trim() || !formData.value.full_name.trim()) { toast.error(t('users.fillEmailName')); return }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.value.email.trim())) { toast.error(t('validation.email')); return }
  if (!formData.value.password.trim()) { toast.error(t('users.passwordRequired')); return }
  if (!formData.value.role_id) { toast.error(t('users.selectRoleRequired')); return }

  isSubmitting.value = true
  try {
    await usersStore.createUser({
      email: formData.value.email,
      password: formData.value.password,
      full_name: formData.value.full_name,
      role_id: formData.value.role_id || undefined,
      is_super_admin: isSuperAdmin.value && formData.value.is_super_admin ? true : undefined,
      whatsapp_account_ids: formData.value.whatsapp_account_ids.length > 0 ? formData.value.whatsapp_account_ids : undefined,
    })
    toast.success(t('common.createdSuccess', { resource: t('resources.User') }))
    closeDialog()
    await fetchUsers()
  } catch (e) { toast.error(getErrorMessage(e, t('common.failedSave', { resource: t('resources.user') }))) }
  finally { isSubmitting.value = false }
}

async function confirmDelete() {
  if (!userToDelete.value) return
  const isMemberRemoval = userToDelete.value.is_member
  isDeleting.value = true
  try {
    await usersStore.deleteUser(userToDelete.value.id)
    toast.success(isMemberRemoval ? t('users.memberRemoved') : t('common.deletedSuccess', { resource: t('resources.User') }))
    closeDeleteDialog()
    await fetchUsers()
  } catch (e) {
    toast.error(getErrorMessage(e, t('common.failedDelete', { resource: t('resources.user') })))
  } finally {
    isDeleting.value = false
  }
}

function getRoleBadgeVariant(name: string): 'default' | 'secondary' | 'outline' { return ROLE_BADGE_VARIANTS[name.toLowerCase()] || 'outline' }
function getRoleIcon(name: string) { return { admin: ShieldCheck, manager: Shield }[name.toLowerCase()] || UserCog }
function getRoleName(user: User) { return user.role?.name || t('users.noRole') }

// Add existing user dialog
const isAddExistingOpen = ref(false)
const addExistingEmail = ref('')
const addExistingRoleId = ref('')
const isAddExistingSubmitting = ref(false)

function openAddExistingDialog() {
  addExistingEmail.value = ''
  addExistingRoleId.value = ''
  isAddExistingOpen.value = true
}

async function submitAddExisting() {
  if (!addExistingEmail.value.trim()) {
    toast.error(t('users.enterEmail'))
    return
  }
  isAddExistingSubmitting.value = true
  try {
    await organizationsStore.addMember({
      email: addExistingEmail.value.trim(),
      role_id: addExistingRoleId.value || undefined,
    })
    toast.success(t('users.existingUserAdded'))
    isAddExistingOpen.value = false
    await fetchUsers()
  } catch (e) {
    toast.error(getErrorMessage(e, t('users.addExistingFailed')))
  } finally {
    isAddExistingSubmitting.value = false
  }
}

async function copyInviteLink() {
  const orgId = organizationsStore.selectedOrgId || authStore.organizationId
  const basePath = ((window as any).__BASE_PATH__ ?? '').replace(/\/$/, '')
  const url = `${window.location.origin}${basePath}/register?org=${orgId}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success(t('users.inviteLinkCopied'))
  } catch {
    toast.error(t('common.clipboardFailed'))
  }
}
</script>

<template>
  <div class="flex flex-col h-full bg-[#0a0a0b] light:bg-gray-50">
    <PageHeader :title="$t('users.title')" :icon="Users" icon-gradient="bg-gradient-to-br from-blue-500 to-indigo-600 shadow-blue-500/20" back-link="/settings" :breadcrumbs="breadcrumbs">
      <template #actions>
        <Button variant="outline" size="sm" @click="copyInviteLink"><Link class="h-4 w-4 mr-2" />{{ $t('users.copyInviteLink') }}</Button>
        <Button v-if="organizationsStore.isMultiOrg && authStore.hasPermission('organizations', 'assign')" variant="outline" size="sm" @click="openAddExistingDialog"><UserPlus class="h-4 w-4 mr-2" />{{ $t('users.addExistingUser') }}</Button>
        <Button variant="outline" size="sm" @click="openCreateDialog"><Plus class="h-4 w-4 mr-2" />{{ $t('users.addUser') }}</Button>
      </template>
    </PageHeader>

    <!-- Error State -->
    <ErrorState
      v-if="error && !isLoading"
      :title="$t('common.loadErrorTitle')"
      :description="$t('common.loadErrorDescription')"
      :retry-label="$t('common.retryLoad')"
      class="flex-1"
      @retry="fetchUsers"
    />

    <ScrollArea v-else class="flex-1">
      <div class="p-6">
        <div>
          <Card>
            <CardHeader>
              <div class="flex items-center justify-between flex-wrap gap-4">
                <div>
                  <div class="flex items-center gap-2">
                    <CardTitle>{{ $t('users.yourUsers') }}</CardTitle>
                    <Badge variant="outline" class="border-green-600 text-green-600 gap-1.5">
                      <span class="inline-block h-2 w-2 rounded-full bg-green-600" />
                      {{ onlineCount }} {{ $t('users.online', 'online') }}
                    </Badge>
                  </div>
                  <CardDescription>{{ $t('users.subtitle') }}. <RouterLink to="/settings/roles" class="text-primary hover:underline">{{ $t('users.manageRoles') }}</RouterLink></CardDescription>
                </div>
                <div class="flex items-center gap-3 flex-wrap">
                  <Select v-model="roleFilter">
                    <SelectTrigger class="w-44 h-9">
                      <SelectValue :placeholder="$t('users.allRoles', 'All roles')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem :value="ALL_ROLES">{{ $t('users.allRoles', 'All roles') }}</SelectItem>
                      <SelectItem v-for="role in rolesStore.roles" :key="role.id" :value="role.id" class="capitalize">
                        {{ role.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <label class="flex items-center gap-2 text-sm">
                    <Switch :checked="onlineOnly" @update:checked="onlineOnly = $event" />
                    <span>{{ $t('users.onlineOnly', 'Online only') }}</span>
                  </label>
                  <SearchInput v-model="searchQuery" :placeholder="$t('users.searchUsers') + '...'" class="w-64" />
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <DataTable :items="users" :columns="columns" :is-loading="isLoading" :empty-icon="UserIcon" :empty-title="searchQuery ? $t('users.noMatchingUsers') : $t('users.noUsersFound')" :empty-description="searchQuery ? $t('users.noMatchingUsersDesc') : $t('users.noUsersFoundDesc')" v-model:sort-key="sortKey" v-model:sort-direction="sortDirection" server-pagination :current-page="currentPage" :total-items="totalItems" :page-size="pageSize" item-name="users" @page-change="handlePageChange">
                <template #cell-user="{ item: user }">
                  <RouterLink :to="`/settings/users/${user.id}`" class="flex items-center gap-3 text-inherit no-underline hover:opacity-80">
                    <div class="h-9 w-9 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                      <component :is="getRoleIcon(getRoleName(user))" class="h-4 w-4 text-primary" />
                    </div>
                    <div class="min-w-0">
                      <div class="flex items-center gap-2">
                        <p class="font-medium truncate">{{ user.full_name }}</p>
                        <Badge v-if="user.id === currentUserId" variant="outline" class="text-xs">{{ $t('users.you') }}</Badge>
                        <Badge v-if="user.is_super_admin" variant="default" class="text-xs">{{ $t('users.superAdmin') }}</Badge>
                        <Badge v-if="user.is_member" variant="secondary" class="text-xs">{{ $t('users.member') }}</Badge>
                      </div>
                    </div>
                  </RouterLink>
                </template>
                <template #cell-email="{ item: user }">
                  <span class="text-sm text-muted-foreground truncate">{{ user.email }}</span>
                </template>
                <template #cell-role="{ item: user }">
                  <Badge :variant="getRoleBadgeVariant(getRoleName(user))" class="capitalize">{{ getRoleName(user) }}</Badge>
                </template>
                <template #cell-status="{ item: user }">
                  <Badge variant="outline" :class="user.is_active ? 'border-green-600 text-green-600' : ''">{{ user.is_active ? $t('common.active') : $t('common.inactive') }}</Badge>
                </template>
                <template #cell-created="{ item: user }">
                  <span class="text-muted-foreground">{{ formatDate(user.created_at) }}</span>
                </template>
                <template #cell-actions="{ item: user }">
                  <div class="flex items-center justify-end gap-1">
                    <IconButton
                      v-if="canWrite && accounts.length > 0"
                      :icon="Smartphone"
                      :label="$t('users.accountAssignments')"
                      class="h-8 w-8"
                      @click="openAccessDialog(user)"
                    />
                    <RouterLink :to="`/settings/users/${user.id}`">
                      <IconButton :icon="Pencil" :label="$t('users.editUserTooltip')" class="h-8 w-8" />
                    </RouterLink>
                    <IconButton
                      :label="user.is_member
                        ? $t('users.removeMemberTooltip')
                        : (user.id === currentUserId ? $t('users.cantDeleteYourself') : $t('users.deleteUserTooltip'))"
                      class="h-8 w-8"
                      :disabled="user.id === currentUserId"
                      @click="openDeleteDialog(user)"
                    >
                      <component :is="user.is_member ? UserMinus : Trash2" class="h-4 w-4 text-destructive" />
                    </IconButton>
                  </div>
                </template>
                <template #empty-action>
                  <Button variant="outline" size="sm" @click="openCreateDialog"><Plus class="h-4 w-4 mr-2" />{{ $t('users.addUser') }}</Button>
                </template>
              </DataTable>
            </CardContent>
          </Card>
        </div>
      </div>
    </ScrollArea>

    <CrudFormDialog v-model:open="isDialogOpen" :is-editing="false" :is-submitting="isSubmitting" :edit-title="$t('users.editUserTitle')" :create-title="$t('users.addUserTitle')" :edit-description="$t('users.editUserDesc')" :create-description="$t('users.addUserDesc')" :edit-submit-label="$t('users.updateUser')" :create-submit-label="$t('users.createUser')" @submit="createUser">
      <div class="space-y-4">
        <div class="space-y-2"><Label for="full_name">{{ $t('users.fullName') }} <span class="text-destructive">*</span></Label><Input id="full_name" v-model="formData.full_name" :placeholder="$t('users.fullNamePlaceholder')" /></div>
        <div class="space-y-2"><Label for="email">{{ $t('common.email') }} <span class="text-destructive">*</span></Label><Input id="email" v-model="formData.email" type="email" :placeholder="$t('users.emailPlaceholder')" /></div>
        <div class="space-y-2"><Label for="password">{{ $t('users.password') }} <span class="text-destructive">*</span></Label><Input id="password" v-model="formData.password" type="password" :placeholder="$t('users.passwordPlaceholder')" /></div>
        <div class="space-y-2">
          <Label for="role">{{ $t('users.role') }} <span class="text-destructive">*</span></Label>
          <Select v-model="formData.role_id">
            <SelectTrigger>
              <SelectValue :placeholder="$t('users.selectRole')">
                <template v-if="formData.role_id">
                  <span class="capitalize">{{ rolesStore.roles.find(r => r.id === formData.role_id)?.name }}</span>
                  <Badge v-if="rolesStore.roles.find(r => r.id === formData.role_id)?.is_system" variant="secondary" class="text-xs ml-2">{{ $t('users.system') }}</Badge>
                </template>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="role in rolesStore.roles" :key="role.id" :value="role.id">
                <div class="flex items-center gap-2">
                  <span class="capitalize">{{ role.name }}</span>
                  <Badge v-if="role.is_system" variant="secondary" class="text-xs">{{ $t('users.system') }}</Badge>
                </div>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="accounts.length > 0" class="space-y-2 border-t pt-4">
          <Label>{{ $t('users.accountAssignments') }}</Label>
          <p class="text-xs text-muted-foreground">{{ $t('users.accountAssignmentsDesc') }}</p>
          <div class="space-y-2 pt-1">
            <div v-for="account in accounts" :key="account.id" class="flex items-center gap-2">
              <Checkbox
                :id="`create-assign-${account.id}`"
                :checked="formData.whatsapp_account_ids.includes(account.id)"
                @update:checked="(checked: boolean | 'indeterminate') => toggleAccount(account.id, checked === true)"
              />
              <label :for="`create-assign-${account.id}`" class="text-sm cursor-pointer">{{ account.name }}</label>
            </div>
          </div>
        </div>
        <div v-if="isSuperAdmin" class="flex items-center justify-between border-t pt-4"><div><Label for="is_super_admin" class="font-normal cursor-pointer">{{ $t('users.superAdminLabel') }}</Label><p class="text-xs text-muted-foreground">{{ $t('users.superAdminDesc') }}</p></div><Switch id="is_super_admin" :checked="formData.is_super_admin" @update:checked="formData.is_super_admin = $event" /></div>
      </div>
    </CrudFormDialog>

    <DeleteConfirmDialog v-model:open="deleteDialogOpen" :title="userToDelete?.is_member ? $t('users.removeMember') : $t('users.deleteUser')" :description="userToDelete?.is_member ? $t('users.removeMemberWarning') : undefined" :item-name="userToDelete?.full_name" :is-submitting="isDeleting" @confirm="confirmDelete" />

    <!-- Account Access Dialog -->
    <Dialog v-model:open="accessDialogOpen">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('users.accountAssignments') }}</DialogTitle>
          <DialogDescription>
            <span v-if="accessUser" class="font-medium text-foreground">{{ accessUser.full_name }}</span>
            — {{ $t('users.accountAssignmentsDesc') }}
          </DialogDescription>
        </DialogHeader>
        <div class="py-2">
          <div v-if="isLoadingAccess" class="flex justify-center py-6">
            <Loader2 class="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
          <p v-else-if="accounts.length === 0" class="text-sm text-muted-foreground py-4 text-center">
            {{ $t('accounts.noAccounts', 'No WhatsApp accounts linked') }}
          </p>
          <ScrollArea v-else class="max-h-72">
            <div class="space-y-2 pr-2">
              <div v-for="account in accounts" :key="account.id" class="flex items-center gap-2">
                <Checkbox
                  :id="`access-${account.id}`"
                  :checked="accessIds.includes(account.id)"
                  @update:checked="(checked: boolean | 'indeterminate') => toggleAccess(account.id, checked === true)"
                />
                <label :for="`access-${account.id}`" class="text-sm cursor-pointer">{{ account.name }}</label>
              </div>
            </div>
          </ScrollArea>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="accessDialogOpen = false">{{ $t('common.cancel') }}</Button>
          <Button @click="saveAccess" :disabled="isSavingAccess || isLoadingAccess">
            <Loader2 v-if="isSavingAccess" class="h-4 w-4 mr-2 animate-spin" />
            {{ $t('common.save') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Add Existing User Dialog -->
    <Dialog v-model:open="isAddExistingOpen">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('users.addExistingUserTitle') }}</DialogTitle>
          <DialogDescription>{{ $t('users.addExistingUserDesc') }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-4">
          <div class="space-y-2">
            <Label for="existing-email">{{ $t('common.email') }} <span class="text-destructive">*</span></Label>
            <Input id="existing-email" v-model="addExistingEmail" type="email" :placeholder="$t('users.existingEmailPlaceholder')" />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('users.role') }}</Label>
            <Select v-model="addExistingRoleId">
              <SelectTrigger>
                <SelectValue :placeholder="$t('users.selectRole')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="role in rolesStore.roles" :key="role.id" :value="role.id">
                  <span class="capitalize">{{ role.name }}</span>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="isAddExistingOpen = false">{{ $t('common.cancel') }}</Button>
          <Button @click="submitAddExisting" :disabled="isAddExistingSubmitting || !addExistingEmail.trim()">
            <Loader2 v-if="isAddExistingSubmitting" class="h-4 w-4 mr-2 animate-spin" />
            {{ $t('users.addExistingUser') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
