# Gowa-UI — Product

## Register
**Product.** Gowa-UI is an authenticated WhatsApp multi-device messaging console. Users (agents, managers, admins) live in a task: triaging pending conversations, claiming and releasing them, sending media, reviewing audit logs. Design SERVES the product. The tool must disappear into the task. Earned familiarity over novelty.

## Users & purpose
- **Agents** — claim pending chats, reply, release back to pending. Primary surface: `/chat/:contactId`.
- **Managers/Admins** — ghost-view any chat, assign, invite collaborators, audit. Surfaces: `/chat/*`, `/settings/audit-logs`, contacts, GOWA devices.
- **Purpose**: turn an inbound WhatsApp queue into a handled, auditable, multi-agent inbox.

## Brand personality
Calm, precise, fast. Linear/Vercel-dark discipline with a single WhatsApp-emerald accent. No decoration that competes with messages. Dense when the data is dense (contact list, message timeline); generous only where focus matters (claim screen, empty states).

## Anti-references
- WhatsApp Web's cluttered header; we keep the header compact.
- SaaS "card-grid" dashboards; the inbox is a list + timeline, not cards.
- Cream/sand backgrounds (cream is the AI default of 2026). We are dark-first with a true light layer; no warm-neutral tints.

## Strategic design principles
1. **One accent (emerald), used sparingly.** Primary actions, current selection, active tab, unread emphasis. Never decoration.
2. **Consistency IS the affordance.** Same button shape, same icon style, same control vocabulary across the app.
3. **Every interactive element has all states.** default / hover / focus / active / disabled / loading.
4. **Messages are sacred.** The timeline is the highest-density, highest-attention surface. Bubbles, spacing, and typography there are tuned for long reading sessions.
5. **RTL-aware.** Arabic is a first-class locale; layout must mirror, not just translate.

## Register-locked command routing
For the `/chat/*` surface, default to: `polish`, `layout`, `typeset`, `harden`. Avoid brand-forward commands (`bolder`, `delight`, `colorize`) unless explicitly asked.
