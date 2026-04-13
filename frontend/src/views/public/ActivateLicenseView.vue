<script setup lang="ts">
import { onMounted } from "vue";
import { useRouter } from "vue-router";
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  KeyRound,
  Loader2,
  LockKeyhole,
  MessageSquare,
  ShieldCheck,
} from "lucide-vue-next";
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
import { useAuthStore } from "@/stores/auth";
import { useLicenseActivation } from "@/composables/useLicenseActivation";

const router = useRouter();
const authStore = useAuthStore();
const {
  licenseStore,
  securityKey,
  bootstrapPending,
  statusVariant,
  statusLabel,
  entitlementLabel,
  licenseMetaLabel,
  quotaCards,
  usagePercentage,
  loadBootstrap,
  copyHWID,
  submitActivation,
} = useLicenseActivation();

onMounted(() => {
  void loadBootstrap();
});

function getQuotaLimitLabel(limit: number) {
  if (licenseStore.isDisabled || limit <= 0) {
    return "Not enforced";
  }
  return String(limit);
}

function getQuotaUsageLabel(
  limit: number,
  overQuota: boolean,
  overage: number,
) {
  if (licenseStore.isDisabled || limit <= 0) {
    return "Licensing is disabled for this deployment";
  }
  if (overQuota) {
    return `Over by ${overage}`;
  }
  return "Within licensed capacity";
}

async function handleActivationSubmit() {
  await submitActivation({
    onSuccess: async () => {
      if (authStore.isAuthenticated) {
        await router.push("/settings/license");
        return;
      }
      await router.push("/login");
    },
  });
}
</script>

<template>
  <div class="auth-shell px-4 py-6 sm:px-6 sm:py-10">
    <div
      class="mx-auto w-full max-w-6xl overflow-hidden rounded-[calc(var(--radius)+1rem)] border border-border/70 bg-card/95 text-card-foreground shadow-2xl backdrop-blur-xl"
    >
      <div
        class="grid gap-0 xl:grid-cols-[minmax(0,1.08fr)_minmax(360px,0.92fr)]"
      >
        <section
          class="relative overflow-hidden border-b border-border/70 bg-card/80 xl:border-b-0 xl:border-r"
        >
          <div
            class="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(30,157,241,0.14),transparent_52%),radial-gradient(circle_at_bottom_right,rgba(15,20,25,0.08),transparent_45%)]"
          />
          <div class="relative space-y-6 p-6 sm:p-8 xl:space-y-8 xl:p-10">
            <div
              class="flex flex-col items-start gap-4 sm:flex-row sm:items-center"
            >
              <div class="brand-badge shrink-0">
                <MessageSquare class="h-7 w-7 text-white" />
              </div>
              <div class="space-y-1">
                <p
                  class="text-xs font-semibold uppercase tracking-[0.22em] text-primary/80"
                >
                  Whatomate Host Licensing
                </p>
                <h1
                  class="max-w-xl text-3xl font-semibold tracking-tight text-foreground"
                >
                  Activate this deployment
                </h1>
              </div>
            </div>

            <div class="space-y-4">
              <div class="flex flex-wrap items-center gap-3">
                <Badge :variant="statusVariant">
                  {{ statusLabel }}
                </Badge>
                <Badge v-if="licenseStore.state.license_kind" variant="outline">
                  {{ entitlementLabel }}
                </Badge>
                <Badge v-if="licenseMetaLabel" variant="outline">
                  {{ licenseMetaLabel }}
                </Badge>
                <Badge v-if="licenseStore.showQuotaOverage" variant="warning">
                  Quota overage
                </Badge>
              </div>

              <p class="max-w-2xl text-sm leading-6 text-muted-foreground">
                Copy the host HWID below, send it to the vendor, then paste the
                signed security key here. The key is bound to this specific
                server and controls both access and quotas.
              </p>
            </div>

            <Alert
              v-if="licenseStore.isGrace"
              variant="warning"
              class="border-amber-500/35"
            >
              <AlertTriangle class="h-4 w-4" />
              <AlertTitle>Grace period</AlertTitle>
              <AlertDescription>
                This license is expired but still within the grace window. Renew
                before
                <span class="font-medium text-foreground">
                  {{ licenseStore.state.grace_deadline || "the deadline" }}
                </span>
                to avoid a hard lock.
              </AlertDescription>
            </Alert>

            <Alert
              v-else-if="licenseStore.showQuotaOverage"
              variant="warning"
              class="border-amber-500/35"
            >
              <AlertTriangle class="h-4 w-4" />
              <AlertTitle>Quota overage</AlertTitle>
              <AlertDescription>
                This deployment exceeds the licensed quota. Normal app usage is
                paused until enough organizations, users, or WhatsApp endpoints
                are deleted.
              </AlertDescription>
            </Alert>

            <Alert
              v-else-if="licenseStore.showExpiryWarning"
              variant="info"
              class="border-primary/25"
            >
              <ShieldCheck class="h-4 w-4" />
              <AlertTitle>License expires soon</AlertTitle>
              <AlertDescription>
                {{
                  licenseStore.daysUntilExpiry === 0
                    ? "This license expires today."
                    : `This license expires in ${licenseStore.daysUntilExpiry} day${licenseStore.daysUntilExpiry === 1 ? "" : "s"}.`
                }}
              </AlertDescription>
            </Alert>

            <div class="grid gap-4 sm:grid-cols-2 2xl:grid-cols-3">
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
                    {{ item.usage.current }}/{{
                      getQuotaLimitLabel(item.usage.limit)
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
                        item.usage.limit,
                        item.usage.over_quota,
                        item.usage.overage,
                      )
                    }}
                  </p>
                </CardContent>
              </Card>
            </div>

            <div
              v-if="licenseStore.state.usage.organization_details.length > 0"
              class="space-y-3 rounded-2xl border border-border/70 bg-background/70 p-4"
            >
              <div
                class="flex items-center gap-2 text-sm font-medium text-foreground"
              >
                <CheckCircle2 class="h-4 w-4 text-emerald-500" />
                Current organization usage
              </div>
              <div class="space-y-2">
                <div
                  v-for="org in licenseStore.state.usage.organization_details"
                  :key="org.organization_id"
                  class="flex flex-col gap-1 rounded-xl border border-border/60 bg-card/80 px-3 py-2 text-sm xl:flex-row xl:items-center xl:justify-between"
                >
                  <span class="font-medium text-foreground">
                    {{ org.organization_name }}
                  </span>
                  <span class="text-muted-foreground">
                    {{ org.user_count }} users,
                    {{ org.whatsapp_endpoint_count }}
                    endpoints
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="bg-background/90 p-6 sm:p-8 xl:p-10">
          <div
            class="mx-auto flex h-full max-w-xl flex-col justify-between gap-6 xl:max-w-none"
          >
            <div class="space-y-6">
              <div class="space-y-2">
                <p
                  class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
                >
                  First-start activation
                </p>
                <h2
                  class="text-2xl font-semibold tracking-tight text-foreground"
                >
                  Server identity and security key
                </h2>
              </div>

              <div class="space-y-3">
                <Label for="hwid" class="text-foreground/80">HWID</Label>
                <div class="flex flex-col gap-2 sm:flex-row">
                  <Input
                    id="hwid"
                    :model-value="licenseStore.state.hwid_full"
                    readonly
                    class="font-mono text-[11px] tracking-[0.02em]"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    class="shrink-0 sm:self-start"
                    :disabled="
                      bootstrapPending || !licenseStore.state.hwid_full
                    "
                    @click="copyHWID"
                  >
                    <Copy class="mr-2 h-4 w-4" />
                    Copy
                  </Button>
                </div>
                <p class="text-xs text-muted-foreground">
                  Short ID: {{ licenseStore.state.hwid_short || "Pending..." }}
                </p>
              </div>

              <div class="space-y-3">
                <Label for="security-key" class="text-foreground/80">
                  Security Key
                </Label>
                <Textarea
                  id="security-key"
                  v-model="securityKey"
                  :rows="8"
                  :disabled="licenseStore.activationPending"
                  class="min-h-[220px] font-mono text-xs"
                  placeholder="Paste the signed offline license token"
                />
              </div>

              <Alert
                v-if="licenseStore.error"
                variant="destructive"
                class="border-destructive/35"
              >
                <LockKeyhole class="h-4 w-4" />
                <AlertTitle>Activation blocked</AlertTitle>
                <AlertDescription>{{ licenseStore.error }}</AlertDescription>
              </Alert>
            </div>

            <div class="space-y-3">
              <Button
                class="w-full"
                :disabled="licenseStore.activationPending || bootstrapPending"
                @click="handleActivationSubmit"
              >
                <Loader2
                  v-if="licenseStore.activationPending"
                  class="mr-2 h-4 w-4 animate-spin"
                />
                <KeyRound v-else class="mr-2 h-4 w-4" />
                Activate
              </Button>
              <Button
                type="button"
                variant="outline"
                class="w-full"
                :disabled="bootstrapPending"
                @click="loadBootstrap(true)"
              >
                <Loader2
                  v-if="bootstrapPending"
                  class="mr-2 h-4 w-4 animate-spin"
                />
                Refresh status
              </Button>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>
