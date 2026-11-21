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
**Needs a user decision:** which reading (viewer + moment + filters) is canonical,
and a live re-check of both totals with viewer/time recorded. Until then, do not treat
either figure as "the" contact count.

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

**UNDEFINED — needs a product decision from the user.** The query defines what the
widget *does*, but "active" as a product concept is undecided: should it mean
(a) any message in the window (current behaviour), (b) an *inbound* message in the
window, (c) a message in the last 30 days regardless of the dashboard window, or
(d) something else (e.g. excluding internal/group contacts)? The fix task must not
pick one of these by guessing.

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
evidence of a bug — but see the open question below.

**UNDEFINED — needs a product decision from the user.** Is "Total Messages" supposed
to mix sent and received, or should the dashboard show them separately? And should
revoked/failed messages count? The current query mixes everything; the product
meaning was never written down.

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

**Definition: UNDEFINED — needs a product decision from the user.** The 86–87 figure
cannot be reproduced from any single query: it depends on how many pages the viewer
had loaded (infinite scroll appends in `loadMoreContacts`, `contacts.ts:674`), which
tab was showing, and the viewer's scope. It is a count of *loaded rows*, capped by
paging, not a count of conversations in the org.

**What "conversations" could mean (for the user to choose one):** (a) contact rows
with `last_message_at IS NOT NULL` (the `has_messages=true` filter the backend
already supports but the frontend never sends — `contacts.go:150` comment says this
is what `/chat` is for); (b) the current `ListContacts` total (§1, includes
never-messaged rows); (c) open (non-closed, non-internal) conversations only. Note
also that (a) and the sidebar's client-side pending rule disagree: the server-side
pending filter (`contacts.go:230–234`) requires `assigned_user_id IS NULL AND
chat_status <> 'closed' AND last_message_at IS NOT NULL AND NOT internal`, while the
client-side Pending tab (`contacts.ts:349`) requires only "unassigned and not closed"
— so it also counts never-messaged unassigned contacts. Any fix must pick one
pending rule and use it in both places.

## 5. Pending tab — 27; Private tab — 39

**Pending (27).** Two different rules exist (see §4): server-side
`chat_status=pending` (`contacts.go:230–234`, needs `last_message_at IS NOT NULL`,
excludes internal) vs client-side "unassigned and not closed" (`contacts.ts:349`).
The badge the user sees is the client-side one, computed over loaded rows only.
**Definition: UNDEFINED — needs a product decision** (same decision as §4: which
pending rule, and whether the badge is a server total or a loaded-rows count).

**Private (39).** The Private tab does not list contact rows: it lists *merged pairs*
of org accounts that messaged each other, from `ListInternalConversations`
(`internal/handlers/internal_contacts.go:186`, `internalConversations` `:205`).
One pair = one row even though the underlying data is two contact rows (one per
account's copy). Manually flagged contacts (`metadata.internal_chat`, `OnlyInternalContactsSQL`,
`internal/models/internal_chat.go:44`) that are not part of any pair appear as extra
rows. **Definition (DEFINED, narrowly):** *the number of merged org-account pairs
plus unpaired manually-flagged internal contacts currently loaded for the viewer.*
But the *product* meaning of "39 private conversations" is **UNDEFINED — needs a
product decision**: should the count be pairs, or the underlying contact rows (which
would be up to ~2× the pair count)? The audit's 39 cannot be checked against the
contact-row count without knowing which unit was intended.

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
`read_state.go`.) The badge is **per viewer**: opening a chat moves only that
user's `contact_read_states.last_read_seq` (`markReadForUser`, `:57`); the
conversation-wide floor `contacts.read_watermark_seq` advances only for automated
handling (`markReadForEveryone`, `:72`). Rendered raw, with no cap
(`frontend/src/views/chat/ChatView.vue:1328–1329`). Counts are computed only for
contacts on the loaded page (`buildContactResponsesMasked`, `contacts.go:1291`).
There is **no org-wide "total unread" number anywhere** in the code.

**Definition (DEFINED, per contact).** *For one viewer and one conversation: the
number of incoming, non-revoked, non-failed, non-deleted messages whose `read_seq`
is past that viewer's read position (or the conversation-wide watermark, whichever
is greater).*

**Reconciliation / flags.**

- A badge as large as 3427 **is producible by this query** (e.g. a conversation the
  viewer has never opened, so `last_read_seq` is 0) — the query has no cap and no
  time window. Whether 3427 is *correct* for that conversation cannot be settled
  from code alone; it needs a live check of that contact's `read_watermark_seq`,
  the viewer's `contact_read_states` row, and the message count above it.
  **Flagged as not reconcilable from the audit data alone.**
- **Double-counting in the Private tab (flagged):** internal messages are stored
  twice (once per account's copy). `ListInternalConversations` sums the two sides'
  unread counts into the merged row (`internal_contacts.go:291`), so one logical
  internal message can count **2×** in the Private tab badge. Whether that is
  intended is **UNDEFINED — needs a product decision**.
- **Client-side drift (flagged):** the store also increments `unread_count` locally
  on incoming WebSocket events (`frontend/src/stores/contacts.ts:875`) and zeroes it
  on open; a badge can therefore disagree with the server count until the next
  refetch. There is no single "unread total" definition to sign off — **UNDEFINED —
  needs a product decision** if the user wants a total unread figure at all.
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
reconciled with a Templates-page total ≠ 1 from the same moment. Possible causes:
different moments (templates created/deleted between readings), a search filter on
one surface, or the audit's "1" referring to a filtered view that was not recorded.
**Needs a user decision:** (a) should the campaign picker be filtered to the
selected account's templates (presumably yes — that is what the UI code intends);
(b) is the Templates-page total org-wide, or per-account? Until (a)/(b) are decided,
the two surfaces cannot be made to agree.

**UNDEFINED — needs a product decision.** `ListTemplates` has no status/approval
filter: a row in the local `templates` table is not necessarily an approved,
sendable WhatsApp template on Meta's side. What "available" means (local row vs
Meta-approved) is undecided.

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
ignoring the list's search/role/page filters; 0 when the WebSocket hub is absent.*

**UNDEFINED — needs a product decision from the user.** "Online" as a product concept
is undecided: (a) live WebSocket connection (current behaviour — an open tab counts,
even if the user is idle or away); (b) `is_available = true` (the availability flag
toggled by `UpdateAvailability`, `users.go:1156` — a *different* thing: an away user
with an open tab is "online" but not "available"); (c) recently active (last-seen
timestamp — no such field is consulted anywhere). The fix task must not pick one.
Also undecided: should `online_count` follow the list filters (so it matches the
rows shown) or stay org-wide?

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
that endpoint leaves no log. Note also that the query has **no `organization_id`
filter** even though the table carries one — a user in two orgs has their breaks
counted in both.

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

**UNDEFINED — needs a product decision from the user.** (a) There is no defined
"total / average break time for the org" — the summary card currently shows the
viewer's own value in the all-agents view. The user must decide what the card
should show (org total, org average, or the viewer's own). (b) Is *any* away period
a "break", or only some away reasons? The log records no reason — only the flag
flip. (c) Should cross-org log rows be excluded (add the missing org filter)?

## 10. Agent Analytics agent list — 12 listed, salama missing

**Traced query.** The table rows come from `calculateAllAgentStats`,
`internal/handlers/agent_analytics.go:191`:

```sql
SELECT * FROM users WHERE users.organization_id = ?
AND ( users.id IN (SELECT team_members.user_id FROM team_members
                   JOIN teams ON teams.id = team_members.team_id
                   WHERE teams.organization_id = ? AND team_members.role = 'agent')
   OR users.id IN (SELECT closed_by_user_id FROM chat_closure_ratings
                   WHERE organization_id = ? AND closed_by_user_id IS NOT NULL) )
```

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
in this org. Dropdown = first 50 users of the org, any role.*

**Reconciliation.** Salama missing from the table while 12 others are listed is
**consistent with the code** if salama is not a team member with team-role `agent`
and has no rated closure row (his 2026-11-21 close of conversation 966561853319
produces a `chat_closure_ratings` row only if the close-rating cycle was rated).
This was not verified against the live DB (read-only diagnosis from code); the
mechanism, not the individual fact, is what is established here.

**UNDEFINED — needs a product decision from the user.** Who counts as an "agent" for
analytics? Candidates the user must choose between: (a) holders of the agent
*role* (32 users); (b) team members with team-role `agent` (current table rule);
(c) anyone who sent a message / closed a chat in the period. The table and the
dropdown must then be built from the *same* population. Also undecided: should the
table be limited to agents with activity in the selected period, or list all agents
with zeros?

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

## Summary — what needs a user decision

The following are flagged **UNDEFINED — needs a product decision from the user** and
were NOT resolved by guessing. The coordinator should bring this list to the user
for sign-off before any fix task starts (Task 1, Step 3).

1. **"Active Contacts" (§2):** any message in window (current) vs inbound-only vs
   last-30-days vs excluding internal/group contacts.
2. **"Total Messages" (§3):** mix sent+received (current) vs split, and whether
   revoked/failed messages count.
3. **"Conversations" in Chat (§4):** contacts with a message (`has_messages=true`)
   vs all contact rows vs open non-internal only; and whether tab badges are server
   totals or loaded-rows counts.
4. **Pending tab (§5):** server rule (needs a message, excludes internal) vs
   client rule (unassigned and not closed, includes never-messaged).
5. **Private tab unit (§5):** merged org-account pairs (current) vs underlying
   contact rows.
6. **Unread (§6):** whether a total-unread figure should exist at all (none does);
   whether the Private tab badge should double-count the two stored copies of one
   internal message.
7. **Templates (§7):** campaign picker filtered by selected account (intended) vs
   org-wide; Templates-page total org-wide vs per-account; what "available" means
   (local row vs Meta-approved).
8. **"Online" (§8):** live WebSocket connection (current) vs `is_available` flag vs
   recently-active; and whether `online_count` should follow the list filters or
   stay org-wide.
9. **Break Time (§9):** what the summary card should show in the all-agents view
   (org total / average / viewer's own); whether every away period is a "break";
   whether to add the missing org filter to the log query.
10. **Who is an "agent" for analytics (§10):** agent-role holders vs team-role
    agents vs anyone active in the period; table and dropdown must use the same
    population; whether zero-activity agents are listed.
11. **Canonical contact total (§1, CONTRADICTION):** which of 29,803 / 30,641 is
    canonical (viewer + moment + filters), to be settled by a live re-check.

## Contradictions and unreconciled values (for the fix tasks)

- **29,803 vs 30,641 contacts (§1):** cannot both be the same number; the agent
  reading being larger than the super-admin reading contradicts scoping. Needs a
  live re-check with viewer/time/filters recorded.
- **Templates page vs "1 template available" (§7):** irreconcilable under the traced
  behaviour (picker shows min(N, 50) of all templates); plus a real parameter-name
  mismatch (`whatsapp_account` sent, `account` read) that makes the picker
  unfiltered by account.
- **Unread badge up to 3427 (§6):** producible by the query but not verifiable from
  audit data alone; needs a live check of read positions for that conversation.
- **6.3K vs 6.5K active contacts and 86.3K vs 90.3K messages (§2–§3):** reconciled
  as month-to-date growth at different moments — not contradictions.
- **12 agents listed, salama missing (§10):** consistent with the code's
  team-role/rated-closer rule; not a contradiction, but the rule itself needs the
  product decision in item 10 above.
- **86–87 Chat conversations (§4):** not a query result at all — a loaded-rows
  count; no contradiction to resolve, but no definition to sign off either.

## Method and limits

- Read-only diagnosis: no code, test, database, or live-system change was made.
  Live values are taken from the 2026-11-21 audit as reported; no live DB query was
  run, so per-row facts (e.g. whether salama has a rated closure, or the exact read
  position behind the 3427 badge) are stated as code-conditional, not verified.
- Every file:line above was checked at HEAD `21556d4`. Two plan references needed
  correction/expansion while verifying: the dashboard has no single stats endpoint
  (all dashboard numbers are widget queries, §0), and the frontend never sends
  `has_messages` (so Chat and Settings totals both include never-messaged contacts,
  §1/§4).
- Incidental finding, recorded for the fix tasks (not fixed here): the campaign
  template picker sends `whatsapp_account` while `ListTemplates` reads `account`
  (§7).
