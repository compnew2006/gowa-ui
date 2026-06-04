<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { ArrowLeft } from "lucide-vue-next";
import type { Component } from "vue";

defineProps<{
  title: string;
  description?: string;
  subtitle?: string;
  icon?: Component;
  iconGradient?: string;
  backLink?: string;
  breadcrumbs?: Array<{ label: string; href?: string }>;
}>();
</script>

<template>
  <header
    class="border-b border-border bg-card/95 text-card-foreground backdrop-blur"
  >
    <div class="flex h-16 items-center px-6">
      <RouterLink v-if="backLink" :to="backLink">
        <Button variant="ghost" size="icon" class="mr-3" :aria-label="$t('common.back')">
          <ArrowLeft class="h-5 w-5" />
        </Button>
      </RouterLink>
      <div
        v-if="icon"
        class="h-8 w-8 rounded-lg flex items-center justify-center mr-3"
        :class="
          iconGradient ||
          'bg-primary/10 text-primary shadow-none'
        "
      >
        <component :is="icon" class="h-4 w-4 text-white" />
      </div>
      <div class="flex-1">
        <h1 class="text-xl font-semibold text-foreground">{{ title }}</h1>
        <Breadcrumb v-if="breadcrumbs?.length" class="mt-1">
          <BreadcrumbList>
            <template v-for="(crumb, index) in breadcrumbs" :key="index">
              <BreadcrumbItem>
                <BreadcrumbLink v-if="crumb.href" :href="crumb.href">
                  {{ crumb.label }}
                </BreadcrumbLink>
                <BreadcrumbPage v-else>{{ crumb.label }}</BreadcrumbPage>
              </BreadcrumbItem>
              <BreadcrumbSeparator v-if="index < breadcrumbs.length - 1" />
            </template>
          </BreadcrumbList>
        </Breadcrumb>
        <p
          v-if="description || subtitle"
          class="mt-1 text-sm text-muted-foreground"
        >
          {{ description || subtitle }}
        </p>
      </div>
      <slot name="actions" />
    </div>
  </header>
</template>
