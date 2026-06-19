# Kingmaster Dashboard — Complete Architecture

## 1. System Architecture (Current PHP)

```
┌──────────────────────────────────────────────────────────────────┐
│                    Apache / Nginx Server                          │
│  (Session-based auth, direct PHP page rendering, .htaccess)      │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                  Security Middleware                          ││
│  │  ┌────────┐ ┌────────┐ ┌───────────┐ ┌─────────┐ ┌───────┐  ││
│  │  │  CSP   │ │  CORS  │ │ RateLimit │ │  CSRF  │ │ Auth  │  ││
│  │  └────────┘ └────────┘ └───────────┘ └─────────┘ └───────┘  ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │              Database Layer (PDO + MySQL InnoDB)              ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐  ││
│  │  │   Core   │ │   FB     │ │   WA/IG  │ │   MLM/Wallet   │  ││
│  │  │  Tables  │ │  Tables  │ │  Tables  │ │   Tables       │  ││
│  │  │ (users,  │ │ (data_fb,│ │ (wa_msg, │ │ (mlm_*, wallet,│  ││
│  │  │ accounts)│ │ fb_page) │ │ ig_dms)  │ │ withdrawals)   │  ││
│  │  └──────────┘ └──────────┘ └──────────┘ └────────────────┘  ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                Include Layer (15 shared partials)              ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐  ││
│  │  │ head.php │ │ navbar * │ │sidebar * │ │   footer.php   │  ││
│  │  └──────────┘ └──────────┘ └──────────┘ └────────────────┘  ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │                  Page Rendering Layer                         ││
│  │  root/*.php — 169 PHP templates with inline PHP code         ││
│  │  ~44 admin HTML SPAs — Single-page admin interfaces          ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │              API Layer (136 API files)                        ││
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────────┐ ││
│  │  │ Account│ │Campaign│ │   FB   │ │   IG   │ │    MLM     │ ││
│  │  │ APIs   │ │  APIs  │ │  APIs  │ │  APIs  │ │    APIs    │ ││
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────────┘ ││
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────────┐ ││
│  │  │   WA   │ │ Content│ │ Wallet │ │Product │ │ Analytics  │ ││
│  │  │  APIs  │ │  APIs  │ │  APIs  │ │  APIs  │ │    APIs    │ ││
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────────┘ ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │              External Service Integrations                    ││
│  │  ┌─────────────────┐ ┌────────────────┐ ┌────────────────┐   ││
│  │  │ Facebook Graph  │ │  Instagram     │ │  WhatsApp Web  │   ││
│  │  │ API v17-v18     │ │  Private API   │ │  (WPPConnect)  │   ││
│  │  └─────────────────┘ └────────────────┘ └────────────────┘   ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │               JS Layer (frontend/js/)                         ││
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────────┐ ││
│  │  │i18n  │ │script│ │accounts│ │wallet│ │content│ │products  ││
│  │  │.js   │ │.js   │ │.js    │ │.js   │ │.js   │ │.js       │ ││
│  │  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────────┘ ││
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐               ││
│  │  │files │ │time- │ │country│ │trans-│ │proxy │               ││
│  │  │.js   │ │zones │ │-detec-│ │lations│ │.php  │               ││
│  │  │      │ │.js   │ │tion.js│ │.js   │ │      │               ││
│  │  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘               ││
│  └──────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘
```

## 2. Kingmaster vs Kingmaster.info Architecture Comparison

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Architecture Comparison                           │
├──────────────────────────┬──────────────────────────────────────────┤
│     kingmaster           │       kingmaster.info                    │
├──────────────────────────┼──────────────────────────────────────────┤
│                          │                                          │
│  env-based config        │  hardcoded credentials                  │
│  (getenv + defaults)     │  (DB_HOST, DB_USER, DB_PASS in code)    │
│                          │                                          │
│  Full security headers   │  Basic .htaccess only                   │
│  CSP, XSS, CSRF, Rate   │  No CSRF, no rate limit                 │
│  Limiting                │  No security headers                    │
│                          │                                          │
│  root/ subdirectory      │  Flat root level                        │
│  for page templates      │  (files at project root)                │
│                          │                                          │
│  ~44 HTML admin SPAs     │  No admin SPAs                          │
│  (modern UI)             │  (only PHP pages)                       │
│                          │                                          │
│  schema_only.sql         │  30+ scattered SQL files                │
│  (full 76-table dump)    │  (dev/older schema variants)            │
│                          │                                          │
│  migrations/ dir         │  sql/ + database/ dirs                  │
│  (single performance     │  (scattered table definitions)          │
│   index migration)       │                                          │
│                          │                                          │
│  136 API files           │  136 API files (same file listing)      │
│  (full security)         │  (basic config, no security)            │
│                          │                                          │
│  12 CSS files            │  (relies on CDN)                        │
│  (custom dark theme)     │                                          │
│                          │                                          │
│  11 JS files             │  1 JS file (proxy.php)                  │
│  (full frontend)         │                                          │
│                          │                                          │
│  AGENTS.md, DESIGN.md    │  No metadata                            │
│  PRD.md, CHANGELOG.md    │                                          │
│                          │                                          │
│  Node.js: server.js,     │  Node.js: none                          │
│  sessionManager.js       │                                          │
│                          │                                          │
├──────────────────────────┴──────────────────────────────────────────┤
│  CORE APPLICATION LOGIC: Identical (same includes & API patterns)  │
│  - Both have same mlm_functions.php, notification_helper.php       │
│  - Both have same API endpoint files (identical listing)           │
│  - Both use the same database schema (kingmaster just has more)    │
└──────────────────────────────────────────────────────────────────────┘
```

## 3. Data Flow Patterns

### 3.1 Request Lifecycle (Current)
```
Browser → Apache → .htaccess (rewrite/security headers) → PHP file
  → startSecureSession() → requireAuthenticatedUser() → enforceRateLimit()
  → verifyCsrfToken() → Business Logic → respondJson()
```

### 3.2 Campaign Execution Flow
```
User creates campaign → POST /api/create_campaign.php
  → INSERT INTO campaigns (status='pending')
  → Campaign runner (PHP script) polls for pending campaigns
  → Updates status to 'running'
  → Executes actions (search, message, extract) via external APIs
  → Tracks success/failure → UPDATE true_count/false_count
  → Updates status to 'finished' when complete
```

### 3.3 MLM Commission Flow
```
User purchases package → process_package_purchase.php
  → 1. Deduct balance from wallet
  → 2. Get referral chain (up to 4 levels) via getReferralChain()
  → 3. Calculate commissions (15%/10%/5%/2.5%)
  → 4. Update commission_wallets for each referrer
  → 5. Log commissions in mlm_commissions
```

### 3.4 Data Extraction Flow
```
User searches people → POST /api/data_fb_search.php
  → 1. Authenticate & rate limit
  → 2. Parse filters (country, work, education, location, etc.)
  → 3. If action='count': SELECT COUNT(*) FROM data_fb WHERE ...
  → 4. If action='extract': BEGIN TRANSACTION
       → SELECT points FROM users FOR UPDATE
       → Verify points >= count
       → SELECT data FROM data_fb WHERE ... LIMIT N
       → UPDATE users SET points = points - N
       → INSERT INTO point_use
       → COMMIT
  → 5. Return results
```

## 4. Security Model (Current)

| Measure | Implementation |
|---------|---------------|
| **Authentication** | PHP sessions + `requireAuthenticatedUser()` |
| **Authorization** | `requireAdminUser()` checks `is_admin` or `user_type` |
| **CSRF** | Token generated via `csrfToken()`, verified via `verifyCsrfToken()` |
| **Rate Limiting** | File-based via `enforceRateLimit(scope, limit, window)` |
| **Input Sanitization** | `cleanText()` (strip tags, truncate), `sanitizeInput()` (htmlspecialchars) |
| **Prepared Statements** | PDO with named/positional parameters |
| **CSP** | Content-Security-Policy header via `applySecurityHeaders()` |
| **CORS** | Configurable via `applyCorsHeaders()` with allowed origins |
| **Password Hashing** | ARGON2ID preferred, BCRYPT cost=12 fallback |
| **Secure Tokens** | `random_bytes()` via `generateSecureToken()` |
| **Session Security** | HttpOnly, Secure, SameSite=Lax cookies |
| **Error Handling** | JSON responses via `respondJson()`/`respondError()` with proper HTTP codes |

## 5. Database Architecture

**76 tables** organized into domains:

| Domain | Tables | Size |
|--------|--------|------|
| **Core** | users, accounts, sessions | Small |
| **Facebook** | data_fb, fb_page, fb_serch, db_camp, groups_list | ~126M rows |
| **WhatsApp** | wa_msg, wa_conv, wa_contacts, wa_members_gb, wpp_events, wpp_messages | ~10M rows |
| **Instagram** | ig_dms, ig_follow, ig_msg, ig_post, ig_retarget, ig_search_* | ~60K rows |
| **Campaigns** | campaigns, content, content_messages, contacts | ~8K rows |
| **MLM** | mlm_referrals, mlm_commissions, mlm_settings, mlm_users, commission_wallets | ~500 rows |
| **Wallet** | wallets, users_wallet, wallet_transactions, transactions | ~800 rows |
| **E-commerce** | products, product_colors, product_sizes, orders, order_status_history, coupons | ~100 rows |
| **System** | notifications, announcements, logs, files, syswalt, sending_settings, points_packages | ~28K rows |
| **Retarget** | retarget_rep, rb_wa | ~53M rows |
| **Chatbot** | wa_flows, messenger_templates | ~80 rows |

## 6. File-by-File: `config/database.php`

Both projects share the same `Database` class pattern but with critical differences:

**Production (`kingmaster`)** — Contains 20+ functions:
- `configValue()` — Read env vars with cascading fallback
- `respondJson()/respondError()` — Proper JSON responses
- `applySecurityHeaders()/applyCorsHeaders()` — Security middleware
- `readJsonBody()` — Parse JSON with size validation
- `startSecureSession()/csrfToken()/verifyCsrfToken()` — CSRF protection
- `enforceRateLimit()` — File-based rate limiting
- `requireAuthenticatedUser()/requireAdminUser()` — Auth guards
- `safeBaseName()/cleanText()/sanitizeInput()` — Input sanitization
- `Database` class — PDO singleton with MySQL, utf8mb4
- `getDB()/executeQuery()/fetchRow()/fetchAll()` — DB helpers
- `hashPassword()/verifyPassword()` — Password utilities

**Dev (`kingmaster.info`)** — Contains 7 basic functions only:
- `Database` class — Same pattern, hardcoded credentials
- `getDB()/executeQuery()/fetchRow()/fetchAll()` — Same DB helpers
- `getLastInsertId()/sanitizeInput()/isValidEmail()/generateSecureToken()/hashPassword()/verifyPassword()`

## 7. File-by-File: `includes/functions.php`

Contains **23 business logic functions**:

| # | Function | Purpose | Used By |
|---|----------|---------|---------|
| 1 | `getUserByUserId()` | Fetch user profile fields | All pages |
| 2 | `getWeeklyStats()` | Campaigns last 7 days grouped by weekday | Dashboard |
| 3 | `getPlatformStats()` | Campaign counts per platform | Dashboard |
| 4 | `getLastPosts()` | Latest system posts | Dashboard |
| 5 | `getCampaignCount()` | Total user campaigns | Dashboard/Admin |
| 6 | `getMonthlyPoints()` | Points per month (12 values for charts) | Dashboard/Charts |
| 7 | `getExtractTrueCount()` | Sum of successful extractions | Dashboard |
| 8 | `getPackageName()` | Package name by ID | Dashboard |
| 9 | `getUserIsAdmin()` | Check admin status | Admin pages |
| 10 | `getReferralCountByReferrerId()` | Direct referral count | Profile/Referral |
| 11 | `getReferralsByReferrerId()` | Last 4 referrals | Dashboard |
| 12 | `getUserData()` | Full user profile with avatar, plan, points | Profile/Navbar |
| 13 | `getActivityLog()` | Last 5 log entries with icons | Profile |
| 14 | `generateTimezoneList()` | All timezones with GMT offset | Settings |
| 15 | `insertSyswalt()` | Insert into syswalt ledger | System |
| 16 | `getAllSyswalt()` | Paginated syswalt with filters | Admin |
| 17-21 | Announcement CRUD | Create/Read/Update/Delete/View | Admin/User |
| 22 | `getcommission_walletsById()` | Commission wallet lookup | Wallet |
| 23 | `getContactCount()` | Contact list count | Contacts |

## 8. File-by-File: `includes/mlm_functions.php`

| # | Function | Purpose |
|---|----------|---------|
| 1 | `MLM_COMMISSION_RATES` | Constant: [1=>15%, 2=>10%, 3=>5%, 4=>2.5%] |
| 2 | `getReferralChain()` | Walk up 4 levels of referrers |
| 3 | `distributeMLMCommissions()` | Calculate & distribute commissions across chain |
| 4 | `registerReferral()` | Register new referral relationship |
| 5 | `getMLMStats()` | Get MLM statistics for a user |

## 9. File-by-File: `includes/notification_helper.php` & `includes/send_otp.php`

| # | Function | Purpose |
|---|----------|---------|
| 1 | `addNotification()` | Insert in-app notification |
| 2 | `addNotificationToMultipleUsers()` | Broadcast notification |
| 3 | `sendOTP()` | Send OTP via WhatsApp API |
| 4 | `sendWhatsAppMessage()` | Send custom WhatsApp message |
