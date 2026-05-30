<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { UsersRound } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookGroupSearch"
    description-key="nav.facebookGroupSearchDesc"
    :icon="UsersRound"
  />
</template>
