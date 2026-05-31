# Conventions

## PHP
- **No framework** — each page is a standalone `.php` file with `session_start()` guard.
- **Page scaffold pattern**: `head.php` → `navbar_*.php` → `sidebar_*.php` → page content → `footer.php`.
- **DB access**: always via `getDB()` (returns PDO instance). Prepared statements with named params (`:user_id`).
- **API endpoints**: standalone PHP files in `includes/` that call `requireAuthenticatedUser()`, read JSON body with `readJsonBody()`, and respond with `respondJson()`.
- **Error handling**: `respondError($message, $statusCode)` — sends JSON and exits.
- **Naming**: snake_case for functions and variables. Arabic comments throughout.
- **Output escaping**: `htmlspecialchars()` on all user-facing data.
- **Admin vs user**: `is_admin` column check. Separate `admin_*.php` includes for admin views.

## JavaScript
- Vanilla JS only, no framework.
- `i18n.js` + `translations.js` for localization.
- Chart.js instances created in `footer.php` or inline `<script>` blocks.
- SweetAlert2 for all modal interactions.

## CSS
- Token-based system with `--km-*` custom properties. See `mem:css_architecture`.
- Cascade order matters: `navbar_styles.css` → `styles.css` → `rtl-ltr.css` → page CSS → `enterprise-polish.css` (loads last, acts as override layer).

## File naming
- PHP pages: `kebab-case.php` (e.g., `data-extraction.php`, `sending-settings.php`).
- API endpoints: `snake_case.php` (e.g., `get_campaigns.php`, `contacts_api.php`).
- CSS: `snake_case.css` (e.g., `enterprise-polish.css`, `navbar_styles.css`).
- JS: `kebab-case.js` or `camelCase.js`.

## HTML
- Arabic RTL-first. `dir="rtl"` on `<html>`.
- Font Awesome icons with `<i class="fas fa-*">` or `<i class="fa-solid fa-*">`.
- Semantic class naming: `km-` prefix for dashboard components.
