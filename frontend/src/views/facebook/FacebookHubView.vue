<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PageHeader } from "@/components/shared";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { moduleKeyForPath } from "@/modules/registry";
import { useConfigStore } from "@/stores/config";
import {
  Facebook,
  MessageCircle,
  Search,
  UserCheck,
  UsersRound,
  ThumbsUp,
  Database,
  Share2,
  Target,
  UserRoundCog,
  ArrowRight,
  ArrowLeft,
  Flame
} from "lucide-vue-next";

const { t, locale } = useI18n();
const router = useRouter();
const configStore = useConfigStore();
const isRTL = computed(() => locale.value === "ar");

// Defined list of Facebook tools to show in the grid
const tools = [
  {
    nameKey: "nav.facebookComments",
    descKey: "nav.facebookCommentsDesc",
    path: "/facebook/comments",
    icon: MessageCircle,
    color: "from-blue-500/10 to-indigo-500/10 dark:from-blue-500/20 dark:to-indigo-500/20 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(59,130,246,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(59,130,246,0.2)]",
    active: true
  },
  {
    nameKey: "nav.facebookPageSearch",
    descKey: "nav.facebookPageSearchDesc",
    path: "/facebook/page-search",
    icon: Search,
    color: "from-emerald-500/10 to-teal-500/10 dark:from-emerald-500/20 dark:to-teal-500/20 text-emerald-600 dark:text-emerald-400 border-emerald-200 dark:border-emerald-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(16,185,129,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(16,185,129,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookPeopleSearch",
    descKey: "nav.facebookPeopleSearchDesc",
    path: "/facebook/people-search",
    icon: UserCheck,
    color: "from-cyan-500/10 to-blue-500/10 dark:from-cyan-500/20 dark:to-blue-500/20 text-cyan-600 dark:text-cyan-400 border-cyan-200 dark:border-cyan-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(6,182,212,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(6,182,212,0.2)]",
    active: true
  },
  {
    nameKey: "nav.facebookGroupSearch",
    descKey: "nav.facebookGroupSearchDesc",
    path: "/facebook/group-search",
    icon: UsersRound,
    color: "from-purple-500/10 to-pink-500/10 dark:from-purple-500/20 dark:to-pink-500/20 text-purple-600 dark:text-purple-400 border-purple-200 dark:border-purple-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(168,85,247,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(168,85,247,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookExtractLikes",
    descKey: "nav.facebookExtractLikesDesc",
    path: "/facebook/extract-likes",
    icon: ThumbsUp,
    color: "from-amber-500/10 to-orange-500/10 dark:from-amber-500/20 dark:to-orange-500/20 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(245,158,11,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(245,158,11,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookPageMessengers",
    descKey: "nav.facebookPageMessengersDesc",
    path: "/facebook/page-messengers",
    icon: MessageCircle,
    color: "from-pink-500/10 to-rose-500/10 dark:from-pink-500/20 dark:to-rose-500/20 text-pink-600 dark:text-pink-400 border-pink-200 dark:border-pink-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(236,72,153,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(236,72,153,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookExtractData",
    descKey: "nav.facebookExtractDataDesc",
    path: "/facebook/extract-data",
    icon: Database,
    color: "from-violet-500/10 to-fuchsia-500/10 dark:from-violet-500/20 dark:to-fuchsia-500/20 text-violet-600 dark:text-violet-400 border-violet-200 dark:border-violet-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(139,92,246,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(139,92,246,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookAutoShare",
    descKey: "nav.facebookAutoShareDesc",
    path: "/facebook/auto-share",
    icon: Share2,
    color: "from-sky-500/10 to-indigo-500/10 dark:from-sky-500/20 dark:to-indigo-500/20 text-sky-600 dark:text-sky-400 border-sky-200 dark:border-sky-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(14,165,233,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(14,165,233,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookRetargeting",
    descKey: "nav.facebookRetargetingDesc",
    path: "/facebook/retargeting",
    icon: Target,
    color: "from-red-500/10 to-orange-500/10 dark:from-red-500/20 dark:to-orange-500/20 text-red-600 dark:text-red-400 border-red-200 dark:border-red-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(239,68,68,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(239,68,68,0.2)]",
    active: false
  },
  {
    nameKey: "nav.facebookAccounts",
    descKey: "nav.facebookAccountsDesc",
    path: "/facebook/accounts",
    icon: UserRoundCog,
    color: "from-blue-600/10 to-cyan-600/10 dark:from-blue-600/20 dark:to-cyan-600/20 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-500/20",
    hoverGlow: "group-hover:shadow-[0_0_30px_rgba(37,99,235,0.15)] dark:group-hover:shadow-[0_0_30px_rgba(37,99,235,0.2)]",
    active: true
  }
];

// Only show tools whose managed module is effective-enabled for the current
// organization. moduleKeyForPath is the single source of truth for the
// path → module-key mapping (see @/modules/registry), so this stays in sync
// with the sidebar gating in AppLayout.vue automatically.
const visibleTools = computed(() =>
  tools.filter((tool) => {
    const moduleKey = moduleKeyForPath(tool.path);
    return moduleKey === undefined || configStore.isModuleEnabled(moduleKey);
  }),
);

function navigateTo(path: string) {
  router.push(path);
}
</script>

<template>
  <div :dir="isRTL ? 'rtl' : 'ltr'" class="relative min-h-screen bg-background text-foreground flex flex-col justify-between overflow-hidden">
    <!-- Glowing background decorative meshes -->
    <div class="absolute -top-40 -left-40 w-96 h-96 bg-blue-600/10 rounded-full blur-3xl pointer-events-none animate-pulse"></div>
    <div class="absolute -bottom-40 -right-40 w-[24rem] h-[24rem] bg-indigo-600/10 rounded-full blur-3xl pointer-events-none animate-pulse duration-5000"></div>

    <div class="flex-1 flex flex-col p-4 md:p-8 relative z-10 max-w-7xl w-full mx-auto pb-24">
      <PageHeader
        :title="t('nav.facebookTools')"
        :subtitle="isRTL ? 'إدارة ونمو قنوات التواصل الاجتماعي الخاصة بك على فيسبوك من لوحة تحكم واحدة' : 'Scale and manage your Facebook social marketing tools from one place'"
        :icon="Facebook"
        icon-gradient="bg-gradient-to-br from-blue-600 to-indigo-700 shadow-blue-500/20"
        :breadcrumbs="[
          { label: t('nav.facebookTools') }
        ]"
      />

      <!-- Grid of Facebook Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-8">
        <Card
          v-for="tool in visibleTools"
          :key="tool.path"
          class="group relative overflow-hidden bg-card/40 dark:bg-slate-900/40 border-border/80 dark:border-slate-800/80 backdrop-blur-xl shadow-xl transition-all duration-300 hover:border-slate-300 dark:hover:border-slate-700 hover:-translate-y-1 hover:shadow-2xl cursor-pointer rounded-2xl flex flex-col justify-between"
          :class="tool.hoverGlow"
          @click="navigateTo(tool.path)"
        >
          <!-- Accent top line gradient -->
          <div class="absolute top-0 inset-x-0 h-[2px] bg-gradient-to-r from-blue-500 via-indigo-500 to-purple-500 opacity-60 group-hover:opacity-100 transition-opacity"></div>
          
          <CardHeader class="p-6 pb-2">
            <div class="flex items-start justify-between">
              <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br border shadow-md transition-transform duration-300 group-hover:scale-110" :class="tool.color">
                <component :is="tool.icon" class="h-6 w-6" />
              </div>
              <Badge v-if="tool.active" variant="outline" class="bg-emerald-500/10 dark:bg-emerald-950/40 border-emerald-200 dark:border-emerald-500/30 text-emerald-600 dark:text-emerald-400">
                {{ isRTL ? 'نشط' : 'Active' }}
              </Badge>
              <Badge v-else variant="outline" class="bg-muted dark:bg-slate-950/40 border-border dark:border-slate-800 text-muted-foreground dark:text-slate-500">
                {{ isRTL ? 'قريباً' : 'Beta' }}
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
            
            <div class="mt-6 flex items-center justify-end text-xs font-semibold text-blue-600 dark:text-blue-400 group-hover:text-blue-700 dark:group-hover:text-blue-300 transition-colors gap-1">
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
        <span>Whatomate Unified Social Platform</span>
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
