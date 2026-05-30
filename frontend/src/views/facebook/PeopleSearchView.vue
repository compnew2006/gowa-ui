<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { UserCheck } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookPeopleSearch"
    description-key="nav.facebookPeopleSearchDesc"
    :icon="UserCheck"
  />
</template>
