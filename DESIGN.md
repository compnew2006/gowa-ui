---
name: Whatomate
description: Multi-tenant WhatsApp dispatch console — agent workspace for processing customer conversations at scale.
colors:
  # Twitter (default) — sky blue
  twitter-bg-light: "#ffffff"
  twitter-bg-dark: "#000000"
  twitter-fg-light: "#0f1419"
  twitter-fg-dark: "#e7e9ea"
  twitter-primary-light: "#1e9df1"
  twitter-primary-dark: "#1c9cf0"
  twitter-card-light: "#f7f8f8"
  twitter-card-dark: "#17181c"
  twitter-muted-light: "#e5e5e6"
  twitter-muted-dark: "#181818"
  twitter-accent-light: "#e3ecf6"
  twitter-accent-dark: "#061622"
  twitter-border-light: "#e1eaef"
  twitter-border-dark: "#242628"
  twitter-destructive: "#f4212e"
  # Ocean-breeze — green
  ocean-breeze-bg-light: "#f0f8ff"
  ocean-breeze-bg-dark: "#0f172a"
  ocean-breeze-fg-light: "#374151"
  ocean-breeze-fg-dark: "#d1d5db"
  ocean-breeze-primary-light: "#22c55e"
  ocean-breeze-primary-dark: "#34d399"
  ocean-breeze-card-light: "#ffffff"
  ocean-breeze-card-dark: "#1e293b"
  ocean-breeze-accent-light: "#d1fae5"
  ocean-breeze-accent-dark: "#374151"
  ocean-breeze-border-light: "#e5e7eb"
  ocean-breeze-border-dark: "#4b5563"
  # Soft-pop / Caffeine — indigo + teal + amber
  softpop-bg-light: "#f7f9f3"
  softpop-bg-dark: "#000000"
  softpop-fg-light: "#000000"
  softpop-fg-dark: "#ffffff"
  softpop-primary-light: "#4f46e5"
  softpop-primary-dark: "#818cf8"
  softpop-secondary-light: "#14b8a6"
  softpop-secondary-dark: "#2dd4bf"
  softpop-accent-light: "#f59e0b"
  softpop-accent-dark: "#fcd34d"
  softpop-border-light: "#000000"
  softpop-border-dark: "#545454"
  # Amber-minimal — warm amber
  amber-bg-light: "#ffffff"
  amber-bg-dark: "#0a0a0a"
  amber-fg-light: "#262626"
  amber-fg-dark: "#fafafa"
  amber-primary-light: "#f59e0b"
  amber-primary-dark: "#fbbf24"
  amber-card-light: "#ffffff"
  amber-card-dark: "#171717"
  amber-accent-light: "#fffbeb"
  amber-accent-dark: "#451a03"
  amber-border-light: "#e5e5e5"
  amber-border-dark: "#404040"
  # Brand — pinned across all themes
  whatsapp-green: "#25D366"
  whatsapp-teal: "#128C7E"
  whatsapp-teal-dark: "#075E54"
  whatsapp-light: "#DCF8C6"
  # Violet scale (legacy)
  violet-50: "#f5f3ff"
  violet-100: "#ede9fe"
  violet-200: "#ddd6fe"
  violet-300: "#c4b5fd"
  violet-400: "#a78bfa"
  violet-500: "#8b5cf6"
  violet-600: "#7c3aed"
  violet-700: "#6d28d9"
  violet-800: "#5b21b6"
  violet-900: "#4c1d95"
  violet-950: "#2e1065"
typography:
  display:
    fontFamily: "var(--font-sans)"
    fontSize: "1.5rem"
    fontWeight: 500
    lineHeight: 1.25
    letterSpacing: "var(--tracking-normal)"
  headline:
    fontFamily: "var(--font-sans)"
    fontSize: "1.25rem"
    fontWeight: 500
    lineHeight: 1.4
  title:
    fontFamily: "var(--font-sans)"
    fontSize: "1.125rem"
    fontWeight: 500
    lineHeight: 1.5
  body:
    fontFamily: "var(--font-sans)"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "var(--font-sans)"
    fontSize: "0.75rem"
    fontWeight: 500
    letterSpacing: "0.025em"
  mono:
    fontFamily: "var(--font-mono)"
    fontSize: "0.8125rem"
    fontWeight: 400
  serif:
    fontFamily: "var(--font-serif)"
    fontSize: "1rem"
    fontWeight: 400
rounded:
  sm: "calc(var(--radius) - 4px)"
  md: "calc(var(--radius) - 2px)"
  DEFAULT: "var(--radius)"
  2xl: "calc(var(--radius) + 0.4rem)"
  full: "9999px"
spacing:
  base: "0.25rem"
  base-px: "4px"
  card-padding: "1rem"
  bubble-padding: "0.625rem 1rem"
  surface-max-w: "65ch"
components:
  button-primary:
    backgroundColor: "{colors.twitter-primary-light}"
    textColor: "#ffffff"
    rounded: "{rounded.full}"
    padding: "0.625rem 1.25rem"
    size: "2.25rem"
  button-primary-hover:
    backgroundColor: "rgb(30 157 241 / 0.9)"
    textColor: "#ffffff"
  button-destructive:
    backgroundColor: "{colors.twitter-destructive}"
    textColor: "#ffffff"
    rounded: "{rounded.full}"
    padding: "0.625rem 1.25rem"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.twitter-fg-light}"
    rounded: "{rounded.full}"
    padding: "0.625rem 1.25rem"
  button-outline:
    backgroundColor: "rgb(247 249 250 / 0.95)"
    textColor: "{colors.twitter-fg-light}"
    rounded: "{rounded.full}"
    padding: "0.625rem 1.25rem"
  card-surface:
    backgroundColor: "rgb(247 248 248 / 0.95)"
    textColor: "{colors.twitter-fg-light}"
    rounded: "calc({rounded.DEFAULT} + 0.35rem)"
    padding: "1rem"
  input-field:
    backgroundColor: "{colors.twitter-bg-light}"
    textColor: "{colors.twitter-fg-light}"
    rounded: "{rounded.md}"
    padding: "0.5rem 0.75rem"
    size: "2.25rem"
  chat-bubble-incoming:
    backgroundColor: "rgb(247 248 248 / 0.98)"
    textColor: "{colors.twitter-fg-light}"
    rounded: "1rem 1rem 1rem 0.375rem"
    padding: "{spacing.bubble-padding}"
  chat-bubble-outgoing:
    backgroundColor: "linear-gradient(180deg, rgb(30 157 241 / 0.16), rgb(30 157 241 / 0.10))"
    textColor: "{colors.twitter-fg-light}"
    rounded: "1rem 1rem 0.375rem 1rem"
    padding: "{spacing.bubble-padding}"
---

# Design System: Whatomate

## 1. Overview

**Creative North Star: "The Dispatch Console."**

Whatomate is a workspace, not a destination. Agents process dozens of WhatsApp conversations per shift and the interface must support that tempo without competing for attention. Visual calm is load-bearing: when the surface is restrained, the conversation becomes the focus. The opposite of a marketing site — this is a tool a person stares at for eight hours, so every pixel answers the question, *does this help an agent respond faster?*

The system is multi-tenant and multi-theme. Operators can switch between four named theme presets (twitter, ocean-breeze, soft-pop, amber-minimal) and toggle dark mode independently, so the same token graph has to look intentional in 8 combinations. Density is high by default — chat list, conversation thread, and contact panel share one viewport — but the visual language stays calm by relying on background-contrast elevation rather than shadow.

**Anti-references.** Not Intercom (too productized, too soft), not Salesforce (no dense data tables, no modal-stacked-on-modal flows), not generic AI-chat tools (no animated gradient hero, no "magic" empty states, no mascot). The system is not a landing page. There is no marketing layer. Sign in and you are in the work.

**Key Characteristics:**
- Restrained color: tinted neutrals carry the surface; one saturated primary per theme carries interactive weight.
- Tonal elevation: depth is conveyed by background shifts (background → card → popover), not by shadow.
- Pill geometry: buttons are fully rounded; cards inherit a generous theme radius.
- Bubble-based chat: customer and agent messages are first-class surfaces with their own typography and density rules.
- Multi-theme core: 4 theme presets × 2 color schemes (light/dark) = 8 validated combinations.
- RTL-first: Arabic locale is a first-class citizen; logical properties and `dir` flows apply throughout.

## 2. Colors

Restrained palette. Tinted neutrals carry ~90% of the surface; one saturated primary per theme carries interactive weight. Destructive red is reserved for irreversible actions only. Status colors (chart-1 through chart-5) are used only in data visualization, never as UI accents.

The active palette is set by `data-theme-preset` on `:root`, and the color scheme by `.dark`. Tokens are RGB triples consumed via `rgb(var(--token) / <alpha-value>)` so they support arbitrary alpha at the utility level.

### Theme Presets

Four presets, each with a light and dark variant. Tokens flow through the same semantic roles (background, foreground, primary, primary-foreground, secondary, muted, accent, border, input, ring, card, popover, destructive, sidebar-*); only the values change.

**twitter (default).** Sky-blue primary (`#1E9DF1` light, `#1C9CF0` dark). Open Sans body, Georgia serif, Menlo mono. Largest radius in the system (1.3rem) — buttons and cards feel like rounded pebbles. Near-zero shadows; the theme relies on background contrast. Dark mode is true black (`#000`) with near-white text — high-contrast dispatch console feel.
*Use when:* the default operator experience, brand familiarity, high-contrast dispatch.

**ocean-breeze.** Green primary (`#22C55E` light, `#34D399` dark). DM Sans body, Lora serif, IBM Plex Mono. Medium radius (0.5rem). Soft, near-imperceptible shadows. Sage and slate accent family.
*Use when:* calmer pacing, support / resolution flows, less visually loud.

**soft-pop / caffeine.** Indigo primary (`#4F46E5` light, `#818CF8` dark) with teal secondary (`#14B8A6` / `#2DD4BF`) and amber accent (`#F59E0B` / `#FCD34D`). DM Sans body, Space Mono. High radius (1rem). Hard, full-saturation black borders in light mode and white borders in dark mode — a neo-brutalist accent within an otherwise restrained system.
*Use when:* marketing-team-tenant branding, or when an operator wants a more expressive surface.

**amber-minimal.** Warm amber primary (`#F59E0B` light, `#FBBF24` dark). Inter body, Source Serif 4, JetBrains Mono. Tightest radius (0.375rem). Near-white background, minimal shadows. The most "document-feeling" theme.
*Use when:* long-form reading, deep work sessions, document-heavy tenants.

### Neutral

- **Surface tokens** (background, card, popover, sidebar, muted, accent, border) — tinted neutrals. They carry 90% of any screen. Their job is to *recede*.
- **Text tokens** (foreground, muted-foreground, card-foreground, popover-foreground) — the only foreground family. Never hex outside this set.
- **Input** — a near-background tint for embedded form fields. Distinct from `border`.

### Brand (pinned, theme-independent)

- **WhatsApp green** (`#25D366`), **teal** (`#128C7E`), **teal-dark** (`#075E54`), **light bubble** (`#DCF8C6`) — used in conversation-rail WhatsApp-specific affordances (channel status, instance tag, brand badge on the auth shell). These do not participate in theming.

### WhatsApp Preview Surfaces (pinned, theme-independent)

Used to render the WA-Web phone-frame preview, list-picker sheet, and debug panel inside the flow builder. Like the brand tokens above, these values are pinned: a teal phone header in ocean-breeze should still render teal, not the preset's primary. The accent and link are identity-anchored; the surfaces reproduce the WA-Web phone frame so designers can preview the customer experience accurately.

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `--wa-panel-canvas` | `#EFEAE2` | `#0B141A` | Phone frame background, chat-list surface |
| `--wa-panel-surface` | `#FFFFFF` | `#111B21` | Debug panel container |
| `--wa-panel-header` | `#075E54` | `#202C33` | Phone header bar (in dark, also bubble-in dark) |
| `--wa-panel-toolbar` | `#F0F2F5` | `#202C33` | Input bar / toolbar |
| `--wa-panel-input` | `#FFFFFF` | `#2A3942` | Text input field |
| `--wa-panel-sheet` | `#FFFFFF` | `#1F2C34` | Bottom sheet / picker |
| `--wa-panel-hover` | `#F0F2F5` | `#2A3942` | List-item hover |
| `--wa-panel-bubble-in` | `#FFFFFF` | `#202C33` | Incoming bubble + debug card (shared surface) |
| `--wa-panel-bubble-out` | `#D9FDD3` | `#005C4B` | Outgoing bubble |
| `--wa-panel-tint` | `#E5DDD5` | `#2A3942` | One-off WhatsApp-style background (e.g. settings preview) |
| `--wa-panel-accent` | `#00A884` | `#00A884` | WhatsApp action green (literal hex, no alpha-via-RGB) |
| `--wa-panel-link` | `#53BDEB` | `#53BDEB` | Read-receipt / link blue (literal hex) |

**Tailwind classes** (consumed via `bg-wa-panel-*`, `text-wa-panel-*`, `border-wa-panel-*`, `hover:bg-wa-panel-*`):

- `bg-wa-panel-canvas` / `bg-wa-panel-surface` / `bg-wa-panel-header` / `bg-wa-panel-toolbar` / `bg-wa-panel-input` / `bg-wa-panel-sheet` / `bg-wa-panel-hover` / `bg-wa-panel-bubble-in` / `bg-wa-panel-bubble-out` / `bg-wa-panel-tint` / `bg-wa-panel-accent` / `bg-wa-panel-link`
- `text-wa-panel-accent` / `text-wa-panel-link`
- `border-wa-panel-accent`
- Alpha suffix works on RGB-backed tokens: `bg-wa-panel-bubble-in/80`, `hover:bg-wa-panel-accent/90`
- Literal-hex tokens (`accent`, `link`) do not support `/<alpha>` — use a sibling RGB token for transparency

**Token resolution path.** RGB triples live in `index.css` under `:root, :root.light, :root[data-theme-preset]` (light values) and `:root.dark, :root.dark[data-theme-preset], .dark` (dark values). They are intentionally placed OUTSIDE the 4 theme-preset blocks (twitter, ocean-breeze, soft-pop, amber-minimal) so a preset's primary/secondary overrides do not leak into the WA phone frame. The Tailwind color map lives in `tailwind.config.cjs` under `theme.extend.colors["wa-panel"]`; `accent` and `link` use literal hex strings (like the existing `whatsapp.*` block), the rest use `colorVar()` for alpha support.

### Named Rules

**The Recede Rule.** 90% of every screen is tinted neutrals. The primary accent occupies ≤10% of the surface; it appears on actions, active state, and the ring of focused controls. A screen where primary dominates is broken.

**The Destructive Singularity.** `destructive` (`#F4212E` light, `#F4212E` dark) is reserved for irreversible actions: delete, revoke, force-disconnect. Never use it for warnings, never for chart-3, never for branding.

**The Hard-Border Exception.** The soft-pop / caffeine preset uses full-saturation borders (`#000` light, `#FFF` dark) deliberately. This is the only preset where borders carry visual weight. Do not transplant the treatment to other themes.

## 3. Typography

**Display / Body Font:** `var(--font-sans)` — theme-driven (Open Sans in twitter, DM Sans in ocean-breeze and soft-pop, Inter in amber-minimal). System sans fallbacks.
**Serif Font:** `var(--font-serif)` — Georgia (twitter), Lora (ocean-breeze), DM Sans (soft-pop, sans-of-sans), Source Serif 4 (amber-minimal).
**Mono Font:** `var(--font-mono)` — Menlo (twitter), IBM Plex Mono (ocean-breeze), Space Mono (soft-pop), JetBrains Mono (amber-minimal).

**Character.** A pragmatic pairing: humanist sans for body (legibility at 14px over long shifts), serif as a deliberate counterweight for marketing surfaces and previews, mono for technical readouts (instance tags, identifiers, timestamps in dense rows). The pairing changes per theme — that is the point. Operators can pick the voice that matches their cognitive preference without changing the work surface.

### Hierarchy

- **Display / h1** (500 weight, 1.5rem / 24px, line-height 1.25): page titles, dashboard greetings.
- **Headline / h2** (500, 1.25rem / 20px, line-height 1.4): section headers, panel titles.
- **Title / h3** (500, 1.125rem / 18px, line-height 1.5): card titles, list group headers.
- **Body** (400, 0.875rem / 14px, line-height 1.5): default. The 14px base is intentional — the surface is dense and we earn back the size through rhythm and whitespace.
- **Label** (500, 0.75rem / 12px, tracking +0.025em): form labels, table headers, widget titles, button micro-copy.
- **Mono** (400, 0.8125rem / 13px): identifiers, instance tags, phone numbers in admin views.
- **Serif** (400, 1rem): used in marketing surfaces (auth shell, public pages). Never in the work console.

**Base size:** 14px (set on `html`). Body antialiasing enabled. Links inherit `text-primary` and `font-semibold` with no underline by default.

### Named Rules

**The Tracking-Opt-Out Rule.** Tracking tokens are defined (`--tracking-tighter` through `--tracking-widest`) but `--tracking-normal` (0em) is the default for body and headings. Apply positive tracking only to uppercase labels and micro-copy. Never to body, never to headings.

**The 14px Compact Rule.** The base size is 14px, not 16px. The interface is dense by design. Do not "fix" this by raising the base to 16px in new components — you will break the rhythm of every existing screen.

## 4. Elevation

This system uses **tonal layering, not shadow**, as its primary elevation mechanism. Surfaces are flat at rest; depth is conveyed by the relationship between `background`, `card`, and `popover` tints. Shadows are present but always low-alpha and used only as a response to interaction state (hover, focus, drag).

**Theme posture:**
- **twitter (default):** all shadow alphas set to 0. Hardly any shadow is rendered. Elevation is purely background-contrast.
- **ocean-breeze and amber-minimal:** subtle shadows at 0.05–0.10 alpha. Shadow contributes to depth but never dominates.
- **soft-pop / caffeine:** near-zero shadows, but hard full-saturation borders carry the structural weight instead. The visual register is "outlined, not raised."

**Glass surfaces** use `--glass-bg` / `--glass-bg-hover` / `--glass-border` for the auth shell, sidebar overlays, and the chat composer surface. Glass is a *targeted* effect — it appears in exactly three places and nowhere else. Never use it as a default card treatment.

### Shadow Vocabulary

All shadows are defined in CSS as `--shadow-{2xs,xs,sm,DEFAULT,md,lg,xl,2xl}`, each composed of two stacked layers for crispness. Themes override the per-layer alpha. The default shadow stack is:

- **shadow-2xs / xs:** invisible at rest; reserved for focused controls (`focus-visible:ring` uses these for halo).
- **shadow-sm:** the workhorse. Used on `card-depth` and `.glass` surfaces.
- **shadow:** same composition as sm; alias used by Tailwind's `shadow` utility.
- **shadow-md / lg:** card-interactive hover lift.
- **shadow-xl:** auth shell auth-card.
- **shadow-2xl:** reserved; rarely used in production code.

### Named Rules

**The Flat-At-Rest Rule.** Cards, panels, and list rows are flat by default. A surface is only elevated in response to: hover (lift via shadow-sm or translateY(-2px)), focus (ring + shadow-xs), or drag (shadow-md). At-rest surfaces are *never* shadowed.

**The Glass-By-Invitation Rule.** Glass is allowed on the auth shell, the sidebar in collapsed-overlay mode, and the chat composer. It is forbidden as a default card or button treatment. If a component is starting to look "glass-y everywhere", the system is drifting.

## 5. Components

### Buttons

- **Shape:** pill — fully rounded (`rounded-full`, 9999px). This is a signature of the system.
- **Height:** 2.25rem (default), 1.75rem (xs), 2.25rem (sm), 2.75rem (lg), 2.25rem square (icon).
- **Primary:** `bg-primary text-primary-foreground shadow-sm`, hover `bg-primary/90`. The pill + shadow-sm combo is the canonical "go" affordance.
- **Destructive:** `bg-destructive text-destructive-foreground` — reserved for irreversible actions.
- **Outline:** `border border-border bg-background/95 text-foreground`, hover `bg-accent`.
- **Active:** `border border-primary/15 bg-accent text-accent-foreground` — used for toggle states (e.g. active filter).
- **Secondary:** `bg-secondary text-secondary-foreground`, hover `opacity-90`.
- **Ghost:** transparent default, hover `bg-accent text-foreground`. The default for list-item actions and toolbar buttons.
- **Link:** `text-primary underline-offset-4 hover:underline`. For inline navigation.
- **Glass:** `border border-border bg-card/90 text-foreground`, hover `bg-accent`. For the auth shell and sidebar overlays.
- **Loading:** swaps the slot for a `Loader2` spinner and adds `aria-busy`. The slot content is preserved in an `opacity-0` span for layout stability.
- **Focus:** `focus-visible:ring-2 focus-visible:ring-ring/25 focus-visible:ring-offset-2 focus-visible:ring-offset-background`. Always.

### Inputs

- **Stroke:** `border border-input`. Background `bg-background` standalone, `bg-muted/50` for embedded inputs.
- **Radius:** `rounded-md` (theme-radius minus 2px). Not pill — pill on text inputs reads as a "search" affordance and we don't want that.
- **Height:** 2.25rem default, 2.5rem for standalone primary inputs.
- **Focus:** `focus-visible:ring-2 focus-visible:ring-ring`. Error state adds `border-destructive` and `ring-destructive/40`.
- **Always** include `aria-label` when no visible label is present.

### Cards

- **Corner style:** `rounded-[calc(var(--radius)+0.35rem)]` — the canonical surface card radius.
- **Background:** `bg-card/95` with `backdrop-blur-sm`. Translucency lets the chat background bleed through subtly when the card is over the conversation pane.
- **Border:** `border border-border`. Soft-pop / caffeine preset makes this border deliberately hard.
- **Internal padding:** 1rem (16px). Dense cards drop to 0.75rem.
- **Static depth card:** uses `.card-depth` — adds a 1px gradient underline that fades in on hover. For non-clickable stat cards.
- **Interactive card:** uses `.card-interactive` — hover lifts `translateY(-2px)` and shifts background to `accent/82`. For clickable cards.

### Chat Bubbles

- **Shape:** 1rem corner radius on three corners; the corner pointing at the conversation edge is 0.375rem (asymmetric tail).
- **Incoming:** `bg-card/98`, border `border/72`, tail bottom-left.
- **Outgoing:** `linear-gradient(180deg, primary/16, primary/10)`, border `primary/24`, tail bottom-right.
- **System:** pill (radius 999px), centered, `bg-accent/92 text-muted-foreground`, font-size 12px.
- **Deleted:** `border rgba(248,113,113,0.35)` with a red-tint overlay at 16% alpha (light) or 10% alpha.
- **Width cap:** 65% of the conversation width.
- **Timestamps:** 10px, float-right inside the bubble, gap 6px. Outgoing timestamps and status icons use `text-primary`; incoming use `text-muted-foreground`.
- **Reply preview:** `border-l-2 border-l-primary`, `bg-background/42`, 32×32 thumbnail with 6px radius.
- **Entrance:** 200ms `cubic-bezier(0.25, 1, 0.5, 1)`. Incoming slides from `translateX(-12px)`, outgoing from `translateX(12px)`, system fades with `scale(0.95)`. Disabled under `prefers-reduced-motion: reduce`.
- **Highlight pulse:** when scroll-to-message fires, the target bubble runs `highlight-pulse` (500ms × 2 iterations) at `primary/15` background.

### Navigation

- **Sidebar:** uses dedicated `sidebar-*` tokens for background, foreground, primary, accent, border, ring. Collapsible with a toggle. Mobile: overlay mode (z-50, slide-in from the trailing edge to support RTL).
- **Sidebar items:** ghost-variant buttons, full width, name + role badge trailing. Active state uses the active button variant.
- **Top bar:** contains the organization switcher, theme switcher, and user menu. Sticky.
- **Mobile bottom bar:** only on `<md`; replaces the sidebar as the primary navigation surface.

### Dialogs

- **Overlay:** `fixed inset-0 bg-black/80 z-50`, fades in/out.
- **Content:** centered (`left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2`), `bg-background border rounded-lg shadow-lg`, max-width `lg` by default. Slides in from 48% top + center, scales from 95 to 100, 200ms.
- **Close button:** absolute top-right (`right-4 top-4`), ghost variant, `sr-only` "Close" label, `Cross2Icon` 16×16.
- **Resizable dialogs:** use the `useResizable` composable — pointer events on a 12×12 grip handle at the bottom-right. Never use native `resize` on `position: fixed` (unreliable cross-browser).
- **Scrollable content:** wrap in a `ScrollArea` (reka-ui primitive). `max-h-[420px]` for list dialogs.

### Contact Lists (selectable)

- **Container:** `role="listbox"` with an `aria-label`.
- **Items:** `role="option"`, `aria-selected`, ghost-variant button, full width. Avatar (circle, first initial, `aria-hidden="true"`) + name + email stacked + role badge trailing.
- **Selected state:** `bg-primary/10 border border-primary/20`.
- **Max height:** `max-h-[420px]` inside a `ScrollArea`.

### Widgets

- **Surface:** `.widget-surface` — `rounded-[calc(var(--radius)+0.35rem)] border border-border bg-card text-card-foreground shadow-sm transition-colors`. Hover `bg-accent/75` (light) or `bg-accent/92` (dark).
- **Title:** `.widget-title` — `text-sm font-medium text-muted-foreground`.
- **Description:** `.widget-description` — `text-xs text-muted-foreground/80`.
- **Action:** `.widget-action` — `text-muted-foreground/70 hover:text-foreground hover:bg-accent`.

### Custom Scrollbars

- 6px wide, `rgb(var(--muted-foreground) / 0.25)` thumb at rest, `/0.45` on hover. Transparent track. Border-radius 3px. Applied via `::-webkit-scrollbar` selectors in `index.css`.

### Auth Shell

- **Page:** `.auth-shell` — `min-h-screen flex items-center justify-center p-4` with a primary-tinted radial gradient (alpha 0.16) over the background color.
- **Card:** `.auth-card` — `w-full max-w-md rounded-[calc(var(--radius)+0.7rem)] border border-border bg-card/95 backdrop-blur-xl shadow-xl`. The largest surface radius in the system.
- **Brand badge:** `.brand-badge` — `h-12 w-12 rounded-full bg-primary text-primary-foreground shadow-sm`.

## 6. Do's and Don'ts

### Do

- **Do** drive every color through `var(--token)`. Never hardcode hex in component templates. The token graph exists so themes can be swapped.
- **Do** test every new component against all 4 theme presets (twitter, ocean-breeze, soft-pop, amber-minimal) in both light and dark. A bug that appears in one of 8 combinations is still a bug.
- **Do** use `bg-card/95 backdrop-blur-sm` for surface cards over the chat background. The translucency is part of the look.
- **Do** keep shadows minimal. If you need to add shadow to a card at rest, you have probably chosen the wrong background contrast.
- **Do** use `transition-colors` for hover, focus, and active state changes. 150ms is the workhorse duration.
- **Do** wrap long lists in `ScrollArea` with a reasonable `max-h` (420px is the convention).
- **Do** use `role="listbox"` / `role="option"` with `aria-selected` for any selectable list. The keyboard model depends on it.
- **Do** provide `aria-label` for inputs without visible labels.
- **Do** use the `useResizable` composable for resizable dialogs.
- **Do** use `$t()` (vue-i18n) for every user-visible string. English and Arabic locales are both first-class.
- **Do** verify RTL flow. Test every layout in `dir="rtl"`. Logical properties (margin-inline-*, padding-inline-*) over physical ones.

### Don't

- **Don't** use `net/http` / `http.Handler` patterns in the backend — this project uses `fasthttp` / `fastglue`. (This is a backend convention, but it reinforces the rule that the system is the way it is for a reason.)
- **Don't** use CSS `resize` on `position: fixed` elements. Unreliable cross-browser.
- **Don't** hardcode English strings in templates. Always `$t()`.
- **Don't** use `min-w` on dialogs that might render on small screens. Use viewport-aware widths.
- **Don't** nest cards inside cards. If a card needs grouping, use a `CardHeader` / `CardContent` split, not a child `<Card>`.
- **Don't** use gradient text (`background-clip: text` with gradients). It conflicts with the system's restraint.
- **Don't** use glassmorphism as a default visual treatment. Glass is by-invitation only (auth shell, sidebar overlay, chat composer).
- **Don't** use side-stripe borders (`> 1px` colored left/right border) as accents on cards or list items. The system uses surface contrast and full borders instead.
- **Don't** use bounce or elastic easing curves. Use `ease-out` for state changes; `cubic-bezier(0.25, 1, 0.5, 1)` (ease-out-quart) for chat bubble entrances.
- **Don't** add decorative animations that don't convey meaning. Every animation must answer: what state is this communicating?
- **Don't** ship a 16px base font size "fix". The 14px base is load-bearing for density.
- **Don't** add a feature that requires a new theme preset without first checking whether the existing 4 can express it. The presets are committed brand positions, not a free palette.
- **Don't** reintroduce hardcoded hex in component templates. The token graph is the single source of truth.
