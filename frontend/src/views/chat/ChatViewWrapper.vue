<script setup lang="ts">
import ChatErrorBoundary from "@/components/chat/ChatErrorBoundary.vue";
import ChatView from "./ChatView.vue";
</script>

<template>
  <ChatErrorBoundary>
    <ChatView v-bind="$attrs" />
  </ChatErrorBoundary>
</template>
