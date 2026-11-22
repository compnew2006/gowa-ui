# Counting definitions — gowa-ui live numbers (diagnosis, no code)

- **Branch / HEAD traced:** `integration/remaining-fixes` @ `21556d4` (merged base of the
  three completed local fix branches). All file:line references below were verified at
  this HEAD. The plan's file table (verified at `db8c58b`) held, except where noted.
- **Audit values used:** live readings from the 2026-11-21 read-only audit —
  contacts 29,803 / 30,641; Active Contacts 6.3K / 6.5K; messages 86.3K / 90.3K;
  Chat conversations 86–87; Pending 27; Private 39; unread badges 438 / 209 / 173
  (up to 3427 reported later).
- **Status key:** **DEFINED** = one defensible definition follows from the traced query.
  **UNDEFINED — needs a product decision from the user** = no single defensible
  definition exists; the number was NOT fixed by guessing. **CONTRADICTION** = two
  live readings for the same org/window cannot both follow from the traced query.
- **Standing rule used throughout:** every count below is per-organization
  (`organization_id = ?`, `ScopeToOrg`, `internal/handlers/cache.go:277`) and
  excludes soft-deleted rows (GORM `Model()` queries). Where a query does something
  else, it is called out explicitly.

## 0. Where dashboard numbers come from (read first)

There is **no** `GET /api/analytics/dashboard` or `GetDashboardStats` endpoint in this
codebase. The dashboard's numbers are per-widget values, not a single stats query:

- `frontend/src/views/dashboard/DashboardView.vue:472` `fetchWidgetData()` calls
  `widgetsService.getAllData({ from, to })`.
- `frontend/src/services/api.ts` `widgetsService.getAllData` → `GET /api/widgets/data`.
- `cmd/gowa-ui/routes.go:535` routes it to `GetAllWidgetsData`,
  `internal/handlers/widgets.go:720`, which requires `analytics:read` (403 otherwise),
  loads the viewer's widgets (`organization_id = ? AND (user_id = ? OR is_shared = true)`,
  ordered by `display_order`), and runs each through `executeWidgetQuery`
  (`widgets.go:763`). A widget whose query errors is **silently skipped** (`continue`).
- The widget rows themselves live in the DB. Defaults are seeded at org creation by
  `SeedDefaultWidgetsForOrg`, `internal/database/postgres.go:1005`:
  "Total Messages" (`messages`/`count`), "Active Contacts" (`contacts`/`count`),
  "Total Campaigns" (`campaigns`/`count`), "Recent Messages" (`messages`/`table`),
  "Quick Actions" (static, no query). **There is no default "Total Contacts" widget** —
  the 29,803 contacts figure does not come from the dashboard at all (see §1).
- Default window: `executeWidgetQuery` (`widgets.go:763`) defaults to the **current
  month, 1st of the month 00:00 UTC → now** when no `from`/`to` are given; explicit
  ranges go through `parseDateRange` (`internal/handlers/helpers.go:171`, UTC-midnight
  start, end extended to end-of-day). The dashboard's date picker defaults to
  `this_month` (`frontend/src/composables/useDateRange.ts`, default preset
  `this_month`; the picker's `from`/`to` are **local** dates) and persists the
  viewer's last choice in localStorage under the `dashboard` key — so two viewers
  (or the same viewer on two days) can legitimately see different windows.
- Consequence: **every dashboard number below is window-dependent and widget-row
  dependent.** A user can edit/delete a widget, so "the dashboard says X" is only
  reproducible for a named widget row in a named window.

## Decisions (best practice, approved 2026-11-22)

The user (2026-11-22) is not technical and delegated all 11 open items below to
best-practice decisions by Muse. Each decision states what the number *should* mean;
where it differs from the traced behaviour in §§1–10, the section says so explicitly.
The traced behaviour descriptions are left intact — decisions change the definition,
not what the code does today.

1. **"Active Contacts" (§2): DECIDED** — a contact is active when it sent at least
   one *inbound* message inside the selected window; internal/group/newsletter
   contacts are excluded. *Rationale: activity means the customer engaged;
   outbound-only is agent activity, not contact activity.*
2. **"Total Messages" (§3): DECIDED** — split sent vs received; count only valid
   messages, excluding revoked and failed. The dashboard widget shows the total of
   valid messages, with the split as the underlying definition. *Rationale: mixing
   directions and dead messages hides what actually happened.*
3. **"Conversations" in Chat (§4): DECIDED** — a conversation is a contact row with
   `last_message_at IS NOT NULL` (the `has_messages=true` filter the backend already
   supports); server-side totals are canonical and tab badges must be server totals,
   not loaded-rows counts. *Rationale: loaded-rows counts are paging artefacts and
   unreproducible.*
4. **Pending tab (§5): DECIDED** — one rule in both places: the server rule
   (unassigned, not closed, has a message, not internal). *Rationale: a
   never-messaged row is not a conversation awaiting a reply.*
5. **Private tab unit (§5): DECIDED** — count merged org-account pairs (one logical
   conversation = one row), not underlying contact rows. *Rationale: the tab lists
   conversations, and a pair is one conversation.*
6. **Unread (§6): DECIDED** — keep the per-conversation badge; do NOT invent an
   org-wide total-unread figure; the Private tab badge must NOT double-count — one
   logical internal message counts once. *Rationale: per-viewer read state makes a
   single total misleading, and a message stored twice is still one message.*
7. **Templates (§7): DECIDED** — campaign picker filtered to the selected account's
   templates; Templates-page total is org-wide; "Available" means Meta-approved /
   sendable. *Rationale: the picker UI intends account filtering, and a local row
   that is not approved cannot be sent.* Data gap: the local `templates` table
   carries no approval status (see §7).
8. **"Online" (§8): DECIDED** — online = live WebSocket connection (presence);
   `is_available` is a separate availability flag, not presence; `online_count`
   follows the list's filters. *Rationale: presence and availability are different
   facts, and a count must agree with the rows shown.*
9. **Break Time (§9): DECIDED** — in the all-agents view the summary card shows the
   org total (sum of all agents' break minutes in the period); every away period
   counts as a break. *Rationale: the card is an org summary, and the log records
   no reason, so no finer distinction is possible.*
10. **Who is an "agent" for analytics (§10): DECIDED** — holders of the agent system
    role in the org; table and dropdown built from the *same* population; list all
    agents including zero-activity ones. *Rationale: the system role is the one
    stable, org-wide definition of "agent", and zeros are informative, not noise.*
11. **Canonical contact total (§1): DECIDED (definition only)** — the canonical
    *definition* is the scoped `ListContacts` total per §1. *Rationale: best
    practice cannot settle a data contradiction — the definition must be
    reproducible, so the 29,803 vs 30,641 values still need a live re-check with
    viewer/time/filters in Task 15 (after deploy); neither figure is canonical as
    a value.*

## 1. Total contacts — 29,803 (and 30,641)

**Traced query.** `GET /api/contacts` → `ListContacts`,
`internal/handlers/contacts.go:140`. The `total` is counted at `:297–299`
(`query.Model(&models.Contact{}).Count(&total)`) over:

```sql
-- ScopeToOrg (cache.go:277) + scopeAssignedContact (contacts.go:570) then, only if sent:
--   has_messages=true  -> last_message_at IS NOT NULL
--   account=?          -> whats_app_account = ?            (single owning account)
--   accounts=a,b       -> whats_app_account IN ? OR EXISTS (messages on those accounts)
--   exclude_groups     -> not group/newsletter (metadata flags + 120362/120363 prefixes)
--   chat_status=closed|pending|internal -> metadata->>'chat_status' filters (see §5)
--   search / tags      -> phone_number / profile_name ILIKE, tags @> ?::jsonb
```

**What the frontend actually sends.** Neither caller sends `has_messages`
(no occurrence in `frontend/src` — the parameter exists only in `contacts.go:150–155`):
Chat (`frontend/src/stores/contacts.ts:548` `fetchContacts`, page 1, limit 50) and
Settings → Contacts (`frontend/src/views/settings/ContactsView.vue:113`, paged) both
call `contactsService.list` with only search/page/limit/account/chat_status/sort.
So the returned `total` counts **all contact rows in scope, including contacts that
have never exchanged a message, group chats, newsletters, internal (Private) contacts
and closed chats** — unless the caller filtered them.

**Visibility.** `scopeAssignedContact` (`contacts.go:570`, `involvementCondition`
`:489`) OR-combines (a) the viewer's assigned WhatsApp accounts (super admins and
users with no assignment fall back to full org visibility) and (b) involvement
(assignee, active collaborator, or holder of an ACTIVE assignment access grant).
Users **without** `contacts:read` see involvement only.

**Definition (DEFINED).** *The number of contact rows in the viewer's org (or, for a
scoped viewer, in the viewer's account/involvement scope) matching the caller's
filters, with no message-activity requirement unless `has_messages=true` was sent —
counted at request time, soft-deleted rows excluded.*

**CONTRADICTION — 29,803 vs 30,641.** The same endpoint with the same (empty) filters
must return the same total for the same viewer at the same moment, so the two audit
readings (29,803 and 30,641, a difference of 838) cannot both be the same number.
They can only be reconciled if the readings differ in viewer, in filters, or in time
(e.g. a contacts sync adding rows between readings, or a search/account filter active
on one reading). One direction is especially suspicious: the 30,641 reading was seen
while signed in as the agent salama, whose scope is *narrower* than a super admin's,
so under `scopeAssignedContact` his total should be ≤ the super admin's, not greater.
**DECIDED (best practice, 2026-11-22) — definition only.** The canonical
*definition* is the scoped `ListContacts` total above (the DEFINED definition):
the number of contact rows in the viewer's org (or, for a scoped viewer, in the
viewer's account/involvement scope) matching the caller's filters, counted at
request time, soft-deleted rows excluded. *Rationale: best practice cannot settle
a data contradiction — the definition must be reproducible (viewer + filters +
moment), so the 29,803 vs 30,641 discrepancy is not resolved by picking a value.*
The two values still require a live re-check with viewer/time/filters recorded in
Task 15 (after deploy). Until then, do not treat either figure as "the" contact
count.

## 2. Active Contacts — 6.3K / 6.5K

**Traced query.** Default widget "Active Contacts" (`postgres.go:1005`, data source
`contacts`, metric `count`, no filters) → `queryContacts`, `internal/handlers/widgets.go:890`:

```sql
SELECT COUNT(*) FROM contacts
WHERE organization_id = ? AND last_message_at >= ? AND last_message_at <= ?
-- plus any widget filters via applyFilter (widgets.go:1279); the default widget has none
```

Window = the dashboard date range (default: current month to date, §0). No
`scopeAssignedContact` is applied (org-wide for anyone with `analytics:read`); no
exclusion of internal contacts, groups/newsletters, or contacts whose last message
was outbound only — `last_message_at` advances on any message, either direction.
The `metric` parameter is ignored (always a count).

**Definition (DEFINED).** *The number of contact rows in the org whose
`last_message_at` falls inside the selected window (default: 1st of the current month
00:00 UTC → now) — i.e. contacts with any message, inbound or outbound, in that
window; internal, group and newsletter contacts included; counted org-wide for
viewers with `analytics:read`.*

**Reconciliation.** 6.3K vs 6.5K is **reconcilable**: the window is month-to-date, so
the number grows through the month, and the two readings were taken at different
moments (and possibly with different persisted date ranges). It is not, on its own,
evidence of a bug.

**DECIDED (best practice, 2026-11-22).** "Active" means the *customer* engaged: a
contact counts as active when it sent at least one *inbound* message inside the
selected window (default: current month to date, §0); internal, group and newsletter
contacts are excluded. *Rationale: activity means the customer engaged;
outbound-only is agent activity, not contact activity.* This **differs from current
behaviour**: the widget today counts any message in either direction and includes
internal/group/newsletter contacts — the fix task must add the inbound-only
condition and the exclusions.

## 3. Total Messages — 86.3K / 90.3K

**Traced query.** Default widget "Total Messages" (`postgres.go:1005`, data source
`messages`, metric `count`) → `queryMessages`, `internal/handlers/widgets.go:859`:

```sql
SELECT COUNT(*) FROM messages
WHERE organization_id = ? AND created_at >= ? AND created_at <= ?
-- plus any widget filters; the default widget has none
```

Window = dashboard date range (default: current month to date, §0). **No direction
filter** (inbound and outbound are mixed), **no status filter** (sent, delivered,
read, revoked and failed all count, subject to GORM soft-delete exclusion), org-wide,
`analytics:read` only.

**Definition (DEFINED).** *The number of message rows in the org whose `created_at`
falls inside the selected window (default: current month to date), all directions and
all statuses, counted org-wide for viewers with `analytics:read`.*

**Reconciliation.** 86.3K vs 90.3K is **reconcilable** the same way as §2: a
month-to-date count read at two different moments grows. It is not, on its own,
evidence of a bug — but see the decision below.

**DECIDED (best practice, 2026-11-22).** "Total Messages" is split into sent vs
received — mixing the two directions hides what actually happened — and counts only
*valid* messages: revoked and failed messages are excluded. The dashboard widget
shows the total of valid messages, with the sent/received split available as the
underlying definition. *Rationale: a total that mixes directions and dead messages
has no single meaning; the split is the meaningful figure and the valid-only total
is the honest headline.* This **differs from current behaviour**: the query today
mixes all directions and all statuses — the fix task must add the split and the
status exclusion.

## 4. Chat conversations — 86–87

**Traced query.** There is **no server-side "conversation count"**. `GET /api/contacts`
returns a paged list and a `total` (§1), but the Chat sidebar never displays that
`total` as a conversation count. The tab badges are client-side lengths of *loaded*
rows:

- `frontend/src/stores/contacts.ts:259` `contactsLimit = 50`; `fetchContacts` (`:548`)
  loads page 1 (limit 50) plus the first closed page merged in (`:637–644`).
- Tab membership is computed client-side (`contacts.ts:349–392`): pending = unassigned
  and not closed; me = assigned (for supervisors: *every* assigned conversation in
  the org; for agents: only their own) and not closed; closed = `chat_status ===
  'closed'`; all = every loaded customer contact; internal = merged org-account
  pairs plus leftover internal rows.
- `pendingCount`/`myCount`/`closedCount`/`allCount`/`internalCount` (`contacts.ts:387–392`)
  are `.length` of those computed arrays; `tabCount` in
  `frontend/src/composables/useChatContactsList.ts:306` renders them.

**Definition: DECIDED (best practice, 2026-11-22).** A "conversation" is a contact
row with `last_message_at IS NOT NULL` — the `has_messages=true` filter the backend
already supports. Server-side totals are canonical: tab badges must be server
totals, not loaded-rows counts. *Rationale: loaded-rows counts are paging artefacts
and unreproducible — a count must not change with how far the viewer scrolled.*
This **differs from current behaviour**: today the 86–87 figure is a count of
*loaded rows*, capped by paging, and the sidebar never displays the server total.

**What was considered:** (a) contact rows with a message (chosen — the
`has_messages=true` filter the backend already supports, and what `contacts.go:150`
says `/chat` is for); (b) the current `ListContacts` total (§1, includes
never-messaged rows — rejected: a row that never exchanged a message is not a
conversation); (c) open (non-closed, non-internal) conversations only — rejected as
the *definition* of "conversations" (openness is a tab filter, not the unit). Note
also that (a) and the sidebar's client-side pending rule disagree: the server-side
pending filter (`contacts.go:230–234`) requires `assigned_user_id IS NULL AND
chat_status <> 'closed' AND last_message_at IS NOT NULL AND NOT internal`, while the
client-side Pending tab (`contacts.ts:349`) requires only "unassigned and not closed"
— so it also counts never-messaged unassigned contacts. The §5 decision picks the
server rule for both places.

## 5. Pending tab — 27; Private tab — 39

**Pending (27).** Two different rules exist (see §4): server-side
`chat_status=pending` (`contacts.go:230–234`, needs `last_message_at IS NOT NULL`,
excludes internal) vs client-side "unassigned and not closed" (`contacts.ts:349`).
The badge the user sees is the client-side one, computed over loaded rows only.
**Definition: DECIDED (best practice, 2026-11-22).** One rule in both places — the
server rule (`contacts.go:230–234`): unassigned, not closed, has a message, not
internal. *Rationale: a never-messaged row is not a conversation awaiting a reply.*
This **differs from current behaviour**: the badge the user sees today is the
client-side rule over loaded rows only — the fix task must use the server rule and
server totals in both places.

**Private (39).** The Private tab does not list contact rows: it lists *merged pairs*
of org accounts that messaged each other, from `ListInternalConversations`
(`internal/handlers/internal_contacts.go:186`, `internalConversations` `:205`).
One pair = one row even though the underlying data is two contact rows (one per
account's copy). Manually flagged contacts (`metadata.internal_chat`, `OnlyInternalContactsSQL`,
`internal/models/internal_chat.go:44`) that are not part of any pair appear as extra
rows. **Definition: DECIDED (best practice, 2026-11-22).** *The number of merged
org-account pairs plus unpaired manually-flagged internal contacts* — one logical
conversation = one row; the underlying contact rows of a pair are never counted
separately. *Rationale: the tab lists conversations, and a pair is one conversation;
counting rows would double-count it.* This **matches current behaviour** for the
unit (the UI already lists pairs); the audit's 39 is now checkable against the pair
count. (The badge itself must still become a server total rather than a loaded-rows
count, per the §4 decision.)

**Related contradiction (flagged, not resolved).** `ExcludeInternalContactsSQL`
(`internal_chat.go:36`) excludes internal chats from the Closed and Pending
server-side tabs, and the client-side tabs filter `is_internal` out of customer
tabs — but the unfiltered `ListContacts` total (§1) *includes* internal contacts.
So "total contacts" and "sum of the Chat tabs" will never agree, by construction.

## 6. Unread badges — 438 / 209 / 173 (up to 3427)

**Traced query.** Per-contact badge = `ContactResponse.unread_count`
(`contacts.go:29`), filled by `unreadCountsForUser` (`internal/handlers/read_state.go:36`)
via `unreadMessagesQuery` (`:17–32`):

```sql
SELECT m.contact_id, COUNT(*) FROM messages m
JOIN contacts c ON c.id = m.contact_id
LEFT JOIN contact_read_states r ON r.contact_id = m.contact_id AND r.user_id = ?
WHERE m.organization_id = ? AND m.contact_id IN ?            -- contacts on the loaded page only
  AND m.direction = 'incoming'
  AND m.status NOT IN ('revoked','failed')
  AND m.deleted_at IS NULL
  AND m.read_seq > COALESCE(GREATEST(c.read_watermark_seq, r.last_read_seq), 0)
GROUP BY m.contact_id
```

(`read_seq` is insert order, not `created_at` — see the header comment in
`read_state.go`.) The exact threshold expression is
`m.read_seq > COALESCE(GREATEST(c.read_watermark_seq, r.last_read_seq), 0)`
(`read_state.go:30`).

**NULL-collapse — checked and NOT a defect (correction, Task 3, 2026-11-22).**
An earlier version of this paragraph claimed PostgreSQL `GREATEST` returns NULL
if any argument is NULL. That is wrong for PostgreSQL (it is the Oracle/MySQL
behaviour): PostgreSQL `GREATEST` ignores NULL arguments and returns NULL only
when every argument is NULL — verified directly on PostgreSQL 16.15
(`GREATEST(NULL, 101) = 101`, `GREATEST(50, NULL) = 50`,
`GREATEST(NULL, NULL) = NULL`) and by the Task 3 threshold tests
(`TestApp_UnreadCount_ReadThreshold`), which pin the behaviour through the
endpoint. Either value can still be NULL: `r.last_read_seq` is NULL when the
LEFT JOIN finds no
`contact_read_states` row for that viewer and contact (the viewer has never opened
the conversation — the column itself is `NOT NULL DEFAULT 0`,
`internal/models/contact_read_state.go:22–28`, but an absent row still yields NULL
from the join), and `c.read_watermark_seq` is a nullable `BIGINT` with no NOT NULL
or default (`internal/database/postgres.go:509` adds the column; the migration at
`:510–514` sets it only for contacts that had a read incoming message). When one
is NULL the other still sets the threshold; only when both are missing does
`COALESCE(..., 0)` make the threshold 0. The threshold therefore already
implements the intended semantics stated in the model's own comment ("a message
is unread for a user when its `read_seq` is above both `LastReadSeq` and the
conversation's shared `read_watermark_seq`", `contact_read_state.go:9–20`) — no
query change was needed, and `read_state.go` now carries a comment saying so.
The per-viewer position is the user's `contact_read_states.last_read_seq`
(`markReadForUser`, `read_state.go:67`); the conversation-wide floor
`contacts.read_watermark_seq` advances only for automated handling
(`markReadForEveryone`, `read_state.go:82`). Rendered raw, with no cap
(`frontend/src/views/chat/ChatView.vue:1328–1329`). Counts are computed only for
contacts on the loaded page (`buildContactResponsesMasked`, `contacts.go:1291`).
There is **no org-wide "total unread" number anywhere** in the code.

**Definition (DEFINED, per contact — as the code actually behaves).** *For one
viewer and one conversation: the number of incoming, non-revoked, non-failed,
non-deleted messages whose `read_seq` is past the greater of that viewer's read
position and the conversation-wide watermark; a missing value counts as 0 inside
the max, so only when both are missing (viewer never opened the conversation
and no watermark was ever set) does the threshold become 0 and the badge count
the conversation's whole inbound history.*

**Reconciliation / flags.**

- A badge as large as 3427 **is producible by this query** (e.g. a conversation the
  viewer has never opened, so there is no `contact_read_states` row) — the query
  has no cap and no time window, and when no watermark was ever set for that
  conversation such a viewer gets its entire inbound history. Whether 3427
  is *correct* for that conversation cannot be settled from code alone; it needs a
  live check of that contact's `read_watermark_seq`, the viewer's
  `contact_read_states` row, and the message count above it.
  **Flagged as not reconcilable from the audit data alone.**
- **Double-counting in the Private tab (DECIDED, best practice, 2026-11-22):**
  internal messages are stored twice (once per account's copy).
  `ListInternalConversations` used to sum the two sides' unread counts into the merged row,
  so one logical internal message counted **2×** in the Private tab badge.
  **Decided: it must NOT double-count** — one logical internal message counts once, even though it is stored twice. *Rationale: a message stored
  twice is still one message.* Fixed in Task 3 (2026-11-22): the merged count now
  de-duplicates by WhatsApp message ID — one logical message counts once even
  when both of its copies are incoming; rows without an ID are counted one by
  one. The merged count is now `pairUnread` (`internal_contacts.go:388`), not a sum of the sides.
  Reading a merged thread also clears both sides for that viewer (fix round 1,
  2026-11-22): `markMessagesAsRead` advances the viewer's read position on the
  pair's other contact too, so the merged badge reaches 0; no extra WhatsApp
  read receipt is sent for the other account's copy. Note the fix only bites when a logical message has an incoming copy on
  both sides; under the current ingest paths each logical message normally has
  exactly one incoming copy (the sender's copy is outgoing), so the old sum was
  already right for fresh data — the de-duplication hardens the invariant for
  legacy/synced rows where both copies are incoming.
- **Client-side drift (flagged):** the store also increments `unread_count` locally
  on incoming WebSocket events (`frontend/src/stores/contacts.ts:875`) and zeroes it
  on open; a badge can therefore disagree with the server count until the next
  refetch. **DECIDED (best practice, 2026-11-22): no org-wide total-unread figure.**
  The per-conversation badge is kept as the only unread number, and no total is
  invented. *Rationale: per-viewer read state makes a single org-wide total
  misleading — it would sum different viewers' unread states into a number that
  belongs to nobody.*
- The three audit badges (438/209/173) are per-conversation values; they were never
  a total, and no code sums them.

## 7. Template count — Templates page vs Create Campaign "1 template available"

**Traced queries (two different ones).**

- Templates page: `GET /api/templates` → `ListTemplates`,
  `internal/handlers/templates.go:55`. Total = `COUNT(*) FROM templates WHERE
  organization_id = ?` (`:77–78`), plus optional `account` / `category` / `search`
  filters (`:66–75`). The page (`frontend/src/views/settings/TemplatesView.vue:99`)
  sends only search/page/limit — no account filter — so its total is **all templates
  in the org**. Pagination default is 50, max 100 (`internal/handlers/helpers.go:97`).
- Create Campaign: `loadTemplates` in
  `frontend/src/views/settings/CampaignDetailView.vue:366` calls
  `GET /api/templates` with params `{ whatsapp_account: <selected account> }`.
  **The backend reads `account`, not `whatsapp_account`** (`templates.go:66`:
  `Peek("account")`). The `whatsapp_account` parameter is silently ignored, so the
  campaign picker returns **page 1 of all org templates (up to 50), unfiltered by
  account**. (The Chat template picker does it correctly:
  `frontend/src/components/chat/TemplatePicker.vue:44` sends `params.account`.)
  This is a real parameter-name mismatch found by tracing, not a counting
  disagreement — recorded here for the fix tasks; no code was changed.

**Definition (DEFINED, per surface).** *Templates page total = all template rows in
the org (matching search, if any). Campaign picker list = first page (≤50) of all
template rows in the org, regardless of the selected account — because the
account filter never reaches the query.*

**CONTRADICTION — "1 template available".** If the org's Templates page shows N
templates, the campaign picker should show min(N, 50) — not 1 — under the traced
behaviour. A reading of "1 template available" in Create Campaign cannot be
reconciled with a Templates-page total ≠ 1 from the same moment. Untested hypotheses
(not verified — no live check was run, so these are guesses to test, not candidate
explanations): different moments (templates created/deleted between readings), a
search filter on one surface, or the audit's "1" referring to a filtered view that
was not recorded.
**DECIDED (best practice, 2026-11-22).** (a) The campaign picker **is** filtered to
the selected account's templates — that is what the UI code intends, and the
`whatsapp_account` vs `account` parameter mismatch is a bug for Task 4, not a
decision. (b) The Templates-page total is org-wide. *Rationale: the picker chooses
templates for one account's campaign, while the Templates page manages the org's
whole template library.* This **differs from current behaviour** on (a): today the
picker returns page 1 of all org templates unfiltered by account.

**DECIDED (best practice, 2026-11-22).** "Available" means Meta-approved / sendable —
a local row that is not approved cannot be sent. *Rationale: only Meta's approval
makes a template sendable; a local row alone proves nothing about sendability.*
**Data gap for Task 4 (verified):** the local `templates` table carries no approval
status — the `Template` model (`internal/models/models.go:463`) has no status or
approval field at all, and `ListTemplates` has no status/approval filter. Until
Task 4 records where approval status lives (Meta sync or a new column), "available"
cannot be computed from local data alone.

## 8. `online_count` — Users page

**Traced query.** `GET /api/users` → `ListUsers`, `internal/handlers/users.go:189`.
Online IDs are fetched once (`:200–205`):

```go
var onlineIDs []uuid.UUID
if a.WSHub != nil { onlineIDs = a.WSHub.OnlineUserIDs(orgID) }
```

`OnlineUserIDs` (`internal/websocket/hub.go:247`) returns every user ID with ≥1
active WebSocket connection in that org. If the hub is nil, the list is nil → 0.

- Normal response (`users.go:283–289`): `"online_count": len(onlineIDs)` — the
  **org-wide** online count, computed **before and independently of** the
  `search` / `role_id` / `online_only` / pagination filters applied to the user
  list. So the badge can disagree with the listed rows (e.g. searching for one user
  still shows the whole org's online count).
- `online_only=true` with nobody online (`users.go:222–231`): early return with
  `"online_count": 0` hardcoded at `:229` and `total: 0`. The 0 is arithmetically
  right in that branch (the list is empty because `len(onlineIDs) == 0`), but the
  value is a literal, not the computed length.
- The user `total` itself counts via the `user_organizations` join
  (`users.go:207–208`), i.e. it includes cross-org members of this org, and only
  non-deleted memberships/users.

**Definition (DEFINED, narrowly).** *`online_count` = the number of user IDs with at
least one live WebSocket connection to this org at the moment of the request,
ignoring the list's search/role/page filters; 0 when the WebSocket hub is absent —
for any caller in the org, at request time.*

**DECIDED (best practice, 2026-11-22).** "Online" means a live WebSocket connection
(presence) — an open tab counts, even if the user is idle or away. `is_available`
is a *separate* availability flag, not presence: an away user with an open tab is
"online" but not "available". And `online_count` follows the list's filters, so the
number agrees with the rows shown. *Rationale: presence and availability are
different facts and must not share one number, and a count must agree with the
rows shown.* This **differs from current behaviour** on the count: today
`online_count` is org-wide regardless of the list's search/role/page filters — the
fix task must scope it to the filtered list. (The recently-active option was
rejected: no last-seen field is consulted anywhere.)

## 9. Break Time — Agent Analytics

**Traced query.** `calculateBreakTime`, `internal/handlers/agent_analytics.go:219`:

```sql
SELECT * FROM user_availability_logs
WHERE user_id = ? AND is_available = false
  AND started_at <= ? AND (ended_at >= ? OR ended_at IS NULL)
-- then, in Go, each log is clipped to [periodStart, periodEnd];
-- an open log ends at time.Now() (capped at periodEnd); a log counts if the
-- clipped duration is > 0; total = sum of clipped minutes, count = # such logs
```

Logs are written only by `UpdateAvailability` (`internal/handlers/users.go:1156`):
when the availability flag *changes*, the open log (if any) is closed and a new log
is opened. **No log exists for a user's initial state**, and a flag changed outside
that endpoint leaves no log. **Defect for the fix tasks (not a product question):**
the query has **no `organization_id` filter** even though the table carries one
(`internal/models/models.go:203`) — a user in two orgs has their breaks counted in
both, contradicting the per-organization standing rule in the header of this
document. The fix tasks should add the org filter; no user decision is needed for
that.

- Per-agent rows: `calculateAgentStats` (`agent_analytics.go:134`) fills
  `total_break_time_mins` / `break_count` from this function; the table in
  `AgentAnalyticsView.vue:468–487` renders those values. Period = the analytics
  date range (default: current month, `agent_analytics.go:72–81`).
- **Org summary: no query exists.** `calculateSummaryStats` (`:121`, used for the
  "all agents" admin view) computes only the close-rating stats — it never touches
  break time, so `summary.total_break_time_mins` is always 0 there.
  `calculateAgentSummaryStats` (`:126`, used for a single-agent view) does compute
  it. The stat card (`AgentAnalyticsView.vue:406–416`) shows
  `my_stats?.total_break_time_mins ?? summary?.total_break_time_mins` — for an admin
  viewing all agents, `my_stats` is the *admin's own* break time, so the card shows
  the viewer's own break time, not any org total.

**Definition (DEFINED, per agent).** *For one agent and one period: the sum, in
minutes, of the parts of that agent's `is_available = false` availability-log
intervals that overlap the period (an interval still open counts up to now), plus
the number of such overlapping intervals.*

**DECIDED (best practice, 2026-11-22).** (a) In the all-agents view the summary card
shows the **org total** — the sum of all agents' break minutes in the period.
(b) Every away period counts as a break. *Rationale: the card is an org summary, so
it must show the org's total, not the viewer's own; and the log records no reason —
only the flag flip — so no finer distinction is possible.* This **differs from
current behaviour**: today the card shows the viewer's own break time in the
all-agents view (and `calculateSummaryStats` never computes break time at all) —
the fix task must compute and show the org total. (The missing `organization_id`
filter on the break-time query remains a scoping defect for the fix tasks, not a
decision — see §9.)

## 10. Agent Analytics agent list — 12 listed, salama missing

**Traced query.** The table rows come from `calculateAllAgentStats`,
`internal/handlers/agent_analytics.go:191`. The GORM call at `:196–204` generates
SQL with the structure below — shown as a faithful unquoted rendering, not
byte-literal GORM output (GORM emits quoted identifiers and `$1` placeholders).
Query structure is unchanged:

```sql
SELECT * FROM users WHERE users.organization_id = ?
AND ( users.id IN (SELECT team_members.user_id FROM team_members
                   JOIN teams ON teams.id = team_members.team_id
                   WHERE teams.organization_id = ? AND team_members.role = 'agent')
   OR users.id IN (SELECT closed_by_user_id FROM chat_closure_ratings
                   WHERE organization_id = ? AND closed_by_user_id IS NOT NULL) )
```

This does not contradict §8: `users` does have a home-organization column
(`organization_id`, `internal/models/models.go:109`). `ListUsers` scopes by the
`user_organizations` join only (`users.go:207–208`) — there is no
`users.organization_id` condition in that query — so it includes cross-org members
(guests whose home org differs). This query scopes by the home-org column only —
so a cross-org member who is a team agent in this org is excluded from the
analytics table. That is a scoping inconsistency recorded as a defect for the fix
tasks, not a product-definition question. (Correction, Task 1b: an earlier version
of this paragraph said `ListUsers` scopes by the home-org column *plus* the join;
that was wrong — verified at `users.go:207–208` — but the defect conclusion is
unaffected.)

Membership is **period-independent** (anyone who ever closed a rated conversation
appears forever; team membership is current). Crucially, `team_members.role` is a
**team** role, not the user's system/custom role: a user whose *role* is "agent"
(32 users hold the agent role) is **not** in the list unless they are a team member
with team-role `agent` or a closer in a rated `chat_closure_ratings` row. Per-agent
numbers are then computed for the selected period (`calculateAgentStats`, `:134`):
outgoing messages sent by that user in the period, break time (§9), and ratings
for closures they closed.

The **filter dropdown is a different population**: `fetchAgents` in
`frontend/src/views/analytics/AgentAnalyticsView.vue:111` calls
`usersStore.fetchUsers()` with **no parameters and no role filter** and maps *all*
returned users into the dropdown. That call returns page 1, limit 50
(`frontend/src/stores/users.ts:67`; the store even documents the hazard at `:156`:
the default call "stops at the first page (50) and silently hides everyone beyond
it"). The org currently has 37 users, so all fit today; beyond 50 the dropdown
would silently drop people. Selecting a non-agent user from the dropdown still
produces stats for them (`GetAgentAnalytics`, `agent_analytics.go:50`, `agent_id`
branch).

**Definition (DEFINED, as implemented).** *Table = users who are current team-members
with team-role `agent` in this org, plus anyone who ever closed a rated conversation
in this org — membership period-independent; per-agent numbers for the selected
period. Dropdown = first 50 users of the org, any role.*

**Reconciliation.** Salama missing from the table while 12 others are listed is
**consistent with the code** if salama is not a team member with team-role `agent`
and has no rated closure row (his 2026-11-21 close of conversation 966561853319
produces a `chat_closure_ratings` row only if the close-rating cycle was rated).
This was not verified against the live DB (read-only diagnosis from code); the
mechanism, not the individual fact, is what is established here.

**DECIDED (best practice, 2026-11-22).** An "agent" for analytics is a holder of the
agent system role in the org. The table and the dropdown must be built from the
*same* population (agent-role holders), and the table lists all agents including
zero-activity ones. *Rationale: the system role is the one stable, org-wide
definition of "agent" — team membership is a grouping, not an identity — and zeros
are informative, not noise.* **Consequence: the current team-role / rated-closer
rule is replaced for the table** — team members with team-role `agent` who do not
hold the agent system role drop out, and agent-role holders with no team membership
and no rated closure come in. The dropdown's page-1-of-50 hazard (it returns the
first 50 users of any role) is fixed by the same change: it must list the same
agent-role population.

**Related permission split (for the record).** `GetAgentAnalytics` (`:50`) branches
on `analytics:read`: without it, a user sees only their own `my_stats` (no table);
with it and an `agent_id`, `my_stats` is the selected agent's; with it and no
`agent_id`, the full table plus the viewer's own `my_stats`. So "salama missing"
has a second possible reading — as a viewer without the permission he would see no
table at all. Which reading the audit saw depends on who was signed in.

## 11. Total Campaigns widget (for completeness)

Default widget "Total Campaigns" → `queryCampaigns`, `widgets.go:903`:
`COUNT(*) FROM bulk_message_campaigns WHERE organization_id = ? AND created_at >= ?
AND created_at <= ?` in the dashboard window. **Definition (DEFINED):** *campaigns
created in the org inside the selected window (default: current month to date).*
The audit recorded 0 campaigns, which is consistent with a count of rows created in
the window — but note this is *created in window*, not *currently existing*.

Note: this section is supplementary to the Task 1 brief (which did not list the
campaigns widget); it is included because the same widget machinery produces it.

## Summary — decisions (best practice, approved 2026-11-22)

All 11 items below are **DECIDED** — the user delegated them to best-practice
decisions by Muse on 2026-11-22 (Task 1b), so no item is still awaiting a user
decision. The one-line decision per item is here; the full decision + rationale
lives in the cited section.

1. **"Active Contacts" (§2): DECIDED** — active = sent ≥1 *inbound* message inside
   the selected window; internal/group/newsletter contacts excluded.
2. **"Total Messages" (§3): DECIDED** — split sent vs received; count only valid
   messages (revoked and failed excluded); widget shows the valid-message total.
3. **"Conversations" in Chat (§4): DECIDED** — a conversation is a contact row with
   `last_message_at IS NOT NULL` (`has_messages=true`); tab badges are server totals.
4. **Pending tab (§5): DECIDED** — one rule in both places: the server rule
   (unassigned, not closed, has a message, not internal).
5. **Private tab unit (§5): DECIDED** — count merged org-account pairs (one logical
   conversation = one row), not underlying contact rows.
6. **Unread (§6): DECIDED** — keep the per-conversation badge; no org-wide
   total-unread figure; the Private tab badge must NOT double-count one logical
   internal message.
7. **Templates (§7): DECIDED** — campaign picker filtered to the selected account's
   templates; Templates-page total is org-wide; "Available" means Meta-approved /
   sendable (local table carries no approval status — data gap for Task 4).
8. **"Online" (§8): DECIDED** — online = live WebSocket connection; `is_available`
   is a separate availability flag; `online_count` follows the list's filters.
9. **Break Time (§9): DECIDED** — the all-agents summary card shows the org total
   (sum of all agents' break minutes in the period); every away period is a "break".
   (The missing `organization_id` filter on the break-time log query is a scoping
   defect for the fix tasks, not a product decision — see §9.)
10. **Who is an "agent" for analytics (§10): DECIDED** — holders of the agent system
    role in the org; table and dropdown built from the same population; all agents
    listed, including zero-activity ones.
11. **Canonical contact total (§1, CONTRADICTION): DECIDED (definition only)** — the
    canonical *definition* is the scoped `ListContacts` total per §1; the 29,803 vs
    30,641 values still need a live re-check with viewer/time/filters in Task 15
    (after deploy) — neither figure is canonical as a value.

## Contradictions and unreconciled values (for the fix tasks)

- **29,803 vs 30,641 contacts (§1):** cannot both be the same number; the agent
  reading being larger than the super-admin reading contradicts scoping. Needs a
  live re-check with viewer/time/filters recorded.
- **Templates page vs "1 template available" (§7):** irreconcilable under the traced
  behaviour (picker shows min(N, 50) of all templates); plus a real parameter-name
  mismatch (`whatsapp_account` sent, `account` read) that makes the picker
  unfiltered by account.
- **Unread badge up to 3427 (§6):** producible by the query but not verifiable from
  audit data alone; a viewer with no read-state row gets the whole inbound
  history only when no watermark was ever set for that conversation — when a
  watermark exists it still bounds the count. Still needs a live check
  of read positions for that conversation.
- **6.3K vs 6.5K active contacts and 86.3K vs 90.3K messages (§2–§3):** reconciled
  as month-to-date growth at different moments — not contradictions.
- **12 agents listed, salama missing (§10):** consistent with the code's
  team-role/rated-closer rule; not a contradiction, but the rule itself is replaced
  by the DECIDED definition in item 10 above.
- **86–87 Chat conversations (§4):** not a query result at all — a loaded-rows
  count; no contradiction to resolve. The definition is now decided in item 3 above.

## Method and limits

- Read-only diagnosis: no code, test, database, or live-system change was made.
  Live values are taken from the 2026-11-21 audit as reported; no live DB query was
  run, so per-row facts (e.g. whether salama has a rated closure, or the exact read
  position behind the 3427 badge) are stated as code-conditional, not verified.
- Every file:line above was checked at HEAD `1819a0e` (re-verified in fix round 1;
  the first version was checked at `21556d4`). Two plan references needed
  correction/expansion while verifying: the dashboard has no single stats endpoint
  (all dashboard numbers are widget queries, §0), and the frontend never sends
  `has_messages` (so Chat and Settings totals both include never-messaged contacts,
  §1/§4).
- **Revision note (fix round 1/5):** three review findings corrected and re-verified
  against the code at `1819a0e` — the missing org filter on the break-time query is
  now recorded as a scoping defect, not a product decision (§9); the unread-badge
  threshold is corrected for PostgreSQL `GREATEST` NULL semantics, with the
  NULL-collapse recorded as a defect (§6); the §10 SQL block is labelled as literal
  GORM-generated SQL and reconciled with §8's `user_organizations` join, with the
  cross-org-member exclusion recorded as a defect (§10). Four minor fixes applied
  in the same round: Private labelled UNDEFINED for sign-off (§5), the §8/§10
  definition sentences completed, the §7 "1 template" causes marked as untested
  hypotheses, and §11 noted as supplementary to the brief.
- **Revision note (Task 1b, 2026-11-22):** all 11 items decided per best practice
  (see "Decisions" section and the Summary); the §10 SQL block is now labelled as a
  faithful unquoted rendering rather than literal GORM output, and the `ListUsers`
  scoping sentence is corrected (join only, no home-org column) with the defect
  conclusion unaffected (§10).
- Incidental finding, recorded for the fix tasks (not fixed here): the campaign
  template picker sends `whatsapp_account` while `ListTemplates` reads `account`
  (§7).
