# Design System

## Creative North Star
The Dispatch Console. A workspace where agents process conversations with minimal friction. Visual calm enables focus; interactive density enables throughput. Every pixel answers the question: does this help an agent respond faster?

## Color Strategy
Restrained. Tinted neutrals carry 90% of the surface. One saturated primary per theme for interactive elements and active states. Destructive red reserved for irreversible actions. Status colors (chart-1 through chart-5) used only in data visualization.

## Theme Presets

### twitter (default)
Light/dark. Sky-blue primary. Open Sans. High radius (1.3rem). Near-zero shadows.

| Token | Light | Dark |
|-------|-------|------|
| background | #fff | #000 |
| foreground | #0f1419 | #e7e9ea |
| primary | #1e9df1 | #1c9cf0 |
| card | #f7f8f8 | #17181c |
| muted | #e5e5e6 | #181818 |
| accent | #e3ecf6 | #061622 |
| border | #e1eaef | #242628 |
| destructive | #f4212e | #f4212e |

### ocean-breeze
Light/dark. Green primary. DM Sans. Medium radius (0.5rem). Subtle shadows.

| Token | Light | Dark |
|-------|-------|------|
| background | #f0f8ff | #0f172a |
| foreground | #374151 | #d1d5db |
| primary | #22c55e | #34d399 |
| card | #fff | #1e293b |
| muted | #f3f4f6 | #19212e |
| accent | #d1fae5 | #374151 |
| border | #e5e7eb | #4b5563 |

### soft-pop / caffeine
Light/dark. Indigo primary, teal secondary, amber accent. DM Sans + Space Mono. High radius (1rem). Near-zero shadows.

| Token | Light | Dark |
|-------|-------|------|
| background | #f7f9f3 | #000 |
| foreground | #000 | #fff |
| primary | #4f46e5 | #818cf8 |
| secondary | #14b8a6 | #2dd4bf |
| accent | #f59e0b | #fcd34d |
| card | #fff | #1a212b |
| muted | #f0f0f0 | #333 |
| border | #000 | #545454 |

### amber-minimal
Light/dark. Warm amber primary. DM Sans. Medium radius (0.5rem). Subtle shadows.

| Token | Light | Dark |
|-------|-------|------|
| background | #fff | #0a0a0a |
| foreground | #262626 | #fafafa |
| primary | #f59e0b | #fbbf24 |
| card | #fff | #171717 |
| muted | #f5f5f5 | #262626 |
| accent | #fffbeb | #451a03 |
| border | #e5e5e5 | #404040 |

## Typography

| Role | Font | Weight | Tracking |
|------|------|--------|----------|
| Body | var(--font-sans) | 400 | normal |
| Emphasis | var(--font-sans) | 600 | normal |
| Mono | var(--font-mono) | 400 | normal |
| Serif | var(--font-serif) | 400 | normal |

Base size: 14px (Tailwind default). Line length capped at 65-75ch.
Per-theme font stacks:
- twitter: Open Sans
- ocean-breeze, soft-pop, amber-minimal: DM Sans
- soft-pop mono: Space Mono
- ocean-breeze mono: IBM Plex Mono

## Elevation

Shadows are theme-dependent. The twitter preset uses near-zero shadows (all alpha set to 0). ocean-breeze and amber-minimal use subtle shadows (0.05-0.1 alpha). Elevation is primarily conveyed through background color shifts (card vs background), not shadows.

Glass surfaces use `--glass-bg`, `--glass-bg-hover`, `--glass-border` for frosted panel effects.

## Spacing

Base unit: 0.25rem (Tailwind default). Consistent 4px grid.
Radius: varies by preset (0.5rem to 1.3rem). All corners use the same radius.

## Component Patterns

### Buttons
- Primary: bg-primary text-primary-foreground, rounded-md
- Ghost: hover:bg-accent, no background by default
- Destructive: bg-destructive text-destructive-foreground
- States: transition-colors for hover/focus, focus-visible:ring-2

### Inputs
- Border: border-input, focus-visible:ring-ring
- Background: bg-background or bg-muted/50 for embedded inputs
- Height: h-9 (36px) default, h-10 for standalone
- Always include aria-label when no visible label

### Dialogs
- Overlay: fixed inset-0 with bg-black/80
- Content: fixed centered, bg-background, rounded (var(--radius)), padded
- Close button: top-right, ghost variant, sr-only label
- Scrollable content: wrap in ScrollArea
- Resizable: use useResizable composable with pointer events, grip handle at bottom-right

### Chat Bubbles
- Customer messages: left-aligned, bg-muted
- Agent messages: right-aligned, bg-primary text-primary-foreground
- Timestamps: text-xs text-muted-foreground
- Extensive CSS in index.css for bubble tails, status indicators

### Contact Lists (Assign Contact pattern)
- Container: role="listbox" with aria-label
- Items: role="option" with aria-selected, Button variant ghost, full width
- Avatar: circle with first initial, aria-hidden="true"
- Layout: name + email stacked, role badge trailing
- Selected state: bg-primary/10 border border-primary/20
- ScrollArea: max-h-[420px]

### Scrollbars
- Custom styled via CSS (thin, themed colors)
- ScrollArea component from shadcn-vue (reka-ui)

### Sidebar
- Uses sidebar-* tokens for background, foreground, accent, border
- Collapsible with toggle

## Accessibility
- Target: WCAG 2.1 AA
- Color contrast: all theme presets must maintain 4.5:1 for normal text, 3:1 for large text
- Interactive elements: focus-visible rings, keyboard navigation
- Screen readers: aria-labels, aria-selected, role attributes
- RTL: Arabic locale fully supported, all layouts must work dir="rtl"

## Do's

- Use CSS custom properties for all color decisions. Never hardcode hex values in components.
- Keep shadows minimal. Let background contrast do the elevation work.
- Test all new components against all 4 theme presets (twitter, ocean-breeze, soft-pop, amber-minimal).
- Use i18n keys for all user-visible strings. Support English and Arabic.
- Use transition-colors for interactive state changes (hover, focus, active).
- Wrap long lists in ScrollArea with a reasonable max-h.
- Use role="listbox"/"option" for selectable lists with aria-selected.
- Provide aria-label for inputs without visible labels.
- Use the useResizable composable for resizable dialogs instead of CSS resize.

## Don'ts

- Don't use CSS resize on position:fixed elements (unreliable cross-browser).
- Don't hardcode English strings in templates. Always use $t() with vue-i18n.
- Don't use min-w on dialogs that might render on small screens. Use viewport-aware widths.
- Don't nest cards inside cards.
- Don't use gradient text (background-clip: text with gradients).
- Don't use glassmorphism as a default visual treatment.
- Don't use side-stripe borders (>1px) as colored accents on cards or list items.
- Don't use bounce or elastic easing curves. Use ease-out-quart or ease-out-expo.
- Don't add decorative animations that don't convey meaning.
