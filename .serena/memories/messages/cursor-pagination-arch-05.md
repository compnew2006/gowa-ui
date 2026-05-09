## ARCH-05 — Cursor-Based Pagination for Messages

Replaced offset-based `page/limit` pagination in `GetMessages` (contacts.go) with cursor-based pagination using `(created_at, id)` tuples.

### Key Changes
- Handler uses `switch { before_id: / after_id: / default: }` pattern
- Removed `COUNT(*)` and `OFFSET` queries entirely
- Added composite DB index `idx_messages_contact_created_id` on `(contact_id, created_at, id)`
- Response includes `next_cursor`, `prev_cursor` alongside backward-compatible `has_more`, `total`, `limit`
- Cursor uses `(created_at < ?) OR (created_at = ? AND id < ?)` for stable pagination with tiebreakers
- Frontend types updated with `after_id` and cursor fields
- 7 new tests in `messages_cursor_pagination_test.go`

### Route
- `GET /api/chats/{id}/messages` and `GET /api/contacts/{id}/messages`

### Backward Compat
- `total` field now returns `len(messages)` instead of real COUNT
- `page` param no longer supported
- `before_id` continues to work as before