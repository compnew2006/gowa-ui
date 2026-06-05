<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  KeyRound,
  Loader2,
  LockKeyhole,
  ShieldCheck,
} from "lucide-vue-next";
import { PageHeader } from "@/components/shared";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { useLicenseActivation } from "@/composables/useLicenseActivation";

const { t } = useI18n();
const {
  licenseStore,
  securityKey,
  bootstrapPending,
  statusVariant,
  statusLabel,
  entitlementLabel,
  licenseMetaLabel,
  quotaCards,
  formatBytes,
  formatQuotaValue,
  usagePercentage,
  loadBootstrap,
  copyHWID,
  submitActivation,
} = useLicenseActivation();

function parseDurationDays(durationLabel?: string | null): number | null {
  if (!durationLabel || durationLabel === "lifetime") {
    return null;
  }

  const dayMatch = durationLabel.match(/^(\d+)d$/i);
  if (!dayMatch) {
    return null;
  }

  const parsed = Number(dayMatch[1]);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function getExpiryMessage(daysUntilExpiry: number | null) {
  if (daysUntilExpiry === 0) {
    return t("licenseSettings.summary.expiresToday");
  }
  if (daysUntilExpiry === 1) {
    return t("licenseSettings.summary.expiresInOneDay");
  }
  if (daysUntilExpiry !== null && daysUntilExpiry > 1) {
    return t("licenseSettings.summary.expiresInDays", {
      count: daysUntilExpiry,
    });
  }
  return t("licenseSettings.summary.healthy");
}

function getQuotaUsageLabel(
  key: string,
  limit: number,
  overQuota: boolean,
  overage: number,
) {
  if (licenseStore.isDisabled || limit <= 0) {
    return t("licenseSettings.quota.notEnforced");
  }
  if (overQuota) {
    return t("licenseSettings.quota.overBy", {
      count: formatQuotaValue(key, overage),
    });
  }
  return t("licenseSettings.quota.withinCapacity");
}

function getQuotaLimitLabel(key: string, limit: number) {
  if (licenseStore.isDisabled || limit <= 0) {
    return t("licenseSettings.quota.notEnforcedShort");
  }
  return formatQuotaValue(key, limit);
}

function getOrganizationUsageLabel(
  userCount: number,
  endpointCount: number,
  storageBytes: number,
) {
  return t("licenseSettings.organizationUsage.summary", {
    users: userCount,
    endpoints: endpointCount,
    storage: formatBytes(storageBytes),
  });
}

const graceDeadlineLabel = computed(
  () =>
    licenseStore.state.grace_deadline ||
    t("licenseSettings.placeholders.deadline"),
);

const shortIdLabel = computed(
  () =>
    licenseStore.state.hwid_short || t("licenseSettings.placeholders.pending"),
);

const statusSummary = computed(() => {
  if (licenseStore.isDisabled) {
    return t("licenseSettings.summary.disabled");
  }
  if (licenseStore.showQuotaOverage) {
    return t("licenseSettings.summary.paused");
  }
  if (licenseStore.isGrace) {
    return t("licenseSettings.summary.grace");
  }
  if (licenseStore.showExpiryWarning) {
    return getExpiryMessage(licenseStore.daysUntilExpiry);
  }
  return t("licenseSettings.summary.healthy");
});

const totalDurationDays = computed(() =>
  parseDurationDays(licenseStore.state.duration_label),
);

const remainingDurationDays = computed(() => {
  const total = totalDurationDays.value;
  const remaining = Math.max(licenseStore.daysUntilExpiry ?? 0, 0);
  if (total === null) {
    return remaining;
  }
  return Math.min(remaining, total);
});

const durationHeadline = computed(() => {
  if (licenseStore.isDisabled) {
    return t("licenseSettings.duration.disabled");
  }

  if (licenseStore.state.duration_label === "lifetime") {
    return t("licenseSettings.duration.lifetime");
  }

  if (totalDurationDays.value !== null) {
    return `${remainingDurationDays.value}/${totalDurationDays.value}`;
  }

  return t("licenseSettings.duration.unavailable");
});

const durationProgressValue = computed(() => {
  if (licenseStore.isDisabled) {
    return 0;
  }

  if (licenseStore.state.duration_label === "lifetime") {
    return 100;
  }

  if (totalDurationDays.value === null) {
    return 0;
  }

  return usagePercentage(remainingDurationDays.value, totalDurationDays.value);
});

const durationUsageLabel = computed(() => {
  if (licenseStore.isDisabled) {
    return t("licenseSettings.duration.disabledDescription");
  }

  if (licenseStore.state.duration_label === "lifetime") {
    return t("licenseSettings.duration.noExpiry");
  }

  if (totalDurationDays.value === null) {
    return t("licenseSettings.duration.unavailable");
  }

  if (remainingDurationDays.value <= 0) {
    return t("licenseSettings.banner.remainingExpiresToday");
  }

  if (remainingDurationDays.value === 1) {
    return t("licenseSettings.banner.remainingOneDay");
  }

  return t("licenseSettings.banner.remainingDays", {
    count: remainingDurationDays.value,
  });
});

onMounted(() => {
  void loadBootstrap();
});

async function refreshStatus() {
  await loadBootstrap(true);
}

async function handleActivationSubmit() {
  await submitActivation({
    onSuccess: async () => {
      await loadBootstrap(true);
    },
  });
}
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      :title="t('licenseSettings.title')"
      :subtitle="t('licenseSettings.subtitle')"
      :icon="ShieldCheck"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
    >
      <template #actions>
        <Button
          variant="outline"
          :disabled="bootstrapPending"
          @click="refreshStatus"
        >
          <Loader2 v-if="bootstrapPending" class="mr-2 h-4 w-4 animate-spin" />
          {{ t("licenseSettings.actions.refresh") }}
        </Button>
      </template>
    </PageHeader>

    <ScrollArea class="flex-1">
      <div class="mx-auto flex max-w-6xl flex-col gap-6 p-6">
        <div class="grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_420px]">
          <section class="space-y-6">
            <Card class="border-border/70 bg-card/95 shadow-sm">
              <CardHeader class="space-y-4">
                <div
                  class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between"
                >
                  <div class="space-y-2">
                    <CardDescription
                      class="text-[11px] uppercase tracking-[0.18em]"
                    >
                      {{ t("licenseSettings.sections.deploymentStatus") }}
                    </CardDescription>
                    <CardTitle class="text-2xl">
                      {{ t("licenseSettings.sections.overview") }}
                    </CardTitle>
                  </div>
                  <div class="flex flex-wrap gap-2">
                    <Badge :variant="statusVariant">{{ statusLabel }}</Badge>
                    <Badge v-if="licenseStore.state.tier" variant="outline">
                      {{ entitlementLabel }}
                    </Badge>
                    <Badge v-if="licenseMetaLabel" variant="outline">
                      {{ licenseMetaLabel }}
                    </Badge>
                    <Badge
                      v-if="licenseStore.showQuotaOverage"
                      variant="warning"
                    >
                      {{ t("licenseSettings.status.quotaOverage") }}
                    </Badge>
                  </div>
                </div>
                <p class="max-w-3xl text-sm leading-6 text-muted-foreground">
                  {{ statusSummary }}
                </p>
              </CardHeader>
              <CardContent class="space-y-4">
                <Alert
                  v-if="licenseStore.isGrace"
                  variant="warning"
                  class="border-amber-500/35"
                >
                  <AlertTriangle class="h-4 w-4" />
                  <AlertTitle>
                    {{ t("licenseSettings.alerts.graceTitle") }}
                  </AlertTitle>
                  <AlertDescription>
                    {{
                      t("licenseSettings.alerts.graceDescription", {
                        deadline: graceDeadlineLabel,
                      })
                    }}
                  </AlertDescription>
                </Alert>

                <Alert
                  v-else-if="licenseStore.showQuotaOverage"
                  variant="warning"
                  class="border-amber-500/35"
                >
                  <AlertTriangle class="h-4 w-4" />
                  <AlertTitle>
                    {{ t("licenseSettings.alerts.quotaTitle") }}
                  </AlertTitle>
                  <AlertDescription>
                    {{ t("licenseSettings.alerts.quotaDescription") }}
                  </AlertDescription>
                </Alert>

                <Alert
                  v-else-if="licenseStore.showExpiryWarning"
                  variant="info"
                  class="border-primary/25"
                >
                  <ShieldCheck class="h-4 w-4" />
                  <AlertTitle>
                    {{ t("licenseSettings.alerts.expiryTitle") }}
                  </AlertTitle>
                  <AlertDescription>
                    {{ getExpiryMessage(licenseStore.daysUntilExpiry) }}
                  </AlertDescription>
                </Alert>

                <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                  <Card
                    v-for="item in quotaCards"
                    :key="item.key"
                    class="border-border/70 bg-background/80 shadow-sm"
                  >
                    <CardHeader class="pb-3">
                      <CardDescription
                        class="text-[11px] uppercase tracking-[0.16em]"
                      >
                        {{ item.label }}
                      </CardDescription>
                      <CardTitle class="text-2xl">
                        {{ formatQuotaValue(item.key, item.usage.current) }}/{{
                          getQuotaLimitLabel(item.key, item.usage.limit)
                        }}
                      </CardTitle>
                    </CardHeader>
                    <CardContent class="space-y-3 pt-0">
                      <Progress
                        :model-value="
                          usagePercentage(item.usage.current, item.usage.limit)
                        "
                      />
                      <p
                        class="text-xs"
                        :class="
                          item.usage.over_quota
                            ? 'text-amber-600 dark:text-amber-400'
                            : 'text-muted-foreground'
                        "
                      >
                        {{
                          getQuotaUsageLabel(
                            item.key,
                            item.usage.limit,
                            item.usage.over_quota,
                            item.usage.overage,
                          )
                        }}
                      </p>
                    </CardContent>
                  </Card>

                  <Card class="border-border/70 bg-background/80 shadow-sm">
                    <CardHeader class="pb-3">
                      <CardDescription
                        class="text-[11px] uppercase tracking-[0.16em]"
                      >
                        {{ t("licenseSettings.quota.subscriptionDays") }}
                      </CardDescription>
                      <CardTitle class="text-2xl">
                        {{ durationHeadline }}
                      </CardTitle>
                    </CardHeader>
                    <CardContent class="space-y-3 pt-0">
                      <Progress :model-value="durationProgressValue" />
                      <p class="text-xs text-muted-foreground">
                        {{ durationUsageLabel }}
                      </p>
                    </CardContent>
                  </Card>
                </div>

                <Card
                  v-if="
                    licenseStore.state.usage.organization_details.length > 0
                  "
                  class="border-border/70 bg-background/80 shadow-sm"
                >
                  <CardHeader class="pb-3">
                    <CardTitle class="flex items-center gap-2 text-base">
                      <CheckCircle2 class="h-4 w-4 text-emerald-500" />
                      {{ t("licenseSettings.sections.currentOrgUsage") }}
                    </CardTitle>
                    <CardDescription>
                      {{ t("licenseSettings.descriptions.orgUsage") }}
                    </CardDescription>
                  </CardHeader>
                  <CardContent class="space-y-2">
                    <div
                      v-for="org in licenseStore.state.usage
                        .organization_details"
                      :key="org.organization_id"
                      class="flex flex-col gap-1 rounded-xl border border-border/60 bg-card px-3 py-2 text-sm md:flex-row md:items-center md:justify-between"
                    >
                      <span class="font-medium text-foreground">
                        {{ org.organization_name }}
                      </span>
                      <span class="text-muted-foreground">
                        {{
                          getOrganizationUsageLabel(
                            org.user_count,
                            org.whatsapp_endpoint_count,
                            org.storage_bytes,
                          )
                        }}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              </CardContent>
            </Card>
          </section>

          <aside class="space-y-6">
            <Card class="border-border/70 bg-card/95 shadow-sm">
              <CardHeader>
                <CardTitle>
                  {{ t("licenseSettings.sections.serverIdentity") }}
                </CardTitle>
                <CardDescription>
                  {{ t("licenseSettings.descriptions.serverIdentity") }}
                </CardDescription>
              </CardHeader>
              <CardContent class="space-y-3">
                <Label for="license-hwid">
                  {{ t("licenseSettings.fields.hwid") }}
                </Label>
                <div class="flex flex-col gap-2">
                  <Input
                    id="license-hwid"
                    :model-value="licenseStore.state.hwid_full"
                    readonly
                    class="font-mono text-[11px] tracking-[0.02em]"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    class="justify-start"
                    :disabled="
                      bootstrapPending || !licenseStore.state.hwid_full
                    "
                    @click="copyHWID"
                  >
                    <Copy class="mr-2 h-4 w-4" />
                    {{ t("licenseSettings.actions.copyHwid") }}
                  </Button>
                </div>
                <p class="text-xs text-muted-foreground">
                  {{
                    t("licenseSettings.fields.shortIdValue", {
                      value: shortIdLabel,
                    })
                  }}
                </p>
              </CardContent>
            </Card>

            <Card class="border-border/70 bg-card/95 shadow-sm">
              <CardHeader>
                <CardTitle>
                  {{ t("licenseSettings.sections.activation") }}
                </CardTitle>
                <CardDescription>
                  {{ t("licenseSettings.descriptions.activation") }}
                </CardDescription>
              </CardHeader>
              <CardContent class="space-y-4">
                <div class="space-y-2">
                  <Label for="license-security-key">
                    {{ t("licenseSettings.fields.securityKey") }}
                  </Label>
                  <Textarea
                    id="license-security-key"
                    v-model="securityKey"
                    :rows="8"
                    :disabled="licenseStore.activationPending"
                    class="min-h-[220px] font-mono text-xs"
                    :placeholder="t('licenseSettings.placeholders.securityKey')"
                  />
                </div>

                <Alert
                  v-if="licenseStore.error"
                  variant="destructive"
                  class="border-destructive/35"
                >
                  <LockKeyhole class="h-4 w-4" />
                  <AlertTitle>
                    {{ t("licenseSettings.errors.activationBlocked") }}
                  </AlertTitle>
                  <AlertDescription>
                    {{ licenseStore.error }}
                  </AlertDescription>
                </Alert>

                <div class="flex flex-col gap-3">
                  <Button
                    :disabled="
                      licenseStore.activationPending || bootstrapPending
                    "
                    @click="handleActivationSubmit"
                  >
                    <Loader2
                      v-if="licenseStore.activationPending"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    <KeyRound v-else class="mr-2 h-4 w-4" />
                    {{ t("licenseSettings.actions.activateLicense") }}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    :disabled="bootstrapPending"
                    @click="refreshStatus"
                  >
                    <Loader2
                      v-if="bootstrapPending"
                      class="mr-2 h-4 w-4 animate-spin"
                    />
                    {{ t("licenseSettings.actions.refreshStatus") }}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </aside>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
