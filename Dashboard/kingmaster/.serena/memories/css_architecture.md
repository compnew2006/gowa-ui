# CSS Architecture

## Token system
- Custom properties with `--km-` prefix defined in `enterprise-polish.css`.
- OKLCH color space throughout (not hex, not hsl).
- Token categories: `--km-primary`, `--km-success`, `--km-info`, `--km-warning`, `--km-danger` + `-soft` variants.
- Neutral scale: `--km-text`, `--km-text-muted`, `--km-text-soft`, `--km-surface`, `--km-surface-muted`, `--km-bg-elevated`.
- Spacing: `--km-radius`, `--km-radius-sm`, `--km-border`, `--km-border-strong`.
- Transitions: `--transition-fast`.

## Cascade order (critical)
Files load in this order via `head.php`:
1. `css/navbar_styles.css` — base navbar layout (contains `position: fixed` with complex `calc()` values and media queries at 1860px, 920px, 640px).
2. `css/styles.css` — global base styles (0 `!important` declarations).
3. `css/rtl-ltr.css` — direction toggle.
4. Page-specific CSS (e.g., `css/index.css` for `index.php`).
5. `css/enterprise-polish.css` — token overlay and overrides. **Loads last**, acts as the override layer.

## !important strategy
- Base CSS files have ~5 `!important` total.
- `enterprise-polish.css` reduced from ~200+ to 26 `!important` declarations.
- Remaining `!important` categories:
  1. Font Awesome animation overrides (2)
  2. Gradient-text removal on icons/titles (8)
  3. Pseudo-element `display:none` (2)
  4. Inline style overrides via attribute selectors (3)
  5. `prefers-reduced-motion` (4)
  6. **Nav container positioning** (7) — required because base `navbar_styles.css` uses `position: fixed` with media queries that would win without `!important`.

## Design patterns
- `.content-card` is the base card component.
- Higher specificity for overrides: `.content-card.welcome-card` beats `.content-card`.
- `.km-` prefix for all dashboard-specific classes.
- `.km-row` grid system with `--top`, `--stats`, `--halves`, `--feeds` variants.
- Responsive breakpoints: 1100px, 768px, 640px.
