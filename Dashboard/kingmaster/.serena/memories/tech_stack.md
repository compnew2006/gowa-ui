# Tech Stack

## Backend
- **PHP** (no framework) — procedural + OOP hybrid. PDO for database.
- **MySQL / MariaDB** — `utf8mb4`, port 3306. Singleton `Database` class in `root/config/database.php`.
- **Composer** — single dependency: `phpoffice/phpspreadsheet ^5.4` (Excel I/O).

## Frontend
- **Raw PHP templates** — no SPA framework, server-rendered HTML.
- **Chart.js** — canvas charts (weekly messages, monthly points, platform distribution).
- **Font Awesome 6** — `fas`/`fa-solid` icons.
- **SweetAlert2** — `Swal.fire()` for confirmation/error dialogs.
- **CSS custom properties** — OKLCH color tokens (`--km-*`). See `mem:css_architecture`.

## Node.js WhatsApp Service
- **Express 4.18** + **@wppconnect-team/wppconnect ^1.30** — REST API for WhatsApp sessions.
- **cors**, **body-parser** — middleware.
- **nodemon** — dev watcher.
- See `mem:whatsapp_service` for endpoint map.

## Infrastructure
- **Apache** — `.htaccess` for routing and security headers.
- **Environment config** — 12-factor via `getenv()` / `$_ENV` (`APP_ENV`, `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASS`, `WPP_API_KEY`, etc.).
- **No build step** — plain CSS, no bundler, no transpiler.

## Security stack
- CSRF tokens (`csrfToken()` / `verifyCsrfToken()`).
- Rate limiting — file-based with `flock()` (`enforceRateLimit()`).
- Argon2id passwords (bcrypt cost 12 fallback).
- CSP, X-Frame-Options, Referrer-Policy, Permissions-Policy headers.
- Input sanitization: `htmlspecialchars()`, `cleanText()`, prepared statements throughout.
