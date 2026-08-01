# Gowa-UI — Design System

Dark-first product console (Linear/Vercel discipline) with a single WhatsApp-emerald accent. Captured from `frontend/src/assets/index.css`, `fonts.css`, and `tailwind.config.cjs`.

## Theme

**Dark-first.** Light mode (`.light` / `.dark` class) is a true neutral layer, never warm-tinted. Background carries a single subtle emerald radial at the top for depth.

```css
/* Dark (default) */
--background: 0 0% 2.5%;   --foreground: 0 0% 98%;
--card: 0 0% 5%;           --popover: 0 0% 7%;
--primary: 160 84% 39%;    /* emerald — the ONLY accent */
--muted-foreground: 0 0% 55%;
--border: 0 0% 12%;        --input: 0 0% 12%;
--destructive: 0 72% 51%;
--radius: 0.75rem;

/* Glass utilities (dark) */
--glass-bg: rgba(255,255,255,0.03);
--glass-bg-hover: rgba(255,255,255,0.06);
--glass-border: rgba(255,255,255,0.06);
```

Accent role rules (emerald): primary actions, current selection, active tab, unread emphasis, focus ring. **Never** decoration, gradients on text, or side-stripes.

## Typography

- **One family: Inter** (self-hosted, weights 400/500/600). No display pairing — product register.
- Base font-size 14px, antialiased. Line-height 1.5 body.
- Fixed rem scale (NOT fluid clamp): h1 `text-2xl`/1.25, h2 `text-xl`/1.4, h3 `text-lg`/1.5, h4–6 `text-base`. Ratio ~1.125–1.2.
- Weight contrast for hierarchy (medium headings), not scale shouting.
- Links: `text-primary font-medium`, no underline.

## Color usage

Restrained. Semantic state vocabulary required on every interactive element: hover / focus / active / disabled / loading / error / warning / success / info.
- Neutral layer for sidebars/panels (the dark `bg-[#0a0a0b]` sidebar).
- Status accents used sparingly: amber (pending/unassigned), blue (collaborator/assigned-other), gray (closed), red (destructive/revoke), emerald (primary/mine/active).
- Contrast floor: body ≥4.5:1, large text ≥3:1. `text-white/40` and `text-white/45` are the established muted scale on dark — do not go lighter than `/40` for text.

## Layout

- 4pt base spacing scale (4/8/12/16/24/32/48/64). Tailwind's scale is the source of truth; no arbitrary values outside it.
- `gap` for siblings, not margins.
- Flexbox 1D, Grid 2D. Sidebar is resizable (`sidebarWidth`).
- Chat surface = sidebar (list) ‖ main panel (header ‖ timeline ‖ composer).
- Density: tight in contact rows & timeline; generous in interstitials (claim/closed/join) and empty states.
- RTL-aware: Arabic mirrors via vue-i18n + logical properties.

## Components

- **Buttons**: `variant="ghost" size="icon"` (h-8 w-8) for header actions; `variant="ghost" size="sm"` for inline actions; primary `Button` for the single CTA in an interstitial. Icon + label, verb + object copy.
- **Inputs**: h-8 in the sidebar search, rounded via `--radius`, `bg-white/[0.04]` dark / `bg-gray-50` light.
- **Tabs**: pill strip (`bg-white/[0.06]` track, active `bg-emerald-600 text-white`), with count badges.
- **Badges**: `h-4 text-[9px]` inline next to names; status-tinted (`bg-{color}-500/20 text-{color}-400`).
- **Bubbles**: outgoing vs incoming alignment; system messages centered pill `bg-white/[0.04]`.
- **Scrollbars**: 6px, `muted-foreground/0.3` thumb.

## Motion

150–250ms transitions. `transition-colors` / `transition-all duration-300`. No bounce/elastic. `prefers-reduced-motion` respected (existing `animate-fade-in` etc. are short crossfades).

## Bans (impeccable, enforced here)

No side-stripe borders. No gradient text. No glassmorphism as decoration (glass is a defined token, used sparingly). No `border + wide box-shadow` pairing on the same element. Card radius ≤16px (we use `--radius: 0.75rem` = 12px). No all-caps body copy. No em dashes in copy.
