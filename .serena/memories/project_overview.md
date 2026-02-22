# Whatomate Project Overview

## Purpose
WhatsApp CRM platform with multi-instance support. Backend in Go, frontend in Vue 3.

## Tech Stack
- **Backend**: Go, GORM, SQLite/PostgreSQL, whatsmeow (WhatsApp Web protocol)
- **Frontend**: Vue 3 + Vite + TypeScript + TailwindCSS + shadcn-vue (reka-ui)
- **State**: Pinia stores
- **Testing**: Playwright (E2E)
- **Styling**: Tailwind + scoped CSS, dark/light mode via `:root.light`

## Structure
- `cmd/` - Go entrypoints
- `internal/` - Go backend packages
- `frontend/src/` - Vue frontend
  - `views/` - Page-level components (ChatView, settings, etc.)
  - `components/` - Reusable components (chat/, ui/, shared/, whatsmeow/)
  - `composables/` - Vue composables (useMediaGroups, useColorMode, etc.)
  - `stores/` - Pinia stores (contacts, auth, instances, etc.)
  - `services/` - API client and WebSocket service
  - `i18n/` - Internationalization (en, pt, etc.)
  - `types/` - TypeScript type definitions

## Conventions
- Components use `<script setup lang="ts">` with composition API
- Icons from `lucide-vue-next`
- UI primitives from `@/components/ui/` (shadcn-vue)
- Dark mode: default, light mode via `:root.light` CSS selectors
- State management via Pinia stores
