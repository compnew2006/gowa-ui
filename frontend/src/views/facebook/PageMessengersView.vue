<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { MessageCircle } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookPageMessengers"
    description-key="nav.facebookPageMessengersDesc"
    :icon="MessageCircle"
  />
</template>
