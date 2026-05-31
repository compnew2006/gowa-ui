# Design

## Theme

Dark-first product dashboard. Light theme available via `.light-theme` class on body. OKLCH color space for all modern tokens (`--km-*`). Legacy hex tokens (`--primary`, `--bg-dark`, etc.) are aliased to OKLCH values via `enterprise-polish.css`.

## Colors

### Core palette (OKLCH)

| Token | Value | Role |
|-------|-------|------|
| `--km-bg` | `oklch(17% 0.025 255)` | Page background |
| `--km-bg-elevated` | `oklch(21% 0.026 255)` | Elevated surface |
| `--km-surface` | `oklch(24% 0.024 255 / 0.94)` | Card background |
| `--km-surface-muted` | `oklch(27% 0.024 255 / 0.9)` | Muted surface, list rows |
| `--km-field` | `oklch(20% 0.022 255 / 0.96)` | Form input background |
| `--km-primary` | `oklch(67% 0.14 257)` | Primary action, links |
| `--km-primary-soft` | `oklch(67% 0.14 257 / 0.14)` | Primary tint background |
| `--km-accent` | `oklch(64% 0.12 285)` | Secondary accent |
| `--km-success` | `oklch(68% 0.14 160)` | Positive states |
| `--km-warning` | `oklch(76% 0.14 80)` | Caution states |
| `--km-danger` | `oklch(62% 0.18 25)` | Destructive/error states |
| `--km-info` | `oklch(66% 0.13 235)` | Informational states |

### Text

| Token | Value | Usage |
|-------|-------|-------|
| `--km-text` | `oklch(93% 0.012 255)` | Primary text |
| `--km-text-muted` | `oklch(73% 0.018 255)` | Secondary text, labels |
| `--km-text-soft` | `oklch(62% 0.018 255)` | Tertiary text, timestamps |

### Borders

| Token | Value | Usage |
|-------|-------|-------|
| `--km-border` | `oklch(91% 0.012 255 / 0.14)` | Default borders |
| `--km-border-strong` | `oklch(91% 0.012 255 / 0.22)` | Hover borders, emphasis |

### Soft status tints

Each status color has a `-soft` variant at ~14-16% opacity for backgrounds: `--km-success-soft`, `--km-warning-soft`, `--km-danger-soft`, `--km-info-soft`.

## Typography

### Font stack

- **Arabic (RTL)**: `'Cairo', sans-serif` via Google Fonts
- **English (LTR)**: `'Roboto', sans-serif` via Google Fonts
- **System fallback**: `-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif`

Body switches between Cairo and Roboto based on `body.rtl` / `body.ltr` class.

### Scale

| Element | Size | Weight |
|---------|------|--------|
| Page heading (h2) | 1.5rem | 600-700 |
| Welcome title | 2rem | 700 |
| Stat value | 1.5rem | 700 |
| Card title | 1.1rem | 600 |
| Body text | 1rem | 400 |
| Small/label | 0.82-0.85rem | 400-600 |
| Tertiary/timestamp | 0.72-0.78rem | 400 |

## Spacing

### Page layout

| Token | Value | Usage |
|-------|-------|-------|
| `--km-page-x` | `clamp(1rem, 2vw, 2rem)` | Horizontal page padding |
| `--km-page-top` | `7.25rem` | Top padding (below fixed nav) |
| `--km-nav-height` | `62px` | Top navigation height |
| `--km-sidebar-width` | `112px` | Left sidebar width |

### Section rhythm (index page)

- Top row to stats: `1.75rem`
- Stats to charts: `2rem`
- Charts to updates: `2rem`

### Internal spacing

| Context | Value |
|---------|-------|
| List item gap | `0.55-0.6rem` |
| Card internal padding | `1-1.1rem` |
| Badge/chip padding | `0.35rem 0.75rem` |
| Button padding | `0.75rem 1.5rem` |

## Border radius

| Token | Value | Usage |
|-------|-------|-------|
| `--km-radius` | `8px` | Cards, modals, buttons |
| `--km-radius-sm` | `6px` | Inputs, small elements |
| Pill/chip | `9999px` | Inline chips, badges |

## Elevation

| Token | Value | Usage |
|-------|-------|-------|
| `--km-shadow` | `0 18px 42px oklch(9% 0.02 255 / 0.32)` | Modals, popovers |
| `--km-shadow-soft` | `0 10px 28px oklch(9% 0.02 255 / 0.22)` | Card hover |

## Components

### Buttons

- **Primary**: `linear-gradient(135deg, --primary, --secondary)`, white text, shadow on hover. Radius 10px.
- **Secondary**: Transparent bg with `--km-primary` at 10% opacity, colored text, border. Radius 10px.
- **Hover**: `translateY(-2px)` + stronger shadow.

### Cards (content-card)

- Background: `--km-surface`, border `--km-border`, radius `--km-radius`.
- Hover: `translateY(-5px)`, border strengthens to `--km-border-strong`.
- Header: icon + title, 1rem bottom margin.

### Chips (km-chip)

- Pill shape (9999px), `--km-surface` background, `--km-border` border.
- Icon + text inline, `0.82rem` weight 600.

### Form inputs

- Background: `--km-field`, border `--km-border`, radius `--km-radius-sm`.
- Focus: border becomes `--km-primary`, `0 0 0 3px` primary ring.

### Dropdowns

- Background: `rgba(30, 41, 59, 0.95)`, backdrop blur 20px.
- Toggle via `.active` class (opacity/visibility/transform transition).
- Close on outside click.

## Icons

Font Awesome 6.4.0 via CDN. Solid style preferred for UI icons (`fa-solid`), regular for decorative.

## Motion

- **Fast transitions**: 160ms ease-out (border, background, color, shadow changes).
- **Normal transitions**: 200ms ease-out (hover state changes).
- **Sidebar**: 300ms ease for expand/collapse.
- Enterprise polish disables `animation` on modals and floating elements by default.

## RTL

Primary direction is RTL (Arabic). Body gets class `rtl` or `ltr` on language change. Sidebar switches from right to left, flex direction reverses, icon margins swap via `body.ltr` overrides.
