# Contact name resolution — GOWA chat name is authoritative, not push name

## Two name sources, only one is stable
1. **GOWA chatstorage `chats.name`** — the name WhatsApp stored when the chat was created/synced. Stable, updated only when the contact genuinely changes their name AND GOWA re-syncs.
2. **`p.FromName` in the webhook payload** — the sender's *current* WhatsApp push name. Volatile: changes anytime the user edits their profile name on their phone, and frequently disagrees with the chat name (e.g. business accounts show their business display name as push name but the chat name is the personal name).

## Bug
`upsertGowaContact` overwrote `contact.ProfileName` with `p.FromName` on every incoming message for 1:1 contacts. This caused the chat header to flip to whatever push name WhatsApp reported at delivery time — `وجآته البشري` (correct, from GOWA) became `مكتبة الأركان المثالية` (the account's business display name) the moment a new message arrived.

## Fix (webhook_gowa.go: upsertGowaContact)
- **Groups**: never overwrite `profile_name` (resolved once at creation via `resolveGowaGroupName`). [from prior fix]
- **1:1**: only set `profile_name` from push name when the contact has NO name yet (`strings.TrimSpace(contact.ProfileName) == ""`). Once a name is established (from GOWA's chatstorage at creation, or from the first push name when none existed), subsequent push name changes do NOT overwrite it.
- The push name is still preserved on the **message** (`metadata.sender_push_name`) for per-message sender display — it's the contact *identity* that's protected, not the per-message attribution.

## Data reconciliation applied
Bulk-updated all 1461 1:1 contacts in whatomate's DB from GOWA's chatstorage `chats.name` (the authoritative source). Future incoming messages will no longer overwrite these names with transient push names.