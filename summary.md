# ARCH-02 — Contacts Store Decomposition

**Date**: 2026-05-08
**Branch**: `agent/arch-02-split-contacts-store`
**Task**: Split `frontend/src/stores/contacts.ts` (1,422 lines) into focused stores

## Summary

Split the monolithic contacts store into 3 separate Pinia stores + shared helpers, with a backward-compatible facade that preserves all existing import paths and API shapes.

## Approach

**3-store split + Proxy facade** — Each concern gets its own Pinia store, and a facade function (`useContactsStore()`) uses a Proxy to delegate property access to the correct underlying store. This provides:

- Full backward compatibility (no consumer code changes needed)
- Zero circular dependencies (filters → nothing, contacts → filters, messages → contacts)
- Each store is independently testable

## Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `frontend/src/stores/contacts/helpers.ts` | ~311 | Shared utilities: normalizeChatStatus, normalizeContact, message filtering, type interfaces |
| `frontend/src/stores/contacts/chat-filters.ts` | ~16 | `useChatFiltersStore` — search query, tag/instance/chat-type filter state |
| `frontend/src/stores/contacts/contacts.ts` | ~560 | `useContactsStore` — contact CRUD, pagination, bucket management, filtered/sorted lists |
| `frontend/src/stores/contacts/messages.ts` | ~295 | `useMessagesStore` — message list, send, status, reactions, older message pagination |
| `frontend/src/stores/contacts/index.ts` | ~80 | Barrel re-export + Proxy facade for backward compatibility |

## Files Modified

- `frontend/src/stores/contacts.ts` — **deleted** (replaced by `contacts/` directory)

## Files Unchanged (backward compatible)

All 30+ consumer files continue to work without changes:
- `ChatView.vue`, `NotificationBell.vue`, `UserMenu.vue`
- All composables (`useChatMessaging`, `useChatActions`, etc.)
- `websocket.ts`, settings views, test files
- The import `from "@/stores/contacts"` resolves to `contacts/index.ts`

## Dependency Graph

```
chat-filters.ts ──→ (no deps)
contacts.ts ──→ chat-filters.ts
messages.ts ──→ contacts.ts
index.ts ──→ contacts.ts + messages.ts + chat-filters.ts
```

## Verification

- `npx vitest run src/stores/contacts.test.ts` — **6 passed** (unchanged)
- `npx vitest run src/services/websocket.test.ts` — **5 passed**
- `npx vitest run src/components/NotificationBell.test.ts` — **1 passed**
- `npx vite build` — **succeeds** (268.96kb brotli ChatView bundle)
- `npx eslint src/stores/contacts/` — **0 errors**
- `npx vue-tsc --noEmit` — **0 new errors** (pre-existing errors unchanged)

## Key Design Decisions

1. **Proxy facade for backward compat**: `useContactsStore()` returns a Proxy that routes property access to the correct underlying store based on property name sets. This avoids needing to update 30+ consumer files.

2. **No circular dependencies**: The dependency chain is strictly one-directional: filters → nothing → contacts → filters → messages → contacts. Messages store uses `useContactsStore()` (core) via static import — safe because contacts never imports from messages.

3. **Pinia store IDs**: Core store keeps ID `"contacts"`, messages uses `"messages"`, filters uses `"chat-filters"`. All three can be accessed independently via `useMessagesStore()` and `useChatFiltersStore()`.

4. **`StripPinia<T>` type utility**: Removes Pinia internal properties (`$id`, `$state`, etc.) from the facade's return type to avoid `never` intersection when merging store types with different `$id` values.

## Next Steps

1. Gradually migrate consumers from the facade to direct store imports
2. Add dedicated tests for `useMessagesStore` and `useChatFiltersStore`
3. Consider extracting filtered/sorted contacts computed into the filters store once consumers are migrated

## Known Limitations

- The Proxy facade adds a thin runtime overhead per property access (negligible for this use case)
- TypeScript type checking through the facade uses `StripPinia` which removes Pinia-specific methods like `$patch`, `$reset` — consumers using these would need to access the core store directly
- The `contacts.test.ts` has a pre-existing TS2349 type error (not introduced by this change, tests pass at runtime)
