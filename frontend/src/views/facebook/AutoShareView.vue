<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { Share2 } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookAutoShare"
    description-key="nav.facebookAutoShareDesc"
    :icon="Share2"
  />
</template>
