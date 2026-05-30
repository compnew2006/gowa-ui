<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Facebook, Sparkles, Activity, ShieldCheck, Flame, ArrowRight, ArrowLeft } from "lucide-vue-next";

const props = defineProps<{
  titleKey: string;
  descriptionKey: string;
  icon: any;
}>();

const { t, locale } = useI18n();
const isRTL = computed(() => locale.value === "ar");
</script>

<template>
  <div :dir="isRTL ? 'rtl' : 'ltr'" class="relative min-h-[calc(100vh-3rem)] md:min-h-screen bg-slate-950/40 text-slate-100 p-4 md:p-8 flex flex-col justify-between overflow-hidden">
    <!-- Dynamic Decorative Gradients for rich aesthetics -->
    <div class="absolute -top-40 -left-40 w-96 h-96 bg-blue-600/10 rounded-full blur-3xl pointer-events-none animate-pulse"></div>
    <div class="absolute -bottom-40 -right-40 w-[24rem] h-[24rem] bg-indigo-600/10 rounded-full blur-3xl pointer-events-none animate-pulse duration-5000"></div>

    <!-- Top Breadcrumb / Title Bar -->
    <div class="relative z-10 flex flex-wrap items-center justify-between gap-4 border-b border-slate-800/60 pb-4 mb-6">
      <div class="flex items-center gap-2">
        <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-600/10 border border-blue-500/20 text-blue-400">
          <Facebook class="h-5 w-5" />
        </div>
        <div>
          <h2 class="text-xs uppercase tracking-wider text-slate-400 font-semibold">{{ t('nav.facebookTools') }}</h2>
          <h1 class="text-lg font-bold bg-gradient-to-r from-blue-400 to-indigo-400 bg-clip-text text-transparent">
            {{ t(titleKey) }}
          </h1>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <Badge variant="outline" class="bg-blue-950/40 border-blue-500/30 text-blue-400 flex items-center gap-1.5 py-1 px-3 rounded-full">
          <Activity class="h-3 w-3 animate-pulse" />
          <span>v1.0 Premium</span>
        </Badge>
      </div>
    </div>

    <!-- Main Content Area -->
    <div class="relative z-10 flex-1 flex items-center justify-center py-6 md:py-12">
      <Card class="w-full max-w-2xl bg-slate-900/40 border-slate-800/80 backdrop-blur-xl shadow-2xl relative overflow-hidden rounded-2xl">
        <!-- Accent border at the top of the card -->
        <div class="absolute top-0 inset-x-0 h-[2px] bg-gradient-to-r from-blue-500 via-indigo-500 to-purple-500"></div>

        <CardHeader class="text-center pt-8 pb-4">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500/10 to-indigo-500/10 border border-blue-500/20 text-blue-400 shadow-lg shadow-blue-500/5 animate-bounce duration-3000">
            <component :is="icon" class="h-8 w-8" />
          </div>
          <CardTitle class="text-2xl font-bold mt-6 tracking-tight bg-gradient-to-r from-white via-slate-100 to-slate-300 bg-clip-text text-transparent">
            {{ t(titleKey) }}
          </CardTitle>
          <p class="text-slate-400 text-sm mt-3 px-6 max-w-md mx-auto leading-relaxed">
            {{ t(descriptionKey) }}
          </p>
        </CardHeader>

        <CardContent class="px-6 md:px-8 pb-8 pt-2">
          <!-- Glassmorphic Feature Grid Mockup -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <div class="flex items-start gap-3 p-4 rounded-xl bg-slate-950/30 border border-slate-800/50 hover:border-slate-700/50 transition-colors duration-200">
              <div class="h-8 w-8 rounded-lg bg-blue-500/10 flex items-center justify-center text-blue-400 shrink-0">
                <Sparkles class="h-4 w-4" />
              </div>
              <div>
                <h4 class="text-sm font-semibold text-slate-200">
                  {{ isRTL ? 'تحليل ذكي فائق' : 'AI-Powered Insights' }}
                </h4>
                <p class="text-xs text-slate-500 mt-1">
                  {{ isRTL ? 'استخراج وتحليل البيانات بالذكاء الاصطناعي بدقة عالية.' : 'Extract and dissect audience parameters with smart heuristics.' }}
                </p>
              </div>
            </div>

            <div class="flex items-start gap-3 p-4 rounded-xl bg-slate-950/30 border border-slate-800/50 hover:border-slate-700/50 transition-colors duration-200">
              <div class="h-8 w-8 rounded-lg bg-indigo-500/10 flex items-center justify-center text-indigo-400 shrink-0">
                <ShieldCheck class="h-4 w-4" />
              </div>
              <div>
                <h4 class="text-sm font-semibold text-slate-200">
                  {{ isRTL ? 'أمن وخصوصية تامة' : 'Safe & Encrypted' }}
                </h4>
                <p class="text-xs text-slate-500 mt-1">
                  {{ isRTL ? 'حماية بياناتك وحساباتك بنظام تشفير عسكري متقدم.' : 'Advanced endpoint encryption guaranteeing full safety.' }}
                </p>
              </div>
            </div>
          </div>

          <!-- Feature coming soon / action bar -->
          <div class="mt-8 p-4 rounded-2xl bg-gradient-to-br from-blue-900/10 to-indigo-900/10 border border-blue-500/10 flex flex-col sm:flex-row items-center justify-between gap-4">
            <div class="flex items-center gap-3 text-center sm:text-left" :class="isRTL && 'sm:text-right flex-row-reverse'">
              <span class="relative flex h-3 w-3">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-3 w-3 bg-blue-500"></span>
              </span>
              <div>
                <span class="text-xs font-semibold text-slate-300 block">
                  {{ isRTL ? 'الميزة قيد التطوير والتحضير' : 'Feature Under Development' }}
                </span>
                <span class="text-[10px] text-slate-500 block mt-0.5">
                  {{ isRTL ? 'احصل على إشعار فور إطلاقها رسمياً.' : 'Be notified as soon as this tool launches.' }}
                </span>
              </div>
            </div>

            <Button class="bg-blue-600 hover:bg-blue-500 text-white font-medium rounded-xl px-5 py-2 flex items-center gap-2 group transition-all duration-300 shadow-lg shadow-blue-600/20">
              <span>{{ isRTL ? 'تفعيل الإشعارات' : 'Notify Me' }}</span>
              <component :is="isRTL ? ArrowLeft : ArrowRight" class="h-4 w-4 transition-transform group-hover:translate-x-1" :class="isRTL && 'group-hover:-translate-x-1'" />
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Bottom Footer Meta -->
    <div class="relative z-10 flex items-center justify-between border-t border-slate-800/60 pt-4 mt-6 text-xs text-slate-500">
      <div class="flex items-center gap-1.5">
        <Flame class="h-4 w-4 text-orange-500" />
        <span>Whatomate Unified Platform</span>
      </div>
      <span>© 2026 Whatomate Inc.</span>
    </div>
  </div>
</template>

<style scoped>
.duration-3000 {
  animation-duration: 3s;
}
.duration-5000 {
  animation-duration: 5s;
}
</style>
