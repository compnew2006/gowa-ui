<script setup lang="ts">
import { ref, onMounted, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  PageHeader,
  SearchInput,
  DataTable,
  CrudFormDialog,
  DeleteConfirmDialog,
  type Column,
} from "@/components/shared";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useUsersStore } from "@/stores/users";
import type { User } from "@/types/auth";
import { useAuthStore } from "@/stores/auth";
import { useRolesStore } from "@/stores/roles";
import { useOrganizationsStore } from "@/stores/organizations";
import { useInstancesStore } from "@/stores/instances";
import { authService, usersService, organizationService } from "@/services/api";
import { toast } from "vue-sonner";
import {
  Plus,
  Pencil,
  Trash2,
  UserMinus,
  User as UserIcon,
  Shield,
  ShieldCheck,
  UserCog,
  Users,
  Link,
  UserPlus,
  Loader2,
} from "lucide-vue-next";
import { useCrudState } from "@/composables/useCrudState";
import { getErrorMessage } from "@/lib/api-utils";
import { formatDate } from "@/lib/utils";
import { ROLE_BADGE_VARIANTS } from "@/lib/constants";
import { useDebounceFn } from "@vueuse/core";

const { t } = useI18n();

const usersStore = useUsersStore();
const authStore = useAuthStore();
const rolesStore = useRolesStore();
const organizationsStore = useOrganizationsStore();
const instancesStore = useInstancesStore();

interface UserFormData {
  email: string;
  password: string;
  full_name: string;
  role_id: string;
  is_active: boolean;
  is_super_admin: boolean;
}

const defaultFormData: UserFormData = {
  email: "",
  password: "",
  full_name: "",
  role_id: "",
  is_active: true,
  is_super_admin: false,
};

const {
  isLoading,
  isSubmitting,
  isDialogOpen,
  editingItem: editingUser,
  deleteDialogOpen,
  itemToDelete: userToDelete,
  formData,
  openCreateDialog: baseOpenCreateDialog,
  openEditDialog: baseOpenEditDialog,
  openDeleteDialog,
  closeDialog,
  closeDeleteDialog,
} = useCrudState<User, UserFormData>(defaultFormData);

const users = ref<User[]>([]);
const searchQuery = ref("");

// Pagination state
const currentPage = ref(1);
const totalItems = ref(0);
const pageSize = 20;

// Debounced search
const debouncedSearch = useDebounceFn(() => {
  currentPage.value = 1;
  fetchUsers();
}, 300);

watch(searchQuery, () => debouncedSearch());

function handlePageChange(page: number) {
  currentPage.value = page;
  fetchUsers();
}

const columns = computed<Column<User>[]>(() => [
  {
    key: "user",
    label: t("users.user"),
    width: "w-[300px]",
    sortable: true,
    sortKey: "full_name",
  },
  { key: "role", label: t("users.role"), sortable: true, sortKey: "role.name" },
  {
    key: "status",
    label: t("users.status"),
    sortable: true,
    sortKey: "is_active",
  },
  {
    key: "created",
    label: t("users.created"),
    sortable: true,
    sortKey: "created_at",
  },
  { key: "actions", label: t("common.actions"), align: "right" },
]);

// Sorting state
const sortKey = ref("full_name");
const sortDirection = ref<"asc" | "desc">("asc");

const currentUserId = computed(() => authStore.user?.id);
const isSuperAdmin = computed(() => authStore.user?.is_super_admin || false);
const canManageSendRestrictions = computed(() =>
  authStore.hasPermission("users", "write"),
);
const breadcrumbs = computed(() => [
  { label: t("nav.settings"), href: "/settings" },
  { label: t("nav.users") },
]);
const getDefaultRoleId = () =>
  rolesStore.roles.find((r) => r.name === "agent" && r.is_system)?.id || "";

function openCreateDialog() {
  formData.value.role_id = getDefaultRoleId();
  baseOpenCreateDialog();
}
function openEditDialog(user: User) {
  baseOpenEditDialog(user, (u) => ({
    email: u.email,
    password: "",
    full_name: u.full_name,
    role_id: u.role_id || "",
    is_active: u.is_active ?? true,
    is_super_admin: u.is_super_admin || false,
  }));
}

watch(
  () => organizationsStore.selectedOrgId,
  () => {
    fetchUsers();
    rolesStore.fetchRoles();
    instancesStore.fetchInstances();
    fetchStrictSendingRestrictions();
  },
);
onMounted(() => {
  fetchUsers();
  rolesStore.fetchRoles();
  instancesStore.fetchInstances();
  fetchStrictSendingRestrictions();
});

async function fetchUsers() {
  isLoading.value = true;
  try {
    const response = await usersStore.fetchUsers({
      search: searchQuery.value || undefined,
      page: currentPage.value,
      limit: pageSize,
    });
    users.value = response.users;
    totalItems.value = response.total;
  } catch {
    toast.error(t("common.failedLoad", { resource: t("resources.users") }));
  } finally {
    isLoading.value = false;
  }
}

async function saveUser() {
  if (!formData.value.email.trim() || !formData.value.full_name.trim()) {
    toast.error(t("users.fillEmailName"));
    return;
  }
  if (!editingUser.value && !formData.value.password.trim()) {
    toast.error(t("users.passwordRequired"));
    return;
  }
  if (!formData.value.role_id) {
    toast.error(t("users.selectRoleRequired"));
    return;
  }

  isSubmitting.value = true;
  try {
    const data: Record<string, unknown> = {
      email: formData.value.email,
      full_name: formData.value.full_name,
      role_id: formData.value.role_id,
    };
    if (editingUser.value) {
      data.is_active = formData.value.is_active;
      if (formData.value.password) data.password = formData.value.password;
      if (isSuperAdmin.value)
        data.is_super_admin = formData.value.is_super_admin;
      await usersStore.updateUser(editingUser.value.id, data);
      toast.success(
        t("common.updatedSuccess", { resource: t("resources.User") }),
      );
    } else {
      await usersStore.createUser({
        email: formData.value.email,
        password: formData.value.password,
        full_name: formData.value.full_name,
        role_id: formData.value.role_id || undefined,
        is_super_admin:
          isSuperAdmin.value && formData.value.is_super_admin
            ? true
            : undefined,
      });
      toast.success(
        t("common.createdSuccess", { resource: t("resources.User") }),
      );
    }
    closeDialog();
    await fetchUsers();
  } catch (e) {
    toast.error(
      getErrorMessage(
        e,
        t("common.failedSave", { resource: t("resources.user") }),
      ),
    );
  } finally {
    isSubmitting.value = false;
  }
}

async function confirmDelete() {
  if (!userToDelete.value) return;
  const isMemberRemoval = userToDelete.value.is_member;
  try {
    await usersStore.deleteUser(userToDelete.value.id);
    toast.success(
      isMemberRemoval
        ? t("users.memberRemoved")
        : t("common.deletedSuccess", { resource: t("resources.User") }),
    );
    closeDeleteDialog();
    await fetchUsers();
  } catch (e) {
    toast.error(
      getErrorMessage(
        e,
        t("common.failedDelete", { resource: t("resources.user") }),
      ),
    );
  }
}

// Member role update dialog
const isMemberRoleOpen = ref(false);
const memberRoleUser = ref<User | null>(null);
const memberRoleId = ref("");
const isMemberRoleSubmitting = ref(false);

function openMemberRoleDialog(user: User) {
  memberRoleUser.value = user;
  memberRoleId.value = user.role_id || "";
  isMemberRoleOpen.value = true;
}

async function submitMemberRole() {
  if (!memberRoleUser.value || !memberRoleId.value) return;
  isMemberRoleSubmitting.value = true;
  try {
    await usersStore.updateUser(memberRoleUser.value.id, {
      role_id: memberRoleId.value,
    });
    toast.success(t("users.memberRoleUpdated"));
    isMemberRoleOpen.value = false;
    await fetchUsers();
  } catch (e) {
    toast.error(
      getErrorMessage(
        e,
        t("common.failedSave", { resource: t("resources.user") }),
      ),
    );
  } finally {
    isMemberRoleSubmitting.value = false;
  }
}

function getRoleBadgeVariant(
  name: string,
): "default" | "secondary" | "outline" {
  return ROLE_BADGE_VARIANTS[name.toLowerCase()] || "outline";
}
function getRoleIcon(name: string) {
  return { admin: ShieldCheck, manager: Shield }[name.toLowerCase()] || UserCog;
}
function getRoleName(user: User) {
  return user.role?.name || t("users.noRole");
}

// Add existing user dialog
const isAddExistingOpen = ref(false);
const addExistingEmail = ref("");
const addExistingRoleId = ref("");
const isAddExistingSubmitting = ref(false);
const isInviteLinkLoading = ref(false);

function openAddExistingDialog() {
  addExistingEmail.value = "";
  addExistingRoleId.value = "";
  isAddExistingOpen.value = true;
}

async function submitAddExisting() {
  if (!addExistingEmail.value.trim()) {
    toast.error(t("users.enterEmail"));
    return;
  }
  isAddExistingSubmitting.value = true;
  try {
    await organizationsStore.addMember({
      email: addExistingEmail.value.trim(),
      role_id: addExistingRoleId.value || undefined,
    });
    toast.success(t("users.existingUserAdded"));
    isAddExistingOpen.value = false;
    await fetchUsers();
  } catch (e) {
    toast.error(getErrorMessage(e, t("users.addExistingFailed")));
  } finally {
    isAddExistingSubmitting.value = false;
  }
}

async function copyInviteLink() {
  const orgId = organizationsStore.selectedOrgId || authStore.organizationId;
  if (!orgId) {
    toast.error(t("auth.invitationRequired"));
    return;
  }

  isInviteLinkLoading.value = true;
  const basePath = ((window as any).__BASE_PATH__ ?? "").replace(/\/$/, "");
  try {
    const response = await authService.createRegisterInvite();
    const token = response.data.data?.token;
    if (!token) {
      throw new Error("missing invite token");
    }
    const url = `${window.location.origin}${basePath}/register?org=${orgId}&invite=${encodeURIComponent(token)}`;
    await navigator.clipboard.writeText(url);
    toast.success(t("users.inviteLinkCopied"));
  } catch (e) {
    toast.error(getErrorMessage(e, t("users.addExistingFailed")));
  } finally {
    isInviteLinkLoading.value = false;
  }
}

interface UserSendRestrictionsPayload {
  enabled: boolean;
  include_all_contacts: boolean;
  authorized_numbers: string[];
  allowed_instance_ids?: string[];
  allowed_instance_id?: string | null;
  prefix_agent_name?: boolean;
  allow_unclaimed_chat_view?: boolean;
  allow_unclaimed_chat_send?: boolean;
}

const isSendRestrictionsOpen = ref(false);
const sendRestrictionsUser = ref<User | null>(null);
const sendRestrictionsEnabled = ref(false);
const sendRestrictionsIncludeAllContacts = ref(false);
const sendRestrictionsNumbers = ref<string[]>([]);
const sendRestrictionsNewNumber = ref("");
const sendRestrictionsAllowedInstanceIDs = ref<string[]>([]);
const sendRestrictionsPrefixAgentName = ref(true);
const sendRestrictionsAllowUnclaimedChatView = ref(false);
const sendRestrictionsAllowUnclaimedChatSend = ref(false);
const isSendRestrictionsLoading = ref(false);
const isSendRestrictionsSubmitting = ref(false);
const sendRestrictionsAvailableInstances = computed(
  () => instancesStore.instances,
);

function normalizeAuthorizedNumber(raw: string): string {
  const noSpaces = raw.trim().replace(/\s+/g, "").replace(/^\+/, "");
  const digitsOnly = noSpaces.replace(/\D+/g, "");
  return digitsOnly;
}

function normalizeAllowedInstanceIDs(values: unknown): string[] {
  if (!Array.isArray(values)) return [];
  return Array.from(
    new Set(
      values
        .map((value) => (typeof value === "string" ? value.trim() : ""))
        .filter(Boolean),
    ),
  );
}

function setSendRestrictionsPayload(
  payload: UserSendRestrictionsPayload | undefined,
) {
  sendRestrictionsEnabled.value = payload?.enabled === true;
  sendRestrictionsIncludeAllContacts.value =
    payload?.include_all_contacts === true;
  sendRestrictionsNumbers.value = Array.from(
    new Set(
      (payload?.authorized_numbers || [])
        .map(normalizeAuthorizedNumber)
        .filter(Boolean),
    ),
  ).sort();
  const fromArray = normalizeAllowedInstanceIDs(payload?.allowed_instance_ids);
  if (fromArray.length > 0) {
    sendRestrictionsAllowedInstanceIDs.value = fromArray;
  } else {
    const legacy =
      typeof payload?.allowed_instance_id === "string"
        ? payload.allowed_instance_id.trim()
        : "";
    sendRestrictionsAllowedInstanceIDs.value = legacy ? [legacy] : [];
  }
  sendRestrictionsPrefixAgentName.value = payload?.prefix_agent_name !== false;
  sendRestrictionsAllowUnclaimedChatView.value =
    payload?.allow_unclaimed_chat_view === true;
  sendRestrictionsAllowUnclaimedChatSend.value =
    payload?.allow_unclaimed_chat_send === true;
  if (
    sendRestrictionsAllowUnclaimedChatSend.value &&
    !sendRestrictionsAllowUnclaimedChatView.value
  ) {
    sendRestrictionsAllowUnclaimedChatView.value = true;
  }
}

async function openSendRestrictionsDialog(user: User) {
  sendRestrictionsUser.value = user;
  isSendRestrictionsOpen.value = true;
  isSendRestrictionsLoading.value = true;
  sendRestrictionsNewNumber.value = "";
  if (sendRestrictionsAvailableInstances.value.length === 0) {
    await instancesStore.fetchInstances();
  }
  try {
    const response = await usersService.getSendRestrictions(user.id);
    const payload = (response.data as any).data || response.data;
    setSendRestrictionsPayload(payload);
  } catch (e) {
    setSendRestrictionsPayload(undefined);
    toast.error(getErrorMessage(e, t("users.sendRestrictionsLoadFailed")));
  } finally {
    isSendRestrictionsLoading.value = false;
  }
}

function addAuthorizedNumber() {
  const normalized = normalizeAuthorizedNumber(sendRestrictionsNewNumber.value);
  if (!normalized) {
    return;
  }
  if (!sendRestrictionsNumbers.value.includes(normalized)) {
    sendRestrictionsNumbers.value = [
      ...sendRestrictionsNumbers.value,
      normalized,
    ].sort();
  }
  sendRestrictionsNewNumber.value = "";
}

function removeAuthorizedNumber(number: string) {
  sendRestrictionsNumbers.value = sendRestrictionsNumbers.value.filter(
    (item) => item !== number,
  );
}

function toggleAllowedInstance(instanceID: string, checked: boolean) {
  const normalized = instanceID.trim();
  if (!normalized) return;
  if (checked) {
    if (!sendRestrictionsAllowedInstanceIDs.value.includes(normalized)) {
      sendRestrictionsAllowedInstanceIDs.value = [
        ...sendRestrictionsAllowedInstanceIDs.value,
        normalized,
      ];
    }
    return;
  }
  sendRestrictionsAllowedInstanceIDs.value =
    sendRestrictionsAllowedInstanceIDs.value.filter((id) => id !== normalized);
}

function updateAllowUnclaimedChatSend(checked: boolean) {
  sendRestrictionsAllowUnclaimedChatSend.value = checked;
  if (checked) {
    sendRestrictionsAllowUnclaimedChatView.value = true;
  }
}

async function saveSendRestrictions() {
  if (!sendRestrictionsUser.value) {
    return;
  }
  if (
    sendRestrictionsEnabled.value &&
    sendRestrictionsAllowedInstanceIDs.value.length === 0
  ) {
    toast.error(t("users.allowedInstanceRequired"));
    return;
  }

  const allowedInstanceIDs = normalizeAllowedInstanceIDs(
    sendRestrictionsAllowedInstanceIDs.value,
  );

  isSendRestrictionsSubmitting.value = true;
  try {
    const response = await usersService.updateSendRestrictions(
      sendRestrictionsUser.value.id,
      {
        enabled: sendRestrictionsEnabled.value,
        include_all_contacts: sendRestrictionsIncludeAllContacts.value,
        authorized_numbers: sendRestrictionsNumbers.value,
        allowed_instance_ids: allowedInstanceIDs,
        allowed_instance_id: allowedInstanceIDs[0] || null,
        prefix_agent_name: sendRestrictionsPrefixAgentName.value,
        allow_unclaimed_chat_view:
          sendRestrictionsAllowUnclaimedChatView.value ||
          sendRestrictionsAllowUnclaimedChatSend.value,
        allow_unclaimed_chat_send: sendRestrictionsAllowUnclaimedChatSend.value,
      },
    );
    const payload = (response.data as any).data || response.data;
    setSendRestrictionsPayload(payload);
    toast.success(t("users.sendRestrictionsSaved"));
    isSendRestrictionsOpen.value = false;
  } catch (e) {
    toast.error(getErrorMessage(e, t("users.sendRestrictionsSaveFailed")));
  } finally {
    isSendRestrictionsSubmitting.value = false;
  }
}

const strictSendingRestrictionsEnabled = ref(false);
const isStrictSendingRestrictionsLoading = ref(false);
const isStrictSendingRestrictionsSubmitting = ref(false);

async function fetchStrictSendingRestrictions() {
  isStrictSendingRestrictionsLoading.value = true;
  try {
    const response = await organizationService.getSettings();
    const payload = (response.data as any).data || response.data;
    strictSendingRestrictionsEnabled.value =
      payload?.settings?.strict_sending_restrictions_enabled === true;
  } catch {
    strictSendingRestrictionsEnabled.value = false;
  } finally {
    isStrictSendingRestrictionsLoading.value = false;
  }
}

async function saveStrictSendingRestrictions() {
  isStrictSendingRestrictionsSubmitting.value = true;
  try {
    await organizationService.updateSettings({
      strict_sending_restrictions_enabled:
        strictSendingRestrictionsEnabled.value,
    });
    toast.success(t("settings.generalSaved"));
  } catch (e) {
    toast.error(
      getErrorMessage(
        e,
        t("common.failedSave", { resource: t("resources.settings") }),
      ),
    );
  } finally {
    isStrictSendingRestrictionsSubmitting.value = false;
  }
}
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="$t('users.title')"
      :icon="Users"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
      back-link="/settings"
      :breadcrumbs="breadcrumbs"
    >
      <template #actions>
        <Button
          variant="outline"
          size="sm"
          @click="copyInviteLink"
          :disabled="isInviteLinkLoading"
        >
          <Loader2
            v-if="isInviteLinkLoading"
            class="h-4 w-4 mr-2 animate-spin"
          />
          <Link v-else class="h-4 w-4 mr-2" />
          {{ $t("users.copyInviteLink") }}
        </Button>
        <Button
          v-if="
            organizationsStore.isMultiOrg &&
            authStore.hasPermission('organizations', 'assign')
          "
          variant="outline"
          size="sm"
          @click="openAddExistingDialog"
          ><UserPlus class="h-4 w-4 mr-2" />{{
            $t("users.addExistingUser")
          }}</Button
        >
        <Button variant="outline" size="sm" @click="openCreateDialog"
          ><Plus class="h-4 w-4 mr-2" />{{ $t("users.addUser") }}</Button
        >
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="p-6">
        <div class="max-w-6xl mx-auto">
          <Card class="mb-6">
            <CardHeader>
              <CardTitle>{{
                $t("settings.strictSendingRestrictions")
              }}</CardTitle>
              <CardDescription>{{
                $t("settings.strictSendingRestrictionsDesc")
              }}</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="flex items-center justify-between gap-4">
                <div class="text-sm text-muted-foreground">
                  {{ $t("users.manageSendRestrictions") }}
                </div>
                <div class="flex items-center gap-2">
                  <Switch
                    :checked="strictSendingRestrictionsEnabled"
                    :disabled="
                      isStrictSendingRestrictionsLoading ||
                      isStrictSendingRestrictionsSubmitting
                    "
                    @update:checked="strictSendingRestrictionsEnabled = $event"
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    :disabled="
                      isStrictSendingRestrictionsLoading ||
                      isStrictSendingRestrictionsSubmitting
                    "
                    @click="saveStrictSendingRestrictions"
                  >
                    <Loader2
                      v-if="isStrictSendingRestrictionsSubmitting"
                      class="h-4 w-4 mr-2 animate-spin"
                    />
                    {{ $t("common.save") }}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div class="flex items-center justify-between flex-wrap gap-4">
                <div>
                  <CardTitle>{{ $t("users.yourUsers") }}</CardTitle>
                  <CardDescription
                    >{{ $t("users.subtitle") }}.
                    <RouterLink
                      to="/settings/roles"
                      class="text-primary hover:underline"
                      >{{ $t("users.manageRoles") }}</RouterLink
                    ></CardDescription
                  >
                </div>
                <SearchInput
                  v-model="searchQuery"
                  :placeholder="$t('users.searchUsers') + '...'"
                  class="w-64"
                />
              </div>
            </CardHeader>
            <CardContent>
              <DataTable
                :items="users"
                :columns="columns"
                :is-loading="isLoading"
                :empty-icon="UserIcon"
                :empty-title="
                  searchQuery
                    ? $t('users.noMatchingUsers')
                    : $t('users.noUsersFound')
                "
                :empty-description="
                  searchQuery
                    ? $t('users.noMatchingUsersDesc')
                    : $t('users.noUsersFoundDesc')
                "
                v-model:sort-key="sortKey"
                v-model:sort-direction="sortDirection"
                server-pagination
                :current-page="currentPage"
                :total-items="totalItems"
                :page-size="pageSize"
                item-name="users"
                @page-change="handlePageChange"
              >
                <template #cell-user="{ item: user }">
                  <div class="flex items-center gap-3">
                    <div
                      class="h-9 w-9 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0"
                    >
                      <component
                        :is="getRoleIcon(getRoleName(user))"
                        class="h-4 w-4 text-primary"
                      />
                    </div>
                    <div class="min-w-0">
                      <div class="flex items-center gap-2">
                        <p class="font-medium truncate">{{ user.full_name }}</p>
                        <Badge
                          v-if="user.id === currentUserId"
                          variant="outline"
                          class="text-xs"
                          >{{ $t("users.you") }}</Badge
                        >
                        <Badge
                          v-if="user.is_super_admin"
                          variant="default"
                          class="text-xs"
                          >{{ $t("users.superAdmin") }}</Badge
                        >
                        <Badge
                          v-if="user.is_member"
                          variant="secondary"
                          class="text-xs"
                          >{{ $t("users.member") }}</Badge
                        >
                      </div>
                      <p class="text-sm text-muted-foreground truncate">
                        {{ user.email }}
                      </p>
                    </div>
                  </div>
                </template>
                <template #cell-role="{ item: user }">
                  <Badge
                    :variant="getRoleBadgeVariant(getRoleName(user))"
                    class="capitalize"
                    >{{ getRoleName(user) }}</Badge
                  >
                </template>
                <template #cell-status="{ item: user }">
                  <Badge
                    variant="outline"
                    :class="
                      user.is_active ? 'border-primary/40 text-primary' : ''
                    "
                    >{{
                      user.is_active
                        ? $t("common.active")
                        : $t("common.inactive")
                    }}</Badge
                  >
                </template>
                <template #cell-created="{ item: user }">
                  <span class="text-muted-foreground">{{
                    formatDate(user.created_at || "")
                  }}</span>
                </template>
                <template #cell-actions="{ item: user }">
                  <div class="flex items-center justify-end gap-1">
                    <Tooltip v-if="canManageSendRestrictions">
                      <TooltipTrigger as-child>
                        <Button
                          variant="ghost"
                          size="icon"
                          class="h-8 w-8"
                          @click="openSendRestrictionsDialog(user)"
                        >
                          <Shield class="h-4 w-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{{
                        $t("users.manageSendRestrictions")
                      }}</TooltipContent>
                    </Tooltip>
                    <template v-if="user.is_member">
                      <!-- Member actions: update role + remove -->
                      <Tooltip
                        ><TooltipTrigger as-child
                          ><Button
                            variant="ghost"
                            size="icon"
                            class="h-8 w-8"
                            @click="openMemberRoleDialog(user)"
                            ><Pencil class="h-4 w-4" /></Button></TooltipTrigger
                        ><TooltipContent>{{
                          $t("users.updateMemberRole")
                        }}</TooltipContent></Tooltip
                      >
                      <Tooltip
                        ><TooltipTrigger as-child
                          ><Button
                            variant="ghost"
                            size="icon"
                            class="h-8 w-8"
                            @click="openDeleteDialog(user)"
                            :disabled="user.id === currentUserId"
                            ><UserMinus
                              class="h-4 w-4 text-destructive" /></Button></TooltipTrigger
                        ><TooltipContent>{{
                          $t("users.removeMemberTooltip")
                        }}</TooltipContent></Tooltip
                      >
                    </template>
                    <template v-else>
                      <!-- Native user actions: full edit + delete -->
                      <Tooltip
                        ><TooltipTrigger as-child
                          ><Button
                            variant="ghost"
                            size="icon"
                            class="h-8 w-8"
                            @click="openEditDialog(user)"
                            ><Pencil class="h-4 w-4" /></Button></TooltipTrigger
                        ><TooltipContent>{{
                          $t("users.editUserTooltip")
                        }}</TooltipContent></Tooltip
                      >
                      <Tooltip
                        ><TooltipTrigger as-child
                          ><Button
                            variant="ghost"
                            size="icon"
                            class="h-8 w-8"
                            @click="openDeleteDialog(user)"
                            :disabled="user.id === currentUserId"
                            ><Trash2
                              class="h-4 w-4 text-destructive" /></Button></TooltipTrigger
                        ><TooltipContent>{{
                          user.id === currentUserId
                            ? $t("users.cantDeleteYourself")
                            : $t("users.deleteUserTooltip")
                        }}</TooltipContent></Tooltip
                      >
                    </template>
                  </div>
                </template>
                <template #empty-action>
                  <Button variant="outline" size="sm" @click="openCreateDialog"
                    ><Plus class="h-4 w-4 mr-2" />{{
                      $t("users.addUser")
                    }}</Button
                  >
                </template>
              </DataTable>
            </CardContent>
          </Card>
        </div>
      </div>
    </ScrollArea>

    <CrudFormDialog
      v-model:open="isDialogOpen"
      :is-editing="!!editingUser"
      :is-submitting="isSubmitting"
      :edit-title="$t('users.editUserTitle')"
      :create-title="$t('users.addUserTitle')"
      :edit-description="$t('users.editUserDesc')"
      :create-description="$t('users.addUserDesc')"
      :edit-submit-label="$t('users.updateUser')"
      :create-submit-label="$t('users.createUser')"
      @submit="saveUser"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <Label for="full_name"
            >{{ $t("users.fullName") }}
            <span class="text-destructive">*</span></Label
          ><Input
            id="full_name"
            v-model="formData.full_name"
            :placeholder="$t('users.fullNamePlaceholder')"
          />
        </div>
        <div class="space-y-2">
          <Label for="email"
            >{{ $t("common.email") }}
            <span class="text-destructive">*</span></Label
          ><Input
            id="email"
            v-model="formData.email"
            type="email"
            :placeholder="$t('users.emailPlaceholder')"
          />
        </div>
        <div class="space-y-2">
          <Label for="password"
            >{{ $t("users.password") }}
            <span v-if="!editingUser" class="text-destructive">*</span
            ><span v-else class="text-muted-foreground">{{
              $t("users.keepExisting")
            }}</span></Label
          ><Input
            id="password"
            v-model="formData.password"
            type="password"
            :placeholder="$t('users.passwordPlaceholder')"
          />
        </div>
        <div class="space-y-2">
          <Label for="role"
            >{{ $t("users.role") }}
            <span class="text-destructive">*</span></Label
          >
          <Select v-model="formData.role_id">
            <SelectTrigger id="role">
              <SelectValue :placeholder="$t('users.selectRole')">
                <template v-if="formData.role_id">
                  <span class="capitalize">{{
                    rolesStore.roles.find((r) => r.id === formData.role_id)
                      ?.name
                  }}</span>
                  <Badge
                    v-if="
                      rolesStore.roles.find((r) => r.id === formData.role_id)
                        ?.is_system
                    "
                    variant="secondary"
                    class="text-xs ml-2"
                    >{{ $t("users.system") }}</Badge
                  >
                </template>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="role in rolesStore.roles"
                :key="role.id"
                :value="role.id"
              >
                <div class="flex items-center gap-2">
                  <span class="capitalize">{{ role.name }}</span>
                  <Badge
                    v-if="role.is_system"
                    variant="secondary"
                    class="text-xs"
                    >{{ $t("users.system") }}</Badge
                  >
                </div>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="editingUser" class="flex items-center justify-between">
          <Label for="is_active" class="font-normal cursor-pointer">{{
            $t("users.accountActive")
          }}</Label
          ><Switch
            id="is_active"
            :checked="formData.is_active"
            @update:checked="formData.is_active = $event"
            :disabled="editingUser?.id === currentUserId"
          />
        </div>
        <div
          v-if="isSuperAdmin"
          class="flex items-center justify-between border-t pt-4"
        >
          <div>
            <Label for="is_super_admin" class="font-normal cursor-pointer">{{
              $t("users.superAdminLabel")
            }}</Label>
            <p class="text-xs text-muted-foreground">
              {{ $t("users.superAdminDesc") }}
            </p>
          </div>
          <Switch
            id="is_super_admin"
            :checked="formData.is_super_admin"
            @update:checked="formData.is_super_admin = $event"
            :disabled="
              editingUser?.id === currentUserId && editingUser?.is_super_admin
            "
          />
        </div>
      </div>
    </CrudFormDialog>

    <DeleteConfirmDialog
      v-model:open="deleteDialogOpen"
      :title="
        userToDelete?.is_member
          ? $t('users.removeMember')
          : $t('users.deleteUser')
      "
      :description="
        userToDelete?.is_member ? $t('users.removeMemberWarning') : undefined
      "
      :item-name="userToDelete?.full_name"
      @confirm="confirmDelete"
    />

    <!-- Member Role Update Dialog -->
    <Dialog v-model:open="isMemberRoleOpen">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t("users.updateMemberRoleTitle") }}</DialogTitle>
          <DialogDescription>{{
            $t("users.updateMemberRoleDesc")
          }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-4">
          <div class="flex items-center gap-3 p-3 rounded-lg bg-muted/50">
            <div
              class="h-9 w-9 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0"
            >
              <UserIcon class="h-4 w-4 text-primary" />
            </div>
            <div class="min-w-0">
              <p class="font-medium truncate">
                {{ memberRoleUser?.full_name }}
              </p>
              <p class="text-sm text-muted-foreground truncate">
                {{ memberRoleUser?.email }}
              </p>
            </div>
          </div>
          <div class="space-y-2">
            <Label
              >{{ $t("users.role") }}
              <span class="text-destructive">*</span></Label
            >
            <Select v-model="memberRoleId">
              <SelectTrigger>
                <SelectValue :placeholder="$t('users.selectRole')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="role in rolesStore.roles"
                  :key="role.id"
                  :value="role.id"
                >
                  <div class="flex items-center gap-2">
                    <span class="capitalize">{{ role.name }}</span>
                    <Badge
                      v-if="role.is_system"
                      variant="secondary"
                      class="text-xs"
                      >{{ $t("users.system") }}</Badge
                    >
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="isMemberRoleOpen = false">{{
            $t("common.cancel")
          }}</Button>
          <Button
            @click="submitMemberRole"
            :disabled="isMemberRoleSubmitting || !memberRoleId"
          >
            <Loader2
              v-if="isMemberRoleSubmitting"
              class="h-4 w-4 mr-2 animate-spin"
            />
            {{ $t("users.updateMemberRole") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Add Existing User Dialog -->
    <Dialog v-model:open="isAddExistingOpen">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t("users.addExistingUserTitle") }}</DialogTitle>
          <DialogDescription>{{
            $t("users.addExistingUserDesc")
          }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-4">
          <div class="space-y-2">
            <Label for="existing-email"
              >{{ $t("common.email") }}
              <span class="text-destructive">*</span></Label
            >
            <Input
              id="existing-email"
              v-model="addExistingEmail"
              type="email"
              :placeholder="$t('users.existingEmailPlaceholder')"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t("users.role") }}</Label>
            <Select v-model="addExistingRoleId">
              <SelectTrigger>
                <SelectValue :placeholder="$t('users.selectRole')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="role in rolesStore.roles"
                  :key="role.id"
                  :value="role.id"
                >
                  <span class="capitalize">{{ role.name }}</span>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="isAddExistingOpen = false">{{
            $t("common.cancel")
          }}</Button>
          <Button
            @click="submitAddExisting"
            :disabled="isAddExistingSubmitting || !addExistingEmail.trim()"
          >
            <Loader2
              v-if="isAddExistingSubmitting"
              class="h-4 w-4 mr-2 animate-spin"
            />
            {{ $t("users.addExistingUser") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="isSendRestrictionsOpen">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t("users.sendRestrictionsTitle") }}</DialogTitle>
          <DialogDescription>
            {{
              $t("users.sendRestrictionsDesc", {
                user: sendRestrictionsUser?.full_name || "",
              })
            }}
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-4 py-2">
          <div class="flex items-center justify-between">
            <div>
              <Label class="font-normal">{{
                $t("users.sendRestrictionsEnabled")
              }}</Label>
              <p class="text-xs text-muted-foreground">
                {{ $t("users.sendRestrictionsEnabledDesc") }}
              </p>
            </div>
            <Switch
              :checked="sendRestrictionsEnabled"
              @update:checked="sendRestrictionsEnabled = $event"
              :disabled="isSendRestrictionsLoading"
            />
          </div>

          <div class="space-y-2">
            <Label>{{ $t("users.allowedInstance") }}</Label>
            <div class="rounded-md border p-3 max-h-48 overflow-y-auto">
              <div
                v-if="isSendRestrictionsLoading"
                class="text-sm text-muted-foreground"
              >
                {{ $t("common.loading") }}...
              </div>
              <div
                v-else-if="sendRestrictionsAvailableInstances.length === 0"
                class="text-sm text-muted-foreground"
              >
                {{ $t("users.allowedInstancePlaceholder") }}
              </div>
              <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <label
                  v-for="instance in sendRestrictionsAvailableInstances"
                  :key="instance.id"
                  :for="`allowed-instance-${instance.id}`"
                  class="flex items-start gap-3 rounded-md px-2 py-1.5 hover:bg-muted/50 cursor-pointer min-w-0"
                >
                  <span class="shrink-0 pt-0.5">
                    <Checkbox
                      :id="`allowed-instance-${instance.id}`"
                      :checked="
                        sendRestrictionsAllowedInstanceIDs.includes(instance.id)
                      "
                      :disabled="isSendRestrictionsLoading"
                      @update:checked="
                        toggleAllowedInstance(instance.id, $event === true)
                      "
                    />
                  </span>
                  <span
                    class="text-sm leading-snug break-words whitespace-normal min-w-0"
                  >
                    {{ instance.name }}
                    <span v-if="instance.phone_number"
                      >({{ instance.phone_number }})</span
                    >
                  </span>
                </label>
              </div>
            </div>
            <div class="flex items-center justify-between gap-2">
              <p class="text-xs text-muted-foreground">
                {{ $t("users.allowedInstanceDesc") }}
              </p>
              <Button
                type="button"
                variant="outline"
                size="sm"
                :disabled="
                  isSendRestrictionsLoading ||
                  sendRestrictionsAllowedInstanceIDs.length === 0
                "
                @click="sendRestrictionsAllowedInstanceIDs = []"
              >
                {{ $t("common.clear") }}
              </Button>
            </div>
          </div>

          <div class="flex items-center justify-between">
            <div>
              <Label class="font-normal">{{
                $t("users.includeAllContacts")
              }}</Label>
              <p class="text-xs text-muted-foreground">
                {{ $t("users.includeAllContactsDesc") }}
              </p>
            </div>
            <Switch
              :checked="sendRestrictionsIncludeAllContacts"
              @update:checked="sendRestrictionsIncludeAllContacts = $event"
              :disabled="isSendRestrictionsLoading"
            />
          </div>

          <div class="flex items-center justify-between">
            <div>
              <Label class="font-normal">{{
                $t("users.prefixAgentName")
              }}</Label>
              <p class="text-xs text-muted-foreground">
                {{ $t("users.prefixAgentNameDesc") }}
              </p>
            </div>
            <Switch
              :checked="sendRestrictionsPrefixAgentName"
              @update:checked="sendRestrictionsPrefixAgentName = $event"
              :disabled="isSendRestrictionsLoading"
            />
          </div>

          <div class="flex items-center justify-between">
            <div>
              <Label class="font-normal">Allow Unclaimed Chat View</Label>
              <p class="text-xs text-muted-foreground">
                Allow viewing pending/unassigned chats without claiming first.
              </p>
            </div>
            <Switch
              :checked="sendRestrictionsAllowUnclaimedChatView"
              :disabled="isSendRestrictionsLoading"
              @update:checked="sendRestrictionsAllowUnclaimedChatView = $event"
            />
          </div>

          <div class="flex items-center justify-between">
            <div>
              <Label class="font-normal">Allow Unclaimed Chat Send</Label>
              <p class="text-xs text-muted-foreground">
                Allow sending messages in pending/unassigned chats without
                claiming first.
              </p>
            </div>
            <Switch
              :checked="sendRestrictionsAllowUnclaimedChatSend"
              :disabled="isSendRestrictionsLoading"
              @update:checked="updateAllowUnclaimedChatSend"
            />
          </div>

          <div class="space-y-2">
            <Label>{{ $t("users.authorizedNumbers") }}</Label>
            <div class="flex gap-2">
              <Input
                v-model="sendRestrictionsNewNumber"
                :placeholder="$t('users.authorizedNumberPlaceholder')"
                :disabled="isSendRestrictionsLoading"
                @keydown.enter.prevent="addAuthorizedNumber"
              />
              <Button
                type="button"
                variant="outline"
                @click="addAuthorizedNumber"
                :disabled="
                  isSendRestrictionsLoading || !sendRestrictionsNewNumber.trim()
                "
              >
                {{ $t("common.add") }}
              </Button>
            </div>
            <div class="rounded-md border p-3 min-h-[56px] space-y-2">
              <div
                v-if="isSendRestrictionsLoading"
                class="text-sm text-muted-foreground"
              >
                {{ $t("common.loading") }}...
              </div>
              <div
                v-else-if="sendRestrictionsNumbers.length === 0"
                class="text-sm text-muted-foreground"
              >
                {{ $t("users.noAuthorizedNumbers") }}
              </div>
              <div v-else class="flex flex-wrap gap-2">
                <Badge
                  v-for="number in sendRestrictionsNumbers"
                  :key="number"
                  variant="secondary"
                  class="pr-1"
                >
                  <span>{{ number }}</span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    class="h-5 w-5 ml-1"
                    @click="removeAuthorizedNumber(number)"
                    :disabled="isSendRestrictionsLoading"
                  >
                    <Trash2 class="h-3 w-3" />
                  </Button>
                </Badge>
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="isSendRestrictionsOpen = false">{{
            $t("common.cancel")
          }}</Button>
          <Button
            @click="saveSendRestrictions"
            :disabled="
              isSendRestrictionsSubmitting || isSendRestrictionsLoading
            "
          >
            <Loader2
              v-if="isSendRestrictionsSubmitting"
              class="h-4 w-4 mr-2 animate-spin"
            />
            {{ $t("common.save") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
