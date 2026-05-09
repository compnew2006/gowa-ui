## ARCH-02: Contacts Store Decomposition (2026-05-08)

### What was done
Split `frontend/src/stores/contacts.ts` (1,422 lines) into 4 files under `frontend/src/stores/contacts/`:
- `helpers.ts` — shared utilities, types, constants
- `chat-filters.ts` — `useChatFiltersStore` (search, tag/instance/chat-type filters)
- `contacts.ts` — `useContactsStore` (contact CRUD, pagination, filtered lists)
- `messages.ts` — `useMessagesStore` (message CRUD, send, status)
- `index.ts` — barrel re-export + Proxy facade for backward compatibility

### Dependency graph (zero circular deps)
- filters → nothing
- contacts → filters
- messages → contacts
- index → all three

### Backward compatibility
All consumers of `from "@/stores/contacts"` continue to work without changes. The old `contacts.ts` monolith was deleted; `contacts/index.ts` serves as the barrel. `useContactsStore()` returns a Proxy facade that delegates to the correct underlying store.

### Key technical note
The Proxy facade uses `StripPinia<T>` to avoid `never` intersection from conflicting `$id` Pinia store types. Messages store imports from contacts store via static import (safe: contacts never imports from messages).

### Branch
`agent/arch-02-split-contacts-store`