# Kingmaster Dashboard — Complete Project Overview

## 1. Project Structure

The Dashboard consists of **two parallel PHP projects** that share the same codebase origin but have diverged:

```
Dashboard/
├── kingmaster/           # → Production-hardened version (env vars, security, full schema)
│   ├── api/              #   136 API endpoint files
│   ├── config/           #   Environment-based database config
│   ├── includes/         #   Shared PHP includes (functions, MLM, notifications, OTP)
│   ├── root/             #   213 page templates (169 PHP + 44 HTML admin SPAs)
│   ├── js/               #   Frontend JavaScript (i18n, charts, accounts, wallet, etc.)
│   ├── css/              #   Stylesheets (RTL-first, dark theme)
│   ├── assets/           #   Additional CSS/JS
│   ├── images/           #   Static images
│   ├── migrations/       #   Performance migration SQL
│   ├── .htaccess         #   Full security headers + CSP + rewrite
│   ├── schema_only.sql   #   Complete production DB schema (76 tables)
│   ├── PRD.md            #   Product requirements
│   ├── DESIGN.md         #   Design system (OKLCH tokens, typography, spacing)
│   ├── AGENTS.md         #   Agent instructions
│   └── CHANGELOG.md      #   Change log
│
├── kingmaster.info/      # → Dev/older deployment (hardcoded creds, scattered SQL)
│   ├── api/              #   136 PHP API files (identical set to kingmaster)
│   ├── config/           #   Hardcoded database credentials
│   ├── includes/         #   Same shared includes (mostly identical)
│   ├── sql/              #   Scattered SQL table definitions (~6 files)
│   ├── database/         #   Database migration SQL files (~18 files)
│   ├── docs/             #   Local documentation
│   ├── test/             #   Test files
│   ├── uploads/          #   Uploaded files
│   ├── vendor/           #   Composer dependencies (PhpSpreadsheet)
│   └── maps/             #   Additional assets
│
├── docs/                 # → Refactoring documentation (this directory)
│   ├── PROJECT_OVERVIEW.md
│   ├── ARCHITECTURE.md
│   ├── API_ENDPOINTS.md
│   ├── DATABASE_SCHEMA.md
│   ├── FACEBOOK_MODELS.md
│   ├── FEATURE_WORKFLOWS.md
│   ├── FUNCTIONS_CATALOG.md
│   └── REFACTOR_MAP.md
│
├── DESIGN.md             # Overall visual design guide (Glassmorphism, dark theme)
├── PRODUCT.md            # Product description & brand personality
├── ecosystem.config.js   # PM2 production configuration
```

## 2. Key Differences: `kingmaster` vs `kingmaster.info`

| Aspect | `kingmaster` (Production) | `kingmaster.info` (Dev/Older) |
|--------|--------------------------|-------------------------------|
| **Config** | Environment variables via `getenv()` | Hardcoded credentials |
| **Security** | CSP, CSRF, rate limiting, XSS, X-Frame | Basic `.htaccess` only |
| **Schema** | Single `schema_only.sql` (76 tables) | Scattered SQL files (30+, ~20 unique tables) |
| **Admin UI** | ~40 HTML admin SPA pages | No HTML SPAs |
| **Composer** | `composer.json` only (no vendor/) | Full `vendor/` installed (PhpSpreadsheet) |
| **DB Rows** | 87M+ data_fb, 52M retarget_rep, 38M fb_serch | Dev data only |
| **Metadata** | AGENTS.md, DESIGN.md, PRD.md, CHANGELOG.md | None |
| **API files** | 136 endpoints | 136 endpoints (same files, different config/security) |
| **DB Port** | Included in DSN (`DB_PORT`) | Not used |

## 3. System Goal

Kingmaster is an **MLM (Multi-Level Marketing) + Social Media Automation Platform** enabling entrepreneurs to:

- **Manage referral networks** and MLM commission structures (4-level, 15%/10%/5%/2.5%)
- **Run bulk marketing campaigns** via Facebook, Instagram, and WhatsApp
- **Extract social media data** (87M+ people database, profiles, posts, comments, likes, members)
- **Send automated messages** across platforms
- **Track business analytics** and performance metrics
- **Manage team accounts** and subscription packages
- **E-commerce** with products, orders, coupons, and wallet system
- **AI Chatbot** with visual flow builder

## 4. Feature Domains

### 4.1 Multi-Platform Account Management
Supports 8 channels: `facebook`, `whatsapp`, `instagram`, `telegram`, `email`, `sms`, `tiktok`, `linkedin`

### 4.2 Facebook Tools (13 categories)
- Page Search, Group Search, People Search
- Post Engagement (Likes, Comments)
- Group Member Extraction
- Page Messenger Extraction
- Data Extraction (People DB — 87M rows, local cached, not live FB)
- Auto-Share to Groups
- Page Messaging & Retargeting
- Post Creation
- Facebook Analytics
- Account Management
- Content Management
- Sending Settings & Scheduling

### 4.3 WhatsApp Tools
- Bulk Message Campaigns
- Group Campaigns
- Number Filtering (304K+ records)
- Contact Extraction (146K+ records)
- Message Extraction (2.2M+ messages)
- Chat Interface
- Polls & Lists
- Proxy Support
- Flow Builder (chatbot automation)

### 4.4 Instagram Tools
- Profile Search & Bio Search
- Hashtag & Location Search
- Data Extraction (followers, following, likes, comments, posts, story viewers, DMs)
- Auto-Post & Auto-Story
- Send DM / Retargeting
- ⚠️ Note: `insta-follow-tool.php` and `insta-unfollow-tool.php` are **misnamed** — they're group search tools, not real IG follow/unfollow

### 4.5 MLM & Commission System
- 4-level referral chain
- Commission rates: 15% / 10% / 5% / 2.5%
- Commission wallets
- Referral tree visualization
- MLM settings (admin)

### 4.6 Wallet & Financial System
- Digital wallet (points + money balance)
- Peer-to-peer transfers
- Wallet OTP verification
- Withdrawal requests
- Transaction history

### 4.7 Points Economy
- Earn/buy points
- Points packages (purchase)
- Points consumption per data extraction (1 point/record)
- Points settings (admin)

### 4.8 E-Commerce
- Product catalog (with colors, sizes, digital goods)
- Shopping cart & checkout
- Order management & tracking
- Coupons & discounts
- Payment methods

### 4.9 Subscription Packages
- Package catalog with features
- Package purchase & expiry
- Account limits per package
- MLM package commissions

### 4.10 Analytics
- User analytics (registrations, activity)
- Campaign analytics (success/failure rates)
- Revenue analytics
- Package performance
- Admin statistics dashboard
- Facebook page insights

### 4.11 System Features
- Announcements system
- In-app notifications
- Multi-language (Arabic/English/French)
- RTL/LTR layout
- Activity logging
- File management
- Proxy management
- Sales targets & CRM

## 5. Technology Stack (Current PHP)

| Layer | Technology |
|-------|-----------|
| **Backend** | PHP 7.4+ (native, no framework) |
| **Database** | MySQL 8 (InnoDB) via PDO |
| **Frontend** | Vanilla HTML/CSS/JS + ~40 HTML admin SPAs |
| **Auth** | Session-based (PHP sessions + CSRF) |
| **APIs** | Facebook Graph API (v17-v18), Instagram Private API, WhatsApp Web (WPPConnect) |
| **CSS** | Custom OKLCH-based dark theme, RTL-first (Arabic) |
| **Charts** | Chart.js |
| **Icons** | Font Awesome 6.4 |
| **Config** | Environment variables (`kingmaster`) / Hardcoded (`kingmaster.info`) |
| **Composer** | `phpoffice/phpspreadsheet ^5.4` |

## 6. Codebase Statistics

| Category | Count |
|----------|-------|
| **API endpoint files** | 136 |
| **Frontend page templates** | 213 (169 PHP + 44 HTML) |
| **JavaScript utility files** | 11 |
| **CSS stylesheets** | 12 |
| **Include partials** | 15 |
| **Database tables** | 76 (production) |
| **Database rows** | ~189M+ total |
| **Business logic functions** | 23 (in `includes/functions.php`) |
| **Database helper functions** | 16+ (in `config/database.php`) |
| **MLM functions** | 4 |
| **Notification functions** | 2 |
| **WhatsApp messaging functions** | 2 |
| **Inline API helper functions** | ~30+ |

## 7. Largest Database Tables

| Table | Rows | Purpose |
|-------|------|---------|
| `data_fb` | 87,542,089 | Facebook scraper people data (local cached DB) |
| `retarget_rep` | 52,008,804 | Retargeting data |
| `fb_serch` | 38,394,474 | Facebook page search results |
| `wpp_events` | 7,357,840 | WhatsApp Web events |
| `wpp_messages` | 2,246,870 | WhatsApp messages |
| `rb_wa` | 752,006 | WhatsApp retargeting |
| `wa_contacts` | 146,066 | WhatsApp contacts |
| `wa_msg` | 67,873 | WhatsApp messages |
| `wa_members_gb` | 28,799 | WhatsApp group members |
| `notifications` | 19,863 | In-app notifications |
| `campaigns` | 5,011 | Automation campaigns |
| `groups_list` | 5,063 | Facebook group directory |
| `ig_dms` | 37,988 | Instagram DMs |
| `ig_follow` | 5,237 | Instagram follows |

## 8. Authentication & Authorization

### Current (PHP):
- Session-based auth via `$_SESSION`
- CSRF token validation for state-changing operations
- File-based rate limiting via temp files
- Input sanitization (`cleanText`, `sanitizeInput`, `safeBaseName`)
- Prepared statements (PDO)
- CSP headers
- Role-based access: `is_admin` column + `user_type` column
- Admin impersonation (`switch_user` / `return_to_admin`)

## 9. Campaign System

The campaigns table is the **unified execution engine** supporting all platforms:

- **Statuses**: `pending`, `running`, `paused`, `stopped`, `finished`
- **Platforms**: `Facebook`, `Instagram`, `WhatsApp`
- **Type Tools**: `Extract`, `Send`, `Reply`
- **Tools**: Page Search, Group Search, People Search, Post Likes, Comment Reply, Send Messages, etc.
- **Speed Levels**: `slow`, `medium`, `fast`
- **Content**: Linked to message templates
- **Intervals**: Linked to sending settings (min/max delay, daily limit)

## 10. Refactoring Target (Go + Vue within Whatomate)

### Target Stack:
| Layer | Technology |
|-------|-----------|
| **Backend** | Go (fasthttp + fastglue) |
| **Database** | PostgreSQL via GORM |
| **Frontend** | Vue 3 (Composition API) + TypeScript |
| **Auth** | JWT (access + refresh tokens) |
| **State** | Pinia stores |
| **Queue** | Redis Streams |
| **WebSocket** | Real-time hub pattern |
| **Plugin** | Whatomate Plugin System |

### What Already Exists in Whatomate:
- Facebook Account model + handlers
- Facebook Comment model + handlers
- Facebook Page Search model + handlers
- Facebook People Search model + handlers
- Campaign system (different design)
- Multi-tenant middleware via `tenant.ScopedDB()`
- JWT authentication
- WebSocket hub pattern

### Key Design Decisions for Refactoring:
1. **Plugin-first** — Features as plugins under `plugin/<name>/`
2. **Multi-tenant** — Organization-scoped data
3. **Reuse Whatomate Facebook models** — 80% already exist
4. **Replace points economy** — With subscription/license tiers
5. **PostgreSQL partitioning** — For 87M+ rows tables
6. **Redis-backed rate limiting** — Replace file-based
