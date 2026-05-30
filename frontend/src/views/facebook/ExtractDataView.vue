<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { Database } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookExtractData"
    description-key="nav.facebookExtractDataDesc"
    :icon="Database"
  />
</template>
