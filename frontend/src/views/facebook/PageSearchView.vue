<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { Search } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookPageSearch"
    description-key="nav.facebookPageSearchDesc"
    :icon="Search"
  />
</template>
