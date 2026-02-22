# Code Style & Conventions

## Vue Components
- Use `<script setup lang="ts">` (Composition API)
- Props via `defineProps<{...}>()`
- Emits via `defineEmits<{...}>()`
- Icons: `lucide-vue-next` (e.g., `Download`, `Loader2`, `Paperclip`)
- UI primitives: `@/components/ui/` (Button, Dialog, Badge, etc.)

## TypeScript
- Strict types on interfaces
- Use `computed()` for derived state
- Export types from composables index

## CSS
- TailwindCSS utility classes
- Scoped `<style scoped>` for component-specific styles
- Dark mode is default; light mode via `:root.light .class-name`
- Color palette: emerald for actions, muted for secondary text

## File Organization
- New features: separate composable (`composables/`) + component (`components/`)
- Composables exported from `composables/index.ts`
- Max ~500 LOC per file (ChatView.vue is an exception as the main view)
