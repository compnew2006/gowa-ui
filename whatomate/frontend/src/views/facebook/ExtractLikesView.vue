<script setup lang="ts">
import FacebookToolPlaceholder from "./FacebookToolPlaceholder.vue";
import { ThumbsUp } from "lucide-vue-next";
</script>

<template>
  <FacebookToolPlaceholder
    title-key="nav.facebookExtractLikes"
    description-key="nav.facebookExtractLikesDesc"
    :icon="ThumbsUp"
  />
</template>
