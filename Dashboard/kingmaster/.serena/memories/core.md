# Kingmaster — Project Core

**Product**: Arabic-first MLM platform for WhatsApp/Facebook/Instagram data extraction, bulk messaging, and referral commissions.

**Register**: product (dashboard/tool UI; design serves the product).

**Source root**: `root/` contains all user-facing PHP pages, includes, config, and static assets. Node.js WhatsApp service lives at project root alongside `root/`.

**Architecture**: Dual-process — Apache serves PHP; separate Node.js Express process handles WhatsApp sessions via WPPConnect. They communicate over HTTP (PHP calls Node REST endpoints).

**Key directories**:
- `root/` — PHP pages (index.php, data-extraction.php, sending-settings.php, etc.)
- `root/includes/` — shared PHP fragments (head, navbar, sidebar, footer) and ~150 API endpoint files
- `root/config/` — database.php (PDO singleton, security helpers, rate limiting, CSRF)
- `root/css/` — token-based OKLCH design system (`enterprise-polish.css` loads last in cascade)
- `root/js/` — i18n, translations, chart logic, feature scripts
- `root/api/` — AJAX endpoint handlers
- `root/images/`, `root/uploads/` — static assets and user uploads
- `migrations/` — SQL migration files

**Module memories**:
- Languages, frameworks, versions: `mem:tech_stack`
- Build/dev/test commands: `mem:suggested_commands`
- Code patterns and conventions: `mem:conventions`
- Done-checklist: `mem:task_completion`
- CSS architecture and design tokens: `mem:css_architecture`
- WhatsApp service details: `mem:whatsapp_service`

**RTL**: Arabic is primary language. `rtl-ltr.css` provides LTR toggle. All UI copy in Arabic.

**LSP note**: Only TypeScript/JavaScript files are indexed by the language server. PHP files are not — use `read_file`, `search_for_pattern`, or direct `Read` tool for PHP exploration.

**Session auth**: PHP sessions (`$_SESSION['user_id']`). Admin check via `is_admin` column. `startSecureSession()` sets secure cookie params. `requireAuthenticatedUser()` / `requireAdminUser()` for API guards.
