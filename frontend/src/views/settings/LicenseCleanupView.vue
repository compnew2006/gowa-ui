<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue-sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  accountsService,
  organizationsService,
  usersService,
  instancesService,
  type Organization,
} from "@/services/api";
import { getErrorMessage, unwrapResponse } from "@/lib/api-utils";
import { useAuthStore } from "@/stores/auth";
import { useLicenseStore } from "@/stores/license";
import { useOrganizationsStore } from "@/stores/organizations";
import type { User } from "@/types/auth";
import type { WhatsAppInstance } from "@/types/whatsmeow";
import {
  AlertTriangle,
  Building2,
  Loader2,
  ShieldAlert,
  Smartphone,
  Trash2,
  Users,
} from "lucide-vue-next";

interface WhatsAppAccount {
  id: string;
  name: string;
  status: string;
  display_name?: string;
  phone_number?: string;
}

const router = useRouter();
const authStore = useAuthStore();
const licenseStore = useLicenseStore();
const organizationsStore = useOrganizationsStore();

const organizations = ref<Organization[]>([]);
const users = ref<User[]>([]);
const accounts = ref<WhatsAppAccount[]>([]);
const instances = ref<WhatsAppInstance[]>([]);
const loading = ref(true);
const reloading = ref(false);
const deletingKey = ref("");

const isSuperAdmin = computed(() => authStore.user?.is_super_admin === true);
const canDeleteUsers = computed(() =>
  authStore.hasPermission("users", "delete"),
);
const canDeleteAccounts = computed(() =>
  authStore.hasPermission("accounts", "delete"),
);
const canDeleteOrganizations = computed(
  () =>
    isSuperAdmin.value && authStore.hasPermission("organizations", "delete"),
);
const currentOrgID = computed(
  () =>
    organizationsStore.selectedOrgId || authStore.user?.organization_id || "",
);
const currentOrgUsage = computed(() =>
  licenseStore.state.usage.organization_details.find(
    (org) => org.organization_id === currentOrgID.value,
  ),
);
const overageItems = computed(() =>
  [
    {
      key: "organizations",
      label: "Organizations",
      value: licenseStore.state.quota_overages.organizations || 0,
    },
    {
      key: "users",
      label: "Users / Org",
      value: licenseStore.state.quota_overages.users || 0,
    },
    {
      key: "whatsapp_endpoints",
      label: "WA Endpoints / Org",
      value: licenseStore.state.quota_overages.whatsapp_endpoints || 0,
    },
    {
      key: "storage_bytes",
      label: "Storage / Org",
      value: licenseStore.state.quota_overages.storage_bytes || 0,
    },
  ].filter((item) => item.value > 0),
);

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let unitIndex = 0;
  for (; size >= 1024 && unitIndex < units.length - 1; unitIndex += 1) {
    size /= 1024;
  }

  const fractionDigits = size >= 100 || unitIndex === 0 ? 0 : 1;
  return `${size.toFixed(fractionDigits)} ${units[unitIndex]}`;
}

async function redirectToAppRoot() {
  const target = router.resolve({ path: "/" }).href || "/";
  window.location.assign(target);
}

async function refreshLicenseState() {
  await licenseStore.fetchBootstrap(true);
  if (!licenseStore.showQuotaOverage) {
    toast.success("License usage is back within quota.");
    await redirectToAppRoot();
  }
}

async function loadOrganizations() {
  if (isSuperAdmin.value) {
    organizationsStore.init();
    await organizationsStore.fetchOrganizations();
    organizations.value = [...organizationsStore.organizations];
    return;
  }

  await organizationsStore.fetchMyOrganizations();
  organizations.value = organizationsStore.myOrganizations.map((org) => ({
    id: org.organization_id,
    name: org.name,
    slug: org.slug,
    created_at: "",
  }));
}

async function loadUsers() {
  const response = await usersService.list({ page: 1, limit: 200 });
  const payload = unwrapResponse<{ users: User[] }>(response);
  users.value = payload.users || [];
}

async function loadAccounts() {
  const response = await accountsService.list();
  accounts.value = response.data?.data?.accounts || [];
}

async function loadInstances() {
  const response = await instancesService.list();
  instances.value = unwrapResponse<WhatsAppInstance[]>(response);
}

async function loadScopedResources() {
  await Promise.allSettled([loadUsers(), loadAccounts(), loadInstances()]);
}

async function loadCleanupData(forceBootstrap = false) {
  const target = forceBootstrap ? reloading : loading;
  target.value = true;
  try {
    await licenseStore.fetchBootstrap(forceBootstrap);
    if (!licenseStore.showQuotaOverage) {
      await redirectToAppRoot();
      return;
    }
    await Promise.allSettled([loadOrganizations(), loadScopedResources()]);
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to load cleanup data"));
  } finally {
    target.value = false;
  }
}

async function afterDeletion(successMessage: string) {
  toast.success(successMessage);
  await Promise.allSettled([loadOrganizations(), loadScopedResources()]);
  await refreshLicenseState();
}

async function deleteUser(user: User) {
  const confirmed = window.confirm(`Delete user "${user.full_name}"?`);
  if (!confirmed) {
    return;
  }

  deletingKey.value = `user:${user.id}`;
  try {
    await usersService.delete(user.id);
    await afterDeletion(`Deleted ${user.full_name}.`);
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to delete user"));
  } finally {
    deletingKey.value = "";
  }
}

async function deleteAccount(account: WhatsAppAccount) {
  const confirmed = window.confirm(
    `Delete WhatsApp account "${account.name}"?`,
  );
  if (!confirmed) {
    return;
  }

  deletingKey.value = `account:${account.id}`;
  try {
    await accountsService.delete(account.id);
    await afterDeletion(`Deleted account ${account.name}.`);
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to delete account"));
  } finally {
    deletingKey.value = "";
  }
}

async function deleteInstance(instance: WhatsAppInstance) {
  const confirmed = window.confirm(`Delete instance "${instance.name}"?`);
  if (!confirmed) {
    return;
  }

  deletingKey.value = `instance:${instance.id}`;
  try {
    await instancesService.delete(instance.id);
    await afterDeletion(`Deleted instance ${instance.name}.`);
  } catch (error) {
    toast.error(getErrorMessage(error, "Failed to delete instance"));
  } finally {
    deletingKey.value = "";
  }
}

async function deleteOrganization(org: Organization) {
  const confirmed = window.confirm(`Delete organization "${org.name}"?`);
  if (!confirmed) {
    return;
  }

  const fallbackOrg =
    organizations.value.find((candidate) => candidate.id !== org.id) || null;
  const deletedSelectedOrg = organizationsStore.selectedOrgId === org.id;

  deletingKey.value = `org:${org.id}`;
  try {
    if (deletedSelectedOrg) {
      organizationsStore.selectOrganization(fallbackOrg?.id || null);
    }

    await organizationsService.delete(org.id);
    await organizationsStore.fetchOrganizations();
    organizations.value = [...organizationsStore.organizations];

    if (
      organizationsStore.selectedOrgId &&
      !organizationsStore.organizations.some(
        (candidate) => candidate.id === organizationsStore.selectedOrgId,
      )
    ) {
      organizationsStore.selectOrganization(
        organizationsStore.organizations[0]?.id || null,
      );
    }

    await afterDeletion(`Deleted organization ${org.name}.`);
  } catch (error) {
    if (deletedSelectedOrg) {
      organizationsStore.selectOrganization(org.id);
    }
    toast.error(getErrorMessage(error, "Failed to delete organization"));
  } finally {
    deletingKey.value = "";
  }
}

watch(
  () => organizationsStore.selectedOrgId,
  () => {
    if (!licenseStore.showQuotaOverage) {
      return;
    }
    void Promise.allSettled([loadScopedResources(), refreshLicenseState()]);
  },
);

onMounted(() => {
  void loadCleanupData();
});
</script>

<template>
  <div class="h-full overflow-y-auto bg-background">
    <div class="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-6 md:px-6">
      <Card
        class="border-amber-500/35 bg-amber-50/70 shadow-sm dark:bg-amber-950/20"
      >
        <CardHeader class="space-y-3">
          <div class="flex items-start justify-between gap-4">
            <div class="space-y-2">
              <div
                class="flex items-center gap-2 text-amber-700 dark:text-amber-300"
              >
                <ShieldAlert class="h-5 w-5" />
                <span class="text-sm font-semibold uppercase tracking-[0.14em]">
                  Cleanup required
                </span>
              </div>
              <CardTitle class="text-2xl">
                License quota overage is blocking normal app usage
              </CardTitle>
              <p class="max-w-3xl text-sm leading-6 text-muted-foreground">
                Delete enough organizations, users, or WhatsApp endpoints to get
                back under the licensed quota. All other app areas stay blocked
                until the overage is resolved.
              </p>
            </div>
            <Button
              variant="outline"
              :disabled="reloading"
              @click="loadCleanupData(true)"
            >
              <Loader2 v-if="reloading" class="mr-2 h-4 w-4 animate-spin" />
              Refresh status
            </Button>
          </div>

          <div class="flex flex-wrap gap-2">
            <Badge
              v-for="item in overageItems"
              :key="item.key"
              variant="warning"
              class="px-3 py-1"
            >
              {{ item.label }}: over by
              {{
                item.key === "storage_bytes"
                  ? formatBytes(item.value)
                  : item.value
              }}
            </Badge>
            <Badge v-if="licenseStore.state.tier" variant="outline">
              {{ licenseStore.state.tier }}
            </Badge>
            <Badge v-if="licenseStore.state.license_kind" variant="outline">
              {{
                licenseStore.state.duration_label
                  ? `${licenseStore.state.license_kind} • ${licenseStore.state.duration_label}`
                  : licenseStore.state.license_kind
              }}
            </Badge>
          </div>
        </CardHeader>
        <CardContent class="grid gap-4 lg:grid-cols-4">
          <div class="rounded-xl border border-border/60 bg-background/80 p-4">
            <div
              class="text-xs uppercase tracking-[0.14em] text-muted-foreground"
            >
              Organizations
            </div>
            <div class="mt-2 text-3xl font-semibold">
              {{ licenseStore.state.usage.organizations.current }}/{{
                licenseStore.state.usage.organizations.limit || 0
              }}
            </div>
          </div>
          <div class="rounded-xl border border-border/60 bg-background/80 p-4">
            <div
              class="text-xs uppercase tracking-[0.14em] text-muted-foreground"
            >
              Users in selected org
            </div>
            <div class="mt-2 text-3xl font-semibold">
              {{ currentOrgUsage?.user_count || 0 }}/{{
                licenseStore.state.max_users_per_org || 0
              }}
            </div>
          </div>
          <div class="rounded-xl border border-border/60 bg-background/80 p-4">
            <div
              class="text-xs uppercase tracking-[0.14em] text-muted-foreground"
            >
              WA endpoints in selected org
            </div>
            <div class="mt-2 text-3xl font-semibold">
              {{ currentOrgUsage?.whatsapp_endpoint_count || 0 }}/{{
                licenseStore.state.max_whatsapp_endpoints_per_org || 0
              }}
            </div>
          </div>
          <div class="rounded-xl border border-border/60 bg-background/80 p-4">
            <div
              class="text-xs uppercase tracking-[0.14em] text-muted-foreground"
            >
              Storage in selected org
            </div>
            <div class="mt-2 text-3xl font-semibold">
              {{ formatBytes(currentOrgUsage?.storage_bytes || 0) }}/{{
                licenseStore.state.max_storage_bytes_per_org > 0
                  ? formatBytes(licenseStore.state.max_storage_bytes_per_org)
                  : "Not enforced"
              }}
            </div>
          </div>
        </CardContent>
      </Card>

      <div
        v-if="!canDeleteOrganizations && !canDeleteUsers && !canDeleteAccounts"
        class="flex items-start gap-3 rounded-xl border border-border/70 bg-card px-4 py-3 text-sm text-muted-foreground"
      >
        <AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
        <p>
          This account can view the overage status but does not have delete
          permissions to resolve it. Sign in with an administrator account.
        </p>
      </div>

      <div class="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2 text-lg">
              <Building2 class="h-4 w-4" />
              Organizations
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea class="h-[320px] pr-4">
              <div class="space-y-3">
                <div
                  v-for="org in organizations"
                  :key="org.id"
                  class="flex items-center justify-between gap-3 rounded-xl border border-border/70 px-4 py-3"
                >
                  <div class="min-w-0">
                    <div class="font-medium">{{ org.name }}</div>
                    <div class="text-xs text-muted-foreground">
                      {{ org.slug || org.id }}
                    </div>
                  </div>
                  <Button
                    v-if="canDeleteOrganizations"
                    variant="destructive"
                    size="sm"
                    :disabled="deletingKey === `org:${org.id}`"
                    @click="deleteOrganization(org)"
                  >
                    <Loader2
                      v-if="deletingKey === `org:${org.id}`"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    <Trash2 v-else class="mr-2 h-4 w-4" />
                    Delete
                  </Button>
                </div>
                <div
                  v-if="organizations.length === 0"
                  class="rounded-xl border border-dashed border-border px-4 py-6 text-sm text-muted-foreground"
                >
                  No organizations loaded.
                </div>
              </div>
            </ScrollArea>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2 text-lg">
              <Users class="h-4 w-4" />
              Users in selected organization
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea class="h-[320px] pr-4">
              <div class="space-y-3">
                <div
                  v-for="user in users"
                  :key="user.id"
                  class="flex items-center justify-between gap-3 rounded-xl border border-border/70 px-4 py-3"
                >
                  <div class="min-w-0">
                    <div class="font-medium">{{ user.full_name }}</div>
                    <div class="text-xs text-muted-foreground">
                      {{ user.email }}
                    </div>
                  </div>
                  <Button
                    v-if="canDeleteUsers"
                    variant="destructive"
                    size="sm"
                    :disabled="deletingKey === `user:${user.id}`"
                    @click="deleteUser(user)"
                  >
                    <Loader2
                      v-if="deletingKey === `user:${user.id}`"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    <Trash2 v-else class="mr-2 h-4 w-4" />
                    Delete
                  </Button>
                </div>
                <div
                  v-if="users.length === 0"
                  class="rounded-xl border border-dashed border-border px-4 py-6 text-sm text-muted-foreground"
                >
                  No users loaded for the selected organization.
                </div>
              </div>
            </ScrollArea>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2 text-lg">
              <Smartphone class="h-4 w-4" />
              Meta WhatsApp accounts
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea class="h-[320px] pr-4">
              <div class="space-y-3">
                <div
                  v-for="account in accounts"
                  :key="account.id"
                  class="flex items-center justify-between gap-3 rounded-xl border border-border/70 px-4 py-3"
                >
                  <div class="min-w-0">
                    <div class="font-medium">{{ account.name }}</div>
                    <div class="text-xs text-muted-foreground">
                      {{
                        account.display_name ||
                        account.phone_number ||
                        account.status
                      }}
                    </div>
                  </div>
                  <Button
                    v-if="canDeleteAccounts"
                    variant="destructive"
                    size="sm"
                    :disabled="deletingKey === `account:${account.id}`"
                    @click="deleteAccount(account)"
                  >
                    <Loader2
                      v-if="deletingKey === `account:${account.id}`"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    <Trash2 v-else class="mr-2 h-4 w-4" />
                    Delete
                  </Button>
                </div>
                <div
                  v-if="accounts.length === 0"
                  class="rounded-xl border border-dashed border-border px-4 py-6 text-sm text-muted-foreground"
                >
                  No Meta accounts loaded for the selected organization.
                </div>
              </div>
            </ScrollArea>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle class="flex items-center gap-2 text-lg">
              <Smartphone class="h-4 w-4" />
              Whatsmeow instances
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea class="h-[320px] pr-4">
              <div class="space-y-3">
                <div
                  v-for="instance in instances"
                  :key="instance.id"
                  class="flex items-center justify-between gap-3 rounded-xl border border-border/70 px-4 py-3"
                >
                  <div class="min-w-0">
                    <div class="font-medium">{{ instance.name }}</div>
                    <div class="text-xs text-muted-foreground">
                      {{ instance.phone_number || instance.status }}
                    </div>
                  </div>
                  <Button
                    v-if="canDeleteAccounts"
                    variant="destructive"
                    size="sm"
                    :disabled="deletingKey === `instance:${instance.id}`"
                    @click="deleteInstance(instance)"
                  >
                    <Loader2
                      v-if="deletingKey === `instance:${instance.id}`"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    <Trash2 v-else class="mr-2 h-4 w-4" />
                    Delete
                  </Button>
                </div>
                <div
                  v-if="instances.length === 0"
                  class="rounded-xl border border-dashed border-border px-4 py-6 text-sm text-muted-foreground"
                >
                  No Whatsmeow instances loaded for the selected organization.
                </div>
              </div>
            </ScrollArea>
          </CardContent>
        </Card>
      </div>
    </div>
  </div>
</template>
