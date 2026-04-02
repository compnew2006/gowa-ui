<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue";
import {
  Database,
  ArrowRightLeft,
  CheckCircle2,
  AlertCircle,
  Loader2,
  RefreshCw,
} from "lucide-vue-next";
import { PageHeader } from "@/components/shared";
import { migrationService, type MigrationOrgStatus } from "@/services/api";
import { useToast } from "@/components/ui/toast";

const { toast } = useToast();

const loading = ref(false);
const migrating = ref(false);
const statusData = ref<{
  overall_complete: boolean;
  organizations: MigrationOrgStatus[];
} | null>(null);
const error = ref("");
let pollInterval: ReturnType<typeof setInterval> | null = null;

const hasOrgs = computed(
  () => statusData.value && statusData.value.organizations?.length > 0,
);

async function fetchStatus() {
  try {
    loading.value = true;
    error.value = "";
    const res = await migrationService.status();
    const payload = (res.data as any)?.data ?? res.data;
    statusData.value = payload as {
      overall_complete: boolean;
      organizations: MigrationOrgStatus[];
    };
  } catch (e: any) {
    const msg =
      e?.response?.data?.message ||
      e?.message ||
      "Failed to fetch migration status";
    error.value = msg;
  } finally {
    loading.value = false;
  }
}

async function triggerMigration(orgId?: string) {
  try {
    migrating.value = true;
    await migrationService.trigger(orgId);
    toast({
      title: "Migration started",
      description: "Migration started in background",
    });
    // Start polling
    startPolling();
  } catch (e: any) {
    const msg =
      e?.response?.data?.message || e?.message || "Failed to start migration";
    toast({ title: "Error", description: msg, variant: "destructive" });
  } finally {
    migrating.value = false;
  }
}

function startPolling() {
  if (pollInterval) clearInterval(pollInterval);
  pollInterval = setInterval(fetchStatus, 3000);
}

function stopPolling() {
  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }
}

function progressPercent(org: MigrationOrgStatus) {
  const total = org.contacts_total + org.messages_total;
  if (total === 0) return 100;
  const done = org.contacts_migrated + org.messages_migrated;
  return Math.round((done / total) * 100);
}

onMounted(fetchStatus);
onUnmounted(stopPolling);
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      title="Data Migration"
      subtitle="Migrate data from Meta WhatsApp Accounts to Whatsmeow Instances."
      :icon="Database"
      icon-gradient="bg-gradient-to-br from-amber-500 to-orange-600 shadow-amber-500/20"
    />

    <div class="flex-1 p-6 overflow-y-auto space-y-6">
      <!-- Error Banner -->
      <div
        v-if="error"
        class="rounded-xl border border-red-500/30 bg-red-500/10 p-4 flex items-start gap-3"
      >
        <AlertCircle class="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" />
        <div>
          <p class="text-red-300 text-sm font-medium">
            Error loading migration status
          </p>
          <p class="text-red-400/80 text-xs mt-1">{{ error }}</p>
        </div>
      </div>

      <!-- Loading -->
      <div
        v-if="loading && !statusData"
        class="flex items-center justify-center py-20"
      >
        <Loader2 class="w-6 h-6 text-amber-400 animate-spin" />
        <span class="ml-2 text-zinc-400 text-sm"
          >Loading migration status…</span
        >
      </div>

      <!-- No Orgs -->
      <div v-else-if="!hasOrgs && !error" class="text-center py-20">
        <Database class="w-12 h-12 text-zinc-600 mx-auto mb-4" />
        <p class="text-zinc-400 text-sm">
          No WhatsApp Accounts found to migrate.
        </p>
        <p class="text-zinc-500 text-xs mt-1">
          Migration is only needed if you have existing Meta accounts.
        </p>
      </div>

      <!-- Migration Dashboard -->
      <template v-else-if="statusData">
        <!-- Overall Status Card -->
        <div class="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div
                :class="[
                  'w-10 h-10 rounded-lg flex items-center justify-center',
                  statusData.overall_complete
                    ? 'bg-emerald-500/20 text-emerald-400'
                    : 'bg-amber-500/20 text-amber-400',
                ]"
              >
                <CheckCircle2
                  v-if="statusData.overall_complete"
                  class="w-5 h-5"
                />
                <ArrowRightLeft v-else class="w-5 h-5" />
              </div>
              <div>
                <p class="text-sm font-medium text-zinc-200">
                  {{
                    statusData.overall_complete
                      ? "Migration Complete"
                      : "Migration Pending"
                  }}
                </p>
                <p class="text-xs text-zinc-500">
                  {{ statusData.organizations?.length ?? 0 }} organization(s)
                  with WhatsApp accounts
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <button
                @click="fetchStatus"
                :disabled="loading"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-zinc-700 text-zinc-300 hover:bg-zinc-800 transition disabled:opacity-50"
              >
                <RefreshCw
                  :class="['w-3.5 h-3.5', loading && 'animate-spin']"
                />
                Refresh
              </button>
              <button
                v-if="!statusData.overall_complete"
                @click="triggerMigration()"
                :disabled="migrating"
                class="inline-flex items-center gap-1.5 px-4 py-1.5 text-xs font-medium rounded-lg bg-gradient-to-r from-amber-500 to-orange-600 text-white hover:from-amber-600 hover:to-orange-700 transition disabled:opacity-50"
              >
                <Loader2 v-if="migrating" class="w-3.5 h-3.5 animate-spin" />
                <ArrowRightLeft v-else class="w-3.5 h-3.5" />
                Migrate All
              </button>
            </div>
          </div>
        </div>

        <!-- Per-Org Cards -->
        <div class="space-y-4">
          <div
            v-for="org in statusData.organizations"
            :key="org.organization_id"
            class="rounded-xl border border-zinc-800 bg-zinc-900/50 p-5"
          >
            <!-- Org Header -->
            <div class="flex items-center justify-between mb-4">
              <div>
                <p class="text-sm font-medium text-zinc-200">
                  {{ org.organization_name || org.organization_id }}
                </p>
                <p class="text-xs text-zinc-500">
                  {{ org.accounts_count }} account(s) →
                  {{ org.instances_count }} instance(s)
                </p>
              </div>
              <div class="flex items-center gap-2">
                <span
                  :class="[
                    'text-xs px-2 py-0.5 rounded-full font-medium',
                    org.migration_complete
                      ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                      : 'bg-amber-500/10 text-amber-400 border border-amber-500/30',
                  ]"
                >
                  {{ org.migration_complete ? "Complete" : "Pending" }}
                </span>
                <button
                  v-if="!org.migration_complete"
                  @click="triggerMigration(org.organization_id)"
                  :disabled="migrating"
                  class="text-xs px-3 py-1 rounded-lg border border-amber-500/30 text-amber-400 hover:bg-amber-500/10 transition disabled:opacity-50"
                >
                  Migrate
                </button>
              </div>
            </div>

            <!-- Progress Bar -->
            <div class="mb-3">
              <div class="flex justify-between text-xs text-zinc-500 mb-1">
                <span>Progress</span>
                <span>{{ progressPercent(org) }}%</span>
              </div>
              <div class="h-2 bg-zinc-800 rounded-full overflow-hidden">
                <div
                  :class="[
                    'h-full rounded-full transition-all duration-500',
                    org.migration_complete
                      ? 'bg-gradient-to-r from-emerald-500 to-green-400'
                      : 'bg-gradient-to-r from-amber-500 to-orange-500',
                  ]"
                  :style="{ width: `${progressPercent(org)}%` }"
                />
              </div>
            </div>

            <!-- Stats Grid -->
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div class="rounded-lg bg-zinc-800/50 p-3">
                <p class="text-xs text-zinc-500">Contacts</p>
                <p class="text-lg font-semibold text-zinc-200">
                  {{ org.contacts_migrated }}
                  <span class="text-xs text-zinc-500 font-normal"
                    >/ {{ org.contacts_total }}</span
                  >
                </p>
              </div>
              <div class="rounded-lg bg-zinc-800/50 p-3">
                <p class="text-xs text-zinc-500">Messages</p>
                <p class="text-lg font-semibold text-zinc-200">
                  {{ org.messages_migrated }}
                  <span class="text-xs text-zinc-500 font-normal"
                    >/ {{ org.messages_total }}</span
                  >
                </p>
              </div>
              <div class="rounded-lg bg-zinc-800/50 p-3">
                <p class="text-xs text-zinc-500">Contacts Pending</p>
                <p
                  class="text-lg font-semibold"
                  :class="
                    org.contacts_pending > 0
                      ? 'text-amber-400'
                      : 'text-emerald-400'
                  "
                >
                  {{ org.contacts_pending }}
                </p>
              </div>
              <div class="rounded-lg bg-zinc-800/50 p-3">
                <p class="text-xs text-zinc-500">Messages Pending</p>
                <p
                  class="text-lg font-semibold"
                  :class="
                    org.messages_pending > 0
                      ? 'text-amber-400'
                      : 'text-emerald-400'
                  "
                >
                  {{ org.messages_pending }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
