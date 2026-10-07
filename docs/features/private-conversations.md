# Private conversations

The chat sidebar's **Private** tab collects conversations that are not customer
service: chats between your organization's own WhatsApp numbers, and any
conversation a user moves there by hand (a supplier, a manager, a partner).
Private conversations stay out of the customer queue and are skipped by every
customer automation.

## What counts as private

A conversation is private when either is true:

- **It is with one of your own connected numbers.** If the Saudi account messages
  the Egypt account, the system recognises the Egypt number as an org number
  (from the account's connected WhatsApp JID; a `:<device>` suffix is ignored).
- **A user moved it to Private.** This is a flag on the conversation that anyone
  can set or clear (see below).

## In the sidebar

- Private conversations appear **only** in the Private tab, not in Me, Pending,
  Closed or All. Search still finds them from any tab.
- The tab shows only while there is at least one private conversation, and each
  user can hide it from their profile.
- With five tabs the strip wraps onto two rows, because the sidebar has a fixed
  width.

### Chats between two of your numbers

WhatsApp delivers every message between two org numbers to both accounts, so
the system holds two copies of the conversation, one per account. The Private
tab merges them into **one row** per pair of accounts:

- The title names both accounts, for example `Saudi-Main ↔ Saudi-Support`. The
  subtitle is the latest message, and the badge counts unread messages on both
  sides.
- Clicking the row opens the side that holds the unread messages, so a reply
  goes out from the account that received the message.
- The conversation header shows **Send as: [Account A] [Account B]**. Switching
  changes the account that sends; the thread flips direction because you now
  see the other account's copy.
- Opening the conversation marks both copies read. While it is open, a new
  message does not raise a notification or unread badge for its second copy.
- With three or more accounts, every pair of accounts gets its own row.

Each side shows that account's own copies. If one account was disconnected for
a while, messages from that period may only exist on the other side.

## Moving a conversation in or out

Open the conversation, then use the **⋮** menu in the header:
**Move to Private** or **Remove from Private**.

- Any user who can reply in the conversation may do this (`chat:write`). Users
  who only have historical read-only access cannot.
- Chats with an org number are always private, so they have no toggle.
- The change is recorded in the audit log, and other users' sidebars update
  immediately.

## Showing or hiding the tab

**Profile → Chat preferences → Show the Private tab.** This is a per-user setting
and is on by default. When it is off, the tab and its conversations leave that
user's sidebar. Search and notifications still reach them, and nothing changes
for other users.

## Automations that skip private conversations

| Automation | Behaviour for private conversations |
|---|---|
| Business-hours away reply | Not sent |
| Close-rating prompt | Not sent |
| Pending rating from before the move | Dropped; the reply is treated as a normal message, with no thank-you |
| Call auto-reject message | The call is still rejected, but the message is not sent |
| Daily reset schedule | Never returned to the pending queue |

New customer automations must check `isInternalContact` (handlers), or append
`models.ExcludeInternalContactsSQL` to queries on the `contacts` table.

## API

| Method | Path | Notes |
|---|---|---|
| `GET` | `/api/contacts/internal-conversations` | One entry per pair of org accounts: `key`, `accounts`, `last_message_at`, `last_message_preview`, `unread_count`, and `sides[]` (`contact_id`, `account`, `peer_account`, per-side unread). Each side respects the caller's contact visibility. |
| `PUT` | `/api/contacts/{id}/internal` | Body `{"internal": true\|false}`. Requires `chat:write` on a conversation the caller can write to. Returns the updated contact and broadcasts `contact_update`. |
| `PUT` | `/api/me/settings` | Partial update: only the fields sent change. `show_internal_tab` (boolean) controls the Private tab. |

Contact responses carry `is_internal` (private for either reason),
`internal_account_name` (set when the number is an org account), and
`internal_marked` (moved to Private by hand). The manual flag is stored as
`metadata.internal_chat` on the contact.
