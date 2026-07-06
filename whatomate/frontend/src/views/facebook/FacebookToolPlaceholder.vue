<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ArrowRight, ArrowLeft } from "lucide-vue-next";

const props = defineProps<{
  titleKey: string;
  descriptionKey: string;
  icon: any;
}>();

const { t, locale } = useI18n();
const isRTL = computed(() => locale.value === "ar");
</script>

<template>
  <div :dir="isRTL ? 'rtl' : 'ltr'" class="min-h-[calc(100vh-3rem)] md:min-h-screen bg-background text-foreground p-4 md:p-8 flex flex-col justify-center">
    <div class="flex-1 flex items-center justify-center py-6 md:py-12">
      <Card class="w-full max-w-lg bg-card border-border shadow-lg relative overflow-hidden rounded-2xl">
        <CardHeader class="text-center pt-8 pb-4">
          <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <component :is="icon" class="h-7 w-7" />
          </div>
          <CardTitle class="text-xl font-bold mt-5 tracking-tight text-foreground">
            {{ t(titleKey) }}
          </CardTitle>
          <p class="text-muted-foreground text-sm mt-2 px-6 max-w-md mx-auto leading-relaxed">
            {{ t(descriptionKey) }}
          </p>
        </CardHeader>

        <CardContent class="px-6 md:px-8 pb-8 pt-2">
          <div class="mt-6 p-4 rounded-xl bg-muted/50 border border-border flex flex-col sm:flex-row items-center justify-between gap-4">
            <div class="flex items-center gap-3 text-center sm:text-left" :class="isRTL && 'sm:text-right flex-row-reverse'">
              <span class="relative flex h-3 w-3">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                <span class="relative inline-flex rounded-full h-3 w-3 bg-primary"></span>
              </span>
              <div>
                <span class="text-xs font-semibold text-foreground/80 block">
                  {{ t('common.comingSoon', 'Coming Soon') }}
                </span>
                <span class="text-[10px] text-muted-foreground block mt-0.5">
                  {{ t('facebook.comingSoonDesc', 'Be notified as soon as this tool launches.') }}
                </span>
              </div>
            </div>

            <Button class="rounded-xl px-5 py-2 flex items-center gap-2 group">
              <span>{{ t('facebook.notifyMe', 'Notify Me') }}</span>
              <component :is="isRTL ? ArrowLeft : ArrowRight" class="h-4 w-4 transition-transform group-hover:translate-x-1" :class="isRTL && 'group-hover:-translate-x-1'" />
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
