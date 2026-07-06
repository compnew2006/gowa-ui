<script setup lang="ts">
import { Activity } from "lucide-vue-next";
import { PageHeader } from "@/components/shared";
import HealthDashboard from "@/components/HealthDashboard.vue";
</script>

<template>
  <div class="flex h-full flex-col bg-background text-foreground">
    <PageHeader
      title="Instance Health Dashboard"
      subtitle="Live uptime, throughput, error, and queue metrics for each connected number."
      :icon="Activity"
      icon-gradient="bg-primary text-primary-foreground shadow-none"
    />
    <div class="flex-1 p-6 overflow-y-auto">
      <HealthDashboard />
    </div>
  </div>
</template>
