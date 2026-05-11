<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Badge } from "@/components/ui/badge";
import {
  Bot,
  Workflow,
  Brain,
  MessageSquare,
  Timer,
  UserPlus,
  ArrowRightLeft,
  CheckCheck,
  Zap,
  KeyRound,
  FileText,
  CalendarClock,
  AlertTriangle,
  Settings,
  Sparkles,
  Shield,
} from "lucide-vue-next";

const { t } = useI18n();

const chatbotFeatures = computed(() => [
  {
    icon: Workflow,
    title: t("landingChatbot.flowsTitle"),
    description: t("landingChatbot.flowsDesc"),
    details: [
      t("landingChatbot.flowsD1"),
      t("landingChatbot.flowsD2"),
      t("landingChatbot.flowsD3"),
      t("landingChatbot.flowsD4"),
    ],
  },
  {
    icon: Brain,
    title: t("landingChatbot.aiTitle"),
    description: t("landingChatbot.aiDesc"),
    details: [
      t("landingChatbot.aiD1"),
      t("landingChatbot.aiD2"),
      t("landingChatbot.aiD3"),
    ],
  },
  {
    icon: KeyRound,
    title: t("landingChatbot.keywordsTitle"),
    description: t("landingChatbot.keywordsDesc"),
    details: [
      t("landingChatbot.keywordsD1"),
      t("landingChatbot.keywordsD2"),
      t("landingChatbot.keywordsD3"),
    ],
  },
  {
    icon: ArrowRightLeft,
    title: t("landingChatbot.transfersTitle"),
    description: t("landingChatbot.transfersDesc"),
    details: [
      t("landingChatbot.transfersD1"),
      t("landingChatbot.transfersD2"),
      t("landingChatbot.transfersD3"),
    ],
  },
  {
    icon: Timer,
    title: t("landingChatbot.slaTitle"),
    description: t("landingChatbot.slaDesc"),
    details: [
      t("landingChatbot.slaD1"),
      t("landingChatbot.slaD2"),
      t("landingChatbot.slaD3"),
    ],
  },
  {
    icon: CalendarClock,
    title: t("landingChatbot.hoursTitle"),
    description: t("landingChatbot.hoursDesc"),
    details: [
      t("landingChatbot.hoursD1"),
      t("landingChatbot.hoursD2"),
      t("landingChatbot.hoursD3"),
    ],
  },
  {
    icon: AlertTriangle,
    title: t("landingChatbot.inactivityTitle"),
    description: t("landingChatbot.inactivityDesc"),
    details: [
      t("landingChatbot.inactivityD1"),
      t("landingChatbot.inactivityD2"),
      t("landingChatbot.inactivityD3"),
    ],
  },
  {
    icon: Settings,
    title: t("landingChatbot.configTitle"),
    description: t("landingChatbot.configDesc"),
    details: [
      t("landingChatbot.configD1"),
      t("landingChatbot.configD2"),
      t("landingChatbot.configD3"),
    ],
  },
]);

const quickCapabilities = computed(() => [
  { icon: Workflow, label: t("landingChatbot.capFlows") },
  { icon: Brain, label: t("landingChatbot.capAI") },
  { icon: KeyRound, label: t("landingChatbot.capKeywords") },
  { icon: ArrowRightLeft, label: t("landingChatbot.capTransfers") },
  { icon: Timer, label: t("landingChatbot.capSLA") },
  { icon: CalendarClock, label: t("landingChatbot.capHours") },
  { icon: AlertTriangle, label: t("landingChatbot.capInactivity") },
  { icon: Zap, label: t("landingChatbot.capSession") },
  { icon: MessageSquare, label: t("landingChatbot.capGreeting") },
  { icon: FileText, label: t("landingChatbot.capTemplates") },
  { icon: UserPlus, label: t("landingChatbot.capPickup") },
  { icon: Shield, label: t("landingChatbot.capFallback") },
  { icon: Sparkles, label: t("landingChatbot.capContext") },
  { icon: Settings, label: t("landingChatbot.capInspector") },
  { icon: Bot, label: t("landingChatbot.capToggle") },
  { icon: CheckCheck, label: t("landingChatbot.capValidation") },
]);
</script>

<template>
  <section class="border-t border-border/50 bg-muted/20 py-20 sm:py-28">
    <div class="mx-auto max-w-6xl px-4 sm:px-6">
      <div class="mx-auto max-w-2xl text-center">
        <Badge variant="outline" class="mb-4 gap-1.5 border-purple-500/25 bg-purple-500/10 px-3 py-1 text-xs text-purple-600">
          <Bot class="h-3 w-3" />
          {{ $t("landingChatbot.badge") }}
        </Badge>
        <h2 class="text-2xl font-bold tracking-tight sm:text-3xl">
          {{ $t("landingChatbot.title") }}
        </h2>
        <p class="mt-3 text-sm leading-7 text-muted-foreground sm:text-base">
          {{ $t("landingChatbot.subtitle") }}
        </p>
      </div>

      <div class="mt-12 grid gap-5 sm:mt-16 md:grid-cols-2">
        <div
          v-for="feature in chatbotFeatures"
          :key="feature.title"
          class="group relative rounded-[calc(var(--radius)+0.5rem)] border border-border bg-card p-6 transition-all duration-200 hover:border-purple-500/20 hover:shadow-md"
        >
          <div class="flex items-start gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-purple-500/10">
              <component :is="feature.icon" class="h-5 w-5 text-purple-600 dark:text-purple-400" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-sm font-semibold text-foreground">{{ feature.title }}</h3>
              <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ feature.description }}</p>
              <ul class="mt-3 space-y-1.5">
                <li
                  v-for="detail in feature.details"
                  :key="detail"
                  class="flex items-start gap-2 text-xs leading-5 text-muted-foreground"
                >
                  <CheckCheck class="mt-0.5 h-3 w-3 shrink-0 text-purple-500/70" />
                  <span>{{ detail }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div
            class="absolute inset-x-0 bottom-0 h-px mx-4 bg-gradient-to-r from-transparent via-purple-500/40 to-transparent opacity-0 transition-opacity duration-200 group-hover:opacity-100"
          />
        </div>
      </div>

      <div class="mt-16">
        <h3 class="text-center text-lg font-semibold text-foreground">
          {{ $t("landingChatbot.capabilitiesTitle") }}
        </h3>
        <p class="mt-2 text-center text-sm text-muted-foreground">
          {{ $t("landingChatbot.capabilitiesSubtitle") }}
        </p>
        <div
          class="mt-8 grid grid-cols-3 gap-2.5 sm:grid-cols-4 md:grid-cols-4 lg:grid-cols-8"
        >
          <div
            v-for="cap in quickCapabilities"
            :key="cap.label"
            class="flex flex-col items-center gap-2 rounded-[calc(var(--radius)+0.25rem)] border border-border bg-card p-3 text-center transition-colors hover:border-purple-500/20 hover:bg-accent/50"
          >
            <component :is="cap.icon" class="h-4 w-4 text-purple-500/80" />
            <span class="text-[11px] font-medium leading-tight text-foreground">{{ cap.label }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
