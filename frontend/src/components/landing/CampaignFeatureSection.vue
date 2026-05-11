<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  Megaphone,
  Pause,
  Play,
  Upload,
  Users,
  FileSpreadsheet,
  CheckCheck,
  CalendarClock,
  Timer,
  Shield,
  BarChart3,
  Zap,
  RefreshCw,
  AlertTriangle,
  UserPlus,
  Eye,
  MessageSquareText,
  FileText,
} from "lucide-vue-next";

const { t } = useI18n();

const campaignFeatures = computed(() => [
  {
    icon: FileSpreadsheet,
    title: t("landingCampaigns.createTitle"),
    description: t("landingCampaigns.createDesc"),
    details: [
      t("landingCampaigns.createD1"),
      t("landingCampaigns.createD2"),
      t("landingCampaigns.createD3"),
    ],
  },
  {
    icon: Upload,
    title: t("landingCampaigns.recipientsTitle"),
    description: t("landingCampaigns.recipientsDesc"),
    details: [
      t("landingCampaigns.recipientsD1"),
      t("landingCampaigns.recipientsD2"),
      t("landingCampaigns.recipientsD3"),
      t("landingCampaigns.recipientsD4"),
    ],
  },
  {
    icon: Play,
    title: t("landingCampaigns.executionTitle"),
    description: t("landingCampaigns.executionDesc"),
    details: [
      t("landingCampaigns.executionD1"),
      t("landingCampaigns.executionD2"),
      t("landingCampaigns.executionD3"),
      t("landingCampaigns.executionD4"),
    ],
  },
  {
    icon: BarChart3,
    title: t("landingCampaigns.trackingTitle"),
    description: t("landingCampaigns.trackingDesc"),
    details: [
      t("landingCampaigns.trackingD1"),
      t("landingCampaigns.trackingD2"),
      t("landingCampaigns.trackingD3"),
    ],
  },
  {
    icon: CalendarClock,
    title: t("landingCampaigns.schedulingTitle"),
    description: t("landingCampaigns.schedulingDesc"),
    details: [
      t("landingCampaigns.schedulingD1"),
      t("landingCampaigns.schedulingD2"),
      t("landingCampaigns.schedulingD3"),
    ],
  },
  {
    icon: Shield,
    title: t("landingCampaigns.policyTitle"),
    description: t("landingCampaigns.policyDesc"),
    details: [
      t("landingCampaigns.policyD1"),
      t("landingCampaigns.policyD2"),
      t("landingCampaigns.policyD3"),
      t("landingCampaigns.policyD4"),
    ],
  },
  {
    icon: Timer,
    title: t("landingCampaigns.rateLimitTitle"),
    description: t("landingCampaigns.rateLimitDesc"),
    details: [
      t("landingCampaigns.rateLimitD1"),
      t("landingCampaigns.rateLimitD2"),
      t("landingCampaigns.rateLimitD3"),
    ],
  },
  {
    icon: RefreshCw,
    title: t("landingCampaigns.recoveryTitle"),
    description: t("landingCampaigns.recoveryDesc"),
    details: [
      t("landingCampaigns.recoveryD1"),
      t("landingCampaigns.recoveryD2"),
    ],
  },
]);

const quickCapabilities = computed(() => [
  { icon: FileSpreadsheet, label: t("landingCampaigns.capCSV") },
  { icon: UserPlus, label: t("landingCampaigns.capContacts") },
  { icon: MessageSquareText, label: t("landingCampaigns.capTemplates") },
  { icon: FileText, label: t("landingCampaigns.capHeaderMedia") },
  { icon: Users, label: t("landingCampaigns.capParams") },
  { icon: BarChart3, label: t("landingCampaigns.capProgress") },
  { icon: Eye, label: t("landingCampaigns.capRecipients") },
  { icon: Pause, label: t("landingCampaigns.capPause") },
  { icon: RefreshCw, label: t("landingCampaigns.capRetry") },
  { icon: CalendarClock, label: t("landingCampaigns.capSchedule") },
  { icon: AlertTriangle, label: t("landingCampaigns.capInbound") },
  { icon: Zap, label: t("landingCampaigns.capAutoCreate") },
]);
</script>

<template>
  <section class="border-t border-border/40 bg-muted/30 py-24 sm:py-32">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-2xl text-center">
        <Badge
          variant="outline"
          class="mb-5 gap-2 rounded-full border-orange-500/20 bg-orange-500/5 px-4 py-1.5 text-xs text-orange-600"
        >
          <Megaphone class="h-3.5 w-3.5" aria-hidden="true" />
          {{ $t("landingCampaigns.badge") }}
        </Badge>
        <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">
          {{ $t("landingCampaigns.title") }}
        </h2>
        <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">
          {{ $t("landingCampaigns.subtitle") }}
        </p>
      </div>

      <div class="mt-16 grid gap-5 md:grid-cols-2">
        <div
          v-for="feature in campaignFeatures"
          :key="feature.title"
          class="group relative rounded-xl border border-border bg-card p-6 transition-[border-color,box-shadow] duration-300 hover:border-orange-500/20 hover:shadow-lg hover:shadow-orange-500/5"
        >
          <div class="flex items-start gap-4">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-orange-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-orange-600 dark:text-orange-400" aria-hidden="true" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-base font-semibold text-foreground">{{ feature.title }}</h3>
              <p class="mt-1.5 text-sm leading-6 text-muted-foreground">{{ feature.description }}</p>
              <ul class="mt-3.5 space-y-2">
                <li v-for="detail in feature.details" :key="detail" class="flex items-start gap-2.5 text-sm leading-5 text-muted-foreground">
                  <CheckCheck class="mt-0.5 h-3.5 w-3.5 shrink-0 text-orange-500/60" aria-hidden="true" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div class="absolute inset-x-0 bottom-0 h-0.5 rounded-b-xl bg-gradient-to-r from-transparent via-orange-500/40 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
        </div>
      </div>

      <div class="mt-20">
        <h3 class="text-center text-lg font-semibold text-foreground">{{ $t("landingCampaigns.capabilitiesTitle") }}</h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">{{ $t("landingCampaigns.capabilitiesSubtitle") }}</p>
        <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
          <div v-for="cap in quickCapabilities" :key="cap.label" class="flex items-center gap-2 rounded-full border border-border bg-card px-4 py-2 text-sm transition-[border-color,background-color] duration-200 hover:border-orange-500/20 hover:bg-accent/50">
            <component :is="cap.icon" class="h-4 w-4 text-orange-500/70" aria-hidden="true" />
            <span class="font-medium text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
