<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useOrganizationsStore } from "@/stores/organizations";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DeleteConfirmDialog } from "@/components/shared";
import { organizationsService } from "@/services/api";
import { toast } from "vue-sonner";
import { getErrorMessage } from "@/lib/api-utils";
import {
  Building2,
  Check,
  ChevronsUpDown,
  Plus,
  Loader2,
  Trash2,
} from "lucide-vue-next";

const props = withDefaults(
  defineProps<{
    expanded?: boolean;
    collapsed?: boolean;
  }>(),
  {
    expanded: undefined,
    collapsed: false,
  },
);
const emit = defineEmits<{
  "overlay-open-change": [value: boolean];
}>();

const { t } = useI18n();
const organizationsStore = useOrganizationsStore();
const authStore = useAuthStore();
const configStore = useConfigStore();
const isOrgMenuOpen = ref(false);

const isSuperAdmin = computed(() => authStore.user?.is_super_admin || false);
const canCreateOrg = computed(() =>
  authStore.hasPermission("organizations", "write"),
);
const canDeleteOrg = computed(
  () =>
    isSuperAdmin.value && authStore.hasPermission("organizations", "delete"),
);

const shouldShowSwitcher = computed(
  () =>
    !configStore.tenant.subdomain_locked &&
    (isSuperAdmin.value || organizationsStore.isMultiOrg),
);

// Build the org list depending on user type
const orgList = computed(() => {
  if (isSuperAdmin.value) {
    return organizationsStore.organizations.map((org) => ({
      id: org.id,
      name: org.name,
    }));
  }
  return organizationsStore.myOrganizations.map((org) => ({
    id: org.organization_id,
    name: org.name,
  }));
});

const currentOrgId = computed(() => {
  if (isSuperAdmin.value) {
    return organizationsStore.selectedOrgId || "";
  }
  return authStore.user?.organization_id || "";
});
const currentOrgName = computed(
  () =>
    orgList.value.find((org) => org.id === currentOrgId.value)?.name ||
    "Select organization",
);
const isExpanded = computed(() => props.expanded ?? !props.collapsed);

onMounted(async () => {
  // Fetch user's org memberships for all authenticated users
  await organizationsStore.fetchMyOrganizations();

  if (isSuperAdmin.value) {
    organizationsStore.init();
    await organizationsStore.fetchOrganizations();

    // If no org selected, default to user's own org
    if (!organizationsStore.selectedOrgId && authStore.user?.organization_id) {
      organizationsStore.selectOrganization(authStore.user.organization_id);
    }
  }
});

// Watch for auth changes
watch(
  () => authStore.user?.is_super_admin,
  async (superAdmin) => {
    if (superAdmin) {
      organizationsStore.init();
      await organizationsStore.fetchOrganizations();
    }
  },
);

watch(isOrgMenuOpen, (open) => {
  emit("overlay-open-change", open);
});

const handleOrgChange = async (
  value: string | number | bigint | Record<string, any> | null,
) => {
  if (!value || typeof value !== "string") return;
  if (value === currentOrgId.value) return;

  if (isSuperAdmin.value) {
    // Super admins: set localStorage header and reload
    organizationsStore.selectOrganization(value);
    window.location.reload();
  } else {
    // Multi-org users: call switchOrg API for new JWT tokens, then reload
    try {
      await authStore.switchOrg(value);
      window.location.reload();
    } catch {
      // If switch fails, don't reload
    }
  }
};

const switchOrganization = async (orgId: string) => {
  isOrgMenuOpen.value = false;
  await handleOrgChange(orgId);
};

// Create org dialog
const isCreateDialogOpen = ref(false);
const newOrgName = ref("");
const newOrgSlug = ref("");
const isCreating = ref(false);
const isDeleteDialogOpen = ref(false);
const deletingOrg = ref<{ id: string; name: string } | null>(null);
const isDeleting = ref(false);

async function submitCreateOrg() {
  if (!newOrgName.value.trim()) return;
  isCreating.value = true;
  try {
    await organizationsService.create({
      name: newOrgName.value.trim(),
      slug: newOrgSlug.value.trim() || undefined,
    });
    toast.success(t("organizations.created"));
    isCreateDialogOpen.value = false;
    newOrgName.value = "";
    newOrgSlug.value = "";
    await refreshOrgs();
  } catch {
    toast.error(t("organizations.createFailed"));
  } finally {
    isCreating.value = false;
  }
}

function openDeleteOrgDialog() {
  const selectedID = currentOrgId.value;
  if (!selectedID) {
    toast.error(t("organizations.selectToDelete"));
    return;
  }

  if (organizationsStore.organizations.length <= 1) {
    toast.error(t("organizations.cannotDeleteLast"));
    return;
  }

  const org = organizationsStore.organizations.find(
    (item) => item.id === selectedID,
  );
  if (!org) {
    toast.error(t("organizations.selectToDelete"));
    return;
  }

  deletingOrg.value = { id: org.id, name: org.name };
  isDeleteDialogOpen.value = true;
}

async function submitDeleteOrg() {
  if (!deletingOrg.value || isDeleting.value) return;

  isDeleting.value = true;
  const deletedOrgID = deletingOrg.value.id;

  try {
    await organizationsService.delete(deletedOrgID);
    toast.success(t("organizations.deleted"));
    isDeleteDialogOpen.value = false;
    deletingOrg.value = null;

    const wasSelected = organizationsStore.selectedOrgId === deletedOrgID;
    await organizationsStore.fetchOrganizations();

    if (wasSelected) {
      const fallback = organizationsStore.organizations.find(
        (item) => item.id !== deletedOrgID,
      );
      organizationsStore.selectOrganization(fallback?.id || null);
      window.location.reload();
    }
  } catch (err) {
    toast.error(getErrorMessage(err, t("organizations.deleteFailed")));
  } finally {
    isDeleting.value = false;
  }
}

const refreshOrgs = async () => {
  if (isSuperAdmin.value) {
    await organizationsStore.fetchOrganizations();
  } else {
    await organizationsStore.fetchMyOrganizations();
  }
};
</script>

<template>
  <div
    v-if="shouldShowSwitcher"
    data-testid="org-switcher"
    class="border-b border-sidebar-border px-2 py-2"
  >
    <div v-if="isExpanded" class="space-y-1">
      <div class="flex items-center justify-between">
        <span
          class="text-[11px] font-medium text-muted-foreground uppercase tracking-wide px-1"
        >
          Organization
        </span>
        <div class="flex items-center gap-1">
          <Button
            v-if="canDeleteOrg"
            variant="ghost"
            size="icon"
            class="h-5 w-5"
            :title="t('organizations.deleteCurrent')"
            @click="openDeleteOrgDialog"
          >
            <Trash2 class="h-3 w-3 text-destructive" />
          </Button>
          <Button
            v-if="canCreateOrg"
            variant="ghost"
            size="icon"
            class="h-5 w-5"
            @click="isCreateDialogOpen = true"
          >
            <Plus class="h-3 w-3" />
          </Button>
        </div>
      </div>
      <Popover v-if="orgList.length > 0" v-model:open="isOrgMenuOpen">
        <PopoverTrigger as-child>
          <Button
            variant="outline"
            data-testid="org-switcher-trigger"
            class="h-8 w-full justify-between rounded-lg border-sidebar-border bg-sidebar/70 px-2.5 text-[13px] font-medium text-sidebar-foreground hover:bg-sidebar-accent/70 hover:text-sidebar-foreground"
            :aria-label="currentOrgName"
          >
            <span class="flex min-w-0 items-center gap-2">
              <Building2
                class="h-3.5 w-3.5 shrink-0 text-sidebar-foreground/60"
              />
              <span class="truncate">{{ currentOrgName }}</span>
            </span>
            <ChevronsUpDown
              class="h-3.5 w-3.5 shrink-0 text-sidebar-foreground/55"
            />
          </Button>
        </PopoverTrigger>
        <PopoverContent
          data-testid="org-switcher-content"
          side="bottom"
          align="start"
          class="w-[var(--radix-popover-trigger-width)] min-w-[14rem] p-1.5"
        >
          <div
            class="px-2 pb-1 text-[11px] font-medium uppercase tracking-wide text-muted-foreground"
          >
            Organization
          </div>
          <div class="max-h-72 space-y-1 overflow-y-auto">
            <button
              v-for="org in orgList"
              :key="org.id"
              type="button"
              :data-testid="`org-switcher-item-${org.id}`"
              :data-org-id="org.id"
              class="flex w-full items-center justify-between rounded-md px-2 py-2 text-left text-sm transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
              :class="
                org.id === currentOrgId
                  ? 'bg-accent/80 text-foreground'
                  : 'text-foreground/85'
              "
              @click="switchOrganization(org.id)"
            >
              <span class="flex min-w-0 items-center gap-2">
                <Building2 class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <span class="truncate">{{ org.name }}</span>
              </span>
              <Check
                v-if="org.id === currentOrgId"
                class="h-4 w-4 shrink-0 text-primary"
              />
            </button>
          </div>
        </PopoverContent>
      </Popover>
      <div
        v-else-if="organizationsStore.loading"
        class="text-[12px] text-muted-foreground px-1"
      >
        Loading...
      </div>
      <div
        v-else-if="organizationsStore.error"
        class="text-[12px] text-destructive px-1"
      >
        {{ organizationsStore.error }}
      </div>
      <div v-else class="text-[12px] text-muted-foreground px-1">
        No organizations found
      </div>
    </div>

    <!-- Collapsed view - just show icon with selected org initial -->
    <div v-else class="flex justify-center">
      <Button
        variant="ghost"
        size="icon"
        class="h-8 w-8"
        :title="
          organizationsStore.selectedOrganization?.name || 'All Organizations'
        "
      >
        <Building2 class="h-4 w-4" />
      </Button>
    </div>
  </div>

  <!-- Create Org Dialog -->
  <Dialog v-model:open="isCreateDialogOpen">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ t("organizations.createTitle") }}</DialogTitle>
        <DialogDescription>{{
          t("organizations.createDesc")
        }}</DialogDescription>
      </DialogHeader>
      <div class="py-4">
        <div class="space-y-3">
          <Input
            v-model="newOrgName"
            :placeholder="t('organizations.namePlaceholder')"
            @keydown.enter="submitCreateOrg"
          />
          <Input
            v-model="newOrgSlug"
            :placeholder="t('organizations.slugPlaceholder')"
            @keydown.enter="submitCreateOrg"
          />
        </div>
      </div>
      <DialogFooter>
        <Button variant="outline" @click="isCreateDialogOpen = false">{{
          t("common.cancel")
        }}</Button>
        <Button
          @click="submitCreateOrg"
          :disabled="isCreating || !newOrgName.trim()"
        >
          <Loader2 v-if="isCreating" class="h-4 w-4 mr-2 animate-spin" />
          {{ t("common.create") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <DeleteConfirmDialog
    v-model:open="isDeleteDialogOpen"
    :title="t('organizations.deleteTitle')"
    :description="t('organizations.deleteWarning')"
    :item-name="deletingOrg?.name"
    :confirm-label="isDeleting ? t('common.loading') : t('common.delete')"
    @confirm="submitDeleteOrg"
  />
</template>
