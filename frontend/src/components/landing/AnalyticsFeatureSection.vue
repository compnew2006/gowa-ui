<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  BarChart3,
  LayoutDashboard,
  Users,
  Bot,
  Star,
  TrendingUp,
  FileSpreadsheet,
  PieChart,
  CheckCheck,
  Activity,
  Download,
  MessageSquare,
  Megaphone,
  ArrowRightLeft,
  Settings,
  Zap,
} from "lucide-vue-next";

const { t } = useI18n();

const analyticsFeatures = computed(() => [
  { icon: LayoutDashboard, title: t("landingAnalytics.dashboardTitle"), description: t("landingAnalytics.dashboardDesc"), details: [t("landingAnalytics.dashboardD1"), t("landingAnalytics.dashboardD2"), t("landingAnalytics.dashboardD3"), t("landingAnalytics.dashboardD4")] },
  { icon: Users, title: t("landingAnalytics.agentTitle"), description: t("landingAnalytics.agentDesc"), details: [t("landingAnalytics.agentD1"), t("landingAnalytics.agentD2"), t("landingAnalytics.agentD3")] },
  { icon: Bot, title: t("landingAnalytics.chatbotTitle"), description: t("landingAnalytics.chatbotDesc"), details: [t("landingAnalytics.chatbotD1"), t("landingAnalytics.chatbotD2"), t("landingAnalytics.chatbotD3")] },
  { icon: Star, title: t("landingAnalytics.csatTitle"), description: t("landingAnalytics.csatDesc"), details: [t("landingAnalytics.csatD1"), t("landingAnalytics.csatD2"), t("landingAnalytics.csatD3"), t("landingAnalytics.csatD4")] },
  { icon: TrendingUp, title: t("landingAnalytics.messagesTitle"), description: t("landingAnalytics.messagesDesc"), details: [t("landingAnalytics.messagesD1"), t("landingAnalytics.messagesD2"), t("landingAnalytics.messagesD3")] },
  { icon: FileSpreadsheet, title: t("landingAnalytics.exportTitle"), description: t("landingAnalytics.exportDesc"), details: [t("landingAnalytics.exportD1"), t("landingAnalytics.exportD2"), t("landingAnalytics.exportD3")] },
  { icon: PieChart, title: t("landingAnalytics.metaTitle"), description: t("landingAnalytics.metaDesc"), details: [t("landingAnalytics.metaD1"), t("landingAnalytics.metaD2"), t("landingAnalytics.metaD3")] },
  { icon: Activity, title: t("landingAnalytics.realtimeTitle"), description: t("landingAnalytics.realtimeDesc"), details: [t("landingAnalytics.realtimeD1"), t("landingAnalytics.realtimeD2"), t("landingAnalytics.realtimeD3")] },
]);

const quickCapabilities = computed(() => [
  { icon: LayoutDashboard, label: t("landingAnalytics.capWidgets") },
  { icon: BarChart3, label: t("landingAnalytics.capCharts") },
  { icon: Users, label: t("landingAnalytics.capAgentCompare") },
  { icon: Star, label: t("landingAnalytics.capCSAT") },
  { icon: TrendingUp, label: t("landingAnalytics.capMessageVol") },
  { icon: Bot, label: t("landingAnalytics.capChatbotStats") },
  { icon: FileSpreadsheet, label: t("landingAnalytics.capCSVExport") },
  { icon: PieChart, label: t("landingAnalytics.capMetaInsights") },
  { icon: MessageSquare, label: t("landingAnalytics.capContactStats") },
  { icon: Megaphone, label: t("landingAnalytics.capCampaignStats") },
  { icon: ArrowRightLeft, label: t("landingAnalytics.capTransferStats") },
  { icon: Settings, label: t("landingAnalytics.capCustomLayout") },
  { icon: Download, label: t("landingAnalytics.capDataSources") },
  { icon: Zap, label: t("landingAnalytics.capLiveCounters") },
  { icon: Activity, label: t("landingAnalytics.perfMetrics") },
  { icon: BarChart3, label: t("landingAnalytics.capTrends") },
]);
</script>

<template>
  <section class="border-t border-border/40 py-24 sm:py-32">
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-5 gap-2 rounded-full border-teal-500/20 bg-teal-500/5 px-4 py-1.5 text-xs text-teal-600">
          <BarChart3 class="h-3.5 w-3.5" aria-hidden="true" />
          {{ $t("landingAnalytics.badge") }}
        </Badge>
        <h2 class="text-3xl font-extrabold tracking-tight sm:text-4xl">{{ $t("landingAnalytics.title") }}</h2>
        <p class="mt-4 text-base leading-7 text-muted-foreground sm:text-lg">{{ $t("landingAnalytics.subtitle") }}</p>
      </div>
      <div class="mt-16 grid gap-5 md:grid-cols-2">
        <div v-for="feature in analyticsFeatures" :key="feature.title" class="group relative rounded-xl border border-border bg-card p-6 transition-[border-color,box-shadow] duration-300 hover:border-teal-500/20 hover:shadow-lg hover:shadow-teal-500/5">
          <div class="flex items-start gap-4">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-teal-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-teal-600 dark:text-teal-400" aria-hidden="true" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-base font-semibold text-foreground">{{ feature.title }}</h3>
              <p class="mt-1.5 text-sm leading-6 text-muted-foreground">{{ feature.description }}</p>
              <ul class="mt-3.5 space-y-2">
                <li v-for="detail in feature.details" :key="detail" class="flex items-start gap-2.5 text-sm leading-5 text-muted-foreground">
                  <CheckCheck class="mt-0.5 h-3.5 w-3.5 shrink-0 text-teal-500/60" aria-hidden="true" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div class="absolute inset-x-0 bottom-0 h-0.5 rounded-b-xl bg-gradient-to-r from-transparent via-teal-500/40 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
        </div>
      </div>
      <div class="mt-20">
        <h3 class="text-center text-lg font-semibold text-foreground">{{ $t("landingAnalytics.capabilitiesTitle") }}</h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">{{ $t("landingAnalytics.capabilitiesSubtitle") }}</p>
        <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
          <div v-for="cap in quickCapabilities" :key="cap.label" class="flex items-center gap-2 rounded-full border border-border bg-card px-4 py-2 text-sm transition-[border-color,background-color] duration-200 hover:border-teal-500/20 hover:bg-accent/50">
            <component :is="cap.icon" class="h-4 w-4 text-teal-500/70" aria-hidden="true" />
            <span class="font-medium text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
