<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PageHeader } from "@/components/shared";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Smartphone,
  Contact,
  MessageSquareText,
  BookOpen,
  Archive,
  Route,
  ArrowRight,
  ArrowLeft,
  Flame
} from "lucide-vue-next";

const { t, locale } = useI18n();
const router = useRouter();
const isRTL = computed(() => locale.value === "ar");

// Defined list of WhatsApp tools to show in the grid
const tools = [
  {
    nameKey: "nav.whatsappAccounts",
    descKey: "nav.whatsappAccountsDesc",
    path: "/whatsapp/instances",
    icon: Smartphone,
    color: "from-emerald-500/10 to-teal-500/10 dark:from-emerald-500/20 dark:to-teal-500/20 text-emerald-600 dark:text-emerald-400 border-emerald-200 dark:border-emerald-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(16,185,129,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(16,185,129,0.2)]",
    active: true
  },
  {
    nameKey: "nav.contacts",
    descKey: "nav.contactsDesc",
    path: "/whatsapp/contacts",
    icon: Contact,
    color: "from-cyan-500/10 to-blue-500/10 dark:from-cyan-500/20 dark:to-blue-500/20 text-cyan-600 dark:text-cyan-400 border-cyan-200 dark:border-cyan-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(6,182,212,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(6,182,212,0.2)]",
    active: true
  },
  {
    nameKey: "nav.cannedResponses",
    descKey: "nav.cannedResponsesDesc",
    path: "/whatsapp/canned-responses",
    icon: MessageSquareText,
    color: "from-purple-500/10 to-pink-500/10 dark:from-purple-500/20 dark:to-pink-500/20 text-purple-600 dark:text-purple-400 border-purple-200 dark:border-purple-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(168,85,247,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(168,85,247,0.2)]",
    active: true
  },
  {
    nameKey: "nav.savedContents",
    descKey: "nav.savedContentsDesc",
    path: "/whatsapp/saved-contents",
    icon: BookOpen,
    color: "from-amber-500/10 to-orange-500/10 dark:from-amber-500/20 dark:to-orange-500/20 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(245,158,11,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(245,158,11,0.2)]",
    active: true
  },
  {
    nameKey: "nav.closedChats",
    descKey: "nav.closedChatsDesc",
    path: "/whatsapp/closed-chats",
    icon: Archive,
    color: "from-pink-500/10 to-rose-500/10 dark:from-pink-500/20 dark:to-rose-500/20 text-pink-600 dark:text-pink-400 border-pink-200 dark:border-pink-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(236,72,153,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(236,72,153,0.2)]",
    active: true
  },
  {
    nameKey: "nav.extract",
    descKey: "nav.extractDesc",
    path: "/whatsapp/extract",
    icon: MessageSquareText,
    color: "from-yellow-500/10 to-amber-500/10 dark:from-yellow-500/20 dark:to-amber-500/20 text-yellow-600 dark:text-yellow-400 border-yellow-200 dark:border-yellow-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(234,179,8,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(234,179,8,0.2)]",
    active: true
  },
  {
    nameKey: "nav.customerRouting",
    descKey: "nav.customerRoutingDesc",
    path: "/whatsapp/agent-selection",
    icon: Route,
    color: "from-indigo-500/10 to-purple-500/10 dark:from-indigo-500/20 dark:to-purple-500/20 text-indigo-600 dark:text-indigo-400 border-indigo-200 dark:border-indigo-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(99,102,241,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(99,102,241,0.2)]",
    active: true
  }
];

function navigateTo(path: string) {
  router.push(path);
}
</script>

<template>
  <div :dir="isRTL ? 'rtl' : 'ltr'" class="relative min-h-screen bg-background text-foreground flex flex-col justify-between overflow-hidden">
    <!-- Glowing background decorative meshes -->
    <div class="absolute -top-40 -left-40 w-96 h-96 bg-emerald-600/10 rounded-full blur-3xl pointer-events-none animate-pulse"></div>
    <div class="absolute -bottom-40 -right-40 w-[24rem] h-[24rem] bg-teal-600/10 rounded-full blur-3xl pointer-events-none animate-pulse duration-5000"></div>

    <div class="flex-1 flex flex-col p-4 md:p-8 relative z-10 max-w-7xl w-full mx-auto pb-24">
      <PageHeader
        :title="t('nav.whatsappTools')"
        :subtitle="isRTL ? 'إدارة وتحسين عمليات التواصل عبر قنوات واتساب والتحكم التام في إعداداتها' : 'Manage and optimize your WhatsApp communications and configuration settings'"
        :icon="Smartphone"
        icon-gradient="bg-gradient-to-br from-emerald-600 to-teal-700 shadow-emerald-500/20"
        :breadcrumbs="[
          { label: t('nav.whatsappTools') }
        ]"
      />

      <!-- Grid of WhatsApp Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-8">
        <Card
          v-for="tool in tools"
          :key="tool.path"
          class="group relative overflow-hidden bg-card/40 dark:bg-slate-900/40 border-border/80 dark:border-slate-800/80 backdrop-blur-xl shadow-xl transition-all duration-300 hover:border-slate-300 dark:hover:border-slate-700 hover:-translate-y-1 hover:shadow-2xl cursor-pointer rounded-2xl flex flex-col justify-between"
          :class="tool.hoverGlow"
          @click="navigateTo(tool.path)"
        >
          <!-- Accent top line gradient -->
          <div class="absolute top-0 inset-x-0 h-[2px] bg-gradient-to-r from-emerald-500 via-teal-500 to-indigo-500 opacity-60 group-hover:opacity-100 transition-opacity"></div>
          
          <CardHeader class="p-6 pb-2">
            <div class="flex items-start justify-between">
              <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br border shadow-md transition-transform duration-300 group-hover:scale-110" :class="tool.color">
                <component :is="tool.icon" class="h-6 w-6" />
              </div>
              <Badge variant="outline" class="bg-emerald-500/10 dark:bg-emerald-950/40 border-emerald-200 dark:border-emerald-500/30 text-emerald-600 dark:text-emerald-400">
                {{ isRTL ? 'نشط' : 'Active' }}
              </Badge>
            </div>
            
            <CardTitle class="text-base font-bold mt-4 tracking-tight text-foreground/90 group-hover:text-foreground transition-colors">
              {{ t(tool.nameKey) }}
            </CardTitle>
          </CardHeader>
          
          <CardContent class="p-6 pt-2 flex-1 flex flex-col justify-between">
            <p class="text-xs text-muted-foreground group-hover:text-foreground/80 transition-colors leading-relaxed">
              {{ t(tool.descKey) }}
            </p>
            
            <div class="mt-6 flex items-center justify-end text-xs font-semibold text-emerald-600 dark:text-emerald-400 group-hover:text-emerald-700 dark:group-hover:text-emerald-300 transition-colors gap-1">
              <span>{{ isRTL ? 'فتح الأداة' : 'Open Tool' }}</span>
              <component :is="isRTL ? ArrowLeft : ArrowRight" class="h-3.5 w-3.5 transition-transform duration-300 group-hover:translate-x-1" :class="isRTL && 'group-hover:-translate-x-1'" />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <!-- Bottom Footer Meta -->
    <div class="relative z-10 flex items-center justify-between border-t border-border/60 p-4 mx-4 md:mx-8 text-xs text-muted-foreground">
      <div class="flex items-center gap-1.5">
        <Flame class="h-4 w-4 text-orange-500" />
        <span>Whatomate Unified WhatsApp Platform</span>
      </div>
      <span>© 2026 Whatomate Inc.</span>
    </div>
  </div>
</template>

<style scoped>
.animate-pulse {
  animation: pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
@keyframes pulse {
  0%, 100% {
    opacity: 0.1;
  }
  50% {
    opacity: 0.3;
  }
}
.duration-5000 {
  animation-duration: 5s;
}
</style>
